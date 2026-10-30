// Package training holds the pure scaling-law formulas of model training
// (DESIGN 7): excess loss, benchmark, FLOPs, cluster throughput, the
// compute-optimal split and the synthetic-data penalty.
package training

import (
	"math"

	"token-tycoon/internal/balance"
)

func ExcessLoss(n, d, arch float64, t *balance.Training) float64 {
	if !(n > 0) || !(d > 0) {
		return math.Inf(1)
	}
	return (t.LossA/math.Pow(n, t.Alpha) + t.LossB/math.Pow(d, t.Beta)) * arch
}

func Benchmark(n, d, arch float64, t *balance.Training) float64 {
	return 100 * math.Exp(-t.BenchmarkK*ExcessLoss(n, d, arch, t))
}

func FLOPs(n, d float64, t *balance.Training) float64 { return t.FlopsPerParamToken * n * d }

func ClusterFLOPS(racks []int, gens []balance.GPUGen, exp float64) float64 {
	f := 0.0
	for g := range gens {
		if g < len(racks) && racks[g] > 0 {
			f += gens[g].FlopsPerRack * math.Pow(float64(racks[g]), exp)
		}
	}
	return f
}

// OptimalTokens is the compute-optimal data size for n parameters under a
// fixed budget C = 6ND: minimizing A/N^a + B/D^b gives aA/N^a = bB/D^b, so
// D = (bB/(aA) * N^a)^(1/b). With this game's constants the ratio D/N grows
// from about 16 at N=1e7 to about 200 at N=1e12.
func OptimalTokens(n float64, t *balance.Training) float64 {
	return math.Pow(t.Beta*t.LossB/(t.Alpha*t.LossA)*math.Pow(n, t.Alpha), 1/t.Beta)
}

func ParamsLevel(t *balance.Training, lifetime float64) int {
	lv := 0
	for i, x := range t.MaxParamsUnlock {
		if lifetime >= x {
			lv = i
		}
	}
	return min(lv, max(0, len(t.MaxParamsLevels)-1))
}

func MaxParams(t *balance.Training, level int) float64 {
	if len(t.MaxParamsLevels) == 0 {
		return 0
	}
	return t.MaxParamsLevels[max(0, min(level, len(t.MaxParamsLevels)-1))]
}

func SyntheticMult(frac float64, t *balance.Training) float64 {
	over := frac - t.SyntheticPenaltyAbove
	if !(over > 0) {
		return 1
	}
	return 1 + t.SyntheticPenalty*math.Min(1, over/(1-t.SyntheticPenaltyAbove))
}

func BestSplit(budget, maxN, maxD, arch float64, t *balance.Training) (n, d, bench float64) {
	if !(budget > 0) || !(maxN >= t.MinParams) || !(maxD > 0) {
		return 0, 0, 0
	}
	lo, hi := math.Log(t.MinParams), math.Log(maxN)
	const steps = 64
	for i := 0; i <= steps; i++ {
		cn := math.Exp(lo + (hi-lo)*float64(i)/steps)
		cd := math.Min(maxD, budget/(t.FlopsPerParamToken*cn))
		if b := Benchmark(cn, cd, arch, t); b > bench {
			n, d, bench = cn, cd, b
		}
	}
	return n, d, bench
}

func LicenseTokens(dt *balance.Data, level int) float64 {
	if len(dt.LicenseTokens) == 0 {
		return 0
	}
	return dt.LicenseTokens[max(0, min(level, len(dt.LicenseTokens)-1))]
}
