package sim

import (
	"math"
	"reflect"
	"testing"

	"token-tycoon/internal/balance"
)

func m5State(t testing.TB, b *balance.Balance) State {
	s := m4State(t, b)
	s.Lifetime.Total = b.AGI.UnlockLifetime
	Step(&s, b, 0.1, nil, nil)
	for _, u := range []Unlock{UnlockSafety, UnlockInvestors, UnlockRegions, UnlockAGI} {
		if !s.IsUnlocked(u) {
			t.Fatalf("%s not unlocked", u.Name())
		}
	}
	s.Money = 1e90
	Recompute(&s, b)
	return s
}

func region(t testing.TB, b *balance.Balance, id string) int {
	t.Helper()
	i, ok := b.RegionIndex(id)
	if !ok {
		t.Fatalf("no region %s", id)
	}
	return i
}

func TestRegionsUnlockIsNeutral(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	before := s.Derived.Market.Price
	s.Lifetime.Total = b.Market.RegionsUnlockLifetime
	Step(&s, b, 0.1, nil, nil)
	s2 := s
	s2.Unlocks.Clear(int(UnlockRegions))
	Recompute(&s2, b)
	if !s.IsUnlocked(UnlockRegions) || s.Derived.Market.RegionMult != 1 || math.Abs(s.Derived.Market.Price-s2.Derived.Market.Price) > 1e-12*s.Derived.Market.Price {
		t.Fatalf("unlock changed the price: mult %v price %v vs %v (before %v)", s.Derived.Market.RegionMult, s.Derived.Market.Price, s2.Derived.Market.Price, before)
	}
}

func TestRegionEntryAndCompliance(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	eu, asia, us := region(t, b, "eu"), region(t, b, "asia"), region(t, b, "us")
	if !RegionEntered(&s, b, us) || !RegionCompliant(&s, b, us) || RegionEntered(&s, b, eu) {
		t.Fatal("initial region state")
	}
	evs := Step(&s, b, 0, []Command{BuyCompliance(eu)}, nil)
	if e, ok := hasEvent(evs, EvRejected); !ok || e.Reason != RejectNeedRegion {
		t.Fatalf("compliance before entry: %+v", evs)
	}
	p0 := s.Derived.Market.Price
	gain := RegionGain(&s, b, eu, false)
	cost := EntryCost(&s, b, eu)
	if want := 120 * 60 * pricingIncome(&s); math.Abs(cost-want) > 1e-9*want {
		t.Fatalf("eu cost %v want %v", cost, want)
	}
	money := s.Money
	evs = Step(&s, b, 0, []Command{EnterRegion(eu)}, nil)
	if _, ok := hasEvent(evs, EvRegion); !ok || s.Money != money-cost {
		t.Fatalf("enter eu: %+v", evs)
	}
	if got := s.Derived.Market.RegionMult; math.Abs(got-1.75) > 1e-9 {
		t.Fatalf("region mult %v want 1.75", got)
	}
	if got := s.Derived.Market.Price/p0 - 1; math.Abs(got-gain) > 1e-9 {
		t.Fatalf("gain preview %v, real %v", gain, got)
	}
	if evs = Step(&s, b, 0, []Command{EnterRegion(eu)}, nil); evs[0].Reason != RejectAlreadyOwned {
		t.Fatal("entered twice")
	}
	Step(&s, b, 0, []Command{EnterRegion(asia)}, nil)
	if math.Abs(s.Derived.Market.RegionMult-2.5) > 1e-9 || math.Abs(s.Derived.Market.EntShare-0.4) > 1e-9 {
		t.Fatalf("all regions: mult %v enterprise share %v", s.Derived.Market.RegionMult, s.Derived.Market.EntShare)
	}
	Step(&s, b, 0, []Command{BuyCompliance(eu), BuyCompliance(asia)}, nil)
	if s.Derived.Market.EntShare != 1 || RegionsCompliant(&s, b) != 3 {
		t.Fatalf("compliance: %v", s.Derived.Market.EntShare)
	}
	safeEU := node(t, b, "safe_eu")
	s.Research.Done.Set(safeEU)
	Recompute(&s, b)
	if got, want := ComplianceCost(&s, b, eu), 60*60*pricingIncome(&s)*0.5; math.Abs(got-want) > 1e-9*want {
		t.Fatalf("compliance research: %v want %v", got, want)
	}
}

