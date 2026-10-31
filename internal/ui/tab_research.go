package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

func (m *Model) roundName(idx int) string {
	id, mega := sim.RoundID(m.b, idx)
	if id != "" {
		return m.cat.T(i18n.RoundKey(id))
	}
	return m.cat.T("round.mega", mega)
}

func (m *Model) lastRound() string {
	if m.s.Prestige.RoundIndex == 0 {
		return m.cat.T("ui.research.no_round")
	}
	return m.cat.T("ui.research.round", m.roundName(m.s.Prestige.RoundIndex-1))
}

func (m *Model) nodeName(i int) string { return m.cat.T("research." + m.b.ResearchNodes[i].ID) }

func (m *Model) nodeEffect(i int) string {
	n := &m.b.ResearchNodes[i]
	key := "researchfx." + n.Effect
	var v string
	switch n.Effect {
	case balance.ResComputeMult, balance.ResEnergyCost, balance.ResIncidentMult, balance.ResPenaltyMult:
		v = pct(1 - n.Value)
	case balance.ResEnterprise, balance.ResContractReward, balance.ResOfflineEff, balance.ResCritChance, balance.ResCardChance:
		v = pct(n.Value)
	case balance.ResAutoPrestige:
		v = "x" + fixed(m.b.Research.AutoPrestigeRatio, 1)
	default:
		v = fixed(n.Value, 0)
	}
	if countVerbs(m.cat.T(key)) == 0 {
		return m.cat.T(key)
	}
	return m.cat.T(key, v)
}

func (m *Model) researchRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	t.text(th.Capital.Render(c.T("ui.research.capital", m.num(s.Prestige.Capital))) + "   " +
		th.Text.Render(m.lastRound()) + "   " +
		th.Dim.Render(c.T("ui.research.speed", pct(s.Derived.Talent.Mult-1))))
	pv := sim.Funding(s, b)
	fund := th.Dim.Render("[ " + c.T("ui.research.funding_locked", m.money(pv.Have), m.money(pv.Need)) + " ]")
	if pv.Allowed {
		fund = th.Bold.Render("[ " + c.T("ui.research.funding", m.roundName(pv.RoundNow), m.num(pv.Gain)) + " ]")
	}
	t.item(actFunding, 0, fund)
	if a := s.Research.Active; a >= 0 {
		var bar strings.Builder
		widgets.Track(&bar, s.Research.Progress, 12, th.Capital, th.Track)
		t.text(th.Text.Render(c.T("ui.research.active", m.nodeName(a))+" ") + bar.String() +
			th.Dim.Render(" "+c.T("ui.research.left", m.dur(sim.ResearchLeftS(s, b)))))
	} else {
		t.text(th.Dim.Render(c.T("ui.research.idle")))
	}
	for br := range balance.Branches {
		t.blank()
		m.section(t, c.T("research.branch."+balance.Branches[br]))
		for i := range b.ResearchNodes {
			if b.NodeBranch(i) != br {
				continue
			}
			t.item(actNode, i, m.nodeLine(i))
		}
	}
}

func (m *Model) nodeLine(i int) string {
	th, c, s, b := m.th, m.cat, m.s, m.b
	n := &b.ResearchNodes[i]
	st := sim.NodeState(s, b, i)
	mark, ps := "[ ]", th.Text
	switch st {
	case sim.NodeDone:
		mark, ps = "[+]", th.Good
	case sim.NodeActive:
		mark, ps = "[~]", th.Capital
	case sim.NodeLocked:
		mark, ps = "[-]", th.Dim
	}
	line := ps.Render(mark+" "+m.cell(m.nodeName(i), 22)+" ") + th.Dim.Render(m.cell(m.nodeEffect(i), 30))
	switch st {
	case sim.NodeDone:
		return line + th.Good.Render(c.T("ui.research.owned"))
	case sim.NodeActive:
		return line + th.Capital.Render(c.T("ui.research.researching"))
	case sim.NodeLocked:
		var req []string
		for _, r := range b.NodeRequires(i) {
			if !s.Research.Done.Has(r) {
				req = append(req, m.nodeName(r))
			}
		}
		return line + th.Dim.Render(c.T("ui.research.requires", strings.Join(req, ", ")))
	}
	cost := th.Capital
	if n.Cost > s.Prestige.Capital {
		cost = th.Bad
	}
	return line + cost.Render(c.T("ui.research.cost", m.num(n.Cost))) + th.Dim.Render(" "+m.dur(sim.ResearchTimeS(s, b, i)))
}

func (m *Model) openFunding() tea.Cmd {
	if !m.s.IsUnlocked(sim.UnlockLayer2) {
		m.setStatus(m.cat.T("funding.not_yet", m.money(m.b.Prestige.Layer2.UnlockLifetime)), true)
		return nil
	}
	m.modal = modal{kind: modalFunding}
	return nil
}

func (m *Model) fundingBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	pv := sim.Funding(m.s, m.b)
	body := []string{"", " " + th.Capital.Render(m.roundName(pv.RoundNow)), ""}
	if !pv.Allowed {
		body = append(body,
			" "+th.Bad.Render(c.T("funding.locked", m.money(pv.Need))),
			" "+th.Text.Render(c.T("funding.progress", m.money(pv.Have), m.money(pv.Need))), "",
			widgets.Center(th.KeyText.Render(c.T("ui.key.close")), w-2))
		return m.box(c.T("funding.title"), body, w)
	}
	body = append(body,
		" "+th.Gold.Render(c.T("funding.gain", m.num(pv.Gain), m.num(pv.CapitalAfter))),
		" "+th.Text.Render(c.T("funding.valuation", m.money(pv.Valuation))),
		" "+th.Text.Render(c.T("funding.round_time", m.dur(pv.RoundS))),
	)
	for _, l := range wrapLines(c.T("ui.funding.round_mult", m.num(pv.RoundMult), m.num(pv.RoundMultNew)), w-4) {
		body = append(body, " "+th.Params.Render(l))
	}
	body = append(body, "")
	for _, l := range wrapLines(c.T("funding.resets"), w-4) {
		body = append(body, " "+th.Dim.Render(l))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.key.confirm"))+"   "+th.KeyText.Render(c.T("ui.key.cancel")), w-2))
	return m.box(c.T("funding.title"), body, w)
}
