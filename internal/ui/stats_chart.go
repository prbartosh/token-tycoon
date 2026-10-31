package ui

import (
	"math"
	"strings"

	"github.com/prbartosh/token-tycoon/internal/format"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

const (
	histCap     = 120
	histStartS  = 1.0
	chartH      = 6
	chartAxisW  = 10
	chartMinW   = 16
	chartFloorL = -2
)

type history struct {
	vals  [histCap]float64
	logs  [histCap]float64
	n     int
	every float64
	acc   float64
	accT  float64
	peak  float64
	spanS float64
}

func (h *history) add(earned, dt float64) {
	if h.every == 0 {
		h.every = histStartS
	}
	h.acc += earned
	h.accT += dt
	if h.accT+1e-9 < h.every {
		return
	}
	v := h.acc / h.accT
	h.acc, h.accT = 0, 0
	h.peak = math.Max(h.peak, v)
	if h.n == histCap {
		for i := range histCap / 2 {
			h.vals[i] = (h.vals[2*i] + h.vals[2*i+1]) / 2
		}
		h.n = histCap / 2
		h.every *= 2
	}
	h.vals[h.n] = v
	h.n++
	h.spanS = float64(h.n) * h.every
}

func logOf(v float64) float64 {
	if !(v > 0) {
		return chartFloorL
	}
	return math.Log10(v)
}

func (m *Model) incomeChart(t *tabList) {
	th, c, h := m.th, m.cat, &m.hist
	m.section(t, c.T("ui.sec.income_chart"))
	if h.n < 2 {
		t.text(th.Dim.Render(c.T("ui.stats.chart_empty")))
		return
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := range h.n {
		h.logs[i] = logOf(h.vals[i])
		lo, hi = math.Min(lo, h.logs[i]), math.Max(hi, h.logs[i])
	}
	lo, hi = math.Floor(lo), math.Ceil(hi)
	if hi <= lo {
		hi = lo + 1
	}
	w := max(chartMinW, t.w-4-chartAxisW)
	m.plot.Reset(w, chartH)
	m.plot.Series(h.logs[:h.n], lo, hi, h.n)
	n := m.notation()
	for r := range chartH {
		label := ""
		switch r {
		case 0:
			label = format.Rate(math.Pow(10, hi), n)
		case chartH - 1:
			label = format.Rate(math.Pow(10, lo), n)
		}
		m.sb.Reset()
		m.plot.Row(&m.sb, r)
		t.text(th.Dim.Render(widgets.FitRight(label, chartAxisW-1)+"┤") + th.Rate.Render(m.sb.String()))
	}
	left := "-" + format.Duration(h.spanS)
	right := c.T("ui.stats.chart_now")
	gap := max(1, w-widgets.Width(left)-widgets.Width(right))
	t.text(th.Dim.Render(widgets.FitRight("", chartAxisW) + left + strings.Repeat(" ", gap) + right))
}
