// Package econ holds the pure economic formulas of the game. Nothing here
// knows about game state; every function is a plain computation.
package econ

import "math"

func UnitCost(base, r float64, owned int) float64 {
	return base * math.Pow(r, float64(owned))
}

func BulkCost(base, r float64, owned, k int) float64 {
	if k <= 0 {
		return 0
	}
	if r == 1 {
		return base * float64(k)
	}
	return UnitCost(base, r, owned) * (math.Pow(r, float64(k)) - 1) / (r - 1)
}

// MaxAffordable is the largest k with BulkCost(base, r, owned, k) <= money:
// floor(log(money*(r-1)/(base*r^owned) + 1) / log r), corrected for rounding.
func MaxAffordable(money, base, r float64, owned int) int {
	if !(money > 0) || !(base > 0) {
		return 0
	}
	unit := UnitCost(base, r, owned)
	if math.IsInf(unit, 0) || unit > money {
		return 0
	}
	var k int
	if r == 1 {
		k = int(math.Floor(money / base))
	} else {
		f := math.Log(money*(r-1)/unit+1) / math.Log(r)
		if f > 1e9 {
			f = 1e9
		}
		k = int(math.Floor(f))
	}
	for k > 0 && BulkCost(base, r, owned, k) > money {
		k--
	}
	for BulkCost(base, r, owned, k+1) <= money && k < 1e9 {
		k++
	}
	return k
}

func SpeedMilestones(owned int, thresholds []int) int {
	n := 0
	for _, t := range thresholds {
		if owned >= t {
			n++
		}
	}
	return n
}

func ProfitMilestones(owned, start, every int) int {
	if owned < start || every <= 0 {
		return 0
	}
	return 1 + (owned-start)/every
}

func GlobalMilestones(minOwned int, thresholds []int, every int) int {
	n := SpeedMilestones(minOwned, thresholds)
	if n < len(thresholds) || len(thresholds) == 0 || every <= 0 {
		return n
	}
	return n + (minOwned-thresholds[len(thresholds)-1])/every
}

func GlobalThreshold(n int, thresholds []int, every int) int {
	if n < 1 || len(thresholds) == 0 {
		return 0
	}
	if n <= len(thresholds) {
		return thresholds[n-1]
	}
	return thresholds[len(thresholds)-1] + (n-len(thresholds))*every
}

func NextGlobalThreshold(minOwned int, thresholds []int, every int) int {
	for _, t := range thresholds {
		if minOwned < t {
			return t
		}
	}
	last := thresholds[len(thresholds)-1]
	return last + ((minOwned-last)/every+1)*every
}

func NextBusinessThreshold(owned int, speed []int, profitStart, profitEvery int) int {
	for _, t := range speed {
		if owned < t {
			return t
		}
	}
	if owned < profitStart {
		return profitStart
	}
	if profitEvery <= 0 {
		return 0
	}
	return profitStart + ((owned-profitStart)/profitEvery+1)*profitEvery
}

func MultPow(mult float64, n int) float64 {
	if n < 0 || n > 64 {
		return math.Pow(mult, float64(n))
	}
	r := 1.0
	for range n {
		r *= mult
	}
	return r
}

func CritEV(p, m float64) float64 { return 1 + p*(m-1) }

func Layer1Params(runRevenue, coef, divisor, exp float64) float64 {
	if !(runRevenue > 0) {
		return 0
	}
	return floorEps(coef * math.Pow(runRevenue/divisor, exp))
}

func Layer1Bonus(params, bonusPerParam float64) float64 { return 1 + params*bonusPerParam }

func Layer2Capital(total, coef, offset, exp float64) float64 {
	if !(total > 0) {
		return 0
	}
	x := math.Log10(total) - offset
	if x <= 0 {
		return 0
	}
	return floorEps(coef * math.Pow(x, exp))
}

func Payback(cost, rateGain float64) float64 {
	if !(rateGain > 0) {
		return math.Inf(1)
	}
	return cost / rateGain
}

func floorEps(v float64) float64 { return math.Floor(v + math.Max(1e-9, math.Abs(v)*1e-12)) }
