package ui

import (
	"strconv"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/market"
	"token-tycoon/internal/sim"
	"token-tycoon/internal/ui/anim"
)

func (m *Model) eventName(i int) string {
	if i < 0 || i >= len(m.b.Event) {
		return ""
	}
	return m.cat.T(i18n.EventKey(m.b.Event[i].ID))
}

func (m *Model) eventDesc(i int, left float64) string {
	ev := &m.b.Event[i]
	key := i18n.EventDescKey(ev.ID)
	var args []any
	switch ev.Effect {
	case balance.EventReaction:
		args = []any{m.dur(left), fixed(ev.RewardMinutes, 0)}
	case balance.EventRackCost, balance.EventHype:
		args = []any{m.dur(left)}
	default:
		args = []any{fixed(ev.Value, 0), m.dur(left)}
	}
	n := countVerbs(m.cat.T(key))
	return m.cat.T(key, args[:min(n, len(args))]...)
}

func countVerbs(s string) int {
	n := 0
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		if s[i+1] == '%' {
			i++
			continue
		}
		n++
	}
	return n
}

func (m *Model) cardKind(i int) anim.Kind {
	switch balance.Rarities[m.b.CardRarity(i)] {
	case "rare":
		return anim.KindRare
	case "epic":
		return anim.KindEpic
	case "legendary":
		return anim.KindLegend
	}
	return anim.KindReveal
}

func (m *Model) modelName(i int) string {
	if i < 0 || i >= len(m.s.Training.Models) {
		return ""
	}
	return m.s.Training.Models[i].Name
}

func (m *Model) energyName(k int) string {
	switch k {
	case sim.EnergyPPA:
		return m.cat.T("energy.ppa")
	case sim.EnergySMR:
		return m.cat.T("energy.smr")
	case sim.EnergyOrbital:
		return m.cat.T("energy.orbital")
	}
	return m.cat.T("energy.grid")
}

