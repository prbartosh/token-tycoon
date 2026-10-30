package ui

import (
	"strings"
	"testing"

	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
)

func m5Game(c *Config) {
	m4Game(c)
	s, b := c.State, c.Balance
	s.Lifetime.Total = 1e120
	s.Money = 1e110
	s.Market.Hype = 1.5
	s.Market.UnlockedRegions = append(s.Market.UnlockedRegions, "eu")
	s.AGI.Invested = 3e94
	s.Investors.Bonus = 0.25
	s.Stats.Quarters, s.Stats.QuartersBeaten = 3, 1
	s.Stats.IncidentsSeen, s.Stats.IncidentsOK = 4, 3
	s.Achievements = append(s.Achievements, "first_manager", "agi_unlocked")
	s.Incident = sim.Incident{ID: "hallucination", EndsAtS: 1e9}
	sim.Normalize(s, b)
	sim.Step(s, b, 0.1, nil, nil)
	s.Incident = sim.Incident{ID: "hallucination", EndsAtS: 1e9}
	sim.Normalize(s, b)
}

func TestM5TabsAndModalsRender(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.EN, i18n.PL} {
		for _, sz := range [][2]int{{120, 40}, {80, 24}} {
			h := newHarness(t, lang, sz[0], sz[1], m5Game)
			for _, k := range []string{"1", "2", "3", "4", "5", "6", "7"} {
				h.key(k)
				h.dump(lang.String() + " tab " + k)
				for range 140 {
					h.clean("tab " + k)
					h.key("down")
				}
			}
			h.m.modal = modal{kind: modalEnding}
			h.dump("ending")
			h.clean("ending")
			h.key("enter")
			if h.m.modal.kind != modalNone {
				t.Fatal("ending modal did not close")
			}
			for i := range 2 * len(headlines) {
				h.m.newsIdx = i
				h.m.status = ""
				h.m.s.Incident = sim.Incident{}
				h.clean("news")
			}
		}
	}
}

func TestCatalogCoversM5(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.EN, i18n.PL} {
		c := i18n.New(lang)
		for _, a := range sim.Achievements {
			for _, k := range []string{"ach." + a.ID, "ach." + a.ID + ".desc"} {
				if !c.Has(k) {
					t.Errorf("%s: missing %q", lang, k)
				}
			}
		}
		for _, hl := range headlines {
			if !c.Has(hl.key) {
				t.Errorf("%s: missing %q", lang, hl.key)
			}
		}
	}
	if n := len(headlines) + genericNews; n < 40 {
		t.Fatalf("%d headlines, want at least 40", n)
	}
}

func TestIncidentKeyAndBanner(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m5Game)
	h.contains("Incident: Hallucination at a client", "R respond")
	rep := h.m.s.Market.Reputation
	h.key("r")
	if h.m.s.Incident.ID != "" || h.m.s.Market.Reputation <= rep {
		t.Fatalf("r did not respond: %+v rep %v", h.m.s.Incident, h.m.s.Market.Reputation)
	}
	h.lacks("R respond")
}

func TestRegionAndAGIActions(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m5Game)
	h.m.s.Incident = sim.Incident{}
	h.key("3")
	asia, _ := h.b.RegionIndex("asia")
	h.m.tsel[tabMarket] = tkey{act: actRegion, idx: asia}
	h.contains("REGIONS 2/3", "INVESTORS", "Valuation")
	h.key("enter")
	if !sim.RegionEntered(h.m.s, h.b, asia) {
		t.Fatalf("enter did not enter asia: %q", h.m.status)
	}
	h.m.tsel[tabMarket] = tkey{act: actCompliance, idx: asia}
	h.key("enter")
	if !sim.RegionCompliant(h.m.s, h.b, asia) {
		t.Fatal("compliance not bought")
	}
	h.key("5")
	h.contains("RACE TO AGI 0/5")
	h.m.tsel[tabResearch] = tkey{act: actAGI, idx: 0}
	h.m.s.Money = 1e200
	h.key("enter")
	if h.m.s.AGI.Stage != 1 {
		t.Fatalf("AGI stage %d after invest", h.m.s.AGI.Stage)
	}
	h.m.s.AGI.Stage = len(h.b.AGIStages) - 1
	h.m.s.Money = 1e200
	h.m.tsel[tabResearch] = tkey{act: actAGI, idx: h.m.s.AGI.Stage}
	h.key("enter")
	if h.m.modal.kind != modalEnding || !sim.AGIEnded(h.m.s, h.b) {
		t.Fatalf("ending modal not shown: %v", h.m.modal.kind)
	}
	h.contains("AGI ACHIEVED", "CREDITS")
	h.key("x")
	h.contains("Show the ending", "Superintelligence level 0")
}

func TestAchievementToastAndList(t *testing.T) {
	h := newHarness(t, i18n.PL, 120, 40, nil)
	h.m.s.Money = 100
	h.key("enter")
	h.step(0.5, 3)
	found := false
	for _, ts := range h.m.anim.Toast.Items() {
		found = found || strings.Contains(ts.Text, "Osiągnięcie")
	}
	if !found || !h.m.newTab[tabStats] {
		t.Fatal("no achievement toast or NEW badge")
	}
	h.key("7")
	for range 200 {
		if strings.Contains(h.view(), "OSIĄGNIĘCIA 1/") {
			break
		}
		h.key("down")
	}
	h.contains("OSIĄGNIĘCIA 1/", "[+] Hello, world")
}
