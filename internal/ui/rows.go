package ui

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/sim"
)

type rowKind uint8

const (
	rowBusiness rowKind = iota
	rowLocked
	rowHeader
	rowUpgrade
)

type row struct {
	kind rowKind
	idx  int
}

func (r row) selectable() bool { return r.kind != rowHeader }

func (r row) slot() int {
	if r.kind == rowUpgrade {
		return upgradeSlot(r.idx)
	}
	return r.idx
}

func (m *Model) buildRows() []row {
	rs := m.rows[:0]
	for i := range sim.N {
		if m.s.IsUnlocked(sim.UnlockBusiness0 + sim.Unlock(i)) {
			rs = append(rs, row{kind: rowBusiness, idx: i})
			continue
		}
		rs = append(rs, row{kind: rowLocked, idx: i})
		break
	}
	rs = append(rs, row{kind: rowHeader, idx: -1})
	for u := range m.b.Upgrades {
		if !m.s.Upgrades.Has(u) {
			rs = append(rs, row{kind: rowUpgrade, idx: u})
		}
	}
	for u := range m.b.Upgrades {
		if m.s.Upgrades.Has(u) {
			rs = append(rs, row{kind: rowUpgrade, idx: u})
		}
	}
	m.rows = rs
	return rs
}

func (m *Model) selIndex(rs []row) int {
	for i, r := range rs {
		if r == m.sel {
			return i
		}
	}
	if m.sel.kind == rowBusiness || m.sel.kind == rowLocked {
		for i, r := range rs {
			if (r.kind == rowBusiness || r.kind == rowLocked) && r.idx == m.sel.idx {
				return i
			}
		}
	}
	return 0
}

func (m *Model) moveSel(d int) {
	rs := m.buildRows()
	i := m.selIndex(rs)
	for j := i + d; j >= 0 && j < len(rs); j += d {
		if rs[j].selectable() {
			m.sel = rs[j]
			break
		}
	}
	m.ensureVisible()
}

func (m *Model) selectRow(i int) {
	rs := m.buildRows()
	if i >= 0 && i < len(rs) && rs[i].selectable() {
		m.sel = rs[i]
	}
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	rs := m.buildRows()
	h := m.listHeight()
	if h <= 0 {
		return
	}
	i := m.selIndex(rs)
	if i < m.scroll {
		m.scroll = i
	}
	if i >= m.scroll+h {
		m.scroll = i - h + 1
	}
	m.scroll = max(0, min(m.scroll, len(rs)-h))
}

func (m *Model) followUpgrade(u int) {
	if m.sel != (row{kind: rowUpgrade, idx: u}) {
		return
	}
	next := -1
	for v := range m.b.Upgrades {
		if !m.s.Upgrades.Has(v) {
			next = v
			if v > u {
				break
			}
		}
	}
	if next >= 0 {
		m.sel = row{kind: rowUpgrade, idx: next}
	}
	m.ensureVisible()
}

func (m *Model) listHeight() int { return m.height - topRows - bottomRows }

type advice struct {
	ok   bool
	kind sim.PurchaseKind
	idx  int
	qty  int
}

func (m *Model) effRate(h *sim.Hypo, mgr *[sim.N]bool, d *sim.Derived) float64 {
	sim.Evaluate(m.b, h, d)
	r := 0.0
	for i := range sim.N {
		if h.Owned[i] == 0 {
			continue
		}
		if mgr[i] {
			r += d.PerSec[i]
		} else {
			c := d.CycleS[i]
			r += d.PerSec[i] * c / (c + manualDelayS)
		}
	}
	return r
}

func payback(cost, gain, money, rate float64) float64 {
	if !(gain > 0) {
		return math.Inf(1)
	}
	wait := 0.0
	if cost > money {
		if !(rate > 0) {
			return math.Inf(1)
		}
		wait = (cost - money) / rate
	}
	return wait + cost/gain
}

func (m *Model) refreshAdvisor() {
	m.advAt = m.s.Clock
	s, b := m.s, m.b
	var mgr [sim.N]bool
	for i := range sim.N {
		mgr[i] = s.Businesses[i].Manager
	}
	base := sim.HypoOf(s)
	var d sim.Derived
	r0 := m.effRate(&base, &mgr, &d)
	rate := m.rate()
	best := math.Inf(1)
	m.adv = advice{}
	consider := func(kind sim.PurchaseKind, idx, qty int, cost, gain float64) {
		if sc := payback(cost, gain, s.Money, rate); sc < best {
			best = sc
			m.adv = advice{ok: true, kind: kind, idx: idx, qty: qty}
		}
	}
	for i := range sim.N {
		if !s.IsUnlocked(sim.UnlockBusiness0 + sim.Unlock(i)) {
			continue
		}
		h := base
		h.Owned[i]++
		consider(sim.PurchaseBusiness, i, 1, sim.BusinessCost(s, b, i, 1), m.effRate(&h, &mgr, &d)-r0)
		if s.Businesses[i].Owned > 0 && !mgr[i] {
			mm := mgr
			mm[i] = true
			consider(sim.PurchaseManager, i, 0, b.ManagerCost(i), m.effRate(&base, &mm, &d)-r0)
		}
	}
	for u := range b.Upgrades {
		if s.Upgrades.Has(u) {
			continue
		}
		h := base
		h.Upgrades.Set(u)
		gain := m.effRate(&h, &mgr, &d) - r0
		if b.Upgrades[u].Effect == balance.EffectCost {
			gain = r0 * (1 - b.Upgrades[u].Value)
		}
		consider(sim.PurchaseUpgrade, u, 0, b.Upgrades[u].Cost, gain)
	}
}

func (m *Model) advisedRow(r row) bool {
	if !m.adv.ok || !m.s.Settings.Advisor {
		return false
	}
	switch m.adv.kind {
	case sim.PurchaseBusiness, sim.PurchaseManager:
		return r.kind == rowBusiness && r.idx == m.adv.idx
	case sim.PurchaseUpgrade:
		return r.kind == rowUpgrade && r.idx == m.adv.idx
	}
	return false
}
