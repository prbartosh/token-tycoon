package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

type harness struct {
	t   *testing.T
	m   *Model
	clk *clock
	b   *balance.Balance
	cat *i18n.Catalog
}

func newHarness(t *testing.T, lang i18n.Lang, w, h int, mod func(*Config)) *harness {
	t.Helper()
	b := balance.Default()
	st := sim.NewState(b, 7)
	clk := &clock{t: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
	cat := i18n.New(lang)
	cfg := Config{Balance: b, State: &st, Catalog: cat, Profile: colorprofile.ASCII, Now: clk.now}
	if mod != nil {
		mod(&cfg)
	}
	hs := &harness{t: t, m: New(cfg), clk: clk, b: b, cat: cat}
	hs.m.Init()
	hs.send(tea.WindowSizeMsg{Width: w, Height: h})
	return hs
}

func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	nm, _ := h.m.Update(msg)
	h.m = nm.(*Model)
}

func (h *harness) frames(total time.Duration, every time.Duration) {
	for el := time.Duration(0); el < total; el += every {
		h.clk.t = h.clk.t.Add(every)
		h.send(frameMsg{gen: h.m.gen, t: h.clk.t})
	}
}

var named = map[string]tea.Key{
	"enter": {Code: tea.KeyEnter}, "space": {Code: tea.KeySpace, Text: " "}, "tab": {Code: tea.KeyTab},
	"up": {Code: tea.KeyUp}, "down": {Code: tea.KeyDown}, "left": {Code: tea.KeyLeft},
	"right": {Code: tea.KeyRight}, "esc": {Code: tea.KeyEscape},
}

func (h *harness) key(keys ...string) {
	h.t.Helper()
	for _, k := range keys {
		kk, ok := named[k]
		if !ok {
			r := []rune(k)[0]
			kk = tea.Key{Code: r, Text: k}
		}
		h.send(tea.KeyPressMsg(kk))
	}
}

func (h *harness) view() string { return ansi.Strip(h.m.View().Content) }

func (h *harness) contains(subs ...string) {
	h.t.Helper()
	v := h.view()
	for _, s := range subs {
		if !strings.Contains(v, s) {
			h.t.Fatalf("view does not contain %q:\n%s", s, v)
		}
	}
}

func (h *harness) lacks(s string) {
	h.t.Helper()
	if v := h.view(); strings.Contains(v, s) {
		h.t.Fatalf("view unexpectedly contains %q:\n%s", s, v)
	}
}

func TestLayoutBothLanguages(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.EN, i18n.PL} {
		for _, sz := range [][2]int{{80, 24}, {120, 40}, {60, 20}} {
			h := newHarness(t, lang, sz[0], sz[1], nil)
			c := h.cat
			h.contains(c.T("game.title"), c.T("business.api_wrapper"), c.T("tab.businesses"),
				"7", c.T("ui.news.label"), c.T("ui.biz.upgrades"), "x1", c.T("key.quit"))
			if sz[0] >= 80 {
				h.contains(c.T("tab.stats"), c.T("ui.detail.title"), c.T("ui.detail.income"))
			}
			if sz[0] >= 120 {
				h.contains(c.T("ui.log.title"))
			}
			lines := strings.Split(h.view(), "\n")
			if len(lines) != sz[1] {
				t.Fatalf("%v %v: %d lines", lang, sz, len(lines))
			}
			for i, l := range lines {
				if w := ansi.StringWidth(l); w > sz[0] {
					t.Fatalf("%v %v: line %d is %d wide: %q", lang, sz, i, w, l)
				}
			}
			v := h.m.View()
			if !v.AltScreen || !v.ReportFocus || v.MouseMode != tea.MouseModeCellMotion {
				t.Fatal("view flags")
			}
		}
	}
}

func TestManualCyclesAndBuying(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	h.contains(h.cat.T("ui.key.start"))
	for range 12 {
		h.key("space")
		h.frames(1100*time.Millisecond, 100*time.Millisecond)
	}
	if h.m.s.Money < 10 {
		t.Fatalf("money after manual cycles: %v", h.m.s.Money)
	}
	h.key("enter")
	if got := h.m.s.Businesses[0].Owned; got != 2 {
		t.Fatalf("owned after buy: %d", got)
	}
	h.contains("x2")
	if h.m.desired() != fpsInterval(h.b.Engine.FPSAnimation) {
		t.Fatal("purchase flash should raise the frame rate")
	}
	h.key("u")
	if got := h.m.s.Businesses[0].Owned; got != 1 {
		t.Fatalf("owned after undo: %d", got)
	}
	h.frames(time.Second, 100*time.Millisecond)
	if h.m.desired() != fpsInterval(h.b.Engine.FPSNormal) {
		t.Fatal("idle model should run at normal fps")
	}
}

