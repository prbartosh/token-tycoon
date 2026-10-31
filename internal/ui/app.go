// Package ui is the Bubble Tea front end of Token Tycoon. It owns the frame
// loop, input, layout and animations; the game itself lives in sim and the UI
// only reads its state and sends commands.
package ui

import (
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/format"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/anim"
	"token-tycoon/internal/ui/theme"
	"token-tycoon/internal/ui/widgets"
)

type Config struct {
	Balance        *balance.Balance
	State          *sim.State
	Catalog        *i18n.Catalog
	Profile        colorprofile.Profile
	Debug          bool
	Save           func(*sim.State) error
	OfflineElapsed float64
	Notice         string
	NewGame        bool
	Now            func() time.Time
}

const (
	tabBusinesses = iota
	tabModels
	tabMarket
	tabInfra
	tabResearch
	tabRules
	tabStats
	numTabs
)

const (
	topRows      = 3
	bottomRows   = 3
	minHeight    = 14
	catchUpLimit = 60 * time.Second
	doubleClick  = 400 * time.Millisecond
	readBaseS    = 5.0
	readPerRuneS = 0.06
	readMaxS     = 12.0
	tickerPauseS = 2.0
	tickerCPS    = 8.0
	tickerKeyLen = 24
	newsEvery    = 20 * time.Second
	logCap       = 64
	sparkLen     = 60
	rateWindow   = 5
	manualDelayS = 1.0
	bell         = "\a"
)

const noUnlock = sim.Unlock(255)

var tabUnlock = [numTabs]sim.Unlock{noUnlock, sim.UnlockTraining, sim.UnlockMarket, sim.UnlockInfra,
	sim.UnlockResearch, sim.UnlockRules, noUnlock}

var tabKeys = [numTabs]string{"tab.businesses", "tab.models", "tab.market", "tab.infra",
	"tab.research", "tab.rules", "tab.stats"}

var qtys = [4]int{sim.Qty1, sim.Qty10, sim.Qty100, sim.QtyMax}

type frameMsg struct {
	gen int
	t   time.Time
}

type away struct {
	res  sim.OfflineResult
	open bool
}

type Model struct {
	b     *balance.Balance
	s     *sim.State
	cat   *i18n.Catalog
	th    *theme.Theme
	now   func() time.Time
	save  func(*sim.State) error
	debug bool

	width, height int
	focused       bool
	tab           int
	newTab        [numTabs]bool
	sel           row
	scroll        int
	qty           int
	help          bool
	away          away

	gen      int
	interval time.Duration
	last     time.Time
	acc      time.Duration
	step     time.Duration
	lastSave time.Time
	fps      float64

	evs  []sim.Event
	cmds []sim.Command
	anim anim.Animator

	spark      [sparkLen]float64
	sparkN     int
	sparkAcc   float64
	sparkSteps int

	status     string
	statusAt   time.Time
	statusFor  time.Duration
	statusWarn bool
	ticker     ticker
	log        []string
	newsIdx    int
	newsAt     time.Time

	adv       advice
	advAt     float64
	lastFloat [sim.N]float64

	lastClick    time.Time
	lastClickRow int

	tsel     [numTabs]tkey
	tscroll  [numTabs]int
	trows    []trow
	wrapped  []trow
	draft    trainDraft
	modal    modal
	retiring string
	wiz      wizard
	ruleLog  []string
	cer      ceremony
	ring     bool
	pal      palette
	tut      tutorial
	later    bool
	helpTop  int
	hist     history
	plot     widgets.Plot
	nums     numCache
	logSeq   int
	logC     logCache
	tabC     tabBarCache
	prefixes [4]string
	upgC     []upgEntry

	rows   []row
	lines  []string
	sb     strings.Builder
	canvas *lipgloss.Canvas
}

