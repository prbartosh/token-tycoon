package sim

import (
	"math"
	"reflect"
	"testing"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/infra"
)

func unlockAll(s *State, upTo Unlock) {
	for u := Unlock(0); u < upTo; u++ {
		s.Unlocks.Set(int(u))
	}
}

func infraState(t testing.TB, b *balance.Balance) State {
	s := NewState(b, 11)
	for i := range 6 {
		s.Businesses[i].Owned = 60
		s.Businesses[i].Manager = true
	}
	Recompute(&s, b)
	s.Lifetime.Total = b.Infra.UnlockLifetime
	Step(&s, b, 0.1, nil, nil)
	if !s.IsUnlocked(UnlockInfra) || !s.Infra.StarterGiven {
		t.Fatal("infra not unlocked or no starter cluster")
	}
	return s
}

func TestGlobalMilestonesUnlockedOnly(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	for i := range 6 {
		s.Businesses[i].Owned = 30
	}
	Recompute(&s, b)
	if s.Derived.GlobalMS != 1 || s.GlobalMilestones != 1 {
		t.Fatalf("global over 6 unlocked businesses = %d, want 1", s.Derived.GlobalMS)
	}
	s.Lifetime.Total = b.Businesses[6].UnlockLifetime
	Step(&s, b, 0.1, nil, nil)
	if !s.IsUnlocked(UnlockBusiness0 + 6) {
		t.Fatal("tier 7 not unlocked")
	}
	if s.Derived.MinOwned != 0 || s.Derived.GlobalMS != 1 {
		t.Fatalf("earned x2 lost after new tier: min %d global %d", s.Derived.MinOwned, s.Derived.GlobalMS)
	}
}

func TestGlobalMilestoneUndoDoesNotKeepEarned(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	for i := range 6 {
		s.Businesses[i].Owned = 30
	}
	s.Businesses[0].Owned = 20
	Recompute(&s, b)
	s.Money = 1e9
	Step(&s, b, 0, []Command{Buy(0, 5)}, nil)
	if s.GlobalMilestones != 1 {
		t.Fatalf("global %d after buy", s.GlobalMilestones)
	}
	Step(&s, b, 0, []Command{UndoLast()}, nil)
	if s.GlobalMilestones != 0 || s.Derived.GlobalMS != 0 {
		t.Fatalf("undo kept global milestone: %d", s.GlobalMilestones)
	}
}

func TestStarterClusterHeadroom(t *testing.T) {
	b := bal(t)
	s := infraState(t, b)
	d := &s.Derived
	if d.Throttle != 1 {
		t.Fatalf("throttle right after unlock = %v", d.Throttle)
	}
	if d.ComputeCap < d.ComputeNeed*b.Infra.StarterHeadroom-1e-9 {
		t.Fatalf("starter cap %v < need %v * headroom", d.ComputeCap, d.ComputeNeed)
	}
	if d.ComputeCap-b.GPUGens[0].Compute >= d.ComputeNeed*b.Infra.StarterHeadroom {
		t.Fatalf("starter cluster oversized: cap %v need %v", d.ComputeCap, d.ComputeNeed)
	}
	if got := s.Infra.Energy.GridMW * b.Energy.GridPeakAvailability; got < d.RackMW*b.Infra.StarterHeadroom-1e-9 {
		t.Fatalf("starter grid at peak %v < rack MW %v * headroom", got, d.RackMW)
	}
	if !(d.EnergyShare > 0 && d.EnergyShare < 0.2) || d.NetMult != 1-d.EnergyShare {
		t.Fatalf("energy share %v net %v", d.EnergyShare, d.NetMult)
	}
}

