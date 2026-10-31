package ui

import (
	"math"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/market"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

type trainDraft struct {
	n, d, share float64
	ok          bool
}

func (m *Model) suggestDraft() {
	share := m.draft.share
	if !(share > 0) {
		share = draftShare
	}
	n, d, _ := sim.SuggestTraining(m.s, m.b, share, draftTimeS)
	if !(n > 0) {
		n = m.b.Training.MinParams
	}
	if !(d > 0) {
		d = 20 * n
	}
	m.draft = trainDraft{n: n, d: d, share: share, ok: true}
	m.clampDraft()
}

func (m *Model) ensureDraft() {
	if !m.draft.ok {
		m.suggestDraft()
	}
}

func (m *Model) clampDraft() {
	t := &m.b.Training
	hi := math.Max(t.MinParams, m.s.Derived.Train.MaxParams)
	m.draft.n = math.Min(hi, math.Max(t.MinParams, m.draft.n))
	m.draft.d = math.Max(1, m.draft.d)
	m.draft.share = math.Min(1, math.Max(shareStep, math.Round(m.draft.share*100)/100))
}

func (m *Model) segName(seg string) string {
	if seg == "" {
		return m.cat.T("ui.model.all")
	}
	return m.cat.T("segment." + seg)
}

func (m *Model) cycleSegment(i int) tea.Cmd {
	ms := m.s.Training.Models
	if i < 0 || i >= len(ms) {
		return nil
	}
	order := [...]string{"", market.SegmentNames[market.SegFree], market.SegmentNames[market.SegPro], market.SegmentNames[market.SegEnterprise]}
	cur := 0
	for k, o := range order {
		if o == ms[i].Segment {
			cur = k
		}
	}
	next := order[(cur+1)%len(order)]
	if next == market.SegmentNames[market.SegEnterprise] && ms[i].Benchmark < m.b.Market.Segments.EnterpriseMinBenchmark {
		next = ""
	}
	return m.exec(sim.AssignModel(i, next))
}

func (m *Model) modelRows(t *tabList) {
	th, c, s := m.th, m.cat, m.s
	tv := &s.Derived.Train
	tb := &m.b.Training
	m.ensureDraft()
	t.text(th.Dim.Render(c.T("training.cluster", m.num(tv.ClusterFLOPS)) + "   " + c.T("training.max_params", m.num(tv.MaxParams)) +
		"   " + c.T("training.arch", fixed(tv.ArchMult, 2))))
	t.text(th.Dim.Render(c.T("data.stock", m.num(s.Training.DataTokens), m.num(s.Training.SynthTokens))))

	t.blank()
	m.section(t, c.T("ui.sec.new_model"))
	dr := &m.draft
	est := sim.EstimateTraining(s, m.b, dr.n, dr.d, dr.share)
	t.item(actTrainN, 0, m.cell(c.T("ui.train.params"), colLabel)+th.Gold.Render(m.cell(m.num(dr.n), 10))+th.Dim.Render(c.T("ui.train.ratio", m.num(dr.d/dr.n))))
	tok := m.cell(c.T("ui.train.tokens"), colLabel) + th.Gold.Render(m.cell(m.num(dr.d), 10))
	if est.Optimal > 0 {
		tok += th.Dim.Render(c.T("training.optimal", m.num(est.Optimal), m.num(est.Optimal/dr.n)))
	}
	t.item(actTrainD, 0, tok)
	t.item(actTrainShare, 0, m.cell(c.T("ui.train.share"), colLabel)+th.Gold.Render(m.cell(pct(dr.share), 10))+th.Dim.Render(c.T("ui.train.share_note")))
	estTxt := c.T("ui.train.estimate", fixed(est.Benchmark, 1), m.dur(est.TimeS), m.num(est.FLOPs))
	if est.SynthFrac > 0 {
		estTxt += "  " + c.T("training.synthetic_penalty", pct(est.SynthFrac))
	}
	t.text(m.cell("", colLabel) + th.Text.Render(estTxt))
	if !est.DataOK {
		t.text(m.cell("", colLabel) + th.Bad.Render(c.T("msg.no_data", m.num(est.Synthetic-s.Training.SynthTokens))))
	}
	start := th.Bold.Render("[ " + c.T("ui.train.start") + " ]")
	if len(s.Training.Queue) >= tb.MaxQueue {
		start = th.Dim.Render("[ " + c.T("msg.queue_full") + " ]")
	}
	t.item(actTrainStart, 0, start)

	t.blank()
	m.section(t, c.T("training.queue", len(s.Training.Queue), tb.MaxQueue))
	if len(s.Training.Queue) == 0 {
		t.text(th.Dim.Render(c.T("training.idle")))
	}
	for i := range s.Training.Queue {
		j := &s.Training.Queue[i]
		eta := c.T("ui.train.waiting")
		if i == 0 {
			eta = c.T("training.eta", m.dur(tv.ETA))
		}
		var bar strings.Builder
		widgets.Track(&bar, j.Progress, 12, th.Magenta, th.Track)
		t.item(actJob, i, th.Text.Render(m.cell(strconv.Itoa(i+1)+". "+m.num(j.Params)+" x "+m.num(j.Tokens), 22))+
			bar.String()+th.Text.Render(" "+m.cell(pct(j.Progress), 6))+th.Dim.Render(m.cell(eta, 18)+c.T("ui.train.job_share", pct(j.ClusterShare))))
	}

	t.blank()
	m.section(t, c.T("training.models", len(s.Training.Models), tb.MaxModels))
	if len(s.Training.Models) == 0 {
		t.text(th.Dim.Render(c.T("ui.model.none")))
	}
	for i := range s.Training.Models {
		md := &s.Training.Models[i]
		t.item(actModel, i, th.Bold.Render(m.cell(md.Name, 8))+th.Gold.Render(m.cell(c.T("ui.model.bench", fixed(md.Benchmark, 1)), 11))+
			th.Text.Render(m.cell(c.T("ui.model.params", m.num(md.Params)), 14))+
			th.Dim.Render(m.cell(c.T("ui.model.age", m.dur(s.Clock-md.ReleasedS)), 14))+th.Text.Render(m.segName(md.Segment)))
	}
	m.archiveRows(t)
}