func New(cfg Config) *Model {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	b := cfg.Balance
	m := &Model{
		b: b, s: cfg.State, cat: cfg.Catalog, now: now, save: cfg.Save, debug: cfg.Debug,
		th:      theme.New(b.UI.Colors, cfg.Profile),
		focused: true,
		evs:     make([]sim.Event, 0, 256),
		cmds:    make([]sim.Command, 0, 4),
		log:     make([]string, 0, logCap),
		rows:    make([]row, 0, 32),
		trows:   make([]trow, 0, 64),
		wrapped: make([]trow, 0, 64),
		lines:   make([]string, 0, 64),
		step:    time.Duration(max(1, b.Engine.SimStepMs)) * time.Millisecond,
		sel:     row{kind: rowBusiness, idx: 0},
		upgC:    make([]upgEntry, len(b.Upgrades)),
	}
	t0 := now()
	m.lastSave, m.newsAt = t0, t0
	if cfg.OfflineElapsed > 0 {
		m.catchUp(cfg.OfflineElapsed, true)
	}
	m.anim.Money.Jump(m.s.Money)
	m.anim.Counter.Rate = 0.08
	if cfg.Notice != "" {
		m.setStatus(cfg.Notice, false)
	}
	m.refreshAdvisor()
	if cfg.NewGame && !m.s.Settings.TutorialDone {
		m.tut.open = true
	}
	return m
}

func (m *Model) State() *sim.State { return m.s }

func (m *Model) Init() tea.Cmd {
	m.interval = m.desired()
	return m.tick(m.interval)
}

func (m *Model) tick(d time.Duration) tea.Cmd {
	gen := m.gen
	return tea.Tick(d, func(t time.Time) tea.Msg { return frameMsg{gen: gen, t: t} })
}

func fpsInterval(fps int) time.Duration {
	if fps <= 0 {
		fps = 1
	}
	return time.Second / time.Duration(fps)
}

func (m *Model) desired() time.Duration {
	e := &m.b.Engine
	switch {
	case !m.focused:
		return fpsInterval(e.FPSBackground)
	case m.anim.Active() || m.revealing() || m.cer.active() || m.away.open && m.anim.Counter.Moving():
		return fpsInterval(e.FPSAnimation)
	}
	return fpsInterval(e.FPSNormal)
}

func (m *Model) kick() tea.Cmd {
	d := m.desired()
	if m.interval != 0 && d >= m.interval {
		return nil
	}
	m.gen++
	m.interval = d
	return m.tick(d)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.update(msg)
	if m.ring {
		m.ring = false
		if m.s.Settings.Bell {
			cmd = tea.Batch(cmd, tea.Raw(bell))
		}
	}
	return m, cmd
}

func (m *Model) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case frameMsg:
		return m.frame(msg)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
	case tea.FocusMsg:
		m.focused = true
		return m.kick()
	case tea.BlurMsg:
		m.focused = false
	case tea.KeyPressMsg:
		_, cmd := m.key(msg)
		return cmd
	case tea.MouseClickMsg:
		return m.click(msg.Mouse())
	case tea.MouseWheelMsg:
		m.wheel(msg.Mouse())
	}
	return nil
}

func (m *Model) frame(msg frameMsg) tea.Cmd {
	if msg.gen != m.gen {
		return nil
	}
	if m.last.IsZero() {
		m.last = msg.t
	}
	dt := msg.t.Sub(m.last)
	m.last = msg.t
	if dt < 0 {
		dt = 0
	}
	m.advance(dt)
	if dt > 0 {
		m.fps = m.fps*0.8 + 0.2/dt.Seconds()
	}
	m.autosave(msg.t)
	m.maybeChoose()
	if now := m.now(); now.Sub(m.newsAt) >= newsEvery {
		m.newsAt = now
		m.newsIdx++
	}
	m.interval = m.desired()
	return m.tick(m.interval)
}

func (m *Model) advance(dt time.Duration) {
	if dt > catchUpLimit {
		m.catchUp(dt.Seconds(), true)
	} else {
		m.acc += dt
		for m.acc >= m.step {
			m.stepOnce()
			m.acc -= m.step
		}
	}
	m.anim.Money.Set(m.s.Money)
	m.anim.Advance(dt.Seconds())
	m.advanceCeremony(dt.Seconds())
}

func (m *Model) stepOnce() {
	before := m.s.Lifetime.Total
	m.evs = sim.Step(m.s, m.b, m.step.Seconds(), nil, m.evs[:0])
	m.consume(m.evs, false)
	m.track(m.s.Lifetime.Total - before)
	if m.s.Clock-m.advAt >= 1 {
		m.refreshAdvisor()
	}
}