func TestThrottleRacksCloudAndSellback(t *testing.T) {
	b := bal(t)
	s := infraState(t, b)
	s.Money = 1e15
	s.Businesses[5].Owned = 200
	Recompute(&s, b)
	Step(&s, b, 0, nil, nil)
	d := &s.Derived
	if !(d.Throttle < 1) {
		t.Fatalf("expected throttling, need %v cap %v", d.ComputeNeed, d.ComputeCap)
	}
	want := d.ComputeCap / d.ComputeNeed
	if math.Abs(d.Throttle-want) > 1e-9 && d.EnergyAvail >= d.EnergyNeed {
		t.Fatalf("throttle %v want %v", d.Throttle, want)
	}
	throttled := d.IncomePerSec
	Step(&s, b, 0, []Command{SetCloud(true)}, nil)
	if d.Throttle != 1 || d.CloudShare != b.Infra.CloudRevenueShare {
		t.Fatalf("cloud: throttle %v share %v", d.Throttle, d.CloudShare)
	}
	Step(&s, b, 0, []Command{SetCloud(false)}, nil)
	gap := d.ComputeNeed - d.ComputeCap
	k := int(math.Ceil(gap / b.GPUGens[0].Compute))
	c0 := RackCost(&s, b, 1, 1)
	owned := s.Infra.Racks[0]
	if math.Abs(c0-b.GPUGens[0].RackCost*math.Pow(b.Infra.RackR, float64(owned))) > 1e-6*c0 {
		t.Fatalf("rack cost %v does not scale by rack_r", c0)
	}
	mw := GridNeededMW(&s, b, float64(k)*b.GPUGens[0].MW)
	Step(&s, b, 0, []Command{BuyRack(1, k), BuyGrid(mw + 1)}, nil)
	if d.Throttle != 1 || !(d.IncomePerSec > throttled) {
		t.Fatalf("after racks: throttle %v income %v vs %v", d.Throttle, d.IncomePerSec, throttled)
	}
	last := b.GPUGens[0].RackCost * math.Pow(b.Infra.RackR, float64(s.Infra.Racks[0]-1))
	m0 := s.Money
	evs := Step(&s, b, 0, []Command{SellRack(1, 1)}, nil)
	if evs[0].Kind != EvRackSold || math.Abs(s.Money-m0-0.3*last) > 1e-6*last {
		t.Fatalf("sellback %v want %v", s.Money-m0, 0.3*last)
	}
	evs = Step(&s, b, 0, []Command{BuyRack(2, 1)}, nil)
	if evs[0].Reason != RejectLocked {
		t.Fatalf("gen 2 should be locked: %+v", evs)
	}
}

func TestComputeUpgradesActive(t *testing.T) {
	b := bal(t)
	s := infraState(t, b)
	need := s.Derived.ComputeNeed
	kv, _ := b.UpgradeIndex("kv_cache")
	s.Upgrades.Set(kv)
	Recompute(&s, b)
	if math.Abs(s.Derived.ComputeNeed-0.8*need) > 1e-9 {
		t.Fatalf("kv_cache: need %v want %v", s.Derived.ComputeNeed, 0.8*need)
	}
	moe, _ := b.UpgradeIndex("moe")
	s.Upgrades.Set(moe)
	Recompute(&s, b)
	if math.Abs(s.Derived.ComputeUnit[4]-b.Businesses[4].Compute*0.8*0.85) > 1e-12 || s.Derived.ComputeUnit[3] != b.Businesses[3].Compute*0.8 {
		t.Fatalf("moe compute: %v %v", s.Derived.ComputeUnit[4], s.Derived.ComputeUnit[3])
	}
}

func TestEnergyIndexAndPeak(t *testing.T) {
	b := bal(t)
	e := &b.Energy
	lo, hi, sum := 100.0, 0.0, 0.0
	n := 0
	for h := 0.0; h < 24*7; h += 0.25 {
		v := infra.EnergyIndex(42, h*3600, e)
		lo, hi = math.Min(lo, v), math.Max(hi, v)
		sum += v
		n++
	}
	if lo < 0 || hi > 100 || math.Abs(sum/float64(n)-e.IndexMean) > 5 || hi-lo < e.IndexDailyAmplitude {
		t.Fatalf("index range %v..%v mean %v", lo, hi, sum/float64(n))
	}
	if infra.EnergyIndex(42, 5000, e) != infra.EnergyIndex(42, 5000, e) || infra.EnergyIndex(42, 5000, e) == infra.EnergyIndex(43, 5000, e) {
		t.Fatal("index not a pure function of seed and clock")
	}
	peak := infra.EnergyIndex(1, 19*3600, e) + infra.EnergyIndex(2, 19*3600, e)
	night := infra.EnergyIndex(1, 7*3600, e) + infra.EnergyIndex(2, 7*3600, e)
	if !(peak > night) {
		t.Fatalf("peak index %v not above night %v", peak, night)
	}
	if !infra.IsPeak(18*3600, e) || infra.IsPeak(12*3600, e) || !infra.IsPeak(86400+20*3600, e) {
		t.Fatal("peak hours")
	}
	l := infra.Dispatch(10, infra.Supply{GridMW: 10, Peak: true}, 0.7)
	if l.EnergyAvail != 7 || infra.Throttle(1, 2, 10, l.EnergyAvail) != 0.7 {
		t.Fatalf("peak dispatch %+v", l)
	}
}

