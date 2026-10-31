package save

import "fmt"

const pacingMigrationFrom = 7

func init() {
	if _, taken := migrations[pacingMigrationFrom]; taken {
		panic(fmt.Sprintf("save: migration from v%d registered twice", pacingMigrationFrom))
	}
	migrations[pacingMigrationFrom] = migrate7to8
}

func migrate7to8(raw map[string]any) error {
	pr, _ := raw["prestige"].(map[string]any)
	if pr == nil {
		pr = map[string]any{}
		raw["prestige"] = pr
	}
	if m, ok := pr["round_mult"].(float64); !ok || !(m >= 1) {
		pr["round_mult"] = 1.0
	}
	return nil
}
