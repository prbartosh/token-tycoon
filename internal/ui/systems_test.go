package ui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
)

var missingKey = regexp.MustCompile(`![a-z_]+\.[a-z_]`)

func lateGame(c *Config) {
	s := c.State
	s.Lifetime.Total = 1e20
	s.Lifetime.Run = 1e14
	s.Money = 1e19
	for i := range s.Businesses {
		s.Businesses[i].Owned = 120
		s.Businesses[i].Manager = true
	}
	s.Cards["dropout"] = 2
	sim.Normalize(s, c.Balance)
}

func (h *harness) step(dt float64, n int) {
	for range n {
		h.m.evs = sim.Step(h.m.s, h.b, dt, nil, h.m.evs[:0])
		h.m.consume(h.m.evs, false)
	}
	h.m.modal = modal{}
}

func (h *harness) dump(name string) {
	if os.Getenv("UI_DUMP") != "" {
		h.t.Logf("%s\n%s", name, h.view())
	}
}

func (h *harness) fits() {
	h.t.Helper()
	for i, l := range strings.Split(h.view(), "\n") {
		if w := len([]rune(l)); w > h.m.width {
			h.t.Fatalf("line %d is %d wide (> %d): %q", i, w, h.m.width, l)
		}
	}
}

func TestSystemTabsRender(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.EN, i18n.PL} {
		for _, sz := range [][2]int{{120, 40}, {80, 24}} {
			h := newHarness(t, lang, sz[0], sz[1], func(c *Config) {
				for _, cd := range c.Balance.Card {
					c.State.Cards[cd.ID] = 1
				}
				lateGame(c)
			})
			h.step(0.1, 5)
			for _, k := range []string{"1", "2", "3", "4", "5", "6", "7"} {
				h.key(k)
				h.dump(lang.String() + " tab " + k)
				for range 60 {
					h.fits()
					if v := h.view(); missingKey.MatchString(v) || strings.Contains(v, "%!") {
						t.Fatalf("untranslated text on tab %s:\n%s", k, v)
					}
					h.key("down")
				}
			}
		}
	}
}

func TestHeaderShowsInfraAndHype(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.step(0.1, 2)
	v := h.view()
	if strings.Contains(v, "PWR --") || strings.Contains(v, "GPU --") || !strings.Contains(v, "HYPE ") {
		t.Fatalf("header lacks values:\n%s", strings.Split(v, "\n")[0])
	}
}

func TestInfraTabActions(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.step(0.1, 2)
	h.key("4")
	racks := h.m.s.Infra.Racks[1]
	h.key("home", "down", "enter")
	if h.m.s.Infra.Racks[1] != racks+1 {
		t.Fatalf("enter on gen 2 bought %d racks, sel %+v status %q", h.m.s.Infra.Racks[1]-racks, h.m.tabCurrent(), h.m.status)
	}
	h.key("x")
	if h.m.s.Infra.Racks[1] != racks {
		t.Fatalf("x did not sell a rack")
	}
	for h.m.tabCurrent().act != actCloud {
		before := h.m.tabCurrent()
		h.key("down")
		if h.m.tabCurrent() == before {
			t.Fatal("cloud row not reachable")
		}
	}
	h.key("enter")
	if !h.m.s.Infra.CloudEnabled {
		t.Fatal("cloud not toggled on")
	}
	h.key("enter")
	if h.m.s.Infra.CloudEnabled {
		t.Fatal("cloud not toggled off")
	}
	grid := h.m.s.Infra.Energy.GridMW
	for h.m.tabCurrent().act != actGrid {
		h.key("up")
	}
	h.key("enter")
	if !(h.m.s.Infra.Energy.GridMW > grid) {
		t.Fatal("grid not bought")
	}
}

