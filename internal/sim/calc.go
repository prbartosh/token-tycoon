package sim

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/cards"
	"token-tycoon/internal/econ"
	"token-tycoon/internal/infra"
)

type Derived struct {
	CycleS       [N]float64
	PerCycle     [N]float64
	PerSec       [N]float64
	ProfitMult   [N]float64
	CostMult     [N]float64
	ComputeUnit  [N]float64
	EventMult    [N]float64
	SpeedMS      [N]int
	ProfitMS     [N]int
	Continuous   [N]bool
	GlobalMS     int
	GlobalMult   float64
	MinOwned     int
	CritChance   float64
	CritMult     float64
	CritEV       float64
	CardChance   float64
	Price        float64
	ComboMult    float64
	IncomePerSec float64
	BaseIncome   float64

	ComputeNeed  float64
	ComputeCap   float64
	RackMW       float64
	EnergyNeed   float64
	EnergyAvail  float64
	EnergyUsed   float64
	GridUsed     float64
	PPAUsed      float64
	FreeUsed     float64
	EnergyIndex  float64
	Peak         bool
	Throttle     float64
	CloudShare   float64
	EnergyShare  float64
	NetMult      float64
	RackCostMult float64
	ComputeTotal float64

	SalaryShare  float64
	TokensPerSec float64
	AGIMult      float64

	Market MarketView
	Train  TrainView
	Talent TalentView
}

type Hypo struct {
	Owned        [N]int
	Upgrades     Bitset
	Params       float64
	GlobalEarned int
	Unlocked     balance.TargetMask
	Cards        cards.Mods
	Infra        Infra
	InfraOn      bool
	Clock        float64
	Seed         uint64
	Event        int
	MarketOn     bool
	Price        float64
	TrainShare   float64
	SynthShare   float64
	Extra        ExtraMods
}

type ExtraMods struct {
	ComputeMult float64
	EnergyCost  float64
	CritChance  float64
	CardChance  float64
	Salary      float64
	ProfitMult  float64
}

func HypoOf(s *State) Hypo {
	h := Hypo{
		Upgrades: s.Upgrades, Params: s.Prestige.Params, GlobalEarned: s.GlobalMilestones,
		Cards: s.CardMods, Infra: s.Infra, InfraOn: s.IsUnlocked(UnlockInfra),
		Clock: s.Clock, Seed: s.Seed, Event: -1,
		MarketOn: s.Derived.Market.On, Price: s.Derived.Market.Price,
		TrainShare: s.Derived.Train.TrainShare, SynthShare: s.Derived.Train.SynthShare,
		Extra: ExtraMods{
			ComputeMult: s.ResMods.ComputeMult * s.Derived.Talent.ComputeMult,
			EnergyCost:  s.ResMods.EnergyCost,
			CritChance:  s.ResMods.CritChance,
			CardChance:  s.ResMods.CardChance,
			Salary:      s.Derived.Talent.Salary,
			ProfitMult:  s.Derived.AGIMult,
		},
	}
	for i := range s.Businesses {
		h.Owned[i] = s.Businesses[i].Owned
		if s.IsUnlocked(UnlockBusiness0 + Unlock(i)) {
			h.Unlocked |= 1 << uint(i)
		}
	}
	return h
}

