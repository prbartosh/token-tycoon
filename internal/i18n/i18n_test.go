package i18n

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"token-tycoon/internal/balance"
)

func TestCatalogsHaveIdenticalKeys(t *testing.T) {
	pk, ek := Keys(PL), Keys(EN)
	for _, k := range pk {
		if _, ok := en[k]; !ok {
			t.Errorf("key %q missing in EN", k)
		}
	}
	for _, k := range ek {
		if _, ok := pl[k]; !ok {
			t.Errorf("key %q missing in PL", k)
		}
	}
}

var verb = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)

func TestFormatVerbsMatch(t *testing.T) {
	for k, p := range pl {
		pv, ev := verb.FindAllString(p, -1), verb.FindAllString(en[k], -1)
		if !slices.Equal(pv, ev) {
			t.Errorf("key %q: PL verbs %v, EN verbs %v", k, pv, ev)
		}
	}
}

func TestNoLongDashes(t *testing.T) {
	for _, m := range []map[string]string{pl, en} {
		for k, v := range m {
			if strings.ContainsAny(v, "–—") {
				t.Errorf("key %q contains a long dash", k)
			}
		}
	}
}

func TestBalanceIDsTranslated(t *testing.T) {
	b := balance.Default()
	var keys []string
	for _, x := range b.Businesses {
		keys = append(keys, BusinessKey(x.ID), BusinessDescKey(x.ID), ManagerKey(x.ID), ManagerLineKey(x.ID))
	}
	for _, x := range b.Upgrades {
		keys = append(keys, UpgradeKey(x.ID), UpgradeDescKey(x.ID))
	}
	for _, x := range b.Card {
		keys = append(keys, CardKey(x.ID), RarityKey(x.Rarity))
	}
	for _, x := range b.Event {
		keys = append(keys, EventKey(x.ID), EventDescKey(x.ID))
	}
	for _, x := range b.Rivals {
		keys = append(keys, RivalKey(x.ID))
	}
	for _, x := range b.Regions {
		keys = append(keys, RegionKey(x.ID))
	}
	for _, x := range b.Prestige.Layer2.Rounds {
		keys = append(keys, RoundKey(x))
	}
	for _, x := range b.TalentClasses {
		keys = append(keys, TalentKey(x.ID))
	}
	for _, x := range b.AGIStages {
		keys = append(keys, AGIKey(x.ID))
	}
	for _, x := range b.TalentTraits {
		keys = append(keys, "talent.trait."+x.ID)
	}
	for _, x := range balance.Branches {
		keys = append(keys, "research.branch."+x)
	}
	for _, x := range b.ResearchNodes {
		keys = append(keys, "research."+x.ID, "researchfx."+x.Effect)
	}
	for _, x := range b.Regions {
		keys = append(keys, "region.entry."+x.Entry)
	}
	for _, x := range b.Incidents {
		keys = append(keys, "incident."+x.ID)
	}
	for _, x := range b.AGIStages {
		keys = append(keys, "agi.fx."+x.Effect)
	}
	c := New(PL)
	for _, k := range keys {
		if !c.Has(k) {
			t.Errorf("balance id has no translation: %q", k)
		}
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		flag, setting, env string
		want               Lang
	}{
		{"pl", "", "en_US.UTF-8", PL},
		{"en", "pl", "pl_PL.UTF-8", EN},
		{"", "pl", "en_US", PL},
		{"", "", "pl_PL.UTF-8", PL},
		{"", "", "de_DE.UTF-8", EN},
		{"", "", "", EN},
	}
	for _, c := range cases {
		t.Setenv("LANG", c.env)
		if got := Detect(c.flag, c.setting); got != c.want {
			t.Errorf("Detect(%q,%q) with LANG=%q = %v, want %v", c.flag, c.setting, c.env, got, c.want)
		}
	}
	if New(EN).T("nope") != "!nope" {
		t.Error("missing key rendering")
	}
	if New(PL).T("msg.crit", "10") != "TOKEN BURST! x10" {
		t.Error("formatting")
	}
}
