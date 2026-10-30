package sim

import (
	"strconv"

	"token-tycoon/internal/balance"
)

// Unlock identifies a game system or business that opens at a revenue
// threshold (DECISIONS 8).
type Unlock uint8

// Unlock ids. Businesses occupy the first N values (tier 1 = 0).
const (
	UnlockBusiness0 Unlock = 0
	UnlockInfra     Unlock = N + iota - 1
	UnlockPrestige1
	UnlockEvents
	UnlockCards
	UnlockMarket
	UnlockSafety
	UnlockTraining
	UnlockData
	UnlockTalent
	UnlockContracts
	UnlockInvestors
	UnlockLayer2
	UnlockResearch
	UnlockRules
	UnlockRegions
	UnlockOrbital
	UnlockAGI
	NumUnlocks
)

var systemNames = [...]string{
	"infra", "prestige1", "events", "cards", "market", "safety", "training", "data", "talent",
	"contracts", "investors", "layer2", "research", "rules", "regions", "orbital", "agi",
}

// Name returns the stable save name of an unlock: "business_1".."business_10"
// for businesses (by tier), else the system name.
func (u Unlock) Name() string {
	if u < UnlockInfra {
		return "business_" + strconv.Itoa(int(u)+1)
	}
	return systemNames[u-UnlockInfra]
}

// IsBusiness reports whether u unlocks a business; Business returns its index.
func (u Unlock) IsBusiness() bool { return u < UnlockInfra }

// Business returns the business index of a business unlock.
func (u Unlock) Business() int { return int(u) }

// ParseUnlock maps a save name back to an Unlock.
func ParseUnlock(name string) (Unlock, bool) {
	for u := Unlock(0); u < NumUnlocks; u++ {
		if u.Name() == name {
			return u, true
		}
	}
	return 0, false
}

// UnlockThreshold returns the revenue needed for u. For UnlockPrestige1 it is
// run revenue (and infrastructure must be unlocked first); for everything
// else it is lifetime total revenue.
func UnlockThreshold(u Unlock, b *balance.Balance) float64 {
	if u < UnlockInfra {
		return b.Businesses[u].UnlockLifetime
	}
	switch u {
	case UnlockInfra:
		return b.Infra.UnlockLifetime
	case UnlockPrestige1:
		return b.Prestige.Layer1.UnlockRunRevenue
	case UnlockEvents:
		return b.Events.UnlockLifetime
	case UnlockCards:
		return b.Cards.UnlockLifetime
	case UnlockMarket:
		return b.Market.UnlockLifetime
	case UnlockSafety:
		return b.Safety.UnlockLifetime
	case UnlockTraining:
		return b.Training.UnlockLifetime
	case UnlockData:
		return b.Data.UnlockLifetime
	case UnlockTalent:
		return b.Talent.UnlockLifetime
	case UnlockContracts:
		return b.Contracts.UnlockLifetime
	case UnlockInvestors:
		return b.Investors.UnlockLifetime
	case UnlockLayer2:
		return b.Prestige.Layer2.UnlockLifetime
	case UnlockResearch:
		return b.Research.UnlockLifetime
	case UnlockRules:
		return b.Rules.UnlockLifetime
	case UnlockRegions:
		return b.Market.RegionsUnlockLifetime
	case UnlockOrbital:
		return b.Energy.OrbitalUnlockLifetime
	case UnlockAGI:
		return b.AGI.UnlockLifetime
	}
	return 0
}

func unlockReached(s *State, b *balance.Balance, u Unlock) bool {
	t := UnlockThreshold(u, b)
	if u == UnlockPrestige1 {
		return s.Unlocks.Has(int(UnlockInfra)) && s.Lifetime.Run >= t
	}
	return s.Lifetime.Total >= t
}

// checkUnlocks sets newly reached unlocks and appends EvUnlock events (out may
// be nil to unlock silently). Unlocks are never revoked.
func checkUnlocks(s *State, b *balance.Balance, out *[]Event) {
	for u := Unlock(0); u < NumUnlocks; u++ {
		if s.Unlocks.Has(int(u)) || !unlockReached(s, b, u) {
			continue
		}
		s.Unlocks.Set(int(u))
		if out != nil {
			*out = append(*out, Event{Kind: EvUnlock, Business: -1, Index: int(u)})
		}
	}
}

// IsUnlocked reports whether u is unlocked.
func (s *State) IsUnlocked(u Unlock) bool { return s.Unlocks.Has(int(u)) }
