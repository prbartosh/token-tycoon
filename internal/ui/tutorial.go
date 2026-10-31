package ui

import (
	"strconv"

	"charm.land/lipgloss/v2"

	"token-tycoon/internal/format"
	"token-tycoon/internal/ui/widgets"
)

type anchor uint8

const (
	anchorCenter anchor = iota
	anchorList
	anchorTop
	anchorKeys
)

type tutStep struct {
	id  string
	at  anchor
	arg func(m *Model) []any
}

var tutSteps = [...]tutStep{
	{id: "start", at: anchorList},
	{id: "boost", at: anchorTop},
	{id: "manager", at: anchorList, arg: func(m *Model) []any { return []any{m.money(m.b.ManagerCost(0))} }},
	{id: "tabs", at: anchorTop, arg: func(m *Model) []any { return []any{m.badge()} }},
	{id: "palette", at: anchorCenter},
	{id: "help", at: anchorKeys},
	{id: "save", at: anchorCenter, arg: func(m *Model) []any { return []any{format.Duration(m.b.Engine.AutosaveS)} }},
}

const tutW = 60

type tutorial struct {
	open bool
	step int
}

func (m *Model) openTutorial() {
	m.closePalette()
	m.help = false
	m.tut = tutorial{open: true}
}

func (m *Model) closeTutorial() {
	m.tut.open = false
	m.s.Settings.TutorialDone = true
	m.setStatus(m.cat.T("ui.tut.reopen"), false)
}

func (m *Model) tutKey(k string) {
	switch k {
	case "esc":
		m.closeTutorial()
	case "enter", "space", "right":
		m.tutNext()
	case "left", "backspace":
		m.tut.step = max(0, m.tut.step-1)
	}
}

func (m *Model) tutNext() {
	if m.tut.step >= len(tutSteps)-1 {
		m.closeTutorial()
		return
	}
	m.tut.step++
}

func (m *Model) tutText() string {
	st := &tutSteps[m.tut.step]
	var args []any
	if st.arg != nil {
		args = st.arg(m)
	}
	return m.cat.T("ui.tut."+st.id, args...)
}

func (m *Model) tutBox() string {
	th, c := m.th, m.cat
	w := min(tutW, m.width-2)
	st := &tutSteps[m.tut.step]
	body := []string{" " + th.Gold.Render(c.T("ui.tut.at."+st.id)), ""}
	for _, l := range widgets.Wrap(m.tutText(), w-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	next := c.T("ui.tut.next")
	if m.tut.step == len(tutSteps)-1 {
		next = c.T("ui.tut.finish")
	}
	keys := th.Key.Render(next) + "  " + th.KeyText.Render(c.T("ui.tut.back")) + "  " + th.KeyText.Render(c.T("ui.tut.skip"))
	body = append(body, "", widgets.Center(keys, w-2))
	title := c.T("ui.tut.title") + " " + strconv.Itoa(m.tut.step+1) + "/" + strconv.Itoa(len(tutSteps))
	return m.box(title, body, w)
}

func (m *Model) tutLayer() *lipgloss.Layer {
	box := m.tutBox()
	bw, bh := lipgloss.Width(box), lipgloss.Height(box)
	x, y := max(0, (m.width-bw)/2), max(0, (m.height-bh)/2)
	switch tutSteps[m.tut.step].at {
	case anchorTop:
		y = min(topRows, max(0, m.height-bh))
	case anchorKeys:
		y = max(0, m.height-1-bh)
	case anchorList:
		x = max(0, m.width-bw-1)
	}
	return lipgloss.NewLayer(box).X(x).Y(y).Z(11)
}