func Evaluate(b *balance.Balance, h *Hypo, d *Derived) {
	m := &b.Milestones
	cm := &h.Cards
	var cycleMult, profitMult, costMult, computeMult [N]float64
	for i := range N {
		cycleMult[i], profitMult[i], costMult[i], computeMult[i] = cm.CycleMult, cm.ProfitBiz[i], cm.CostMult, cm.ComputeMult
		if x := h.Extra.ComputeMult; x > 0 {
			computeMult[i] *= x
		}
	}
	for ui := range b.Upgrades {
		if !h.Upgrades.Has(ui) {
			continue
		}
		u := &b.Upgrades[ui]
		mask := b.UpgradeMask(ui)
		for i := range N {
			if !mask.Has(i) {
				continue
			}
			switch u.Effect {
			case balance.EffectProfit:
				profitMult[i] *= u.Value
			case balance.EffectCycle:
				cycleMult[i] *= u.Value
			case balance.EffectCost:
				costMult[i] *= u.Value
			case balance.EffectCompute:
				computeMult[i] *= u.Value
			}
			if u.ComputeMult > 0 {
				computeMult[i] *= u.ComputeMult
			}
		}
	}
	minOwned := 0
	if h.Unlocked != 0 {
		minOwned = math.MaxInt
		for i := range N {
			if h.Unlocked.Has(i) {
				minOwned = min(minOwned, h.Owned[i])
			}
		}
	}
	d.MinOwned = minOwned
	d.GlobalMS = max(h.GlobalEarned, econ.GlobalMilestones(minOwned, m.GlobalThresholds, m.GlobalEvery))
	d.GlobalMult = econ.MultPow(m.GlobalMult, d.GlobalMS) *
		econ.Layer1Bonus(h.Params, b.Prestige.Layer1.BonusPerParam) * cm.ProfitAll
	if x := h.Extra.ProfitMult; x > 0 {
		d.GlobalMult *= x
	}
	d.CritChance = math.Min(0.5, b.Crit.Chance+cm.CritChance+h.Extra.CritChance)
	d.CritMult = b.Crit.Mult + cm.CritMult
	d.CritEV = econ.CritEV(d.CritChance, d.CritMult)
	d.Price = b.Market.PriceBase
	if h.MarketOn {
		d.Price = h.Price
	}

	cardEv, rackEv := 1.0, 1.0
	for i := range N {
		d.EventMult[i] = 1
	}
	if h.Event >= 0 && h.Event < len(b.Event) {
		ev := &b.Event[h.Event]
		switch ev.Effect {
		case balance.EventProfit:
			for i := range N {
				d.EventMult[i] = ev.Value
			}
		case balance.EventProfitTiers:
			for _, t := range ev.Tiers {
				if t >= 1 && t <= N {
					d.EventMult[t-1] = ev.Value
				}
			}
		case balance.EventCardChance:
			cardEv = ev.Value
		case balance.EventRackCost:
			rackEv = ev.Value
		}
	}
	d.CardChance = math.Min(1, b.Cards.DropChancePerCycle*(1+cm.CardChance+h.Extra.CardChance)*cardEv)
	d.RackCostMult = cm.RackCost * rackEv

	need := 0.0
	for i := range N {
		bs := &b.Businesses[i]
		owned := h.Owned[i]
		d.SpeedMS[i] = econ.SpeedMilestones(owned, m.SpeedThresholds)
		d.ProfitMS[i] = econ.ProfitMilestones(owned, m.ProfitStart, m.ProfitEvery)
		d.CycleS[i] = bs.CycleS * cycleMult[i] * econ.MultPow(m.SpeedMult, d.SpeedMS[i])
		d.ProfitMult[i] = profitMult[i] * econ.MultPow(m.ProfitMult, d.ProfitMS[i])
		d.CostMult[i] = costMult[i]
		d.ComputeUnit[i] = bs.Compute * computeMult[i]
		need += d.ComputeUnit[i] * float64(owned)
		d.PerCycle[i] = bs.Tokens * float64(owned) * d.ProfitMult[i] * d.GlobalMult * d.Price * d.EventMult[i]
		d.Continuous[i] = d.CycleS[i] < b.BusinessDefaults.ContinuousBarBelowS
	}
	evaluateInfra(b, h, d, need)
	d.SalaryShare = h.Extra.Salary
	d.NetMult = math.Max(0, d.NetMult-d.SalaryShare)
	net := d.Throttle * d.NetMult
	for i := range N {
		d.PerCycle[i] *= net
		d.PerSec[i] = d.PerCycle[i] / d.CycleS[i] * d.CritEV
	}
}

