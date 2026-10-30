package ui

import (
	"math"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/format"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/anim"
	"token-tycoon/internal/ui/theme"
	"token-tycoon/internal/ui/widgets"
)

const (
	labelW     = 11
	valueW     = 9
	ownedW     = 5
	nameMax    = 20
	nameMin    = 8
	minBarW    = 4
	toastMaxW  = 48
	helpW      = 76
	awayW      = 50
	sparkMin   = 8
	sparkMax   = 30
	costColW   = 12
	tabGap     = 2
	debugLabel = "fps "
)

type geometry struct {
	panels               int
	listW, detailW, logW int
	bodyH                int
}

func (m *Model) geom() geometry {
	w, ui := m.width, &m.b.UI
	g := geometry{bodyH: m.height - topRows - bottomRows}
	switch {
	case w >= ui.ThreePanelWidth:
		g.panels = 3
		g.logW = max(26, min(40, w/4))
		g.detailW = max(32, min(40, w*3/10))
		g.listW = w - 4 - g.logW - g.detailW
	case w >= ui.TwoPanelWidth:
		g.panels = 2
		g.detailW = max(30, min(40, w*2/5))
		g.listW = w - 3 - g.detailW
	default:
		g.panels = 1
		g.listW = w - 2
	}
	return g
}

func (m *Model) minWidth() int { return max(40, m.b.UI.MinWidth) }

func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.ReportFocus = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = m.cat.T("game.title")
	if m.th.BG != nil {
		v.BackgroundColor = m.th.BG
	}
	return v
}

func (m *Model) render() string {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return ""
	}
	if w < m.minWidth() || h < minHeight {
		return m.tooSmall(w, h)
	}
	g := m.geom()
	lines := m.lines[:0]
	lines = append(lines, m.header(w), m.tabBar(w))
	lines = m.body(lines, g)
	lines = append(lines, m.newsLine(w), m.keyBar(w))
	m.lines = lines
	base := strings.Join(lines, "\n")
	layers := m.overlays(g)
	if len(layers) == 0 {
		return base
	}
	if m.canvas == nil {
		m.canvas = lipgloss.NewCanvas(w, h)
	} else if m.canvas.Width() != w || m.canvas.Height() != h {
		m.canvas.Resize(w, h)
	}
	m.canvas.Clear()
	all := make([]*lipgloss.Layer, 0, len(layers)+1)
	all = append(all, lipgloss.NewLayer(base))
	all = append(all, layers...)
	m.canvas.Compose(lipgloss.NewCompositor(all...))
	return m.canvas.Render()
}

func (m *Model) tooSmall(w, h int) string {
	msg := m.cat.T("ui.too_small", w, h, m.minWidth(), minHeight)
	var sb strings.Builder
	top := max(0, h/2-1)
	for range top {
		sb.WriteByte('\n')
	}
	for i, l := range strings.Split(ansi.Wrap(msg, max(1, w), ""), "\n") {
		if top+i >= h {
			break
		}
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(m.th.Text.Render(widgets.Center(l, w)))
	}
	return sb.String()
}

func (m *Model) versionName() string {
	if len(m.b.Prestige.Layer1.Versions) == 0 {
		return ""
	}
	return sim.VersionName(m.b, m.s.Prestige.VersionIndex)
}

