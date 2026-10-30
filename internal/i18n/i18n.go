// Package i18n holds the Polish and English catalogs. Every user-facing string
// of the game goes through it: business, upgrade, card and event names,
// messages, help and simulator reports.
package i18n

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Lang is a supported language.
type Lang uint8

const (
	EN Lang = iota
	PL
)

// String returns the language code.
func (l Lang) String() string {
	if l == PL {
		return "pl"
	}
	return "en"
}

// Parse maps "pl"/"en" (any case, region suffixes allowed) to a Lang.
func Parse(s string) (Lang, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(s, "pl"):
		return PL, true
	case strings.HasPrefix(s, "en"):
		return EN, true
	}
	return EN, false
}

// Detect picks the language: the --lang flag if set, else the saved setting,
// else the LANG environment variable (pl* = PL, anything else EN).
func Detect(flagValue, setting string) Lang {
	if l, ok := Parse(flagValue); ok {
		return l
	}
	if l, ok := Parse(setting); ok {
		return l
	}
	return FromEnv(os.Getenv("LANG"))
}

// FromEnv maps a LANG value such as "pl_PL.UTF-8" to a Lang.
func FromEnv(lang string) Lang {
	if strings.HasPrefix(strings.ToLower(lang), "pl") {
		return PL
	}
	return EN
}

// Catalog translates keys for one language.
type Catalog struct {
	lang Lang
	m    map[string]string
}

// New returns the catalog for a language.
func New(l Lang) *Catalog {
	if l == PL {
		return &Catalog{lang: PL, m: pl}
	}
	return &Catalog{lang: EN, m: en}
}

// Lang returns the catalog's language.
func (c *Catalog) Lang() Lang { return c.lang }

// T returns the translation of key, formatted with args (fmt verbs) when args
// are given. A missing key renders as "!key" so it is visible but harmless.
func (c *Catalog) T(key string, args ...any) string {
	s, ok := c.m[key]
	if !ok {
		return "!" + key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// Has reports whether key exists.
func (c *Catalog) Has(key string) bool {
	_, ok := c.m[key]
	return ok
}

// Keys returns the sorted keys of a language's catalog.
func Keys(l Lang) []string {
	m := en
	if l == PL {
		m = pl
	}
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// Helpers building the conventional keys for balance ids.
func BusinessKey(id string) string     { return "business." + id }
func BusinessDescKey(id string) string { return "business." + id + ".desc" }
func ManagerKey(id string) string      { return "manager." + id }
func ManagerLineKey(id string) string  { return "manager." + id + ".line" }
func UpgradeKey(id string) string      { return "upgrade." + id }
func UpgradeDescKey(id string) string  { return "upgrade." + id + ".desc" }
func CardKey(id string) string         { return "card." + id }
func EventKey(id string) string        { return "event." + id }
func EventDescKey(id string) string    { return "event." + id + ".desc" }
func RivalKey(id string) string        { return "rival." + id }
func RegionKey(id string) string       { return "region." + id }
func RoundKey(id string) string        { return "round." + id }
func TalentKey(id string) string       { return "talent." + id }
func AGIKey(id string) string          { return "agi." + id }
func UnlockKey(id string) string       { return "unlock." + id }
func RarityKey(id string) string       { return "rarity." + id }
