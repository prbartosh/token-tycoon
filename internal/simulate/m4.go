package simulate

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/sim"
)

const (
	m4EveryS       = 30
	salaryCap      = 0.06
	hireBudget     = 0.5
	contractMargin = 1.25
	declineBelow   = 0.5
	roundMinS      = 4 * 3600
	roundStaleS    = 2 * 86400
	growthWindowS  = 600
)

var botRules = [...]sim.Rule{
	{If: [sim.MaxConds]sim.Cond{{Metric: "gpu_util", Op: sim.OpGreater, Value: 0.95}}, Then: sim.Action{Kind: "buy_rack", Qty: sim.Qty10}, Enabled: true},
	{If: [sim.MaxConds]sim.Cond{{Metric: "energy_index", Op: sim.OpLess, Value: 35}, {Metric: "energy_util", Op: sim.OpGreater, Value: 0.5}}, Then: sim.Action{Kind: "buy_ppa"}, Enabled: true},
	{If: [sim.MaxConds]sim.Cond{{Metric: "training_idle", Op: sim.OpGreater, Value: 0.5}}, Then: sim.Action{Kind: "train"}, Enabled: true},
	{If: [sim.MaxConds]sim.Cond{{Metric: "money_x_best", Op: sim.OpGreater, Value: 10}}, Then: sim.Action{Kind: "buy_best", Qty: sim.Qty10}, Enabled: true},
	{If: [sim.MaxConds]sim.Cond{{Metric: "money_minutes", Op: sim.OpGreater, Value: 60}}, Then: sim.Action{Kind: "buy_license"}, Enabled: true},
}

func (bt *bot) maybeFunding() bool {
	s := &bt.s
	pv := sim.Funding(s, bt.b)
	if !pv.Allowed || len(s.Contracts) > 0 {
		return false
	}
	doubled := pv.Gain >= math.Max(5, s.Prestige.CapitalLifetime)
	if pv.RoundS < roundMinS && s.Prestige.ResetsL2 > 0 || !(doubled || pv.RoundS >= roundStaleS) {
		return false
	}
	bt.exec(sim.FundingRound())
	for _, e := range bt.evs {
		if e.Kind == sim.EvFunding {
			bt.res.Rounds = append(bt.res.Rounds, RoundAt{T: s.Clock, Capital: e.Amount, Valuation: e.Value})
			bt.idle = [sim.N]float64{}
			return true
		}
	}
	return false
}

func (bt *bot) maintainM4() {
	s := &bt.s
	if s.Clock < bt.nextM4 {
		return
	}
	bt.nextM4 = s.Clock + m4EveryS
	bt.maintainTalent()
	bt.maintainContracts()
	bt.maintainResearch()
	bt.maintainRules()
}

func (bt *bot) maintainTalent() {
	s := &bt.s
	b := bt.b
	if !s.IsUnlocked(sim.UnlockTalent) || len(s.Training.Queue) == 0 && s.Research.Active < 0 {
		return
	}
	for range b.Talent.MaxStaff {
		if len(s.Talent) >= b.Talent.MaxStaff {
			return
		}
		pick := -1
		for c := len(b.TalentClasses) - 1; c >= 0; c-- {
			cl := &b.TalentClasses[c]
			if s.Derived.Talent.Salary+cl.SalaryShare <= salaryCap && sim.HireCost(s, b, c) <= s.Money*hireBudget {
				pick = c
				break
			}
		}
		if pick < 0 {
			return
		}
		bt.exec(sim.Hire(pick))
	}
}

func (bt *bot) maintainContracts() {
	s := &bt.s
	b := bt.b
	if !s.IsUnlocked(sim.UnlockContracts) {
		return
	}
	g := bt.measuredGrowth()
	pv := sim.Preview(s, b)
	soon := pv.Allowed && pv.BonusAfter >= 2*pv.BonusNow
	bench := sim.BestBenchmark(s, b)
	for i := len(s.Offers) - 1; i >= 0; i-- {
		o := s.Offers[i]
		exp := s.Derived.TokensPerSec * o.DurationS * (1 + g*o.DurationS/2)
		ok := bench >= o.MinBenchmark
		switch {
		case ok && !soon && exp >= o.Volume*contractMargin && len(s.Contracts) < sim.ContractSlots(s, b) && s.Clock+o.DurationS < bt.activeEnd:
			bt.exec(sim.AcceptContract(i))
		case !ok || exp < o.Volume*declineBelow:
			bt.exec(sim.DeclineContract(i))
		}
	}
}

func (bt *bot) measuredGrowth() float64 {
	now, inc := bt.s.Clock, bt.s.Derived.TokensPerSec
	smp := bt.res.Samples
	for i := len(smp) - 1; i >= 0; i-- {
		if dt := now - smp[i].T; dt >= growthWindowS {
			if smp[i].TokensPerS > 0 && inc > smp[i].TokensPerS && smp[i].Prestiges == bt.s.Prestige.ResetsL1 {
				return math.Log(inc/smp[i].TokensPerS) / dt
			}
			return 0
		}
	}
	return 0
}

func (bt *bot) maintainResearch() {
	s := &bt.s
	b := bt.b
	if !s.IsUnlocked(sim.UnlockResearch) || s.Research.Active >= 0 {
		return
	}
	pick, cost := -1, math.Inf(1)
	for i := range b.ResearchNodes {
		n := &b.ResearchNodes[i]
		if sim.NodeState(s, b, i) == sim.NodeAvailable && n.Cost <= s.Prestige.Capital && n.Cost < cost {
			pick, cost = i, n.Cost
		}
	}
	if pick >= 0 {
		bt.exec(sim.Research(pick))
	}
}

func (bt *bot) maintainRules() {
	s := &bt.s
	if !s.IsUnlocked(sim.UnlockRules) {
		return
	}
	for n := len(s.Rules); n < min(len(botRules), sim.RuleSlots(s, bt.b)); n = len(s.Rules) {
		bt.exec(sim.AddRule(botRules[n]))
		if len(s.Rules) == n {
			return
		}
	}
}

func (bt *bot) poachReaction(e sim.Event) {
	if e.Index < 0 || e.Index >= len(bt.s.Talent) || bt.s.Talent[e.Index].Class == bt.b.TalentClasses[0].ID {
		return
	}
	if bt.cfg.Strategy == Passive && bt.session == nil {
		return
	}
	p, lo, hi := bt.reactProfile()
	if bt.rnd.Float64() < p {
		bt.reacts = append(bt.reacts, pendingReact{at: bt.s.Clock + lo + (hi-lo)*bt.rnd.Float64(), counter: true})
	}
}
