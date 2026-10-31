package ui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"token-tycoon/internal/format"
	"token-tycoon/internal/ui/anim"
	"token-tycoon/internal/ui/theme"
	"token-tycoon/internal/ui/widgets"
)

const helpKeyW = 14

func (m *Model) overlays(g geometry) []*lipgloss.Layer {
	var ls []*lipgloss.Layer
	if m.tab == tabBusinesses {
		ls = m.floaters(ls, g)
	}
	ls = m.toasts(ls)
	switch {
	case m.tut.open:
		ls = append(ls, m.tutLayer())
	case m.pal.open:
		ls = append(ls, m.topCentered(m.paletteBox(), 12))
	case m.away.open:
		ls = append(ls, m.centered(m.awayBox(), 10))
	case m.modal.kind != modalNone:
		ls = append(ls, m.centered(m.modalBox(), 10))
	case m.help:
		ls = append(ls, m.centered(m.helpBox(), 10))
	case m.cer.active():
		ls = append(ls, m.centered(m.ceremonyBox(), 8))
	}
	return ls
}

func (m *Model) floaters(ls []*lipgloss.Layer, g geometry) []*lipgloss.Layer {
	items := m.anim.Float.Items()
	rs := m.rows
	for k := range items {
		f := &items[k]
		if !f.Alive() {
			continue
		}
		ri := -1
		for i, r := range rs {
			if r.kind == rowBusiness && r.idx == f.Slot {
				ri = i
				break
			}
		}
		if ri < 0 {
			continue
		}
		y := topRows + ri - m.scroll - f.Offset()
		if y < topRows || y >= topRows+g.bodyH {
			continue
		}
		step := min(theme.FadeSteps-1, int(f.Fade()*theme.FadeSteps))
		st := m.th.Fade[step]
		if f.Kind == anim.KindCrit {
			st = m.th.GoldFade[step]
		}
		tw := widgets.Width(f.Text)
		x := max(1, 1+g.listW-tw-1)
		txt := f.Text
		if x+tw > 1+g.listW {
			txt = ansi.Truncate(txt, max(0, g.listW-x+1), "")
		}
		ls = append(ls, lipgloss.NewLayer(st.Render(txt)).X(x).Y(y).Z(2))
	}
	return ls
}

const revealS = 0.5

func (m *Model) revealing() bool {
	for _, t := range m.anim.Toast.Items() {
		if t.Age < revealS && t.Kind >= anim.KindRare {
			return true
		}
	}
	return false
}

func (m *Model) toastStyle(t *anim.Toast) theme.Paint {
	k := t.Kind
	reveal := [...]theme.Paint{m.th.ToastInfo, m.th.ToastRare, m.th.ToastEpic, m.th.ToastGold}
	switch k {
	case anim.KindReveal, anim.KindRare, anim.KindEpic, anim.KindLegend:
		if t.Age < revealS {
			return reveal[int(t.Age*16)%len(reveal)]
		}
	}
	switch k {
	case anim.KindRare:
		return m.th.ToastRare
	case anim.KindEpic:
		return m.th.ToastEpic
	case anim.KindLegend:
		return m.th.ToastGold
	case anim.KindMilestone:
		return m.th.ToastMagic
	case anim.KindCrit:
		return m.th.ToastGold
	case anim.KindWarn:
		return m.th.ToastWarn
	}
	return m.th.ToastInfo
}

func (m *Model) toasts(ls []*lipgloss.Layer) []*lipgloss.Layer {
	w := m.width
	maxW := min(toastMaxW, max(24, w/3), w-4)
	items := m.anim.Toast.Items()
	y := topRows
	for i := range items {
		t := &items[i]
		parts := widgets.Wrap(t.Text, max(1, maxW-2))
		tw := 0
		for _, p := range parts {
			tw = max(tw, widgets.Width(p)+2)
		}
		x := w - 1 - tw + int(t.Pos*float64(tw+1))
		top := y
		y += len(parts)
		if x >= w-1 {
			continue
		}
		st := m.toastStyle(t)
		for k, p := range parts {
			if top+k >= m.height-bottomRows {
				break
			}
			txt := widgets.Fit(" "+p, tw)
			vis := ansi.Truncate(txt, w-1-x, "")
			ls = append(ls, lipgloss.NewLayer(st.Render(vis)).X(x).Y(top+k).Z(5))
		}
	}
	return ls
}

func (m *Model) centered(box string, z int) *lipgloss.Layer {
	bw, bh := lipgloss.Width(box), lipgloss.Height(box)
	x := max(0, (m.width-bw)/2)
	y := max(0, (m.height-bh)/2)
	return lipgloss.NewLayer(box).X(x).Y(y).Z(z)
}

