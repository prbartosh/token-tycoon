package ui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

type wizStep uint8

const (
	wzMetric wizStep = iota
	wzValue
	wzMore
	wzAction
	wzBusiness
	wzQty
	wzFree
	wzConfirm
)

type wizChoice struct {
	label  string
	metric int
	op     string
	value  float64
	index  int
}

type wizFrame struct {
	step wizStep
	cond int
}

type wizard struct {
	step  wizStep
	cond  int
	sel   int
	rule  sim.Rule
	trail []wizFrame
}

const ruleLogShown = 12

func (m *Model) metricName(id string) string { return m.cat.T("rule.metric." + id) }

func (m *Model) metricValue(k sim.MetricKind, v float64) string {
	switch k {
	case sim.KindTimes:
		return "x" + strconv.FormatFloat(v, 'g', 4, 64)
	case sim.KindMinutes:
		return m.cat.T("rule.minutes", strconv.FormatFloat(v, 'g', 4, 64))
	case sim.KindRate:
		return m.money(v) + "/s"
	case sim.KindPercent:
		return pct(v)
	case sim.KindSeconds:
		return m.dur(v)
	}
	return strconv.FormatFloat(v, 'g', 4, 64)
}

func (m *Model) condText(cd *sim.Cond) string {
	mi := sim.MetricIndex(cd.Metric)
	if mi < 0 {
		return "?"
	}
	def := &sim.Metrics[mi]
	if cd.Op == "" {
		return m.metricName(cd.Metric) + " ..."
	}
	if def.Kind == sim.KindBool {
		if cd.Op == sim.OpGreater {
			return m.cat.T("rule.is_true", m.metricName(cd.Metric))
		}
		return m.cat.T("rule.is_false", m.metricName(cd.Metric))
	}
	return m.metricName(cd.Metric) + " " + cd.Op + " " + m.metricValue(def.Kind, cd.Value)
}

func qtyText(q int) string {
	if q == sim.QtyMax {
		return "MAX"
	}
	return "x" + strconv.Itoa(q)
}

func (m *Model) actionText(a *sim.Action) string {
	ai := sim.ActionIndex(a.Kind)
	if ai < 0 {
		return "?"
	}
	name := m.cat.T("rule.action." + a.Kind)
	switch sim.Actions[ai].Param {
	case sim.ParamQty:
		return name + " " + qtyText(a.Qty)
	case sim.ParamBusinessQty:
		if a.Index >= 0 && a.Index < sim.N {
			return name + " " + m.bizName(a.Index) + " " + qtyText(a.Qty)
		}
	case sim.ParamFree:
		return name + " " + pct(a.Value)
	}
	return name
}

func (m *Model) ruleText(r *sim.Rule) string {
	c := m.cat
	s := c.T("rule.if") + " " + m.condText(&r.If[0])
	for k := 1; k < sim.MaxConds; k++ {
		if r.If[k].Metric != "" {
			s += " " + c.T("rule.and") + " " + m.condText(&r.If[k])
		}
	}
	if r.Then.Kind != "" {
		s += " " + c.T("rule.then") + " " + m.actionText(&r.Then)
	}
	return s
}

