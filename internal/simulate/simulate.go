// Package simulate runs bots against the simulation to check the balance
// curve (ARCHITECTURE 9): greedy, passive (short daily sessions + offline)
// and random strategies, CSV export and an acceptance report. Bots buy
// businesses and infrastructure, prestige by the preview, react to events with
// a human-like probability and delay, and claim the daily grant.
package simulate

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/sim"
)

const (
	Greedy  = "greedy"
	Passive = "passive"
	Random  = "random"
)

type Config struct {
	Duration      float64
	Strategy      string
	Seed          uint64
	SessionS      float64
	ComboRate     float64
	ClickLatencyS float64
	StepS         float64
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
	Throttle   float64
	Cards      int
	Prestiges  int
}

type Purchase struct {
	T    float64
	Kind sim.PurchaseKind
	Idx  int
	Qty  int
	Cost float64
}

type UnlockAt struct {
	T      float64
	Unlock sim.Unlock
}

type PrestigeAt struct {
	T       float64
	Gain    float64
	RunS    float64
	ReturnS float64
}

type Session struct {
	StartT       float64
	FirstUnlockT float64
}

type Result struct {
	Config        Config
	Final         sim.State
	Samples       []Sample
	Purchases     []Purchase
	Unlocks       []UnlockAt
	Sessions      []Session
	FirstManagerT float64
	InfraT        float64
	PrestigeT     float64
	LongestFlat   float64
	LongestFlatAt float64
	OfflineS      float64
	OfflineEarned float64
	Businesses    int
	Managers      int
	UpgradesN     int
	Racks         int
	EnergyBuys    int
	Prestiges     []PrestigeAt
	Reactions     int
	ReactTried    int
	InfraS        float64
	ThrottleS     float64
	ThrottledS    float64
	CloudS        float64
}

var grantEpoch = time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)

type pendingReact struct {
	at  float64
	key string
}

type bot struct {
	cfg         Config
	b           *balance.Balance
	s           sim.State
	res         *Result
	evs         []sim.Event
	cmds        []sim.Command
	idle        [sim.N]float64
	pressAc     float64
	rnd         *rand.Rand
	nextDec     float64
	nextSmp     float64
	session     *Session
	flatHorizon float64
	reacts      []pendingReact
	grantDay    int
}

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
		cfg:      cfg,
		b:        b,
		s:        sim.NewState(b, cfg.Seed),
		res:      &Result{Config: cfg, FirstManagerT: -1, InfraT: -1, PrestigeT: -1},
		evs:      make([]sim.Event, 0, 256),
		cmds:     make([]sim.Command, 0, 16),
		rnd:      rand.New(rand.NewPCG(cfg.Seed, 0xB07)),
		grantDay: -1,
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
			bt.s.Clock += res.Elapsed - res.Simulated
			offEvs = evs
		}
		bt.res.Sessions = append(bt.res.Sessions, Session{StartT: bt.s.Clock, FirstUnlockT: -1})
		bt.session = &bt.res.Sessions[len(bt.res.Sessions)-1]
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

func (bt *bot) active(d float64) {
	end := bt.s.Clock + d
	dt := bt.cfg.StepS
	for bt.s.Clock < end-1e-9 {
		step := math.Min(dt, end-bt.s.Clock)
		bt.cmds = bt.cmds[:0]
		bt.clicks(step)
		bt.reactions()
		bt.grant()
		if bt.s.Clock >= bt.nextDec {
			bt.decide()
			every := 0.5
			if bt.s.Clock-bt.s.Prestige.RunStartS > 3600 {
				every = 2
			}
			bt.nextDec = bt.s.Clock + every
		}
		bt.evs = sim.Step(&bt.s, bt.b, step, bt.cmds, bt.evs[:0])
		bt.observe(bt.evs, bt.s.Clock)
		bt.track(step)
		bt.sample(false)
	}
}

func (bt *bot) track(dt float64) {
	if !bt.s.IsUnlocked(sim.UnlockInfra) {
		return
	}
	d := &bt.s.Derived
	bt.res.InfraS += dt
	bt.res.ThrottleS += d.Throttle * dt
	if d.Throttle < 0.95 {
		bt.res.ThrottledS += dt
	}
	if bt.s.Infra.CloudEnabled {
		bt.res.CloudS += dt
	}
}

func (bt *bot) reactProfile() (p, lo, hi float64) {
	switch bt.cfg.Strategy {
	case Passive:
		return 0.75, 0.8, 5
	case Random:
		return 0.5, 0.5, 8
	}
	return 0.85, 0.6, 3.5
}

func (bt *bot) reactions() {
	keep := bt.reacts[:0]
	for _, r := range bt.reacts {
		if bt.s.Clock >= r.at {
			bt.cmds = append(bt.cmds, sim.React(r.key))
			continue
		}
		keep = append(keep, r)
	}
	bt.reacts = keep
}

func (bt *bot) grant() {
	day := int(bt.s.Clock / 86400)
	if day <= bt.grantDay || !(bt.s.Derived.BaseIncome > 0) {
		return
	}
	bt.grantDay = day
	bt.cmds = append(bt.cmds, sim.ClaimGrant(grantEpoch.AddDate(0, 0, day).Format("2006-01-02")))
}

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
		case sim.EvEventStart:
			ev := &bt.b.Event[e.Index]
			if ev.Effect != balance.EventReaction || (bt.cfg.Strategy == Passive && bt.session == nil) {
				continue
			}
			bt.res.ReactTried++
			p, lo, hi := bt.reactProfile()
			if bt.rnd.Float64() < p {
				bt.reacts = append(bt.reacts, pendingReact{at: t + lo + (hi-lo)*bt.rnd.Float64(), key: ev.Key})
			}
		case sim.EvReact:
			bt.res.Reactions++
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
		Online:   bt.session != nil || bt.cfg.Strategy != Passive,
		Throttle: s.Derived.Throttle, Cards: s.CardMods.Unique, Prestiges: s.Prestige.ResetsL1,
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
