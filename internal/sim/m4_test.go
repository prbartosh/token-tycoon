package sim

import (
	"math"
	"reflect"
	"testing"

	"token-tycoon/internal/balance"
)

func m4State(t testing.TB, b *balance.Balance) State {
	s := m3State(t, b)
	s.Lifetime.Total = b.Rules.UnlockLifetime
	s.Lifetime.Layer2Total = b.Prestige.Layer2.UnlockLifetime
	Step(&s, b, 0.1, nil, nil)
	for _, u := range []Unlock{UnlockTalent, UnlockContracts, UnlockLayer2, UnlockResearch, UnlockRules} {
		if !s.IsUnlocked(u) {
			t.Fatalf("%s not unlocked", u.Name())
		}
	}
	s.Money = 1e45
	return s
}

func node(t testing.TB, b *balance.Balance, id string) int {
	t.Helper()
	i, ok := b.NodeIndex(id)
	if !ok {
		t.Fatalf("no node %q", id)
	}
	return i
}

func research(t testing.TB, s *State, b *balance.Balance, id string) {
	t.Helper()
	s.Prestige.Capital += b.ResearchNodes[node(t, b, id)].Cost
	evs := Step(s, b, 0, []Command{Research(node(t, b, id))}, nil)
	if _, ok := hasEvent(evs, EvResearchStart); !ok {
		t.Fatalf("research %s did not start: %+v", id, evs)
	}
	for k := 0; s.Research.Active >= 0; k++ {
		Step(s, b, 60, nil, nil)
		if k > 10000 {
			t.Fatal("research never finishes")
		}
	}
}

func TestTalentHireSalaryAndTrainingSpeed(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	rate0 := plannedRate(&s, 0.5)
	net0 := s.Derived.NetMult
	cost := HireCost(&s, b, 1)
	s.Money = 3 * cost
	money := s.Money
	evs := Step(&s, b, 0, []Command{Hire(1), Hire(0)}, nil)
	if n := len(s.Talent); n != 2 || s.Talent[0].Class != "senior" || s.Talent[0].Name == "" {
		t.Fatalf("hire: %+v %+v", s.Talent, evs)
	}
	if !(money-s.Money >= cost) {
		t.Fatal("hire fee not paid")
	}
	if got := s.Derived.Talent.Salary; math.Abs(got-0.02) > 1e-12 {
		t.Fatalf("salary share %v", got)
	}
	if math.Abs(s.Derived.NetMult-(net0-0.02)) > 1e-9 {
		t.Fatalf("salary not taken from revenue: %v -> %v", net0, s.Derived.NetMult)
	}
	if r := plannedRate(&s, 0.5) / rate0; math.Abs(r-1.2) > 1e-9 {
		t.Fatalf("talent multiplier on training %v", r)
	}
	if r := ResearchTimeS(&s, b, 0); math.Abs(r-b.ResearchNodes[0].TimeH*3600/1.2) > 1e-6 {
		t.Fatalf("research time %v", r)
	}
	Step(&s, b, 0, []Command{Fire(0)}, nil)
	if len(s.Talent) != 1 || s.Talent[0].Class != "junior" {
		t.Fatal("fire")
	}
}

func TestStarTraitApplies(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	for i := range b.TalentTraits {
		s.Talent = append(s.Talent[:0], Researcher{Class: "star", Name: "X", Trait: b.TalentTraits[i].ID})
		Recompute(&s, b)
		tv := s.Derived.Talent
		tr := b.TalentTraits[i]
		if math.Abs(tv.Mult-(1.4+tr.Speed)) > 1e-12 {
			t.Fatalf("%s speed %v", tr.ID, tv.Mult)
		}
		switch tr.ID {
		case "perfectionist":
			if tv.LossMult != tr.LossMult || ArchMult(&s, b) != tr.LossMult {
				t.Fatal("perfectionist quality")
			}
		case "evangelist":
			if !(tv.HypePerS > 0) || !(s.Derived.Market.HypeGain >= tv.HypePerS) {
				t.Fatal("evangelist hype")
			}
		case "frugal":
			if tv.ComputeMult != tr.ComputeMult || math.Abs(s.Derived.ComputeUnit[0]-b.Businesses[0].Compute*tr.ComputeMult) > 1e-12 {
				t.Fatalf("frugal compute %v", s.Derived.ComputeUnit[0])
			}
		}
	}
	s.Talent = s.Talent[:0]
	Step(&s, b, 0, []Command{Hire(2)}, nil)
	if s.Talent[0].Trait == "" {
		t.Fatal("star hired without a trait")
	}
}

