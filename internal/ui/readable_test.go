package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/colorprofile"

	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/anim"
	"token-tycoon/internal/ui/theme"
	"token-tycoon/internal/ui/widgets"
)

func (h *harness) noEllipsis(where string) {
	h.t.Helper()
	for _, l := range strings.Split(h.view(), "\n") {
		i := strings.Index(l, "…")
		if i < 0 {
			continue
		}
		if h.m.tab == tabBusinesses && h.truncatedName(l[:i]) {
			continue
		}
		h.t.Fatalf("%s: truncated text %q\n%s", where, l, h.view())
	}
}

func (h *harness) truncatedName(prefix string) bool {
	head := strings.TrimSpace(strings.TrimLeft(prefix, "│ >*"))
	if head == "" {
		return false
	}
	for i := range sim.N {
		if strings.HasPrefix(h.m.bizName(i), head) {
			return true
		}
	}
	for u := range h.b.Upgrades {
		if strings.HasPrefix(h.m.upgName(u), head) {
			return true
		}
	}
	return false
}

func TestDescriptiveTextWrapsInsteadOfTruncating(t *testing.T) {
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], m5Game)
			where := lang.String() + " "
			for _, k := range []string{"1", "2", "3", "4", "5", "6", "7"} {
				h.key(k)
				for range 40 {
					h.noEllipsis(where + "tab " + k)
					h.fits()
					h.key("down")
				}
			}
			h.key("1", "?")
			for range 30 {
				h.noEllipsis(where + "help")
				h.fits()
				h.key("down")
			}
			h.key("esc", ":")
			h.noEllipsis(where + "palette")
			h.key("esc")
			for k := modalPrestige; k <= modalSettings; k++ {
				h.m.modal = modal{kind: k}
				h.noEllipsis(where + "modal")
				h.fits()
			}
			h.m.modal = modal{}
			h.m.openWizard()
			for range 6 {
				h.noEllipsis(where + "wizard")
				h.key("enter")
			}
			h.m.modal = modal{}
		}
	}
}

func TestBusinessDetailShowsFullName(t *testing.T) {
	h := newHarness(t, i18n.PL, 80, 24, lateGame)
	for i := range sim.N {
		h.m.sel = row{kind: rowBusiness, idx: i}
		h.m.ensureVisible()
		v := strings.Join(strings.Fields(h.view()), " ")
		for _, w := range strings.Fields(strings.ToUpper(h.m.bizName(i))) {
			if !strings.Contains(v, w) {
				t.Fatalf("detail misses %q of %q", w, h.m.bizName(i))
			}
		}
	}
}

func TestLongTabRowsWrapAndKeepSelection(t *testing.T) {
	h := newHarness(t, i18n.PL, 80, 24, m5Game)
	h.key("5")
	rs := h.m.tabRows(h.m.mainWidth())
	cont := 0
	for i, r := range rs {
		if widgets.Width(r.text) > h.m.mainWidth()-2 {
			t.Fatalf("row %d not wrapped: %q", i, r.text)
		}
		if r.cont {
			cont++
			if r.selectable() || i == 0 || rs[i-1].key != r.key {
				t.Fatalf("continuation row %d detached", i)
			}
		}
	}
	if cont == 0 {
		t.Fatal("expected wrapped research rows at 80x24")
	}
	before := h.m.tabCurrent()
	h.key("down")
	if h.m.tabCurrent() == before {
		t.Fatal("selection stuck on continuation rows")
	}
}