func (m *Model) topCentered(box string, z int) *lipgloss.Layer {
	bw := lipgloss.Width(box)
	return lipgloss.NewLayer(box).X(max(0, (m.width-bw)/2)).Y(min(topRows-1, max(0, m.height-lipgloss.Height(box)))).Z(z)
}

func (m *Model) box(title string, body []string, w int) string {
	th := m.th
	inner := w - 2
	var sb strings.Builder
	t := " " + title + " "
	sb.WriteString(th.Border.Render("╭─"))
	sb.WriteString(th.Title.Render(t))
	sb.WriteString(th.Border.Render(strings.Repeat("─", max(0, inner-1-widgets.Width(t))) + "╮"))
	for _, l := range body {
		for _, p := range widgets.WrapHang(l, inner, 1) {
			sb.WriteByte('\n')
			sb.WriteString(th.Border.Render("│"))
			sb.WriteString(widgets.Fit(p, inner))
			sb.WriteString(th.Border.Render("│"))
		}
	}
	sb.WriteByte('\n')
	sb.WriteString(th.Border.Render("╰" + strings.Repeat("─", inner) + "╯"))
	return sb.String()
}

var helpTabIDs = [numTabs]string{"businesses", "models", "market", "infra", "research", "rules", "stats"}

func (m *Model) helpEntries(dst []string) []string {
	th := m.th
	descW := max(8, min(helpW, m.width-2)-3-helpKeyW)
	pad := strings.Repeat(" ", 1+helpKeyW)
	add := func(key string) {
		k, desc, _ := strings.Cut(m.cat.T(key), "|")
		for i, p := range widgets.Wrap(desc, descW) {
			if i == 0 {
				dst = append(dst, " "+th.Key.Render(m.cell(k, helpKeyW))+th.Text.Render(p))
				continue
			}
			dst = append(dst, pad+th.Text.Render(p))
		}
	}
	pre := "ui.help." + helpTabIDs[m.tab] + "."
	if m.cat.Has(pre + "1") {
		dst = append(dst, " "+th.Gold.Render(strings.ToUpper(m.cat.T(tabKeys[m.tab]))))
		for i := 1; m.cat.Has(pre + strconv.Itoa(i)); i++ {
			add(pre + strconv.Itoa(i))
		}
		dst = append(dst, "", " "+th.Gold.Render(m.cat.T("ui.help.general")))
	}
	for i := 1; m.cat.Has("ui.help." + strconv.Itoa(i)); i++ {
		add("ui.help." + strconv.Itoa(i))
	}
	return dst
}

func (m *Model) helpRows() int { return max(3, m.height-6) }

func (m *Model) helpKey(k string) {
	n := len(m.helpEntries(m.lines[:0]))
	maxTop := max(0, n-m.helpRows())
	switch k {
	case "t":
		m.openTutorial()
		return
	case "?", "esc", "enter":
		m.help = false
	case "up", "k":
		m.helpTop--
	case "down", "j":
		m.helpTop++
	case "pgup":
		m.helpTop -= m.helpRows()
	case "pgdown", "space":
		m.helpTop += m.helpRows()
	case "home":
		m.helpTop = 0
	case "end":
		m.helpTop = maxTop
	}
	m.helpTop = max(0, min(m.helpTop, maxTop))
}

func (m *Model) helpBox() string {
	w := min(helpW, m.width-2)
	all := m.helpEntries(make([]string, 0, 40))
	rows := m.helpRows()
	top := max(0, min(m.helpTop, len(all)-rows))
	body := all[top:min(len(all), top+rows)]
	title := m.cat.T("ui.help.title")
	if len(all) > rows {
		body = append(body[:len(body):len(body)], widgets.Center(m.th.KeyText.Render(m.cat.T("ui.help.scroll", top+1, min(len(all), top+rows), len(all))), w-2))
	}
	return m.box(title, body, w)
}

func (m *Model) awayBox() string {
	w := min(awayW, m.width-2)
	th, r := m.th, &m.away.res
	body := []string{
		"",
		" " + th.Text.Render(m.cat.T("ui.away.elapsed", format.Duration(r.Elapsed))),
	}
	if r.Simulated+1 < r.Elapsed {
		body = append(body, " "+th.Dim.Render(m.cat.T("ui.away.capped", format.Duration(r.Simulated))))
	}
	body = append(body,
		"",
		widgets.Center(th.Money.Render(m.money(m.anim.Counter.Value)), w-2),
		"",
		" "+th.Dim.Render(m.cat.T("ui.away.efficiency", strconv.Itoa(int(r.Efficiency*100+0.5)))),
		"",
		widgets.Center(th.KeyText.Render(m.cat.T("ui.key.continue")), w-2),
	)
	return m.box(m.cat.T("ui.away.title"), body, w)
}
