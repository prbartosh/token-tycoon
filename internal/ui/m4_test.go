package ui

import (
	"strings"
	"testing"

	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
)

func m4Game(c *Config) {
	lateGame(c)
	s, b := c.State, c.Balance
	s.Lifetime.Total = 1e45
	s.Lifetime.Layer2Total = 1e45
	s.Money = 1e44
	s.Prestige.Capital = 40
	s.Training.Models = []sim.Model{{Name: "TT-1", Params: 1e9, Benchmark: 75}}
	s.Talent = append(s.Talent, sim.Researcher{Class: "star", Name: "Ada", Trait: "frugal"}, sim.Researcher{Class: "junior", Name: "Jan"})
	s.Talent[1].RaiseUntilS = 1000
	s.Offers = append(s.Offers, sim.Contract{ID: 3, Volume: 1e30, MinBenchmark: 50, DurationS: 1200, RewardMinutes: 30, ExpiresS: 900},
		sim.Contract{ID: 4, Volume: 1e31, MinBenchmark: 90, DurationS: 600, RewardMinutes: 40, ExpiresS: 900})
	s.Contracts = append(s.Contracts, sim.Contract{ID: 2, Volume: 1e32, Delivered: 3e31, MinBenchmark: 40, DurationS: 1800, DeadlineS: 1500, RewardMinutes: 25})
	s.ContractSeq = 4
	eff1, _ := b.NodeIndex("eff_1")
	s.Research.Done.Set(eff1)
	s.Research.Active, _ = b.NodeIndex("eff_2")
	s.Research.Progress = 0.4
	s.Rules = append(s.Rules,
		sim.Rule{If: [sim.MaxConds]sim.Cond{{Metric: "gpu_util", Op: ">", Value: 0.9}, {Metric: "training_idle", Op: "<", Value: 0.5}},
			Then: sim.Action{Kind: "buy_rack", Qty: sim.QtyMax}, Enabled: true, Fired: 3, LastS: 10},
		sim.Rule{If: [sim.MaxConds]sim.Cond{{Metric: "income", Op: "<", Value: 1e30}}, Then: sim.Action{Kind: "buy_business", Index: 2, Qty: 10}},
		sim.Rule{If: [sim.MaxConds]sim.Cond{{Metric: "money_minutes", Op: ">", Value: 5}}, Then: sim.Action{Kind: "set_free", Value: 0.2}, Enabled: true})
	s.Poach = sim.Poach{Active: true, Index: 1, EndsAtS: 1e9}
	sim.Normalize(s, b)
}

func (h *harness) clean(where string) {
	h.t.Helper()
	h.fits()
	if v := h.view(); missingKey.MatchString(v) || strings.Contains(v, "%!") {
		h.t.Fatalf("untranslated text %s:\n%s", where, v)
	}
}

func TestM4TabsAndModalsRender(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.EN, i18n.PL} {
		for _, sz := range [][2]int{{120, 40}, {80, 24}} {
			h := newHarness(t, lang, sz[0], sz[1], m4Game)
			h.m.addRuleLog("[1m] " + h.m.ruleFired(&sim.Event{Kind: sim.EvRuleFired, Index: 0}))
			for _, k := range []string{"2", "3", "4", "5", "6", "7"} {
				h.key(k)
				h.dump(lang.String() + " tab " + k)
				for range 80 {
					h.clean("tab " + k)
					h.key("down")
				}
			}
			h.key("P")
			h.dump("funding")
			h.clean("funding modal")
			h.key("esc")
			h.m.event(&sim.Event{Kind: sim.EvFunding, Index: 12, Amount: 30, Value: 1e50}, false)
			h.clean("ceremony")
			h.key("enter")
			h.key("6")
			h.m.openWizard()
			for step := 0; h.m.modal.kind == modalWizard && step < 12; step++ {
				for range 30 {
					h.clean("wizard")
					h.key("down")
				}
				h.key("home", "enter")
			}
			h.m.modal = modal{}
		}
	}
}

func TestTalentUI(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.contains("Jan", "o counter-offer")
	h.key("o")
	if h.m.s.Poach.Active || len(h.m.s.Talent) != 2 || !(h.m.s.Talent[1].RaiseUntilS > h.m.s.Clock) {
		t.Fatalf("counter-offer from key: %+v", h.m.s.Talent)
	}
	h.key("2")
	for h.m.tabCurrent().act != actHire {
		h.key("down")
	}
	h.contains("Enter hire")
	h.key("enter")
	if len(h.m.s.Talent) != 3 {
		t.Fatal("hire from Models tab")
	}
	for h.m.tabCurrent().act != actStaff {
		h.key("down")
	}
	h.key("x")
	if len(h.m.s.Talent) != 2 {
		t.Fatal("let go")
	}
}

func TestContractsUI(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.key("3")
	h.contains("ENTERPRISE CONTRACTS 1/3", "offer #3")
	for h.m.tabCurrent().act != actOffer {
		h.key("down")
	}
	h.key("enter")
	if len(h.m.s.Contracts) != 2 || len(h.m.s.Offers) != 1 {
		t.Fatalf("accept: %d %d", len(h.m.s.Contracts), len(h.m.s.Offers))
	}
	for h.m.tabCurrent().act != actOffer {
		h.key("down")
	}
	h.key("x")
	if len(h.m.s.Offers) != 0 {
		t.Fatal("decline")
	}
}

