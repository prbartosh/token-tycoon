package ui

import (
	"strconv"

	"github.com/prbartosh/token-tycoon/internal/format"
	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/anim"
)

const (
	flashPurchase  = 0.12
	flashMilestone = 0.3
	flashCrit      = 0.3
	flashWarn      = 0.25
	floatLife      = 0.3
	critHoldS      = 4.0
	floatGap       = 0.25
	floatRise      = 3
)

func upgradeSlot(u int) int { return sim.N + u }

func (m *Model) consume(evs []sim.Event, quiet bool) {
	for i := range evs {
		m.event(&evs[i], quiet)
	}
}

func (m *Model) event(e *sim.Event, quiet bool) {
	a := &m.anim
	switch e.Kind {
	case sim.EvUnlock:
		u := sim.Unlock(e.Index)
		for t := range numTabs {
			if (tabUnlock[t] == u || extraUnlock(t, u)) && t != m.tab {
				m.newTab[t] = true
			}
		}
		msg := m.cat.T("msg.unlock", m.unlockName(u))
		m.addLog(msg)
		if !quiet {
			a.Toast.Push(msg, anim.KindMilestone, m.toastS(msg))
		}
	}
	if e.Kind >= sim.EvRack {
		m.systemEvent(e, quiet)
		return
	}
	if quiet {
		return
	}
	switch e.Kind {
	case sim.EvCycle:
		if e.Business < 0 || m.s.Businesses[e.Business].Manager {
			return
		}
		if m.s.Clock-m.lastFloat[e.Business] < floatGap {
			return
		}
		m.lastFloat[e.Business] = m.s.Clock
		a.Float.Spawn(e.Business, "+"+m.num(e.Amount), anim.KindMoney, floatLife, floatRise)
	case sim.EvCrit:
		txt := m.cat.T("msg.crit", format.Number(m.b.Crit.Mult, format.Letters))
		a.Flash.Start(e.Business, anim.KindCrit, flashCrit)
		txt += " +" + m.num(e.Amount)
		a.Float.Replace(e.Business, txt, anim.KindCrit, critHoldS+readS(txt), 0)
		m.addLog(m.bizName(e.Business) + ": " + txt)
	case sim.EvPurchase:
		a.Flash.Start(e.Business, anim.KindPurchase, flashPurchase)
		m.addLog(m.cat.T("msg.purchase", m.bizName(e.Business), e.Count, m.money(e.Amount)))
	case sim.EvManager:
		id := m.b.Businesses[e.Business].ID
		msg := m.cat.T("msg.manager", m.cat.T(i18n.ManagerKey(id)), m.cat.T(i18n.ManagerLineKey(id)))
		a.Flash.Start(e.Business, anim.KindPurchase, flashPurchase)
		a.Toast.Push(msg, anim.KindInfo, m.toastS(msg))
		m.addLog(msg)
	case sim.EvUpgrade:
		msg := m.cat.T("msg.upgrade", m.upgName(e.Index))
		a.Flash.Start(upgradeSlot(e.Index), anim.KindPurchase, flashPurchase)
		a.Toast.Push(msg, anim.KindInfo, m.toastS(msg))
		m.followUpgrade(e.Index)
		m.addLog(msg)
	case sim.EvMilestoneSpeed, sim.EvMilestoneProfit:
		key := "msg.milestone.speed"
		if e.Kind == sim.EvMilestoneProfit {
			key = "msg.milestone.profit"
		}
		msg := m.cat.T(key, m.bizName(e.Business), e.Index, m.num(e.Amount))
		a.Flash.Start(e.Business, anim.KindMilestone, flashMilestone)
		a.Toast.Push(msg, anim.KindMilestone, m.toastS(msg))
		m.addLog(msg)
	case sim.EvMilestoneGlobal:
		msg := m.cat.T("msg.milestone.global", e.Index, m.num(e.Amount))
		for i := range sim.N {
			a.Flash.Start(i, anim.KindMilestone, flashMilestone)
		}
		a.Toast.Push(msg, anim.KindMilestone, m.toastS(msg))
		m.addLog(msg)
	case sim.EvUndo:
		m.setStatus(m.cat.T("msg.undo", m.undoName(e), m.money(e.Amount)), false)
		m.addLog(m.status)
	case sim.EvRejected:
		m.rejected(e)
	case sim.EvCombo:
		a.Pulse = 0.15
	}
}

func extraUnlock(t int, u sim.Unlock) bool {
	switch t {
	case tabInfra:
		return u == sim.UnlockData
	case tabStats:
		return u == sim.UnlockCards
	case tabModels:
		return u == sim.UnlockTalent
	case tabMarket:
		return u == sim.UnlockContracts || u == sim.UnlockRegions || u == sim.UnlockInvestors
	case tabResearch:
		return u == sim.UnlockLayer2 || u == sim.UnlockAGI
	}
	return false
}

func (m *Model) undoName(e *sim.Event) string {
	switch e.Purchase {
	case sim.PurchaseRack:
		return m.cat.T("infra.gen", e.Index) + " x" + strconv.Itoa(e.Count)
	case sim.PurchaseGrid:
		return m.cat.T("energy.grid") + " " + m.mw(e.Value)
	}
	switch {
	case e.Business >= 0 && e.Count > 0:
		return m.bizName(e.Business) + " x" + format.Number(float64(e.Count), format.Letters)
	case e.Business >= 0:
		return m.cat.T(i18n.ManagerKey(m.b.Businesses[e.Business].ID))
	case e.Index >= 0 && e.Index < len(m.b.Upgrades):
		return m.upgName(e.Index)
	}
	return ""
}

func (m *Model) rejected(e *sim.Event) {
	msg := m.rejectText(e)
	switch e.Reason {
	case sim.RejectCannotAfford:
		msg = m.cat.T("msg.cannot_afford", m.money(e.Amount))
	case sim.RejectLocked, sim.RejectNoEvent, sim.RejectAlreadyClaimed, sim.RejectBadDate,
		sim.RejectNoData, sim.RejectQueueFull, sim.RejectTooLarge, sim.RejectNeedBenchmark,
		sim.RejectTeamFull, sim.RejectNeedResearch, sim.RejectNoCapital, sim.RejectNeedRegion, sim.RejectFinished:
	case sim.RejectNothingToUndo:
		msg = m.cat.T("msg.undo_expired", format.Duration(m.b.Engine.UndoWindowS))
	case sim.RejectNotOwned:
		msg = m.cat.T("ui.msg.not_owned")
	case sim.RejectAlreadyOwned:
		msg = m.cat.T("ui.msg.already")
	case sim.RejectBusy:
		msg = m.cat.T("ui.msg.busy")
	default:
		msg = m.cat.T("ui.msg.invalid")
	}
	m.setStatus(msg, true)
	slot := e.Business
	if e.Cmd == sim.CmdBuyUpgrade {
		slot = upgradeSlot(e.Index)
	}
	m.anim.Flash.Start(slot, anim.KindWarn, flashWarn)
}
