package sim

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/econ"
)

func validGen(b *balance.Balance, gen int) bool { return gen >= 1 && gen <= len(b.GPUGens) }

func RackCost(s *State, b *balance.Balance, gen, k int) float64 {
	if !validGen(b, gen) {
		return math.Inf(1)
	}
	g := &b.GPUGens[gen-1]
	return econ.BulkCost(g.RackCost*s.Derived.RackCostMult, b.Infra.RackR, s.Infra.Racks[gen-1], k)
}

func MaxRacks(s *State, b *balance.Balance, gen int) int {
	if !validGen(b, gen) {
		return 0
	}
	g := &b.GPUGens[gen-1]
	return econ.MaxAffordable(s.Money, g.RackCost*s.Derived.RackCostMult, b.Infra.RackR, s.Infra.Racks[gen-1])
}

func RackSellValue(s *State, b *balance.Balance, gen, k int) float64 {
	if !validGen(b, gen) {
		return 0
	}
	owned := s.Infra.Racks[gen-1]
	k = min(k, owned)
	if k <= 0 {
		return 0
	}
	g := &b.GPUGens[gen-1]
	return b.Infra.RackSellback * econ.BulkCost(g.RackCost*s.CardMods.RackCost, b.Infra.RackR, owned-k, k)
}

func GridCost(b *balance.Balance, mw float64) float64 { return mw * b.Energy.GridMWCost }

func PPACost(b *balance.Balance, mw float64) float64 { return mw * b.Energy.PPAMWCost }

func SMRCost(s *State, b *balance.Balance) float64 {
	return b.Energy.SMRCostMinutes * 60 * pricingIncome(s)
}

func OrbitalCost(s *State, b *balance.Balance) float64 {
	return b.Energy.OrbitalCostMinutes * 60 * pricingIncome(s)
}

func pricingIncome(s *State) float64 {
	return math.Max(s.Derived.BaseIncome, s.Stats.BestIncome)
}

func GridNeededMW(s *State, b *balance.Balance, extraRackMW float64) float64 {
	d := &s.Derived
	e := &s.Infra.Energy
	free := float64(e.SMR)*b.Energy.SMRMW + float64(e.Orbital)*b.Energy.OrbitalMW
	if e.PPA.ExpiresS > s.Clock {
		free += e.PPA.MW
	}
	want := (d.RackMW+extraRackMW-free)/b.Energy.GridPeakAvailability - e.GridMW
	return math.Max(0, want)
}

func buyRack(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockInfra) {
		return reject(out, c, RejectLocked, b.Infra.UnlockLifetime)
	}
	if !validGen(b, c.Gen) {
		return reject(out, c, RejectInvalid, 0)
	}
	if c.Gen > s.Infra.GPUGen {
		return reject(out, c, RejectLocked, b.GPUGens[c.Gen-1].UnlockLifetime)
	}
	k := c.Qty
	if k == QtyMax {
		k = max(1, MaxRacks(s, b, c.Gen))
	}
	k = max(1, k)
	cost := RackCost(s, b, c.Gen, k)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Infra.Racks[c.Gen-1] += k
	s.Stats.Purchases++
	s.Undo = Undo{Kind: PurchaseRack, Business: -1, Gen: c.Gen, Qty: k, Cost: cost, AtS: s.Clock, PrevGlobal: s.GlobalMilestones}
	Recompute(s, b)
	return append(out, Event{Kind: EvRack, Business: -1, Index: c.Gen, Count: k, Amount: cost})
}

func sellRack(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !validGen(b, c.Gen) {
		return reject(out, c, RejectInvalid, 0)
	}
	k := c.Qty
	if k == QtyMax {
		k = s.Infra.Racks[c.Gen-1]
	}
	if k < 1 || s.Infra.Racks[c.Gen-1] < k {
		return reject(out, c, RejectNotOwned, 0)
	}
	v := RackSellValue(s, b, c.Gen, k)
	s.Money += v
	s.Infra.Racks[c.Gen-1] -= k
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvRackSold, Business: -1, Index: c.Gen, Count: k, Amount: v})
}

