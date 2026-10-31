package ui

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/market"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

type cerKind uint8

const (
	cerNone cerKind = iota
	cerPrestige
	cerRelease
	cerFunding
)

const (
	cerAnimS   = 2.5
	cerHoldS   = 3.5
	cerW       = 58
	lossPoints = 60
	chartRows  = 4
	cerBars    = 3
	cerLabelW  = 14
	cerNumW    = 7
)

type ceremony struct {
	kind      cerKind
	age       float64
	name      string
	from      string
	value     float64
	extra     float64
	rank      int
	rankFrom  int
	total     int
	rivals    [cerBars]float64
	rivalName [cerBars]string
	nRivals   int
}

var lossCurve = func() [lossPoints]float64 {
	var c [lossPoints]float64
	for i := range c {
		x := float64(i) / float64(lossPoints-1)
		c[i] = 1.7 + 2.6*math.Exp(-4.2*x) + 0.09*math.Sin(31*x)*(1-x)
	}
	return c
}()

func (c *ceremony) active() bool { return c.kind != cerNone }

func (c *ceremony) frac() float64 { return math.Min(1, c.age/cerAnimS) }

func ease(f float64) float64 { return 1 - math.Pow(1-math.Min(1, math.Max(0, f)), 3) }

func (m *Model) advanceCeremony(dt float64) {
	c := &m.cer
	if !c.active() {
		return
	}
	c.age += dt
	if c.age >= cerAnimS+cerHoldS {
		*c = ceremony{}
	}
}

func (m *Model) skipCeremony() { m.cer = ceremony{} }

func (m *Model) startPrestige(e *sim.Event) {
	m.cer = ceremony{kind: cerPrestige, name: sim.VersionName(m.b, e.Index), from: sim.VersionName(m.b, e.Index-1),
		value: e.Amount, extra: e.Value}
	m.ring = true
}

func (m *Model) startFunding(e *sim.Event, name string) {
	m.cer = ceremony{kind: cerFunding, name: name, value: e.Value, extra: e.Amount}
	m.ring = true
}

func (m *Model) startRelease(e *sim.Event, name string) {
	mv := &m.s.Derived.Market
	rivals := mv.RivalBench[:mv.Rivals]
	prev := -1.0
	for k := range m.s.Training.Models {
		if k != e.Index {
			prev = math.Max(prev, m.s.Training.Models[k].Benchmark)
		}
	}
	from := len(rivals) + 1
	if prev >= 0 {
		from = market.Rank(prev, rivals)
	}
	c := ceremony{kind: cerRelease, name: name, value: e.Value, extra: e.Amount, rank: e.Count, rankFrom: max(from, e.Count),
		total: len(rivals) + 1}
	var used [len(mv.RivalBench)]bool
	for c.nRivals < cerBars {
		best := -1
		for i, v := range rivals {
			if !used[i] && (best < 0 || v > rivals[best]) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		used[best] = true
		c.rivals[c.nRivals] = rivals[best]
		if best < len(m.b.Rivals) {
			c.rivalName[c.nRivals] = m.cat.T(i18n.RivalKey(m.b.Rivals[best].ID))
		}
		c.nRivals++
	}
	m.cer = c
	m.ring = true
}

func (m *Model) ceremonyBox() string {
	switch m.cer.kind {
	case cerPrestige:
		return m.prestigeCeremony()
	case cerRelease:
		return m.releaseCeremony()
	case cerFunding:
		return m.fundingCeremony()
	}
	return ""
}

func (m *Model) cerFooter(w int) string {
	return widgets.Center(m.th.KeyText.Render(m.cat.T("ui.cer.skip")), w-2)
}

func (m *Model) prestigeCeremony() string {
	th, c, cr := m.th, m.cat, &m.cer
	w := min(cerW, m.width-2)
	f := cr.frac()
	n := max(1, int(math.Round(f*float64(lossPoints-1)))+1)
	lo, hi := lossCurve[lossPoints-1]-0.1, lossCurve[0]+0.1
	m.plot.Reset(max(4, w-6), chartRows)
	m.plot.Series(lossCurve[:n], lo, hi, lossPoints)
	body := []string{"", widgets.Center(th.Params.Render(cr.from+"  >  ")+th.Gold.Render(cr.name), w-2), ""}
	for r := range chartRows {
		m.sb.Reset()
		m.plot.Row(&m.sb, r)
		body = append(body, "  "+th.Magenta.Render(m.sb.String()))
	}
	body = append(body, " "+th.Dim.Render(c.T("ui.cer.loss", fixed(lossCurve[n-1], 3), pct(f))))
	if f >= 1 {
		body = append(body, " "+th.Gold.Render(c.T("ui.cer.params", m.num(cr.value), m.num(cr.extra))))
	} else {
		body = append(body, "")
	}
	body = append(body, "", m.cerFooter(w))
	return m.box(c.T("ui.cer.prestige"), body, w)
}

func (m *Model) cerBar(label string, bench, frac float64, gold bool, w int) string {
	th := m.th
	barW := max(4, w-6-cerLabelW-cerNumW)
	m.sb.Reset()
	fill := th.Rate
	if gold {
		fill = th.Gold
	}
	widgets.Track(&m.sb, bench/100*frac, barW, fill, th.Track)
	st := th.Text
	if gold {
		st = th.Gold
	}
	return " " + st.Render(m.cell(label, cerLabelW)) + m.sb.String() + st.Render(widgets.FitRight(fixed(bench*frac, 1), cerNumW))
}

func (m *Model) releaseCeremony() string {
	th, c, cr := m.th, m.cat, &m.cer
	w := min(cerW, m.width-2)
	e := ease(cr.frac())
	body := []string{"", widgets.Center(th.Gold.Render(cr.name), w-2), ""}
	for i := range cr.nRivals {
		body = append(body, m.cerBar(cr.rivalName[i], cr.rivals[i], 1, false, w))
	}
	body = append(body, m.cerBar(cr.name, cr.value, e, true, w), "")
	rank := cr.rankFrom - int(math.Round(float64(cr.rankFrom-cr.rank)*e))
	rk := c.T("ui.cer.rank", rank, cr.total)
	if cr.rankFrom != cr.rank {
		rk = c.T("ui.cer.rank_jump", cr.rankFrom, rank, cr.total)
	}
	body = append(body, " "+th.Text.Render(rk), " "+th.Text.Render(c.T("market.share", pct(cr.extra*e))), "", m.cerFooter(w))
	return m.box(c.T("ui.release.title"), body, w)
}

func (m *Model) fundingCeremony() string {
	th, c, cr := m.th, m.cat, &m.cer
	w := min(cerW, m.width-2)
	e := ease(cr.frac())
	body := []string{
		"",
		widgets.Center(th.Capital.Render(cr.name), w-2),
		"",
		widgets.Center(th.Text.Render(c.T("funding.valuation_label")), w-2),
		widgets.Center(th.Money.Render(m.money(cr.value*e)), w-2),
		"",
		" " + th.Gold.Render(c.T("funding.raised", m.num(cr.extra), m.num(m.s.Prestige.Capital))),
		"",
		m.cerFooter(w),
	}
	return m.box(c.T("funding.ceremony"), body, w)
}