func TestToastsLastLongEnoughAndQueue(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	short, long := h.m.toastS("ok"), h.m.toastS(strings.Repeat("x", 400))
	if short < 5 || long != readMaxS {
		t.Fatalf("durations %v %v", short, long)
	}
	if mid := h.m.toastS(strings.Repeat("y", 60)); mid <= short || mid >= long {
		t.Fatalf("duration does not grow with length: %v", mid)
	}
	msgs := make([]string, anim.MaxToasts+3)
	for i := range msgs {
		msgs[i] = "toast number " + string(rune('A'+i))
		h.m.anim.Toast.Push(msgs[i], anim.KindInfo, h.m.toastS(msgs[i]))
	}
	h.frames(3500*time.Millisecond, 100*time.Millisecond)
	h.contains(msgs[0])
	seen := map[string]bool{}
	for range 400 {
		for _, ts := range h.m.anim.Toast.Items() {
			seen[ts.Text] = true
		}
		h.frames(100*time.Millisecond, 100*time.Millisecond)
	}
	for _, s := range msgs {
		if !seen[s] {
			t.Fatalf("toast %q was dropped", s)
		}
	}
	if h.m.anim.Toast.Pending() != 0 || len(h.m.anim.Toast.Items()) != 0 {
		t.Fatal("toast queue did not drain")
	}
}

func TestLongToastWraps(t *testing.T) {
	for _, sz := range sizes {
		h := newHarness(t, i18n.PL, sz[0], sz[1], nil)
		msg := "Bardzo długie powiadomienie, które musi się zawinąć zamiast zostać ucięte w połowie zdania koniec"
		h.m.anim.Toast.Push(msg, anim.KindInfo, h.m.toastS(msg))
		h.frames(time.Second, 50*time.Millisecond)
		h.noEllipsis("toast")
		h.fits()
		h.contains("koniec", "Bardzo")
	}
}

func TestTokenBurstStaysReadable(t *testing.T) {
	h := newHarness(t, i18n.EN, 80, 24, nil)
	h.m.event(&sim.Event{Kind: sim.EvCrit, Business: 0, Count: 1, Amount: 10}, false)
	h.m.event(&sim.Event{Kind: sim.EvCrit, Business: 0, Count: 1, Amount: 20}, false)
	alive := 0
	for _, f := range h.m.anim.Float.Items() {
		if f.Alive() && f.Kind == anim.KindCrit {
			alive++
		}
	}
	if alive != 1 {
		t.Fatalf("crit floaters not merged: %d", alive)
	}
	h.frames(4*time.Second, 100*time.Millisecond)
	h.contains("TOKEN BURST")
	if h.m.anim.Active() {
		t.Fatal("a resting burst must not keep 30 FPS")
	}
	h.frames(4*time.Second, 100*time.Millisecond)
	h.lacks("TOKEN BURST")
}

func TestStatusLastsAndTickerScrolls(t *testing.T) {
	h := newHarness(t, i18n.PL, 80, 24, nil)
	msg := "Początek " + strings.Repeat("środek komunikatu ", 8) + "KONIEC"
	h.m.setStatus(msg, false)
	h.frames(4*time.Second, 100*time.Millisecond)
	h.contains("środek")
	if h.m.statusFor < time.Duration(readBaseS*float64(time.Second)) {
		t.Fatalf("status too short: %v", h.m.statusFor)
	}
	saw := false
	for el := 4 * time.Second; el < h.m.statusFor; el += 200 * time.Millisecond {
		if strings.Contains(h.view(), "KONIEC") {
			saw = true
			break
		}
		h.frames(200*time.Millisecond, 100*time.Millisecond)
	}
	if !saw {
		t.Fatal("ticker never reached the end of the status")
	}
	h.noEllipsis("news")
}