func (m *Model) rulesRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	slots := sim.RuleSlots(s, b)
	off := c.T("ui.rules.offline_off")
	if s.ResMods.OfflineRules {
		off = c.T("ui.rules.offline_on")
	}
	t.text(th.Text.Render(c.T("ui.rules.slots", len(s.Rules), slots, b.Rules.SlotsMax)) + "   " + th.Dim.Render(off))
	t.text(th.Dim.Render(c.T("ui.rules.note", fixed(b.Rules.EvalEveryS, 0))))
	t.blank()
	m.section(t, c.T("ui.sec.rules"))
	for i := range s.Rules {
		r := &s.Rules[i]
		st, on := th.Text, c.T("ui.rules.on")
		if !r.Enabled {
			st, on = th.Dim, c.T("ui.rules.off")
		}
		if i >= slots {
			st = th.Dim
		}
		info := c.T("ui.rules.fired", r.Fired)
		if r.Fired > 0 {
			info = c.T("ui.rules.fired_ago", r.Fired, m.dur(s.Clock-r.LastS))
		}
		t.item(actRule, i, st.Render(strconv.Itoa(i+1)+". ["+on+"] "+m.ruleText(r)+"  ")+th.Dim.Render(info))
	}
	if len(s.Rules) < slots {
		t.item(actRuleNew, 0, th.Bold.Render("[ "+c.T("ui.rules.new")+" ]"))
	} else {
		t.text(th.Dim.Render(c.T("ui.rules.full")))
	}
	t.blank()
	m.section(t, c.T("ui.sec.rule_log"))
	if len(m.ruleLog) == 0 {
		t.text(th.Dim.Render(c.T("ui.rules.log_empty")))
	}
	for i, n := len(m.ruleLog)-1, 0; i >= 0 && n < ruleLogShown; i, n = i-1, n+1 {
		t.text(th.Dim.Render(m.ruleLog[i]))
	}
}

func (m *Model) addRuleLog(s string) {
	if len(m.ruleLog) == logCap {
		copy(m.ruleLog, m.ruleLog[1:])
		m.ruleLog = m.ruleLog[:logCap-1]
	}
	m.ruleLog = append(m.ruleLog, s)
}

func (m *Model) openWizard() {
	m.wiz = wizard{rule: sim.Rule{Enabled: true}}
	m.modal = modal{kind: modalWizard}
}

func (m *Model) incomePresets() []float64 {
	base := m.s.Derived.BaseIncome
	if !(base > 0) {
		return []float64{1e3, 1e6, 1e9}
	}
	out := make([]float64, 0, 4)
	for _, f := range [...]float64{0.5, 1, 2, 10} {
		out = append(out, niceCeil(base*f))
	}
	return out
}

func (m *Model) wizChoices() []wizChoice {
	c, w := m.cat, &m.wiz
	var out []wizChoice
	switch w.step {
	case wzMetric, wzMore:
		if w.step == wzMore {
			out = append(out, wizChoice{label: c.T("ui.wizard.no_more"), metric: -1})
		}
		for i := range sim.Metrics {
			out = append(out, wizChoice{label: m.metricName(sim.Metrics[i].ID), metric: i})
		}
	case wzValue:
		mi := sim.MetricIndex(w.rule.If[w.cond].Metric)
		def := &sim.Metrics[mi]
		if def.Kind == sim.KindBool {
			for _, op := range [...]string{sim.OpGreater, sim.OpLess} {
				cd := sim.Cond{Metric: def.ID, Op: op, Value: 0.5}
				out = append(out, wizChoice{label: m.condText(&cd), op: op, value: 0.5})
			}
			break
		}
		presets := def.Presets
		if def.Kind == sim.KindRate {
			presets = m.incomePresets()
		}
		for _, op := range [...]string{sim.OpGreater, sim.OpLess} {
			for _, v := range presets {
				cd := sim.Cond{Metric: def.ID, Op: op, Value: v}
				out = append(out, wizChoice{label: m.condText(&cd), op: op, value: v})
			}
		}
	case wzAction:
		for i := range sim.Actions {
			out = append(out, wizChoice{label: c.T("rule.action." + sim.Actions[i].ID), index: i})
		}
	case wzBusiness:
		for i := range sim.N {
			if m.s.IsUnlocked(sim.UnlockBusiness0 + sim.Unlock(i)) {
				out = append(out, wizChoice{label: m.bizName(i), index: i})
			}
		}
	case wzQty:
		for _, q := range sim.RuleQtys {
			out = append(out, wizChoice{label: qtyText(q), index: q})
		}
	case wzFree:
		for _, v := range sim.FreePresets {
			out = append(out, wizChoice{label: pct(v), value: v})
		}
	case wzConfirm:
		out = append(out, wizChoice{label: c.T("ui.wizard.save")}, wizChoice{label: c.T("ui.wizard.discard")})
	}
	return out
}

func (m *Model) wizGo(next wizStep) {
	m.wiz.trail = append(m.wiz.trail, wizFrame{step: m.wiz.step, cond: m.wiz.cond})
	m.wiz.step, m.wiz.sel = next, 0
}

