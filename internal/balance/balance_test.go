package balance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	b, err := Parse(defaultTOML, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.ManagerCost(0); got != 60 {
		t.Errorf("tier-1 manager cost = %v, want 60", got)
	}
	if got := b.ManagerCost(1); got != 120*100 {
		t.Errorf("tier-2 manager cost = %v, want 12000", got)
	}
	if b.Start.Money != 0 || b.FreeOwned()[0] != 1 {
		t.Errorf("start = %v / %v, want 0 $ and one free api_wrapper", b.Start.Money, b.FreeOwned())
	}
	if b.UpgradeMask(1) != AllBusinesses {
		t.Errorf("quant_4bit mask = %b", b.UpgradeMask(1))
	}
	if m := b.UpgradeMask(0); !m.Has(0) || !m.Has(1) || m.Has(2) {
		t.Errorf("prompt_caching mask = %b", m)
	}
	if n := len(b.Card); n < 36 || n > 44 {
		t.Errorf("cards = %d, want about 40", n)
	}
	for r := range Rarities {
		if len(b.CardsOfRarity(r)) == 0 {
			t.Errorf("no cards of rarity %s", Rarities[r])
		}
	}
	if i, ok := b.CardIndex("lora_paper"); !ok || b.CardTarget(i) != 2 {
		t.Errorf("lora_paper target index")
	}
	if b.Milestones.GlobalEvery != 100 || len(b.Milestones.GlobalThresholds) != 3 {
		t.Errorf("global milestones = %v every %d", b.Milestones.GlobalThresholds, b.Milestones.GlobalEvery)
	}
}

func TestNoUnlockDayLeft(t *testing.T) {
	if strings.Contains(string(defaultTOML), "\nunlock_day") {
		t.Fatal("balance.toml still contains unlock_day keys (DECISIONS item 8)")
	}
}

func TestOverrideMergesOnlyPresentKeys(t *testing.T) {
	over := []byte(`
[crit]
chance = 0.05

[[business]]
id = "rag_chatbot"
cost = 999

[[upgrade]]
id = "new_thing"
cost = 1
target = ["all"]
effect = "profit"
value = 2
`)
	b, err := Parse(defaultTOML, over)
	if err != nil {
		t.Fatal(err)
	}
	if b.Crit.Chance != 0.05 || b.Crit.Mult != 10 {
		t.Errorf("crit = %+v", b.Crit)
	}
	if b.Businesses[1].Cost != 999 || b.Businesses[1].R != 1.15 || b.Businesses[1].CycleS != 3 {
		t.Errorf("rag_chatbot = %+v", b.Businesses[1])
	}
	if b.Businesses[0].Cost != 10 {
		t.Errorf("api_wrapper changed: %+v", b.Businesses[0])
	}
	if len(b.Upgrades) != 10 {
		t.Errorf("upgrades = %d, want 10 (appended)", len(b.Upgrades))
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name, over, want string
	}{
		{"r<=1", "[[business]]\nid = \"api_wrapper\"\nr = 1.0\n", "business[0](api_wrapper).r: must be > 1"},
		{"unknown key", "[crit]\nchancee = 1\n", "unknown keys"},
		{"bad target", "[[upgrade]]\nid = \"mcp\"\ntarget = [\"nope\"]\n", `unknown business id "nope"`},
		{"bad effect", "[[upgrade]]\nid = \"mcp\"\neffect = \"magic\"\n", `unknown effect "magic"`},
		{"global every", "[milestones]\nglobal_every = 0\n", "milestones.global_every"},
		{"thresholds order", "[milestones]\nspeed_thresholds = [50, 25]\n", "milestones.speed_thresholds"},
		{"crit chance", "[crit]\nchance = 2.0\n", "crit.chance: must be in [0, 1]"},
		{"prestige before infra", "[prestige.layer1]\nunlock_run_revenue = 1e3\n", "infrastructure comes first"},
		{"syntax", "[crit\n", "parse"},
		{"card effect", "[[card]]\nid = \"dropout\"\neffect = \"magic\"\n", `unknown card effect "magic"`},
		{"card reduction sign", "[[card]]\nid = \"layer_norm\"\nvalue = 0.2\n", "cost_all must be in (-1, 0)"},
		{"compute_mult", "[[upgrade]]\nid = \"moe\"\ncompute_mult = 1.5\n", "compute_mult: must be in (0, 1]"},
		{"gen order", "[[gpu_gen]]\ngen = 2\nunlock_lifetime = 1\n", "generations must unlock in order"},
		{"max level", "[cards]\nmax_level = 0\n", "cards.max_level"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(defaultTOML, []byte(c.over))
			if err == nil {
				t.Fatalf("expected error containing %q", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err, c.want)
			}
		})
	}
}

func TestLoadWithOverrideFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadWithOverride(filepath.Join(dir, "missing.toml")); err != nil {
		t.Fatalf("missing override must not fail: %v", err)
	}
	p := filepath.Join(dir, "balance.toml")
	if err := os.WriteFile(p, []byte("[combo]\nstep = 0.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadWithOverride(p)
	if err != nil {
		t.Fatal(err)
	}
	if b.Combo.Step != 0.2 || b.Combo.MaxMult != 5 {
		t.Errorf("combo = %+v", b.Combo)
	}
}
