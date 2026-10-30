// Package save stores the game state as JSON: atomic writes, rotating
// backups, versioned format with a migration chain, and fallback to backups
// when the main file is corrupt.
package save

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/paths"
	"token-tycoon/internal/sim"
)

const CurrentVersion = 4

const FileName = "save.json"

var (
	ErrNoSave       = errors.New("save: no save file")
	ErrNewerVersion = errors.New("save: file is from a newer version of the game")
)

func DefaultPath() (string, error) {
	dir, err := paths.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

type file struct {
	Version    int            `json:"version"`
	SavedAt    time.Time      `json:"saved_at"`
	Businesses []sim.Business `json:"businesses"`
	Upgrades   []string       `json:"upgrades"`
	Unlocks    []string       `json:"unlocks"`
	*sim.State
}

type Options struct {
	Backups int
	Indent  bool
}

func Marshal(s *sim.State, b *balance.Balance, savedAt time.Time, indent bool) ([]byte, error) {
	f := file{Version: CurrentVersion, SavedAt: savedAt.UTC(), State: s}
	f.Businesses = make([]sim.Business, 0, sim.N)
	for i := range s.Businesses {
		f.Businesses = append(f.Businesses, s.Businesses[i])
	}
	f.Upgrades = []string{}
	for i, u := range b.Upgrades {
		if s.Upgrades.Has(i) {
			f.Upgrades = append(f.Upgrades, u.ID)
		}
	}
	f.Unlocks = []string{}
	for u := sim.Unlock(0); u < sim.NumUnlocks; u++ {
		if s.IsUnlocked(u) {
			f.Unlocks = append(f.Unlocks, u.Name())
		}
	}
	if indent {
		return json.MarshalIndent(f, "", "  ")
	}
	return json.Marshal(f)
}

func Unmarshal(data []byte, b *balance.Balance) (*sim.State, time.Time, error) {
	var raw map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, time.Time{}, fmt.Errorf("save: parse: %w", err)
	}
	if err := migrate(raw); err != nil {
		return nil, time.Time{}, err
	}
	migrated, err := json.Marshal(raw)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("save: re-encode: %w", err)
	}
	s := &sim.State{}
	f := file{State: s}
	d2 := json.NewDecoder(bytes.NewReader(migrated))
	d2.DisallowUnknownFields()
	if err := d2.Decode(&f); err != nil {
		return nil, time.Time{}, fmt.Errorf("save: decode v%d: %w", CurrentVersion, err)
	}
	for _, bz := range f.Businesses {
		i, ok := b.BusinessIndex(bz.ID)
		if !ok {
			return nil, time.Time{}, fmt.Errorf("save: unknown business %q", bz.ID)
		}
		s.Businesses[i] = bz
	}
	for _, id := range f.Upgrades {
		i, ok := b.UpgradeIndex(id)
		if !ok {
			return nil, time.Time{}, fmt.Errorf("save: unknown upgrade %q", id)
		}
		s.Upgrades.Set(i)
	}
	for _, name := range f.Unlocks {
		u, ok := sim.ParseUnlock(name)
		if !ok {
			return nil, time.Time{}, fmt.Errorf("save: unknown unlock %q", name)
		}
		s.Unlocks.Set(int(u))
	}
	if err := check(s); err != nil {
		return nil, time.Time{}, err
	}
	sim.Normalize(s, b)
	return s, f.SavedAt, nil
}

func check(s *sim.State) error {
	bad := func(x float64) bool { return math.IsNaN(x) || math.IsInf(x, 0) || x < 0 }
	switch {
	case bad(s.Money):
		return fmt.Errorf("save: invalid money %v", s.Money)
	case bad(s.Lifetime.Run) || bad(s.Lifetime.Total) || bad(s.Lifetime.Layer2Total):
		return errors.New("save: invalid lifetime revenue")
	case bad(s.Clock):
		return fmt.Errorf("save: invalid clock %v", s.Clock)
	}
	for _, bz := range s.Businesses {
		if bz.Owned < 0 || bad(bz.Progress) {
			return fmt.Errorf("save: invalid business %q", bz.ID)
		}
	}
	return nil
}

func Save(path string, s *sim.State, b *balance.Balance, savedAt time.Time, o Options) error {
	data, err := Marshal(s, b, savedAt, o.Indent)
	if err != nil {
		return fmt.Errorf("save: marshal: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	tmp := path + ".tmp"
	if err := writeSync(tmp, data); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("save: write: %w", err)
	}
	if err := rotate(path, o.Backups); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("save: rotate backups: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("save: rename: %w", err)
	}
	syncDir(dir)
	return nil
}

func writeSync(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
}

func backupName(path string, n int) string { return path + "." + strconv.Itoa(n) }

func rotate(path string, backups int) error {
	if backups <= 0 {
		return nil
	}
	for n := backups - 1; n >= 1; n-- {
		if err := renameIfExists(backupName(path, n), backupName(path, n+1)); err != nil {
			return err
		}
	}
	b1 := backupName(path, 1)
	if err := os.Remove(b1); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	err := os.Link(path, b1)
	switch {
	case err == nil, errors.Is(err, fs.ErrNotExist):
		return nil
	}
	return renameIfExists(path, b1)
}

func renameIfExists(from, to string) error {
	err := os.Rename(from, to)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

type Loaded struct {
	State   *sim.State
	SavedAt time.Time
	Source  string
	Skipped []error
}

func (l *Loaded) Recovered() bool { return len(l.Skipped) > 0 }

func Load(path string, b *balance.Balance, backups int) (*Loaded, error) {
	cands := []string{path}
	for n := 1; n <= backups; n++ {
		cands = append(cands, backupName(path, n))
	}
	l := &Loaded{}
	found := false
	for _, c := range cands {
		data, err := os.ReadFile(c)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		found = true
		if err != nil {
			l.Skipped = append(l.Skipped, fmt.Errorf("%s: %w", c, err))
			continue
		}
		s, at, err := Unmarshal(data, b)
		if errors.Is(err, ErrNewerVersion) {
			return nil, fmt.Errorf("%s: %w", c, err)
		}
		if err != nil {
			l.Skipped = append(l.Skipped, fmt.Errorf("%s: %w", c, err))
			continue
		}
		l.State, l.SavedAt, l.Source = s, at, c
		return l, nil
	}
	if !found {
		return nil, ErrNoSave
	}
	return nil, fmt.Errorf("save: every save and backup is unreadable: %w", errors.Join(l.Skipped...))
}