func TestEnterpriseOnlyInCompliantRegions(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 90, ReleasedS: s.Clock}}
	Recompute(&s, b)
	Step(&s, b, 0, []Command{SetSegments(0, 0.7, 0.3), EnterRegion(region(t, b, "eu"))}, nil)
	mv := &s.Derived.Market
	pro, ent := mv.SegPrice[1], mv.SegPrice[2]
	al := mv.Alloc
	c := 0.4 / 0.7
	want := 1.75 * (al[1]*pro + al[2]*(c*ent+(1-c)*pro))
	if math.Abs(mv.Price-want) > 1e-9*want {
		t.Fatalf("price %v want %v", mv.Price, want)
	}
}

func TestIncidentRateAndResearch(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	if got := IncidentEveryH(&s, b); got != b.Safety.IncidentEveryHBase {
		t.Fatalf("base every %v", got)
	}
	for _, id := range []string{"safe_1", "safe_2", "safe_3", "safe_4"} {
		s.Research.Done.Set(node(t, b, id))
	}
	Recompute(&s, b)
	if got := IncidentEveryH(&s, b); got != b.Safety.IncidentEveryHMax {
		t.Fatalf("full safety branch every %v, want %v", got, b.Safety.IncidentEveryHMax)
	}
}

func TestIncidentLifecycle(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	s.Market.Reputation = 3
	seen := 0
	for k := 0; k < 400000 && seen == 0; k++ {
		evs := Step(&s, b, 1, nil, nil)
		if _, ok := hasEvent(evs, EvIncident); ok {
			seen++
		}
	}
	if seen == 0 || s.Incident.ID == "" {
		t.Fatal("no incident in ~4.6 game days")
	}
	evs := Step(&s, b, 0, []Command{Respond("x")}, nil)
	if evs[0].Reason != RejectNoEvent {
		t.Fatal("wrong key accepted")
	}
	rep := s.Market.Reputation
	evs = Step(&s, b, 0, []Command{Respond("R")}, nil)
	if _, ok := hasEvent(evs, EvIncidentOK); !ok || s.Market.Reputation != rep+b.Safety.ReputationBonus || s.Incident.ID != "" {
		t.Fatalf("respond: %+v rep %v", evs, s.Market.Reputation)
	}

	s.Market.Hype = 2
	s.Incident = Incident{ID: "data_leak", EndsAtS: s.Clock + 1}
	evs = Step(&s, b, 2, nil, nil)
	e, ok := hasEvent(evs, EvIncidentMiss)
	want := 2 * math.Min(1, b.Safety.HypeLoss*1.5)
	if !ok || math.Abs(e.Amount-want) > 1e-9 || s.Stats.IncidentsMissed != 1 {
		t.Fatalf("miss: %+v want loss %v", evs, want)
	}

	s.Incident = Incident{ID: "jailbreak", EndsAtS: s.Clock + 5}
	hype := s.Market.Hype
	Offline(&s, b, 600, nil)
	if s.Incident.ID != "" || s.Stats.IncidentsMissed != 1 || s.Market.Hype > hype {
		t.Fatal("offline must cancel an incident without a penalty")
	}
	before := s.Stats.IncidentsSeen
	Offline(&s, b, 7*86400, nil)
	if s.Stats.IncidentsSeen != before {
		t.Fatal("incidents happened offline")
	}
}

