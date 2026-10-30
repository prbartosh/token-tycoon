package save

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/rng"
	"token-tycoon/internal/sim"
)

var at = time.Date(2026, 10, 30, 14, 22, 5, 0, time.UTC)

func played(b *balance.Balance) sim.State {
	s := sim.NewState(b, 42)
	s.Money = 1e6
	evs := sim.Step(&s, b, 0.1, []sim.Command{sim.Buy(0, 10), sim.HireManager(0), sim.Buy(1, 3), sim.BuyUpgrade(0)}, nil)
	for range 200 {
		evs = sim.Step(&s, b, 0.1, nil, evs[:0])
	}
	s.Cards["dropout"] = 2
	s.Infra.Racks[1] = 3
	s.Lifetime.Total = 2e12
	s.Lifetime.Run = 2e12
	for range 3000 {
		evs = sim.Step(&s, b, 0.1, []sim.Command{sim.React("h"), sim.ClaimGrant("2026-10-30")}, evs[:0])
	}
	sim.Step(&s, b, 0, []sim.Command{sim.BuyRack(1, 1), sim.BuyGrid(5), sim.BuyPPA(2), sim.SetCloud(true)}, nil)
	return s
}

func TestRoundTrip(t *testing.T) {
	b := balance.Default()
	s := played(b)
	path := filepath.Join(t.TempDir(), "save.json")
	if err := Save(path, &s, b, at, Options{Backups: 3}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(path, b, 3)
	if err != nil {
		t.Fatal(err)
	}
	got := *l.State
	if !l.SavedAt.Equal(at) || l.Source != path || l.Recovered() {
		t.Fatalf("meta: %v %s %v", l.SavedAt, l.Source, l.Skipped)
	}
	sim.Normalize(&s, b)
	s.Undo, got.Undo = sim.Undo{}, sim.Undo{}
	if !reflect.DeepEqual(s, got) {
		t.Fatalf("round trip mismatch\nwant %+v\ngot  %+v", s, got)
	}
	a, c := s, got
	for range 1000 {
		sim.Step(&a, b, 0.1, nil, nil)
		sim.Step(&c, b, 0.1, nil, nil)
	}
	if a.Money != c.Money || a.Stats.Crits != c.Stats.Crits {
		t.Fatal("restored game diverged")
	}
}

func TestRotationAndCorruptFallback(t *testing.T) {
	b := balance.Default()
	path := filepath.Join(t.TempDir(), "save.json")
	for i := range 5 {
		s := sim.NewState(b, 1)
		s.Money = float64(i)
		if err := Save(path, &s, b, at, Options{Backups: 3}); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"", ".1", ".2", ".3"} {
		if _, err := os.Stat(path + n); err != nil {
			t.Fatalf("missing %s: %v", path+n, err)
		}
	}
	if _, err := os.Stat(path + ".4"); err == nil {
		t.Fatal("more than 3 backups kept")
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("temp file left behind")
	}
	if err := os.WriteFile(path, []byte(`{"version": 2, "money": `), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := Load(path, b, 3)
	if err != nil {
		t.Fatal(err)
	}
	if l.Source != path+".1" || !l.Recovered() || l.State.Money != 3 {
		t.Fatalf("fallback: source %s money %v", l.Source, l.State.Money)
	}
	if err := os.WriteFile(path+".1", []byte(`{"version":2,"seed":1,"money":-5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err = Load(path, b, 3)
	if err != nil || l.Source != path+".2" || l.State.Money != 2 {
		t.Fatalf("fallback to .2: %v %+v", err, l)
	}
	for _, n := range []string{".2", ".3"} {
		os.WriteFile(path+n, []byte("garbage"), 0o644)
	}
	if _, err := Load(path, b, 3); err == nil || errors.Is(err, ErrNoSave) {
		t.Fatalf("expected unreadable error, got %v", err)
	}
}

func TestNoSaveAndNewerVersion(t *testing.T) {
	b := balance.Default()
	dir := t.TempDir()
	if _, err := Load(filepath.Join(dir, "save.json"), b, 3); !errors.Is(err, ErrNoSave) {
		t.Fatalf("want ErrNoSave, got %v", err)
	}
	p := filepath.Join(dir, "new.json")
	os.WriteFile(p, []byte(`{"version": 99}`), 0o644)
	os.WriteFile(p+".1", []byte(`{"version": 2, "seed": 1}`), 0o644)
	if _, err := Load(p, b, 3); !errors.Is(err, ErrNewerVersion) {
		t.Fatalf("want ErrNewerVersion, got %v", err)
	}
}

func TestMigrateV1Example(t *testing.T) {
	b := balance.Default()
	data, err := os.ReadFile(filepath.Join("testdata", "v1_example.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, savedAt, err := Unmarshal(data, b)
	if err != nil {
		t.Fatal(err)
	}
	if savedAt.IsZero() || s.Seed != 8812734512 || s.Money != 1.284e15 || s.Lifetime.Total != 4.4e21 {
		t.Fatalf("basics: %v %d %v %v", savedAt, s.Seed, s.Money, s.Lifetime.Total)
	}
	if bz := s.Businesses[1]; bz.ID != "rag_chatbot" || bz.Owned != 87 || !bz.Manager {
		t.Fatalf("rag: %+v", bz)
	}
	si, _ := b.UpgradeIndex("spec_decoding")
	if !s.Upgrades.Has(si) || s.Upgrades.Count() != 3 {
		t.Fatalf("upgrades: %v", s.Upgrades)
	}
	if s.Infra.Racks[0] != 14 || s.Infra.Racks[1] != 6 || s.Infra.GPUGen != 4 || !s.Infra.StarterGiven {
		t.Fatalf("racks: %+v", s.Infra)
	}
	if !s.IsUnlocked(sim.UnlockMarket) || !s.IsUnlocked(sim.UnlockTraining) || s.IsUnlocked(sim.UnlockContracts) {
		t.Fatal("unlocks not rebuilt from lifetime revenue")
	}
	if s.Talent[1].Trait != "perfectionist" || s.Prestige.Params != 412 || s.Cards["dropout"] != 3 || s.Prestige.RunStartS != s.Clock {
		t.Fatalf("misc: %+v %+v", s.Talent, s.Prestige)
	}
	if len(s.Rules) != 1 || !s.Rules[0].Enabled || len(s.Training.Queue[0].Arch) != 1 {
		t.Fatalf("rules/training: %+v %+v", s.Rules, s.Training)
	}
	s2, _, _ := Unmarshal(data, b)
	if !reflect.DeepEqual(s.RNG, s2.RNG) {
		t.Fatal("migrated RNG differs between loads")
	}
	out, err := Marshal(s, b, savedAt, false)
	if err != nil {
		t.Fatal(err)
	}
	s3, _, err := Unmarshal(out, b)
	if err != nil {
		t.Fatal(err)
	}
	if s3.Money != s.Money || s3.Businesses != s.Businesses {
		t.Fatal("v2 re-save mismatch")
	}
}

func TestLoadedGameContinuesIdentically(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 7)
	s.Money = 1e9
	sim.Step(&s, b, 0, []sim.Command{sim.Buy(0, 120), sim.HireManager(0), sim.Buy(1, 60), sim.HireManager(1)}, nil)
	for range 37 {
		sim.Step(&s, b, 0.1, nil, nil)
	}
	if bz := s.Businesses[0]; bz.DrawBudget == b.BusinessDefaults.DrawCapPerS {
		t.Fatalf("test needs a partially used budget, got %v", bz.DrawBudget)
	}
	path := filepath.Join(t.TempDir(), "save.json")
	if err := Save(path, &s, b, at, Options{Backups: 1}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(path, b, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, c := s, *l.State
	for range 500 {
		sim.Step(&a, b, 0.1, nil, nil)
		sim.Step(&c, b, 0.1, nil, nil)
	}
	if a.Money != c.Money || a.Stats.Crits != c.Stats.Crits || a.RNG != c.RNG {
		t.Fatalf("loaded game diverged: money %v vs %v, crits %d vs %d", a.Money, c.Money, a.Stats.Crits, c.Stats.Crits)
	}
}

func TestRotationKeepsPreviousAsBackup(t *testing.T) {
	b := balance.Default()
	path := filepath.Join(t.TempDir(), "save.json")
	for i := range 2 {
		s := sim.NewState(b, 1)
		s.Money = float64(i + 1)
		if err := Save(path, &s, b, at, Options{Backups: 3}); err != nil {
			t.Fatal(err)
		}
	}
	main, err := Load(path, b, 0)
	if err != nil || main.State.Money != 2 {
		t.Fatalf("main: %v %+v", err, main)
	}
	data, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	prev, _, err := Unmarshal(data, b)
	if err != nil || prev.Money != 1 {
		t.Fatalf("backup .1: %v %v", err, prev)
	}
}

func TestMigrateV2(t *testing.T) {
	b := balance.Default()
	v2 := `{"version":2,"saved_at":"2026-10-30T10:00:00Z","seed":7,"clock_s":5000,"money":10,
"lifetime":{"run":2e7,"total":2e7,"layer2_total":2e7},
"businesses":[{"id":"api_wrapper","owned":40,"manager":true}],
"upgrades":[],"unlocks":[],
"infra":{"gpu_gen":0,"racks":[0,0,0,0,0,0,0,0],"energy":{"grid_mw":0,"ppa":{"mw":0,"index":0,"expires_s":0},"smr":0,"orbital":0},"cloud_enabled":false,"starter_given":false},
"prestige":{"params":0,"params_lifetime":0,"capital":0,"version_index":0,"round_index":0,"resets_l1":0,"resets_l2":0}}`
	s, _, err := Unmarshal([]byte(v2), b)
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsUnlocked(sim.UnlockInfra) || s.Infra.StarterGiven || s.Prestige.RunStartS != 0 || s.Infra.GPUGen != 1 {
		t.Fatalf("v2 migration: %+v %+v", s.Infra, s.Prestige)
	}
	sim.Step(s, b, 0.1, nil, nil)
	if !s.Infra.StarterGiven || s.Infra.Racks[0] < 1 || s.Derived.Throttle != 1 {
		t.Fatalf("M0 save with infra unlocked did not get a starter cluster: %+v", s.Infra)
	}
}

func TestMigrateV3(t *testing.T) {
	b := balance.Default()
	v3 := `{"version":3,"saved_at":"2026-10-30T10:00:00Z","seed":7,"clock_s":9000,"money":10,
"lifetime":{"run":2e13,"total":2e13,"layer2_total":2e13},
"businesses":[{"id":"api_wrapper","owned":40,"manager":true}],
"upgrades":[],"unlocks":["infra","market"],
"market":{"unlocked_regions":[],"segments":{"free":0,"pro":1,"enterprise":0},"hype":0.5,"reputation":0,"price_pressure":1},
"training":{"models":[],"queue":[{"params":1e8,"tokens":2e9,"cluster_share":0.5,"progress":0.1,"arch":[]}],"data_tokens":0},
"prestige":{"params":0,"params_lifetime":0,"capital":0,"version_index":0,"round_index":0,"resets_l1":0,"resets_l2":0,"run_start_s":0}}`
	s, _, err := Unmarshal([]byte(v3), b)
	if err != nil {
		t.Fatal(err)
	}
	if s.Market.OpenedS != 9000 || s.Market.Hype != 0.5 || s.Training.Queue[0].ArchMult != 1 {
		t.Fatalf("v3 migration: %+v %+v", s.Market, s.Training.Queue)
	}
	if !s.Derived.Market.On || !(s.Derived.Price > 0) {
		t.Fatal("market view after load")
	}
}

func TestRoundTripM3(t *testing.T) {
	b := balance.Default()
	s := played(b)
	s.Unlocks.Set(int(sim.UnlockMarket))
	s.Market.OpenedS = s.Clock
	s.Market.Segments = sim.Segments{Free: 0.2, Pro: 0.8}
	s.Market.Hype = 0.7
	s.Training = sim.Training{
		Models:   []sim.Model{{Name: "TT-1", Params: 1e9, Tokens: 2e10, Benchmark: 61, ReleasedS: 10, Segment: "pro"}},
		Queue:    []sim.TrainJob{{Params: 1e8, Tokens: 2e9, ClusterShare: 0.4, Progress: 0.3, Arch: []string{"moe"}, ArchMult: 0.85, Synthetic: 0.2}},
		Scrapers: 3, Licenses: 2, Released: 1, DataTokens: 5e9, SynthTokens: 1e8, SynthShare: 0.1,
	}
	sim.Normalize(&s, b)
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(data, b)
	if err != nil {
		t.Fatal(err)
	}
	s.Undo, got.Undo = sim.Undo{}, sim.Undo{}
	if !reflect.DeepEqual(s, *got) {
		t.Fatalf("round trip mismatch\nwant %+v\ngot  %+v", s.Training, got.Training)
	}
}

func TestRoundTripAllSystemsContinues(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 99)
	s.Lifetime.Total, s.Lifetime.Run, s.Money = 1e20, 1e14, 1e19
	for i := range s.Businesses {
		s.Businesses[i].Owned, s.Businesses[i].Manager = 150, true
	}
	s.Cards["dropout"], s.Cards["chinchilla"] = 3, 1
	sim.Normalize(&s, b)
	evs := sim.Step(&s, b, 0.1, nil, nil)
	cmds := []sim.Command{
		sim.BuyRack(2, 5), sim.BuyGrid(100), sim.BuyPPA(50), sim.BuySMR(), sim.BuyScraper(4), sim.BuyLicense(),
		sim.SetSegments(0.1, 0.9, 0), sim.ClaimGrant("2026-10-29"),
	}
	evs = sim.Step(&s, b, 0, cmds, evs[:0])
	evs = sim.Step(&s, b, 0, []sim.Command{sim.Train(1e8, 2e9, 0.4)}, evs[:0])
	for range 600 {
		evs = sim.Step(&s, b, 1, nil, evs[:0])
	}
	evs = sim.Step(&s, b, 0, []sim.Command{sim.PrestigeReset(), sim.ClaimGrant("2026-10-30")}, evs[:0])
	evs = sim.Step(&s, b, 0, []sim.Command{sim.Train(1e7, 2e8, 0.2), sim.AssignModel(0, "free")}, evs[:0])
	for range 50 {
		evs = sim.Step(&s, b, 0.1, nil, evs[:0])
	}
	if s.Prestige.ResetsL1 != 1 || len(s.Training.Models) == 0 || s.Streak.Day != 2 || s.Infra.Energy.PPA.MW > 0 {
		t.Fatalf("setup: resets %d models %d streak %+v ppa %+v", s.Prestige.ResetsL1, len(s.Training.Models), s.Streak, s.Infra.Energy.PPA)
	}
	s.Lifetime.Total, s.Lifetime.Layer2Total = 1e45, 1e45
	evs = sim.Step(&s, b, 0.1, nil, evs[:0])
	evs = sim.Step(&s, b, 0, []sim.Command{sim.FundingRound()}, evs[:0])
	s.Money = 1e40
	for i := range s.Businesses {
		s.Businesses[i].Owned, s.Businesses[i].Manager = 100, true
	}
	s.Prestige.Capital += 50
	eff1, _ := b.NodeIndex("eff_1")
	rule := sim.Rule{If: [sim.MaxConds]sim.Cond{{Metric: "money_x_best", Op: ">", Value: 5}, {Metric: "gpu_util", Op: "<", Value: 0.9}},
		Then: sim.Action{Kind: "buy_best", Qty: 1}, Enabled: true}
	evs = sim.Step(&s, b, 0, []sim.Command{sim.Hire(0), sim.Hire(2), sim.Research(eff1), sim.AddRule(rule)}, evs[:0])
	for range 2000 {
		evs = sim.Step(&s, b, 1, nil, evs[:0])
	}
	scale1, _ := b.NodeIndex("scale_1")
	evs = sim.Step(&s, b, 0, []sim.Command{sim.Research(scale1)}, evs[:0])
	if len(s.Offers) > 0 {
		s.Offers[0].MinBenchmark = 0
		evs = sim.Step(&s, b, 0, []sim.Command{sim.AcceptContract(0)}, evs[:0])
	}
	s.Poach = sim.Poach{Active: true, Index: 1, EndsAtS: s.Clock + 30}
	s.Talent[0].RaiseUntilS = s.Clock + 1000
	if s.Prestige.ResetsL2 != 1 || len(s.Talent) != 2 || len(s.Contracts) != 1 || s.Research.Active < 0 || !s.Research.Done.Has(eff1) || len(s.Rules) != 1 || s.Stats.RulesFired == 0 {
		t.Fatalf("M4 setup: l2 %d talent %d contracts %d research %+v rules %+v", s.Prestige.ResetsL2, len(s.Talent), len(s.Contracts), s.Research, s.Rules)
	}
	s.Lifetime.Total = 1e95
	evs = sim.Step(&s, b, 0.1, nil, evs[:0])
	s.Money = 1e100
	eu, _ := b.RegionIndex("eu")
	evs = sim.Step(&s, b, 0, []sim.Command{sim.EnterRegion(eu), sim.BuyCompliance(eu)}, evs[:0])
	s.Money = b.AGIStages[0].Cost / 3
	evs = sim.Step(&s, b, 0, []sim.Command{sim.AGIInvest()}, evs[:0])
	for range 200 {
		evs = sim.Step(&s, b, 1, nil, evs[:0])
	}
	s.Incident = sim.Incident{ID: "data_leak", EndsAtS: s.Clock + 5}
	s.Investors.Bonus = 0.5
	if !s.IsUnlocked(sim.UnlockAGI) || sim.RegionsCompliant(&s, b) != 2 || !(s.AGI.Invested > 0) || !(s.Investors.QuarterEndS > 0) || len(s.Achievements) == 0 {
		t.Fatalf("M5 setup: regions %v agi %+v investors %+v achievements %v", s.Market, s.AGI, s.Investors, s.Achievements)
	}
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(data, b)
	if err != nil {
		t.Fatal(err)
	}
	want := s
	sim.Normalize(&want, b)
	want.Undo, got.Undo = sim.Undo{}, sim.Undo{}
	want.Combo, got.Combo = sim.Combo{Mult: 1}, sim.Combo{Mult: 1}
	if !reflect.DeepEqual(want, *got) {
		t.Fatalf("round trip mismatch\nwant %+v\ngot  %+v", want, *got)
	}
	a, c := want, *got
	var ea, ec []sim.Event
	for range 3000 {
		ea = sim.Step(&a, b, 0.5, nil, ea[:0])
		ec = sim.Step(&c, b, 0.5, nil, ec[:0])
	}
	if a.Money != c.Money || a.RNG != c.RNG || a.Market.Hype != c.Market.Hype || len(a.Training.Models) != len(c.Training.Models) ||
		a.Stats != c.Stats || a.Investors != c.Investors || a.AGI != c.AGI || !reflect.DeepEqual(a.Achievements, c.Achievements) {
		t.Fatalf("restored game diverged: %v vs %v", a.Money, c.Money)
	}
}

func TestMigrateV4(t *testing.T) {
	b := balance.Default()
	v4 := `{"version":4,"saved_at":"2026-10-30T10:00:00Z","seed":7,"clock_s":9000,"money":10,
"lifetime":{"run":2e23,"total":2e23,"layer2_total":2e23},
"businesses":[{"id":"api_wrapper","owned":40,"manager":true}],
"upgrades":[],"unlocks":["infra","market","talent","contracts"],
"talent":[{"class":"star","name":"Ada","trait":"fast"}],
"contracts":[{"id":"c-0007","volume":1e20,"delivered":1e19,"min_benchmark":40,"deadline_s":9500,"reward_minutes":30}],
"research":["eff_1","bogus"],
"rules":[{"if":[["money",">","10x_cost:best_roi"]],"then":["buy","best_roi",10],"enabled":true},{"if":[["hype","<","3"]],"then":["noop"],"enabled":true}],
"prestige":{"params":0,"params_lifetime":0,"capital":3,"version_index":0,"round_index":0,"resets_l1":0,"resets_l2":0,"run_start_s":0}}`
	s, _, err := Unmarshal([]byte(v4), b)
	if err != nil {
		t.Fatal(err)
	}
	eff1, _ := b.NodeIndex("eff_1")
	if !s.Research.Done.Has(eff1) || s.Research.Done.Count() != 1 || s.Research.Active != -1 {
		t.Fatalf("research: %+v", s.Research)
	}
	if len(s.Contracts) != 1 || s.Contracts[0].ID != 7 || s.ContractSeq != 7 {
		t.Fatalf("contracts: %+v", s.Contracts)
	}
	if len(s.Rules) != 1 || s.Rules[0].If[0].Metric != "money_x_best" || s.Rules[0].Then.Qty != 10 || !s.Rules[0].Enabled {
		t.Fatalf("rules: %+v", s.Rules)
	}
	if s.Talent[0].Trait != "fast" || !(s.Derived.Talent.Mult > 1.6) || s.ResMods.ComputeMult != 0.9 {
		t.Fatalf("talent/research effects: %+v %+v", s.Derived.Talent, s.ResMods)
	}
}

func TestMigrateV5(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 77)
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["version"] = 5
	delete(raw["rng"].(map[string]any), "safety")
	for _, k := range []string{"incident", "investors"} {
		delete(raw, k)
	}
	delete(raw["market"].(map[string]any), "compliance")
	raw["achievements"] = []any{"first_manager", "first_crit", "gone"}
	raw["agi"] = map[string]any{"stage": 0, "invested": 0}
	old, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(old, b)
	if err != nil {
		t.Fatal(err)
	}
	if got.RNG.Safety != rng.NewStream(77, rng.KeySafety) {
		t.Fatal("safety stream not derived from the seed")
	}
	if !reflect.DeepEqual(got.Achievements, []string{"first_manager", "first_crit"}) || got.Achieved.Count() != 2 {
		t.Fatalf("achievements: %v", got.Achievements)
	}
	if got.Market.Compliance == nil || got.Incident.ID != "" || got.Investors.QuarterEndS != 0 {
		t.Fatalf("new fields: %+v %+v", got.Market, got.Investors)
	}
}