func (m *Model) wizKey(k string) tea.Cmd {
	w := &m.wiz
	ch := m.wizChoices()
	switch k {
	case "esc":
		m.modal = modal{}
		return nil
	case "up", "k":
		w.sel = max(0, w.sel-1)
	case "down", "j":
		w.sel = min(len(ch)-1, w.sel+1)
	case "backspace", "left":
		if n := len(w.trail); n > 0 {
			f := w.trail[n-1]
			w.step, w.cond, w.sel = f.step, f.cond, 0
			w.trail = w.trail[:n-1]
		}
	case "enter", "right":
		if w.sel < 0 || w.sel >= len(ch) {
			return nil
		}
		return m.wizChoose(ch[w.sel])
	}
	return nil
}

func (m *Model) wizChoose(o wizChoice) tea.Cmd {
	w := &m.wiz
	switch w.step {
	case wzMetric, wzMore:
		if o.metric < 0 {
			w.rule.If[1] = sim.Cond{}
			m.wizGo(wzAction)
			return nil
		}
		if w.step == wzMore {
			w.cond = 1
		}
		w.rule.If[w.cond] = sim.Cond{Metric: sim.Metrics[o.metric].ID}
		m.wizGo(wzValue)
	case wzValue:
		w.rule.If[w.cond].Op, w.rule.If[w.cond].Value = o.op, o.value
		if w.cond+1 < sim.MaxConds {
			m.wizGo(wzMore)
		} else {
			m.wizGo(wzAction)
		}
	case wzAction:
		a := &sim.Actions[o.index]
		w.rule.Then = sim.Action{Kind: a.ID}
		switch a.Param {
		case sim.ParamQty:
			m.wizGo(wzQty)
		case sim.ParamBusinessQty:
			m.wizGo(wzBusiness)
		case sim.ParamFree:
			m.wizGo(wzFree)
		default:
			m.wizGo(wzConfirm)
		}
	case wzBusiness:
		w.rule.Then.Index = o.index
		m.wizGo(wzQty)
	case wzQty:
		w.rule.Then.Qty = o.index
		m.wizGo(wzConfirm)
	case wzFree:
		w.rule.Then.Value = o.value
		m.wizGo(wzConfirm)
	case wzConfirm:
		m.modal = modal{}
		if o.label == m.cat.T("ui.wizard.save") {
			return m.exec(sim.AddRule(w.rule))
		}
	}
	return nil
}

func (m *Model) wizardBox() string {
	th, c, w := m.th, m.cat, &m.wiz
	bw := min(modalW+8, m.width-2)
	body := []string{"", " " + th.Gold.Render(c.T("ui.wizard.step."+strconv.Itoa(int(w.step)))), ""}
	preview := m.ruleText(&w.rule)
	if w.rule.If[0].Metric == "" {
		preview = c.T("rule.if") + " ..."
	}
	for _, l := range wrapLines(preview, bw-4) {
		body = append(body, " "+th.Text.Render(l))
	}
	body = append(body, "")
	ch := m.wizChoices()
	h := max(3, min(len(ch), m.height-10-len(body)))
	top := max(0, min(w.sel-h/2, len(ch)-h))
	for i := top; i < len(ch) && i < top+h; i++ {
		pre, st := "   ", th.Text
		if i == w.sel {
			pre, st = " "+th.Marker.Render(">")+" ", th.Bold
		}
		body = append(body, pre+st.Render(widgets.Fit(ch[i].label, bw-6)))
	}
	body = append(body, "", widgets.Center(th.KeyText.Render(c.T("ui.wizard.keys")), bw-2))
	return m.box(c.T("ui.wizard.title"), body, bw)
}

func (m *Model) ruleFired(e *sim.Event) string {
	i := e.Index
	if i < 0 || i >= len(m.s.Rules) {
		return m.cat.T("msg.rule_fired", i+1, "")
	}
	return m.cat.T("msg.rule_fired", i+1, m.actionText(&m.s.Rules[i].Then))
}
