package ui

import (
	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

func (m *Model) openChoice() {
	if _, ok := sim.PendingRelease(m.s); !ok || m.modal.kind != modalNone {
		return
	}
	m.modal = modal{kind: modalChoose, sel: max(0, sim.WeakestModel(m.s))}
}

func (m *Model) maybeChoose() {
	if m.later || m.away.open || m.help || m.pal.open {
		return
	}
	m.openChoice()
}

func (m *Model) pendingName() string {
	if md, ok := sim.PendingRelease(m.s); ok {
		return md.Name
	}
	return ""
}

func (m *Model) archivedName(i int) string {
	if i < 0 || i >= len(m.s.Training.Archive) {
		return ""
	}
	return m.s.Training.Archive[i].Name
}

func (m *Model) choiceRows() int { return len(m.s.Training.Models) + 1 }

func (m *Model) pickKey(k string, rows int) (int, bool) {
	md := &m.modal
	switch k {
	case "up", "k":
		md.sel = (md.sel + rows - 1) % rows
	case "down", "j":
		md.sel = (md.sel + 1) % rows
	case "enter":
		return md.sel, true
	}
	return 0, false
}

func (m *Model) choiceKey(k string) tea.Cmd {
	if _, ok := sim.PendingRelease(m.s); !ok {
		m.modal = modal{}
		return nil
	}
	if k == "esc" {
		m.modal = modal{}
		m.later = true
		return nil
	}
	sel, ok := m.pickKey(k, m.choiceRows())
	if !ok {
		return nil
	}
	m.modal = modal{}
	if sel >= len(m.s.Training.Models) {
		sel = -1
	}
	cmd := m.exec(sim.ResolveRelease(sel))
	if _, more := sim.PendingRelease(m.s); more {
		m.openChoice()
	}
	return cmd
}

func (m *Model) modelLine(md *sim.Model, w int) string {
	th, c := m.th, m.cat
	return th.Bold.Render(m.cell(md.Name, 8)) + th.Gold.Render(m.cell(c.T("ui.model.bench", fixed(md.Benchmark, 1)), 11)) +
		th.Text.Render(m.cell(c.T("ui.model.params", m.num(md.Params)), 14)) +
		th.Dim.Render(widgets.Fit(m.segName(md.Segment), max(0, w-33)))
}

func (m *Model) pickLine(i int, text string) string {
	if i == m.modal.sel {
		return m.th.Marker.Render(" > ") + text
	}
	return "   " + text
}

func (m *Model) choiceBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	nw, ok := sim.PendingRelease(m.s)
	if !ok {
		return ""
	}
	body := []string{""}
	for _, l := range wrapLines(c.T("ui.choice.body", nw.Name, fixed(nw.Benchmark, 1), len(m.s.Training.Models)), w-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	body = append(body, "")
	for i := range m.s.Training.Models {
		body = append(body, m.pickLine(i, m.modelLine(&m.s.Training.Models[i], w-6)))
	}
	body = append(body, m.pickLine(len(m.s.Training.Models), m.modelLine(&nw, w-6-12)+th.Magenta.Render(c.T("ui.choice.new"))))
	if n := len(m.s.Training.Pending); n > 1 {
		body = append(body, "", " "+th.Dim.Render(c.T("ui.choice.more", n-1)))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.choice.keys")), w-2))
	return m.box(c.T("ui.choice.title"), body, w)
}

func (m *Model) swapKey(k string) tea.Cmd {
	if k == "esc" {
		m.modal = modal{}
		return nil
	}
	sel, ok := m.pickKey(k, max(1, len(m.s.Training.Models)))
	if !ok {
		return nil
	}
	i := m.modal.idx
	m.modal = modal{}
	return m.exec(sim.SwapArchived(i, sel))
}

func (m *Model) swapBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	i := m.modal.idx
	if i < 0 || i >= len(m.s.Training.Archive) {
		return ""
	}
	in := &m.s.Training.Archive[i]
	body := []string{""}
	for _, l := range wrapLines(c.T("ui.swap.body", in.Name, fixed(in.Benchmark, 1)), w-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	body = append(body, "")
	for k := range m.s.Training.Models {
		body = append(body, m.pickLine(k, m.modelLine(&m.s.Training.Models[k], w-6)))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.swap.keys")), w-2))
	return m.box(c.T("ui.swap.title"), body, w)
}

func (m *Model) deleteArchivedBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	name, bench := "", 0.0
	if i := m.modal.idx; i < len(m.s.Training.Archive) {
		name, bench = m.s.Training.Archive[i].Name, m.s.Training.Archive[i].Benchmark
	}
	body := []string{""}
	for _, l := range wrapLines(c.T("ui.archive.delete_body", name, fixed(bench, 1)), w-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.key.confirm"))+"   "+th.KeyText.Render(c.T("ui.key.cancel")), w-2))
	return m.box(c.T("ui.archive.delete_title"), body, w)
}

func (m *Model) archiveEnter(i int) tea.Cmd {
	if i < 0 || i >= len(m.s.Training.Archive) {
		return nil
	}
	if len(m.s.Training.Models) < m.s.Derived.Train.MaxModels {
		return m.exec(sim.SwapArchived(i, -1))
	}
	m.modal = modal{kind: modalSwap, idx: i, sel: max(0, sim.WeakestModel(m.s))}
	return nil
}

func (m *Model) archiveRows(t *tabList) {
	th, c, s := m.th, m.cat, m.s
	if nw, ok := sim.PendingRelease(s); ok {
		t.blank()
		m.section(t, c.T("ui.sec.pending", len(s.Training.Pending)))
		t.item(actPending, 0, th.Magenta.Render("[!] ")+m.modelLine(&nw, t.w-8)+th.Dim.Render(c.T("ui.pending.hint")))
	}
	if len(s.Training.Archive) == 0 {
		return
	}
	t.blank()
	m.section(t, c.T("ui.sec.archive", len(s.Training.Archive)))
	for i := range s.Training.Archive {
		md := &s.Training.Archive[i]
		t.item(actArchived, i, th.Dim.Render(m.cell(md.Name, 8))+th.Gold.Render(m.cell(c.T("ui.model.bench", fixed(md.Benchmark, 1)), 11))+
			th.Dim.Render(m.cell(c.T("ui.model.params", m.num(md.Params)), 14)+c.T("ui.model.age", m.dur(s.Clock-md.ReleasedS))))
	}
}
