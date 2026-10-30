package balance

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"token-tycoon/internal/paths"
)

//go:embed balance.toml
var defaultTOML []byte

// OverrideFileName is the name of the optional local override file inside the
// config directory.
const OverrideFileName = "balance.toml"

// Default returns the embedded balance. It panics if the embedded file is
// invalid, which tests guarantee never happens.
func Default() *Balance {
	b, err := Parse(defaultTOML, nil)
	if err != nil {
		panic("balance: embedded balance.toml is invalid: " + err.Error())
	}
	return b
}

// DefaultTOML returns a copy of the embedded file.
func DefaultTOML() []byte { return bytes.Clone(defaultTOML) }

// Load returns the embedded balance merged with the local override at
// ~/.config/token-tycoon/balance.toml, if that file exists.
func Load() (*Balance, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return Parse(defaultTOML, nil)
	}
	return LoadWithOverride(filepath.Join(dir, OverrideFileName))
}

// LoadWithOverride merges the embedded balance with the file at path. A missing
// file is not an error; an empty path means no override.
func LoadWithOverride(path string) (*Balance, error) {
	if path == "" {
		return Parse(defaultTOML, nil)
	}
	over, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Parse(defaultTOML, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("balance: read override %s: %w", path, err)
	}
	b, err := Parse(defaultTOML, over)
	if err != nil {
		return nil, fmt.Errorf("balance: override %s: %w", path, err)
	}
	return b, nil
}

// Parse decodes base, merges override (may be nil) on top and validates the
// result. Override semantics: tables merge key by key; arrays of tables whose
// elements carry an "id" (or "gen") key merge element by element, unknown ids
// are appended; any other value, including plain arrays, replaces the base.
func Parse(base, override []byte) (*Balance, error) {
	doc := base
	if len(override) > 0 {
		var bm, om map[string]any
		if err := toml.Unmarshal(base, &bm); err != nil {
			return nil, fmt.Errorf("base: %w", err)
		}
		if err := toml.Unmarshal(override, &om); err != nil {
			return nil, fmt.Errorf("parse: %w", err)
		}
		mergeTables(bm, om)
		merged, err := toml.Marshal(bm)
		if err != nil {
			return nil, fmt.Errorf("re-encode merged balance: %w", err)
		}
		doc = merged
	}
	var b Balance
	dec := toml.NewDecoder(bytes.NewReader(doc)).DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		var sm *toml.StrictMissingError
		if errors.As(err, &sm) {
			return nil, fmt.Errorf("unknown keys:\n%s", sm.String())
		}
		var de *toml.DecodeError
		if errors.As(err, &de) {
			row, col := de.Position()
			return nil, fmt.Errorf("decode error at line %d col %d: %s", row, col, de.Error())
		}
		return nil, err
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	b.buildIndex()
	return &b, nil
}

func mergeTables(dst, src map[string]any) {
	for k, sv := range src {
		dv, ok := dst[k]
		if !ok {
			dst[k] = sv
			continue
		}
		switch s := sv.(type) {
		case map[string]any:
			if d, ok := dv.(map[string]any); ok {
				mergeTables(d, s)
				continue
			}
		case []any:
			if d, ok := dv.([]any); ok {
				if merged, ok := mergeArrayOfTables(d, s); ok {
					dst[k] = merged
					continue
				}
			}
		}
		dst[k] = sv
	}
}

// mergeArrayOfTables merges src into dst by "id" or "gen" if every element of
// both arrays is a table carrying that key. It reports false otherwise.
func mergeArrayOfTables(dst, src []any) ([]any, bool) {
	key := ""
	for _, cand := range []string{"id", "gen"} {
		if allTablesHave(dst, cand) && allTablesHave(src, cand) {
			key = cand
			break
		}
	}
	if key == "" || len(dst) == 0 {
		return nil, false
	}
	out := append([]any(nil), dst...)
	for _, se := range src {
		sm := se.(map[string]any)
		found := false
		for _, de := range out {
			dm := de.(map[string]any)
			if fmt.Sprint(dm[key]) == fmt.Sprint(sm[key]) {
				mergeTables(dm, sm)
				found = true
				break
			}
		}
		if !found {
			out = append(out, sm)
		}
	}
	return out, true
}

func allTablesHave(arr []any, key string) bool {
	for _, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			return false
		}
		if _, ok := m[key]; !ok {
			return false
		}
	}
	return true
}