func TestInvestorQuarters(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	iv := &s.Investors
	if !(iv.QuarterEndS > s.Clock) || !(iv.Target > 0) {
		t.Fatalf("quarter not started: %+v", *iv)
	}
	iv.Target = 1
	s.Clock = iv.QuarterEndS
	evs := Step(&s, b, 0.1, nil, nil)
	e, ok := hasEvent(evs, EvQuarter)
	if !ok || e.Count != 1 || iv.Bonus != b.Investors.BeatBonusCapital || s.Stats.QuartersBeaten != 1 {
		t.Fatalf("beaten quarter: %+v %+v", evs, *iv)
	}
	g := b.Investors.TargetGrowth
	if r := iv.Target / iv.LastRevenue; r < 1+g[0]-1e-9 || r > 1+g[1]+1e-9 {
		t.Fatalf("next target ratio %v", r)
	}
	iv.Target = math.Inf(1)
	s.Clock = iv.QuarterEndS
	evs = Step(&s, b, 0.1, nil, nil)
	if e, ok := hasEvent(evs, EvQuarter); !ok || e.Count != 0 || s.Stats.Quarters != 2 || s.Stats.QuartersBeaten != 1 {
		t.Fatalf("missed quarter: %+v", evs)
	}
	s.Lifetime.Layer2Total = 1e45
	base := baseFundingGain(&s, b)
	if got := FundingGain(&s, b); got != math.Floor(base*1.25) {
		t.Fatalf("bonus gain %v from %v", got, base)
	}
	Step(&s, b, 0, []Command{FundingRound()}, nil)
	if s.Investors.Bonus != 0 {
		t.Fatal("bonus not consumed by the round")
	}
}

func TestValuationGrowth(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	iv := &s.Investors
	iv.Day = 10
	for d := 0; d <= 10; d++ {
		iv.DayTotals[d%DaySnapshots] = math.Pow(2, float64(d))
	}
	if got := Growth7d(&s); math.Abs(got-127) > 1e-9 {
		t.Fatalf("growth %v want 127", got)
	}
	v := Valuation(&s, b)
	want := v.Annual * (1 + math.Max(0, s.Derived.Market.Hype)) * (1 + b.Investors.GrowthWeight*math.Log10(128)) * b.Prestige.Layer2.ValuationMult
	if math.Abs(v.Valuation-want) > 1e-9*want {
		t.Fatalf("valuation %v want %v", v.Valuation, want)
	}
}

func TestAGIRace(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	st := b.AGIStages
	s.Money = st[0].Cost / 2
	evs := Step(&s, b, 0, []Command{AGIInvest()}, nil)
	if _, ok := hasEvent(evs, EvAGIInvest); !ok || s.AGI.Invested != st[0].Cost/2 || s.Money != 0 || s.AGI.Stage != 0 {
		t.Fatalf("partial invest: %+v", s.AGI)
	}
	s.Lifetime.Total = 1e100
	Step(&s, b, 0, []Command{PrestigeReset()}, nil)
	if s.AGI.Invested != st[0].Cost/2 {
		t.Fatal("prestige lost the AGI investment")
	}
	s.Money = 1e200
	s.Incident = Incident{ID: "jailbreak", EndsAtS: s.Clock + 5}
	Step(&s, b, 0, []Command{AGIInvest()}, nil)
	if s.AGI.Stage != 1 || !AGIDone(&s, b, balance.AGINoIncidents) || s.Incident.ID != "" || IncidentsOn(&s, b) {
		t.Fatalf("alignment: %+v", s.AGI)
	}
	g0 := s.Derived.GlobalMult
	Step(&s, b, 0, []Command{AGIInvest()}, nil)
	if s.AGI.Stage != 2 || math.Abs(s.Derived.GlobalMult/g0-2) > 1e-9 {
		t.Fatalf("compute stage: mult %v", s.Derived.GlobalMult/g0)
	}
	Step(&s, b, 0, []Command{AGIInvest()}, nil)
	if s.AGI.Stage != 3 || synthMult(&s, b, 1) != 1 {
		t.Fatal("data stage")
	}
	evs = Step(&s, b, 0, []Command{AGIInvest()}, nil)
	if _, ok := hasEvent(evs, EvAGITraining); !ok || s.AGI.Stage != 3 || !(s.AGI.TrainEndsS > s.Clock) {
		t.Fatalf("final training did not start: %+v", s.AGI)
	}
	if evs = Step(&s, b, 0, []Command{AGIInvest()}, nil); evs[0].Reason != RejectBusy {
		t.Fatal("invest during the final training")
	}
	for range 24 {
		Step(&s, b, 3600, nil, nil)
	}
	if s.AGI.Stage != 4 || s.Stats.BestBench != b.AGI.FinalBenchmark || len(s.Training.Models) > MaxModels(&s, b) {
		t.Fatalf("final training result: %+v best %v", s.AGI, s.Stats.BestBench)
	}
	found := false
	for _, md := range s.Training.Models {
		found = found || md.Benchmark == b.AGI.FinalBenchmark
	}
	if !found {
		t.Fatal("benchmark 100 model missing")
	}
	evs = Step(&s, b, 0, []Command{AGIInvest()}, nil)
	if _, ok := hasEvent(evs, EvAGIEnding); !ok || !AGIEnded(&s, b) || s.AGI.EndedS != s.Clock {
		t.Fatalf("ending: %+v", evs)
	}
	if evs = Step(&s, b, 0, []Command{AGIInvest()}, nil); evs[0].Reason != RejectFinished {
		t.Fatal("invest after the ending")
	}
	g := s.Derived.GlobalMult
	s.Lifetime.Total = NextSIThreshold(&s, b)
	s.Clock += 100
	evs = Step(&s, b, 0.1, nil, nil)
	if e, ok := hasEvent(evs, EvSILevel); !ok || s.AGI.SILevel != 1 || e.Index != 1 || s.AGI.SIRecords[0] < 100 {
		t.Fatalf("SI level: %+v", s.AGI)
	}
	if want := 1 + b.AGI.SIBonusPerLevel; math.Abs(s.Derived.GlobalMult/g-want) > 1e-9 {
		t.Fatalf("SI bonus %v want %v", s.Derived.GlobalMult/g, want)
	}
}