func TestPoachAndCounterOffer(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	s.Talent = append(s.Talent, Researcher{Class: "senior", Name: "A"}, Researcher{Class: "junior", Name: "B"})
	Recompute(&s, b)
	var ev Event
	for k := 0; k < 200000 && !s.Poach.Active; k++ {
		evs := Step(&s, b, 1, nil, nil)
		if e, ok := hasEvent(evs, EvPoach); ok {
			ev = e
		}
	}
	if !s.Poach.Active || ev.Value != b.Talent.PoachWindowS {
		t.Fatal("no poach attempt")
	}
	who := s.Poach.Index
	Step(&s, b, 0, []Command{CounterOffer()}, nil)
	r := s.Talent[who]
	if s.Poach.Active || len(s.Talent) != 2 || math.Abs(r.RaiseUntilS-s.Clock-3*86400) > 1e-6 {
		t.Fatalf("counter-offer: %+v %+v", s.Poach, s.Talent)
	}
	base := 0.005 + 0.015
	cls, _ := b.TalentClassIndex(r.Class)
	if want := base + b.TalentClasses[cls].SalaryShare; math.Abs(s.Derived.Talent.Salary-want) > 1e-12 {
		t.Fatalf("raise not doubled: %v want %v", s.Derived.Talent.Salary, want)
	}
	s.Clock = r.RaiseUntilS + 1
	Recompute(&s, b)
	if math.Abs(s.Derived.Talent.Salary-base) > 1e-12 {
		t.Fatal("raise does not end")
	}
	s.Poach = Poach{Active: true, Index: 1, EndsAtS: s.Clock + 0.5}
	evs := run(&s, b, 1, nil)
	if _, ok := hasEvent(evs, EvPoached); !ok || len(s.Talent) != 1 || s.Stats.Poached != 1 {
		t.Fatal("ignored poach does not lose the person")
	}
	s.Poach = Poach{Active: true, Index: 0, EndsAtS: s.Clock + 30}
	Offline(&s, b, 3600, nil)
	if s.Poach.Active || len(s.Talent) != 1 {
		t.Fatal("poach must not resolve against the player offline")
	}
}

