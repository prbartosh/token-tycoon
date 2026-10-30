package econ

import (
	"math"
	"testing"
)

func near(a, b float64) bool {
	if a == b {
		return true
	}
	return math.Abs(a-b) <= 1e-9*math.Max(math.Abs(a), math.Abs(b))
}

func TestCosts(t *testing.T) {
	cases := []struct {
		base, r    float64
		owned, k   int
		unit, bulk float64
	}{
		{10, 1.07, 0, 1, 10, 10},
		{10, 1.07, 1, 1, 10.7, 10.7},
		{10, 1.07, 0, 10, 10, 138.16448},
		{120, 1.15, 5, 3, 120 * math.Pow(1.15, 5), 120 * math.Pow(1.15, 5) * (1 + 1.15 + 1.15*1.15)},
		{5, 1, 3, 4, 5, 20},
		{10, 1.07, 0, 0, 10, 0},
	}
	for _, c := range cases {
		if got := UnitCost(c.base, c.r, c.owned); !near(got, c.unit) {
			t.Errorf("UnitCost(%v,%v,%d) = %v, want %v", c.base, c.r, c.owned, got, c.unit)
		}
		got := BulkCost(c.base, c.r, c.owned, c.k)
		if math.Abs(got-c.bulk) > 1e-4*math.Max(1, c.bulk) {
			t.Errorf("BulkCost(%v,%v,%d,%d) = %v, want %v", c.base, c.r, c.owned, c.k, got, c.bulk)
		}
	}
}

func TestMaxAffordable(t *testing.T) {
	cases := []struct {
		money, base, r float64
		owned, want    int
	}{
		{0, 10, 1.07, 0, 0},
		{9.99, 10, 1.07, 0, 0},
		{10, 10, 1.07, 0, 1},
		{20.69, 10, 1.07, 0, 1},
		{20.7, 10, 1.07, 0, 2},
		{BulkCost(10, 1.07, 0, 10), 10, 1.07, 0, 10},
		{BulkCost(10, 1.07, 0, 10) - 1e-6, 10, 1.07, 0, 9},
		{BulkCost(120, 1.15, 37, 100), 120, 1.15, 37, 100},
		{100, 5, 1, 0, 20},
		{1e300, 10, 1.07, 0, 10136},
		{1e10, 10, 1.07, 100000, 0},
	}
	for _, c := range cases {
		got := MaxAffordable(c.money, c.base, c.r, c.owned)
		if got != c.want {
			t.Errorf("MaxAffordable(%v,%v,%v,%d) = %d, want %d", c.money, c.base, c.r, c.owned, got, c.want)
		}
		if got > 0 && BulkCost(c.base, c.r, c.owned, got) > c.money {
			t.Errorf("MaxAffordable result not affordable")
		}
	}
}

func TestMilestones(t *testing.T) {
	speed := []int{25, 50, 100}
	global := []int{25, 50, 100}
	cases := []struct {
		owned, speed, profit, global, nextGlobal, nextBiz int
	}{
		{0, 0, 0, 0, 25, 25},
		{24, 0, 0, 0, 25, 25},
		{25, 1, 0, 1, 50, 50},
		{99, 2, 0, 2, 100, 100},
		{100, 3, 0, 3, 200, 200},
		{199, 3, 0, 3, 200, 200},
		{200, 3, 1, 4, 300, 300},
		{299, 3, 1, 4, 300, 300},
		{300, 3, 2, 5, 400, 400},
		{1000, 3, 9, 12, 1100, 1100},
	}
	for _, c := range cases {
		if got := SpeedMilestones(c.owned, speed); got != c.speed {
			t.Errorf("Speed(%d) = %d, want %d", c.owned, got, c.speed)
		}
		if got := ProfitMilestones(c.owned, 200, 100); got != c.profit {
			t.Errorf("Profit(%d) = %d, want %d", c.owned, got, c.profit)
		}
		if got := GlobalMilestones(c.owned, global, 100); got != c.global {
			t.Errorf("Global(%d) = %d, want %d", c.owned, got, c.global)
		}
		if got := NextGlobalThreshold(c.owned, global, 100); got != c.nextGlobal {
			t.Errorf("NextGlobal(%d) = %d, want %d", c.owned, got, c.nextGlobal)
		}
		if got := NextBusinessThreshold(c.owned, speed, 200, 100); got != c.nextBiz {
			t.Errorf("NextBiz(%d) = %d, want %d", c.owned, got, c.nextBiz)
		}
	}
	// The n-th global milestone's threshold is consistent with the counter.
	for n := 1; n <= 12; n++ {
		th := GlobalThreshold(n, global, 100)
		if GlobalMilestones(th, global, 100) != n || GlobalMilestones(th-1, global, 100) != n-1 {
			t.Errorf("GlobalThreshold(%d) = %d inconsistent with GlobalMilestones", n, th)
		}
	}
	if GlobalThreshold(0, global, 100) != 0 || GlobalThreshold(4, global, 100) != 200 {
		t.Error("GlobalThreshold edge cases")
	}
	if MultPow(2, 3) != 8 || MultPow(0.5, 0) != 1 {
		t.Error("MultPow")
	}
}

func TestPrestige(t *testing.T) {
	cases := []struct{ run, want float64 }{
		{0, 0}, {1e11, 31}, {1e12, 100}, {4e12, 200}, {1e14, 1000}, {1e24, 1e8},
	}
	for _, c := range cases {
		if got := Layer1Params(c.run, 100, 1e12, 0.5); got != c.want {
			t.Errorf("Layer1Params(%g) = %v, want %v", c.run, got, c.want)
		}
	}
	if got := Layer1Bonus(100, 0.02); !near(got, 3) {
		t.Errorf("Layer1Bonus = %v", got)
	}
	l2 := []struct{ total, want float64 }{
		{1e39, 0}, {1e40, 5}, {1e43, 40}, {1e48, 135}, {0, 0},
	}
	for _, c := range l2 {
		if got := Layer2Capital(c.total, 5, 39, 1.5); got != c.want {
			t.Errorf("Layer2Capital(%g) = %v, want %v", c.total, got, c.want)
		}
	}
}

func TestCritEV(t *testing.T) {
	cases := []struct{ p, m, want float64 }{{0.03, 10, 1.27}, {0, 10, 1}, {1, 10, 10}, {0.5, 1, 1}}
	for _, c := range cases {
		if got := CritEV(c.p, c.m); !near(got, c.want) {
			t.Errorf("CritEV(%v,%v) = %v, want %v", c.p, c.m, got, c.want)
		}
	}
	if !math.IsInf(Payback(10, 0), 1) || Payback(10, 2) != 5 {
		t.Error("Payback")
	}
}
