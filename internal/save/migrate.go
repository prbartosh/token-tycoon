package save

import (
	"encoding/json"
	"fmt"
	"strconv"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/rng"
)

var migrations = map[int]func(raw map[string]any) error{
	1: migrate1to2,
}

func migrate(raw map[string]any) error {
	v, err := intOf(raw["version"])
	if err != nil {
		return fmt.Errorf("save: version: %w", err)
	}
	if v > CurrentVersion {
		return fmt.Errorf("%w (file v%d, this build v%d)", ErrNewerVersion, v, CurrentVersion)
	}
	if v < 1 {
		return fmt.Errorf("save: unsupported version %d", v)
	}
	for ; v < CurrentVersion; v++ {
		m, ok := migrations[v]
		if !ok {
			return fmt.Errorf("save: no migration from v%d", v)
		}
		if err := m(raw); err != nil {
			return fmt.Errorf("save: migrate v%d->v%d: %w", v, v+1, err)
		}
		raw["version"] = v + 1
	}
	return nil
}

const maxSkip = 10_000_000

var v1Streams = []struct {
	name string
	key  uint64
}{
	{"events", rng.KeyEvents}, {"crits", rng.KeyCrits}, {"cards", rng.KeyCards},
	{"rivals", rng.KeyRivals}, {"energy", rng.KeyEnergy}, {"misc", rng.KeyMisc},
}

func migrate1to2(raw map[string]any) error {
	seed, err := uint64Of(raw["seed"])
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	counters, _ := raw["rng"].(map[string]any)
	streams := map[string]any{}
	for _, st := range v1Streams {
		s := rng.NewStream(seed, st.key)
		if c, ok := counters[st.name]; ok {
			n, err := uint64Of(c)
			if err != nil {
				return fmt.Errorf("rng.%s: %w", st.name, err)
			}
			s.Skip(min(n, maxSkip))
		}
		b, err := s.MarshalJSON()
		if err != nil {
			return err
		}
		var h string
		if err := json.Unmarshal(b, &h); err != nil {
			return err
		}
		streams[st.name] = h
	}
	raw["rng"] = streams

	if infra, ok := raw["infra"].(map[string]any); ok {
		if racks, ok := infra["racks"].(map[string]any); ok {
			arr := make([]any, balance.MaxGPUGens)
			for i := range arr {
				arr[i] = 0
			}
			for k, v := range racks {
				g, err := strconv.Atoi(k)
				if err != nil || g < 1 || g > balance.MaxGPUGens {
					return fmt.Errorf("infra.racks: bad generation %q", k)
				}
				n, err := intOf(v)
				if err != nil {
					return fmt.Errorf("infra.racks.%s: %w", k, err)
				}
				arr[g-1] = n
			}
			infra["racks"] = arr
		}
	}
	if _, ok := raw["unlocks"]; !ok {
		raw["unlocks"] = []any{}
	}
	return nil
}

func intOf(v any) (int, error) {
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			f, ferr := x.Float64()
			if ferr != nil {
				return 0, err
			}
			return int(f), nil
		}
		return int(n), nil
	case float64:
		return int(x), nil
	case int:
		return x, nil
	}
	return 0, fmt.Errorf("not a number: %v", v)
}

func uint64Of(v any) (uint64, error) {
	switch x := v.(type) {
	case json.Number:
		return strconv.ParseUint(x.String(), 10, 64)
	case float64:
		return uint64(x), nil
	}
	return 0, fmt.Errorf("not an unsigned integer: %v", v)
}
