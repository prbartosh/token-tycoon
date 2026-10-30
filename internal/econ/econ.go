// Package econ holds the pure economic formulas of the game. Nothing here
// knows about game state; every function is a plain computation.
package econ

import "math"

// UnitCost is the price of the next unit when `owned` are already owned:
// base * r^owned.
func UnitCost(base, r float64, owned int) float64 {
	return base * math.Pow(r, float64(owned))
}

// BulkCost is the price of k more units: base * r^owned * (r^k - 1) / (r - 1).
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

// SpeedMilestones counts the speed thresholds reached by `owned`.
func SpeedMilestones(owned int, thresholds []int) int {
	n := 0
	for _, t := range thresholds {
		if owned >= t {
			n++
		}
	}
	return n
}

// ProfitMilestones counts profit milestones: one at `start`, then one every
// `every` units forever.
func ProfitMilestones(owned, start, every int) int {
	if owned < start || every <= 0 {
		return 0
	}
	return 1 + (owned-start)/every
}

// GlobalMilestones counts global milestones reached when the smallest owned
// count among all businesses is minOwned: the listed thresholds, then one
// every `every` units after the last threshold, forever.
func GlobalMilestones(minOwned int, thresholds []int, every int) int {
	n := SpeedMilestones(minOwned, thresholds)
	if n < len(thresholds) || len(thresholds) == 0 || every <= 0 {
		return n
	}
	return n + (minOwned-thresholds[len(thresholds)-1])/every
}

// GlobalThreshold returns the owned count required by the n-th global
// milestone (n >= 1): the listed thresholds, then every `every` units after
// the last one. It returns 0 for n < 1.
func GlobalThreshold(n int, thresholds []int, every int) int {
	if n < 1 || len(thresholds) == 0 {
		return 0
	}
	if n <= len(thresholds) {
		return thresholds[n-1]
	}
	return thresholds[len(thresholds)-1] + (n-len(thresholds))*every
}

// NextGlobalThreshold returns the next global milestone threshold above minOwned.
func NextGlobalThreshold(minOwned int, thresholds []int, every int) int {
	for _, t := range thresholds {
		if minOwned < t {
			return t
		}
	}
	last := thresholds[len(thresholds)-1]
	return last + ((minOwned-last)/every+1)*every
}

// NextBusinessThreshold returns the next speed or profit milestone above owned
// (0 if there is none, which cannot happen while profit milestones repeat).
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

// MultPow returns mult^n, the effect of n stacked milestones.
func MultPow(mult float64, n int) float64 {
	if n == 0 {
		return 1
	}
	return math.Pow(mult, float64(n))
}

// CritEV is the expected revenue multiplier of a cycle with crit chance p and
// crit multiplier m: 1 + p*(m-1). It does not depend on the cycle length.
func CritEV(p, m float64) float64 { return 1 + p*(m-1) }

// Layer1Params is the parameter reward of prestige layer 1:
// floor(coef * (runRevenue/divisor)^exp).
func Layer1Params(runRevenue, coef, divisor, exp float64) float64 {
	if !(runRevenue > 0) {
		return 0
	}
	return floorEps(coef * math.Pow(runRevenue/divisor, exp))
}

// Layer1Bonus is the revenue multiplier from params: 1 + params*bonusPerParam.
func Layer1Bonus(params, bonusPerParam float64) float64 { return 1 + params*bonusPerParam }

// Layer2Capital is the VC capital of prestige layer 2:
// floor(coef * (log10(total) - offset)^exp), 0 when the log is below offset.
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

// Payback is cost / rateGain in seconds (+Inf when the gain is not positive).
func Payback(cost, rateGain float64) float64 {
	if !(rateGain > 0) {
		return math.Inf(1)
	}
	return cost / rateGain
}

// floorEps floors v, treating values within a relative 1e-9 below an integer
// as that integer, so 4.9999999999 from log10 rounding becomes 5.
func floorEps(v float64) float64 { return math.Floor(v + math.Max(1e-9, math.Abs(v)*1e-12)) }
