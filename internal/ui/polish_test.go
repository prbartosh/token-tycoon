package ui

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
)

var sizes = [][2]int{{120, 40}, {80, 24}}

var langs = []i18n.Lang{i18n.EN, i18n.PL}

func fullSlots(c *Config) {
	m4Game(c)
	s := c.State
	s.Training.Models = []sim.Model{{Name: "TT-1", Params: 1e9, Benchmark: 75}, {Name: "TT-2", Params: 1e9, Benchmark: 40},
		{Name: "TT-3", Params: 1e9, Benchmark: 60}}
	s.Training.Released = 3
	s.Training.Queue = []sim.TrainJob{{Params: 1e9, Tokens: 2e10, ClusterShare: 1, Progress: 1, ArchMult: 1, Manual: true}}
	s.Training.Pending, s.Training.Archive = nil, nil
	sim.Normalize(s, c.Balance)
}

func (h *harness) live(n int) {
	for range n {
		h.m.evs = sim.Step(h.m.s, h.b, 0.1, nil, h.m.evs[:0])
		h.m.consume(h.m.evs, false)
	}
}

func (h *harness) ctrl(r rune) { h.send(tea.KeyPressMsg(tea.Key{Code: r, Mod: tea.ModCtrl})) }

func (h *harness) typ(s string) {
	for _, r := range s {
		if r == ' ' {
			h.key("space")
			continue
		}
		h.key(string(r))
	}
}

func hasBraille(s string) bool {
	for _, r := range s {
		if r > 0x2800 && r <= 0x28FF {
			return true
		}
	}
	return false
}

func TestReleaseChoiceRejectNewAndArchiveSwap(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], fullSlots)
			c := h.cat
			h.live(2)
			if h.m.modal.kind != modalChoose || len(h.m.s.Training.Pending) != 1 {
				t.Fatalf("no choice modal: %v pending %d", h.m.modal.kind, len(h.m.s.Training.Pending))
			}
			h.contains(c.T("ui.choice.title"), "TT-4", "TT-1", "TT-2", "TT-3")
			h.clean("choice modal")
			if h.m.modal.sel != 1 {
				t.Fatalf("default selection is not the weakest model: %d", h.m.modal.sel)
			}
			h.key("down", "down", "enter")
			if len(h.m.s.Training.Models) != 3 || len(h.m.s.Training.Archive) != 1 || h.m.s.Training.Archive[0].Name != "TT-4" {
				t.Fatalf("reject new: %+v", h.m.s.Training)
			}
			h.key("2")
			h.m.tsel[tabModels] = tkey{act: actArchived, idx: 0}
			h.contains(strings.ToUpper(c.T("ui.sec.archive", 1)), "TT-4")
			h.clean("archive rows")
			h.key("enter")
			if h.m.modal.kind != modalSwap {
				t.Fatalf("no swap modal: %v", h.m.modal.kind)
			}
			h.contains(c.T("ui.swap.title"))
			h.clean("swap modal")
			h.key("enter")
			names := h.m.s.Training.Models[0].Name + h.m.s.Training.Models[1].Name + h.m.s.Training.Models[2].Name
			if !strings.Contains(names, "TT-4") || h.m.s.Training.Archive[0].Name != "TT-2" {
				t.Fatalf("swap: %+v", h.m.s.Training)
			}
			h.m.tsel[tabModels] = tkey{act: actArchived, idx: 0}
			h.key("x")
			h.contains(c.T("ui.archive.delete_title"))
			h.key("esc")
			if len(h.m.s.Training.Archive) != 1 {
				t.Fatal("esc deleted the archived model")
			}
			h.key("x", "enter")
			if len(h.m.s.Training.Archive) != 0 {
				t.Fatal("archived model not deleted")
			}
		}
	}
}

