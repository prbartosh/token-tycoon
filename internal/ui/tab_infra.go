package ui

import (
	"strconv"

	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

func (m *Model) usageText(need, avail float64) string {
	if !(avail > 0) {
		return "--"
	}
	u := need / avail
	s := pct(u)
	if u > 1 {
		if m.s.Settings.Colorblind {
			s = "!" + s
		}
		return m.th.Bad.Render(s)
	}
	return m.th.Good.Render(s)
}

func (m *Model) infraRows(t *tabList) {
	th, d, c := m.th, &m.s.Derived, m.cat
	s := m.s
	t.text(m.label("ui.infra.compute") + th.Text.Render(m.num(d.ComputeNeed)+" / "+m.num(d.ComputeCap)+"  ") +
		th.Dim.Render("GPU ") + m.usageText(d.ComputeNeed, d.ComputeCap))
	t.text(m.label("ui.infra.energy") + th.Text.Render(m.num(d.EnergyNeed)+" / "+m.mw(d.EnergyAvail)+"  ") +
		th.Dim.Render("PWR ") + m.usageText(d.EnergyNeed, d.EnergyAvail))
	peak := ""
	if d.Peak {
		peak = " (" + c.T("energy.peak") + ")"
	}
	thr := th.Good
	if d.Throttle < 0.999 {
		thr = th.Bad
	}
	t.text(m.label("ui.infra.throttle") + thr.Render(pct(d.Throttle)) + th.Dim.Render("   "+c.T("energy.index", fixed(d.EnergyIndex, 0))+peak))
	t.text(m.label("ui.infra.costs") + th.Text.Render(c.T("ui.infra.costs_val", pct(d.EnergyShare), pct(d.CloudShare))))

	t.blank()
	m.section(t, c.T("ui.sec.racks"))
	for g := range min(len(m.b.GPUGens), s.Infra.GPUGen+1) {
		gen := g + 1
		gg := &m.b.GPUGens[g]
		name := m.cell(c.T("infra.gen", gen), colLabel)
		if gen > s.Infra.GPUGen {
			t.item(actRack, gen, th.Dim.Render(name+c.T("ui.infra.rack_locked", m.money(gg.UnlockLifetime), m.etaLifetime(gg.UnlockLifetime))))
			continue
		}
		q, cost := m.qtyRacks(gen)
		spec := c.T("ui.infra.rack", m.num(gg.Compute), m.num(gg.MW))
		t.item(actRack, gen, th.Text.Render(name)+th.Gold.Render(m.cell("x"+strconv.Itoa(s.Infra.Racks[g]), colOwned))+
			th.Dim.Render(m.cell(spec, 34))+th.Dim.Render("x"+strconv.Itoa(q)+" ")+m.costText(cost))
	}

	t.blank()
	m.section(t, c.T("ui.sec.energy"))
	e := &s.Infra.Energy
	gs := m.gridStep()
	t.item(actGrid, 0, th.Text.Render(m.cell(c.T("energy.grid"), colLabel))+
		th.Text.Render(m.cell(c.T("ui.infra.grid", m.mw(e.GridMW), m.mw(d.GridUsed)), 30))+
		th.Dim.Render(c.T("ui.infra.buy_mw", m.num(gs)))+" "+m.costText(sim.GridCost(m.b, gs)))
	ppa := c.T("ui.infra.ppa_none")
	if e.PPA.MW > 0 && e.PPA.ExpiresS > s.Clock {
		ppa = c.T("ui.infra.ppa", m.mw(e.PPA.MW), fixed(e.PPA.Index, 0), m.dur(e.PPA.ExpiresS-s.Clock))
	}
	ps := m.ppaStep()
	t.item(actPPA, 0, th.Text.Render(m.cell(c.T("energy.ppa"), colLabel))+th.Text.Render(m.cell(ppa, 30))+
		th.Dim.Render(c.T("ui.infra.buy_mw", m.num(ps)))+" "+m.costText(sim.PPACost(m.b, ps)))
	t.item(actSMR, 0, th.Text.Render(m.cell(c.T("energy.smr"), colLabel))+
		th.Text.Render(m.cell(c.T("ui.infra.durable", e.SMR, m.mw(m.b.Energy.SMRMW)), 30))+m.costText(sim.SMRCost(s, m.b)))
	if s.IsUnlocked(sim.UnlockOrbital) {
		t.item(actOrbital, 0, th.Text.Render(m.cell(c.T("energy.orbital"), colLabel))+
			th.Text.Render(m.cell(c.T("ui.infra.durable", e.Orbital, m.mw(m.b.Energy.OrbitalMW)), 30))+m.costText(sim.OrbitalCost(s, m.b)))
	}
	cloud := th.Dim.Render(c.T("ui.infra.cloud_off", pct(m.b.Infra.CloudRevenueShare)))
	if s.Infra.CloudEnabled {
		cloud = th.Magenta.Render(c.T("ui.infra.cloud_on", pct(m.b.Infra.CloudRevenueShare)))
	}
	t.item(actCloud, 0, th.Text.Render(m.cell(c.T("infra.cloud"), colLabel))+cloud)

	if s.IsUnlocked(sim.UnlockData) {
		m.dataRows(t)
	}
}

func (m *Model) dataRows(t *tabList) {
	th, c, s := m.th, m.cat, m.s
	tv := &s.Derived.Train
	t.blank()
	m.section(t, c.T("data.title"))
	t.text(th.Text.Render(c.T("data.stock", m.num(s.Training.DataTokens), m.num(s.Training.SynthTokens))))
	if tv.ScraperPaused {
		t.text(th.Bad.Render(c.T("data.scraper_paused", m.dur(s.Training.ScraperPauseS-s.Clock))))
	}
	q, cost := m.qtyScrapers()
	t.item(actScraper, 0, th.Text.Render(m.cell(c.T("data.scrapers", s.Training.Scrapers, m.num(tv.DataPerSec)), 43))+
		th.Dim.Render("x"+strconv.Itoa(q)+" ")+m.costText(cost))
	t.item(actLicense, 0, th.Text.Render(m.cell(c.T("ui.data.license", m.num(tv.LicenseTokens)), 43))+m.costText(tv.LicenseCost))
	if tv.SynthOK {
		t.item(actSynth, 0, th.Text.Render(widgets.Fit(c.T("ui.data.synth", pct(s.Training.SynthShare), m.num(tv.SynthPerSec)), t.w-4)))
	} else {
		t.item(actSynth, 0, th.Dim.Render(c.T("data.synthetic_locked", fixed(m.b.Training.SyntheticMinBenchmark, 0))))
	}
}
