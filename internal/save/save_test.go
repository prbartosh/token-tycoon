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
	// Transient fields are rebuilt on load; compare after normalizing both.
	sim.Normalize(&s, b)
	s.Undo, got.Undo = sim.Undo{}, sim.Undo{}
	if !reflect.DeepEqual(s, got) {
		t.Fatalf("round trip mismatch\nwant %+v\ngot  %+v", s, got)
	}
	// Restored RNG continues the same sequence.
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
	// Corrupt main file: the newest backup is used.
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
	// Semantically invalid file (negative money) also falls back.
	if err := os.WriteFile(path+".1", []byte(`{"version":2,"seed":1,"money":-5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err = Load(path, b, 3)
	if err != nil || l.Source != path+".2" || l.State.Money != 2 {
		t.Fatalf("fallback to .2: %v %+v", err, l)
	}
	// Everything corrupt.
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
	if s.Infra.Racks[0] != 14 || s.Infra.Racks[1] != 6 || s.Infra.GPUGen != 2 {
		t.Fatalf("racks: %+v", s.Infra)
	}
	if !s.IsUnlocked(sim.UnlockMarket) || !s.IsUnlocked(sim.UnlockTraining) || s.IsUnlocked(sim.UnlockContracts) {
		t.Fatal("unlocks not rebuilt from lifetime revenue")
	}
	if s.Talent[1].Trait != "perfectionist" || s.Prestige.Params != 412 || s.Cards["dropout"] != 3 {
		t.Fatalf("misc: %+v %+v", s.Talent, s.Prestige)
	}
	if len(s.Rules) != 1 || !s.Rules[0].Enabled || len(s.Training.Queue[0].Arch) != 1 {
		t.Fatalf("rules/training: %+v %+v", s.Rules, s.Training)
	}
	// Migration is deterministic.
	s2, _, _ := Unmarshal(data, b)
	if !reflect.DeepEqual(s.RNG, s2.RNG) {
		t.Fatal("migrated RNG differs between loads")
	}
	// And the result saves as the current version.
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

// A game saved and loaded mid-run continues exactly like the uninterrupted
// one, including the crit draw budget of continuous businesses.
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

// Rotation keeps the main file in place until the new one replaces it.
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
