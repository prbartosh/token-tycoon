package ui

import (
	"strconv"

	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/market"
	"github.com/prbartosh/token-tycoon/internal/sim"
)

type headline struct {
	key  string
	ok   func(m *Model) bool
	args func(m *Model) []any
}

func (m *Model) mv() *sim.MarketView { return &m.s.Derived.Market }

func (m *Model) topRival() (string, float64) {
	mv := m.mv()
	best := -1
	for i := range mv.Rivals {
		if best < 0 || mv.RivalBench[i] > mv.RivalBench[best] {
			best = i
		}
	}
	if best < 0 {
		return "", 0
	}
	return m.cat.T(i18n.RivalKey(m.b.Rivals[best].ID)), mv.RivalBench[best]
}

func (m *Model) bestModelName() string {
	best := -1
	for i := range m.s.Training.Models {
		if best < 0 || m.s.Training.Models[i].Benchmark > m.s.Training.Models[best].Benchmark {
			best = i
		}
	}
	return m.modelName(best)
}

func marketOn(m *Model) bool { return m.mv().On }

var headlines = [...]headline{
	{"news.early_wrapper", func(m *Model) bool { return !m.s.IsUnlocked(sim.UnlockInfra) }, nil},
	{"news.first_manager", func(m *Model) bool { return m.s.Businesses[0].Manager && !m.s.IsUnlocked(sim.UnlockInfra) }, nil},
	{"news.income", func(m *Model) bool { return m.s.Derived.BaseIncome > 0 }, func(m *Model) []any { return []any{m.money(m.s.Derived.BaseIncome)} }},
	{"news.money", func(m *Model) bool { return m.s.Money > 1e6 }, func(m *Model) []any { return []any{m.money(m.s.Money)} }},
	{"news.combo", func(m *Model) bool { return m.s.Combo.Mult >= 2 }, func(m *Model) []any { return []any{fixed(m.s.Combo.Mult, 1)} }},
	{"news.crits", func(m *Model) bool { return m.s.Stats.Crits >= 100 }, func(m *Model) []any { return []any{m.num(float64(m.s.Stats.Crits))} }},
	{"news.gpu_full", func(m *Model) bool {
		d := &m.s.Derived
		return m.s.IsUnlocked(sim.UnlockInfra) && d.ComputeCap > 0 && d.ComputeNeed >= 0.95*d.ComputeCap
	}, nil},
	{"news.energy_high", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockInfra) && m.s.Derived.EnergyIndex >= 65 },
		func(m *Model) []any { return []any{fixed(m.s.Derived.EnergyIndex, 0)} }},
	{"news.energy_low", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockInfra) && m.s.Derived.EnergyIndex <= 35 },
		func(m *Model) []any { return []any{fixed(m.s.Derived.EnergyIndex, 0)} }},
	{"news.cloud", func(m *Model) bool { return m.s.Infra.CloudEnabled }, nil},
	{"news.smr", func(m *Model) bool { return m.s.Infra.Energy.SMR > 0 }, func(m *Model) []any { return []any{m.s.Infra.Energy.SMR} }},
	{"news.version", func(m *Model) bool { return m.s.Prestige.ResetsL1 > 0 }, func(m *Model) []any { return []any{m.versionName()} }},
	{"news.params", func(m *Model) bool { return m.s.Prestige.Params > 0 }, func(m *Model) []any { return []any{m.num(m.s.Prestige.Params)} }},
	{"news.cards", func(m *Model) bool { return m.s.CardMods.Unique > 0 }, func(m *Model) []any { return []any{m.s.CardMods.Unique} }},
	{"news.rival_lead", func(m *Model) bool { _, x := m.topRival(); return marketOn(m) && x > m.mv().Benchmark },
		func(m *Model) []any { n, x := m.topRival(); return []any{n, fixed(x, 1)} }},
	{"news.rank_1", func(m *Model) bool { return marketOn(m) && m.mv().Rank == 1 }, nil},
	{"news.share_low", func(m *Model) bool { return marketOn(m) && m.mv().Share < 0.1 }, func(m *Model) []any { return []any{pct(m.mv().Share)} }},
	{"news.share_high", func(m *Model) bool { return marketOn(m) && m.mv().Share >= 0.4 }, func(m *Model) []any { return []any{pct(m.mv().Share)} }},
	{"news.hype_high", func(m *Model) bool { return marketOn(m) && m.mv().Hype >= 1 }, func(m *Model) []any { return []any{fixed(1+m.mv().Hype, 1)} }},
	{"news.hype_zero", func(m *Model) bool { return marketOn(m) && m.mv().Hype < 0.05 }, nil},
	{"news.free_tier", func(m *Model) bool { return marketOn(m) && m.mv().Alloc[market.SegFree] >= 0.2 }, func(m *Model) []any { return []any{pct(m.mv().Alloc[market.SegFree])} }},
	{"news.open_weights", func(m *Model) bool { return marketOn(m) && m.mv().Pressure < 0.9 }, func(m *Model) []any { return []any{fixed(m.mv().Pressure, 2)} }},
	{"news.training", func(m *Model) bool { return len(m.s.Training.Queue) > 0 }, func(m *Model) []any { return []any{m.num(m.s.Training.Queue[0].Params)} }},
	{"news.model", func(m *Model) bool { return len(m.s.Training.Models) > 0 },
		func(m *Model) []any { return []any{m.bestModelName(), fixed(m.s.Stats.BestBench, 1)} }},
	{"news.scrapers", func(m *Model) bool { return m.s.Training.Scrapers > 0 }, func(m *Model) []any { return []any{m.s.Training.Scrapers} }},
	{"news.synthetic", func(m *Model) bool { return m.s.Training.SynthShare > 0 }, nil},
	{"news.star", func(m *Model) bool { return len(m.s.Talent) > 0 }, func(m *Model) []any { return []any{m.s.Talent[len(m.s.Talent)-1].Name} }},
	{"news.poached", func(m *Model) bool { return m.s.Stats.Poached > 0 }, func(m *Model) []any { return []any{m.s.Stats.Poached} }},
	{"news.reputation", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockContracts) }, func(m *Model) []any { return []any{fixed(m.s.Market.Reputation, 0)} }},
	{"news.contracts", func(m *Model) bool { return m.s.Stats.ContractsDone > 0 }, func(m *Model) []any { return []any{m.s.Stats.ContractsDone} }},
	{"news.round", func(m *Model) bool { return m.s.Prestige.ResetsL2 > 0 },
		func(m *Model) []any { return []any{m.roundName(m.s.Prestige.RoundIndex - 1)} }},
	{"news.valuation", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockInvestors) },
		func(m *Model) []any { return []any{m.money(sim.Valuation(m.s, m.b).Valuation)} }},
	{"news.quarter", func(m *Model) bool { return m.s.Stats.Quarters > 0 },
		func(m *Model) []any { return []any{m.s.Stats.QuartersBeaten, m.s.Stats.Quarters} }},
	{"news.research", func(m *Model) bool { return m.s.Stats.ResearchDone > 0 }, func(m *Model) []any { return []any{m.s.Stats.ResearchDone} }},
	{"news.rules", func(m *Model) bool { return m.s.Stats.RulesFired > 0 }, func(m *Model) []any { return []any{m.num(float64(m.s.Stats.RulesFired))} }},
	{"news.incidents", func(m *Model) bool { return m.s.Stats.IncidentsSeen > 0 },
		func(m *Model) []any { return []any{m.s.Stats.IncidentsOK, m.s.Stats.IncidentsSeen} }},
	{"news.safety_none", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockSafety) && m.s.Stats.IncidentsSeen == 0 }, nil},
	{"news.eu_out", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockRegions) && m.regionOut("eu") }, nil},
	{"news.eu_in", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockRegions) && !m.regionOut("eu") }, nil},
	{"news.asia", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockRegions) && !m.regionOut("asia") }, nil},
	{"news.agi_race", func(m *Model) bool { return m.s.IsUnlocked(sim.UnlockAGI) && !sim.AGIEnded(m.s, m.b) },
		func(m *Model) []any { return []any{min(m.s.AGI.Stage+1, len(m.b.AGIStages)), len(m.b.AGIStages)} }},
	{"news.agi_done", func(m *Model) bool { return sim.AGIEnded(m.s, m.b) }, nil},
	{"news.si", func(m *Model) bool { return m.s.AGI.SILevel > 0 }, func(m *Model) []any { return []any{m.s.AGI.SILevel} }},
	{"news.achievements", func(m *Model) bool { return m.s.Achieved.Count() > 0 },
		func(m *Model) []any { return []any{m.s.Achieved.Count(), len(sim.Achievements)} }},
	{"news.offline", func(m *Model) bool { return m.s.Stats.OfflineS > 3600 }, func(m *Model) []any { return []any{m.dur(m.s.Stats.OfflineS)} }},
}

const genericNews = 12

func (m *Model) regionOut(id string) bool {
	i, ok := m.b.RegionIndex(id)
	return ok && !sim.RegionEntered(m.s, m.b, i)
}

func (m *Model) headline() string {
	if m.newsIdx%2 == 1 {
		return m.cat.T("ui.news." + strconv.Itoa((m.newsIdx/2)%genericNews+1))
	}
	var idx [len(headlines)]int
	n := 0
	for i := range headlines {
		if headlines[i].ok(m) {
			idx[n] = i
			n++
		}
	}
	if n == 0 {
		return m.cat.T("ui.news." + strconv.Itoa((m.newsIdx/2)%genericNews+1))
	}
	h := &headlines[idx[(m.newsIdx/2)%n]]
	if h.args == nil {
		return m.cat.T(h.key)
	}
	return m.cat.T(h.key, h.args(m)...)
}
