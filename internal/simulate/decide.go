package simulate

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
	"token-tycoon/internal/sim"
)

type candidate struct {
	cmds [2]sim.Command
	n    int
	kind sim.PurchaseKind
	idx  int
	qty  int
	cost float64
	gain float64
}

func one(c sim.Command) ([2]sim.Command, int) { return [2]sim.Command{c}, 1 }

func (bt *bot) effRate(h *sim.Hypo, managers *[sim.N]bool, d *sim.Derived) float64 {
	sim.Evaluate(bt.b, h, d)
	r := 0.0
	for i := range sim.N {
		if h.Owned[i] == 0 {
			continue
		}
		if managers[i] {
			r += d.PerSec[i]
		} else {
			c := d.CycleS[i]
			r += d.PerSec[i] * c / (c + bt.cfg.ClickLatencyS)
		}
	}
	return r
}

func (bt *bot) managers() [sim.N]bool {
	var mgr [sim.N]bool
	for i := range sim.N {
		mgr[i] = bt.s.Businesses[i].Manager
	}
	return mgr
}

func (bt *bot) candidates(out []candidate) ([]candidate, float64) {
	s := &bt.s
	b := bt.b
	mgr := bt.managers()
	base := sim.HypoOf(s)
	var d sim.Derived
	r0 := bt.effRate(&base, &mgr, &d)
	m := &b.Milestones
	for i := range sim.N {
		if !s.IsUnlocked(sim.UnlockBusiness0 + sim.Unlock(i)) {
			continue
		}
		owned := s.Businesses[i].Owned
		next := econ.NextBusinessThreshold(owned, m.SpeedThresholds, m.ProfitStart, m.ProfitEvery)
		qtys := [2]int{1, next - owned}
		for qi, k := range qtys {
			if k < 1 || (qi == 1 && k == 1) || k > 100 {
				continue
			}
			h := base
			h.Owned[i] += k
			gain := bt.effRate(&h, &mgr, &d) - r0
			c := candidate{kind: sim.PurchaseBusiness, idx: i, qty: k, cost: sim.BusinessCost(s, b, i, k), gain: gain}
			c.cmds, c.n = one(sim.Buy(i, k))
			out = append(out, c)
		}
		if owned > 0 && !mgr[i] {
			mm := mgr
			mm[i] = true
			gain := bt.effRate(&base, &mm, &d) - r0
			c := candidate{kind: sim.PurchaseManager, idx: i, cost: b.ManagerCost(i), gain: gain}
			c.cmds, c.n = one(sim.HireManager(i))
			out = append(out, c)
		}
	}
	for ui := range b.Upgrades {
		if s.Upgrades.Has(ui) {
			continue
		}
		h := base
		h.Upgrades.Set(ui)
		gain := bt.effRate(&h, &mgr, &d) - r0
		if b.Upgrades[ui].Effect == balance.EffectCost {
			gain = r0 * (1 - b.Upgrades[ui].Value)
		}
		c := candidate{kind: sim.PurchaseUpgrade, idx: ui, cost: b.Upgrades[ui].Cost, gain: gain}
		c.cmds, c.n = one(sim.BuyUpgrade(ui))
		out = append(out, c)
	}
	if s.IsUnlocked(sim.UnlockInfra) {
		out = bt.infraCandidates(out, &base, &mgr, r0)
	}
	if s.IsUnlocked(sim.UnlockRegions) {
		out = bt.regionCandidates(out, r0)
	}
	return out, r0
}

func (bt *bot) infraCandidates(out []candidate, base *sim.Hypo, mgr *[sim.N]bool, r0 float64) []candidate {
	s := &bt.s
	b := bt.b
	sd := &s.Derived
	var d sim.Derived
	deficit := sd.ComputeNeed*1.1 - sd.ComputeCap
	for gen := 1; deficit > 0 && gen <= s.Infra.GPUGen; gen++ {
		g := &b.GPUGens[gen-1]
		qtys := [2]int{1, 0}
		if deficit > g.Compute {
			qtys[1] = min(200, int(math.Ceil(deficit/g.Compute)))
		}
		for _, k := range qtys {
			if k < 1 {
				continue
			}
			mw := sim.GridNeededMW(s, b, float64(k)*g.MW)
			h := *base
			h.Infra.Racks[gen-1] += k
			h.Infra.Energy.GridMW += mw
			gain := bt.effRate(&h, mgr, &d) - r0
			c := candidate{kind: sim.PurchaseRack, idx: gen, qty: k, cost: sim.RackCost(s, b, gen, k) + sim.GridCost(b, mw), gain: gain}
			c.cmds[0], c.n = sim.BuyRack(gen, k), 1
			if mw > 0 {
				c.cmds[1], c.n = sim.BuyGrid(mw), 2
			}
			out = append(out, c)
		}
	}
	if mw := sim.GridNeededMW(s, b, 0); mw > 0 {
		h := *base
		h.Infra.Energy.GridMW += mw
		gain := bt.effRate(&h, mgr, &d) - r0
		c := candidate{kind: sim.PurchaseGrid, cost: sim.GridCost(b, mw), gain: gain}
		c.cmds, c.n = one(sim.BuyGrid(mw))
		out = append(out, c)
	}
	if sd.EnergyUsed > sd.FreeUsed {
		h := *base
		h.Infra.Energy.SMR++
		gain := bt.effRate(&h, mgr, &d) - r0
		c := candidate{kind: sim.PurchaseGrid, idx: sim.EnergySMR, cost: sim.SMRCost(s, b), gain: gain}
		c.cmds, c.n = one(sim.BuySMR())
		out = append(out, c)
	}
	return out
}