func (m *Model) header(w int) string {
	th, n := m.th, m.notation()
	var left strings.Builder
	left.WriteByte(' ')
	left.WriteString(th.Title.Render(m.cat.T("game.title")))
	left.WriteByte(' ')
	left.WriteString(th.Params.Render(m.versionName()))
	left.WriteString("   ")
	left.WriteString(th.Money.Render(m.money(m.anim.Money.Value)))
	left.WriteString("  ")
	left.WriteString(th.Rate.Render(format.Rate(m.rate(), n)))
	if w >= m.b.UI.TwoPanelWidth {
		sw := max(sparkMin, min(sparkMax, w/6))
		var sp strings.Builder
		widgets.Sparkline(&sp, m.spark[sparkLen-m.sparkN:], sw)
		left.WriteByte(' ')
		left.WriteString(th.Rate.Render(sp.String()))
	}
	if c := m.s.Combo.Mult; c > 1 {
		st := th.Magenta
		if m.anim.Pulse > 0 {
			st = th.FlashMilestone
		}
		left.WriteString("  ")
		left.WriteString(st.Render(m.cat.T("ui.header.combo", format.Number(c, format.Letters))))
	}
	var right strings.Builder
	d := &m.s.Derived
	if m.s.IsUnlocked(sim.UnlockInfra) {
		right.WriteString(th.Text.Render(m.cat.T("ui.header.pwr", m.usageText(d.EnergyNeed, d.EnergyAvail))))
		right.WriteString("  ")
		right.WriteString(th.Text.Render(m.cat.T("ui.header.gpu", m.usageText(d.ComputeNeed, d.ComputeCap))))
		right.WriteString("  ")
	}
	if m.s.IsUnlocked(sim.UnlockMarket) {
		right.WriteString(th.Magenta.Render(m.cat.T("ui.header.hype", fixed(1+d.Market.Hype, 1)+"x")))
		right.WriteString("  ")
	}
	if m.s.IsUnlocked(sim.UnlockLayer2) {
		right.WriteString(th.Capital.Render(m.cat.T("ui.header.vc", m.num(m.s.Prestige.Capital))))
		right.WriteString("  ")
	}
	if m.debug {
		right.WriteString(th.Dim.Render(debugLabel + strconv.FormatFloat(m.fps, 'f', 1, 64)))
		right.WriteString(" ")
	}
	l, r := left.String(), right.String()
	gap := w - widgets.Width(l) - widgets.Width(r)
	if gap < 1 {
		return widgets.Fit(l, w)
	}
	return l + strings.Repeat(" ", gap) + r
}

func (m *Model) tabLabel(t int, compact bool) string {
	lbl := strconv.Itoa(t+1) + " " + m.cat.T(tabKeys[t])
	if compact && t != m.tab {
		lbl = strconv.Itoa(t + 1)
	}
	if t == m.tab {
		return "[" + lbl + "]"
	}
	return lbl
}

func (m *Model) tabCompact() bool {
	full := m.spans(false)
	return full[numTabs-1][1] > m.width
}

func (m *Model) tabSpans() [numTabs][2]int { return m.spans(m.tabCompact()) }

func (m *Model) spans(compact bool) [numTabs][2]int {
	var sp [numTabs][2]int
	x := 1
	for t := range numTabs {
		wl := widgets.Width(m.tabLabel(t, compact))
		if m.newTab[t] {
			wl += 1 + widgets.Width(m.badge())
		}
		sp[t] = [2]int{x, x + wl}
		x += wl + tabGap
	}
	return sp
}

func (m *Model) badge() string { return "•" + m.cat.T("tab.new") }

func (m *Model) tabBar(w int) string {
	th := m.th
	compact := m.tabCompact()
	var sb strings.Builder
	sb.WriteByte(' ')
	for t := range numTabs {
		if t > 0 {
			sb.WriteString(strings.Repeat(" ", tabGap))
		}
		st := th.TabIdle
		switch {
		case t == m.tab:
			st = th.TabActive
		case !m.unlocked(tabUnlock[t]):
			st = th.TabLocked
		}
		sb.WriteString(st.Render(m.tabLabel(t, compact)))
		if m.newTab[t] {
			sb.WriteByte(' ')
			sb.WriteString(th.Badge.Render(m.badge()))
		}
	}
	return widgets.Fit(sb.String(), w)
}

func (m *Model) border(left, mid, right string, g geometry, titles [3]string) string {
	th := m.th
	var sb strings.Builder
	sb.WriteString(left)
	seg := func(w int, title string) {
		if title == "" || w < widgets.Width(title)+4 {
			sb.WriteString(strings.Repeat("─", max(0, w)))
			return
		}
		sb.WriteString("─ ")
		sb.WriteString(title)
		sb.WriteByte(' ')
		sb.WriteString(strings.Repeat("─", w-3-widgets.Width(title)))
	}
	widths := m.panelWidths(g)
	for i, pw := range widths {
		if pw == 0 {
			continue
		}
		if i > 0 {
			sb.WriteString(mid)
		}
		seg(pw, titles[i])
	}
	sb.WriteString(right)
	return th.Border.Render(sb.String())
}

