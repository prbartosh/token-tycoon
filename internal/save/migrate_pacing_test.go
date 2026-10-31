package save

import (
	"encoding/json"
	"testing"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/sim"
)

func TestMigrateV7(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 77)
	s.Prestige.ResetsL2, s.Prestige.Capital = 2, 40
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["version"] = 7
	delete(raw["prestige"].(map[string]any), "round_mult")
	old, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(old, b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prestige.RoundMult != 1 || got.Derived.RoundMult != 1 || got.Prestige.ResetsL2 != 2 {
		t.Fatalf("round multiplier after migration: %+v %v", got.Prestige, got.Derived.RoundMult)
	}
}

func TestRoundMultSurvivesSave(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 78)
	s.Prestige.ResetsL2, s.Prestige.RoundMult = 1, 3.5e20
	sim.Normalize(&s, b)
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(data, b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prestige.RoundMult != 3.5e20 || got.Derived.RoundMult != 3.5e20 {
		t.Fatalf("round multiplier lost: %v %v", got.Prestige.RoundMult, got.Derived.RoundMult)
	}
}
