package save

import "fmt"

const uiMigrationFrom = 6

func init() {
	if _, taken := migrations[uiMigrationFrom]; taken {
		panic(fmt.Sprintf("save: migration from v%d registered twice", uiMigrationFrom))
	}
	migrations[uiMigrationFrom] = migrate6to7
}

func migrate6to7(raw map[string]any) error {
	tr, _ := raw["training"].(map[string]any)
	if tr == nil {
		tr = map[string]any{}
		raw["training"] = tr
	}
	for _, k := range []string{"archive", "pending_releases"} {
		if _, ok := tr[k].([]any); !ok {
			tr[k] = []any{}
		}
	}
	if q, ok := tr["queue"].([]any); ok {
		for _, j := range q {
			if job, ok := j.(map[string]any); ok {
				if _, set := job["manual"]; !set {
					job["manual"] = true
				}
			}
		}
	}
	st, _ := raw["settings"].(map[string]any)
	if st == nil {
		st = map[string]any{}
		raw["settings"] = st
	}
	if n, _ := st["notation"].(string); n == "" {
		st["notation"] = "letters"
	}
	return nil
}