func (m *Model) panelWidths(g geometry) [3]int {
	if m.tab != tabBusinesses {
		main := g.listW
		if g.panels >= 2 {
			main += 1 + g.detailW
		}
		return [3]int{main, 0, g.logW}
	}
	return [3]int{g.listW, g.detailW, g.logW}
}

func (m *Model) body(lines []string, g geometry) []string {
	th := m.th
	titles := [3]string{m.cat.T(tabKeys[m.tab]), "", ""}
	if g.panels == 3 {
		titles[2] = m.cat.T("ui.log.title")
	}
	if m.tab == tabBusinesses && g.panels >= 2 {
		titles[1] = m.cat.T("ui.detail.title")
	}
	lines = append(lines, m.border("┌", "┬", "┐", g, titles))
	widths := m.panelWidths(g)
	var cols [3][]string
	if m.tab == tabBusinesses {
		cols[0] = m.listLines(widths[0], g.bodyH)
		if widths[1] > 0 {
			cols[1] = m.detailLines(widths[1], g.bodyH)
		}
	} else {
		cols[0] = m.tabLines(widths[0], g.bodyH)
	}
	if widths[2] > 0 {
		cols[2] = m.logLines(widths[2], g.bodyH)
	}
	bar := th.Border.Render("│")
	for y := range g.bodyH {
		var sb strings.Builder
		sb.WriteString(bar)
		for i, pw := range widths {
			if pw == 0 {
				continue
			}
			if i > 0 {
				sb.WriteString(bar)
			}
			cell := ""
			if y < len(cols[i]) {
				cell = cols[i][y]
			}
			sb.WriteString(widgets.Fit(cell, pw))
		}
		sb.WriteString(bar)
		lines = append(lines, sb.String())
	}
	return append(lines, m.border("├", "┴", "┤", g, [3]string{}))
}

func (m *Model) nameWidth(w int) int {
	return max(nameMin, min(nameMax, w-2-1-ownedW-1-minBarW-1-valueW))
}

func (m *Model) listLines(w, h int) []string {
	rs := m.buildRows()
	out := make([]string, 0, h)
	si := m.selIndex(rs)
	for i := m.scroll; i < len(rs) && len(out) < h; i++ {
		out = append(out, m.rowLine(rs[i], i == si, w))
	}
	return out
}

func (m *Model) flashStyle(slot int) (theme.Paint, bool) {
	k, frac := m.anim.Flash.At(slot)
	th := m.th
	switch k {
	case anim.KindPurchase:
		return th.FlashPurchase, true
	case anim.KindCrit:
		return th.FlashCrit, true
	case anim.KindWarn:
		return th.FlashWarn, true
	case anim.KindMilestone:
		return th.FlashMilestone, int(frac*6)%2 == 0
	}
	return theme.Paint{}, false
}

func (m *Model) prefix(r row, selected bool) string {
	var sb strings.Builder
	if selected {
		sb.WriteString(m.th.Marker.Render(">"))
	} else {
		sb.WriteByte(' ')
	}
	if m.advisedRow(r) {
		sb.WriteString(m.th.Star.Render("*"))
	} else {
		sb.WriteByte(' ')
	}
	return sb.String()
}

func (m *Model) costText(cost float64) string {
	if math.IsInf(cost, 0) || math.IsNaN(cost) {
		return m.th.Dim.Render("--")
	}
	s := m.money(cost)
	ok := cost <= m.s.Money
	if m.s.Settings.Colorblind {
		if ok {
			s = "+" + s
		} else {
			s = "-" + s
		}
	}
	if ok {
		return m.th.Good.Render(s)
	}
	return m.th.Bad.Render(s)
}