func TestAchievements(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	if cap(s.Achievements) < len(Achievements) {
		t.Fatal("achievement slice not preallocated")
	}
	s.Money = 100
	evs := Step(&s, b, 0.5, []Command{Buy(0, 1)}, nil)
	if _, ok := hasEvent(evs, EvAchievement); ok {
		t.Fatal("achievements checked inside a game second")
	}
	evs = Step(&s, b, 0.6, nil, nil)
	e, ok := hasEvent(evs, EvAchievement)
	if !ok || Achievements[e.Index].ID != "first_purchase" || len(s.Achievements) != 1 {
		t.Fatalf("first purchase: %+v %v", evs, s.Achievements)
	}
	Step(&s, b, 1, nil, nil)
	if len(s.Achievements) != 1 {
		t.Fatal("achievement repeated")
	}
	s.Achievements = append(s.Achievements, "bogus", "first_purchase")
	Normalize(&s, b)
	if !reflect.DeepEqual(s.Achievements, []string{"first_purchase"}) || s.Achieved.Count() != 1 {
		t.Fatalf("normalize: %v", s.Achievements)
	}
	seen := map[string]bool{}
	for _, a := range Achievements {
		if seen[a.ID] {
			t.Fatalf("duplicate achievement %s", a.ID)
		}
		seen[a.ID] = true
	}
	if n := len(Achievements); n < 55 || n > 80 {
		t.Fatalf("%d achievements, want about 60", n)
	}
}

func TestStepNoAllocsM5(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	Step(&s, b, 0, []Command{EnterRegion(region(t, b, "eu")), Hire(2)}, nil)
	s.AGI.Stage = len(b.AGIStages)
	s.AGI.EndedS = s.Clock
	s.Incident = Incident{ID: "jailbreak", EndsAtS: s.Clock + 1e6}
	Normalize(&s, b)
	evs := make([]Event, 0, 256)
	cmd := []Command{Boost(0)}
	for range 3000 {
		evs = Step(&s, b, 0.1, cmd, evs[:0])
	}
	allocs := testing.AllocsPerRun(2000, func() {
		evs = Step(&s, b, 0.1, cmd, evs[:0])
	})
	if allocs > 0 {
		t.Fatalf("Step allocates %v times per call", allocs)
	}
}

func TestDeterminismM5(t *testing.T) {
	b := bal(t)
	runIt := func() State {
		s := m5State(t, b)
		evs := make([]Event, 0, 64)
		for k := range 30000 {
			var cmds []Command
			switch k {
			case 0:
				cmds = []Command{EnterRegion(1), AGIInvest()}
			case 500:
				cmds = []Command{Respond("r")}
			}
			evs = Step(&s, b, 1, cmds, evs[:0])
		}
		return s
	}
	a, c := runIt(), runIt()
	if a.Money != c.Money || a.RNG != c.RNG || a.Stats != c.Stats || a.Investors != c.Investors || !reflect.DeepEqual(a.Achievements, c.Achievements) {
		t.Fatal("M5 run is not deterministic")
	}
}
