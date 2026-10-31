package ui

import (
	"strconv"
	"strings"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/market"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/anim"
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
		if !quiet {
			m.startPrestige(e)
		}
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
			m.startRelease(e, name)
		}
	case sim.EvReleasePending:
		msg, kind, toast = c.T("msg.release_pending", m.pendingName(), fixed(e.Value, 1)), anim.KindMilestone, true
		m.later = false
		m.ring = true
		if !quiet {
			m.openChoice()
		}
	case sim.EvModelArchived:
		key := "msg.archived_new"
		if e.Count == sim.ArchivedDisplaced {
			key = "msg.archived_old"
		}
		msg = c.T(key, m.archivedName(e.Index), fixed(e.Value, 1))
		toast = e.Count == sim.ArchivedNew
	case sim.EvArchiveSwap:
		msg = c.T("msg.archive_swap", m.modelName(e.Index), fixed(e.Value, 1))
	case sim.EvArchiveDeleted:
		msg = c.T("msg.archive_deleted")
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
	case sim.EvHired:
		msg = c.T("msg.hired", m.staffName(e.Index), m.money(e.Amount))
		if e.Count >= 0 && e.Count < len(m.b.TalentTraits) {
			msg, kind, toast = c.T("msg.hired_star", m.staffName(e.Index), m.traitName(m.b.TalentTraits[e.Count].ID)), anim.KindEpic, true
		}
	case sim.EvFired:
		msg = c.T("msg.fired")
	case sim.EvPoach:
		msg, kind, toast = c.T("msg.poach", m.staffName(e.Index), m.dur(e.Value)), anim.KindWarn, true
	case sim.EvPoached:
		msg, kind, toast = c.T("msg.poached"), anim.KindWarn, true
	case sim.EvCounterOffer:
		msg, kind, toast = c.T("msg.counter_offer", m.staffName(e.Index), m.dur(e.Value)), anim.KindInfo, true
	case sim.EvContractOffer:
		msg = c.T("msg.contract_offer", e.Count)
		toast = m.tab == tabMarket
	case sim.EvContractAccepted:
		msg = c.T("msg.contract_accepted", e.Count, m.num(e.Amount), m.dur(e.Value))
	case sim.EvContractDeclined:
		msg = c.T("msg.contract_declined", e.Count)
	case sim.EvContractDone:
		msg, kind, toast = c.T("msg.contract_done", e.Count, m.money(e.Amount), fixed(e.Value, 0)), anim.KindCrit, true
	case sim.EvContractFailed:
		msg, kind, toast = c.T("msg.contract_failed", e.Count, fixed(e.Value, 0)), anim.KindWarn, true
	case sim.EvFunding:
		name := m.roundName(e.Index)
		msg, kind, toast = c.T("msg.funding", name, m.num(e.Amount)), anim.KindCrit, true
		m.draft = trainDraft{}
		m.anim.Money.Jump(m.s.Money)
		m.spark, m.sparkN, m.sparkAcc, m.sparkSteps = [sparkLen]float64{}, 0, 0, 0
		m.sel = row{kind: rowBusiness}
		m.scroll = 0
		if !quiet {
			m.startFunding(e, name)
		}
	case sim.EvResearchStart:
		msg = c.T("msg.research_start", m.nodeName(e.Index), m.num(e.Amount))
	case sim.EvResearchDone:
		msg, kind, toast = c.T("msg.research_done", m.nodeName(e.Index)), anim.KindMilestone, true
		if m.tab != tabResearch {
			m.newTab[tabResearch] = true
		}
	case sim.EvRuleAdded:
		msg = c.T("msg.rule_added", e.Index+1)
	case sim.EvRuleDeleted:
		msg = c.T("msg.rule_deleted", e.Index+1)
	case sim.EvRuleToggled:
		msg = c.T(onOff("msg.rule", e.Count == 1), e.Index+1)
	case sim.EvRuleMoved:
		m.setStatus(c.T("msg.rule_moved", e.Count+1, e.Index+1), false)
		return
	case sim.EvRuleFired:
		m.addRuleLog("[" + m.dur(m.s.Clock) + "] " + m.ruleFired(e))
		return
	case sim.EvAutoPrestige:
		msg = c.T("msg.auto_prestige", m.num(e.Amount))
	case sim.EvRegion:
		msg, kind, toast = c.T("msg.region", m.regionName(e.Index), m.money(e.Amount), fixed(e.Value, 2)), anim.KindMilestone, true
	case sim.EvCompliance:
		msg = c.T("msg.compliance", m.regionName(e.Index), m.money(e.Amount))
	case sim.EvIncident:
		m.ring = true
		msg, kind, toast = c.T("msg.incident", m.incidentName(e.Index), strings.ToUpper(m.b.Safety.Key), m.dur(e.Value)), anim.KindWarn, true
	case sim.EvIncidentOK:
		msg, kind, toast = c.T("msg.incident_ok", m.incidentName(e.Index), fixed(e.Value, 0)), anim.KindCrit, true
	case sim.EvIncidentMiss:
		msg, kind, toast = c.T("msg.incident_miss", m.incidentName(e.Index), fixed(e.Amount, 2)), anim.KindWarn, true
	case sim.EvQuarter:
		if e.Count == 1 {
			msg, kind, toast = c.T("msg.quarter_beat", m.money(e.Amount), m.money(e.Value), pct(m.s.Investors.Bonus)), anim.KindCrit, true
		} else {
			msg, kind, toast = c.T("msg.quarter_miss", m.money(e.Amount), m.money(e.Value)), anim.KindInfo, true
		}
	case sim.EvAGIInvest:
		m.setStatus(c.T("msg.agi_invest", m.money(e.Amount), m.agiName(e.Index), pct(e.Value)), false)
		return
	case sim.EvAGITraining:
		msg, kind, toast = c.T("msg.agi_training", m.dur(e.Value)), anim.KindMilestone, true
	case sim.EvAGIStage:
		msg, kind, toast = c.T("msg.agi_stage", m.agiName(e.Index), m.agiEffectText(e.Index)), anim.KindCrit, true
	case sim.EvAGIEnding:
		msg, kind, toast = c.T("msg.agi_ending", m.dur(e.Value)), anim.KindLegend, true
		if !quiet {
			m.modal = modal{kind: modalEnding}
		}
	case sim.EvSILevel:
		msg, kind, toast = c.T("msg.si_level", e.Index, m.dur(e.Value)), anim.KindLegend, true
	case sim.EvAchievement:
		msg, kind, toast = c.T("msg.achievement", c.T("ach."+sim.Achievements[e.Index].ID)), anim.KindCrit, true
		if m.tab != tabStats {
			m.newTab[tabStats] = true
		}
	default:
		return
	}
	m.addLog(msg)
	if quiet {
		return
	}
	if toast {
		a.Toast.Push(msg, kind, m.toastS(msg))
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
	case sim.RejectTeamFull:
		return c.T("msg.full")
	case sim.RejectNeedResearch:
		return c.T("msg.need_research")
	case sim.RejectNoCapital:
		return c.T("msg.no_capital", m.num(e.Amount))
	case sim.RejectNeedRegion:
		return c.T("msg.need_region")
	case sim.RejectFinished:
		return c.T("msg.agi_finished")
	}
	return ""
}

func (m *Model) staffName(i int) string {
	if i < 0 || i >= len(m.s.Talent) {
		return ""
	}
	return m.s.Talent[i].Name
}