func (m *Model) eta(cost float64) string {
	if cost <= m.s.Money {
		return m.cat.T("ui.now")
	}
	r := m.rate()
	if !(r > 0) {
		return "--"
	}
	return format.Duration((cost - m.s.Money) / r)
}

func (m *Model) rowLine(r row, selected bool, w int) string {
	th := m.th
	switch r.kind {
	case rowHeader:
		t := " " + m.cat.T("ui.biz.upgrades") + " "
		return th.Dim.Render("──" + t + strings.Repeat("─", max(0, w-2-widgets.Width(t))))
	case rowUpgrade:
		return m.upgradeLine(r, selected, w)
	}
	i := r.idx
	nameW := m.nameWidth(w)
	barW := max(0, w-2-nameW-1-ownedW-1-1-valueW)
	nameSt := th.Text
	if selected {
		nameSt = th.Selected
	}
	bz := &m.s.Businesses[i]
	if r.kind == rowLocked || bz.Owned == 0 {
		nameSt = th.Dim
	}
	fs, flashing := m.flashStyle(r.slot())
	if flashing {
		nameSt = fs
	}
	var sb strings.Builder
	sb.WriteString(m.prefix(r, selected))
	sb.WriteString(nameSt.Render(widgets.Fit(m.bizName(i), nameW)))
	sb.WriteByte(' ')
	switch {
	case r.kind == rowLocked:
		sb.WriteString(th.Dim.Render(widgets.Fit("--", ownedW)))
		sb.WriteByte(' ')
		need := m.b.Businesses[i].UnlockLifetime
		txt := m.cat.T("ui.biz.locked", m.money(need), m.etaLifetime(need))
		sb.WriteString(th.Dim.Render(widgets.Fit(txt, barW+1+valueW)))
	case bz.Owned == 0:
		sb.WriteString(th.Dim.Render(widgets.Fit("--", ownedW)))
		sb.WriteByte(' ')
		_, cost := sim.ResolveQty(m.s, m.b, i, qtys[m.qty])
		c := m.costText(cost)
		e := th.Dim.Render(" " + m.eta(cost))
		sb.WriteString(widgets.Fit(c+e, barW+1+valueW))
	default:
		ownSt := th.Gold
		if flashing {
			ownSt = fs
		}
		sb.WriteString(ownSt.Render(widgets.Fit("x"+strconv.Itoa(bz.Owned), ownedW)))
		sb.WriteByte(' ')
		sb.WriteString(m.cycleBar(i, barW))
		sb.WriteByte(' ')
		sb.WriteString(m.valueCell(i))
	}
	return sb.String()
}

func (m *Model) cycleBar(i, w int) string {
	if w <= 0 {
		return ""
	}
	tier := min(i, theme.Tiers-1)
	var sb strings.Builder
	if m.s.Derived.Continuous[i] && (m.s.Businesses[i].Manager || m.s.Businesses[i].Running) {
		widgets.Flow(&sb, w, int(m.s.Clock*10), m.th.Flow[tier][:])
		return sb.String()
	}
	frac := m.s.Businesses[i].Progress
	if !m.s.Businesses[i].Manager && !m.s.Businesses[i].Running {
		frac = 0
	}
	widgets.Bar(&sb, frac, w)
	return m.th.Tier[tier].Render(sb.String())
}

func (m *Model) valueCell(i int) string {
	d := &m.s.Derived
	combo := max(1, m.s.Combo.Mult)
	bz := &m.s.Businesses[i]
	if bz.Manager || d.Continuous[i] {
		return m.th.Rate.Render(widgets.FitRight(m.num(d.PerSec[i]*combo)+"/s", valueW))
	}
	st := m.th.Text
	if !bz.Running {
		st = m.th.Dim
	}
	return st.Render(widgets.FitRight(m.num(d.PerCycle[i]*combo), valueW))
}

