package save

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/rng"
)

var migrations = map[int]func(raw map[string]any) error{
	1: migrate1to2,
	2: migrate2to3,
	3: migrate3to4,
	4: migrate4to5,
	5: migrate5to6,
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

func migrate2to3(raw map[string]any) error {
	if infra, ok := raw["infra"].(map[string]any); ok {
		racks, _ := infra["racks"].([]any)
		hasRacks := false
		for i, v := range racks {
			n, err := intOf(v)
			if err != nil {
				return fmt.Errorf("infra.racks[%d]: %w", i, err)
			}
			hasRacks = hasRacks || n > 0
		}
		if hasRacks {
			infra["starter_given"] = true
		}
	}
	p, _ := raw["prestige"].(map[string]any)
	if p == nil {
		return nil
	}
	if _, ok := p["run_start_s"]; ok {
		return nil
	}
	resets := 0
	if v, ok := p["resets_l1"]; ok {
		n, err := intOf(v)
		if err != nil {
			return fmt.Errorf("prestige.resets_l1: %w", err)
		}
		resets = n
	}
	p["run_start_s"] = 0
	if resets > 0 {
		if c, ok := raw["clock_s"]; ok {
			p["run_start_s"] = c
		}
	}
	return nil
}

func migrate3to4(raw map[string]any) error {
	if m, ok := raw["market"].(map[string]any); ok {
		delete(m, "price_pressure")
		if _, ok := m["opened_s"]; !ok {
			m["opened_s"] = 0
			if u, _ := raw["unlocks"].([]any); u != nil {
				for _, x := range u {
					if x == "market" {
						if c, ok := raw["clock_s"]; ok {
							m["opened_s"] = c
						}
					}
				}
			}
		}
	}
	t, _ := raw["training"].(map[string]any)
	if t == nil {
		return nil
	}
	q, _ := t["queue"].([]any)
	for i, x := range q {
		j, ok := x.(map[string]any)
		if !ok {
			return fmt.Errorf("training.queue[%d]: not an object", i)
		}
		if _, ok := j["arch_mult"]; !ok {
			j["arch_mult"] = 1
		}
	}
	return nil
}

func migrate4to5(raw map[string]any) error {
	done := []any{}
	switch r := raw["research"].(type) {
	case []any:
		done = r
	case map[string]any:
		if d, ok := r["done"].([]any); ok {
			done = d
		}
	}
	raw["research"] = map[string]any{"done": done, "active": "", "progress": 0}
	if cs, ok := raw["contracts"].([]any); ok {
		for i, x := range cs {
			c, ok := x.(map[string]any)
			if !ok {
				return fmt.Errorf("contracts[%d]: not an object", i)
			}
			c["id"] = contractID(c["id"])
			if _, ok := c["duration_s"]; !ok {
				c["duration_s"] = 0
			}
		}
	}
	rules, _ := raw["rules"].([]any)
	out := []any{}
	for _, x := range rules {
		r, ok := x.(map[string]any)
		if !ok {
			continue
		}
		if nr := legacyRule(r); nr != nil {
			out = append(out, nr)
		}
	}
	raw["rules"] = out
	return nil
}

func contractID(v any) int {
	switch x := v.(type) {
	case string:
		n := 0
		for _, ch := range x {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			}
		}
		return n
	default:
		n, _ := intOf(v)
		return n
	}
}

func legacyRule(r map[string]any) map[string]any {
	conds, ok := r["if"].([]any)
	if !ok || len(conds) == 0 || len(conds) > 2 {
		return nil
	}
	if _, structured := conds[0].(map[string]any); structured {
		return r
	}
	var ifs []any
	for _, c := range conds {
		arr, ok := c.([]any)
		if !ok || len(arr) != 3 {
			return nil
		}
		metric, _ := arr[0].(string)
		op, _ := arr[1].(string)
		val, _ := arr[2].(string)
		if metric != "money" || (op != ">" && op != "<") {
			return nil
		}
		n, rest, ok := strings.Cut(val, "x_cost:")
		if !ok || rest != "best_roi" {
			return nil
		}
		x, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return nil
		}
		ifs = append(ifs, map[string]any{"metric": "money_x_best", "op": op, "value": x})
	}
	then, ok := r["then"].([]any)
	if !ok || len(then) != 3 || then[0] != "buy" || then[1] != "best_roi" {
		return nil
	}
	qty, err := intOf(then[2])
	if err != nil || qty < 1 {
		return nil
	}
	enabled, _ := r["enabled"].(bool)
	return map[string]any{
		"if":      ifs,
		"then":    map[string]any{"action": "buy_best", "index": 0, "qty": qty, "value": 0},
		"enabled": enabled,
	}
}

func migrate5to6(raw map[string]any) error {
	streams, _ := raw["rng"].(map[string]any)
	if streams == nil {
		streams = map[string]any{}
		raw["rng"] = streams
	}
	if _, ok := streams["safety"]; ok {
		return nil
	}
	seed, err := uint64Of(raw["seed"])
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	st := rng.NewStream(seed, rng.KeySafety)
	data, err := st.MarshalJSON()
	if err != nil {
		return err
	}
	var h string
	if err := json.Unmarshal(data, &h); err != nil {
		return err
	}
	streams["safety"] = h
	return nil
}
