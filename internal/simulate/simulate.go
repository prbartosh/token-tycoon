// Package simulate runs bots against the simulation to check the balance
// curve (ARCHITECTURE 9): greedy, passive (short daily sessions + offline)
// and random strategies, CSV export and an acceptance report.
package simulate

import (
	"fmt"
	"math"
	"math/rand/v2"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/sim"
)

// Strategy names.
const (
	Greedy  = "greedy"
	Passive = "passive"
	Random  = "random"
)

// Config describes one simulator run.
type Config struct {
	Duration float64 // game seconds to simulate
	Strategy string
	Seed     uint64
	// SessionS is the active time per day for the passive strategy (the rest
	// of each day is offline catch-up). Default 600 (10m/day).
	SessionS float64
	// ComboRate is how many Space presses per second the bot makes while
	// active. 0 (default) disables the combo. A plausible human rate is about
	// 3 to 5 presses/s in short bursts; sustained for hours it is not
	// plausible, so it stays opt-in.
	ComboRate float64
	// ClickLatencyS is how long a business without a manager stays idle
	// before the bot starts its next manual cycle (human reaction plus
	// attention switching). Default 0.5 s.
	ClickLatencyS float64
	// StepS is the simulation step while active. Default 0.1 s (the game's).
	StepS float64
}

func (c *Config) defaults() {
	if c.SessionS <= 0 {
		c.SessionS = 600
	}
	if c.ClickLatencyS <= 0 {
		c.ClickLatencyS = 0.5
	}
	if c.StepS <= 0 {
		c.StepS = 0.1
	}
	if c.Strategy == "" {
		c.Strategy = Greedy
	}
}

// Sample is one CSV row.
type Sample struct {
	T          float64
	Money      float64
	IncomePerS float64
	Lifetime   float64
	Run        float64
	Params     float64
	Capital    float64
	Benchmark  float64
	Share      float64
	Unlocks    int
	Owned      [sim.N]int
	Managers   int
	Upgrades   int
	Combo      float64
	Online     bool
}

// Purchase records one bot purchase.
type Purchase struct {
	T    float64
	Kind sim.PurchaseKind
	Idx  int // business or upgrade index
	Qty  int
	Cost float64
}

// UnlockAt records when an unlock opened.
type UnlockAt struct {
	T      float64
	Unlock sim.Unlock
}

// Session is one active session of the passive strategy.
type Session struct {
	StartT       float64
	FirstUnlockT float64 // seconds after the session start, -1 if none
}

// Result is everything a run produced.
type Result struct {
	Config        Config
	Final         sim.State
	Samples       []Sample
	Purchases     []Purchase
	Unlocks       []UnlockAt
	Sessions      []Session
	FirstManagerT float64 // -1 if not reached
	InfraT        float64 // lifetime >= infra threshold
	PrestigeT     float64 // run >= prestige threshold (and infra)
	LongestFlat   float64 // longest gap without a purchase in the first hour
	LongestFlatAt float64 // start of that gap
	OfflineS      float64
	OfflineEarned float64
	Businesses    int // business purchase commands
	Managers      int
	UpgradesN     int
}

type bot struct {
	cfg     Config
	b       *balance.Balance
	s       sim.State
	res     *Result
	evs     []sim.Event
	cmds    []sim.Command
	idle    [sim.N]float64 // seconds each manual business has been idle
	pressAc float64
	rnd     *rand.Rand
	nextDec float64
	nextSmp float64
	session *Session
	// flatHorizon ends the window of the flat-stretch check: the first hour
	// of play, cut at the end of the first active span (passive strategy).
	flatHorizon float64
}

// Run executes a simulation. It is deterministic for a given config.
func Run(b *balance.Balance, cfg Config) (*Result, error) {
	cfg.defaults()
	switch cfg.Strategy {
	case Greedy, Passive, Random:
	default:
		return nil, fmt.Errorf("simulate: unknown strategy %q", cfg.Strategy)
	}
	if !(cfg.Duration > 0) {
		return nil, fmt.Errorf("simulate: duration must be > 0")
	}
	bt := &bot{
		cfg:  cfg,
		b:    b,
		s:    sim.NewState(b, cfg.Seed),
		res:  &Result{Config: cfg, FirstManagerT: -1, InfraT: -1, PrestigeT: -1},
		evs:  make([]sim.Event, 0, 256),
		cmds: make([]sim.Command, 0, 16),
		rnd:  rand.New(rand.NewPCG(cfg.Seed, 0xB07)),
	}
	if cfg.Strategy == Passive {
		bt.runPassive()
	} else {
		bt.active(cfg.Duration)
	}
	bt.sample(true)
	bt.finish()
	return bt.res, nil
}

