package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

type modalKind uint8

const (
	modalNone modalKind = iota
	modalPrestige
	modalRetire
	modalFunding
	modalWizard
	modalEnding
	modalChoose
	modalSwap
	modalDeleteArchived
	modalAGI
	modalSettings
)

type modal struct {
	kind    modalKind
	idx     int
	sel     int
	confirm bool
}

const modalW = 64

func (m *Model) modalKey(k string) tea.Cmd {
	md := m.modal
	switch md.kind {
	case modalEnding:
		m.modal = modal{}
		return nil
	case modalChoose:
		return m.choiceKey(k)
	case modalSwap:
		return m.swapKey(k)
	case modalAGI:
		return m.agiKey(k)
	case modalSettings:
		return m.settingsKey(k)
	case modalDeleteArchived:
		switch k {
		case "enter", "y":
			m.modal = modal{}
			return m.exec(sim.DeleteArchived(md.idx))
		case "esc", "n":
			m.modal = modal{}
		}
		return nil
	case modalWizard:
		return m.wizKey(k)
	case modalFunding:
		switch k {
		case "enter", "y":
			m.modal = modal{}
			if sim.Funding(m.s, m.b).Allowed {
				return m.exec(sim.FundingRound())
			}
		case "esc", "n", "P", "shift+p":
			m.modal = modal{}
		}
		return nil
	case modalPrestige, modalRetire:
		switch k {
		case "enter", "y":
			m.modal = modal{}
			if md.kind == modalPrestige {
				if !sim.Preview(m.s, m.b).Allowed {
					return nil
				}
				return m.exec(sim.PrestigeReset())
			}
			if md.idx < len(m.s.Training.Models) {
				m.retiring = m.s.Training.Models[md.idx].Name
				return m.exec(sim.RetireModel(md.idx))
			}
		case "esc", "n", "p":
			m.modal = modal{}
		}
	}
	return nil
}

func (m *Model) modalBox() string {
	switch m.modal.kind {
	case modalPrestige:
		return m.prestigeBox()
	case modalRetire:
		return m.retireBox()
	case modalFunding:
		return m.fundingBox()
	case modalChoose:
		return m.choiceBox()
	case modalSwap:
		return m.swapBox()
	case modalDeleteArchived:
		return m.deleteArchivedBox()
	case modalAGI:
		return m.agiBox()
	case modalSettings:
		return m.settingsBox()
	case modalWizard:
		return m.wizardBox()
	case modalEnding:
		return m.endingBox()
	}
	return ""
}

func (m *Model) prestigeBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	pv := sim.Preview(m.s, m.b)
	body := []string{""}
	body = append(body, " "+th.Params.Render(sim.VersionName(m.b, pv.VersionNow)+"  >  "+sim.VersionName(m.b, pv.VersionAfter)))
	body = append(body, "")
	if !pv.Allowed {
		lock := c.T("prestige.locked", m.money(pv.NeedRun))
		if m.s.IsUnlocked(sim.UnlockPrestige1) && m.s.Lifetime.Run >= pv.NeedRun {
			lock = c.T("ui.prestige.too_short")
		}
		body = append(body, " "+th.Bad.Render(lock))
		body = append(body, " "+th.Text.Render(c.T("ui.prestige.progress", m.money(m.s.Lifetime.Run), m.money(pv.NeedRun))), "")
		body = append(body, widgets.Center(th.KeyText.Render(c.T("ui.key.close")), w-2))
		return m.box(c.T("prestige.title"), body, w)
	}
	body = append(body,
		" "+th.Gold.Render(c.T("prestige.gain", m.num(pv.Gain), m.num(pv.BonusNow), m.num(pv.BonusAfter))),
		" "+th.Text.Render(c.T("ui.prestige.run", m.money(m.s.Lifetime.Run), m.dur(pv.RunS))),
		" "+th.Text.Render(c.T("prestige.return", m.dur(pv.ReturnS))),
	)
	if pv.RawGain > pv.Gain*1.05 {
		for _, l := range wrapLines(c.T("ui.prestige.leap", m.num(pv.RawGain)), w-4) {
			body = append(body, " "+th.Dim.Render(l))
		}
	}
	body = append(body, "")
	for _, l := range wrapLines(c.T("prestige.resets"), w-4) {
		body = append(body, " "+th.Dim.Render(l))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.key.confirm"))+"   "+th.KeyText.Render(c.T("ui.key.cancel")), w-2))
	return m.box(c.T("prestige.title"), body, w)
}

func (m *Model) retireBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	name, bench := "", 0.0
	if i := m.modal.idx; i < len(m.s.Training.Models) {
		name, bench = m.s.Training.Models[i].Name, m.s.Training.Models[i].Benchmark
	}
	body := []string{""}
	for _, l := range wrapLines(c.T("ui.retire.body", name, fixed(bench, 1)), w-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.key.confirm"))+"   "+th.KeyText.Render(c.T("ui.key.cancel")), w-2))
	return m.box(c.T("ui.retire.title"), body, w)
}

func wrapLines(s string, w int) []string {
	return strings.Split(ansi.Wrap(s, max(1, w), ""), "\n")
}

func (m *Model) openPrestige() { m.modal = modal{kind: modalPrestige} }
