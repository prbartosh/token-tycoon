package ui

import (
	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/sim"
)

func (m *Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	switch k {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	if m.away.open {
		m.away.open = false
		m.anim.Counter.Jump(m.away.res.Earned)
		return m, nil
	}
	if m.help {
		switch k {
		case "?", "esc", "enter":
			m.help = false
		}
		return m, nil
	}
	switch k {
	case "?":
		m.help = true
		return m, nil
	case "left":
		m.setTab((m.tab + numTabs - 1) % numTabs)
		return m, nil
	case "right":
		m.setTab((m.tab + 1) % numTabs)
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7":
		m.setTab(int(k[0] - '1'))
		return m, nil
	case "n":
		if m.notation() == 0 {
			m.s.Settings.Notation = "scientific"
		} else {
			m.s.Settings.Notation = "letters"
		}
		m.setStatus(m.cat.T("ui.msg.notation", m.cat.T("ui.notation."+m.notation().String())), false)
		return m, nil
	case "a":
		m.s.Settings.Advisor = !m.s.Settings.Advisor
		m.setStatus(m.cat.T(onOff("ui.msg.advisor", m.s.Settings.Advisor)), false)
		return m, nil
	case "c":
		m.s.Settings.Colorblind = !m.s.Settings.Colorblind
		m.setStatus(m.cat.T(onOff("ui.msg.colorblind", m.s.Settings.Colorblind)), false)
		return m, nil
	case "space":
		return m, m.exec(sim.Boost(m.selBusiness()))
	case "u":
		return m, m.exec(sim.UndoLast())
	}
	if m.tab != tabBusinesses {
		return m, nil
	}
	switch k {
	case "up", "k":
		m.moveSel(-1)
	case "down", "j":
		m.moveSel(1)
	case "home":
		m.selectRow(0)
	case "end":
		m.selectRow(len(m.buildRows()) - 1)
	case "pgup":
		m.moveSel(-max(1, m.listHeight()-1))
	case "pgdown":
		m.moveSel(max(1, m.listHeight()-1))
	case "tab":
		m.qty = (m.qty + 1) % len(qtys)
	case "shift+tab":
		m.qty = (m.qty + len(qtys) - 1) % len(qtys)
	case "enter":
		return m, m.primary()
	case "m":
		if m.sel.kind == rowBusiness {
			return m, m.exec(sim.HireManager(m.sel.idx))
		}
	}
	return m, nil
}

func onOff(prefix string, on bool) string {
	if on {
		return prefix + "_on"
	}
	return prefix + "_off"
}

func (m *Model) setTab(t int) {
	if t < 0 || t >= numTabs {
		return
	}
	m.tab = t
	m.newTab[t] = false
}

func (m *Model) selBusiness() int {
	if m.tab == tabBusinesses && m.sel.kind == rowBusiness {
		return m.sel.idx
	}
	return -1
}

func (m *Model) primary() tea.Cmd {
	switch m.sel.kind {
	case rowBusiness, rowLocked:
		return m.exec(sim.Primary(m.sel.idx, qtys[m.qty]))
	case rowUpgrade:
		return m.exec(sim.BuyUpgrade(m.sel.idx))
	}
	return nil
}

func (m *Model) willStart(i int) bool {
	_, cost := sim.ResolveQty(m.s, m.b, i, qtys[m.qty])
	bz := &m.s.Businesses[i]
	return cost > m.s.Money && bz.Owned > 0 && !bz.Manager && !bz.Running
}

func (m *Model) click(ms tea.Mouse) tea.Cmd {
	if m.away.open {
		m.away.open = false
		return nil
	}
	if m.help {
		m.help = false
		return nil
	}
	if ms.Button != tea.MouseLeft {
		return nil
	}
	if ms.Y == 1 {
		for t, sp := range m.tabSpans() {
			if ms.X >= sp[0] && ms.X < sp[1] {
				m.setTab(t)
				return nil
			}
		}
		return nil
	}
	g := m.geom()
	if m.tab != tabBusinesses || ms.Y < topRows || ms.Y >= topRows+g.bodyH || ms.X < 1 || ms.X > g.listW {
		return nil
	}
	ri := m.scroll + ms.Y - topRows
	rs := m.buildRows()
	if ri < 0 || ri >= len(rs) || !rs[ri].selectable() {
		return nil
	}
	now := m.now()
	double := ri == m.lastClickRow && !m.lastClick.IsZero() && now.Sub(m.lastClick) <= doubleClick
	m.sel = rs[ri]
	m.ensureVisible()
	if double {
		m.lastClick = now.Add(-2 * doubleClick)
		return m.primary()
	}
	m.lastClick, m.lastClickRow = now, ri
	return nil
}

func (m *Model) wheel(ms tea.Mouse) {
	if m.tab != tabBusinesses || m.help || m.away.open {
		return
	}
	switch ms.Button {
	case tea.MouseWheelUp:
		m.moveSel(-1)
	case tea.MouseWheelDown:
		m.moveSel(1)
	}
}
