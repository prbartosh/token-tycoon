package ui

import (
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

var agiPortions = [...]float64{0.1, 0.5, 1, 1}

const agiMissing = len(agiPortions) - 1

func (m *Model) agiNeed() float64 {
	v := sim.AGIStatus(m.s, m.b)
	return math.Max(0, v.Cost-v.Invested)
}

func (m *Model) agiAmount(i int) (float64, bool) {
	need := m.agiNeed()
	money := math.Max(0, m.s.Money)
	if i == agiMissing {
		return need, money >= need && need > 0
	}
	x := math.Min(money*agiPortions[i], need)
	return x, x > 0
}

func (m *Model) openAGIInvest() {
	m.modal = modal{kind: modalAGI, sel: agiMissing}
	if _, ok := m.agiAmount(agiMissing); !ok {
		m.modal.sel = 2
	}
}

func (m *Model) agiKey(k string) tea.Cmd {
	md := &m.modal
	if md.confirm {
		switch k {
		case "enter", "y":
			sel := md.sel
			m.modal = modal{}
			if _, can := m.agiAmount(sel); !can {
				return nil
			}
			return m.exec(sim.Command{Kind: sim.CmdAGIInvest, Business: -1, Share: agiPortions[sel]})
		case "esc", "n":
			md.confirm = false
		}
		return nil
	}
	switch k {
	case "esc", "n":
		m.modal = modal{}
		return nil
	case "1", "2", "3", "4":
		md.sel = int(k[0] - '1')
		return nil
	}
	sel, ok := m.pickKey(k, len(agiPortions))
	if !ok {
		return nil
	}
	if _, can := m.agiAmount(sel); can {
		md.confirm = true
	}
	return nil
}

func (m *Model) agiBox() string {
	th, c, b := m.th, m.cat, m.b
	w := min(modalW, m.width-2)
	v := sim.AGIStatus(m.s, b)
	if v.Ended || v.Stage >= len(b.AGIStages) {
		return ""
	}
	cost := b.AGIStages[v.Stage].Cost
	body := []string{"", " " + th.Gold.Render(m.agiName(v.Stage)) + th.Dim.Render("  "+c.T("agi.fx."+b.AGIStages[v.Stage].Effect))}
	var bar strings.Builder
	widgets.Track(&bar, v.Invested/cost, 20, th.Gold, th.Track)
	body = append(body, " "+bar.String()+th.Text.Render(" "+c.T("ui.agi.progress", m.money(v.Invested), m.money(cost))), "")
	for i := range agiPortions {
		x, ok := m.agiAmount(i)
		label := c.T("ui.agi.portion", pct(agiPortions[i]))
		if i == agiMissing {
			label = c.T("ui.agi.missing")
		}
		line := th.Text.Render(m.cell(label, 22))
		switch {
		case ok:
			after := (v.Invested + x) / cost
			line += th.Good.Render(m.cell(m.money(x), 14)) + th.Dim.Render(c.T("ui.agi.after", pct(math.Min(1, after))))
		case i == agiMissing:
			line = th.Dim.Render(m.cell(label, 22)) + th.Bad.Render(c.T("ui.agi.short", m.money(m.agiNeed()-m.s.Money)))
		default:
			line = th.Dim.Render(m.cell(label, 22) + "--")
		}
		body = append(body, m.pickLine(i, line))
	}
	body = append(body, "")
	if x, ok := m.agiAmount(m.modal.sel); ok && m.modal.confirm {
		for _, l := range wrapLines(c.T("ui.agi.confirm", m.money(x), m.agiName(v.Stage)), w-4) {
			body = append(body, " "+th.Gold.Render(l))
		}
		body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.key.confirm"))+"   "+th.KeyText.Render(c.T("ui.key.back")), w-2))
		return m.box(c.T("ui.agi.title"), body, w)
	}
	body = append(body, widgets.Center(th.Key.Render(c.T("ui.agi.keys")), w-2))
	return m.box(c.T("ui.agi.title"), body, w)
}
