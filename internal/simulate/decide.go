package simulate

import (
	"math"

	"token-tycoon/internal/balance"

	"token-tycoon/internal/econ"
	"token-tycoon/internal/sim"
)

type candidate struct {
	cmd  sim.Command
	kind sim.PurchaseKind
	idx  int
	qty  int
	cost float64
	gain float64 // increase of the bot's effective $/s
}

// effRate is the bot's effective income for a hypothetical state: automated
// businesses at full expected value, manual ones at the efficiency the bot's
// click latency allows (cycle / (cycle + latency)).
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

// candidates lists purchases worth considering: one unit and "up to the next
// milestone" of every unlocked business, every missing manager and upgrade.
func (bt *bot) candidates(out []candidate) ([]candidate, float64) {
	s := &bt.s
	b := bt.b
	var mgr [sim.N]bool
	for i := range sim.N {
		mgr[i] = s.Businesses[i].Manager
	}
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
			out = append(out, candidate{cmd: sim.Buy(i, k), kind: sim.PurchaseBusiness, idx: i, qty: k,
				cost: sim.BusinessCost(s, b, i, k), gain: gain})
		}
		if owned > 0 && !mgr[i] {
			mm := mgr
			mm[i] = true
			gain := bt.effRate(&base, &mm, &d) - r0
			out = append(out, candidate{cmd: sim.HireManager(i), kind: sim.PurchaseManager, idx: i,
				cost: b.ManagerCost(i), gain: gain})
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
			// A cost cut has no direct $/s gain. The bot spends roughly all
			// income on purchases, so it is worth (1 - value) of the income.
			gain = r0 * (1 - b.Upgrades[ui].Value)
		}
		out = append(out, candidate{cmd: sim.BuyUpgrade(ui), kind: sim.PurchaseUpgrade, idx: ui,
			cost: b.Upgrades[ui].Cost, gain: gain})
	}
	return out, r0
}

// score is the time until the purchase has paid for itself: time to afford
// it at the current rate plus its payback time. Lower is better.
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

// decide makes purchases for this decision point.
func (bt *bot) decide() {
	var buf [64]candidate
	for range 50 { // several purchases per decision when cash allows
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
				// While saving for the best option, take an affordable one
				// that pays for itself before the best becomes affordable:
				// it never delays the best purchase.
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
		bt.evs = sim.Step(&bt.s, bt.b, 0, []sim.Command{pick.cmd}, bt.evs[:0])
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
			}
		}
		bt.observe(bt.evs, bt.s.Clock)
		if !ok {
			return
		}
		bt.res.Purchases = append(bt.res.Purchases, Purchase{T: bt.s.Clock, Kind: pick.kind, Idx: pick.idx, Qty: pick.qty, Cost: pick.cost})
	}
}