func TestContractsLifecycle(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	for k := 0; k < 1000 && len(s.Offers) < b.Contracts.Offers; k++ {
		Step(&s, b, 10, nil, nil)
	}
	if len(s.Offers) != b.Contracts.Offers {
		t.Fatalf("offers %d", len(s.Offers))
	}
	o := s.Offers[0]
	cc := &b.Contracts
	ratio := o.Volume / (s.Derived.TokensPerSec * o.DurationS)
	if o.DurationS < cc.DurationMin[0]*60 || o.DurationS > cc.DurationMin[1]*60 || ratio < cc.VolumeMult[0]*0.5 || ratio > cc.VolumeMult[1]*2 {
		t.Fatalf("offer %+v ratio %v", o, ratio)
	}
	s.Offers[0].MinBenchmark = 99.5
	evs := Step(&s, b, 0, []Command{AcceptContract(0)}, nil)
	if e, ok := hasEvent(evs, EvRejected); !ok || e.Reason != RejectNeedBenchmark {
		t.Fatal("benchmark requirement not enforced")
	}
	s.Offers[0].MinBenchmark = 0
	s.Offers[0].Volume = s.Derived.TokensPerSec * 5
	s.Offers[1].MinBenchmark = 0
	s.Offers[1].Volume = s.Derived.TokensPerSec * 1e6
	Step(&s, b, 0, []Command{AcceptContract(1), AcceptContract(0), DeclineContract(0)}, nil)
	if len(s.Contracts) != 2 || len(s.Offers) != 0 {
		t.Fatalf("accept/decline: %d active %d offers", len(s.Contracts), len(s.Offers))
	}
	s.Money = 0
	money := s.Money
	evs = run(&s, b, 120, nil)
	done, ok := hasEvent(evs, EvContractDone)
	if !ok || s.Market.Reputation != cc.ReputationGain || !(s.Money > money) || done.Amount <= 0 {
		t.Fatalf("contract not delivered: %+v rep %v", s.Contracts, s.Market.Reputation)
	}
	dl := s.Contracts[0].DeadlineS
	Offline(&s, b, 7200, nil)
	if len(s.Contracts) != 1 || s.Contracts[0].DeadlineS != dl+7200 {
		t.Fatalf("contract deadline must freeze offline: %+v", s.Contracts)
	}
	s.Contracts[0].DeadlineS = s.Clock + 1
	evs = run(&s, b, 2, nil)
	if _, ok := hasEvent(evs, EvContractFailed); !ok || s.Market.Reputation != 0 || s.Stats.ContractsFailed != 1 {
		t.Fatal("failure only costs reputation")
	}
}

func TestFundingRound(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	s.Lifetime.Layer2Total = 1e45
	s.Prestige.Params = 1e6
	s.Infra.Racks[3] = 50
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 80}}
	pv := Funding(&s, b)
	if want := math.Floor(5 * math.Pow(6, 1.5)); !pv.Allowed || pv.Gain != want || !(pv.Valuation > 0) {
		t.Fatalf("preview %+v want gain %v", pv, want)
	}
	evs := Step(&s, b, 0, []Command{FundingRound()}, nil)
	e, ok := hasEvent(evs, EvFunding)
	if !ok || e.Index != 0 || s.Prestige.Capital != pv.Gain || s.Prestige.RoundIndex != 1 || s.Prestige.ResetsL2 != 1 {
		t.Fatalf("round: %+v", s.Prestige)
	}
	if s.Prestige.Params != 0 || s.Lifetime.Layer2Total != 0 || s.Businesses[3].Owned != 0 || s.Infra.Racks[3] != 0 || len(s.Training.Models) != 1 {
		t.Fatal("funding round reset wrong")
	}
	if !s.IsUnlocked(UnlockResearch) {
		t.Fatal("unlocks must survive")
	}
	if Funding(&s, b).Allowed {
		t.Fatal("second round without new revenue")
	}
	if id, mega := RoundID(b, 0); id != "pre_seed" || mega != 0 {
		t.Fatal("round name")
	}
	if id, mega := RoundID(b, len(b.Prestige.Layer2.Rounds)+1); id != "" || mega != 2 {
		t.Fatal("mega round")
	}
}