func evaluateInfra(b *balance.Balance, h *Hypo, d *Derived, need float64) {
	e := &b.Energy
	d.ComputeNeed = need
	d.EnergyIndex = infra.EnergyIndex(h.Seed, h.Clock, e)
	d.Peak = infra.IsPeak(h.Clock, e)
	d.ComputeCap, d.RackMW, d.EnergyNeed, d.EnergyAvail, d.EnergyUsed, d.ComputeTotal = 0, 0, 0, 0, 0, 0
	tv := &d.Train
	tv.TrainShare, tv.SynthShare, tv.EnergyFrac = 0, 0, 0
	d.GridUsed, d.PPAUsed, d.FreeUsed = 0, 0, 0
	d.Throttle, d.CloudShare, d.EnergyShare, d.NetMult = 1, 0, 0, 1
	if !h.InfraOn {
		return
	}
	in := &h.Infra
	for g := range b.GPUGens {
		n := float64(in.Racks[g])
		d.ComputeCap += n * b.GPUGens[g].Compute
		d.RackMW += n * b.GPUGens[g].MW
	}
	orbital := 1 + float64(in.Energy.Orbital)*e.OrbitalComputeBonus
	d.ComputeTotal = d.ComputeCap * orbital
	reserve := h.TrainShare + h.SynthShare
	d.ComputeCap = d.ComputeTotal * (1 - reserve)
	util := 1.0
	if d.ComputeCap > 0 {
		util = math.Min(1, need/d.ComputeCap)
	}
	d.EnergyNeed = d.RackMW * ((1-reserve)*util + reserve)
	sup := infra.Supply{
		FreeMW: float64(in.Energy.SMR)*e.SMRMW + float64(in.Energy.Orbital)*e.OrbitalMW,
		GridMW: in.Energy.GridMW,
		Peak:   d.Peak,
	}
	if in.Energy.PPA.ExpiresS > h.Clock {
		sup.PPAMW = in.Energy.PPA.MW
	}
	l := infra.Dispatch(d.EnergyNeed, sup, e.GridPeakAvailability)
	d.EnergyAvail, d.EnergyUsed = l.EnergyAvail, l.EnergyUsed
	d.GridUsed, d.PPAUsed, d.FreeUsed = l.GridUsed, l.PPAUsed, l.FreeUsed
	d.Throttle = infra.Throttle(need, d.ComputeCap, d.EnergyNeed, l.EnergyAvail)
	energyFrac := 1.0
	if d.EnergyNeed > 0 {
		energyFrac = math.Min(1, l.EnergyAvail/d.EnergyNeed)
	}
	tv.TrainShare, tv.SynthShare, tv.EnergyFrac = h.TrainShare, h.SynthShare, energyFrac
	if in.CloudEnabled {
		d.Throttle = 1
		d.CloudShare = b.Infra.CloudRevenueShare
	}
	d.EnergyShare = infra.EnergyCostShare(l, d.EnergyIndex, in.Energy.PPA.Index, e.GridCostCoef) * h.Cards.EnergyCost
	if x := h.Extra.EnergyCost; x > 0 {
		d.EnergyShare *= x
	}
	d.NetMult = math.Max(0, 1-d.CloudShare-d.EnergyShare)
}

func Recompute(s *State, b *balance.Balance) {
	computeResearch(s, b)
	computeTalent(s, b)
	s.Derived.AGIMult = AGIProfitMult(s, b)
	ev := timedEvent(s, b)
	evaluateMarket(s, b, ev)
	s.Derived.Train.TrainShare, s.Derived.Train.SynthShare = computeReserve(s, b)
	h := HypoOf(s)
	h.Event = ev
	Evaluate(b, &h, &s.Derived)
	d := &s.Derived
	s.GlobalMilestones = d.GlobalMS
	d.ComboMult = s.Combo.Mult
	inc, steady := 0.0, 0.0
	for i := range N {
		if s.Businesses[i].Manager {
			inc += d.PerSec[i]
			steady += d.PerSec[i] / d.EventMult[i]
		}
	}
	d.BaseIncome = inc
	d.TokensPerSec = 0
	if den := d.Price * d.NetMult; den > 0 {
		d.TokensPerSec = steady / den
	}
	d.IncomePerSec = inc * max(1, s.Combo.Mult)
	s.Stats.BestIncome = math.Max(s.Stats.BestIncome, inc)
	finishTrainView(s, b)
}

func BusinessCost(s *State, b *balance.Balance, i, k int) float64 {
	bs := &b.Businesses[i]
	return econ.BulkCost(bs.Cost*s.Derived.CostMult[i], bs.R, s.Businesses[i].Owned, k)
}

func MaxBuy(s *State, b *balance.Balance, i int) int {
	bs := &b.Businesses[i]
	return econ.MaxAffordable(s.Money, bs.Cost*s.Derived.CostMult[i], bs.R, s.Businesses[i].Owned)
}

func ResolveQty(s *State, b *balance.Balance, i, qty int) (int, float64) {
	if qty == QtyMax {
		k := MaxBuy(s, b, i)
		if k == 0 {
			k = 1
		}
		return k, BusinessCost(s, b, i, k)
	}
	if qty < 1 {
		qty = 1
	}
	return qty, BusinessCost(s, b, i, qty)
}

func UnlockETA(s *State, b *balance.Balance, u Unlock) float64 {
	t := UnlockThreshold(u, b)
	have := s.Lifetime.Total
	if u == UnlockPrestige1 {
		have = s.Lifetime.Run
	}
	if have >= t {
		return 0
	}
	if s.Derived.IncomePerSec <= 0 {
		return math.Inf(1)
	}
	return (t - have) / s.Derived.IncomePerSec
}

func MinutesOfIncome(s *State, minutes float64) float64 {
	return minutes * 60 * s.Derived.BaseIncome
}
