package sim

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
)

type mode struct {
	offline    bool
	efficiency float64
}

var online = mode{efficiency: 1}

func Step(s *State, b *balance.Balance, dt float64, cmds []Command, out []Event) []Event {
	return step(s, b, dt, cmds, out, online)
}

func step(s *State, b *balance.Balance, dt float64, cmds []Command, out []Event, md mode) []Event {
	for i := range cmds {
		out = apply(s, b, &cmds[i], out)
	}
	if !(dt > 0) {
		Recompute(s, b)
		return out
	}

	if md.offline {
		s.Combo.Mult, s.Combo.IdleS = 1, 0
	} else if s.Combo.Mult > 1 {
		s.Combo.IdleS += dt
		if s.Combo.IdleS >= b.Combo.DecayAfterS {
			s.Combo.Mult = 1
			out = append(out, Event{Kind: EvComboEnd, Business: -1})
		}
	}

	d := &s.Derived
	combo := max(1, s.Combo.Mult)
	critP, critM := b.Crit.Chance, b.Crit.Mult
	capPerS := b.BusinessDefaults.DrawCapPerS
	earned := 0.0
	for i := range N {
		bz := &s.Businesses[i]
		if bz.Owned == 0 || !(bz.Manager || bz.Running) {
			continue
		}
		bz.DrawBudget = math.Min(bz.DrawBudget+dt*capPerS, capPerS)
		bz.Progress += dt / d.CycleS[i]
		if bz.Progress < 1 {
			continue
		}
		var n float64
		if bz.Manager {
			n = math.Floor(bz.Progress)
			bz.Progress -= n
		} else {
			n = 1
			bz.Progress, bz.Running = 0, false
		}
		rolled := 0.0
		if !md.offline {
			rolled = math.Min(n, math.Floor(bz.DrawBudget))
			bz.DrawBudget -= rolled
		}
		per := d.PerCycle[i] * combo * md.efficiency
		rev := per * (n - rolled) * econ.CritEV(critP, critM)
		if rolled > 0 {
			crits := 0
			for range int(rolled) {
				if s.RNG.Crits.Chance(critP) {
					crits++
				}
			}
			rev += per * rolled
			if crits > 0 {
				bonus := per * float64(crits) * (critM - 1)
				rev += bonus
				s.Stats.Crits += crits
				out = append(out, Event{Kind: EvCrit, Business: i, Count: crits, Amount: per*float64(crits) + bonus})
			}
			out = append(out, Event{Kind: EvCycle, Business: i, Count: int(rolled), Amount: per * rolled})
		}
		earned += rev
	}
	s.Money += earned
	s.Lifetime.Run += earned
	s.Lifetime.Total += earned
	s.Lifetime.Layer2Total += earned
	s.Clock += dt
	if md.offline {
		s.Stats.OfflineS += dt
	} else {
		s.Stats.PlayS += dt
	}
	checkUnlocks(s, b, &out)
	Recompute(s, b)
	return out
}

func reject(out []Event, c *Command, r Reject, amount float64) []Event {
	return append(out, Event{Kind: EvRejected, Business: c.Business, Index: c.Upgrade, Reason: r, Amount: amount, Cmd: c.Kind})
}

func validBusiness(i int) bool { return i >= 0 && i < N }

func apply(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	switch c.Kind {
	case CmdBuyBusiness:
		return buyBusiness(s, b, c, out)
	case CmdBuyManager:
		return buyManager(s, b, c, out)
	case CmdBuyUpgrade:
		return buyUpgrade(s, b, c, out)
	case CmdStart:
		return start(s, c, out, true)
	case CmdBoost:
		s.Combo.Mult = math.Min(max(1, s.Combo.Mult)+b.Combo.Step, b.Combo.MaxMult)
		s.Combo.IdleS = 0
		s.Stats.BoostPresses++
		s.Stats.BestCombo = math.Max(s.Stats.BestCombo, s.Combo.Mult)
		out = append(out, Event{Kind: EvCombo, Business: c.Business, Amount: s.Combo.Mult})
		if validBusiness(c.Business) {
			out = start(s, c, out, false)
		}
		s.Derived.ComboMult = s.Combo.Mult
		return out
	case CmdPrimary:
		if !validBusiness(c.Business) {
			return reject(out, c, RejectInvalid, 0)
		}
		_, cost := ResolveQty(s, b, c.Business, c.Qty)
		bz := &s.Businesses[c.Business]
		if cost > s.Money && bz.Owned > 0 && !bz.Manager && !bz.Running &&
			s.IsUnlocked(UnlockBusiness0+Unlock(c.Business)) {
			return start(s, c, out, true)
		}
		return buyBusiness(s, b, c, out)
	case CmdUndo:
		return undo(s, b, c, out)
	}
	return reject(out, c, RejectInvalid, 0)
}

