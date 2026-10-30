// Package market holds the pure market formulas: the seeded rival release
// schedule, market share, quality premium, demand factor and model deflation.
// Rivals are derived from seed and game clock and never saved (DESIGN 16.6).
package market

import (
	"math"

	"token-tycoon/internal/balance"
)

const Day = 86400.0

type Release struct {
	K         int
	AtS       float64
	Benchmark float64
}

func hash(seed, a, b uint64) float64 {
	x := seed ^ (a * 0xD6E8FEB86659FD93) ^ (b * 0x9E3779B97F4A7C15) ^ 0x4A11_E7
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	x ^= x >> 31
	return float64(x>>11) * (1.0 / (1 << 53))
}

func releaseAt(seed uint64, i int, r *balance.Rival, k int) float64 {
	t := float64(k+1) * r.ReleaseEveryDays
	if r.ReleaseJitterDays > 0 {
		t += (2*hash(seed, uint64(i)+1, uint64(k)) - 1) * r.ReleaseJitterDays
	}
	return math.Max(0, t) * Day
}

func styleNoise(r *balance.Rival) float64 {
	switch r.Style {
	case "jumpy":
		return 6
	case "open_source":
		return 1
	}
	return 0
}

// benchmarkAt follows a linear schedule from start to end over spanDays,
// sampled at the planned release day; the style adds seeded noise.
func benchmarkAt(seed uint64, i int, r *balance.Rival, k int, spanDays float64) float64 {
	p := math.Min(1, float64(k+1)*r.ReleaseEveryDays/spanDays)
	v := r.StartBenchmark + (r.EndBenchmark-r.StartBenchmark)*p
	if n := styleNoise(r); n > 0 {
		v += (2*hash(seed, uint64(i)+101, uint64(k)) - 1) * n
	}
	return math.Min(100, math.Max(1, v))
}

func LastRelease(seed uint64, i int, r *balance.Rival, clock, spanDays float64) Release {
	every := r.ReleaseEveryDays
	t := clock / Day
	hi := int(math.Floor((t+r.ReleaseJitterDays)/every)) + 1
	lo := int(math.Floor((t-r.ReleaseJitterDays)/every)) - 2
	for k := hi; k >= max(0, lo); k-- {
		if at := releaseAt(seed, i, r, k); at <= clock {
			return Release{K: k, AtS: at, Benchmark: benchmarkAt(seed, i, r, k, spanDays)}
		}
	}
	return Release{K: -1, AtS: -1, Benchmark: r.StartBenchmark}
}

func Median(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return 0
	}
	var buf [16]float64
	s := buf[:0]
	if n > len(buf) {
		s = make([]float64, 0, n)
	}
	for _, x := range xs {
		i := len(s)
		s = append(s, x)
		for i > 0 && s[i-1] > x {
			s[i] = s[i-1]
			i--
		}
		s[i] = x
	}
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func Share(player float64, rivals []float64, exp float64) float64 {
	p := math.Pow(math.Max(0, player), exp)
	sum := p
	for _, r := range rivals {
		sum += math.Pow(math.Max(0, r), exp)
	}
	if !(sum > 0) {
		return 0
	}
	return p / sum
}

func clamp(x float64, c []float64) float64 { return math.Min(c[1], math.Max(c[0], x)) }

func Quality(player, median float64, m *balance.Market) float64 {
	return clamp(1+(player-median)/m.QualityDivisor, m.QualityClamp)
}

func Demand(share, hype float64, m *balance.Market) float64 {
	return clamp(math.Sqrt(share/m.ShareNeutral*(1+math.Max(0, hype))), m.DemandClamp)
}

func Deflation(ageS float64, m *balance.Market) float64 {
	return math.Pow(m.DeflationPerDay, math.Max(0, ageS)/Day)
}

// HypeAfter integrates dh/dt = gain - h*ln2/halfLife exactly over dt, so the
// result does not depend on the step length (online 0.1 s or offline 60 s).
func HypeAfter(h, gainPerS, halfLifeS, dt float64) float64 {
	lambda := math.Ln2 / halfLifeS
	eq := gainPerS / lambda
	return eq + (h-eq)*math.Exp(-lambda*dt)
}
