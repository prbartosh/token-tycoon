package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"token-tycoon/internal/econ"
	"token-tycoon/internal/format"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/theme"
	"token-tycoon/internal/ui/widgets"
)

type lines struct {
	out []string
	w   int
}

func (l *lines) add(s string) { l.out = append(l.out, " "+s) }

func (l *lines) blank() { l.out = append(l.out, "") }

func (l *lines) wrap(s string, st theme.Paint) {
	for _, p := range strings.Split(ansi.Wrap(s, max(1, l.w-2), ""), "\n") {
		l.add(st.Render(p))
	}
}

func (m *Model) label(key string) string {
	return m.th.Dim.Render(widgets.Fit(m.cat.T(key), labelW))
}

func (m *Model) labelText(s string) string { return m.th.Dim.Render(widgets.Fit(s, labelW)) }

func (m *Model) titleLine(name, right string, w int) string {
	th := m.th
	r := ""
	if right != "" {
		r = th.Gold.Render(right)
	}
	nw := max(1, w-2-widgets.Width(right)-1)
	return th.Bold.Render(widgets.Fit(strings.ToUpper(name), nw)) + " " + r
}

func (m *Model) detailLines(w, h int) []string {
	l := &lines{w: w, out: make([]string, 0, h)}
	switch m.sel.kind {
	case rowBusiness:
		m.businessDetail(l, m.sel.idx)
	case rowLocked:
		m.lockedDetail(l, m.sel.idx)
	case rowUpgrade:
		m.upgradeDetail(l, m.sel.idx)
	}
	if len(l.out) > h {
		l.out = l.out[:h]
	}
	return l.out
}

func (m *Model) businessDetail(l *lines, i int) {
	th, d, bz := m.th, &m.s.Derived, &m.s.Businesses[i]
	id := m.b.Businesses[i].ID
	right := ""
	if bz.Owned > 0 {
		right = "x" + strconv.Itoa(bz.Owned)
	}
	l.add(m.titleLine(m.bizName(i), right, l.w))
	l.wrap(m.cat.T(i18n.BusinessDescKey(id)), th.Dim)
	l.blank()
	combo := max(1, m.s.Combo.Mult)
	if bz.Owned > 0 {
		l.add(m.label("ui.detail.income") + th.Text.Render(m.cat.T("ui.detail.per_cycle",
			m.num(d.PerCycle[i]*combo), format.Duration(d.CycleS[i]))))
		l.add(m.labelText("") + th.Rate.Render(m.num(d.PerSec[i]*combo)+"/s"))
		if bz.Manager {
			l.add(m.label("ui.detail.manager") + th.Good.Render(m.cat.T(i18n.ManagerKey(id))))
		} else {
			l.add(m.label("ui.detail.manager") + m.cat.T("ui.detail.hire", m.costText(m.b.ManagerCost(i))))
		}
		m.milestoneLines(l, i)
	}
	l.blank()
	k, cost := sim.ResolveQty(m.s, m.b, i, qtys[m.qty])
	buy := m.cat.T("ui.detail.buy", strconv.Itoa(k))
	l.add(m.labelText(buy) + m.costText(cost))
	if cost > m.s.Money {
		l.add(m.labelText("") + th.Dim.Render(m.cat.T("ui.detail.afford_in", m.eta(cost))))
	}
	m.adviceLine(l, row{kind: rowBusiness, idx: i})
}

func (m *Model) milestoneLines(l *lines, i int) {
	th, ms, d := m.th, &m.b.Milestones, &m.s.Derived
	owned := m.s.Businesses[i].Owned
	next := econ.NextBusinessThreshold(owned, ms.SpeedThresholds, ms.ProfitStart, ms.ProfitEvery)
	eff := m.cat.T("ui.detail.ms_profit", m.num(ms.ProfitMult))
	if slices.Contains(ms.SpeedThresholds, next) {
		eff = m.cat.T("ui.detail.ms_speed", m.num(ms.SpeedMult))
	}
	l.add(m.label("ui.detail.milestone") + th.Text.Render(strconv.Itoa(owned)+"/"+strconv.Itoa(next)+"  ") + th.Magenta.Render(eff))
	var sb strings.Builder
	frac := 0.0
	if next > 0 {
		frac = float64(owned) / float64(next)
	}
	widgets.Track(&sb, frac, max(1, l.w-2-labelW), th.Magenta, th.Track)
	l.add(m.labelText("") + sb.String())
	gNext := econ.NextGlobalThreshold(d.MinOwned, ms.GlobalThresholds, ms.GlobalEvery)
	l.add(m.label("ui.detail.global") + th.Text.Render(m.cat.T("ui.detail.global_val", d.MinOwned, gNext, m.num(ms.GlobalMult))))
}

func (m *Model) adviceLine(l *lines, r row) {
	if !m.advisedRow(r) {
		return
	}
	key := "ui.detail.best_buy"
	switch m.adv.kind {
	case sim.PurchaseManager:
		key = "ui.detail.best_manager"
	case sim.PurchaseUpgrade:
		key = "ui.detail.best_upgrade"
	}
	l.blank()
	l.add(m.th.Star.Render("* " + m.cat.T(key)))
}

func (m *Model) lockedDetail(l *lines, i int) {
	th := m.th
	l.add(m.titleLine(m.bizName(i), "", l.w))
	l.wrap(m.cat.T(i18n.BusinessDescKey(m.b.Businesses[i].ID)), th.Dim)
	l.blank()
	need := m.b.Businesses[i].UnlockLifetime
	l.wrap(m.cat.T("ui.detail.unlock_at", m.money(need)), th.Text)
	l.add(m.label("ui.detail.eta") + th.Text.Render(m.etaLifetime(need)))
}

func (m *Model) upgradeDetail(l *lines, u int) {
	th := m.th
	up := &m.b.Upgrades[u]
	l.add(m.titleLine(m.upgName(u), "", l.w))
	l.wrap(m.cat.T(i18n.UpgradeDescKey(up.ID)), th.Dim)
	l.blank()
	if m.s.Upgrades.Has(u) {
		l.add(m.label("ui.detail.status") + th.Good.Render(m.cat.T("ui.detail.purchased")))
		return
	}
	l.add(m.label("ui.detail.cost") + m.costText(up.Cost))
	if up.Cost > m.s.Money {
		l.add(m.labelText("") + th.Dim.Render(m.cat.T("ui.detail.afford_in", m.eta(up.Cost))))
	}
	m.adviceLine(l, row{kind: rowUpgrade, idx: u})
}
