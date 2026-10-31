package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/prbartosh/token-tycoon/internal/format"
	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

func (m *Model) regionName(i int) string {
	if i < 0 || i >= len(m.b.Regions) {
		return ""
	}
	return m.cat.T(i18n.RegionKey(m.b.Regions[i].ID))
}

func (m *Model) incidentName(i int) string {
	if i < 0 || i >= len(m.b.Incidents) {
		return ""
	}
	return m.cat.T("incident." + m.b.Incidents[i].ID)
}

func (m *Model) agiName(i int) string {
	if i < 0 || i >= len(m.b.AGIStages) {
		return ""
	}
	return m.cat.T(i18n.AGIKey(m.b.AGIStages[i].ID))
}

func (m *Model) regionRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	mv := &s.Derived.Market
	t.blank()
	m.section(t, c.T("ui.sec.regions", sim.RegionsEntered(s, b), len(b.Regions)))
	t.text(th.Dim.Render(c.T("ui.regions.summary", "x"+fixed(mv.RegionMult, 2), pct(mv.EntShare))))
	for i := range b.Regions {
		r := &b.Regions[i]
		name := th.Text.Render(m.cell(m.regionName(i), 8)) + th.Dim.Render(m.cell(pct(r.MarketShare), 6))
		if sim.RegionEntered(s, b, i) {
			t.item(actRegion, i, name+th.Good.Render(m.cell(c.T("ui.regions.entered"), 14))+
				th.Dim.Render(c.T("region.entry."+r.Entry)))
		} else {
			t.item(actRegion, i, name+m.costText(sim.EntryCost(s, b, i))+
				th.Dim.Render("  "+c.T("region.entry."+r.Entry)+"  "+c.T("ui.regions.gain", pct(sim.RegionGain(s, b, i, false)))))
		}
		if !(r.EnterpriseCostMin > 0) {
			continue
		}
		label := th.Dim.Render("   " + m.cell(c.T("ui.regions.enterprise"), 24))
		switch {
		case sim.RegionCompliant(s, b, i):
			t.item(actCompliance, i, label+th.Good.Render(c.T("ui.regions.compliant")))
		case !sim.RegionEntered(s, b, i):
			t.item(actCompliance, i, label+th.Dim.Render(c.T("ui.regions.enter_first")))
		default:
			t.item(actCompliance, i, label+m.costText(sim.ComplianceCost(s, b, i)))
		}
	}
}

func (m *Model) investorRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	v := sim.Valuation(s, b)
	t.blank()
	m.section(t, c.T("ui.sec.investors"))
	t.text(th.Money.Render(c.T("ui.investors.valuation", m.money(v.Valuation))) + "   " +
		th.Dim.Render(c.T("ui.investors.mult", m.money(v.Annual), fixed(v.HypeMult, 2), fixed(v.GrowthMult, 2), pct(v.Growth7d))))
	var bar strings.Builder
	widgets.Track(&bar, v.Progress, 14, th.Rate, th.Track)
	t.text(th.Text.Render(m.cell(c.T("ui.investors.quarter"), 10)) + bar.String() +
		th.Text.Render(" "+c.T("ui.investors.target", m.money(v.Revenue), m.money(v.Target))) +
		th.Dim.Render("  "+c.T("ui.investors.left", m.dur(v.LeftS))))
	t.text(th.Dim.Render(c.T("ui.investors.record", s.Stats.QuartersBeaten, s.Stats.Quarters)) + "   " +
		th.Capital.Render(c.T("ui.investors.bonus", pct(v.Bonus))))
}

func (m *Model) agiRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	v := sim.AGIStatus(s, b)
	m.section(t, c.T("ui.sec.agi", min(v.Stage, v.Stages), v.Stages))
	for i := range b.AGIStages {
		st := &b.AGIStages[i]
		line := m.cell(strconv.Itoa(i+1)+". "+m.agiName(i), 18)
		fx := th.Dim.Render(m.cell(c.T("agi.fx."+st.Effect), 34))
		switch {
		case i < v.Stage:
			t.item(actAGI, i, th.Good.Render("[+] "+line)+fx+th.Good.Render(c.T("ui.agi.done")))
		case i == v.Stage && v.Training:
			var bar strings.Builder
			full := b.AGI.FinalTrainingH * 3600
			widgets.Track(&bar, 1-v.TrainLeftS/full, 12, th.Capital, th.Track)
			t.item(actAGI, i, th.Capital.Render("[~] "+line)+fx+bar.String()+th.Dim.Render(" "+c.T("ui.research.left", m.dur(v.TrainLeftS))))
		case i == v.Stage:
			var bar strings.Builder
			widgets.Track(&bar, v.Invested/st.Cost, 12, th.Gold, th.Track)
			t.item(actAGI, i, th.Gold.Render("[ ] "+line)+fx+bar.String()+" "+m.costText(st.Cost-v.Invested))
		default:
			t.item(actAGI, i, th.Dim.Render("[-] "+line)+fx+th.Dim.Render(m.money(st.Cost)))
		}
	}
	if !v.Ended {
		t.blank()
		return
	}
	t.item(actEnding, 0, th.Gold.Render("[ "+c.T("ui.agi.ending_row")+" ]"))
	t.text(th.Magenta.Render(c.T("ui.agi.si", v.SILevel, "x"+fixed(sim.AGIProfitMult(s, b), 1))) + "   " +
		th.Dim.Render(c.T("ui.agi.next_si", m.money(v.NextSI))))
	t.text(th.Dim.Render(c.T("ui.agi.ended_at", m.dur(s.AGI.EndedS))))
	for k := 0; k < v.SILevel; k++ {
		t.text(th.Dim.Render("  " + c.T("ui.agi.record", k+1, m.dur(s.AGI.SIRecords[k]))))
	}
	t.blank()
}