func start(s *State, c *Command, out []Event, report bool) []Event {
	if !validBusiness(c.Business) {
		if report {
			return reject(out, c, RejectInvalid, 0)
		}
		return out
	}
	bz := &s.Businesses[c.Business]
	switch {
	case bz.Owned == 0:
		if report {
			out = reject(out, c, RejectNotOwned, 0)
		}
		return out
	case bz.Manager || bz.Running:
		if report {
			out = reject(out, c, RejectBusy, 0)
		}
		return out
	}
	bz.Running = true
	s.Stats.ManualStarts++
	return append(out, Event{Kind: EvStart, Business: c.Business})
}

func buyBusiness(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := c.Business
	if !validBusiness(i) {
		return reject(out, c, RejectInvalid, 0)
	}
	if !s.IsUnlocked(UnlockBusiness0 + Unlock(i)) {
		return reject(out, c, RejectLocked, b.Businesses[i].UnlockLifetime)
	}
	k, cost := ResolveQty(s, b, i, c.Qty)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	before := s.Derived
	s.Money -= cost
	s.Businesses[i].Owned += k
	s.Stats.Purchases++
	s.Undo = Undo{Kind: PurchaseBusiness, Business: i, Qty: k, Cost: cost, AtS: s.Clock}
	Recompute(s, b)
	out = append(out, Event{Kind: EvPurchase, Business: i, Count: k, Amount: cost})
	return milestoneEvents(s, b, &before, i, out)
}

func milestoneEvents(s *State, b *balance.Balance, before *Derived, i int, out []Event) []Event {
	d := &s.Derived
	m := &b.Milestones
	if d.SpeedMS[i] > before.SpeedMS[i] {
		out = append(out, Event{Kind: EvMilestoneSpeed, Business: i, Index: m.SpeedThresholds[d.SpeedMS[i]-1],
			Amount: econ.MultPow(m.SpeedMult, d.SpeedMS[i])})
	}
	if d.ProfitMS[i] > before.ProfitMS[i] {
		th := m.ProfitStart + (d.ProfitMS[i]-1)*m.ProfitEvery
		out = append(out, Event{Kind: EvMilestoneProfit, Business: i, Index: th,
			Amount: econ.MultPow(m.ProfitMult, d.ProfitMS[i])})
	}
	if d.GlobalMS > before.GlobalMS {
		th := econ.GlobalThreshold(d.GlobalMS, m.GlobalThresholds, m.GlobalEvery)
		out = append(out, Event{Kind: EvMilestoneGlobal, Business: -1, Index: th,
			Amount: econ.MultPow(m.GlobalMult, d.GlobalMS)})
	}
	return out
}

func buyManager(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := c.Business
	if !validBusiness(i) {
		return reject(out, c, RejectInvalid, 0)
	}
	bz := &s.Businesses[i]
	if !s.IsUnlocked(UnlockBusiness0 + Unlock(i)) {
		return reject(out, c, RejectLocked, b.Businesses[i].UnlockLifetime)
	}
	if bz.Manager {
		return reject(out, c, RejectAlreadyOwned, 0)
	}
	if bz.Owned == 0 {
		return reject(out, c, RejectNotOwned, 0)
	}
	cost := b.ManagerCost(i)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	bz.Manager = true
	bz.Running = false
	s.Stats.Purchases++
	s.Undo = Undo{Kind: PurchaseManager, Business: i, Cost: cost, AtS: s.Clock}
	Recompute(s, b)
	return append(out, Event{Kind: EvManager, Business: i, Amount: cost})
}

func buyUpgrade(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	u := c.Upgrade
	if u < 0 || u >= len(b.Upgrades) {
		return reject(out, c, RejectInvalid, 0)
	}
	if s.Upgrades.Has(u) {
		return reject(out, c, RejectAlreadyOwned, 0)
	}
	cost := b.Upgrades[u].Cost
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Upgrades.Set(u)
	s.Stats.Purchases++
	s.Undo = Undo{Kind: PurchaseUpgrade, Business: -1, Upgrade: u, Cost: cost, AtS: s.Clock}
	Recompute(s, b)
	return append(out, Event{Kind: EvUpgrade, Business: -1, Index: u, Amount: cost})
}

func undo(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	u := s.Undo
	if u.Kind == PurchaseNone || s.Clock-u.AtS > b.Engine.UndoWindowS {
		s.Undo = Undo{}
		return reject(out, c, RejectNothingToUndo, 0)
	}
	switch u.Kind {
	case PurchaseBusiness:
		bz := &s.Businesses[u.Business]
		bz.Owned -= u.Qty
		if bz.Owned <= 0 {
			bz.Owned, bz.Progress, bz.Running = 0, 0, false
		}
	case PurchaseManager:
		s.Businesses[u.Business].Manager = false
	case PurchaseUpgrade:
		s.Upgrades.Clear(u.Upgrade)
	}
	s.Money += u.Cost
	s.Stats.Purchases--
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvUndo, Business: u.Business, Index: u.Upgrade, Count: u.Qty, Amount: u.Cost})
}