func TestCycleBarFlowsWhenSpeedMaxed(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, func(c *Config) {
		c.Profile = colorprofile.TrueColor
		s := c.State
		for i := range 3 {
			s.Businesses[i].Owned = 60
			s.Businesses[i].Manager = true
			s.Businesses[i].Progress = 0.1
		}
		sim.Normalize(s, c.Balance)
	})
	m := h.m
	const w = 20
	if m.speedMaxed(2) || m.s.Derived.Continuous[2] {
		t.Fatal("business 3 should not be maxed at 60")
	}
	if got := strings.Count(m.cycleBar(2, w), widgets.Full); got == w {
		t.Fatal("unmaxed bar should show progress")
	}
	set := func(owned int) {
		m.s.Businesses[2].Owned = owned
		sim.Normalize(m.s, m.b)
	}
	last := m.b.Milestones.SpeedThresholds[len(m.b.Milestones.SpeedThresholds)-1]
	set(last)
	if !m.speedMaxed(2) || m.s.Derived.Continuous[2] {
		t.Fatal("business 3 should be speed maxed but not continuous")
	}
	if got := strings.Count(m.cycleBar(2, w), widgets.Full); got != w {
		t.Fatalf("maxed bar should flow, got %d full cells", got)
	}
	if m.extraColors(2) != 0 {
		t.Fatal("no extra colors right at the cap")
	}
	base := m.cycleBar(2, w)
	prev := 0
	for k := 1; k <= theme.MixMax+2; k++ {
		set(last + k*m.b.Milestones.ProfitEvery)
		n := m.extraColors(2)
		if n < prev || n > theme.MixMax {
			t.Fatalf("extra colors %d after %d profit milestones", n, k)
		}
		prev = n
	}
	if prev != theme.MixMax {
		t.Fatalf("extra colors should cap at %d, got %d", theme.MixMax, prev)
	}
	if m.cycleBar(2, w) == base {
		t.Fatal("extra colors did not change the bar")
	}
	th := m.th
	for j := 1; j < theme.MixMax; j++ {
		if th.Mix[2][j] == th.Mix[2][j-1] {
			t.Fatalf("mix %d equals mix %d", j+2, j+1)
		}
	}
	if th.Mix[2][0] == th.Flow[2] {
		t.Fatal("first mix equals the plain flow")
	}
	plain := theme.New(m.b.UI.Colors, colorprofile.ASCII)
	var sb strings.Builder
	widgets.Flow(&sb, w, 3, plain.Mix[0][2][:])
	if strings.Contains(sb.String(), "38;") {
		t.Fatal("no-color flow emitted colors")
	}
}

func TestTutorialOnlyOnNewGame(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, nil)
	if h.m.tut.open {
		t.Fatal("tutorial shown for an existing save")
	}
	h = newHarness(t, i18n.EN, 120, 40, func(c *Config) {
		c.NewGame = true
		c.State.Settings.TutorialDone = true
	})
	if h.m.tut.open {
		t.Fatal("tutorial shown although done")
	}
	for _, lang := range langs {
		for _, sz := range sizes {
			h := newHarness(t, lang, sz[0], sz[1], func(c *Config) { c.NewGame = true })
			if !h.m.tut.open {
				t.Fatal("tutorial not shown on a new game")
			}
			for step := range tutSteps {
				h.contains(h.cat.T("ui.tut.title"))
				h.clean("tutorial step")
				h.noEllipsis("tutorial step")
				if h.m.tut.step != step {
					t.Fatalf("step %d, want %d", h.m.tut.step, step)
				}
				h.key("enter")
			}
			if h.m.tut.open || !h.m.s.Settings.TutorialDone {
				t.Fatal("tutorial did not finish")
			}
		}
	}
}

func TestTutorialBlocksInputSkipsAndReopens(t *testing.T) {
	h := newHarness(t, i18n.PL, 80, 24, func(c *Config) { c.NewGame = true })
	h.key("space", "2", "m")
	if h.m.s.Combo.Mult > 1 || h.m.tab != tabBusinesses || h.m.s.Businesses[0].Running {
		t.Fatal("game reacted to keys under the tutorial")
	}
	h.key("enter", "enter", "left")
	if h.m.tut.step != 2 {
		t.Fatalf("back did not work: %d", h.m.tut.step)
	}
	h.key("esc")
	if h.m.tut.open || !h.m.s.Settings.TutorialDone {
		t.Fatal("esc did not skip")
	}
	h.contains("?")
	h.key(":")
	h.typ(h.cat.T("ui.cmd.tutorial"))
	h.key("enter")
	if !h.m.tut.open || h.m.tut.step != 0 || h.m.pal.open {
		t.Fatal("palette did not reopen the tutorial")
	}
	h.key("esc", "?", "t")
	if !h.m.tut.open || h.m.help {
		t.Fatal("help did not reopen the tutorial")
	}
	h.key("esc")
	h.key("space")
	if h.m.s.Combo.Mult <= 1 {
		t.Fatal("game input blocked after the tutorial")
	}
}