func (m *Model) achievementRows(t *tabList) {
	c, s, th := m.cat, m.s, m.th
	m.section(t, c.T("ui.sec.achievements", s.Achieved.Count(), len(sim.Achievements)))
	for i := range sim.Achievements {
		id := sim.Achievements[i].ID
		if s.Achieved.Has(i) {
			t.text(th.Gold.Render("[+] "+m.cell(c.T("ach."+id), 26)) + th.Dim.Render(c.T("ach."+id+".desc")))
		} else {
			t.text(th.Dim.Render("[ ] " + m.cell(c.T("ach."+id), 26) + c.T("ach."+id+".desc")))
		}
	}
}

func (m *Model) m5Stats(row func(key, val string)) {
	s, b, c := m.s, m.b, m.cat
	if s.IsUnlocked(sim.UnlockSafety) {
		row("ui.stats.incidents", c.T("ui.stats.incidents_val", s.Stats.IncidentsOK, s.Stats.IncidentsMissed, s.Stats.IncidentsSeen))
	}
	if s.IsUnlocked(sim.UnlockInvestors) {
		row("ui.stats.quarters", strconv.Itoa(s.Stats.QuartersBeaten)+" / "+strconv.Itoa(s.Stats.Quarters))
	}
	if s.IsUnlocked(sim.UnlockRegions) {
		row("ui.stats.regions", c.T("ui.stats.regions_val", sim.RegionsEntered(s, b), sim.RegionsCompliant(s, b), len(b.Regions)))
	}
	if s.IsUnlocked(sim.UnlockAGI) {
		row("ui.stats.agi", strconv.Itoa(min(s.AGI.Stage, len(b.AGIStages)))+"/"+strconv.Itoa(len(b.AGIStages)))
		row("ui.stats.agi_invested", m.money(s.Stats.AGIInvested))
	}
}

func (m *Model) agiEnter(k tkey) tea.Cmd {
	if k.idx == m.s.AGI.Stage && !sim.AGIEnded(m.s, m.b) && m.s.AGI.TrainEndsS == 0 {
		m.openAGIInvest()
	}
	return nil
}

func (m *Model) endingBox() string {
	th, c, s, b := m.th, m.cat, m.s, m.b
	w := min(modalW+8, m.width-2)
	body := []string{"", widgets.Center(th.Gold.Render(c.T("ending.headline")), w-2), ""}
	for _, l := range wrapLines(c.T("ending.text"), w-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	body = append(body, "")
	stat := func(key, val string) {
		body = append(body, " "+th.Dim.Render(m.cell(c.T(key), 26))+th.Text.Render(val))
	}
	stat("ending.time", format.Duration(s.AGI.EndedS))
	stat("ui.stats.play", format.Duration(s.Stats.PlayS))
	stat("ui.stats.lifetime", m.money(s.Lifetime.Total))
	stat("ending.prestiges", strconv.Itoa(s.Prestige.ResetsL1)+" / "+strconv.Itoa(s.Prestige.ResetsL2))
	stat("ui.stats.models", strconv.Itoa(s.Stats.Models))
	stat("ui.stats.crits", strconv.Itoa(s.Stats.Crits))
	stat("ending.cards", strconv.Itoa(s.CardMods.Unique)+"/"+strconv.Itoa(len(b.Card)))
	stat("ending.achievements", strconv.Itoa(s.Achieved.Count())+"/"+strconv.Itoa(len(sim.Achievements)))
	body = append(body, "", widgets.Center(th.Title.Render(c.T("ending.credits")), w-2))
	for k := 1; k <= endingCredits; k++ {
		body = append(body, widgets.Center(th.Dim.Render(c.T("ending.credit."+strconv.Itoa(k))), w-2))
	}
	body = append(body, "", widgets.Center(th.KeyText.Render(c.T("ending.continue")), w-2))
	return m.box(c.T("ending.title"), body, w)
}

const endingCredits = 4

func (m *Model) incidentBanner() (string, bool) {
	i, left := sim.ActiveIncident(m.s, m.b)
	if i < 0 || !(left > 0) {
		return "", false
	}
	return m.cat.T("ui.incident.banner", m.incidentName(i), strings.ToUpper(m.b.Safety.Key), m.dur(left)), true
}

func (m *Model) agiEffectText(i int) string {
	if i < 0 || i >= len(m.b.AGIStages) {
		return ""
	}
	return m.cat.T("agi.fx." + m.b.AGIStages[i].Effect)
}