func (m *Model) upgradeLine(r row, selected bool, w int) string {
	th := m.th
	u := r.idx
	owned := m.s.Upgrades.Has(u)
	nameSt := th.Text
	if selected {
		nameSt = th.Selected
	}
	cost := m.b.Upgrades[u].Cost
	if owned || cost > m.s.Money {
		nameSt = th.Dim
	}
	if fs, ok := m.flashStyle(r.slot()); ok {
		nameSt = fs
	}
	nameW := max(4, w-2-1-costColW)
	var sb strings.Builder
	sb.WriteString(m.prefix(r, selected))
	sb.WriteString(nameSt.Render(widgets.Fit(m.upgName(u), nameW)))
	sb.WriteByte(' ')
	if owned {
		sb.WriteString(th.Dim.Render(widgets.FitRight(m.cat.T("ui.biz.owned"), costColW)))
	} else {
		sb.WriteString(widgets.FitRight(m.costText(cost), costColW))
	}
	return sb.String()
}

func (m *Model) etaLifetime(need float64) string {
	have := m.s.Lifetime.Total
	if have >= need {
		return m.cat.T("ui.now")
	}
	r := m.rate()
	if !(r > 0) {
		return "--"
	}
	return format.Duration((need - have) / r)
}

func (m *Model) qtyLabel() string {
	if qtys[m.qty] == sim.QtyMax {
		return m.cat.T("qty.max")
	}
	return "x" + strconv.Itoa(qtys[m.qty])
}

func (m *Model) logLines(w, h int) []string {
	out := make([]string, 0, h)
	for i := len(m.log) - 1; i >= 0 && len(out) < h; i-- {
		st := m.th.Text
		if len(out) >= 3 {
			st = m.th.Dim
		}
		for _, l := range strings.Split(ansi.Wrap(m.log[i], max(1, w-1), ""), "\n") {
			if len(out) >= h {
				break
			}
			out = append(out, st.Render(" "+l))
		}
	}
	return out
}

func (m *Model) newsLine(w int) string {
	th := m.th
	txt, warn := m.newsText()
	st := th.Text
	if warn {
		st = th.Bad
		if m.s.Settings.Colorblind {
			txt = "! " + txt
		}
	}
	inner := " " + th.Gold.Render(m.cat.T("ui.news.label")) + "  " + st.Render(txt)
	bar := th.Border.Render("│")
	return bar + widgets.Fit(inner, w-2) + bar
}

func (m *Model) newsText() (string, bool) {
	ev, left := sim.ActiveEvent(m.s, m.b)
	if ev >= 0 && m.b.Event[ev].Effect == balance.EventReaction {
		return m.eventBanner(ev, left), true
	}
	if txt, ok := m.incidentBanner(); ok {
		return txt, true
	}
	if m.s.Poach.Active {
		return m.cat.T("ui.poach.banner", m.staffName(m.s.Poach.Index), m.dur(sim.PoachLeft(m.s))), true
	}
	if m.status != "" && m.now().Sub(m.statusAt) < statusFor {
		return m.status, m.statusWarn
	}
	if ev >= 0 {
		return m.eventBanner(ev, left), false
	}
	if h := m.hint(); h != "" {
		return h, false
	}
	if g := m.grant(); g.Claimable && g.Amount > 0 {
		return m.cat.T("ui.hint.grant", g.Day, m.money(g.Amount)), false
	}
	return m.headline(), false
}

func (m *Model) eventBanner(ev int, left float64) string {
	s := ">> " + m.eventName(ev) + ": " + m.eventDesc(ev, left)
	if m.b.Event[ev].Effect != balance.EventReaction {
		s += "  " + m.cat.T("ui.event.left", m.dur(left))
	}
	return s
}

func (m *Model) hint() string {
	for i := range sim.N {
		if m.s.Businesses[i].Manager {
			return ""
		}
	}
	if m.s.Money >= m.b.ManagerCost(0) && m.s.Businesses[0].Owned > 0 {
		return m.cat.T("ui.hint.manager")
	}
	return m.cat.T("ui.hint.start")
}