func TestPPAAndSMR(t *testing.T) {
	b := bal(t)
	s := infraState(t, b)
	s.Money = 1e15
	need := s.Derived.EnergyNeed
	Step(&s, b, 0, []Command{BuyPPA(need * 2)}, nil)
	if s.Derived.GridUsed != 0 || s.Derived.PPAUsed == 0 {
		t.Fatalf("PPA not dispatched: %+v", s.Derived)
	}
	idx := s.Infra.Energy.PPA.Index
	if math.Abs(s.Derived.EnergyShare-b.Energy.GridCostCoef*idx/50) > 1e-12 {
		t.Fatalf("PPA share %v", s.Derived.EnergyShare)
	}
	s.Clock = s.Infra.Energy.PPA.ExpiresS
	evs := Step(&s, b, 0.1, nil, nil)
	found := false
	for _, e := range evs {
		found = found || e.Kind == EvPPAExpired
	}
	if !found || s.Infra.Energy.PPA.MW != 0 {
		t.Fatal("PPA did not expire")
	}
	cost := SMRCost(&s, b)
	if !(cost > 0) || math.Abs(cost-b.Energy.SMRCostMinutes*60*math.Max(s.Derived.BaseIncome, s.Stats.BestIncome)) > 1e-6*cost {
		t.Fatalf("smr cost %v", cost)
	}
	Step(&s, b, 0, []Command{BuySMR()}, nil)
	if s.Infra.Energy.SMR != 1 || s.Derived.EnergyShare != 0 {
		t.Fatalf("SMR: %d share %v", s.Infra.Energy.SMR, s.Derived.EnergyShare)
	}
}

func prestigeReady(t testing.TB, b *balance.Balance) State {
	s := infraState(t, b)
	s.Lifetime.Run = 4e12
	s.Lifetime.Total = 4e12
	s.Prestige.RunStartS = 0
	s.Clock = 3600
	Step(&s, b, 0.1, nil, nil)
	if !s.IsUnlocked(UnlockPrestige1) {
		t.Fatal("prestige not unlocked")
	}
	return s
}

func TestPrestigeLayer1(t *testing.T) {
	b := bal(t)
	b.Prestige.Layer1.LeapS = 0
	s := NewState(b, 1)
	s.Lifetime.Run = 1e13
	evs := Step(&s, b, 0, []Command{PrestigeReset()}, nil)
	if evs[0].Reason != RejectLocked {
		t.Fatal("prestige allowed without infra")
	}
	s = prestigeReady(t, b)
	s.Cards["dropout"] = 2
	Normalize(&s, b)
	pv := Preview(&s, b)
	if !pv.Allowed || pv.Gain != 200 || pv.BonusAfter != 5 || pv.VersionAfter != 1 {
		t.Fatalf("preview %+v", pv)
	}
	if !(pv.ReturnS >= 0 && pv.ReturnS < pv.RunS) {
		t.Fatalf("return estimate %v of run %v", pv.ReturnS, pv.RunS)
	}
	total := s.Lifetime.Total
	evs = Step(&s, b, 0, []Command{PrestigeReset()}, nil)
	if evs[0].Kind != EvPrestige || evs[0].Amount != 200 {
		t.Fatalf("prestige event %+v", evs)
	}
	if s.Money != 0 || s.Lifetime.Run != 0 || s.Lifetime.Total != total || s.Businesses[0].Owned != 1 || s.Businesses[1].Owned != 0 ||
		s.Upgrades.Count() != 0 || s.Infra.Racks[0] != 0 || s.Infra.StarterGiven || s.Prestige.Params != 200 {
		t.Fatalf("reset wrong: %+v", s.Prestige)
	}
	if s.Cards["dropout"] != 2 || s.CardMods.ProfitAll == 1 {
		t.Fatal("cards did not survive prestige")
	}
	if s.Derived.GlobalMult < 5 {
		t.Fatalf("params bonus not applied: %v", s.Derived.GlobalMult)
	}
	Step(&s, b, 0.1, nil, nil)
	if !s.Infra.StarterGiven || s.Infra.Racks[0] != 1 {
		t.Fatalf("no starter cluster after prestige: %+v", s.Infra)
	}
	if VersionName(b, 0) != "v1.0" || VersionName(b, 1) != "v2.0" || VersionName(b, 7) != "v5.0" || VersionName(b, 8) != "v6.0" || VersionName(b, 10) != "v8.0" {
		t.Fatal("version names")
	}
	evs = Step(&s, b, 0, []Command{PrestigeReset()}, nil)
	if evs[0].Reason != RejectLocked {
		t.Fatal("second prestige allowed with empty run")
	}
}

