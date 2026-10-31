package ui

import (
	"math"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"token-tycoon/internal/format"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/widgets"
)

type act uint8

const (
	actNone act = iota
	actRack
	actGrid
	actPPA
	actSMR
	actOrbital
	actCloud
	actScraper
	actLicense
	actSynth
	actTrainN
	actTrainD
	actTrainShare
	actTrainStart
	actJob
	actModel
	actSegment
	actHire
	actStaff
	actOffer
	actContract
	actFunding
	actNode
	actRule
	actRuleNew
	actRegion
	actCompliance
	actAGI
	actEnding
	actPending
	actArchived
)

type tkey struct {
	act act
	idx int
}

type trow struct {
	text string
	key  tkey
}

func (r trow) selectable() bool { return r.key.act != actNone }

type tabList struct {
	rows []trow
	w    int
}

func (t *tabList) text(s string) { t.rows = append(t.rows, trow{text: s}) }

func (t *tabList) blank() { t.text("") }

func (t *tabList) item(a act, i int, s string) {
	t.rows = append(t.rows, trow{text: s, key: tkey{act: a, idx: i}})
}

const (
	colLabel   = 14
	colOwned   = 7
	trainStep  = 0.25
	shareStep  = 0.1
	synthStep  = 0.05
	segStep    = 0.05
	synthMax   = 0.9
	draftShare = 0.3
	draftTimeS = 1800
)

func (m *Model) section(t *tabList, title string) {
	ttl := " " + strings.ToUpper(title) + " "
	t.text(m.th.Dim.Render("──" + ttl + strings.Repeat("─", max(0, t.w-4-widgets.Width(ttl)))))
}

func (m *Model) cell(s string, w int) string { return widgets.Fit(s, w) }

func (m *Model) mainWidth() int { return m.panelWidths(m.geom())[0] }

func (m *Model) tabRows(w int) []trow {
	t := &tabList{rows: m.trows[:0], w: w}
	switch {
	case m.tab == tabStats:
		m.statsRows(t)
	case !m.unlocked(tabUnlock[m.tab]):
		m.lockedRows(t)
	case m.tab == tabInfra:
		m.infraRows(t)
	case m.tab == tabModels:
		m.modelRows(t)
		if m.s.IsUnlocked(sim.UnlockTalent) {
			m.talentRows(t)
		}
	case m.tab == tabMarket:
		m.marketRows(t)
		if m.s.IsUnlocked(sim.UnlockRegions) {
			m.regionRows(t)
		}
		if m.s.IsUnlocked(sim.UnlockInvestors) {
			m.investorRows(t)
		}
		if m.s.IsUnlocked(sim.UnlockContracts) {
			m.contractRows(t)
		}
	case m.tab == tabResearch:
		if m.s.IsUnlocked(sim.UnlockAGI) {
			m.agiRows(t)
		}
		m.researchRows(t)
	case m.tab == tabRules:
		m.rulesRows(t)
	}
	m.trows = t.rows
	return t.rows
}

func (m *Model) lockedRows(t *tabList) {
	u := tabUnlock[m.tab]
	need := sim.UnlockThreshold(u, m.b)
	t.blank()
	t.text(m.th.Dim.Render(m.cat.T("ui.tab.locked", m.unlockName(u), m.money(need))))
	t.text(m.label("ui.detail.eta") + m.th.Text.Render(m.etaLifetime(need)))
}

func (m *Model) tabSelIndex(rs []trow) int {
	first := -1
	for i, r := range rs {
		if !r.selectable() {
			continue
		}
		if r.key == m.tsel[m.tab] {
			return i
		}
		if first < 0 {
			first = i
		}
	}
	if first >= 0 {
		m.tsel[m.tab] = rs[first].key
	} else {
		m.tsel[m.tab] = tkey{}
	}
	return first
}

