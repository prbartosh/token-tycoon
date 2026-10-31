package sim

import (
	"math"
	"reflect"
	"testing"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
)

func bal(t testing.TB) *balance.Balance {
	t.Helper()
	return balance.Default()
}

func run(s *State, b *balance.Balance, seconds float64, cmds []Command) []Event {
	var evs []Event
	steps := int(math.Round(seconds / 0.1))
	for k := range steps {
		var c []Command
		if k == 0 {
			c = cmds
		}
		evs = Step(s, b, 0.1, c, evs)
	}
	return evs
}

func TestNewGame(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	if s.Money != 0 || s.Businesses[0].Owned != 1 || s.Businesses[0].Manager {
		t.Fatalf("start: money %v, wrapper %+v", s.Money, s.Businesses[0])
	}
	if !s.IsUnlocked(UnlockBusiness0) || s.IsUnlocked(UnlockBusiness0+6) || s.IsUnlocked(UnlockInfra) {
		t.Fatal("initial unlocks wrong")
	}
	run(&s, b, 5, nil)
	if s.Money != 0 {
		t.Fatalf("idle wrapper earned %v", s.Money)
	}
}

func TestManualCycleAndPrimary(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	evs := Step(&s, b, 0.1, []Command{Primary(0, 1)}, nil)
	if !s.Businesses[0].Running || evs[0].Kind != EvStart {
		t.Fatalf("primary did not start: %+v", evs)
	}
	run(&s, b, 1.0, nil)
	if s.Businesses[0].Running {
		t.Fatal("manual cycle should stop after one cycle")
	}
	if s.Money != 1 && s.Money != 10 {
		t.Fatalf("money after one cycle = %v", s.Money)
	}
	if s.Lifetime.Total != s.Money || s.Lifetime.Run != s.Money {
		t.Fatal("lifetime not tracked")
	}
}

func TestBuyManagerAndUndo(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	s.Money = 100
	evs := Step(&s, b, 0.1, []Command{HireManager(0)}, nil)
	if !s.Businesses[0].Manager || s.Money != 100-b.ManagerCost(0) || evs[0].Kind != EvManager {
		t.Fatalf("manager: %+v money %v", evs, s.Money)
	}
	evs = Step(&s, b, 0.1, []Command{UndoLast()}, evs[:0])
	if s.Businesses[0].Manager || s.Money != 100 || evs[0].Kind != EvUndo {
		t.Fatalf("undo: %+v money %v", evs, s.Money)
	}
	evs = Step(&s, b, 0.1, []Command{UndoLast()}, evs[:0])
	if evs[0].Kind != EvRejected || evs[0].Reason != RejectNothingToUndo {
		t.Fatalf("second undo: %+v", evs)
	}
	Step(&s, b, 0.1, []Command{Buy(0, 1)}, nil)
	run(&s, b, 5.2, nil)
	evs = Step(&s, b, 0.1, []Command{UndoLast()}, evs[:0])
	if evs[0].Kind != EvRejected || s.Businesses[0].Owned != 2 {
		t.Fatalf("expired undo: %+v owned %d", evs, s.Businesses[0].Owned)
	}
}

func TestBulkBuyAndMilestones(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	s.Money = econ.BulkCost(b.Businesses[0].Cost, b.Businesses[0].R, 1, 24)
	evs := Step(&s, b, 0, []Command{Buy(0, QtyMax)}, nil)
	if s.Businesses[0].Owned != 25 {
		t.Fatalf("owned %d after MAX", s.Businesses[0].Owned)
	}
	if math.Abs(s.Money) > 1e-6 {
		t.Fatalf("money left %v", s.Money)
	}
	found := false
	for _, e := range evs {
		if e.Kind == EvMilestoneSpeed && e.Index == 25 && e.Amount == 0.5 {
			found = true
		}
	}
	if !found {
		t.Fatalf("no speed milestone event: %+v", evs)
	}
	if s.Derived.CycleS[0] != 0.5 {
		t.Fatalf("cycle %v, want 0.5", s.Derived.CycleS[0])
	}
	evs = Step(&s, b, 0, []Command{Buy(1, Qty10)}, evs[:0])
	if evs[0].Kind != EvRejected || evs[0].Reason != RejectCannotAfford || !(evs[0].Amount > 0) {
		t.Fatalf("x10: %+v", evs)
	}
	s.Money = 1e12
	evs = Step(&s, b, 0, []Command{Buy(6, 1)}, evs[:0])
	if evs[0].Reason != RejectLocked {
		t.Fatalf("locked: %+v", evs)
	}
}

