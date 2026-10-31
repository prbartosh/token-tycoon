package sim

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

const dayS = 86400

const DaySnapshots = 9

type Investors struct {
	QuarterEndS float64               `json:"quarter_end_s"`
	StartTotal  float64               `json:"start_total"`
	Target      float64               `json:"target"`
	LastRevenue float64               `json:"last_revenue"`
	Bonus       float64               `json:"bonus"`
	Day         int                   `json:"day"`
	DayTotals   [DaySnapshots]float64 `json:"day_totals"`
}

type InvestorView struct {
	Valuation   float64
	Annual      float64
	HypeMult    float64
	Growth7d    float64
	GrowthMult  float64
	Revenue     float64
	Target      float64
	LeftS       float64
	Progress    float64
	Bonus       float64
	LastRevenue float64
}

func snapshot(iv *Investors, d int) float64 {
	return iv.DayTotals[((d%DaySnapshots)+DaySnapshots)%DaySnapshots]
}

func lastDayRevenue(s *State) float64 {
	iv := &s.Investors
	d := iv.Day
	if d < 1 {
		return 0
	}
	return math.Max(0, snapshot(iv, d)-snapshot(iv, d-1))
}

func Growth7d(s *State) float64 {
	iv := &s.Investors
	d := iv.Day
	if d < 8 {
		return 0
	}
	now, before := snapshot(iv, d)-snapshot(iv, d-1), snapshot(iv, d-7)-snapshot(iv, d-8)
	if !(now > 0) || !(before > 0) {
		return 0
	}
	return now/before - 1
}

func AnnualRevenue(s *State) float64 {
	return math.Max(s.Derived.BaseIncome*yearS, lastDayRevenue(s)*365)
}

func Valuation(s *State, b *balance.Balance) InvestorView {
	v := InvestorView{Annual: AnnualRevenue(s), HypeMult: 1 + math.Max(0, s.Derived.Market.Hype), Growth7d: Growth7d(s)}
	v.GrowthMult = 1 + b.Investors.GrowthWeight*math.Log10(1+math.Max(0, v.Growth7d))
	v.Valuation = v.Annual * v.HypeMult * v.GrowthMult * b.Prestige.Layer2.ValuationMult
	iv := &s.Investors
	v.Bonus, v.LastRevenue, v.Target = iv.Bonus, iv.LastRevenue, iv.Target
	if iv.QuarterEndS > 0 {
		v.Revenue = math.Max(0, s.Lifetime.Total-iv.StartTotal)
		v.LeftS = math.Max(0, iv.QuarterEndS-s.Clock)
		if iv.Target > 0 {
			v.Progress = math.Min(1, v.Revenue/iv.Target)
		}
	}
	return v
}

func startQuarter(s *State, b *balance.Balance, prev float64) {
	iv := &s.Investors
	g := b.Investors.TargetGrowth
	iv.Target = prev * (1 + s.RNG.Misc.Range(g[0], g[1]))
	iv.StartTotal = s.Lifetime.Total
	iv.QuarterEndS = s.Clock + b.Investors.QuarterH*3600
}

func tickInvestors(s *State, b *balance.Balance, out []Event) []Event {
	iv := &s.Investors
	if d := int(s.Clock / dayS); d > iv.Day {
		for k := max(iv.Day+1, d-DaySnapshots+1); k <= d; k++ {
			iv.DayTotals[k%DaySnapshots] = s.Lifetime.Total
		}
		iv.Day = d
	}
	if !s.IsUnlocked(UnlockInvestors) {
		return out
	}
	if !(iv.QuarterEndS > 0) {
		if inc := s.Derived.BaseIncome; inc > 0 {
			startQuarter(s, b, inc*b.Investors.QuarterH*3600)
		}
		return out
	}
	if s.Clock < iv.QuarterEndS {
		return out
	}
	rev := math.Max(0, s.Lifetime.Total-iv.StartTotal)
	beat := rev >= iv.Target
	n := 0
	if beat {
		iv.Bonus += b.Investors.BeatBonusCapital
		s.Stats.QuartersBeaten++
		n = 1
	}
	s.Stats.Quarters++
	target := iv.Target
	iv.LastRevenue = rev
	startQuarter(s, b, rev)
	return append(out, Event{Kind: EvQuarter, Business: -1, Count: n, Amount: rev, Value: target})
}

func normalizeInvestors(s *State) {
	iv := &s.Investors
	bad := func(x float64) bool { return !(x >= 0) || math.IsInf(x, 0) }
	if bad(iv.QuarterEndS) || bad(iv.StartTotal) || bad(iv.Target) {
		iv.QuarterEndS, iv.StartTotal, iv.Target = 0, 0, 0
	}
	if bad(iv.LastRevenue) {
		iv.LastRevenue = 0
	}
	if bad(iv.Bonus) {
		iv.Bonus = 0
	}
	for k := range iv.DayTotals {
		if bad(iv.DayTotals[k]) {
			iv.DayTotals[k] = 0
		}
	}
	if iv.Day < 0 || iv.Day > int(s.Clock/dayS) {
		iv.Day = int(s.Clock / dayS)
	}
}