func TestReleaseChoiceLaterAndRetireOld(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, fullSlots)
	h.live(2)
	h.key("esc")
	if h.m.modal.kind != modalNone || len(h.m.s.Training.Pending) != 1 {
		t.Fatal("esc did not postpone the choice")
	}
	h.frames(time.Second, 100*time.Millisecond)
	if h.m.modal.kind != modalNone {
		t.Fatal("postponed choice reopened by itself")
	}
	h.key("2")
	h.contains("WAITING FOR A DECISION (1)")
	h.m.tsel[tabModels] = tkey{act: actPending}
	h.key("enter")
	if h.m.modal.kind != modalChoose {
		t.Fatal("pending row did not reopen the choice")
	}
	h.key("enter")
	tr := &h.m.s.Training
	if len(tr.Pending) != 0 || len(tr.Archive) != 1 || tr.Archive[0].Name != "TT-2" || tr.Models[len(tr.Models)-1].Name != "TT-4" {
		t.Fatalf("retire weakest: %+v", *tr)
	}
	if h.m.cer.kind != cerRelease {
		t.Fatal("release ceremony missing after the choice")
	}
}

func TestOfflineReleaseNeedsNoModal(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, fullSlots)
	h.m.catchUp(120, true)
	tr := &h.m.s.Training
	if h.m.modal.kind != modalNone || len(tr.Pending) != 0 || len(tr.Archive) != 1 || tr.Archive[0].Name != "TT-2" {
		t.Fatalf("offline release: modal %v %+v", h.m.modal.kind, *tr)
	}
}

func TestAGIInvestPortions(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], m5Game)
			h.m.s.Incident = sim.Incident{}
			v := sim.AGIStatus(h.m.s, h.b)
			need := v.Cost - v.Invested
			h.m.s.Money = need / 2
			h.key("5")
			h.m.tsel[tabResearch] = tkey{act: actAGI, idx: h.m.s.AGI.Stage}
			h.key("enter")
			if h.m.modal.kind != modalAGI || h.m.modal.sel != 2 {
				t.Fatalf("agi modal %v sel %d", h.m.modal.kind, h.m.modal.sel)
			}
			h.contains(h.cat.T("ui.agi.title"), h.cat.T("ui.agi.missing"))
			h.clean("agi modal")
			h.key("4", "enter")
			if h.m.modal.confirm {
				t.Fatal("unaffordable missing amount reached confirmation")
			}
			h.key("1", "enter")
			if !h.m.modal.confirm {
				t.Fatal("no confirmation step")
			}
			h.clean("agi confirm")
			h.key("esc")
			if h.m.modal.kind != modalAGI || h.m.modal.confirm {
				t.Fatal("esc in confirmation did not go back")
			}
			m0, inv0 := h.m.s.Money, h.m.s.AGI.Invested
			h.key("enter", "enter")
			if math.Abs(h.m.s.AGI.Invested-inv0-m0*0.1) > 1e-9*m0 || h.m.modal.kind != modalNone {
				t.Fatalf("10%%: invested %v of %v", h.m.s.AGI.Invested-inv0, m0)
			}
		}
	}
}

func TestPaletteCommands(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.key(":")
	if !h.m.pal.open {
		t.Fatal("':' did not open the palette")
	}
	h.contains("COMMANDS", "buy max", "train", "settings")
	h.typ("buy rag 10")
	h.contains("buy RAG Chatbot x10")
	n0 := h.m.s.Businesses[1].Owned
	h.key("enter")
	if h.m.pal.open || h.m.s.Businesses[1].Owned != n0+10 {
		t.Fatalf("buy rag 10: owned %d -> %d", n0, h.m.s.Businesses[1].Owned)
	}
	h.ctrl('p')
	if !h.m.pal.open {
		t.Fatal("ctrl+p did not open the palette")
	}
	h.typ("go stat")
	h.key("enter")
	if h.m.tab != tabStats {
		t.Fatalf("go stat: tab %d", h.m.tab)
	}
	h.key(":")
	h.typ("idź badania")
	h.key("enter")
	if h.m.tab != tabResearch {
		t.Fatalf("Polish alias: tab %d", h.m.tab)
	}
	h.key(":")
	h.typ("q")
	if !h.m.pal.open || string(h.m.pal.query) != "q" {
		t.Fatal("q inside the palette must be text")
	}
	h.key("esc")
	if h.m.pal.open {
		t.Fatal("esc did not close the palette")
	}
	h.key(":")
	h.typ("notation")
	h.key("enter")
	if h.m.s.Settings.Notation != "scientific" {
		t.Fatal("notation command")
	}
	h.key(":")
	h.typ("zzzzqqq")
	h.contains("No matching commands")
	h.key("esc", ":")
	h.typ("prestige")
	h.key("enter")
	if h.m.modal.kind != modalPrestige {
		t.Fatal("prestige preview command")
	}
}

