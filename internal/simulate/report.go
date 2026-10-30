package simulate

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"token-tycoon/internal/balance"
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

func Checks(r *Result, c *i18n.Catalog) []Check {
	dur := func(t float64) string {
		if t < 0 {
			return c.T("report.not_reached")
		}
		return format.Duration(t)
	}
	var out []Check
	out = append(out, Check{
		Text: c.T("report.crit.flat", format.Duration(r.LongestFlat), format.Duration(r.LongestFlatAt)),
		Pass: r.LongestFlat <= 120,
	})
	out = append(out, window(c.T("report.crit.manager", dur(r.FirstManagerT)), r.FirstManagerT, 30, 60, r.Config.Duration))
	out = append(out, window(c.T("report.crit.infra", dur(r.InfraT)), r.InfraT, 20*60, 30*60, r.Config.Duration))
	out = append(out, window(c.T("report.crit.prestige", dur(r.PrestigeT)), r.PrestigeT, 45*60, 90*60, r.Config.Duration))
	order := Check{Text: c.T("report.crit.order")}
	switch {
	case r.PrestigeT >= 0:
		order.Pass = r.InfraT >= 0 && r.InfraT <= r.PrestigeT
	default:
		order.NA = true
	}
	out = append(out, order)
	if len(r.Sessions) > 0 {
		ok := 0
		for _, s := range r.Sessions {
			if s.FirstUnlockT >= 0 && s.FirstUnlockT <= 300 {
				ok++
			}
		}
		out = append(out, Check{Text: c.T("report.crit.session", ok, len(r.Sessions)), Pass: ok == len(r.Sessions)})
	}
	return out
}

func window(text string, t, lo, hi, duration float64) Check {
	if t < 0 {
		if duration > hi {
			return Check{Text: text}
		}
		return Check{Text: text, NA: true}
	}
	return Check{Text: text, Pass: t >= lo && t <= hi}
}

func Report(w io.Writer, r *Result, b *balance.Balance, c *i18n.Catalog) int {
	n := format.Letters
	s := &r.Final
	cfg := r.Config
	p := func(format string, a ...any) { fmt.Fprintf(w, format, a...) }
	p("%s\n%s\n", c.T("report.title"), strings.Repeat("=", len([]rune(c.T("report.title")))))
	p("%s\n", c.T("report.strategy", cfg.Strategy, cfg.Seed, format.Duration(cfg.Duration)))
	if cfg.Strategy == Passive {
		p("%s\n", c.T("report.sessions", format.Duration(cfg.SessionS), format.Number(b.Offline.EfficiencyStart, n)))
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
	p("%s\n\n", c.T("report.crits", s.Stats.Crits))

	p("%s\n", c.T("report.timeline"))
	tl := func(label string, t float64) {
		v := c.T("report.not_reached")
		if t >= 0 {
			v = format.Duration(t)
		}
		p("  %-44s %s\n", label, v)
	}
	tl(c.T("report.first_manager"), r.FirstManagerT)
	tl(c.T("report.infra"), r.InfraT)
	tl(c.T("report.prestige"), r.PrestigeT)
	p("\n%s\n", c.T("report.unlocks"))
	for _, u := range r.Unlocks {
		p("  %-10s %s\n", format.Duration(u.T), unlockName(u.Unlock, b, c))
	}
	p("\n%s\n", c.T("report.criteria"))
	fails := 0
	for _, ch := range Checks(r, c) {
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

func unlockName(u sim.Unlock, b *balance.Balance, c *i18n.Catalog) string {
	if u.IsBusiness() {
		return c.T("unlock.business", c.T(i18n.BusinessKey(b.Businesses[u.Business()].ID)))
	}
	return c.T(i18n.UnlockKey(u.Name()))
}
