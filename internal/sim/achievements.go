package sim

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

type AchievementDef struct {
	ID   string
	Test func(s *State, b *balance.Balance) bool
}

func ownedTotal(s *State) int {
	n := 0
	for i := range s.Businesses {
		n += s.Businesses[i].Owned
	}
	return n
}

func maxOwned(s *State) int {
	n := 0
	for i := range s.Businesses {
		n = max(n, s.Businesses[i].Owned)
	}
	return n
}

func managers(s *State) int {
	n := 0
	for i := range s.Businesses {
		if s.Businesses[i].Manager {
			n++
		}
	}
	return n
}

func racks(s *State) int {
	n := 0
	for _, r := range s.Infra.Racks {
		n += r
	}
	return n
}

func hasStar(s *State, b *balance.Balance) bool {
	for i := range s.Talent {
		if ci, ok := b.TalentClassIndex(s.Talent[i].Class); ok && b.TalentClasses[ci].Trait {
			return true
		}
	}
	return false
}

func hasLegendary(s *State, b *balance.Balance) bool {
	for i := range b.Card {
		if balance.Rarities[b.CardRarity(i)] == "legendary" && s.Cards[b.Card[i].ID] > 0 {
			return true
		}
	}
	return false
}

func anySet(s *State) bool {
	for _, done := range s.CardMods.SetsComplete {
		if done {
			return true
		}
	}
	return false
}

func lifetimeAtLeast(x float64) func(*State, *balance.Balance) bool {
	return func(s *State, _ *balance.Balance) bool { return s.Lifetime.Total >= x }
}

func agiStageAtLeast(n int) func(*State, *balance.Balance) bool {
	return func(s *State, _ *balance.Balance) bool { return s.AGI.Stage >= n }
}