func score(c *candidate, money, rate float64) float64 {
	if !(c.gain > 0) {
		return math.Inf(1)
	}
	wait := 0.0
	if c.cost > money {
		if !(rate > 0) {
			return math.Inf(1)
		}
		wait = (c.cost - money) / rate
	}
	return wait + c.cost/c.gain
}

func (bt *bot) maintainInfra() {
	s := &bt.s
	if !s.IsUnlocked(sim.UnlockInfra) {
		return
	}
	mgr := bt.managers()
	var d sim.Derived
	h := sim.HypoOf(s)
	r0 := bt.effRate(&h, &mgr, &d)
	h.Infra.CloudEnabled = !s.Infra.CloudEnabled
	r1 := bt.effRate(&h, &mgr, &d)
	if r1 > r0*1.01 {
		bt.exec(sim.SetCloud(h.Infra.CloudEnabled))
	}
	e := &s.Infra.Energy
	need := s.Derived.RackMW
	if e.PPA.ExpiresS <= s.Clock && need > 0 && s.Derived.EnergyIndex < 35 {
		if cost := sim.PPACost(bt.b, need); cost < s.Money*0.2 {
			bt.exec(sim.BuyPPA(need))
		}
	}
}

const minRunS = 600

func (bt *bot) maybePrestige() bool {
	pv := sim.Preview(&bt.s, bt.b)
	if !pv.Allowed {
		return false
	}
	ratio := pv.BonusAfter / pv.BonusNow
	if pv.RunS < minRunS || (ratio < 2 && !(ratio >= 1.3 && pv.ReturnS <= 0.35*pv.RunS)) || len(bt.s.Contracts) > 0 {
		return false
	}
	bt.exec(sim.PrestigeReset())
	for _, e := range bt.evs {
		if e.Kind == sim.EvPrestige {
			bt.res.Prestiges = append(bt.res.Prestiges, PrestigeAt{T: bt.s.Clock, Gain: e.Amount, RunS: pv.RunS, ReturnS: pv.ReturnS})
			bt.idle = [sim.N]float64{}
			return true
		}
	}
	return false
}

func (bt *bot) exec(c sim.Command) {
	bt.evs = sim.Step(&bt.s, bt.b, 0, []sim.Command{c}, bt.evs[:0])
	bt.observe(bt.evs, bt.s.Clock)
}

func (bt *bot) decide() {
	if bt.maybePrestige() {
		bt.maybeFunding()
		return
	}
	if bt.maybeFunding() {
		return
	}
	bt.maintainInfra()
	bt.maintainMarket()
	bt.maintainTraining()
	bt.maintainM4()
	if bt.maintainAGI() {
		return
	}
	var buf [128]candidate
	for range 50 {
		cands, rate := bt.candidates(buf[:0])
		var pick *candidate
		switch bt.cfg.Strategy {
		case Random:
			var afford []int
			for i := range cands {
				if cands[i].cost <= bt.s.Money && cands[i].gain > 0 {
					afford = append(afford, i)
				}
			}
			if len(afford) == 0 || bt.rnd.Float64() < 0.3 {
				return
			}
			pick = &cands[afford[bt.rnd.IntN(len(afford))]]
		default:
			best := math.Inf(1)
			for i := range cands {
				if sc := score(&cands[i], bt.s.Money, rate); sc < best {
					best, pick = sc, &cands[i]
				}
			}
			if pick == nil {
				return
			}
			if pick.cost > bt.s.Money {
				wait := (pick.cost - bt.s.Money) / rate
				var filler *candidate
				fb := wait
				for i := range cands {
					c := &cands[i]
					if c.cost <= bt.s.Money && c.gain > 0 {
						if pb := c.cost / c.gain; pb < fb {
							fb, filler = pb, c
						}
					}
				}
				if filler == nil {
					return
				}
				pick = filler
			}
		}
		if pick.cost > bt.s.Money {
			return
		}
		bt.evs = sim.Step(&bt.s, bt.b, 0, pick.cmds[:pick.n], bt.evs[:0])
		ok := false
		for _, e := range bt.evs {
			switch e.Kind {
			case sim.EvPurchase:
				ok = true
				bt.res.Businesses++
			case sim.EvManager:
				ok = true
				bt.res.Managers++
			case sim.EvUpgrade:
				ok = true
				bt.res.UpgradesN++
			case sim.EvRack:
				ok = true
				bt.res.Racks += e.Count
			case sim.EvEnergy:
				ok = true
				bt.res.EnergyBuys++
			case sim.EvRegion, sim.EvCompliance:
				ok = true
			}
		}
		bt.observe(bt.evs, bt.s.Clock)
		if !ok {
			return
		}
		bt.res.Purchases = append(bt.res.Purchases, Purchase{T: bt.s.Clock, Kind: pick.kind, Idx: pick.idx, Qty: pick.qty, Cost: pick.cost})
	}
}
