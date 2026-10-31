package ui

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

type palCmd uint8

const (
	pcBuy palCmd = iota
	pcBuyMax
	pcTrain
	pcGo
	pcPrestige
	pcFunding
	pcPolicies
	pcSave
	pcSettings
	pcLanguage
	pcNotation
	pcColorblind
	pcAdvisor
	pcBell
	pcHelp
	pcTutorial
	pcQuit
)

type palEntry struct {
	cmd palCmd
	key string
}

var palEntries = [...]palEntry{
	{pcBuyMax, "ui.cmd.buy_max"},
	{pcTrain, "ui.cmd.train"},
	{pcPrestige, "ui.cmd.prestige"},
	{pcFunding, "ui.cmd.funding"},
	{pcPolicies, "ui.cmd.policies"},
	{pcSave, "ui.cmd.save"},
	{pcSettings, "ui.cmd.settings"},
	{pcLanguage, "ui.cmd.language"},
	{pcNotation, "ui.cmd.notation"},
	{pcColorblind, "ui.cmd.colorblind"},
	{pcAdvisor, "ui.cmd.advisor"},
	{pcBell, "ui.cmd.bell"},
	{pcHelp, "ui.cmd.help"},
	{pcTutorial, "ui.cmd.tutorial"},
	{pcQuit, "ui.cmd.quit"},
}

const (
	palW       = 60
	palMaxRows = 10
	palQtyMax  = "max"
)

type palItem struct {
	cmd   palCmd
	arg   int
	qty   int
	label string
	score int
}

type palette struct {
	open  bool
	query []rune
	sel   int
	items []palItem
}

var palCats = [2]*i18n.Catalog{i18n.New(i18n.EN), i18n.New(i18n.PL)}

func (m *Model) openPalette() {
	m.pal.open = true
	m.pal.query = m.pal.query[:0]
	m.pal.sel = 0
	m.refreshPalette()
}

func (m *Model) closePalette() { m.pal.open = false }