func TestPaletteRendersBothLanguages(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], lateGame)
			h.key(":")
			h.clean("palette")
			h.typ(h.cat.T("ui.cmd.buy") + " ")
			h.contains(h.cat.T("ui.cmd.buy") + " " + h.m.bizName(0))
			h.clean("palette buy")
			h.key("esc")
		}
	}
}

func TestFuzzyMatching(t *testing.T) {
	if fuzzy("idz bad", "idź badania") < 0 || fuzzy("xyz", "settings") >= 0 || fuzzy("", "x") != 0 {
		t.Fatal("fuzzy basics")
	}
	if fuzzy("set", "settings") <= fuzzy("set", "colorblind mode settings") {
		t.Fatal("prefix match should rank higher")
	}
}

func TestSettingsScreenLiveSwitch(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, lateGame)
	h.key(":")
	h.typ("settings")
	h.key("enter")
	if h.m.modal.kind != modalSettings {
		t.Fatal("settings not opened")
	}
	h.contains("SETTINGS", "English")
	h.key("enter")
	if h.m.cat.Lang() != i18n.PL || h.m.s.Settings.Lang != "pl" {
		t.Fatal("language not switched")
	}
	h.contains("USTAWIENIA", "polski", "Biznesy")
	h.key("down", "enter", "down", "enter", "down", "enter", "down", "enter")
	st := h.m.s.Settings
	if st.Notation != "scientific" || !st.Colorblind || st.Advisor == h.b.UI.AdvisorDefault || !st.Bell {
		t.Fatalf("settings not toggled: %+v", st)
	}
	for _, sz := range sizes {
		h.send(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		h.clean("settings")
	}
	h.key("esc")
	if h.m.modal.kind != modalNone {
		t.Fatal("esc did not close settings")
	}
	h.key("p")
	h.key("enter")
	if h.m.cer.kind != cerPrestige {
		t.Fatal("no prestige ceremony")
	}
	_, cmd := h.m.Update(frameMsg{gen: h.m.gen, t: h.clk.t.Add(100 * time.Millisecond)})
	if cmd == nil {
		t.Fatal("no command after frame")
	}
}

func TestCeremoniesAnimateAndSkip(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], lateGame)
			h.key("p", "enter")
			if h.m.cer.kind != cerPrestige {
				t.Fatal("no prestige ceremony")
			}
			h.frames(time.Second, 100*time.Millisecond)
			h.contains(h.cat.T("ui.cer.prestige"))
			if !hasBraille(h.view()) {
				t.Fatalf("no loss chart:\n%s", h.view())
			}
			h.clean("prestige ceremony")
			if h.m.desired() != fpsInterval(h.b.Engine.FPSAnimation) {
				t.Fatal("ceremony does not animate")
			}
			h.key("2")
			if h.m.tab != tabModels || !h.m.cer.active() {
				t.Fatal("ceremony blocks input")
			}
			h.key("esc")
			if h.m.cer.active() {
				t.Fatal("esc did not skip")
			}
		}
	}
	h := newHarness(t, i18n.EN, 120, 40, fullSlots)
	h.m.s.Training.Models = h.m.s.Training.Models[:2]
	h.live(2)
	if h.m.cer.kind != cerRelease {
		t.Fatal("no release ceremony")
	}
	h.contains("MODEL RELEASE", "TT-4", "Ranking")
	h.frames(time.Duration(cerAnimS*float64(time.Second))+time.Second, 100*time.Millisecond)
	h.clean("release ceremony")
	h.frames(time.Duration(cerHoldS*float64(time.Second)), 100*time.Millisecond)
	if h.m.cer.active() {
		t.Fatal("release ceremony never ends")
	}
}

func TestStatsChartAndHistory(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], lateGame)
			h.key("7")
			h.contains(h.cat.T("ui.stats.chart_empty"))
			h.frames(20*time.Second, 200*time.Millisecond)
			h.contains(strings.ToUpper(h.cat.T("ui.sec.income_chart")), strings.ToUpper(h.cat.T("ui.sec.records")), h.cat.T("ui.stats.peak_income"))
			if !hasBraille(h.view()) {
				t.Fatalf("no income chart:\n%s", h.view())
			}
			h.clean("stats chart")
		}
	}
	var hs history
	for range 1000 {
		hs.add(10, 1)
	}
	if hs.n > histCap || hs.every <= histStartS || math.Abs(hs.vals[0]-10) > 1e-9 || hs.peak != 10 {
		t.Fatalf("history compression: n %d every %v v0 %v", hs.n, hs.every, hs.vals[0])
	}
}

