package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/sim"
)

func (m *Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}
	if m.pal.open {
		return m, m.paletteKey(msg)
	}
	if k == "q" {
		return m, tea.Quit
	}
	if cmd, ok := m.reaction(k); ok {
		return m, cmd
	}
	if m.away.open {
		m.away.open = false
		m.anim.Counter.Jump(m.away.res.Earned)
		return m, nil
	}
	if m.modal.kind != modalNone {
		return m, m.modalKey(k)
	}
	if m.help {
		m.helpKey(k)
		return m, nil
	}
	if k == "esc" && m.cer.active() {
		m.skipCeremony()
		return m, nil
	}
	switch k {
	case ":", "ctrl+p":
		m.openPalette()
		return m, nil
	case "?":
		m.openHelp()
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
		m.toggleNotation()
		return m, nil
	case "a":
		m.toggleAdvisor()
		return m, nil
	case "c":
		m.toggleColorblind()
		return m, nil
	case "space":
		return m, m.exec(sim.Boost(m.selBusiness()))
	case "u":
		return m, m.exec(sim.UndoLast())
	case "p":
		m.openPrestige()
		return m, nil
	case "P", "shift+p":
		return m, m.openFunding()
	case "o", "r", "R", "h", "H", "g", "G":
		return m, nil
	case "d":
		if g := m.grant(); g.Claimable && g.Amount > 0 {
			return m, m.exec(sim.ClaimGrant(m.today()))
		}
		return m, nil
	}
	if m.tab != tabBusinesses {
		return m, m.tabKey(k)
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

func (m *Model) reaction(k string) (tea.Cmd, bool) {
	if i, left := sim.ActiveIncident(m.s, m.b); i >= 0 && left > 0 && strings.EqualFold(k, m.b.Safety.Key) {
		return m.exec(sim.Respond(m.b.Safety.Key)), true
	}
	if key := m.reactionKey(); key != "" && strings.EqualFold(key, k) {
		return m.exec(sim.React(key)), true
	}
	if m.s.Poach.Active && k == "o" {
		return m.exec(sim.CounterOffer()), true
	}
	return nil, false
}

func (m *Model) today() string { return m.now().Format(time.DateOnly) }

func (m *Model) grant() sim.GrantInfo { return sim.GrantPreview(m.s, m.b, m.today()) }

func (m *Model) reactionKey() string {
	i, left := sim.ActiveEvent(m.s, m.b)
	if i < 0 || left <= 0 || m.b.Event[i].Key == "" {
		return ""
	}
	return m.b.Event[i].Key
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
	if m.pal.open {
		m.closePalette()
		return nil
	}
	if m.away.open {
		m.away.open = false
		return nil
	}
	if m.help {
		m.help = false
		return nil
	}
	if m.modal.kind == modalEnding {
		m.modal = modal{}
		return nil
	}
	if m.cer.active() && m.modal.kind == modalNone {
		m.skipCeremony()
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
	if m.modal.kind != modalNone {
		return nil
	}
	if m.tab != tabBusinesses {
		return m.tabClick(ms, g)
	}
	if ms.Y < topRows || ms.Y >= topRows+g.bodyH || ms.X < 1 || ms.X > g.listW {
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

func (m *Model) tabClick(ms tea.Mouse, g geometry) tea.Cmd {
	if ms.Y < topRows || ms.Y >= topRows+g.bodyH || ms.X < 1 || ms.X > m.mainWidth() {
		return nil
	}
	rs := m.tabRows(m.mainWidth())
	ri := m.tscroll[m.tab] + ms.Y - topRows
	if ri < 0 || ri >= len(rs) || !rs[ri].selectable() {
		return nil
	}
	now := m.now()
	double := ri == m.lastClickRow && !m.lastClick.IsZero() && now.Sub(m.lastClick) <= doubleClick
	m.tsel[m.tab] = rs[ri].key
	if double {
		m.lastClick = now.Add(-2 * doubleClick)
		return m.tabEnter()
	}
	m.lastClick, m.lastClickRow = now, ri
	return nil
}

func (m *Model) wheel(ms tea.Mouse) {
	if m.help {
		switch ms.Button {
		case tea.MouseWheelUp:
			m.helpKey("up")
		case tea.MouseWheelDown:
			m.helpKey("down")
		}
		return
	}
	if m.away.open || m.modal.kind != modalNone || m.pal.open {
		return
	}
	if m.tab != tabBusinesses {
		switch ms.Button {
		case tea.MouseWheelUp:
			m.tabMove(-1)
		case tea.MouseWheelDown:
			m.tabMove(1)
		}
		return
	}
	switch ms.Button {
	case tea.MouseWheelUp:
		m.moveSel(-1)
	case tea.MouseWheelDown:
		m.moveSel(1)
	}
}