func fold(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ą':
			r = 'a'
		case 'ć':
			r = 'c'
		case 'ę':
			r = 'e'
		case 'ł':
			r = 'l'
		case 'ń':
			r = 'n'
		case 'ó':
			r = 'o'
		case 'ś':
			r = 's'
		case 'ź', 'ż':
			r = 'z'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func fuzzy(q, s string) int {
	if q == "" {
		return 0
	}
	fq, fs := fold(q), fold(s)
	bonus := 0
	if i := strings.Index(fs, fq); i >= 0 {
		bonus = 400
		if i == 0 || fs[i-1] == ' ' || fs[i-1] == '(' {
			bonus = 800
		}
	}
	qs, ts := []rune(fq), []rune(fs)
	score, qi, run := 0, 0, 0
	for ti := 0; ti < len(ts) && qi < len(qs); ti++ {
		if qs[qi] == ' ' && ts[ti] != ' ' {
			continue
		}
		if ts[ti] != qs[qi] {
			run = 0
			continue
		}
		run++
		score += run * 2
		if ti == 0 || !unicode.IsLetter(ts[ti-1]) && !unicode.IsDigit(ts[ti-1]) {
			score += 6
		}
		qi++
	}
	if qi < len(qs) {
		return -1
	}
	return bonus + score*10 - len(ts)
}

func bestScore(q string, labels ...string) int {
	best := -1
	for _, l := range labels {
		best = max(best, fuzzy(q, l))
	}
	return best
}

func parseQty(tok string) (int, bool) {
	t := fold(tok)
	t = strings.TrimPrefix(t, "x")
	if t == palQtyMax || t == "maks" {
		return sim.QtyMax, true
	}
	n, err := strconv.Atoi(t)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func (m *Model) isBuyVerb(tok string) bool {
	t := fold(tok)
	for _, c := range palCats {
		if t == fold(c.T("ui.cmd.buy")) {
			return true
		}
	}
	return false
}

func (m *Model) buyItems(fields []string, items []palItem) []palItem {
	rest := fields[1:]
	qty := sim.Qty1
	if n := len(rest); n > 0 {
		if q, ok := parseQty(rest[n-1]); ok {
			qty, rest = q, rest[:n-1]
		}
	}
	bq := strings.Join(rest, " ")
	for i := range sim.N {
		if !m.s.IsUnlocked(sim.UnlockBusiness0 + sim.Unlock(i)) {
			continue
		}
		id := m.b.Businesses[i].ID
		sc := bestScore(bq, palCats[0].T(i18n.BusinessKey(id)), palCats[1].T(i18n.BusinessKey(id)), strings.ReplaceAll(id, "_", " "))
		if sc < 0 {
			continue
		}
		q := "x" + strconv.Itoa(qty)
		if qty == sim.QtyMax {
			q = m.cat.T("qty.max")
		}
		_, cost := sim.ResolveQty(m.s, m.b, i, qty)
		label := m.cat.T("ui.cmd.buy") + " " + m.bizName(i) + " " + q + "  " + m.money(cost)
		items = append(items, palItem{cmd: pcBuy, arg: i, qty: qty, label: label, score: sc + 10000})
	}
	return items
}

func (m *Model) refreshPalette() {
	q := strings.TrimSpace(string(m.pal.query))
	items := m.pal.items[:0]
	fields := strings.Fields(q)
	if len(fields) > 0 && m.isBuyVerb(fields[0]) && (len(fields) > 1 || strings.HasSuffix(string(m.pal.query), " ")) {
		items = m.buyItems(fields, items)
	}
	for _, e := range palEntries {
		if sc := bestScore(q, palCats[0].T(e.key), palCats[1].T(e.key)); sc >= 0 {
			items = append(items, palItem{cmd: e.cmd, label: m.cat.T(e.key), score: sc})
		}
	}
	for t := range numTabs {
		lbl := func(c *i18n.Catalog) string { return c.T("ui.cmd.go") + " " + strings.ToLower(c.T(tabKeys[t])) }
		if sc := bestScore(q, lbl(palCats[0]), lbl(palCats[1])); sc >= 0 {
			items = append(items, palItem{cmd: pcGo, arg: t, label: lbl(m.cat), score: sc})
		}
	}
	if q == "" {
		items = append(items, palItem{cmd: pcBuy, arg: -1, label: m.cat.T("ui.cmd.buy_hint")})
	} else {
		sort.SliceStable(items, func(a, b int) bool { return items[a].score > items[b].score })
	}
	m.pal.items = items
	m.pal.sel = min(max(0, m.pal.sel), max(0, len(items)-1))
}

func (m *Model) paletteKey(msg tea.KeyPressMsg) tea.Cmd {
	p := &m.pal
	switch k := msg.String(); k {
	case "esc", "ctrl+p":
		m.closePalette()
		return nil
	case "enter":
		if p.sel < len(p.items) {
			it := p.items[p.sel]
			m.closePalette()
			return m.runPalette(it)
		}
		return nil
	case "up", "ctrl+k":
		if n := len(p.items); n > 0 {
			p.sel = (p.sel + n - 1) % n
		}
		return nil
	case "down", "ctrl+j", "tab":
		if n := len(p.items); n > 0 {
			p.sel = (p.sel + 1) % n
		}
		return nil
	case "backspace":
		if n := len(p.query); n > 0 {
			p.query = p.query[:n-1]
		}
	case "ctrl+u":
		p.query = p.query[:0]
	case "space":
		p.query = append(p.query, ' ')
	default:
		if msg.Text == "" {
			return nil
		}
		for _, r := range msg.Text {
			if unicode.IsPrint(r) {
				p.query = append(p.query, r)
			}
		}
	}
	p.sel = 0
	m.refreshPalette()
	return nil
}

func (m *Model) buyTarget() int {
	if m.tab == tabBusinesses && m.sel.kind == rowBusiness {
		return m.sel.idx
	}
	if m.adv.ok && m.adv.kind == sim.PurchaseBusiness {
		return m.adv.idx
	}
	return 0
}

func (m *Model) runPalette(it palItem) tea.Cmd {
	switch it.cmd {
	case pcBuy:
		if it.arg < 0 {
			m.openPalette()
			m.pal.query = append(m.pal.query, []rune(m.cat.T("ui.cmd.buy")+" ")...)
			m.refreshPalette()
			return nil
		}
		return m.exec(sim.Buy(it.arg, it.qty))
	case pcBuyMax:
		return m.exec(sim.Buy(m.buyTarget(), sim.QtyMax))
	case pcTrain:
		m.setTab(tabModels)
		if !m.s.IsUnlocked(sim.UnlockTraining) {
			return nil
		}
		m.ensureDraft()
		return m.exec(sim.ManualTrain(m.draft.n, m.draft.d, m.draft.share))
	case pcGo:
		m.setTab(it.arg)
	case pcPrestige:
		m.openPrestige()
	case pcFunding:
		return m.openFunding()
	case pcPolicies:
		m.setTab(tabRules)
	case pcSave:
		m.saveNow()
	case pcSettings:
		m.modal = modal{kind: modalSettings}
	case pcLanguage:
		m.toggleLang()
	case pcNotation:
		m.toggleNotation()
	case pcColorblind:
		m.toggleColorblind()
	case pcAdvisor:
		m.toggleAdvisor()
	case pcBell:
		m.toggleBell()
	case pcHelp:
		m.openHelp()
	case pcTutorial:
		m.openTutorial()
	case pcQuit:
		return tea.Quit
	}
	return nil
}

func (m *Model) paletteBox() string {
	th, c, p := m.th, m.cat, &m.pal
	w := min(palW, m.width-2)
	rows := max(1, min(palMaxRows, m.height-8))
	body := []string{" " + th.Key.Render(":") + " " + th.Bold.Render(string(p.query)) + th.Marker.Render("_"), ""}
	if len(p.items) == 0 {
		body = append(body, " "+th.Dim.Render(c.T("ui.palette.none")))
	}
	top := max(0, min(p.sel-rows+1, len(p.items)-rows))
	for i := top; i < len(p.items) && i < top+rows; i++ {
		txt := p.items[i].label
		if i == p.sel {
			body = append(body, th.Marker.Render(" > ")+th.Selected.Render(txt))
		} else {
			body = append(body, "   "+th.Text.Render(txt))
		}
	}
	body = append(body, "", widgets.Center(th.KeyText.Render(c.T("ui.palette.keys")), w-2))
	return m.box(c.T("ui.palette.title"), body, w)
}
