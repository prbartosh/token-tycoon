package sim

import (
	"math"
	"slices"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

func RegionEntered(s *State, b *balance.Balance, i int) bool {
	if i < 0 || i >= len(b.Regions) {
		return false
	}
	r := &b.Regions[i]
	return r.Entry == balance.EntryNone || slices.Contains(s.Market.UnlockedRegions, r.ID)
}

func RegionCompliant(s *State, b *balance.Balance, i int) bool {
	if !RegionEntered(s, b, i) {
		return false
	}
	r := &b.Regions[i]
	return !(r.EnterpriseCostMin > 0) || slices.Contains(s.Market.Compliance, r.ID)
}

func regionFactors(s *State, b *balance.Balance) (mult, ent float64) {
	if !s.IsUnlocked(UnlockRegions) || len(b.Regions) == 0 {
		return 1, 1
	}
	home, entered, comp := 0.0, 0.0, 0.0
	for i := range b.Regions {
		r := &b.Regions[i]
		if r.Entry == balance.EntryNone {
			home += r.MarketShare
		}
		if RegionEntered(s, b, i) {
			entered += r.MarketShare
			if RegionCompliant(s, b, i) {
				comp += r.MarketShare
			}
		}
	}
	if !(home > 0) || !(entered > 0) {
		return 1, 1
	}
	return entered / home, comp / entered
}

func complianceMult(s *State) float64 {
	if x := s.ResMods.ComplianceCost; x > 0 {
		return x
	}
	return 1
}

func EntryCost(s *State, b *balance.Balance, i int) float64 {
	if i < 0 || i >= len(b.Regions) {
		return math.Inf(1)
	}
	r := &b.Regions[i]
	c := r.EntryCostMinutes * 60 * pricingIncome(s)
	if r.Entry == balance.EntryCompliance {
		c *= complianceMult(s)
	}
	return c
}

func ComplianceCost(s *State, b *balance.Balance, i int) float64 {
	if i < 0 || i >= len(b.Regions) {
		return math.Inf(1)
	}
	return b.Regions[i].EnterpriseCostMin * 60 * pricingIncome(s) * complianceMult(s)
}

func RegionGain(s *State, b *balance.Balance, i int, compliance bool) float64 {
	if i < 0 || i >= len(b.Regions) || !s.IsUnlocked(UnlockRegions) {
		return 0
	}
	mv := &s.Derived.Market
	if !(mv.Price > 0) {
		return 0
	}
	saved := s.Market
	r := &b.Regions[i]
	if compliance {
		s.Market.Compliance = append(s.Market.Compliance[:len(s.Market.Compliance):len(s.Market.Compliance)], r.ID)
	} else {
		s.Market.UnlockedRegions = append(s.Market.UnlockedRegions[:len(s.Market.UnlockedRegions):len(s.Market.UnlockedRegions)], r.ID)
	}
	mult, ent := regionFactors(s, b)
	s.Market = saved
	return marketPrice(mv, mult, ent)/mv.Price - 1
}

func enterRegion(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockRegions) {
		return reject(out, c, RejectLocked, b.Market.RegionsUnlockLifetime)
	}
	i := c.Index
	if i < 0 || i >= len(b.Regions) {
		return reject(out, c, RejectInvalid, 0)
	}
	if RegionEntered(s, b, i) {
		return reject(out, c, RejectAlreadyOwned, 0)
	}
	cost := EntryCost(s, b, i)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Market.UnlockedRegions = append(s.Market.UnlockedRegions, b.Regions[i].ID)
	s.Stats.Purchases++
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvRegion, Business: -1, Index: i, Amount: cost, Value: s.Derived.Market.RegionMult})
}

func buyCompliance(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockRegions) {
		return reject(out, c, RejectLocked, b.Market.RegionsUnlockLifetime)
	}
	i := c.Index
	if i < 0 || i >= len(b.Regions) {
		return reject(out, c, RejectInvalid, 0)
	}
	if !RegionEntered(s, b, i) {
		return reject(out, c, RejectNeedRegion, 0)
	}
	if RegionCompliant(s, b, i) {
		return reject(out, c, RejectAlreadyOwned, 0)
	}
	cost := ComplianceCost(s, b, i)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Market.Compliance = append(s.Market.Compliance, b.Regions[i].ID)
	s.Stats.Purchases++
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvCompliance, Business: -1, Index: i, Amount: cost})
}

func RegionsEntered(s *State, b *balance.Balance) int {
	n := 0
	for i := range b.Regions {
		if RegionEntered(s, b, i) {
			n++
		}
	}
	return n
}

func RegionsCompliant(s *State, b *balance.Balance) int {
	n := 0
	for i := range b.Regions {
		if RegionCompliant(s, b, i) {
			n++
		}
	}
	return n
}

func keepKnown(ids []string, b *balance.Balance) []string {
	if ids == nil {
		ids = make([]string, 0, len(b.Regions))
	}
	keep := ids[:0]
	for _, id := range ids {
		if _, ok := b.RegionIndex(id); ok && !slices.Contains(keep, id) {
			keep = append(keep, id)
		}
	}
	return keep
}