func (m *Model) catchUp(elapsed float64, show bool) {
	res, evs := sim.Offline(m.s, m.b, elapsed, m.evs[:0])
	m.evs = evs
	m.consume(m.evs, true)
	if show && res.Simulated >= m.b.Engine.OfflineStepS {
		m.away = away{res: res, open: true}
		m.anim.Counter.Jump(0)
		m.anim.Counter.Set(res.Earned)
		m.anim.Money.Jump(m.s.Money)
	}
	m.refreshAdvisor()
}

func (m *Model) track(earned float64) {
	m.hist.add(earned, m.step.Seconds())
	m.sparkAcc += earned
	m.sparkSteps++
	if float64(m.sparkSteps)*m.step.Seconds() < 1 {
		return
	}
	v := m.sparkAcc / (float64(m.sparkSteps) * m.step.Seconds())
	copy(m.spark[:], m.spark[1:])
	m.spark[sparkLen-1] = v
	m.sparkN = min(sparkLen, m.sparkN+1)
	m.sparkAcc, m.sparkSteps = 0, 0
}

func (m *Model) rate() float64 {
	n := min(rateWindow, m.sparkN)
	measured := 0.0
	if n > 0 {
		for _, v := range m.spark[sparkLen-n:] {
			measured += v
		}
		measured /= float64(n)
	}
	return max(measured, m.s.Derived.IncomePerSec)
}

func (m *Model) autosave(now time.Time) {
	if m.save == nil || now.Sub(m.lastSave).Seconds() < m.b.Engine.AutosaveS {
		return
	}
	m.lastSave = now
	if err := m.save(m.s); err != nil {
		m.setStatus(m.cat.T("ui.msg.save_failed", err), true)
	}
}

func (m *Model) exec(c sim.Command) tea.Cmd {
	m.cmds = append(m.cmds[:0], c)
	m.evs = sim.Step(m.s, m.b, 0, m.cmds, m.evs[:0])
	m.consume(m.evs, false)
	m.anim.Money.Set(m.s.Money)
	m.refreshAdvisor()
	return m.kick()
}

func (m *Model) setStatus(s string, warn bool) {
	m.status, m.statusAt, m.statusWarn = s, m.now(), warn
	secs := m.toastS(s)
	if over := widgets.Width(s) - m.newsRoom(); m.width > 0 && over > 0 {
		secs = max(secs, 2*tickerPauseS+float64(over)/tickerCPS)
	}
	m.statusFor = time.Duration(secs * float64(time.Second))
}

func readS(s string) float64 { return readPerRuneS * float64(utf8.RuneCountInString(s)) }

func (m *Model) toastS(s string) float64 { return min(readMaxS, readBaseS+readS(s)) }

func (m *Model) addLog(s string) {
	if len(m.log) == logCap {
		copy(m.log, m.log[1:])
		m.log = m.log[:logCap-1]
	}
	m.log = append(m.log, s)
	m.logSeq++
}

func (m *Model) notation() format.Notation { return format.ParseNotation(m.s.Settings.Notation) }

func (m *Model) num(v float64) string { return m.nums.get(v, m.notation(), false) }

func (m *Model) money(v float64) string { return m.nums.get(v, m.notation(), true) }

func (m *Model) bizName(i int) string {
	return m.cat.T(i18n.BusinessKey(m.b.Businesses[i].ID))
}

func (m *Model) upgName(u int) string { return m.cat.T(i18n.UpgradeKey(m.b.Upgrades[u].ID)) }

func (m *Model) unlocked(u sim.Unlock) bool { return u == noUnlock || m.s.IsUnlocked(u) }

func (m *Model) unlockName(u sim.Unlock) string {
	if u.IsBusiness() {
		return m.cat.T("unlock.business", m.bizName(u.Business()))
	}
	return m.cat.T(i18n.UnlockKey(u.Name()))
}

func (m *Model) undoLeft() float64 {
	u := &m.s.Undo
	if u.Kind == sim.PurchaseNone {
		return 0
	}
	return max(0, m.b.Engine.UndoWindowS-(m.s.Clock-u.AtS))
}