func TestQuantityAndManager(t *testing.T) {
	h := newHarness(t, i18n.PL, 80, 24, nil)
	h.key("tab")
	h.contains(h.cat.T("key.qty", "x10"))
	h.key("tab", "tab")
	h.contains(h.cat.T("key.qty", h.cat.T("qty.max")))
	h.m.s.Money = 1000
	h.key("m")
	if !h.m.s.Businesses[0].Manager {
		t.Fatal("manager not hired")
	}
	h.frames(300*time.Millisecond, 33*time.Millisecond)
	h.contains("Agent Ping")
	h.frames(3*time.Second, 100*time.Millisecond)
	if h.m.s.Money <= 940 {
		t.Fatalf("manager does not earn: %v", h.m.s.Money)
	}
}

func TestSelectionAndUpgrades(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	h.key("j", "down")
	if h.m.sel != (row{kind: rowBusiness, idx: 2}) {
		t.Fatalf("sel %+v", h.m.sel)
	}
	h.key("k")
	if h.m.sel.idx != 1 {
		t.Fatalf("sel %+v", h.m.sel)
	}
	for range 20 {
		h.key("down")
	}
	if h.m.sel.kind != rowUpgrade {
		t.Fatalf("sel %+v", h.m.sel)
	}
	h.m.s.Money = 1e12
	h.key("enter")
	if h.m.s.Upgrades.Count() != 1 {
		t.Fatal("upgrade not bought")
	}
	if h.m.sel.kind != rowUpgrade || h.m.s.Upgrades.Has(h.m.sel.idx) {
		t.Fatalf("selection did not follow to an unbought upgrade: %+v", h.m.sel)
	}
	h.contains("> " + h.m.upgName(h.m.sel.idx))
}

func TestTabsKeysAndHelp(t *testing.T) {
	h := newHarness(t, i18n.EN, 100, 30, nil)
	h.key("4")
	h.contains(h.cat.T("ui.tab.locked", h.cat.T("unlock.infra"), h.m.money(h.b.Infra.UnlockLifetime)))
	h.key("h", "l")
	if h.m.tab != tabInfra {
		t.Fatal("h and l must not switch tabs")
	}
	h.key("right")
	if h.m.tab != tabResearch {
		t.Fatal("right")
	}
	h.key("left", "left", "left", "left", "left")
	if h.m.tab != tabStats {
		t.Fatalf("left wraps: %d", h.m.tab)
	}
	h.contains(h.cat.T("ui.stats.play"))
	h.key("1", "?")
	h.contains(h.cat.T("ui.help.title"), h.cat.T("ui.key.close"))
	h.key("esc")
	h.lacks(h.cat.T("ui.help.title"))
	h.key("n")
	if h.m.s.Settings.Notation != "scientific" {
		t.Fatal("notation toggle")
	}
}

func TestTinySizesDoNotPanic(t *testing.T) {
	h := newHarness(t, i18n.PL, 1, 1, nil)
	for w := 0; w <= 130; w += 7 {
		for ht := 0; ht <= 45; ht += 4 {
			h.send(tea.WindowSizeMsg{Width: w, Height: ht})
			h.key("down", "?", "esc", "space")
			h.frames(200*time.Millisecond, 100*time.Millisecond)
			_ = h.m.View()
		}
	}
	h.send(tea.WindowSizeMsg{Width: 50, Height: 20})
	h.contains("50x20")
}

func TestAwayScreen(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, func(c *Config) {
		c.State.Businesses[0].Manager = true
		c.OfflineElapsed = 3600
	})
	h.contains(h.cat.T("ui.away.title"), h.cat.T("ui.away.elapsed", "1h00m"))
	if h.m.desired() != fpsInterval(h.b.Engine.FPSAnimation) {
		t.Fatal("away counter should animate")
	}
	h.key("x")
	h.lacks(h.cat.T("ui.away.title"))
	h.frames(100*time.Millisecond, 100*time.Millisecond)
	h.frames(2*time.Hour, 2*time.Hour)
	h.contains(h.cat.T("ui.away.title"), "2h00m")
}