func TestContextHelpMatchesDecisions(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], m5Game)
			h.m.s.Incident = sim.Incident{}
			for tab := range numTabs {
				h.m.setTab(tab)
				h.key("?")
				if !h.m.help {
					t.Fatal("? did not open help")
				}
				h.contains(strings.ToUpper(h.cat.T(tabKeys[tab])))
				h.clean("help " + helpTabIDs[tab])
				for range 30 {
					h.key("down")
				}
				h.clean("help scrolled")
				if sz[1] == 24 && h.m.helpTop == 0 {
					t.Fatal("help does not scroll at 80x24")
				}
				h.key("esc")
				if h.m.help {
					t.Fatal("esc did not close help")
				}
			}
		}
	}
	c := i18n.New(i18n.EN)
	all := ""
	for i := 1; c.Has("ui.help." + strconv.Itoa(i)); i++ {
		all += c.T("ui.help."+strconv.Itoa(i)) + "\n"
	}
	for _, want := range []string{"← →  1-7|", "↑ ↓  k j|", "h and l do not switch", "Space|", "+0.1", "x5", "2 s", ":  Ctrl+P|", "H  G|"} {
		if !strings.Contains(all, want) {
			t.Fatalf("help misses %q", want)
		}
	}
	h := newHarness(t, i18n.EN, 120, 40, nil)
	for _, k := range []string{"l", "h"} {
		h.key(k)
		if h.m.tab != tabBusinesses {
			t.Fatalf("%s switched tabs", k)
		}
	}
	h.key("right")
	if h.m.tab != tabModels {
		t.Fatal("right arrow")
	}
	h.key("left", "7")
	if h.m.tab != tabStats {
		t.Fatal("digit tabs")
	}
	h.key("1", "j")
	if h.m.sel.idx != 1 {
		t.Fatal("j does not move down")
	}
	h.key("k")
	if h.m.sel.idx != 0 {
		t.Fatal("k does not move up")
	}
	h.key("space")
	if math.Abs(h.m.s.Combo.Mult-(1+h.b.Combo.Step)) > 1e-9 || !h.m.s.Businesses[0].Running {
		t.Fatalf("space: combo %v running %v", h.m.s.Combo.Mult, h.m.s.Businesses[0].Running)
	}
}

func TestPreviewsExplainLeapAndRoundMult(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], m4Game)
			c := h.cat
			h.m.s.Prestige.Params = 50
			h.m.s.Prestige.ResetsL2, h.m.s.Prestige.RoundMult = 1, 4
			sim.Normalize(h.m.s, h.b)
			fv := sim.Funding(h.m.s, h.b)
			if !fv.Allowed || fv.RoundMultNew <= fv.RoundMult {
				t.Fatalf("funding preview %+v", fv)
			}
			h.key("P")
			h.contains("x" + h.m.num(fv.RoundMult))
			h.clean("funding preview")
			h.key("esc")
			h.m.s.Lifetime.Run = 1e20
			h.m.s.Prestige.RunStartS = h.m.s.Clock - 600
			pv := sim.Preview(h.m.s, h.b)
			if !pv.Allowed || pv.RawGain <= pv.Gain*1.05 {
				t.Fatalf("leap limit not binding: %+v", pv)
			}
			h.key("p")
			h.contains(h.m.num(pv.RawGain))
			h.clean("prestige preview")
			h.key("esc")
			h.m.s.Prestige.RunStartS = h.m.s.Clock
			h.key("p")
			h.contains(strings.Fields(c.T("ui.prestige.too_short"))[0])
			h.clean("prestige too short")
			h.key("esc", "7")
			found := false
			for range 80 {
				if strings.Contains(h.view(), c.T("ui.stats.round_mult")) {
					found = true
					break
				}
				h.key("down")
			}
			if !found {
				t.Fatal("round multiplier missing from statistics")
			}
		}
	}
}
