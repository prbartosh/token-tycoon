package simulate

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
	"token-tycoon/internal/format"
	"token-tycoon/internal/i18n"
	"token-tycoon/internal/sim"
)

func WriteCSV(w io.Writer, r *Result, b *balance.Balance) error {
	cw := csv.NewWriter(w)
	head := []string{"t_s", "income_per_s", "money", "params", "capital", "benchmark", "share", "unlocks",
		"lifetime_total", "run", "managers", "upgrades", "combo", "online"}
	for i := range sim.N {
		head = append(head, "owned_"+b.Businesses[i].ID)
	}
	head = append(head, "throttle", "cards", "prestiges")
	if err := cw.Write(head); err != nil {
		return err
	}
	f := func(x float64) string { return strconv.FormatFloat(x, 'g', 6, 64) }
	row := make([]string, 0, len(head))
	for _, s := range r.Samples {
		row = append(row[:0], f(s.T), f(s.IncomePerS), f(s.Money), f(s.Params), f(s.Capital), f(s.Benchmark),
			f(s.Share), strconv.Itoa(s.Unlocks), f(s.Lifetime), f(s.Run), strconv.Itoa(s.Managers),
			strconv.Itoa(s.Upgrades), f(s.Combo), strconv.FormatBool(s.Online))
		for i := range sim.N {
			row = append(row, strconv.Itoa(s.Owned[i]))
		}
		row = append(row, f(s.Throttle), strconv.Itoa(s.Cards), strconv.Itoa(s.Prestiges))
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

type Check struct {
	Text string
	Pass bool
	NA   bool
}

func Checks(r *Result, b *balance.Balance, c *i18n.Catalog) []Check {
	pc := &b.Pacing
	var out []Check
	out = append(out, Check{
		Text: c.T("report.crit2.flat", format.Duration(pc.FlatMaxS), format.Duration(r.LongestFlat), format.Duration(r.LongestFlatAt)),
		Pass: r.LongestFlat <= pc.FlatMaxS,
	})
	play := r.Final.Stats.PlayS
	design := r.Config.Strategy == Greedy
	out = append(out, playWindow(c, "report.crit2.manager", r.ManagerPlayS, pc.FirstManagerS, 1, play))
	if design {
		out = append(out, playWindow(c, "report.crit2.infra", r.InfraPlayS, pc.InfraMin, 60, play))
		out = append(out, playWindow(c, "report.crit2.prestige", r.PrestigePlayS, pc.PrestigeMin, 60, play))
	}
	order := Check{Text: c.T("report.crit.order")}
	switch {
	case r.PrestigeT >= 0:
		order.Pass = r.InfraT >= 0 && r.InfraT <= r.PrestigeT
	default:
		order.NA = true
	}
	out = append(out, order)
	if len(r.Sessions) > 1 {
		ok := 0
		for _, s := range r.Sessions[1:] {
			if s.FirstUnlockT >= 0 && s.FirstUnlockT <= pc.SessionNewS {
				ok++
			}
		}
		out = append(out, Check{Text: c.T("report.crit2.session", format.Duration(pc.SessionNewS), ok, len(r.Sessions)-1), Pass: ok == len(r.Sessions)-1})
	}
	if !design {
		if r.Config.Strategy == Passive {
			out = append(out, dayWindow(c, "report.crit2.end_passive", r.EndingT, pc.PassiveEndDays, r.Config.Duration))
		}
		return out
	}
	days := int(math.Ceil(r.Config.Duration/86400 - 1e-9))
	if last := min(days, pc.GoalUntilDay); last > 0 {
		var missing []string
		for d := 1; d <= last; d++ {
			if !r.goalOnDay(d) {
				missing = append(missing, strconv.Itoa(d))
			}
		}
		out = append(out, Check{Text: c.T("report.crit2.goals", last, listOr(missing, c)), Pass: len(missing) == 0})
	}
	if bands := r.bandMisses(pc); bands.checked > 0 {
		out = append(out, Check{Text: c.T("report.crit2.bands", format.Number(pc.IncomeToleranceOOM, format.Letters), bands.checked, listOr(bands.misses, c)),
			Pass: len(bands.misses) == 0})
	}
	out = append(out, dayWindow(c, "report.crit2.funding", r.Layer2T, pc.FundingDays, r.Config.Duration))
	out = append(out, dayWindow(c, "report.crit2.agi", r.AGIT, pc.AGIDays, r.Config.Duration))
	return append(out, dayWindow(c, "report.crit2.end_greedy", r.EndingT, pc.GreedyEndDays, r.Config.Duration))
}

func listOr(xs []string, c *i18n.Catalog) string {
	if len(xs) == 0 {
		return c.T("report.none")
	}
	return strings.Join(xs, ", ")
}

func playWindow(c *i18n.Catalog, key string, t float64, win []float64, unit, playS float64) Check {
	lo, hi := win[0]*unit, win[1]*unit
	measured := c.T("report.not_reached")
	if t >= 0 {
		measured = format.Duration(t)
	}
	ch := Check{Text: c.T(key, format.Duration(lo), format.Duration(hi), measured)}
	switch {
	case t >= 0:
		ch.Pass = t >= lo && t <= hi
	case playS <= hi:
		ch.NA = true
	}
	return ch
}

func dayWindow(c *i18n.Catalog, key string, t float64, win []float64, duration float64) Check {
	measured := c.T("report.not_reached")
	if t >= 0 {
		measured = c.T("report.day_n", dayOf(t))
	}
	ch := Check{Text: c.T(key, int(win[0]), int(win[1]), measured)}
	switch {
	case t >= 0:
		d := float64(dayOf(t))
		ch.Pass = d >= win[0] && d <= win[1]
	case duration < win[1]*86400:
		ch.NA = true
	}
	return ch
}

func dayOf(t float64) int { return int(t/86400) + 1 }

func (r *Result) goalOnDay(d int) bool {
	for _, g := range r.Goals {
		if dayOf(g.T) == d {
			return true
		}
	}
	return false
}

type bandResult struct {
	checked int
	misses  []string
}

func (r *Result) bandMisses(pc *balance.Pacing) bandResult {
	var br bandResult
	full := int(r.Config.Duration/86400 + 1e-9)
	for i, inc := range r.DayIncome {
		d := i + 1
		if d > full {
			break
		}
		for _, band := range pc.IncomeBands {
			if float64(d) < band[0] || float64(d) > band[1] {
				continue
			}
			br.checked++
			lo, hi := band[2]-pc.IncomeToleranceOOM, band[3]+pc.IncomeToleranceOOM
			l := math.Inf(-1)
			if inc > 0 {
				l = math.Log10(inc)
			}
			if l < lo || l > hi {
				br.misses = append(br.misses, fmt.Sprintf("%d (%.0f)", d, math.Max(-1, l)))
			}
			break
		}
	}
	return br
}

func Report(w io.Writer, r *Result, b *balance.Balance, c *i18n.Catalog) int {
	n := format.Letters
	s := &r.Final
	cfg := r.Config
	p := func(format string, a ...any) { fmt.Fprintf(w, format, a...) }
	p("%s\n%s\n", c.T("report.title"), strings.Repeat("=", len([]rune(c.T("report.title")))))
	p("%s\n", c.T("report.strategy", cfg.Strategy, cfg.Seed, format.Duration(cfg.Duration)))
	if r.SessionsDay > 0 {
		p("%s\n", c.T("report.sessions2", r.SessionsDay, format.Duration(r.SessionS), format.Number(b.Offline.EfficiencyStart, n)))
		p("%s\n", c.T("report.offline_total", format.Duration(r.OfflineS), format.Number(r.OfflineEarned, n)))
	}
	if cfg.ComboRate > 0 {
		p("%s\n", c.T("report.combo", format.Number(cfg.ComboRate, n)))
	}
	p("%s\n", c.T("report.final", format.Number(s.Money, n), format.Rate(s.Derived.IncomePerSec, n),
		format.Number(s.Lifetime.Total, n), format.Number(s.Lifetime.Run, n)))
	p("%s\n", c.T("report.purchases", len(r.Purchases), r.Businesses, r.Managers, r.UpgradesN))
	var owned []string
	for i := range sim.N {
		bz := s.Businesses[i]
		if bz.Owned == 0 {
			continue
		}
		m := ""
		if bz.Manager {
			m = "*"
		}
		owned = append(owned, fmt.Sprintf("%s %d%s", c.T(i18n.BusinessKey(bz.ID)), bz.Owned, m))
	}
	p("%s\n", c.T("report.owned", strings.Join(owned, ", ")))
	p("%s\n", c.T("report.crits", s.Stats.Crits))
	p1 := &b.Prestige.Layer1
	p("%s\n", c.T("report.prestiges", len(r.Prestiges), format.Number(s.Prestige.Params, n),
		format.Number(econ.Layer1Bonus(s.Prestige.Params, p1.BonusPerParam), n), sim.VersionName(b, s.Prestige.VersionIndex)))
	for _, pr := range r.Prestiges {
		p("  %-10s %s\n", format.Duration(pr.T), c.T("report.prestige_line", format.Number(pr.Gain, n), format.Duration(pr.RunS), format.Duration(pr.ReturnS)))
	}
	sets := 0
	for _, done := range s.CardMods.SetsComplete {
		if done {
			sets++
		}
	}
	p("%s\n", c.T("report.cards", s.CardMods.Unique, len(b.Card), s.Stats.CardsDropped, sets))
	p("%s\n", c.T("report.events", s.Stats.EventsSeen, s.Stats.EventsCaught, s.Stats.EventsMissed))
	if s.IsUnlocked(sim.UnlockInfra) {
		var racks []string
		for g := range b.GPUGens {
			if s.Infra.Racks[g] > 0 {
				racks = append(racks, fmt.Sprintf("g%d x%d", g+1, s.Infra.Racks[g]))
			}
		}
		cloud := c.T("report.off")
		if s.Infra.CloudEnabled {
			cloud = c.T("report.on")
		}
		d := &s.Derived
		p("%s\n", c.T("report.infra_state", strings.Join(racks, ", "), format.Number(d.ComputeNeed, n), format.Number(d.ComputeCap, n),
			pct(d.Throttle), pct(d.EnergyShare), cloud))
		if r.InfraS > 0 {
			p("%s\n", c.T("report.infra_avg", pct(r.ThrottleS/r.InfraS), format.Duration(r.ThrottledS), format.Duration(r.CloudS)))
		}
	}
	if mv := &s.Derived.Market; mv.On {
		al := mv.Alloc
		p("%s\n", c.T("report.market", format.Number(s.Derived.Price, n), pct(mv.Share), mv.Rank,
			format.Number(mv.Hype, n), pct(al[0]), pct(al[1]), pct(al[2])))
		var rv []string
		for i := range mv.Rivals {
			rv = append(rv, fmt.Sprintf("%s %s", c.T("rival."+b.Rivals[i].ID), format.Number(mv.RivalBench[i], n)))
		}
		p("%s\n", c.T("report.rivals", strings.Join(rv, ", ")))
	}
	if s.IsUnlocked(sim.UnlockData) {
		tr := &s.Training
		p("%s\n", c.T("report.models", s.Stats.Models, format.Number(s.Stats.BestBench, n), format.Number(tr.DataTokens, n),
			format.Number(tr.SynthTokens, n), tr.Scrapers, tr.Licenses, s.Stats.Incidents))
		for _, md := range tr.Models {
			p("  %s\n", c.T("report.model_line", md.Name, format.Number(md.Params, n), format.Number(md.Tokens, n), format.Number(md.Benchmark, n)))
		}
		if r.TrainS > 0 {
			p("%s\n", c.T("report.training_time", format.Duration(r.TrainS), pct(r.TrainShareS/r.TrainS)))
		}
	}
	if s.IsUnlocked(sim.UnlockTalent) {
		p("%s\n", c.T("report.talent", len(s.Talent), s.Stats.Hired, s.Stats.Poached, s.Stats.Retained,
			pct(s.Derived.Talent.Mult-1), pct(s.Derived.Talent.Salary)))
	}
	if s.IsUnlocked(sim.UnlockContracts) {
		p("%s\n", c.T("report.contracts", s.Stats.ContractsDone, s.Stats.ContractsFailed, format.Number(s.Market.Reputation, n)))
	}
	if s.IsUnlocked(sim.UnlockLayer2) {
		p("%s\n", c.T("report.rounds", len(r.Rounds), format.Number(s.Prestige.CapitalLifetime, n), format.Number(s.Prestige.Capital, n)))
		for _, rd := range r.Rounds {
			p("  %-10s %s\n", format.Duration(rd.T), c.T("report.round_line", format.Number(rd.Capital, n), format.Number(rd.Valuation, n)))
		}
	}
	if s.IsUnlocked(sim.UnlockResearch) {
		p("%s\n", c.T("report.research", s.Research.Done.Count(), len(b.ResearchNodes)))
	}
	if s.IsUnlocked(sim.UnlockRules) {
		p("%s\n", c.T("report.rules", len(s.Rules), s.Stats.RulesFired))
	}
	if s.IsUnlocked(sim.UnlockSafety) {
		p("%s\n", c.T("report.incidents", s.Stats.IncidentsSeen, s.Stats.IncidentsOK, s.Stats.IncidentsMissed))
	}
	if s.IsUnlocked(sim.UnlockInvestors) {
		v := sim.Valuation(s, b)
		p("%s\n", c.T("report.investors", s.Stats.QuartersBeaten, s.Stats.Quarters, format.Number(v.Valuation, n), pct(v.Bonus)))
	}
	if s.IsUnlocked(sim.UnlockRegions) {
		p("%s\n", c.T("report.regions", sim.RegionsEntered(s, b), len(b.Regions), sim.RegionsCompliant(s, b),
			format.Number(s.Derived.Market.RegionMult, n)))
	}
	if s.IsUnlocked(sim.UnlockAGI) {
		p("%s\n", c.T("report.agi", min(s.AGI.Stage, len(b.AGIStages)), len(b.AGIStages), format.Number(s.Stats.AGIInvested, n), s.AGI.SILevel))
	}
	p("%s\n", c.T("report.achievements", s.Achieved.Count(), len(sim.Achievements)))
	if r.EndingT >= 0 {
		p("%s\n", c.T("report.ending_day", int(r.EndingT/86400)+1, format.Duration(r.EndingT)))
	} else {
		p("%s\n", c.T("report.ending_not", min(s.AGI.Stage, len(b.AGIStages)), len(b.AGIStages), format.Number(s.Lifetime.Total, n),
			format.Number(b.AGI.UnlockLifetime, n)))
	}
	p("%s\n\n", c.T("report.grants", s.Stats.Grants, s.Streak.Day))

	p("%s\n", c.T("report.timeline"))
	tl := func(label string, t float64) {
		v := c.T("report.not_reached")
		if t >= 0 {
			v = format.Duration(t)
		}
		p("  %-44s %s\n", label, v)
	}
	th := func(u sim.Unlock) string { return format.Number(sim.UnlockThreshold(u, b), format.Scientific) }
	tl(c.T("report.first_manager"), r.FirstManagerT)
	tl(c.T("report.infra", th(sim.UnlockInfra)), r.InfraT)
	tl(c.T("report.prestige", format.Number(b.Prestige.Layer1.UnlockRunRevenue, format.Scientific)), r.PrestigeT)
	first := -1.0
	if len(r.Prestiges) > 0 {
		first = r.Prestiges[0].T
	}
	tl(c.T("report.first_reset"), first)
	tl(c.T("report.market_unlock", th(sim.UnlockMarket)), r.MarketT)
	tl(c.T("report.training_unlock", th(sim.UnlockTraining)), r.TrainingT)
	firstModel := -1.0
	for _, md := range r.Models {
		if md.T >= 0 {
			firstModel = md.T
			break
		}
	}
	tl(c.T("report.first_model"), firstModel)
	tl(c.T("report.layer2_unlock", th(sim.UnlockLayer2)), r.Layer2T)
	firstRound := -1.0
	if len(r.Rounds) > 0 {
		firstRound = r.Rounds[0].T
	}
	tl(c.T("report.first_round"), firstRound)
	tl(c.T("report.regions_unlock", th(sim.UnlockRegions)), r.RegionsT)
	tl(c.T("report.agi_unlock", th(sim.UnlockAGI)), r.AGIT)
	for i, t := range r.AGIStages {
		if i < len(b.AGIStages) {
			tl(c.T("report.agi_stage", i+1, c.T(i18n.AGIKey(b.AGIStages[i].ID))), t)
		}
	}
	tl(c.T("report.ending"), r.EndingT)
	p("\n%s\n", c.T("report.unlocks"))
	for _, u := range r.Unlocks {
		p("  %-10s %s\n", format.Duration(u.T), unlockName(u.Unlock, b, c))
	}
	if len(r.DayIncome) > 0 {
		days := make([]string, 0, len(r.DayIncome))
		for i, inc := range r.DayIncome {
			l := "-"
			if inc > 0 {
				l = strconv.FormatFloat(math.Log10(inc), 'f', 0, 64)
			}
			days = append(days, fmt.Sprintf("%d:%s", i+1, l))
		}
		p("\n%s\n  %s\n", c.T("report.day_income"), strings.Join(days, " "))
	}
	p("\n%s\n", c.T("report.criteria2"))
	fails := 0
	for _, ch := range Checks(r, b, c) {
		mark := c.T("report.pass")
		switch {
		case ch.NA:
			mark = c.T("report.na")
		case !ch.Pass:
			mark = c.T("report.fail")
			fails++
		}
		p("  [%-4s] %s\n", mark, ch.Text)
	}
	if fails == 0 {
		p("\n%s\n", c.T("report.summary_ok"))
	} else {
		p("\n%s\n", c.T("report.summary_fail", fails))
	}
	return fails
}

func pct(x float64) string { return strconv.FormatFloat(x*100, 'f', 1, 64) + "%" }

func unlockName(u sim.Unlock, b *balance.Balance, c *i18n.Catalog) string {
	if u.IsBusiness() {
		return c.T("unlock.business", c.T(i18n.BusinessKey(b.Businesses[u.Business()].ID)))
	}
	return c.T(i18n.UnlockKey(u.Name()))
}
