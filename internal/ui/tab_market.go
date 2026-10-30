package ui

import (
	"math"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/i18n"
	"token-tycoon/internal/market"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

type rankEntry struct {
	name  string
	bench float64
	share float64
	rival int
}

func (m *Model) ranking() []rankEntry {
	mv := &m.s.Derived.Market
	exp := m.b.Market.ShareExponent
	out := make([]rankEntry, 0, mv.Rivals+1)
	out = append(out, rankEntry{name: m.cat.T("market.you"), bench: mv.Benchmark, rival: -1})
	for i := range mv.Rivals {
		out = append(out, rankEntry{name: m.cat.T(i18n.RivalKey(m.b.Rivals[i].ID)), bench: mv.RivalBench[i], rival: i})
	}
	sum := 0.0
	for _, e := range out {
		sum += math.Pow(math.Max(0, e.bench), exp)
	}
	for i := range out {
		if sum > 0 {
			out[i].share = math.Pow(math.Max(0, out[i].bench), exp) / sum
		}
	}
	slices.SortStableFunc(out, func(a, b rankEntry) int {
		switch {
		case a.bench > b.bench:
			return -1
		case a.bench < b.bench:
			return 1
		}
		return 0
	})
	return out
}

func (m *Model) marketRows(t *tabList) {
	th, c, s := m.th, m.cat, m.s
	mv := &s.Derived.Market
	t.text(th.Money.Render(c.T("market.price", m.money(mv.Price))) + "   " + th.Text.Render(c.T("market.share", pct(mv.Share))) +
		"   " + th.Text.Render(c.T("market.rank", mv.Rank, mv.Rivals+1)) + "   " + th.Magenta.Render(c.T("market.hype", fixed(1+mv.Hype, 2)+"x")))
	q := &mv.Quote[market.SegPro]
	t.text(th.Dim.Render(c.T("market.quality", fixed(q.Quality, 2)) + "  " + c.T("market.demand", fixed(q.Demand, 2)) + "  " +
		c.T("market.deflation", fixed(q.Deflation, 2))))
	t.text(th.Dim.Render(c.T("market.pressure", fixed(mv.Pressure, 2)) + "  " + c.T("market.median", fixed(mv.Median, 1))))
	if mv.Model[market.SegPro] < 0 {
		t.text(th.Dim.Render(c.T("market.base_product")))
	}

	t.blank()
	m.section(t, c.T("ui.sec.ranking"))
	for i, e := range m.ranking() {
		var bar strings.Builder
		widgets.Track(&bar, e.share, 12, th.Rate, th.Track)
		st := th.Text
		if e.rival < 0 {
			st = th.Gold
		}
		line := st.Render(m.cell(strconv.Itoa(i+1)+". "+e.name, 18)) + st.Render(m.cell(c.T("ui.model.bench", fixed(e.bench, 1)), 11)) +
			bar.String() + " " + st.Render(pct(e.share))
		if e.rival >= 0 && mv.RivalLastS[e.rival] > 0 {
			line += th.Dim.Render("  " + c.T("ui.market.released", m.dur(s.Clock-mv.RivalLastS[e.rival])))
		}
		t.text(line)
	}

	t.blank()
	m.section(t, c.T("ui.sec.segments"))
	ms := &m.b.Market.Segments
	for seg := range sim.NumSegments {
		name := market.SegmentNames[seg]
		var bar strings.Builder
		widgets.Track(&bar, mv.Alloc[seg], 10, th.Magenta, th.Track)
		desc := c.T("segment." + name + ".desc")
		if seg == market.SegEnterprise {
			desc = c.T("segment.enterprise.desc", fixed(ms.EnterprisePrice, 0), fixed(ms.EnterpriseMinBenchmark, 0), pct(ms.EnterpriseMaxAlloc))
		}
		model := "--"
		if mi := mv.Model[seg]; mi >= 0 && mi < len(s.Training.Models) {
			model = s.Training.Models[mi].Name
		}
		st := th.Text
		if seg == market.SegEnterprise && !mv.EnterpriseOK {
			st = th.Dim
		}
		t.item(actSegment, seg, st.Render(m.cell(c.T("segment."+name), 12))+st.Render(m.cell(pct(mv.Alloc[seg]), 6))+bar.String()+" "+
			widgets.Fit(m.costText(mv.SegPrice[seg]), 10)+th.Dim.Render(" "+m.cell(model, 8)+" "+desc))
	}
	t.text(th.Dim.Render(c.T("ui.market.hype_gain", fixed(mv.HypeGain*3600, 3), fixed(sim.SteadyHype(s, m.b, mv.Alloc[market.SegFree]), 2))))
}

func (m *Model) adjustSegment(seg int, delta float64) tea.Cmd {
	sg := m.s.Market.Segments
	v := [sim.NumSegments]float64{sg.Free, sg.Pro, sg.Enterprise}
	sum := v[0] + v[1] + v[2]
	if !(sum > 0) {
		v = [sim.NumSegments]float64{0, 1, 0}
	} else {
		for i := range v {
			v[i] /= sum
		}
	}
	from := market.SegPro
	if seg == market.SegPro {
		from = market.SegFree
		if v[from] <= 0 && delta > 0 {
			from = market.SegEnterprise
		}
	}
	room := v[from]
	if delta < 0 {
		room = v[seg]
	}
	d := math.Min(math.Abs(delta), room)
	if !(d > 0) {
		return nil
	}
	if delta < 0 {
		d = -d
	}
	v[seg] += d
	v[from] -= d
	for i := range v {
		v[i] = math.Round(math.Max(0, v[i])*1000) / 1000
	}
	return m.exec(sim.SetSegments(v[0], v[1], v[2]))
}
