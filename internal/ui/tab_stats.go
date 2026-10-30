package ui

import (
	"strconv"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/format"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/theme"
)

func (m *Model) statLabel(key string) string {
	return m.th.Dim.Render(m.cell(m.cat.T(key), 2*labelW))
}

func (m *Model) statsRows(t *tabList) {
	s, th := m.s, m.th
	owned, managers := 0, 0
	for i := range sim.N {
		owned += s.Businesses[i].Owned
		if s.Businesses[i].Manager {
			managers++
		}
	}
	row := func(key, val string) { t.text(m.statLabel(key) + th.Text.Render(val)) }
	row("ui.stats.play", format.Duration(s.Stats.PlayS))
	row("ui.stats.offline", format.Duration(s.Stats.OfflineS))
	row("ui.stats.lifetime", m.money(s.Lifetime.Total))
	row("ui.stats.run", m.money(s.Lifetime.Run))
	row("ui.stats.businesses", strconv.Itoa(owned))
	row("ui.stats.managers", strconv.Itoa(managers))
	row("ui.stats.upgrades", strconv.Itoa(s.Upgrades.Count())+"/"+strconv.Itoa(len(m.b.Upgrades)))
	row("ui.stats.purchases", strconv.Itoa(s.Stats.Purchases))
	row("ui.stats.crits", strconv.Itoa(s.Stats.Crits))
	row("ui.stats.best_combo", "x"+format.Number(max(1, s.Stats.BestCombo), format.Letters))
	row("ui.stats.boosts", strconv.Itoa(s.Stats.BoostPresses))
	row("ui.stats.manual", strconv.Itoa(s.Stats.ManualStarts))
	row("ui.stats.version", sim.VersionName(m.b, s.Prestige.VersionIndex))
	row("ui.stats.params", m.num(s.Prestige.Params))
	row("ui.stats.resets", strconv.Itoa(s.Prestige.ResetsL1))
	row("ui.stats.events", strconv.Itoa(s.Stats.EventsCaught)+" / "+strconv.Itoa(s.Stats.EventsMissed)+" / "+strconv.Itoa(s.Stats.EventsSeen))
	row("ui.stats.grants", m.cat.T("ui.stats.streak_val", s.Stats.Grants, s.Streak.Day))
	row("ui.stats.models", strconv.Itoa(s.Stats.Models))
	row("ui.stats.best_bench", fixed(s.Stats.BestBench, 1))
	if s.IsUnlocked(sim.UnlockTalent) {
		row("ui.stats.talent", m.cat.T("ui.stats.talent_val", s.Stats.Hired, s.Stats.Retained, s.Stats.Poached))
	}
	if s.IsUnlocked(sim.UnlockContracts) {
		row("ui.stats.contracts", m.cat.T("ui.stats.contracts_val", s.Stats.ContractsDone, s.Stats.ContractsFailed, fixed(s.Market.Reputation, 0)))
	}
	if s.IsUnlocked(sim.UnlockLayer2) {
		row("ui.stats.rounds", strconv.Itoa(s.Prestige.ResetsL2))
		row("ui.stats.capital", m.num(s.Prestige.CapitalLifetime))
		row("ui.stats.research", strconv.Itoa(s.Research.Done.Count())+"/"+strconv.Itoa(len(m.b.ResearchNodes)))
	}
	if s.IsUnlocked(sim.UnlockRules) {
		row("ui.stats.rules", strconv.Itoa(s.Stats.RulesFired))
	}
	m.m5Stats(row)
	if s.IsUnlocked(sim.UnlockCards) {
		t.blank()
		m.collectionRows(t)
	}
	t.blank()
	m.achievementRows(t)
}

func (m *Model) rarityPaint(r int) theme.Paint {
	switch balance.Rarities[r] {
	case "rare":
		return m.th.Rare
	case "epic":
		return m.th.Epic
	case "legendary":
		return m.th.Gold
	}
	return m.th.Text
}

func (m *Model) collectionRows(t *tabList) {
	c, s, th := m.cat, m.s, m.th
	m.section(t, c.T("ui.sec.cards")+"  "+c.T("cards.collection", s.CardMods.Unique, len(m.b.Card)))
	for r := range balance.Rarities {
		ids := m.b.CardsOfRarity(r)
		have := 0
		for _, i := range ids {
			if s.Cards[m.b.Card[i].ID] > 0 {
				have++
			}
		}
		st := m.rarityPaint(r)
		head := st.Render(c.T(i18n.RarityKey(balance.Rarities[r])) + " " + strconv.Itoa(have) + "/" + strconv.Itoa(len(ids)))
		if s.CardMods.SetsComplete[r] {
			head += "  " + th.Gold.Render(c.T("cards.set_bonus", fixed(m.b.Cards.SetBonus[balance.Rarities[r]]*100, 0)))
		}
		t.text(head)
		for _, i := range ids {
			lv := sim.CardLevel(s, m.b, i)
			if lv == 0 {
				t.text(th.Dim.Render("  " + c.T("ui.cards.unknown")))
				continue
			}
			t.text("  " + st.Render(m.cell(c.T(i18n.CardKey(m.b.Card[i].ID)), 28)) +
				th.Text.Render(m.cell(c.T("cards.level", lv), 10)) + th.Dim.Render(m.cardEffect(i)))
		}
	}
}

func (m *Model) cardEffect(i int) string {
	cd := &m.b.Card[i]
	v := sim.CardBonus(m.s, m.b, i)
	key := "cardfx." + cd.Effect
	switch cd.Effect {
	case balance.CardProfitBusiness:
		name := ""
		if tg := m.b.CardTarget(i); tg >= 0 {
			name = m.bizName(tg)
		}
		return m.cat.T(key, name, fixed(v*100, 1))
	case balance.CardRevealRatio:
		return m.cat.T(key)
	case balance.CardCritMult:
		return m.cat.T(key, fixed(v, 2))
	}
	return m.cat.T(key, fixed(v*100, 1))
}
