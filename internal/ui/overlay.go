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

const helpLines = 18

func (m *Model) overlays(g geometry) []*lipgloss.Layer {
	var ls []*lipgloss.Layer
	if m.tab == tabBusinesses {
		ls = m.floaters(ls, g)
	}
	ls = m.toasts(ls)
	switch {
	case m.away.open:
		ls = append(ls, m.centered(m.awayBox(), 10))
	case m.modal.kind != modalNone:
		ls = append(ls, m.centered(m.modalBox(), 10))
	case m.help:
		ls = append(ls, m.centered(m.helpBox(), 10))
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
		step := min(theme.FadeSteps-1, int(f.Progress()*theme.FadeSteps))
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
	for i := range items {
		t := &items[i]
		txt := " " + t.Text + " "
		if widgets.Width(txt) > maxW {
			txt = ansi.Truncate(txt, maxW-1, "…") + " "
		}
		tw := widgets.Width(txt)
		x := w - 1 - tw + int(t.Pos*float64(tw+1))
		if x >= w-1 {
			continue
		}
		vis := ansi.Truncate(txt, w-1-x, "")
		ls = append(ls, lipgloss.NewLayer(m.toastStyle(t).Render(vis)).X(x).Y(topRows+i).Z(5))
	}
	return ls
}

func (m *Model) centered(box string, z int) *lipgloss.Layer {
	bw, bh := lipgloss.Width(box), lipgloss.Height(box)
	x := max(0, (m.width-bw)/2)
	y := max(0, (m.height-bh)/2)
	return lipgloss.NewLayer(box).X(x).Y(y).Z(z)
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
		sb.WriteByte('\n')
		sb.WriteString(th.Border.Render("│"))
		sb.WriteString(widgets.Fit(l, inner))
		sb.WriteString(th.Border.Render("│"))
	}
	sb.WriteByte('\n')
	sb.WriteString(th.Border.Render("╰" + strings.Repeat("─", inner) + "╯"))
	return sb.String()
}

func (m *Model) helpBox() string {
	w := min(helpW, m.width-2)
	th := m.th
	body := make([]string, 0, helpLines+2)
	body = append(body, "")
	for i := 1; i <= helpLines; i++ {
		k, desc, _ := strings.Cut(m.cat.T("ui.help."+strconv.Itoa(i)), "|")
		body = append(body, " "+th.Key.Render(widgets.Fit(k, 14))+th.Text.Render(desc))
	}
	body = append(body, "")
	return m.box(m.cat.T("ui.help.title"), body, w)
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