func TestGlobalMilestones(t *testing.T) {
	b := bal(t)
	base := NewState(b, 1)
	h := HypoOf(&base)
	h.Unlocked = balance.AllBusinesses
	for i := range N {
		h.Owned[i] = 250
	}
	var d Derived
	Evaluate(b, &h, &d)
	if d.GlobalMS != 6 || d.GlobalMult != 64 {
		t.Fatalf("global %d x%v", d.GlobalMS, d.GlobalMult)
	}
	for i := range N {
		h.Owned[i] = 1000
	}
	Evaluate(b, &h, &d)
	if d.GlobalMS != 21 {
		t.Fatalf("global at 1000 = %d, want 21 (forever every 50)", d.GlobalMS)
	}
	if d.ProfitMS[0] != 37 || d.SpeedMS[0] != 3 {
		t.Fatalf("profit %d speed %d", d.ProfitMS[0], d.SpeedMS[0])
	}
}

func TestUpgrades(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	s.Money = 1e6
	base := s.Derived.PerCycle[0]
	ui, _ := b.UpgradeIndex("prompt_caching")
	Step(&s, b, 0, []Command{BuyUpgrade(ui)}, nil)
	if s.Derived.PerCycle[0] != 3*base {
		t.Fatalf("prompt caching: %v vs %v", s.Derived.PerCycle[0], base)
	}
	qi, _ := b.UpgradeIndex("quant_4bit")
	c0 := BusinessCost(&s, b, 2, 1)
	Step(&s, b, 0, []Command{BuyUpgrade(qi)}, nil)
	if c1 := BusinessCost(&s, b, 2, 1); math.Abs(c1-0.9*c0) > 1e-9 {
		t.Fatalf("quant cost %v vs %v", c1, c0)
	}
	si, _ := b.UpgradeIndex("spec_decoding")
	Step(&s, b, 0, []Command{BuyUpgrade(si)}, nil)
	if s.Derived.CycleS[1] != 2.25 {
		t.Fatalf("spec decoding cycle %v", s.Derived.CycleS[1])
	}
	evs := Step(&s, b, 0, []Command{BuyUpgrade(si)}, nil)
	if evs[0].Reason != RejectAlreadyOwned {
		t.Fatal("double upgrade not rejected")
	}
}

func TestComboBoost(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	for range 50 {
		Step(&s, b, 0.05, []Command{Boost(0)}, nil)
	}
	if s.Combo.Mult != b.Combo.MaxMult {
		t.Fatalf("combo %v, want cap %v", s.Combo.Mult, b.Combo.MaxMult)
	}
	if s.Stats.ManualStarts == 0 {
		t.Fatal("Space did not start the idle selected business")
	}
	evs := run(&s, b, 2.1, nil)
	if s.Combo.Mult != 1 {
		t.Fatalf("combo did not decay: %v", s.Combo.Mult)
	}
	ended := false
	for _, e := range evs {
		ended = ended || e.Kind == EvComboEnd
	}
	if !ended {
		t.Fatal("no EvComboEnd")
	}
}

func TestComboMultipliesRevenue(t *testing.T) {
	b := balanceNoCrit(t)
	s := NewState(b, 1)
	s.Businesses[0].Manager = true
	Recompute(&s, b)
	run(&s, b, 1, nil)
	plain := s.Money
	s2 := NewState(b, 1)
	s2.Businesses[0].Manager = true
	for range 10 {
		Step(&s2, b, 0, []Command{Boost(-1)}, nil)
	}
	run(&s2, b, 1, nil)
	if math.Abs(s2.Money-2*plain) > 1e-9 {
		t.Fatalf("combo x2 revenue %v vs %v", s2.Money, plain)
	}
}

