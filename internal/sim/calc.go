package sim

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
)

type Derived struct {
	CycleS       [N]float64
	PerCycle     [N]float64
	PerSec       [N]float64
	ProfitMult   [N]float64
	CostMult     [N]float64
	SpeedMS      [N]int
	ProfitMS     [N]int
	Continuous   [N]bool
	GlobalMS     int
	GlobalMult   float64
	MinOwned     int
	CritEV       float64
	Price        float64
	ComboMult    float64
	IncomePerSec float64
}

type Hypo struct {
	Owned    [N]int
	Upgrades Bitset
	Params   float64
}

func HypoOf(s *State) Hypo {
	h := Hypo{Upgrades: s.Upgrades, Params: s.Prestige.Params}
	for i := range s.Businesses {
		h.Owned[i] = s.Businesses[i].Owned
	}
	return h
}

func Evaluate(b *balance.Balance, h *Hypo, d *Derived) {
	m := &b.Milestones
	var cycleMult, profitMult, costMult [N]float64
	for i := range N {
		cycleMult[i], profitMult[i], costMult[i] = 1, 1, 1
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
			}
		}
	}
	minOwned := math.MaxInt
	for i := range N {
		minOwned = min(minOwned, h.Owned[i])
	}
	d.MinOwned = minOwned
	d.GlobalMS = econ.GlobalMilestones(minOwned, m.GlobalThresholds, m.GlobalEvery)
	d.GlobalMult = econ.MultPow(m.GlobalMult, d.GlobalMS) *
		econ.Layer1Bonus(h.Params, b.Prestige.Layer1.BonusPerParam)
	d.CritEV = econ.CritEV(b.Crit.Chance, b.Crit.Mult)
	d.Price = b.Market.PriceBase
	for i := range N {
		bs := &b.Businesses[i]
		owned := h.Owned[i]
		d.SpeedMS[i] = econ.SpeedMilestones(owned, m.SpeedThresholds)
		d.ProfitMS[i] = econ.ProfitMilestones(owned, m.ProfitStart, m.ProfitEvery)
		d.CycleS[i] = bs.CycleS * cycleMult[i] * econ.MultPow(m.SpeedMult, d.SpeedMS[i])
		d.ProfitMult[i] = profitMult[i] * econ.MultPow(m.ProfitMult, d.ProfitMS[i])
		d.CostMult[i] = costMult[i]
		d.PerCycle[i] = bs.Tokens * float64(owned) * d.ProfitMult[i] * d.GlobalMult * d.Price
		d.PerSec[i] = d.PerCycle[i] / d.CycleS[i] * d.CritEV
		d.Continuous[i] = d.CycleS[i] < b.BusinessDefaults.ContinuousBarBelowS
	}
}

func Recompute(s *State, b *balance.Balance) {
	h := HypoOf(s)
	Evaluate(b, &h, &s.Derived)
	s.Derived.ComboMult = s.Combo.Mult
	inc := 0.0
	for i := range N {
		if s.Businesses[i].Manager {
			inc += s.Derived.PerSec[i]
		}
	}
	s.Derived.IncomePerSec = inc * max(1, s.Combo.Mult)
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