func TestPrestigeLeapLimit(t *testing.T) {
	b := bal(t)
	s := prestigeReady(t, b)
	s.Lifetime.Run = 1e20
	full := NewState(b, 1)
	b0 := bal(t)
	b0.Prestige.Layer1.LeapS = 0
	full.Lifetime.Run = s.Lifetime.Run
	raw := Layer1Gain(&full, b0)
	limit := b.Prestige.Layer1.ParamCoef * (s.Clock - s.Prestige.RunStartS) / b.Prestige.Layer1.LeapS
	got := Layer1Gain(&s, b)
	if !(got < raw && got < limit && got > limit*0.9) {
		t.Fatalf("leap gain %v raw %v limit %v", got, raw, limit)
	}
	s.Prestige.RunStartS = s.Clock
	if g := Layer1Gain(&s, b); g != 0 {
		t.Fatalf("zero-length run gains %v", g)
	}
}

func TestCardsDropLevelAndSets(t *testing.T) {
	b, err := balance.Parse(balance.DefaultTOML(), []byte("[cards]\ndrop_chance_per_cycle = 1.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := NewState(b, 5)
	unlockAll(&s, UnlockMarket)
	s.Businesses[0].Owned = 10
	s.Businesses[0].Manager = true
	Recompute(&s, b)
	evs := make([]Event, 0, 64)
	cardsSeen, sets := 0, 0
	for range 20000 {
		evs = Step(&s, b, 0.1, nil, evs[:0])
		for _, e := range evs {
			switch e.Kind {
			case EvCard:
				cardsSeen++
			case EvCardSet:
				sets++
			}
		}
	}
	if cardsSeen != s.Stats.CardsDropped || cardsSeen < 1500 {
		t.Fatalf("drops %d stats %d", cardsSeen, s.Stats.CardsDropped)
	}
	if s.CardMods.Unique != len(b.Card) || sets != 4 {
		t.Fatalf("unique %d sets %d", s.CardMods.Unique, sets)
	}
	common := 0
	for i := range b.Card {
		if b.Card[i].Rarity == "common" {
			common += s.Cards[b.Card[i].ID]
		}
	}
	if f := float64(common) / float64(cardsSeen); f < 0.65 || f > 0.75 {
		t.Fatalf("common share %v", f)
	}
	if CardLevel(&s, b, 0) != b.Cards.MaxLevel {
		t.Fatalf("level cap: %d", CardLevel(&s, b, 0))
	}
	want := 1.0
	for i := range b.Card {
		if b.Card[i].Effect == balance.CardProfitAll {
			want += CardBonus(&s, b, i)
		}
	}
	for _, r := range balance.Rarities {
		want += b.Cards.SetBonus[r]
	}
	if math.Abs(s.CardMods.ProfitAll-want) > 1e-9 {
		t.Fatalf("profit_all %v want %v", s.CardMods.ProfitAll, want)
	}
}

func TestCardDrawsRespectContinuousCap(t *testing.T) {
	b, err := balance.Parse(balance.DefaultTOML(), []byte("[cards]\ndrop_chance_per_cycle = 1.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := NewState(b, 5)
	unlockAll(&s, UnlockMarket)
	s.Businesses[0].Owned = 100
	s.Businesses[0].Manager = true
	si, _ := b.UpgradeIndex("spec_decoding")
	s.Upgrades.Set(si)
	Recompute(&s, b)
	const secs = 200.0
	for range int(secs * 10) {
		Step(&s, b, 0.1, nil, nil)
	}
	if perS := float64(s.Stats.CardsDropped) / secs; perS > b.BusinessDefaults.DrawCapPerS+0.05 {
		t.Fatalf("card draws %.2f/s above cap", perS)
	}
}

func eventState(t testing.TB, b *balance.Balance) State {
	s := NewState(b, 3)
	s.Businesses[0].Owned = 30
	s.Businesses[0].Manager = true
	Recompute(&s, b)
	s.Lifetime.Total = b.Events.UnlockLifetime
	unlockAll(&s, UnlockPrestige1)
	return s
}

func TestRandomEventsTimed(t *testing.T) {
	b, err := balance.Parse(balance.DefaultTOML(), []byte("[[event]]\nid = \"hallucination\"\nweight = 0.0001\n[[event]]\nid = \"angel_investor\"\nweight = 0.0001\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := eventState(t, b)
	evs := make([]Event, 0, 64)
	starts, ends := 0, 0
	var startAt []float64
	for range 36000 {
		evs = Step(&s, b, 0.1, nil, evs[:0])
		for _, e := range evs {
			switch e.Kind {
			case EvEventStart:
				starts++
				startAt = append(startAt, s.Clock)
				if b.Event[e.Index].Effect == balance.EventHype {
					t.Fatal("hype event before the market")
				}
				if i := timedEvent(&s, b); i == e.Index && b.Event[i].Effect == balance.EventProfit && s.Derived.EventMult[0] != b.Event[i].Value {
					t.Fatalf("viral demo mult %v", s.Derived.EventMult[0])
				}
			case EvEventEnd:
				ends++
			}
		}
	}
	if starts < 3600/300 || starts > 3600/120+1 || ends < starts-1 {
		t.Fatalf("starts %d ends %d in 1 h", starts, ends)
	}
	for i := 1; i < len(startAt); i++ {
		if g := startAt[i] - startAt[i-1]; g < 120 {
			t.Fatalf("events %v s apart", g)
		}
	}
}

func forceEvent(s *State, b *balance.Balance, id string) {
	i, _ := b.EventIndex(id)
	s.RandomEvent.ActiveID = id
	s.RandomEvent.EndsAtS = s.Clock + b.Event[i].WindowS + b.Event[i].DurationS
	Recompute(s, b)
}

func TestReactionEvents(t *testing.T) {
	b := bal(t)
	s := eventState(t, b)
	Step(&s, b, 0.1, nil, nil)
	forceEvent(&s, b, "hallucination")
	evs := Step(&s, b, 0, []Command{React("g")}, nil)
	if evs[0].Reason != RejectNoEvent {
		t.Fatal("wrong key accepted")
	}
	inc := s.Derived.BaseIncome
	m0 := s.Money
	evs = Step(&s, b, 0, []Command{React("H")}, nil)
	if evs[0].Kind != EvReact || math.Abs(s.Money-m0-inc*300) > 1e-6*inc || s.RandomEvent.ActiveID != "" || s.Stats.EventsCaught != 1 {
		t.Fatalf("hallucination reward: %+v money %v", evs, s.Money-m0)
	}
	s.Money = 1e9
	forceEvent(&s, b, "hallucination")
	var miss *Event
	for range 60 {
		evs = Step(&s, b, 0.1, nil, evs[:0])
		for i := range evs {
			if evs[i].Kind == EvReactMiss {
				miss = &evs[i]
			}
		}
		if miss != nil {
			break
		}
	}
	if miss == nil || math.Abs(miss.Amount-inc*60) > 1e-3*inc {
		t.Fatalf("hallucination miss: %+v", miss)
	}
	forceEvent(&s, b, "angel_investor")
	m0 = s.Money
	Step(&s, b, 0, []Command{React("g")}, nil)
	if math.Abs(s.Money-m0-inc*900) > 1e-2*inc {
		t.Fatalf("angel reward %v", s.Money-m0)
	}
}

func TestOfflineHasNoReactionEvents(t *testing.T) {
	b := bal(t)
	s := eventState(t, b)
	Step(&s, b, 0.1, nil, nil)
	forceEvent(&s, b, "hallucination")
	m0 := s.Money
	_, evs := Offline(&s, b, 8*3600, nil)
	for _, e := range evs {
		if e.Kind == EvReactMiss || (e.Kind == EvEventStart && b.Event[e.Index].Effect == balance.EventReaction) {
			t.Fatalf("reaction event offline: %+v", e)
		}
	}
	if s.Money < m0 {
		t.Fatal("offline penalty")
	}
	if s.Stats.EventsSeen == 0 {
		t.Fatal("no timed events offline")
	}
}

func TestDailyGrantStreak(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	s.Businesses[0].Manager = true
	s.Businesses[0].Owned = 50
	Recompute(&s, b)
	inc := s.Derived.BaseIncome
	claim := func(date string) []Event { return Step(&s, b, 0, []Command{ClaimGrant(date)}, nil) }
	evs := claim("2026-10-30")
	if evs[0].Kind != EvGrant || evs[0].Count != 1 || math.Abs(evs[0].Amount-inc*600) > 1e-6*inc {
		t.Fatalf("day 1: %+v", evs)
	}
	if claim("2026-10-30")[0].Reason != RejectAlreadyClaimed {
		t.Fatal("double claim")
	}
	if claim("2026-13-01")[0].Reason != RejectBadDate || claim("x")[0].Reason != RejectBadDate {
		t.Fatal("bad date accepted")
	}
	days := []string{"2026-10-31", "2026-11-01", "2026-11-02", "2026-11-03", "2026-11-04", "2026-11-05", "2026-11-06"}
	want := []int{2, 3, 4, 5, 6, 7, 7}
	for i, d := range days {
		if evs := claim(d); evs[0].Count != want[i] {
			t.Fatalf("%s: day %d want %d", d, evs[0].Count, want[i])
		}
	}
	if evs := claim("2026-11-09"); evs[0].Count != 6 {
		t.Fatalf("after a break: day %d want 6", evs[0].Count)
	}
	s.Streak = Streak{Day: 1, LastClaim: "2028-02-28"}
	if evs := claim("2028-02-29"); evs[0].Count != 2 {
		t.Fatal("leap day")
	}
	if evs := claim("2028-03-01"); evs[0].Count != 3 {
		t.Fatal("month rollover")
	}
	if claim("2028-02-01")[0].Reason != RejectAlreadyClaimed {
		t.Fatal("going back in time accepted")
	}
}

func TestStepNoAllocsAllSystems(t *testing.T) {
	b := bal(t)
	s := infraState(t, b)
	unlockAll(&s, UnlockMarket)
	s.Cards["dropout"] = 1
	Normalize(&s, b)
	evs := make([]Event, 0, 256)
	cmds := []Command{Boost(0)}
	for range 3000 {
		evs = Step(&s, b, 0.1, cmds, evs[:0])
	}
	allocs := testing.AllocsPerRun(2000, func() {
		evs = Step(&s, b, 0.1, cmds, evs[:0])
	})
	if allocs > 0 {
		t.Fatalf("Step allocates %v times per call", allocs)
	}
}

func TestDeterminismAllSystems(t *testing.T) {
	b := bal(t)
	runIt := func() State {
		s := infraState(t, b)
		unlockAll(&s, UnlockMarket)
		evs := make([]Event, 0, 64)
		for k := range 30000 {
			var cmds []Command
			if k%100 == 0 {
				cmds = []Command{React("h"), React("g"), BuyRack(1, 1)}
			}
			evs = Step(&s, b, 0.1, cmds, evs[:0])
		}
		return s
	}
	a, c := runIt(), runIt()
	if !reflect.DeepEqual(a, c) {
		t.Fatal("nondeterministic")
	}
	if a.Stats.EventsSeen == 0 {
		t.Fatal("no events")
	}
}