func balanceNoCrit(t testing.TB) *balance.Balance {
	b, err := balance.Parse(balance.DefaultTOML(), []byte("[crit]\nchance = 0.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCritCapContinuous(t *testing.T) {
	b := bal(t)
	s := NewState(b, 7)
	s.Businesses[0].Owned = 100
	s.Businesses[0].Manager = true
	si, _ := b.UpgradeIndex("spec_decoding")
	s.Upgrades.Set(si)
	Recompute(&s, b)
	if !s.Derived.Continuous[0] {
		t.Fatal("expected continuous mode")
	}
	const secs = 2000.0
	evs := make([]Event, 0, 64)
	rolled := 0
	for range int(secs * 10) {
		evs = Step(&s, b, 0.1, nil, evs[:0])
		for _, e := range evs {
			if e.Kind == EvCycle {
				rolled += e.Count
			}
		}
	}
	if perS := float64(rolled) / secs; perS > b.BusinessDefaults.DrawCapPerS+0.01 {
		t.Fatalf("rolled %.2f cycles/s, cap %v", perS, b.BusinessDefaults.DrawCapPerS)
	}
	want := s.Derived.PerSec[0] * secs
	if rel := math.Abs(s.Money-want) / want; rel > 0.01 {
		t.Fatalf("continuous revenue %v vs EV %v (%.2f%%)", s.Money, want, rel*100)
	}
}

func TestCritAverageIndependentOfCycle(t *testing.T) {
	b := bal(t)
	s := NewState(b, 3)
	s.Businesses[0].Manager = true
	Recompute(&s, b)
	const secs = 20000.0
	evs := make([]Event, 0, 16)
	for range int(secs * 10) {
		evs = Step(&s, b, 0.1, nil, evs[:0])
	}
	want := s.Derived.PerSec[0] * secs
	if rel := math.Abs(s.Money-want) / want; rel > 0.03 {
		t.Fatalf("1 s cycle revenue %v vs EV %v", s.Money, want)
	}
}

func scripted(b *balance.Balance, seed uint64) State {
	s := NewState(b, seed)
	evs := make([]Event, 0, 64)
	for k := range 20000 {
		var cmds []Command
		switch {
		case k%7 == 0:
			cmds = []Command{Primary(0, 1), Boost(0)}
		case k%50 == 0:
			cmds = []Command{HireManager(0), Buy(1, QtyMax), HireManager(1), Buy(0, Qty10)}
		case k%300 == 0:
			cmds = []Command{UndoLast(), BuyUpgrade(0)}
		}
		evs = Step(&s, b, 0.1, cmds, evs[:0])
	}
	return s
}

func TestDeterminism(t *testing.T) {
	b := bal(t)
	a, c := scripted(b, 99), scripted(b, 99)
	if !reflect.DeepEqual(a, c) {
		t.Fatal("same seed and commands produced different states")
	}
	if a.Lifetime.Total == 0 || a.Stats.Crits == 0 {
		t.Fatalf("scripted run did nothing useful: %+v", a.Stats)
	}
	d := scripted(b, 100)
	if d.Stats.Crits == a.Stats.Crits && d.Money == a.Money {
		t.Fatal("different seeds produced identical runs")
	}
}

func TestUnlocksByRevenue(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	s.Lifetime.Total = max(b.Infra.UnlockLifetime, b.Events.UnlockLifetime, b.Cards.UnlockLifetime)
	s.Lifetime.Run = b.Prestige.Layer1.UnlockRunRevenue
	evs := Step(&s, b, 0.1, nil, nil)
	got := map[Unlock]bool{}
	for _, e := range evs {
		if e.Kind == EvUnlock {
			got[Unlock(e.Index)] = true
		}
	}
	if !got[UnlockInfra] || !got[UnlockEvents] || !got[UnlockCards] || !got[UnlockPrestige1] || got[UnlockMarket] {
		t.Fatalf("unlocks: %v", got)
	}
	for u := Unlock(0); u < NumUnlocks; u++ {
		if p, ok := ParseUnlock(u.Name()); !ok || p != u {
			t.Fatalf("unlock name round trip failed for %d", u)
		}
	}
}

func TestOffline(t *testing.T) {
	b := balanceNoCrit(t)
	s := NewState(b, 1)
	s.Businesses[0].Manager = true
	s.Businesses[1].Owned = 10
	s.Businesses[1].Manager = true
	Recompute(&s, b)
	rate := s.Derived.IncomePerSec
	res, _ := Offline(&s, b, 3600, nil)
	want := rate * 3600 * b.Offline.EfficiencyStart
	if math.Abs(res.Earned-want)/want > 0.01 {
		t.Fatalf("offline earned %v, want ~%v", res.Earned, want)
	}
	if res.Steps != 60 || s.Stats.OfflineS != 3600 {
		t.Fatalf("steps %d offline_s %v", res.Steps, s.Stats.OfflineS)
	}
	res, _ = Offline(&s, b, 48*3600, nil)
	if res.Simulated != b.Offline.CapHoursStart*3600 {
		t.Fatalf("offline cap not applied: %v", res.Simulated)
	}
}

func TestStepNoAllocs(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	for i := range 6 {
		s.Businesses[i].Owned = 50
		s.Businesses[i].Manager = true
	}
	Recompute(&s, b)
	evs := make([]Event, 0, 256)
	cmds := []Command{Boost(0)}
	allocs := testing.AllocsPerRun(1000, func() {
		evs = Step(&s, b, 0.1, cmds, evs[:0])
	})
	if allocs > 0 {
		t.Fatalf("Step allocates %v times per call", allocs)
	}
}

func BenchmarkStep(bm *testing.B) {
	b := balance.Default()
	s := NewState(b, 1)
	for i := range 6 {
		s.Businesses[i].Owned = 120
		s.Businesses[i].Manager = true
	}
	Recompute(&s, b)
	evs := make([]Event, 0, 256)
	for bm.Loop() {
		evs = Step(&s, b, 0.1, nil, evs[:0])
	}
}

func BenchmarkOffline24h(bm *testing.B) {
	b, err := balance.Parse(balance.DefaultTOML(), []byte("[offline]\ncap_hours_start = 24\n"))
	if err != nil {
		bm.Fatal(err)
	}
	base := NewState(b, 1)
	for i := range 6 {
		base.Businesses[i].Owned = 120
		base.Businesses[i].Manager = true
	}
	Recompute(&base, b)
	evs := make([]Event, 0, 64)
	for bm.Loop() {
		s := base
		_, evs = Offline(&s, b, 24*3600, evs[:0])
	}
}

func TestOffline24hFast(t *testing.T) {
	r := testing.Benchmark(BenchmarkOffline24h)
	if ms := float64(r.NsPerOp()) / 1e6; ms > 200 {
		t.Fatalf("offline 24h took %.1f ms, budget 200 ms", ms)
	}
}

func TestPrimaryRunningUnaffordableReportsShortfall(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	Step(&s, b, 0, []Command{Primary(0, 1)}, nil)
	evs := Step(&s, b, 0, []Command{Primary(0, 1)}, nil)
	if len(evs) != 1 || evs[0].Kind != EvRejected || evs[0].Reason != RejectCannotAfford || !(evs[0].Amount > 0) {
		t.Fatalf("second Enter while running: %+v", evs)
	}
}

func TestGlobalMilestoneEventThreshold(t *testing.T) {
	b := bal(t)
	s := NewState(b, 1)
	for u := range NumUnlocks {
		s.Unlocks.Set(int(u))
	}
	for i := range N {
		s.Businesses[i].Owned = 30
	}
	s.Businesses[0].Owned = 20
	Recompute(&s, b)
	s.Money = 1e300
	evs := Step(&s, b, 0, []Command{Buy(0, 40)}, nil)
	found := false
	for _, e := range evs {
		if e.Kind == EvMilestoneGlobal {
			found = true
			if e.Index != 25 || e.Amount != 2 {
				t.Fatalf("global event %+v, want threshold 25 x2", e)
			}
		}
	}
	if !found {
		t.Fatalf("no global milestone event: %+v", evs)
	}
}