func (m *Model) systemEvent(e *sim.Event, quiet bool) {
	c, a := m.cat, &m.anim
	msg, kind, toast := "", anim.KindInfo, false
	switch e.Kind {
	case sim.EvRack:
		msg = c.T("msg.rack", e.Count, e.Index, m.money(e.Amount))
	case sim.EvRackSold:
		msg = c.T("msg.rack_sold", e.Count, e.Index, m.money(e.Amount))
	case sim.EvEnergy:
		msg = c.T("msg.energy", m.energyName(e.Index), m.num(e.Value), m.money(e.Amount))
	case sim.EvPPAExpired:
		msg, kind, toast = c.T("msg.ppa_expired", m.num(e.Value)), anim.KindWarn, true
	case sim.EvCloud:
		msg = c.T(onOff("msg.cloud", e.Count == 1))
	case sim.EvStarter:
		msg, kind, toast = c.T("msg.starter", e.Count, m.num(e.Value)), anim.KindMilestone, true
	case sim.EvGPUGen:
		msg, kind, toast = c.T("msg.gpu_gen", e.Index), anim.KindMilestone, true
		if m.tab != tabInfra {
			m.newTab[tabInfra] = true
		}
	case sim.EvPrestige:
		msg, kind, toast = c.T("msg.prestige", sim.VersionName(m.b, e.Index), m.num(e.Amount)), anim.KindCrit, true
		m.draft = trainDraft{}
		m.anim.Money.Jump(m.s.Money)
		m.spark, m.sparkN, m.sparkAcc, m.sparkSteps = [sparkLen]float64{}, 0, 0, 0
		m.sel = row{kind: rowBusiness}
		m.scroll = 0
	case sim.EvCard:
		name := c.T(i18n.CardKey(m.b.Card[e.Index].ID))
		if e.Count > 1 {
			msg = c.T("msg.card_level", name, min(e.Count, m.b.Cards.MaxLevel))
		} else {
			msg = c.T("msg.card", name, c.T(i18n.RarityKey(balance.Rarities[m.b.CardRarity(e.Index)])))
		}
		kind, toast = m.cardKind(e.Index), true
	case sim.EvCardSet:
		msg, kind, toast = c.T("msg.card_set", c.T(i18n.RarityKey(balance.Rarities[e.Index]))), anim.KindLegend, true
	case sim.EvEventStart:
		msg, kind, toast = m.eventName(e.Index)+": "+m.eventDesc(e.Index, e.Value), anim.KindMilestone, true
	case sim.EvEventEnd:
		msg = c.T("msg.event_end", m.eventName(e.Index))
	case sim.EvReact:
		msg, kind, toast = c.T("msg.react_ok", m.eventName(e.Index), m.money(e.Amount)), anim.KindCrit, true
	case sim.EvReactMiss:
		msg, kind, toast = c.T("msg.react_miss", m.eventName(e.Index), m.money(e.Amount)), anim.KindWarn, true
		if !(e.Amount > 0) {
			msg, toast = c.T("msg.event_end", m.eventName(e.Index)), false
		}
	case sim.EvGrant:
		msg, kind, toast = c.T("msg.grant", e.Count, m.money(e.Amount)), anim.KindCrit, true
	case sim.EvRivalRelease:
		msg = c.T("msg.rival_release", c.T(i18n.RivalKey(m.b.Rivals[e.Index].ID)), fixed(e.Value, 1))
		toast = m.tab == tabMarket
	case sim.EvSegments:
		mv := &m.s.Derived.Market
		m.setStatus(c.T("msg.segments", pct(mv.Alloc[market.SegFree]), pct(mv.Alloc[market.SegPro]), pct(mv.Alloc[market.SegEnterprise])), false)
		return
	case sim.EvAssign:
		if e.Count < 0 {
			msg = c.T("msg.unassign", m.modelName(e.Index))
		} else {
			msg = c.T("msg.assign", m.modelName(e.Index), c.T("segment."+market.SegmentNames[e.Count]))
		}
	case sim.EvModelRetired:
		msg = c.T("msg.model_retired", m.retiring)
	case sim.EvModelRelease:
		name := m.modelName(e.Index)
		msg, kind, toast = c.T("msg.model_release", name, fixed(e.Value, 1), e.Count, pct(e.Amount)), anim.KindCrit, true
		if !quiet {
			m.modal = modal{kind: modalRelease, name: name, bench: e.Value, share: e.Amount, rank: e.Count}
		}
	case sim.EvTrainQueued:
		msg = c.T("msg.train_queued", m.num(e.Value), m.num(e.Amount))
	case sim.EvTrainCancel:
		msg = c.T("msg.train_cancel", m.num(e.Value))
	case sim.EvTrainShare:
		m.setStatus(c.T("msg.train_share", pct(e.Value)), false)
		return
	case sim.EvSynthetic:
		m.setStatus(c.T("msg.synthetic", pct(e.Value)), false)
		return
	case sim.EvScraper:
		msg = c.T("msg.scraper", e.Count, m.money(e.Amount))
	case sim.EvLicense:
		msg = c.T("msg.license", m.num(e.Value), m.money(e.Amount))
	case sim.EvScraperIncident:
		msg, kind, toast = c.T("msg.scraper_incident", m.money(e.Amount), m.dur(e.Value)), anim.KindWarn, true
	default:
		return
	}
	m.addLog(msg)
	if quiet {
		return
	}
	if toast {
		a.Toast.Push(msg, kind, toastLife)
	} else {
		m.setStatus(msg, false)
	}
}

func (m *Model) rejectText(e *sim.Event) string {
	c := m.cat
	switch e.Reason {
	case sim.RejectLocked:
		if e.Cmd == sim.CmdPrestige {
			return c.T("prestige.locked", m.money(e.Amount))
		}
		if e.Business >= 0 && e.Business < sim.N {
			return c.T("msg.locked", m.bizName(e.Business), m.money(e.Amount))
		}
		return c.T("ui.msg.locked_need", m.money(e.Amount))
	case sim.RejectNoEvent:
		return c.T("msg.no_event")
	case sim.RejectAlreadyClaimed:
		return c.T("msg.grant_claimed")
	case sim.RejectBadDate:
		return c.T("msg.bad_date")
	case sim.RejectNoData:
		return c.T("msg.no_data", m.num(e.Amount))
	case sim.RejectQueueFull:
		return c.T("msg.queue_full")
	case sim.RejectTooLarge:
		return c.T("msg.too_large", m.num(e.Amount))
	case sim.RejectNeedBenchmark:
		return c.T("msg.need_benchmark", strconv.FormatFloat(e.Amount, 'f', 0, 64))
	}
	return ""
}