func TestMouse(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	h.m.s.Money = 1e4
	h.send(tea.MouseClickMsg{X: 5, Y: topRows + 1, Button: tea.MouseLeft})
	if h.m.sel.idx != 1 {
		t.Fatalf("click select: %+v", h.m.sel)
	}
	h.send(tea.MouseClickMsg{X: 5, Y: topRows + 1, Button: tea.MouseLeft})
	if h.m.s.Businesses[1].Owned != 1 {
		t.Fatal("double click should buy")
	}
	h.send(tea.MouseWheelMsg{X: 5, Y: topRows + 1, Button: tea.MouseWheelDown})
	if h.m.sel.idx != 2 {
		t.Fatalf("wheel: %+v", h.m.sel)
	}
	sp := h.m.tabSpans()
	h.send(tea.MouseClickMsg{X: sp[tabStats][0] + 1, Y: 1, Button: tea.MouseLeft})
	if h.m.tab != tabStats {
		t.Fatal("tab click")
	}
}

func TestLoopAutosaveAndFocus(t *testing.T) {
	saves := 0
	h := newHarness(t, i18n.EN, 120, 40, func(c *Config) {
		c.Save = func(*sim.State) error { saves++; return nil }
	})
	h.frames(250*time.Millisecond, 50*time.Millisecond)
	if c := h.m.s.Clock; c < 0.15 || c > 0.25 {
		t.Fatalf("clock after 250ms: %v", c)
	}
	stale := frameMsg{gen: h.m.gen - 1, t: h.clk.t.Add(time.Hour)}
	h.send(stale)
	if h.m.s.Clock > 1 {
		t.Fatal("stale frame advanced the sim")
	}
	h.frames(61*time.Second, 500*time.Millisecond)
	if saves != 1 {
		t.Fatalf("autosaves: %d", saves)
	}
	h.send(tea.BlurMsg{})
	if h.m.desired() != fpsInterval(h.b.Engine.FPSBackground) {
		t.Fatal("background fps")
	}
	h.send(tea.FocusMsg{})
	if h.m.desired() == fpsInterval(h.b.Engine.FPSBackground) {
		t.Fatal("focus fps")
	}
	nm, cmd := h.m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil || nm == nil {
		t.Fatal("q must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q must return tea.Quit")
	}
}

func TestCritAndMilestoneAnimations(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	h.m.event(&sim.Event{Kind: sim.EvCrit, Business: 0, Count: 1, Amount: 10}, false)
	h.m.event(&sim.Event{Kind: sim.EvMilestoneSpeed, Business: 0, Index: 25, Amount: 0.5}, false)
	h.frames(100*time.Millisecond, 33*time.Millisecond)
	h.contains("TOKEN BURST")
	if !h.m.anim.Active() {
		t.Fatal("animations should be active")
	}
	h.frames(4*time.Second, 100*time.Millisecond)
	if h.m.anim.Active() {
		t.Fatal("animations should settle")
	}
}

func TestColorProfilesRender(t *testing.T) {
	for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI} {
		h := newHarness(t, i18n.EN, 120, 40, func(c *Config) { c.Profile = p })
		h.m.s.Money = 1e6
		h.key("tab", "tab", "enter", "m", "space")
		h.frames(time.Second, 100*time.Millisecond)
		if !h.m.s.Derived.Continuous[0] {
			t.Fatal("expected continuous bar")
		}
		for i, l := range strings.Split(h.view(), "\n") {
			if w := ansi.StringWidth(l); w > 120 {
				t.Fatalf("%v: line %d is %d wide", p, i, w)
			}
		}
		if !strings.Contains(h.m.View().Content, "\x1b[") {
			t.Fatalf("%v: no styling", p)
		}
		h.contains("API Wrapper")
	}
}

func BenchmarkFrameAndView(b *testing.B) {
	t := &testing.T{}
	h := newHarness(t, i18n.EN, 120, 40, func(c *Config) { c.Profile = colorprofile.TrueColor })
	h.m.s.Money = 1e6
	h.key("m", "down", "enter", "m")
	b.ReportAllocs()
	for b.Loop() {
		h.clk.t = h.clk.t.Add(100 * time.Millisecond)
		h.send(frameMsg{gen: h.m.gen, t: h.clk.t})
		_ = h.m.View()
	}
}