func buyEnergy(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockInfra) {
		return reject(out, c, RejectLocked, b.Infra.UnlockLifetime)
	}
	e := &s.Infra.Energy
	var kind int
	var cost, mw float64
	switch c.Kind {
	case CmdBuyGrid, CmdBuyPPA:
		if !(c.MW > 0) || math.IsInf(c.MW, 0) {
			return reject(out, c, RejectInvalid, 0)
		}
		mw = c.MW
		kind, cost = EnergyGrid, GridCost(b, mw)
		if c.Kind == CmdBuyPPA {
			kind, cost = EnergyPPA, PPACost(b, mw)
		}
	case CmdBuySMR:
		kind, cost, mw = EnergySMR, SMRCost(s, b), b.Energy.SMRMW
	case CmdBuyOrbital:
		if !s.IsUnlocked(UnlockOrbital) {
			return reject(out, c, RejectLocked, b.Energy.OrbitalUnlockLifetime)
		}
		kind, cost, mw = EnergyOrbital, OrbitalCost(s, b), b.Energy.OrbitalMW
	}
	if !(cost > 0) {
		return reject(out, c, RejectInvalid, 0)
	}
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Stats.Purchases++
	s.Undo = Undo{}
	switch kind {
	case EnergyGrid:
		e.GridMW += mw
		s.Undo = Undo{Kind: PurchaseGrid, Business: -1, MW: mw, Cost: cost, AtS: s.Clock, PrevGlobal: s.GlobalMilestones}
	case EnergyPPA:
		idx := s.Derived.EnergyIndex
		if e.PPA.ExpiresS > s.Clock && e.PPA.MW > 0 {
			idx = (e.PPA.Index*e.PPA.MW + idx*mw) / (e.PPA.MW + mw)
			e.PPA.MW += mw
		} else {
			e.PPA.MW = mw
		}
		e.PPA.Index = idx
		e.PPA.ExpiresS = s.Clock + b.Energy.PPADays*86400
	case EnergySMR:
		e.SMR++
	case EnergyOrbital:
		e.Orbital++
	}
	Recompute(s, b)
	return append(out, Event{Kind: EvEnergy, Business: -1, Index: kind, Amount: cost, Value: mw})
}

func setCloud(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockInfra) {
		return reject(out, c, RejectLocked, b.Infra.UnlockLifetime)
	}
	s.Infra.CloudEnabled = c.On
	Recompute(s, b)
	n := 0
	if c.On {
		n = 1
	}
	return append(out, Event{Kind: EvCloud, Business: -1, Count: n})
}

func grantStarter(s *State, b *balance.Balance, out []Event) []Event {
	if len(b.GPUGens) == 0 {
		return out
	}
	g := &b.GPUGens[0]
	head := b.Infra.StarterHeadroom
	gap := s.Derived.ComputeNeed*head - s.Derived.ComputeCap
	racks := 0
	if gap > 0 || s.Infra.Racks == [balance.MaxGPUGens]int{} {
		racks = max(1, int(math.Ceil(gap/g.Compute)))
	}
	s.Infra.Racks[0] += racks
	s.Infra.StarterGiven = true
	rackMW := 0.0
	for i := range b.GPUGens {
		rackMW += float64(s.Infra.Racks[i]) * b.GPUGens[i].MW
	}
	grid := rackMW * head / b.Energy.GridPeakAvailability
	s.Infra.Energy.GridMW = math.Max(s.Infra.Energy.GridMW, grid)
	return append(out, Event{Kind: EvStarter, Business: -1, Index: 1, Count: racks, Value: s.Infra.Energy.GridMW})
}

func updateGens(s *State, b *balance.Balance, out []Event) []Event {
	if !s.IsUnlocked(UnlockInfra) {
		return out
	}
	for s.Infra.GPUGen < len(b.GPUGens) && s.Lifetime.Total >= b.GPUGens[s.Infra.GPUGen].UnlockLifetime &&
		(!b.GPUGens[s.Infra.GPUGen].Research || s.ResMods.GPUGen > s.Infra.GPUGen) {
		s.Infra.GPUGen++
		out = append(out, Event{Kind: EvGPUGen, Business: -1, Index: s.Infra.GPUGen})
	}
	return out
}
