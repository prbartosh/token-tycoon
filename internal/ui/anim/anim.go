// Package anim holds the UI animation primitives: value tweens, row flashes,
// floating numbers and spring toasts. It knows nothing about game rules; the
// UI translates simulation events into calls on an Animator.
package anim

import (
	"math"

	"github.com/charmbracelet/harmonica"
)

type Kind uint8

const (
	KindNone Kind = iota
	KindPurchase
	KindMilestone
	KindCrit
	KindMoney
	KindInfo
	KindWarn
	KindRare
	KindEpic
	KindReveal
	KindLegend
)

const (
	MaxSlots     = 32
	MaxFloaters  = 64
	MaxToasts    = 4
	refFPS       = 30
	defaultRate  = 0.2
	toastFreq    = 14
	toastDamping = 0.8
)

type Tween struct {
	Value  float64
	Target float64
	Rate   float64
}

func (t *Tween) Set(v float64) { t.Target = v }

func (t *Tween) Jump(v float64) { t.Value, t.Target = v, v }

func (t *Tween) Advance(dt float64) {
	if !(dt > 0) {
		return
	}
	rate := t.Rate
	if !(rate > 0 && rate < 1) {
		rate = defaultRate
	}
	t.Value += (t.Target - t.Value) * (1 - math.Pow(1-rate, dt*refFPS))
	if math.Abs(t.Target-t.Value) <= math.Max(1e-4*math.Abs(t.Target), 0.005) {
		t.Value = t.Target
	}
}

func (t *Tween) Moving() bool {
	gap := math.Abs(t.Target - t.Value)
	return gap > 0.5 && gap > 0.01*math.Abs(t.Target)
}

type Flash struct {
	Kind Kind
	Left float64
	Dur  float64
}

type Flashes struct{ slots [MaxSlots]Flash }

func (f *Flashes) Start(slot int, k Kind, dur float64) {
	if slot < 0 || slot >= MaxSlots || !(dur > 0) {
		return
	}
	f.slots[slot] = Flash{Kind: k, Left: dur, Dur: dur}
}

func (f *Flashes) At(slot int) (Kind, float64) {
	if slot < 0 || slot >= MaxSlots {
		return KindNone, 0
	}
	s := &f.slots[slot]
	if s.Left <= 0 {
		return KindNone, 0
	}
	return s.Kind, s.Left / s.Dur
}

func (f *Flashes) Advance(dt float64) {
	for i := range f.slots {
		if f.slots[i].Left > 0 {
			f.slots[i].Left -= dt
		}
	}
}

func (f *Flashes) Active() bool {
	for i := range f.slots {
		if f.slots[i].Left > 0 {
			return true
		}
	}
	return false
}

type Floater struct {
	Slot int
	Text string
	Kind Kind
	Age  float64
	Life float64
	Rise int
}

func (f *Floater) Alive() bool { return f.Life > 0 && f.Age < f.Life }

func (f *Floater) Progress() float64 {
	if !(f.Life > 0) {
		return 1
	}
	return math.Min(1, f.Age/f.Life)
}

func (f *Floater) Offset() int { return int(math.Round(f.Progress() * float64(f.Rise))) }

type Floaters struct {
	pool [MaxFloaters]Floater
	next int
}

func (fs *Floaters) Spawn(slot int, text string, k Kind, life float64, rise int) {
	for i := range fs.pool {
		if !fs.pool[i].Alive() {
			fs.pool[i] = Floater{Slot: slot, Text: text, Kind: k, Life: life, Rise: rise}
			return
		}
	}
	fs.pool[fs.next] = Floater{Slot: slot, Text: text, Kind: k, Life: life, Rise: rise}
	fs.next = (fs.next + 1) % MaxFloaters
}

func (fs *Floaters) Advance(dt float64) {
	for i := range fs.pool {
		if fs.pool[i].Alive() {
			fs.pool[i].Age += dt
		}
	}
}

func (fs *Floaters) Items() []Floater { return fs.pool[:] }

func (fs *Floaters) Active() bool {
	for i := range fs.pool {
		if fs.pool[i].Alive() {
			return true
		}
	}
	return false
}

type Toast struct {
	Text string
	Kind Kind
	Pos  float64
	Vel  float64
	Age  float64
	Life float64
}

func (t *Toast) target() float64 {
	if t.Age >= t.Life {
		return 1
	}
	return 0
}

func (t *Toast) moving() bool {
	return math.Abs(t.Pos-t.target()) > 0.01 || math.Abs(t.Vel) > 0.05
}

type Toasts struct {
	items [MaxToasts]Toast
	n     int
}

func (ts *Toasts) Push(text string, k Kind, life float64) {
	if ts.n == MaxToasts {
		copy(ts.items[:], ts.items[1:])
		ts.n--
	}
	ts.items[ts.n] = Toast{Text: text, Kind: k, Pos: 1, Life: life}
	ts.n++
}

func (ts *Toasts) Advance(dt float64) {
	if !(dt > 0) {
		return
	}
	sp := harmonica.NewSpring(dt, toastFreq, toastDamping)
	w := 0
	for i := range ts.n {
		t := ts.items[i]
		t.Age += dt
		t.Pos, t.Vel = sp.Update(t.Pos, t.Vel, t.target())
		if t.Age >= t.Life && t.Pos >= 0.99 {
			continue
		}
		ts.items[w] = t
		w++
	}
	ts.n = w
}

func (ts *Toasts) Items() []Toast { return ts.items[:ts.n] }

func (ts *Toasts) Active() bool {
	for i := range ts.n {
		if ts.items[i].moving() {
			return true
		}
	}
	return false
}

type Animator struct {
	Money   Tween
	Counter Tween
	Flash   Flashes
	Float   Floaters
	Toast   Toasts
	Pulse   float64
}

func (a *Animator) Advance(dt float64) {
	if !(dt > 0) {
		return
	}
	a.Money.Advance(dt)
	a.Counter.Advance(dt)
	a.Flash.Advance(dt)
	a.Float.Advance(dt)
	a.Toast.Advance(dt)
	if a.Pulse > 0 {
		a.Pulse = math.Max(0, a.Pulse-dt)
	}
}

func (a *Animator) Active() bool {
	return a.Money.Moving() || a.Counter.Moving() || a.Pulse > 0 ||
		a.Flash.Active() || a.Float.Active() || a.Toast.Active()
}