func TestResearchEffectsApply(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	evs := Step(&s, b, 0, []Command{Research(node(t, b, "eff_2"))}, nil)
	if e, ok := hasEvent(evs, EvRejected); !ok || e.Reason != RejectNeedResearch {
		t.Fatal("prerequisite not enforced")
	}
	evs = Step(&s, b, 0, []Command{Research(node(t, b, "eff_1"))}, nil)
	if e, ok := hasEvent(evs, EvRejected); !ok || e.Reason != RejectNoCapital {
		t.Fatal("capital not required")
	}
	unit := s.Derived.ComputeUnit[0]
	maxN := s.Derived.Train.MaxParams
	crit := s.Derived.CritChance
	energy := s.Derived.EnergyShare
	research(t, &s, b, "eff_1")
	if math.Abs(s.Derived.ComputeUnit[0]-unit*0.9) > 1e-12 {
		t.Fatal("compute research")
	}
	research(t, &s, b, "eff_2")
	research(t, &s, b, "eff_sparsity")
	if energy > 0 && !(s.Derived.EnergyShare < energy) {
		t.Fatal("energy research")
	}
	research(t, &s, b, "scale_1")
	if s.Derived.Train.MaxParams != maxN*10 {
		t.Fatal("scale research")
	}
	research(t, &s, b, "luck_crit_1")
	if math.Abs(s.Derived.CritChance-(b.Crit.Chance+s.CardMods.CritChance+0.01)) > 1e-12 || crit <= 0 {
		t.Fatal("crit research")
	}
	research(t, &s, b, "prod_enterprise")
	if EnterpriseCap(&s, b) != b.Market.Segments.EnterpriseMaxAlloc+0.2 {
		t.Fatal("enterprise research")
	}
	research(t, &s, b, "prod_contracts")
	if ContractSlots(&s, b) != b.Contracts.MaxActive+1 {
		t.Fatal("contract slots")
	}
	research(t, &s, b, "prod_models")
	if MaxModels(&s, b) != b.Training.MaxModels+1 {
		t.Fatal("model limit")
	}
	cap0, eff0 := OfflineCapS(&s, b), OfflineEfficiency(&s, b)
	for _, id := range []string{"auto_offline_1", "auto_offline_cap", "auto_offline_2"} {
		research(t, &s, b, id)
	}
	if OfflineCapS(&s, b) != 24*3600 || OfflineEfficiency(&s, b) != 1 || cap0 >= 24*3600 || eff0 >= 1 {
		t.Fatalf("offline research %v %v", OfflineCapS(&s, b), OfflineEfficiency(&s, b))
	}
	for _, id := range []string{"auto_rules_1", "auto_rules_2", "auto_rules_3"} {
		research(t, &s, b, id)
	}
	if RuleSlots(&s, b) != b.Rules.SlotsMax {
		t.Fatal("rule slots")
	}
	gen := s.Infra.GPUGen
	if gen != 6 {
		t.Fatalf("gen %d before research", gen)
	}
	research(t, &s, b, "scale_2")
	research(t, &s, b, "scale_gpu7")
	if s.Infra.GPUGen != 7 {
		t.Fatalf("gpu gen 7 not unlocked: %d", s.Infra.GPUGen)
	}
	for _, id := range []string{"safe_1", "safe_audit", "luck_cards", "luck_rare"} {
		research(t, &s, b, id)
	}
	if s.ResMods.IncidentMult != 0.5 || s.ResMods.PenaltyMult != 0.5 || s.ResMods.RareCards != 2 || !(s.Derived.CardChance > 0) {
		t.Fatalf("mods %+v", s.ResMods)
	}
	if s.Stats.ResearchDone != 20 {
		t.Fatalf("research done %d", s.Stats.ResearchDone)
	}
}

func TestAutoPrestige(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	research(t, &s, b, "auto_prestige")
	s.Lifetime.Run = 1e20
	evs := run(&s, b, 1.1, nil)
	if _, ok := hasEvent(evs, EvAutoPrestige); !ok || s.Prestige.ResetsL1 != 1 {
		t.Fatal("auto prestige did not fire")
	}
}

func rule(metric, op string, v float64, action string, qty int) Rule {
	return Rule{If: [MaxConds]Cond{{Metric: metric, Op: op, Value: v}}, Then: Action{Kind: action, Qty: qty}, Enabled: true}
}

