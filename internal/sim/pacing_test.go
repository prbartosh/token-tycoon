package sim

import (
	"math"
	"testing"
)

func TestHugeMultipliersStayFinite(t *testing.T) {
	b := bal(t)
	s := NewState(b, 3)
	for i := range s.Businesses {
		s.Businesses[i].Owned, s.Businesses[i].Manager = 3000, true
		s.Unlocks.Set(i)
	}
	s.Prestige.Params = 1e200
	s.Prestige.ResetsL2, s.Prestige.RoundMult = 1, 1e250
	s.Money, s.Lifetime.Total = 1e299, 1e299
	Normalize(&s, b)
	for range 50 {
		Step(&s, b, 1, nil, nil)
	}
	d := &s.Derived
	for _, x := range []float64{d.IncomePerSec, d.BaseIncome, s.Money, s.Lifetime.Total, d.GlobalMult} {
		if math.IsNaN(x) || math.IsInf(x, 0) || x <= 0 {
			t.Fatalf("value not finite: income %v money %v lifetime %v global %v", d.IncomePerSec, s.Money, s.Lifetime.Total, d.GlobalMult)
		}
	}
	if s.Money > MaxStock || d.BaseIncome > float64(N)*MaxRate {
		t.Fatalf("caps not applied: money %v income %v", s.Money, d.BaseIncome)
	}
}