func (bt *bot) runPassive() {
	const day = 86400.0
	for t := 0.0; t < bt.cfg.Duration-1e-9; t += day {
		var offEvs []sim.Event
		if t > 0 {
			res, evs := sim.Offline(&bt.s, bt.b, day-bt.cfg.SessionS, bt.evs[:0])
			bt.res.OfflineS += res.Simulated
			bt.res.OfflineEarned += res.Earned
			// The clock only advanced by the simulated (capped) time; the
			// rest of the absence earned nothing.
			bt.s.Clock += res.Elapsed - res.Simulated
			offEvs = evs
		}
		bt.res.Sessions = append(bt.res.Sessions, Session{StartT: bt.s.Clock, FirstUnlockT: -1})
		bt.session = &bt.res.Sessions[len(bt.res.Sessions)-1]
		// Unlocks reached during the absence are shown at session start.
		bt.observe(offEvs, bt.s.Clock)
		if t > 0 {
			bt.sample(true)
		}
		active := math.Min(bt.cfg.SessionS, bt.cfg.Duration-t)
		bt.active(active)
		if t == 0 {
			bt.flatHorizon = bt.s.Clock
		}
		bt.session = nil
	}
}

// active plays for d seconds.
func (bt *bot) active(d float64) {
	end := bt.s.Clock + d
	dt := bt.cfg.StepS
	for bt.s.Clock < end-1e-9 {
		step := math.Min(dt, end-bt.s.Clock)
		bt.cmds = bt.cmds[:0]
		bt.clicks(step)
		if bt.s.Clock >= bt.nextDec {
			bt.decide()
			// Decide often while purchases are cheap and frequent, less later.
			every := 0.5
			if bt.s.Clock > 3600 {
				every = 2
			}
			bt.nextDec = bt.s.Clock + every
		}
		bt.evs = sim.Step(&bt.s, bt.b, step, bt.cmds, bt.evs[:0])
		bt.observe(bt.evs, bt.s.Clock)
		bt.sample(false)
	}
}

// clicks issues manual starts and combo presses for this step.
func (bt *bot) clicks(dt float64) {
	for i := range sim.N {
		bz := &bt.s.Businesses[i]
		if bz.Owned == 0 || bz.Manager || bz.Running {
			bt.idle[i] = 0
			continue
		}
		bt.idle[i] += dt
		if bt.idle[i] >= bt.cfg.ClickLatencyS {
			bt.cmds = append(bt.cmds, sim.Start(i))
			bt.idle[i] = 0
		}
	}
	if bt.cfg.ComboRate > 0 {
		bt.pressAc += bt.cfg.ComboRate * dt
		for bt.pressAc >= 1 {
			bt.cmds = append(bt.cmds, sim.Boost(-1))
			bt.pressAc--
		}
	}
}

func (bt *bot) observe(evs []sim.Event, t float64) {
	for _, e := range evs {
		switch e.Kind {
		case sim.EvUnlock:
			bt.res.Unlocks = append(bt.res.Unlocks, UnlockAt{T: t, Unlock: sim.Unlock(e.Index)})
			if bt.session != nil && bt.session.FirstUnlockT < 0 {
				bt.session.FirstUnlockT = math.Max(0, t-bt.session.StartT)
			}
			switch sim.Unlock(e.Index) {
			case sim.UnlockInfra:
				bt.res.InfraT = t
			case sim.UnlockPrestige1:
				bt.res.PrestigeT = t
			}
		case sim.EvManager:
			if bt.res.FirstManagerT < 0 {
				bt.res.FirstManagerT = t
			}
		}
	}
}

func (bt *bot) sample(force bool) {
	t := bt.s.Clock
	if !force && t < bt.nextSmp {
		return
	}
	every := 10.0
	if t > 3600 {
		every = 60
	}
	if t > 86400 {
		every = 600
	}
	bt.nextSmp = t + every
	s := &bt.s
	sm := Sample{
		T: t, Money: s.Money, IncomePerS: s.Derived.IncomePerSec, Lifetime: s.Lifetime.Total,
		Run: s.Lifetime.Run, Params: s.Prestige.Params, Capital: s.Prestige.Capital,
		Unlocks: s.Unlocks.Count(), Upgrades: s.Upgrades.Count(), Combo: s.Combo.Mult,
		Online: bt.session != nil || bt.cfg.Strategy != Passive,
	}
	for i := range sim.N {
		sm.Owned[i] = s.Businesses[i].Owned
		if s.Businesses[i].Manager {
			sm.Managers++
		}
	}
	bt.res.Samples = append(bt.res.Samples, sm)
}

func (bt *bot) finish() {
	r := bt.res
	r.Final = bt.s
	horizon := math.Min(3600, r.Config.Duration)
	if bt.flatHorizon > 0 {
		horizon = math.Min(horizon, bt.flatHorizon)
	}
	last := 0.0
	for _, p := range r.Purchases {
		if p.T > horizon {
			break
		}
		if g := p.T - last; g > r.LongestFlat {
			r.LongestFlat, r.LongestFlatAt = g, last
		}
		last = p.T
	}
	if g := horizon - last; g > r.LongestFlat {
		r.LongestFlat, r.LongestFlatAt = g, last
	}
}