func (m *Model) keyItems() []string {
	c := m.cat
	items := make([]string, 0, 12)
	switch {
	case m.away.open:
		return append(items, c.T("ui.key.continue"), c.T("key.quit"))
	case m.help:
		return append(items, c.T("ui.key.close"), c.T("key.quit"))
	case m.modal.kind == modalRelease || m.modal.kind == modalEnding || m.modal.kind == modalFunded:
		return append(items, c.T("ui.key.continue"), c.T("key.quit"))
	case m.modal.kind != modalNone:
		return append(items, c.T("ui.key.confirm"), c.T("ui.key.cancel"), c.T("key.quit"))
	}
	if key := m.reactionKey(); key != "" {
		items = append(items, c.T("ui.key.react", strings.ToUpper(key)))
	}
	if m.s.Poach.Active {
		items = append(items, c.T("ui.key.counter"))
	}
	if i, left := sim.ActiveIncident(m.s, m.b); i >= 0 && left > 0 {
		items = append(items, c.T("ui.key.respond", strings.ToUpper(m.b.Safety.Key)))
	}
	if m.tab == tabBusinesses {
		switch m.sel.kind {
		case rowBusiness:
			i := m.sel.idx
			if m.willStart(i) {
				items = append(items, c.T("ui.key.start"))
			} else {
				_, cost := sim.ResolveQty(m.s, m.b, i, qtys[m.qty])
				items = append(items, c.T("ui.key.buy", m.qtyLabel(), m.money(cost)))
			}
			items = append(items, c.T("key.qty", m.qtyLabel()))
			if bz := &m.s.Businesses[i]; bz.Owned > 0 && !bz.Manager {
				items = append(items, c.T("ui.key.manager", m.money(m.b.ManagerCost(i))))
			}
		case rowUpgrade:
			if !m.s.Upgrades.Has(m.sel.idx) {
				items = append(items, c.T("ui.key.buy_upgrade", m.money(m.b.Upgrades[m.sel.idx].Cost)))
			}
		}
	}
	if m.tab != tabBusinesses {
		items = m.tabKeyItems(items)
	}
	if g := m.grant(); g.Claimable && g.Amount > 0 {
		items = append(items, c.T("ui.key.grant", m.money(g.Amount)))
	}
	items = append(items, c.T("key.boost"))
	if left := m.undoLeft(); left > 0 {
		items = append(items, c.T("ui.key.undo", format.Duration(left)))
	}
	if m.s.IsUnlocked(sim.UnlockPrestige1) {
		items = append(items, c.T("key.prestige"))
	}
	if m.s.IsUnlocked(sim.UnlockLayer2) {
		items = append(items, c.T("ui.key.funding"))
	}
	items = append(items, c.T("key.select"))
	return append(items, c.T("key.tabs"), c.T("key.help"), c.T("key.quit"))
}

func (m *Model) keyBar(w int) string {
	th := m.th
	items := m.keyItems()
	tail := 2
	if len(items) < tail {
		tail = len(items)
	}
	widthOf := func(its []string) int {
		n := 0
		for i, it := range its {
			if i > 0 {
				n += 2
			}
			n += widgets.Width(it)
		}
		return n
	}
	avail := w - 4
	keep := items[len(items)-tail:]
	head := items[:len(items)-tail]
	for len(head) > 0 && widthOf(head)+2+widthOf(keep) > avail {
		head = head[:len(head)-1]
	}
	shown := append(head[:len(head):len(head)], keep...)
	var sb strings.Builder
	used := 0
	for i, it := range shown {
		sep := 0
		if i > 0 {
			sep = 2
		}
		if used+sep+widgets.Width(it) > avail {
			break
		}
		if i > 0 {
			sb.WriteString("  ")
		}
		k, rest, _ := strings.Cut(it, " ")
		sb.WriteString(th.Key.Render(k))
		if rest != "" {
			sb.WriteString(th.KeyText.Render(" " + rest))
		}
		used += sep + widgets.Width(it)
	}
	fill := max(0, avail-used)
	return th.Border.Render("└ ") + sb.String() + th.Border.Render(" "+strings.Repeat("─", fill)+"┘")
}