func TestRulesEngine(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	for k := range Metrics {
		if v := MetricValue(&s, b, Metrics[k].ID); math.IsNaN(v) {
			t.Fatalf("metric %s", Metrics[k].ID)
		}
	}
	add := []Command{
		AddRule(rule("money_x_best", OpGreater, 10, "buy_best", 10)),
		AddRule(rule("gpu_util", OpGreater, 2, "buy_rack", QtyMax)),
		AddRule(Rule{If: [MaxConds]Cond{{Metric: "training_idle", Op: OpGreater, Value: 0.5}, {Metric: "share", Op: OpGreater, Value: 2}},
			Then: Action{Kind: "train"}, Enabled: true}),
		AddRule(rule("nope", OpGreater, 1, "buy_best", 1)),
	}
	evs := Step(&s, b, 0, add, nil)
	if len(s.Rules) != 3 {
		t.Fatalf("rules %d", len(s.Rules))
	}
	if e, ok := hasEvent(evs, EvRejected); !ok || e.Reason != RejectInvalid {
		t.Fatal("invalid rule accepted")
	}
	evs = Step(&s, b, 0, []Command{AddRule(rule("hype", OpGreater, 0, "buy_license", 0))}, evs[:0])
	if e, ok := hasEvent(evs, EvRejected); !ok || e.Reason != RejectTeamFull {
		t.Fatal("slot limit not enforced")
	}
	owned := 0
	for i := range N {
		owned += s.Businesses[i].Owned
	}
	evs = run(&s, b, 1.05, nil)
	fired, ok := hasEvent(evs, EvRuleFired)
	if !ok || fired.Index != 0 || s.Rules[0].Fired != 1 || s.Rules[1].Fired != 0 || s.Rules[2].Fired != 0 {
		t.Fatalf("rule firing: %+v", s.Rules)
	}
	after := 0
	for i := range N {
		after += s.Businesses[i].Owned
	}
	if after != owned+10 {
		t.Fatalf("buy_best x10 bought %d", after-owned)
	}
	Step(&s, b, 0, []Command{ToggleRule(0), MoveRule(2, -1)}, nil)
	if s.Rules[0].Enabled || s.Rules[1].Then.Kind != "train" {
		t.Fatal("toggle / move")
	}
	s.Rules[1].If[1] = Cond{}
	s.Training.DataTokens = 1e30
	evs = run(&s, b, 1.05, nil)
	if _, ok := hasEvent(evs, EvTrainQueued); !ok || len(s.Training.Queue) != 1 {
		t.Fatal("train rule did not queue a job")
	}
	Step(&s, b, 0, []Command{DeleteRule(0)}, nil)
	if len(s.Rules) != 2 {
		t.Fatal("delete")
	}
	s.Rules = append(s.Rules[:0], rule("hype", OpGreater, 0, "buy_best", 1))
	before := s.Stats.RulesFired
	Offline(&s, b, 600, nil)
	if s.Stats.RulesFired != before {
		t.Fatal("rules ran offline without research")
	}
	research(t, &s, b, "auto_rules_1")
	research(t, &s, b, "auto_offline_1")
	research(t, &s, b, "auto_offline_rules")
	before = s.Stats.RulesFired
	Offline(&s, b, 600, nil)
	if s.Stats.RulesFired == before {
		t.Fatal("rules did not run offline with research")
	}
}

func TestRulesEveryAction(t *testing.T) {
	b := bal(t)
	for k := range Actions {
		s := m4State(t, b)
		s.Training.DataTokens = 1e30
		r := rule("hype", OpGreater, 0, Actions[k].ID, 1)
		switch Actions[k].Param {
		case ParamBusinessQty:
			r.Then.Index = 2
		case ParamFree:
			r.Then.Value = 0.2
		}
		if Actions[k].ID == "prestige1" {
			s.Lifetime.Run = 1e20
		}
		if Actions[k].ID == "buy_grid" {
			s.Infra.Energy.GridMW = 0
			Recompute(&s, b)
		}
		Step(&s, b, 0, []Command{AddRule(r)}, nil)
		evs := run(&s, b, 1.05, nil)
		if _, ok := hasEvent(evs, EvRuleFired); !ok {
			t.Errorf("action %s never fired: %+v", Actions[k].ID, evs)
		}
	}
}

