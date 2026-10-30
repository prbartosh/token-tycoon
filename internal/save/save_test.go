package save

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"token-tycoon/internal/balance"
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
