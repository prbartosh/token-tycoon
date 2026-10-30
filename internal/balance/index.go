package balance

const MaxGPUGens = 8

type TargetMask uint16

const AllBusinesses TargetMask = 1<<MaxBusinesses - 1

func (m TargetMask) Has(i int) bool { return m&(1<<uint(i)) != 0 }

type index struct {
	business    map[string]int
	upgrade     map[string]int
	upgradeMask []TargetMask
	freeOwned   [MaxBusinesses]int
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