func TestStepNoAllocsM4(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 70, ReleasedS: s.Clock}}
	cmds := []Command{Hire(0), Hire(1), Hire(2),
		AddRule(rule("money_x_best", OpGreater, 1e300, "buy_best", 1)),
		AddRule(rule("gpu_util", OpGreater, 5, "buy_rack", 1)),
		AddRule(rule("training_idle", OpLess, 0.5, "set_free", 0))}
	Step(&s, b, 0, cmds, nil)
	s.Prestige.Capital = 1e9
	Step(&s, b, 0, []Command{Research(node(t, b, "eff_1"))}, nil)
	for len(s.Offers) == 0 {
		Step(&s, b, 1, nil, nil)
	}
	s.Offers[0].MinBenchmark = 0
	s.Offers[0].Volume = 1e300
	Step(&s, b, 0, []Command{AcceptContract(0)}, nil)
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

func TestDeterminismM4(t *testing.T) {
	b := bal(t)
	runIt := func() State {
		s := m4State(t, b)
		s.Prestige.Capital = 100
		evs := make([]Event, 0, 64)
		for k := range 20000 {
			var cmds []Command
			switch k {
			case 0:
				cmds = []Command{Hire(2), Hire(1), Research(0), AddRule(rule("money_x_best", OpGreater, 5, "buy_best", 1))}
			case 3000:
				if len(s.Offers) > 0 {
					cmds = []Command{AcceptContract(0)}
				}
			}
			evs = Step(&s, b, 0.5, cmds, evs[:0])
		}
		return s
	}
	a, c := runIt(), runIt()
	if !reflect.DeepEqual(a, c) {
		t.Fatal("nondeterministic")
	}
	if a.Stats.RulesFired == 0 || a.Stats.ResearchDone == 0 {
		t.Fatalf("systems idle: %+v", a.Stats)
	}
}

func TestNormalizeKeepsResearchModelSlots(t *testing.T) {
	b := balance.Default()
	s := m4State(t, b)
	s.Research.Done.Set(node(t, b, "prod_enterprise"))
	s.Research.Done.Set(node(t, b, "prod_models"))
	want := b.Training.MaxModels + 1
	s.Training.Models = s.Training.Models[:0]
	for k := range want {
		s.Training.Models = append(s.Training.Models, Model{Name: "M" + string(rune('A'+k)), Benchmark: float64(20 + k)})
	}
	s.ResMods = ResearchMods{}
	Normalize(&s, b)
	if len(s.Training.Models) != want || s.Derived.Train.MaxModels != want {
		t.Fatalf("models after normalize: %d (limit %d), want %d", len(s.Training.Models), s.Derived.Train.MaxModels, want)
	}
}

func TestHireCostNotFreeAfterReset(t *testing.T) {
	b := balance.Default()
	s := m4State(t, b)
	before := HireCost(&s, b, 2)
	s.Lifetime.Run = b.Prestige.Layer1.UnlockRunRevenue * 4
	Step(&s, b, 0.1, nil, nil)
	evs := Step(&s, b, 0, []Command{PrestigeReset()}, nil)
	if _, ok := hasEvent(evs, EvPrestige); !ok {
		t.Fatalf("prestige: %+v", evs)
	}
	if after := HireCost(&s, b, 2); !(after > 0) || after < before {
		t.Fatalf("hire cost after reset %v, before %v", after, before)
	}
}

func TestFirstQuarterWaitsForIncome(t *testing.T) {
	b := balance.Default()
	s := NewState(b, 3)
	s.Lifetime.Total = b.Investors.UnlockLifetime
	checkUnlocks(&s, b, nil)
	s.Derived.BaseIncome = 0
	tickInvestors(&s, b, nil)
	if s.Investors.QuarterEndS != 0 || s.Investors.Target != 0 {
		t.Fatalf("quarter started without income: %+v", s.Investors)
	}
	s.Derived.BaseIncome = 100
	tickInvestors(&s, b, nil)
	if !(s.Investors.Target > 0) {
		t.Fatalf("quarter target %+v", s.Investors)
	}
}
