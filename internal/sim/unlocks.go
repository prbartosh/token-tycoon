package sim

import (
	"strconv"

	"token-tycoon/internal/balance"
)

type Unlock uint8

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

func (u Unlock) Name() string {
	if u < UnlockInfra {
		return "business_" + strconv.Itoa(int(u)+1)
	}
	return systemNames[u-UnlockInfra]
}

func (u Unlock) IsBusiness() bool { return u < UnlockInfra }

func (u Unlock) Business() int { return int(u) }

func ParseUnlock(name string) (Unlock, bool) {
	for u := Unlock(0); u < NumUnlocks; u++ {
		if u.Name() == name {
			return u, true
		}
	}
	return 0, false
}

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

func checkUnlocks(s *State, b *balance.Balance, out *[]Event) {
	for u := Unlock(0); u < NumUnlocks; u++ {
		if s.Unlocks.Has(int(u)) || !unlockReached(s, b, u) {
			continue
		}
		s.Unlocks.Set(int(u))
		if u == UnlockMarket {
			s.Market.OpenedS = s.Clock
		}
		if out != nil {
			*out = append(*out, Event{Kind: EvUnlock, Business: -1, Index: int(u)})
		}
	}
}

func (s *State) IsUnlocked(u Unlock) bool { return s.Unlocks.Has(int(u)) }
