package simulate

import (
	"token-tycoon/internal/sim"
)

const agiHorizonS = 2 * 3600

func (bt *bot) incidentReaction(t float64) {
	if bt.cfg.Strategy == Passive && bt.session == nil {
		return
	}
	p, lo, hi := bt.reactProfile()
	if bt.rnd.Float64() < p {
		bt.reacts = append(bt.reacts, pendingReact{at: t + lo + (hi-lo)*bt.rnd.Float64(), key: bt.b.Safety.Key, incident: true})
	}
}

func (bt *bot) regionCandidates(out []candidate, r0 float64) []candidate {
	s := &bt.s
	b := bt.b
	for i := range b.Regions {
		if !sim.RegionEntered(s, b, i) {
			c := candidate{idx: i, cost: sim.EntryCost(s, b, i), gain: r0 * sim.RegionGain(s, b, i, false)}
			c.cmds, c.n = one(sim.EnterRegion(i))
			out = append(out, c)
			continue
		}
		if !sim.RegionCompliant(s, b, i) {
			c := candidate{idx: i, cost: sim.ComplianceCost(s, b, i), gain: r0 * sim.RegionGain(s, b, i, true)}
			c.cmds, c.n = one(sim.BuyCompliance(i))
			out = append(out, c)
		}
	}
	return out
}

func (bt *bot) maintainAGI() bool {
	s := &bt.s
	b := bt.b
	if !s.IsUnlocked(sim.UnlockAGI) || sim.AGIEnded(s, b) || s.AGI.TrainEndsS > 0 || !(s.Money > 0) {
		return false
	}
	need := b.AGIStages[s.AGI.Stage].Cost - s.AGI.Invested
	if need > s.Money && need > s.Derived.BaseIncome*agiHorizonS {
		return false
	}
	bt.exec(sim.AGIInvest())
	return true
}