func TestModelsTrainingFlow(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.m.s.Training.DataTokens = 1e15
	h.step(0.1, 2)
	h.key("2")
	h.contains("Start training")
	n0 := h.m.draft.n
	h.key("home", "+")
	if !(h.m.draft.n > n0) {
		t.Fatalf("+ did not grow N: %v -> %v", n0, h.m.draft.n)
	}
	h.key("s", "end")
	for h.m.tabCurrent().act != actTrainStart {
		h.key("up")
	}
	h.key("enter")
	if len(h.m.s.Training.Queue) != 1 {
		t.Fatalf("queue = %d, status %q", len(h.m.s.Training.Queue), h.m.status)
	}
	h.key("down")
	if h.m.tabCurrent().act != actJob {
		t.Fatalf("selection after start = %+v", h.m.tabCurrent())
	}
	h.key("+")
	if h.m.s.Training.Queue[0].ClusterShare <= draftShare {
		t.Fatalf("job share not raised: %v", h.m.s.Training.Queue[0].ClusterShare)
	}
	for range 2000 {
		h.m.evs = sim.Step(h.m.s, h.b, 60, nil, h.m.evs[:0])
		h.m.consume(h.m.evs, false)
		if len(h.m.s.Training.Models) > 0 {
			break
		}
	}
	if len(h.m.s.Training.Models) == 0 {
		t.Fatal("model never released")
	}
	if h.m.modal.kind != modalRelease {
		t.Fatal("no release ceremony")
	}
	h.contains("MODEL RELEASE", h.m.s.Training.Models[0].Name)
	h.dump("release")
	h.key("z")
	if h.m.modal.kind != modalNone {
		t.Fatal("ceremony not dismissed")
	}
	for h.m.tabCurrent().act != actModel {
		h.key("down")
	}
	h.key("enter")
	if h.m.s.Training.Models[0].Segment != "free" {
		t.Fatalf("segment = %q", h.m.s.Training.Models[0].Segment)
	}
	h.dump("models")
	h.key("3")
	h.dump("market with model")
	h.key("2")
	h.key("x")
	h.contains("RETIRE MODEL")
	h.key("esc")
	if len(h.m.s.Training.Models) != 1 {
		t.Fatal("esc retired the model")
	}
	h.key("x", "enter")
	if len(h.m.s.Training.Models) != 0 {
		t.Fatal("model not retired")
	}
}

func TestMarketSegments(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.step(0.1, 2)
	h.key("3")
	h.contains("RANKING", "Northwind", "SEGMENTS")
	h.key("home", "+", "+")
	if f := h.m.s.Market.Segments.Free; f < 0.09 || f > 0.11 {
		t.Fatalf("free = %v", f)
	}
	h.key("-")
	if f := h.m.s.Market.Segments.Free; f < 0.04 || f > 0.06 {
		t.Fatalf("free after - = %v", f)
	}
}

func TestPrestigeConfirm(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.step(0.1, 2)
	h.key("p")
	h.contains("New model version")
	h.dump("prestige")
	h.key("esc")
	if h.m.s.Prestige.ResetsL1 != 0 {
		t.Fatal("esc reset")
	}
	h.key("p", "enter")
	if h.m.s.Prestige.ResetsL1 != 1 || h.m.s.Prestige.Params <= 0 {
		t.Fatalf("prestige not done: %+v", h.m.s.Prestige)
	}
	h.key("p")
	h.contains("Requires")
	h.key("enter")
	if h.m.s.Prestige.ResetsL1 != 1 {
		t.Fatal("locked prestige went through")
	}
}

func TestReactionAndGrant(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.step(0.1, 2)
	i, _ := h.b.EventIndex("angel_investor")
	h.m.s.RandomEvent.ActiveID = "angel_investor"
	h.m.s.RandomEvent.EndsAtS = h.m.s.Clock + 8
	h.contains("Angel Investor", "G react")
	h.dump("event")
	money := h.m.s.Money
	h.key("h")
	if h.m.s.RandomEvent.ActiveID == "" {
		t.Fatal("wrong key reacted")
	}
	h.key("G")
	if h.m.s.RandomEvent.ActiveID != "" || !(h.m.s.Money > money) {
		t.Fatalf("reaction failed (event %d)", i)
	}
	h.contains("d grant")
	h.key("d")
	if h.m.s.Streak.Day != 1 || h.m.s.Streak.LastClaim != "2026-10-30" {
		t.Fatalf("grant not claimed: %+v", h.m.s.Streak)
	}
	h.lacks("d grant")
}

func TestCardToastAndCollection(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.step(0.1, 2)
	i, _ := h.b.CardIndex("dropout")
	h.m.event(&sim.Event{Kind: sim.EvCard, Business: -1, Index: i, Count: 3}, false)
	if !h.m.revealing() {
		t.Fatal("card toast does not reveal")
	}
	if h.m.desired() != fpsInterval(h.b.Engine.FPSAnimation) {
		t.Fatal("reveal does not run at animation FPS")
	}
	h.key("7")
	h.contains("RESEARCH CARDS", "Dropout", "???")
}

func TestUnlockBadges(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	h.m.event(&sim.Event{Kind: sim.EvUnlock, Business: -1, Index: int(sim.UnlockInfra)}, false)
	h.m.event(&sim.Event{Kind: sim.EvUnlock, Business: -1, Index: int(sim.UnlockCards)}, false)
	if !h.m.newTab[tabInfra] || !h.m.newTab[tabStats] {
		t.Fatalf("badges: %v", h.m.newTab)
	}
	h.key("4")
	if h.m.newTab[tabInfra] {
		t.Fatal("badge stays after visit")
	}
}
