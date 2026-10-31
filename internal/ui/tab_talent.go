package ui

import (
	"math"
	"strconv"
	"strings"

	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
	"github.com/prbartosh/token-tycoon/internal/ui/widgets"
)

func (m *Model) traitName(id string) string {
	if id == "" {
		return ""
	}
	return m.cat.T("talent.trait." + id)
}

func (m *Model) talentRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	tv := &s.Derived.Talent
	t.blank()
	m.section(t, c.T("ui.sec.talent", len(s.Talent), b.Talent.MaxStaff))
	t.text(th.Dim.Render(c.T("ui.talent.summary", pct(tv.Mult-1), pct(tv.Salary))))
	for i := range b.TalentClasses {
		cl := &b.TalentClasses[i]
		desc := c.T("ui.talent.class", pct(cl.SalaryShare), pct(cl.Speed))
		if cl.Trait {
			desc += " " + c.T("ui.talent.with_trait")
		}
		t.item(actHire, i, th.Text.Render(m.cell(c.T("ui.talent.hire", c.T(i18n.TalentKey(cl.ID))), 18)+" ")+
			th.Dim.Render(m.cell(desc, 34))+m.costText(sim.HireCost(s, b, i)))
	}
	for i := range s.Talent {
		r := &s.Talent[i]
		line := th.Bold.Render(m.cell(r.Name, 10)) + th.Text.Render(m.cell(c.T(i18n.TalentKey(r.Class)), 10))
		if r.Trait != "" {
			line += th.Magenta.Render(m.cell(m.traitName(r.Trait), 16))
		} else {
			line += m.cell("", 16)
		}
		switch {
		case s.Poach.Active && s.Poach.Index == i:
			line += th.Bad.Render(c.T("ui.talent.poached", m.dur(sim.PoachLeft(s))))
		case r.RaiseUntilS > s.Clock:
			line += th.Dim.Render(c.T("ui.talent.raise", m.dur(r.RaiseUntilS-s.Clock)))
		}
		t.item(actStaff, i, line)
	}
}

func (m *Model) contractRows(t *tabList) {
	th, c, s, b := m.th, m.cat, m.s, m.b
	t.blank()
	m.section(t, c.T("ui.sec.contracts", len(s.Contracts), sim.ContractSlots(s, b)))
	t.text(th.Dim.Render(c.T("ui.contracts.summary", fixed(s.Market.Reputation, 0), pct(s.Market.Reputation*b.Contracts.RepRewardBonus),
		m.num(sim.SteadyTokensPerSec(s)), fixed(sim.BestBenchmark(s, b), 1))))
	for i := range s.Contracts {
		ct := &s.Contracts[i]
		var bar strings.Builder
		widgets.Track(&bar, ct.Delivered/ct.Volume, 12, th.Rate, th.Track)
		left := math.Max(0, ct.DeadlineS-s.Clock)
		t.item(actContract, i, th.Gold.Render(m.cell("#"+strconv.Itoa(ct.ID), 6))+bar.String()+
			th.Text.Render(" "+m.cell(pct(ct.Delivered/ct.Volume), 6))+
			th.Dim.Render(m.cell(c.T("ui.contracts.left", m.dur(left)), 16))+
			th.Money.Render(c.T("ui.contracts.reward", m.money(sim.ContractReward(s, b, ct)))))
	}
	if len(s.Offers) == 0 {
		t.text(th.Dim.Render(c.T("ui.contracts.none")))
	}
	for i := range s.Offers {
		o := &s.Offers[i]
		ok := sim.BestBenchmark(s, b) >= o.MinBenchmark
		st := th.Text
		if !ok {
			st = th.Dim
		}
		need := 0.0
		if p := sim.SteadyTokensPerSec(s) * o.DurationS; p > 0 {
			need = o.Volume / p
		}
		t.item(actOffer, i, st.Render(m.cell(c.T("ui.contracts.offer", o.ID), 10))+
			st.Render(m.cell(c.T("ui.contracts.volume", m.num(o.Volume), m.dur(o.DurationS)), 26))+
			st.Render(m.cell(c.T("ui.contracts.need", fixed(need, 1)), 15)+" ")+
			st.Render(m.cell(c.T("ui.contracts.min_bench", fixed(o.MinBenchmark, 0)), 7)+" ")+
			th.Money.Render(c.T("ui.contracts.reward", m.money(sim.ContractReward(s, b, o)))))
	}
}
