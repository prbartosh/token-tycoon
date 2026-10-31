package sim

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

type TalentView struct {
	Count       int
	Speed       float64
	Mult        float64
	Salary      float64
	LossMult    float64
	HypePerS    float64
	ComputeMult float64
}

type Poach struct {
	Active  bool    `json:"active"`
	Index   int     `json:"index"`
	EndsAtS float64 `json:"ends_at_s"`
}

func computeTalent(s *State, b *balance.Balance) {
	v := TalentView{Mult: 1, LossMult: 1, ComputeMult: 1}
	for k := range s.Talent {
		r := &s.Talent[k]
		ci, ok := b.TalentClassIndex(r.Class)
		if !ok {
			continue
		}
		c := &b.TalentClasses[ci]
		v.Count++
		v.Speed += c.Speed
		sal := c.SalaryShare
		if r.RaiseUntilS > s.Clock {
			sal *= b.Talent.PoachCounterMult
		}
		v.Salary += sal
		if ti, ok := b.TraitIndex(r.Trait); ok {
			t := &b.TalentTraits[ti]
			v.Speed += t.Speed
			v.HypePerS += t.HypePerH / 3600
			if t.LossMult > 0 {
				v.LossMult *= t.LossMult
			}
			if t.ComputeMult > 0 {
				v.ComputeMult *= t.ComputeMult
			}
		}
	}
	v.Mult = math.Max(0.1, 1+v.Speed)
	v.Salary = math.Min(0.9, v.Salary)
	s.Derived.Talent = v
}

func HireCost(s *State, b *balance.Balance, class int) float64 {
	if class < 0 || class >= len(b.TalentClasses) {
		return math.Inf(1)
	}
	return b.TalentClasses[class].HireMinutes * 60 * pricingIncome(s)
}

func hire(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockTalent) {
		return reject(out, c, RejectLocked, b.Talent.UnlockLifetime)
	}
	ci := c.Index
	if ci < 0 || ci >= len(b.TalentClasses) {
		return reject(out, c, RejectInvalid, 0)
	}
	if len(s.Talent) >= b.Talent.MaxStaff {
		return reject(out, c, RejectTeamFull, 0)
	}
	cost := HireCost(s, b, ci)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	cl := &b.TalentClasses[ci]
	r := Researcher{Class: cl.ID}
	names := b.Talent.Names
	r.Name = names[s.RNG.Misc.IntN(len(names))]
	trait := -1
	if cl.Trait && len(b.TalentTraits) > 0 {
		trait = s.RNG.Misc.IntN(len(b.TalentTraits))
		r.Trait = b.TalentTraits[trait].ID
	}
	s.Talent = append(s.Talent, r)
	s.Stats.Hired++
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvHired, Business: -1, Index: len(s.Talent) - 1, Count: trait, Amount: cost})
}

func fire(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := c.Index
	if i < 0 || i >= len(s.Talent) {
		return reject(out, c, RejectInvalid, 0)
	}
	removeResearcher(s, i)
	Recompute(s, b)
	return append(out, Event{Kind: EvFired, Business: -1, Index: i})
}

func removeResearcher(s *State, i int) {
	s.Talent = append(s.Talent[:i], s.Talent[i+1:]...)
	p := &s.Poach
	switch {
	case !p.Active:
	case p.Index == i:
		*p = Poach{}
	case p.Index > i:
		p.Index--
	}
}

func counterOffer(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	p := &s.Poach
	if !p.Active || p.Index < 0 || p.Index >= len(s.Talent) || s.Clock >= p.EndsAtS {
		return reject(out, c, RejectNoEvent, 0)
	}
	r := &s.Talent[p.Index]
	r.RaiseUntilS = s.Clock + b.Talent.PoachCounterDays*86400
	i := p.Index
	*p = Poach{}
	s.Stats.Retained++
	Recompute(s, b)
	return append(out, Event{Kind: EvCounterOffer, Business: -1, Index: i, Value: r.RaiseUntilS - s.Clock})
}

func tickTalent(s *State, b *balance.Balance, dt float64, out []Event, md mode) []Event {
	p := &s.Poach
	if p.Active {
		switch {
		case md.offline || p.Index < 0 || p.Index >= len(s.Talent):
			*p = Poach{}
		case s.Clock+dt >= p.EndsAtS:
			i := p.Index
			*p = Poach{}
			removeResearcher(s, i)
			s.Stats.Poached++
			out = append(out, Event{Kind: EvPoached, Business: -1, Index: i})
			Recompute(s, b)
		}
		return out
	}
	if md.offline || len(s.Talent) == 0 || !s.IsUnlocked(UnlockTalent) {
		return out
	}
	rate := b.Talent.PoachChancePerH / 3600
	if !(rate > 0) || !s.RNG.Misc.Chance(1-math.Exp(-rate*dt)) {
		return out
	}
	i := s.RNG.Misc.IntN(len(s.Talent))
	*p = Poach{Active: true, Index: i, EndsAtS: s.Clock + b.Talent.PoachWindowS}
	return append(out, Event{Kind: EvPoach, Business: -1, Index: i, Value: b.Talent.PoachWindowS})
}

func PoachLeft(s *State) float64 {
	if !s.Poach.Active {
		return 0
	}
	return math.Max(0, s.Poach.EndsAtS-s.Clock)
}

func normalizeTalent(s *State, b *balance.Balance) {
	keep := s.Talent[:0]
	for _, r := range s.Talent {
		if _, ok := b.TalentClassIndex(r.Class); ok {
			if _, ok := b.TraitIndex(r.Trait); !ok {
				r.Trait = ""
			}
			if !(r.RaiseUntilS >= 0) || math.IsInf(r.RaiseUntilS, 0) {
				r.RaiseUntilS = 0
			}
			keep = append(keep, r)
		}
	}
	s.Talent = keep
	if len(s.Talent) > b.Talent.MaxStaff {
		s.Talent = s.Talent[:b.Talent.MaxStaff]
	}
	if s.Talent == nil {
		s.Talent = make([]Researcher, 0, b.Talent.MaxStaff)
	}
	if p := &s.Poach; p.Active && (p.Index < 0 || p.Index >= len(s.Talent)) {
		*p = Poach{}
	}
}