func (m *Model) tabLines(w, h int) []string {
	rs := m.tabRows(w)
	si := m.tabSelIndex(rs)
	sc := m.tscroll[m.tab]
	if si >= 0 {
		if si < sc {
			sc = si
		}
		if si >= sc+h {
			sc = si - h + 1
		}
	}
	sc = max(0, min(sc, len(rs)-h))
	m.tscroll[m.tab] = sc
	out := make([]string, 0, h)
	for i := sc; i < len(rs) && len(out) < h; i++ {
		pre := "  "
		if i == si {
			pre = m.th.Marker.Render(">") + " "
		}
		out = append(out, pre+rs[i].text)
	}
	return out
}

func (m *Model) tabMove(d int) {
	rs := m.tabRows(m.mainWidth())
	i := m.tabSelIndex(rs)
	if i < 0 {
		m.tscroll[m.tab] = max(0, m.tscroll[m.tab]+d)
		return
	}
	step := 1
	if d < 0 {
		step = -1
	}
	for n, j := 0, i+step; j >= 0 && j < len(rs) && n < abs(d); j += step {
		if rs[j].selectable() {
			m.tsel[m.tab] = rs[j].key
			n++
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m *Model) tabCurrent() tkey {
	rs := m.tabRows(m.mainWidth())
	if m.tabSelIndex(rs) < 0 {
		return tkey{}
	}
	return m.tsel[m.tab]
}

func (m *Model) tabKey(k string) tea.Cmd {
	page := max(1, m.listHeight()-1)
	switch k {
	case "up", "k":
		m.tabMove(-1)
	case "down", "j":
		m.tabMove(1)
	case "home":
		m.tabMove(-1 << 20)
	case "end":
		m.tabMove(1 << 20)
	case "pgup":
		m.tabMove(-page)
	case "pgdown":
		m.tabMove(page)
	case "tab":
		m.qty = (m.qty + 1) % len(qtys)
	case "shift+tab":
		m.qty = (m.qty + len(qtys) - 1) % len(qtys)
	case "enter":
		return m.tabEnter()
	case "+", "=":
		return m.tabAdjust(1)
	case "-", "_":
		return m.tabAdjust(-1)
	case "x":
		return m.tabRemove()
	case "s":
		if m.tab == tabModels && m.unlocked(tabUnlock[tabModels]) {
			m.suggestDraft()
		}
	}
	return nil
}

func (m *Model) tabEnter() tea.Cmd {
	k := m.tabCurrent()
	s := m.s
	switch k.act {
	case actRack:
		return m.exec(sim.BuyRack(k.idx, qtys[m.qty]))
	case actGrid:
		return m.exec(sim.BuyGrid(m.gridStep()))
	case actPPA:
		return m.exec(sim.BuyPPA(m.ppaStep()))
	case actSMR:
		return m.exec(sim.BuySMR())
	case actOrbital:
		return m.exec(sim.BuyOrbital())
	case actCloud:
		return m.exec(sim.SetCloud(!s.Infra.CloudEnabled))
	case actScraper:
		return m.exec(sim.BuyScraper(qtys[m.qty]))
	case actLicense:
		return m.exec(sim.BuyLicense())
	case actSynth:
		if s.Training.SynthShare > 0 {
			return m.exec(sim.SetSynthetic(0))
		}
		return m.exec(sim.SetSynthetic(2 * synthStep))
	case actTrainN, actTrainD, actTrainShare, actTrainStart:
		m.ensureDraft()
		return m.exec(sim.ManualTrain(m.draft.n, m.draft.d, m.draft.share))
	case actModel:
		return m.cycleSegment(k.idx)
	case actHire:
		return m.exec(sim.Hire(k.idx))
	case actOffer:
		return m.exec(sim.AcceptContract(k.idx))
	case actFunding:
		return m.openFunding()
	case actNode:
		return m.exec(sim.Research(k.idx))
	case actRule:
		return m.exec(sim.ToggleRule(k.idx))
	case actRuleNew:
		m.openWizard()
	case actRegion:
		return m.exec(sim.EnterRegion(k.idx))
	case actCompliance:
		return m.exec(sim.BuyCompliance(k.idx))
	case actAGI:
		return m.agiEnter(k)
	case actEnding:
		m.modal = modal{kind: modalEnding}
	case actPending:
		m.later = false
		m.openChoice()
	case actArchived:
		return m.archiveEnter(k.idx)
	}
	return nil
}

func (m *Model) tabAdjust(dir int) tea.Cmd {
	k := m.tabCurrent()
	s := m.s
	f := math.Pow(10, trainStep*float64(dir))
	switch k.act {
	case actTrainN:
		m.ensureDraft()
		m.draft.n *= f
		m.clampDraft()
	case actTrainD:
		m.ensureDraft()
		m.draft.d *= f
		m.clampDraft()
	case actTrainShare:
		m.ensureDraft()
		m.draft.share += shareStep * float64(dir)
		m.clampDraft()
	case actJob:
		if k.idx < len(s.Training.Queue) {
			sh := math.Round((s.Training.Queue[k.idx].ClusterShare+shareStep*float64(dir))*100) / 100
			return m.exec(sim.SetTrainShare(k.idx, math.Min(1, math.Max(shareStep, sh))))
		}
	case actSynth:
		sh := math.Round((s.Training.SynthShare+synthStep*float64(dir))*100) / 100
		return m.exec(sim.SetSynthetic(math.Min(synthMax, math.Max(0, sh))))
	case actSegment:
		return m.adjustSegment(k.idx, float64(dir)*segStep)
	case actRule:
		if j := k.idx - dir; j >= 0 && j < len(s.Rules) {
			m.tsel[m.tab] = tkey{act: actRule, idx: j}
			return m.exec(sim.MoveRule(k.idx, -dir))
		}
	}
	return nil
}

func (m *Model) tabRemove() tea.Cmd {
	k := m.tabCurrent()
	switch k.act {
	case actRack:
		return m.exec(sim.SellRack(k.idx, 1))
	case actJob:
		return m.exec(sim.CancelTrain(k.idx))
	case actModel:
		if k.idx < len(m.s.Training.Models) {
			m.modal = modal{kind: modalRetire, idx: k.idx}
		}
	case actStaff:
		return m.exec(sim.Fire(k.idx))
	case actOffer:
		return m.exec(sim.DeclineContract(k.idx))
	case actRule:
		return m.exec(sim.DeleteRule(k.idx))
	case actArchived:
		if k.idx < len(m.s.Training.Archive) {
			m.modal = modal{kind: modalDeleteArchived, idx: k.idx}
		}
	}
	return nil
}

func niceCeil(x float64) float64 {
	if !(x > 0) || math.IsInf(x, 0) {
		return 1
	}
	p := math.Pow(10, math.Floor(math.Log10(x))-1)
	return math.Ceil(x/p-1e-9) * p
}

func (m *Model) gridStep() float64 {
	if need := sim.GridNeededMW(m.s, m.b, 0); need > 0 {
		return niceCeil(need)
	}
	return niceCeil(math.Max(1, m.s.Infra.Energy.GridMW*0.25))
}

func (m *Model) ppaStep() float64 {
	return niceCeil(math.Max(1, m.s.Derived.GridUsed))
}

func pct(v float64) string {
	x := v * 100
	switch {
	case !(math.Abs(x) < 1e6):
		return "--"
	case math.Abs(x) >= 10 || x == math.Trunc(x):
		return strconv.FormatFloat(math.Round(x), 'f', 0, 64) + "%"
	}
	return strconv.FormatFloat(x, 'f', 1, 64) + "%"
}

func fixed(v float64, prec int) string { return strconv.FormatFloat(v, 'f', prec, 64) }

func (m *Model) mw(v float64) string { return m.num(v) + " MW" }

func (m *Model) dur(s float64) string { return format.Duration(s) }

func (m *Model) qtyRacks(gen int) (int, float64) {
	q := qtys[m.qty]
	if q == sim.QtyMax {
		q = max(1, sim.MaxRacks(m.s, m.b, gen))
	}
	return q, sim.RackCost(m.s, m.b, gen, q)
}

func (m *Model) qtyScrapers() (int, float64) {
	q := qtys[m.qty]
	if q == sim.QtyMax {
		q = max(1, sim.MaxScrapers(m.s, m.b))
	}
	return q, sim.ScraperCost(m.s, m.b, q)
}

func (m *Model) tabKeyItems(items []string) []string {
	c := m.cat
	k := m.tabCurrent()
	switch k.act {
	case actRack:
		q, cost := m.qtyRacks(k.idx)
		items = append(items, c.T("ui.key.buy", "x"+strconv.Itoa(q), m.money(cost)), c.T("key.qty", m.qtyLabel()))
		if m.s.Infra.Racks[k.idx-1] > 0 {
			items = append(items, c.T("ui.key.sell"))
		}
	case actGrid:
		items = append(items, c.T("ui.key.buy", m.mw(m.gridStep()), m.money(sim.GridCost(m.b, m.gridStep()))))
	case actPPA:
		items = append(items, c.T("ui.key.buy", m.mw(m.ppaStep()), m.money(sim.PPACost(m.b, m.ppaStep()))))
	case actSMR:
		items = append(items, c.T("ui.key.buy_upgrade", m.money(sim.SMRCost(m.s, m.b))))
	case actOrbital:
		items = append(items, c.T("ui.key.buy_upgrade", m.money(sim.OrbitalCost(m.s, m.b))))
	case actCloud, actSynth:
		items = append(items, c.T("ui.key.toggle"))
		if k.act == actSynth {
			items = append(items, c.T("ui.key.adjust"))
		}
	case actScraper:
		q, cost := m.qtyScrapers()
		items = append(items, c.T("ui.key.buy", "x"+strconv.Itoa(q), m.money(cost)), c.T("key.qty", m.qtyLabel()))
	case actLicense:
		items = append(items, c.T("ui.key.buy_upgrade", m.money(m.s.Derived.Train.LicenseCost)))
	case actTrainN, actTrainD, actTrainShare, actTrainStart:
		items = append(items, c.T("ui.key.train"))
		if k.act != actTrainStart {
			items = append(items, c.T("ui.key.adjust"))
		}
		items = append(items, c.T("ui.key.suggest"))
	case actJob:
		items = append(items, c.T("ui.key.adjust"), c.T("ui.key.cancel_job"))
	case actModel:
		items = append(items, c.T("ui.key.assign"), c.T("ui.key.retire"))
	case actSegment:
		items = append(items, c.T("ui.key.adjust"))
	case actHire:
		items = append(items, c.T("ui.key.hire", m.money(sim.HireCost(m.s, m.b, k.idx))))
	case actStaff:
		items = append(items, c.T("ui.key.fire"))
	case actOffer:
		items = append(items, c.T("ui.key.accept"), c.T("ui.key.decline"))
	case actFunding:
		items = append(items, c.T("ui.key.preview"))
	case actNode:
		if sim.NodeState(m.s, m.b, k.idx) == sim.NodeAvailable {
			items = append(items, c.T("ui.key.research", m.num(m.b.ResearchNodes[k.idx].Cost)))
		}
	case actRule:
		items = append(items, c.T("ui.key.toggle"), c.T("ui.key.priority"), c.T("ui.key.delete_rule"))
	case actRuleNew:
		items = append(items, c.T("ui.key.new_rule"))
	case actRegion:
		if !sim.RegionEntered(m.s, m.b, k.idx) {
			items = append(items, c.T("ui.key.enter_region", m.money(sim.EntryCost(m.s, m.b, k.idx))))
		}
	case actCompliance:
		if sim.RegionEntered(m.s, m.b, k.idx) && !sim.RegionCompliant(m.s, m.b, k.idx) {
			items = append(items, c.T("ui.key.compliance", m.money(sim.ComplianceCost(m.s, m.b, k.idx))))
		}
	case actAGI:
		if k.idx == m.s.AGI.Stage && !sim.AGIEnded(m.s, m.b) && m.s.AGI.TrainEndsS == 0 {
			items = append(items, c.T("ui.key.agi_invest"))
		}
	case actEnding:
		items = append(items, c.T("ui.key.ending"))
	case actPending:
		items = append(items, c.T("ui.key.choose"))
	case actArchived:
		items = append(items, c.T("ui.key.swap"), c.T("ui.key.delete_archived"))
	}
	return items
}