func TestResearchAndFundingUI(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.key("5")
	h.contains("VC capital 40", "Researching Fused kernels II", "EFFICIENCY", "No round yet")
	scale1, _ := h.b.NodeIndex("scale_1")
	h.m.s.Research.Active = -1
	h.m.tsel[tabResearch] = tkey{act: actNode, idx: scale1}
	h.key("enter")
	if h.m.s.Research.Active != scale1 || h.m.s.Prestige.Capital != 40-h.b.ResearchNodes[scale1].Cost {
		t.Fatalf("research from tab: %+v", h.m.s.Research)
	}
	h.key("P")
	h.contains("FUNDING ROUND", "Valuation")
	h.key("enter")
	if h.m.s.Prestige.ResetsL2 != 1 || h.m.modal.kind != modalFunded {
		t.Fatalf("funding: %+v modal %v", h.m.s.Prestige, h.m.modal.kind)
	}
	h.contains("ROUND CLOSED", "Pre-seed")
	if h.m.desired() != fpsInterval(h.b.Engine.FPSAnimation) {
		t.Fatal("valuation counter does not animate")
	}
	h.key("enter")
	h.key("P")
	h.contains("Requires")
	h.key("enter")
	if h.m.s.Prestige.ResetsL2 != 1 {
		t.Fatal("locked round went through")
	}
}

func TestRulesWizardUI(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.m.s.Rules = h.m.s.Rules[:1]
	h.key("6")
	h.contains("POLICIES", "IF GPU use > 90% AND not training idle THEN buy GPU rack MAX")
	for h.m.tabCurrent().act != actRuleNew {
		h.key("down")
	}
	h.key("enter")
	if h.m.modal.kind != modalWizard {
		t.Fatal("wizard not opened")
	}
	h.contains("NEW POLICY", "Step 1")
	h.key("enter")
	h.key("down", "down", "down", "enter")
	h.key("enter")
	h.key("enter")
	h.key("down", "enter")
	h.contains("Save this policy?")
	h.key("enter")
	if len(h.m.s.Rules) != 2 {
		t.Fatalf("rule not saved: %+v", h.m.s.Rules)
	}
	r := h.m.s.Rules[1]
	if r.If[0].Metric != "money_x_best" || r.If[0].Value != 10 || r.If[1].Metric != "" || r.Then.Kind != "buy_best" || r.Then.Qty != 10 || !r.Enabled {
		t.Fatalf("wizard rule %+v", r)
	}
	h.m.tsel[tabRules] = tkey{act: actRule, idx: 1}
	h.key("+")
	if h.m.s.Rules[0].Then.Kind != "buy_best" {
		t.Fatal("priority up")
	}
	h.key("enter")
	if h.m.s.Rules[0].Enabled {
		t.Fatal("toggle")
	}
	h.key("x")
	if len(h.m.s.Rules) != 1 {
		t.Fatal("delete")
	}
	h.m.event(&sim.Event{Kind: sim.EvRuleFired, Index: 0}, true)
	h.contains("policy 1: buy GPU rack MAX")
}

func TestWizardBackRestoresCondition(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.m.s.Rules = h.m.s.Rules[:1]
	h.key("6")
	h.m.openWizard()
	h.key("enter", "enter")
	h.key("down", "enter")
	if h.m.wiz.step != wzValue || h.m.wiz.cond != 1 {
		t.Fatalf("second condition: %+v", h.m.wiz)
	}
	h.key("backspace", "backspace")
	if h.m.wiz.step != wzValue || h.m.wiz.cond != 0 {
		t.Fatalf("back to first value: %+v", h.m.wiz)
	}
	h.key("down", "enter")
	if h.m.wiz.step != wzMore || h.m.wiz.rule.If[0].Value != 2 {
		t.Fatalf("first value not edited: %+v", h.m.wiz)
	}
}

func TestReactionKeysWorkInsideModal(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.m.s.Incident = sim.Incident{ID: h.m.b.Incidents[0].ID, EndsAtS: h.m.s.Clock + 5}
	h.m.openWizard()
	h.key("r")
	if h.m.s.Incident.ID != "" || h.m.modal.kind != modalWizard {
		t.Fatalf("incident %+v modal %v", h.m.s.Incident, h.m.modal.kind)
	}
	h.key("o")
	if h.m.s.Poach.Active || h.m.modal.kind != modalWizard {
		t.Fatalf("counter-offer inside modal: %+v", h.m.s.Poach)
	}
}

func TestIncidentKeyFollowsBalance(t *testing.T) {
	h := newHarness(t, i18n.EN, 120, 40, m4Game)
	h.m.b.Safety.Key = "v"
	h.m.s.Incident = sim.Incident{ID: h.m.b.Incidents[0].ID, EndsAtS: h.m.s.Clock + 5}
	h.key("r")
	if h.m.s.Incident.ID == "" {
		t.Fatal("r answered an incident bound to v")
	}
	h.key("v")
	if h.m.s.Incident.ID != "" {
		t.Fatal("v did not answer the incident")
	}
}
