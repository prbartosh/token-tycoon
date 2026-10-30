// Package training holds the pure scaling-law formulas of model training
// (DESIGN 7): excess loss, benchmark, FLOPs and cluster throughput.
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
// fixed budget C = 6ND: minimizing A/N^a + B/D^b gives D/N = (bB/aA)^(1/b) N^(a/b - 1),
// which this game's constants put near 20 tokens per parameter.
func OptimalTokens(n float64, t *balance.Training) float64 {
	r := math.Pow(t.Beta*t.LossB/(t.Alpha*t.LossA)*math.Pow(n, t.Alpha), 1/t.Beta)
	return r
}

func MaxParams(t *balance.Training, level int) float64 {
	if len(t.MaxParamsLevels) == 0 {
		return 0
	}
	return t.MaxParamsLevels[max(0, min(level, len(t.MaxParamsLevels)-1))]
}
