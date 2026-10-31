package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

const (
	setLang = iota
	setNotation
	setColorblind
	setAdvisor
	setBell
	numSettings
)

const settingsLabelW = 24

func (m *Model) toggleNotation() {
	if m.notation() == 0 {
		m.s.Settings.Notation = "scientific"
	} else {
		m.s.Settings.Notation = "letters"
	}
	m.setStatus(m.cat.T("ui.msg.notation", m.cat.T("ui.notation."+m.notation().String())), false)
}

func (m *Model) toggleAdvisor() {
	m.s.Settings.Advisor = !m.s.Settings.Advisor
	m.setStatus(m.cat.T(onOff("ui.msg.advisor", m.s.Settings.Advisor)), false)
}

func (m *Model) toggleColorblind() {
	m.s.Settings.Colorblind = !m.s.Settings.Colorblind
	m.setStatus(m.cat.T(onOff("ui.msg.colorblind", m.s.Settings.Colorblind)), false)
}

func (m *Model) toggleBell() {
	m.s.Settings.Bell = !m.s.Settings.Bell
	m.setStatus(m.cat.T(onOff("ui.msg.bell", m.s.Settings.Bell)), false)
}

func (m *Model) toggleLang() {
	l := i18n.PL
	if m.cat.Lang() == i18n.PL {
		l = i18n.EN
	}
	m.setLang(l)
}

func (m *Model) setLang(l i18n.Lang) {
	m.cat = i18n.New(l)
	m.s.Settings.Lang = l.String()
	m.newsIdx = 0
	m.setStatus(m.cat.T("ui.msg.lang"), false)
}

func (m *Model) saveNow() {
	if m.save == nil {
		return
	}
	m.lastSave = m.now()
	if err := m.save(m.s); err != nil {
		m.setStatus(m.cat.T("ui.msg.save_failed", err), true)
		return
	}
	m.setStatus(m.cat.T("ui.msg.saved"), false)
}

func (m *Model) openHelp() {
	m.help = true
	m.helpTop = 0
}

func (m *Model) toggleSetting(i int) {
	switch i {
	case setLang:
		m.toggleLang()
	case setNotation:
		m.toggleNotation()
	case setColorblind:
		m.toggleColorblind()
	case setAdvisor:
		m.toggleAdvisor()
	case setBell:
		m.toggleBell()
	}
}

func (m *Model) settingsKey(k string) tea.Cmd {
	md := &m.modal
	switch k {
	case "esc":
		m.modal = modal{}
	case "up", "k":
		md.sel = (md.sel + numSettings - 1) % numSettings
	case "down", "j":
		md.sel = (md.sel + 1) % numSettings
	case "enter", "space", "left", "right":
		m.toggleSetting(md.sel)
	}
	return nil
}

func (m *Model) settingValue(i int) string {
	c, st := m.cat, &m.s.Settings
	onOffText := func(on bool) string {
		if on {
			return c.T("ui.settings.on")
		}
		return c.T("ui.settings.off")
	}
	switch i {
	case setLang:
		return c.T("ui.settings.lang_name")
	case setNotation:
		return c.T("ui.notation." + m.notation().String())
	case setColorblind:
		return onOffText(st.Colorblind)
	case setAdvisor:
		return onOffText(st.Advisor)
	}
	return onOffText(st.Bell)
}

var settingKeys = [numSettings]string{"ui.settings.lang", "ui.settings.notation", "ui.settings.colorblind",
	"ui.settings.advisor", "ui.settings.bell"}

func (m *Model) settingsBox() string {
	th, c := m.th, m.cat
	w := min(modalW, m.width-2)
	body := []string{""}
	for i := range numSettings {
		line := th.Text.Render(m.cell(c.T(settingKeys[i]), settingsLabelW)) + th.Gold.Render("< "+m.settingValue(i)+" >")
		if i == m.modal.sel {
			body = append(body, th.Marker.Render(" > ")+line)
		} else {
			body = append(body, "   "+line)
		}
	}
	body = append(body, "")
	for _, l := range wrapLines(c.T("ui.settings.note"), w-4) {
		body = append(body, " "+th.Dim.Render(l))
	}
	body = append(body, "", widgets.Center(th.Key.Render(c.T("ui.settings.keys")), w-2))
	return m.box(c.T("ui.settings.title"), body, w)
}
