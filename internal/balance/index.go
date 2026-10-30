package balance

import (
	"maps"
	"slices"
)

const MaxGPUGens = 8

type TargetMask uint16

const AllBusinesses TargetMask = 1<<MaxBusinesses - 1

func (m TargetMask) Has(i int) bool { return m&(1<<uint(i)) != 0 }

type index struct {
	business    map[string]int
	upgrade     map[string]int
	upgradeMask []TargetMask
	freeOwned   [MaxBusinesses]int
	card        map[string]int
	cardRarity  []int
	cardTarget  []int
	byRarity    [len(Rarities)][]int
	event       map[string]int
	arch        []Arch
}

func (b *Balance) buildIndex() {
	b.idx.business = make(map[string]int, len(b.Businesses))
	for i, bs := range b.Businesses {
		b.idx.business[bs.ID] = i
	}
	b.idx.upgrade = make(map[string]int, len(b.Upgrades))
	b.idx.upgradeMask = make([]TargetMask, len(b.Upgrades))
	for i, u := range b.Upgrades {
		b.idx.upgrade[u.ID] = i
		var m TargetMask
		for _, t := range u.Target {
			if t == "all" {
				m = AllBusinesses
				break
			}
			m |= 1 << uint(b.idx.business[t])
		}
		b.idx.upgradeMask[i] = m
	}
	for id, n := range b.Start.FreeBusinesses {
		b.idx.freeOwned[b.idx.business[id]] = n
	}
	b.idx.card = make(map[string]int, len(b.Card))
	b.idx.cardRarity = make([]int, len(b.Card))
	b.idx.cardTarget = make([]int, len(b.Card))
	for i, c := range b.Card {
		b.idx.card[c.ID] = i
		r := RarityIndex(c.Rarity)
		b.idx.cardRarity[i] = r
		b.idx.byRarity[r] = append(b.idx.byRarity[r], i)
		b.idx.cardTarget[i] = -1
		if c.Effect == CardProfitBusiness {
			b.idx.cardTarget[i] = b.idx.business[c.Target]
		}
	}
	b.idx.event = make(map[string]int, len(b.Event))
	for i, e := range b.Event {
		b.idx.event[e.ID] = i
	}
	b.idx.arch = b.idx.arch[:0]
	for _, id := range slices.Sorted(maps.Keys(b.Training.Architecture)) {
		card := -1
		if c, ok := b.idx.card[b.Training.ArchitectureCard[id]]; ok {
			card = c
		}
		b.idx.arch = append(b.idx.arch, Arch{ID: id, Mult: b.Training.Architecture[id], Card: card})
	}
}

func (b *Balance) Archs() []Arch { return b.idx.arch }

func RarityIndex(r string) int {
	for i, x := range Rarities {
		if x == r {
			return i
		}
	}
	return 0
}

func (b *Balance) CardIndex(id string) (int, bool) {
	i, ok := b.idx.card[id]
	return i, ok
}

func (b *Balance) CardRarity(i int) int { return b.idx.cardRarity[i] }

func (b *Balance) CardTarget(i int) int { return b.idx.cardTarget[i] }

func (b *Balance) CardsOfRarity(r int) []int { return b.idx.byRarity[r] }

func (b *Balance) EventIndex(id string) (int, bool) {
	i, ok := b.idx.event[id]
	return i, ok
}

func (b *Balance) BusinessIndex(id string) (int, bool) {
	i, ok := b.idx.business[id]
	return i, ok
}

func (b *Balance) UpgradeIndex(id string) (int, bool) {
	i, ok := b.idx.upgrade[id]
	return i, ok
}

func (b *Balance) UpgradeMask(i int) TargetMask { return b.idx.upgradeMask[i] }

func (b *Balance) FreeOwned() [MaxBusinesses]int { return b.idx.freeOwned }

func (b *Balance) ManagerCost(i int) float64 {
	bs := b.Businesses[i]
	if bs.ManagerCost > 0 {
		return bs.ManagerCost
	}
	return bs.Cost * b.BusinessDefaults.ManagerMult
}
