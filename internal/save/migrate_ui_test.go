package save

import (
	"encoding/json"
	"reflect"
	"testing"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/sim"
)

func TestMigrateV6(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 66)
	s.Training.Queue = []sim.TrainJob{{Params: 1e7, Tokens: 2e8, ClusterShare: 0.5, ArchMult: 1}}
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["version"] = 6
	tr := raw["training"].(map[string]any)
	delete(tr, "archive")
	delete(tr, "pending_releases")
	delete(tr["queue"].([]any)[0].(map[string]any), "manual")
	raw["settings"].(map[string]any)["notation"] = ""
	old, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(old, b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Training.Archive == nil || got.Training.Pending == nil || len(got.Training.Archive) != 0 {
		t.Fatalf("archive fields: %+v", got.Training)
	}
	if len(got.Training.Queue) != 1 || !got.Training.Queue[0].Manual {
		t.Fatalf("queued job must ask the player: %+v", got.Training.Queue)
	}
	if got.Settings.Notation != "letters" {
		t.Fatalf("notation %q", got.Settings.Notation)
	}
}

func TestRoundTripArchiveAndSettings(t *testing.T) {
	b := balance.Default()
	s := sim.NewState(b, 5)
	s.Lifetime.Total = b.Training.UnlockLifetime
	sim.Normalize(&s, b)
	s.Training.Models = []sim.Model{{Name: "TT-1", Benchmark: 20, Segment: "pro"}, {Name: "TT-2", Benchmark: 30}, {Name: "TT-3", Benchmark: 40}}
	s.Training.Pending = []sim.Model{{Name: "TT-5", Benchmark: 55, ReleasedS: 12}}
	s.Training.Archive = []sim.Model{{Name: "TT-4", Benchmark: 10}}
	s.Training.Queue = []sim.TrainJob{{Params: 1e7, Tokens: 2e8, ClusterShare: 0.5, ArchMult: 1, Manual: true}}
	s.Training.Released = 5
	s.Settings = sim.Settings{Lang: "pl", Notation: "scientific", Advisor: false, Colorblind: true, Bell: true}
	data, err := Marshal(&s, b, at, false)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Unmarshal(data, b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Training, s.Training) || got.Settings != s.Settings {
		t.Fatalf("round trip\nwant %+v %+v\ngot  %+v %+v", s.Training, s.Settings, got.Training, got.Settings)
	}
	evs := sim.Step(got, b, 0, []sim.Command{sim.ResolveRelease(0)}, nil)
	if len(got.Training.Pending) != 0 || got.Training.Models[2].Name != "TT-5" || len(got.Training.Archive) != 2 || len(evs) == 0 {
		t.Fatalf("resolve after load: %+v", got.Training)
	}
}