var Achievements = [...]AchievementDef{
	{"first_purchase", func(s *State, _ *balance.Balance) bool { return s.Stats.Purchases >= 1 }},
	{"first_manager", func(s *State, _ *balance.Balance) bool { return managers(s) >= 1 }},
	{"managers_6", func(s *State, _ *balance.Balance) bool { return managers(s) >= 6 }},
	{"managers_all", func(s *State, _ *balance.Balance) bool { return managers(s) >= N }},
	{"biz_100", func(s *State, _ *balance.Balance) bool { return maxOwned(s) >= 100 }},
	{"biz_500", func(s *State, _ *balance.Balance) bool { return maxOwned(s) >= 500 }},
	{"owned_1000", func(s *State, _ *balance.Balance) bool { return ownedTotal(s) >= 1000 }},
	{"global_ms", func(s *State, _ *balance.Balance) bool { return s.GlobalMilestones >= 1 }},
	{"global_ms_5", func(s *State, _ *balance.Balance) bool { return s.GlobalMilestones >= 5 }},
	{"lifetime_1m", lifetimeAtLeast(1e6)},
	{"lifetime_1b", lifetimeAtLeast(1e9)},
	{"lifetime_1t", lifetimeAtLeast(1e12)},
	{"lifetime_1e20", lifetimeAtLeast(1e20)},
	{"lifetime_1e30", lifetimeAtLeast(1e30)},
	{"lifetime_1e50", lifetimeAtLeast(1e50)},
	{"lifetime_1e75", lifetimeAtLeast(1e75)},
	{"lifetime_1e100", lifetimeAtLeast(1e100)},
	{"lifetime_1e150", lifetimeAtLeast(1e150)},
	{"first_crit", func(s *State, _ *balance.Balance) bool { return s.Stats.Crits >= 1 }},
	{"crits_1000", func(s *State, _ *balance.Balance) bool { return s.Stats.Crits >= 1000 }},
	{"combo_max", func(s *State, b *balance.Balance) bool { return s.Stats.BestCombo >= b.Combo.MaxMult }},
	{"boosts_1000", func(s *State, _ *balance.Balance) bool { return s.Stats.BoostPresses >= 1000 }},
	{"first_event", func(s *State, _ *balance.Balance) bool { return s.Stats.EventsCaught >= 1 }},
	{"events_50", func(s *State, _ *balance.Balance) bool { return s.Stats.EventsCaught >= 50 }},
	{"first_card", func(s *State, _ *balance.Balance) bool { return s.Stats.CardsDropped >= 1 }},
	{"card_collector_1", func(s *State, _ *balance.Balance) bool { return s.CardMods.Unique >= 10 }},
	{"card_collector_2", func(s *State, _ *balance.Balance) bool { return s.CardMods.Unique >= 25 }},
	{"card_collector_3", func(s *State, b *balance.Balance) bool { return s.CardMods.Unique >= len(b.Card) }},
	{"legendary_card", hasLegendary},
	{"card_set", func(s *State, _ *balance.Balance) bool { return anySet(s) }},
	{"infra", func(s *State, _ *balance.Balance) bool { return s.IsUnlocked(UnlockInfra) }},
	{"racks_100", func(s *State, _ *balance.Balance) bool { return racks(s) >= 100 }},
	{"first_smr", func(s *State, _ *balance.Balance) bool { return s.Infra.Energy.SMR >= 1 }},
	{"first_orbital", func(s *State, _ *balance.Balance) bool { return s.Infra.Energy.Orbital >= 1 }},
	{"first_prestige", func(s *State, _ *balance.Balance) bool { return s.Prestige.ResetsL1 >= 1 }},
	{"prestige_10", func(s *State, _ *balance.Balance) bool { return s.Prestige.ResetsL1 >= 10 }},
	{"prestige_50", func(s *State, _ *balance.Balance) bool { return s.Prestige.ResetsL1 >= 50 }},
	{"market", func(s *State, _ *balance.Balance) bool { return s.IsUnlocked(UnlockMarket) }},
	{"rank_1", func(s *State, _ *balance.Balance) bool { return s.Derived.Market.On && s.Derived.Market.Rank == 1 }},
	{"share_50", func(s *State, _ *balance.Balance) bool { return s.Derived.Market.Share >= 0.5 }},
	{"hype_3", func(s *State, _ *balance.Balance) bool { return s.Market.Hype >= 2 }},
	{"first_model", func(s *State, _ *balance.Balance) bool { return s.Stats.Models >= 1 }},
	{"bench_60", func(s *State, _ *balance.Balance) bool { return s.Stats.BestBench >= 60 }},
	{"bench_90", func(s *State, _ *balance.Balance) bool { return s.Stats.BestBench >= 90 }},
	{"models_25", func(s *State, _ *balance.Balance) bool { return s.Stats.Models >= 25 }},
	{"first_hire", func(s *State, _ *balance.Balance) bool { return s.Stats.Hired >= 1 }},
	{"star_hire", hasStar},
	{"retained_5", func(s *State, _ *balance.Balance) bool { return s.Stats.Retained >= 5 }},
	{"first_contract", func(s *State, _ *balance.Balance) bool { return s.Stats.ContractsDone >= 1 }},
	{"contracts_25", func(s *State, _ *balance.Balance) bool { return s.Stats.ContractsDone >= 25 }},
	{"reputation_max", func(s *State, b *balance.Balance) bool { return s.Market.Reputation >= b.Contracts.ReputationMax }},
	{"first_round", func(s *State, _ *balance.Balance) bool { return s.Prestige.ResetsL2 >= 1 }},
	{"ipo", func(s *State, b *balance.Balance) bool { return s.Prestige.RoundIndex >= len(b.Prestige.Layer2.Rounds) }},
	{"research_10", func(s *State, _ *balance.Balance) bool { return s.Stats.ResearchDone >= 10 }},
	{"research_all", func(s *State, b *balance.Balance) bool { return s.Research.Done.Count() >= len(b.ResearchNodes) }},
	{"first_rule", func(s *State, _ *balance.Balance) bool { return s.Stats.RulesFired >= 1 }},
	{"rules_1000", func(s *State, _ *balance.Balance) bool { return s.Stats.RulesFired >= 1000 }},
	{"region_new", func(s *State, b *balance.Balance) bool { return len(s.Market.UnlockedRegions) >= 1 }},
	{"regions_all", func(s *State, b *balance.Balance) bool {
		return len(b.Regions) > 0 && s.IsUnlocked(UnlockRegions) && RegionsCompliant(s, b) == len(b.Regions)
	}},
	{"incident_ok", func(s *State, _ *balance.Balance) bool { return s.Stats.IncidentsOK >= 1 }},
	{"incidents_10", func(s *State, _ *balance.Balance) bool { return s.Stats.IncidentsOK >= 10 }},
	{"quarter_beaten", func(s *State, _ *balance.Balance) bool { return s.Stats.QuartersBeaten >= 1 }},
	{"quarters_5", func(s *State, _ *balance.Balance) bool { return s.Stats.QuartersBeaten >= 5 }},
	{"agi_unlocked", func(s *State, _ *balance.Balance) bool { return s.IsUnlocked(UnlockAGI) }},
	{"agi_stage_1", agiStageAtLeast(1)},
	{"agi_stage_3", agiStageAtLeast(3)},
	{"agi_ending", func(s *State, b *balance.Balance) bool { return AGIEnded(s, b) }},
	{"si_level_1", func(s *State, _ *balance.Balance) bool { return s.AGI.SILevel >= 1 }},
	{"streak_7", func(s *State, b *balance.Balance) bool { return s.Streak.Day >= b.Streak.MaxDay }},
	{"play_24h", func(s *State, _ *balance.Balance) bool { return s.Stats.PlayS >= dayS }},
	{"offline_24h", func(s *State, _ *balance.Balance) bool { return s.Stats.OfflineS >= dayS }},
}

func AchievementIndex(id string) int {
	for i := range Achievements {
		if Achievements[i].ID == id {
			return i
		}
	}
	return -1
}

func checkAchievements(s *State, b *balance.Balance, from float64, out []Event) []Event {
	if math.Floor(s.Clock) == math.Floor(from) {
		return out
	}
	for i := range Achievements {
		if s.Achieved.Has(i) || !Achievements[i].Test(s, b) {
			continue
		}
		s.Achieved.Set(i)
		s.Achievements = append(s.Achievements, Achievements[i].ID)
		out = append(out, Event{Kind: EvAchievement, Business: -1, Index: i})
	}
	return out
}

func normalizeAchievements(s *State) {
	s.Achieved = Bitset{}
	keep := make([]string, 0, len(Achievements))
	for _, id := range s.Achievements {
		if i := AchievementIndex(id); i >= 0 && !s.Achieved.Has(i) {
			s.Achieved.Set(i)
			keep = append(keep, id)
		}
	}
	s.Achievements = keep
}

func normalizeM5(s *State, b *balance.Balance) {
	s.Market.UnlockedRegions = keepKnown(s.Market.UnlockedRegions, b)
	s.Market.Compliance = keepKnown(s.Market.Compliance, b)
	if s.Incident.ID != "" {
		if _, ok := b.IncidentIndex(s.Incident.ID); !ok || !(s.Incident.EndsAtS >= 0) {
			s.Incident = Incident{}
		}
	}
	normalizeInvestors(s)
	normalizeAGI(s, b)
	normalizeAchievements(s)
	if !(s.Stats.FastestRunS >= 0) || math.IsInf(s.Stats.FastestRunS, 0) {
		s.Stats.FastestRunS = 0
	}
}
