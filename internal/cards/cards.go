// Package cards turns a research-card collection into effect multipliers and
// draws new cards. It is pure: no state beyond what the caller passes in.
package cards

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/rng"
)

const NumRarities = len(balance.Rarities)

type Mods struct {
	ProfitAll      float64
	ProfitBiz      [balance.MaxBusinesses]float64
	CycleMult      float64
	CostMult       float64
	ComputeMult    float64
	CritChance     float64
	CritMult       float64
	CardChance     float64
	ReactionReward float64
	EnergyCost     float64
	RackCost       float64
	Offline        float64
	TrainingSpeed  float64
	RevealRatio    bool
	Unique         int
	Drops          int
	SetsComplete   [NumRarities]bool
}

func Neutral() Mods {
	m := Mods{ProfitAll: 1, CycleMult: 1, CostMult: 1, ComputeMult: 1, EnergyCost: 1, RackCost: 1}
	for i := range m.ProfitBiz {
		m.ProfitBiz[i] = 1
	}
	return m
}

func LevelBonus(b *balance.Balance, value float64, level int) float64 {
	if level <= 0 {
		return 0
	}
	level = min(level, b.Cards.MaxLevel)
	return value * math.Pow(b.Cards.LevelBonusMult, float64(level-1))
}

func Compute(b *balance.Balance, owned map[string]int, m *Mods) {
	var profitAll, cycle, cost, compute, energy, rack float64
	var biz [balance.MaxBusinesses]float64
	*m = Mods{}
	var have, total [NumRarities]int
	for i := range b.Card {
		c := &b.Card[i]
		r := b.CardRarity(i)
		total[r]++
		lv := owned[c.ID]
		if lv <= 0 {
			continue
		}
		have[r]++
		m.Unique++
		m.Drops += lv
		v := LevelBonus(b, c.Value, lv)
		switch c.Effect {
		case balance.CardProfitAll:
			profitAll += v
		case balance.CardProfitBusiness:
			if t := b.CardTarget(i); t >= 0 {
				biz[t] += v
			}
		case balance.CardCycleAll:
			cycle += v
		case balance.CardCostAll:
			cost += v
		case balance.CardComputeAll:
			compute += v
		case balance.CardEnergyCost:
			energy += v
		case balance.CardRackCost:
			rack += v
		case balance.CardCritChance:
			m.CritChance += v
		case balance.CardCritMult:
			m.CritMult += v
		case balance.CardCardChance:
			m.CardChance += v
		case balance.CardReactionReward:
			m.ReactionReward += v
		case balance.CardOffline:
			m.Offline += v
		case balance.CardTrainingSpeed:
			m.TrainingSpeed += v
		case balance.CardRevealRatio:
			m.RevealRatio = true
		}
	}
	for r := range NumRarities {
		if total[r] > 0 && have[r] == total[r] {
			m.SetsComplete[r] = true
			profitAll += b.Cards.SetBonus[balance.Rarities[r]]
		}
	}
	m.ProfitAll = 1 + profitAll
	for i := range biz {
		m.ProfitBiz[i] = 1 + biz[i]
	}
	m.CycleMult = 1 / (1 + cycle)
	m.CostMult = reduction(cost)
	m.ComputeMult = reduction(compute)
	m.EnergyCost = reduction(energy)
	m.RackCost = reduction(rack)
}

func reduction(sum float64) float64 { return math.Max(0.1, 1+sum) }

func Draw(b *balance.Balance, r *rng.Stream, rare float64) int {
	if !(rare >= 1) {
		rare = 1
	}
	weight := func(i int, name string) float64 {
		w := b.Cards.RarityWeights[name]
		if i >= 2 {
			w *= rare
		}
		return w
	}
	total := 0.0
	for i, name := range balance.Rarities {
		if len(b.CardsOfRarity(i)) > 0 {
			total += weight(i, name)
		}
	}
	if !(total > 0) {
		return -1
	}
	x := r.Float64() * total
	pick := -1
	for i, name := range balance.Rarities {
		if len(b.CardsOfRarity(i)) == 0 {
			continue
		}
		pick = i
		x -= weight(i, name)
		if x < 0 {
			break
		}
	}
	ids := b.CardsOfRarity(pick)
	return ids[r.IntN(len(ids))]
}
