// Package market holds the pure market formulas (DESIGN 6): the seeded rival
// release schedule, market share, quality premium, demand factor, model
// deflation, open-weights price pressure, hype decay and segments. Rivals are
// derived from seed and game clock and never saved (DESIGN 16.6).
package market

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

const Day = 86400.0

const (
	SegFree = iota
	SegPro
	SegEnterprise
	NumSegments
)

var SegmentNames = [NumSegments]string{"free", "pro", "enterprise"}

func SegmentIndex(name string) int {
	for i, n := range SegmentNames {
		if n == name {
			return i
		}
	}
	return -1
}

type Release struct {
	K         int
	AtS       float64
	Benchmark float64
}

func hash(seed, a, b uint64) float64 {
	x := seed ^ (a * 0xD6E8FEB86659FD93) ^ (b * 0x9E3779B97F4A7C15) ^ 0x4A11E7
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

func benchmarkAt(seed uint64, i int, r *balance.Rival, k int, at, spanDays float64) float64 {
	p := math.Min(1, math.Max(0, (at/Day-1)/(spanDays-1)))
	v := r.StartBenchmark + (r.EndBenchmark-r.StartBenchmark)*p
	if n := styleNoise(r); n > 0 {
		v += (2*hash(seed, uint64(i)+101, uint64(k)) - 1) * n
	}
	return math.Min(100, math.Max(1, v))
}

func kRange(r *balance.Rival, fromS, toS float64) (lo, hi int) {
	lo = max(0, int(math.Floor((fromS/Day-r.ReleaseJitterDays)/r.ReleaseEveryDays))-2)
	hi = int(math.Floor((toS/Day+r.ReleaseJitterDays)/r.ReleaseEveryDays)) + 1
	return lo, hi
}

func LastRelease(seed uint64, i int, r *balance.Rival, clock, spanDays float64) Release {
	best := Release{K: -1, AtS: -1, Benchmark: r.StartBenchmark}
	lo, hi := kRange(r, clock, clock)
	for k := lo; k <= hi; k++ {
		if at := releaseAt(seed, i, r, k); at <= clock && k > best.K {
			best = Release{K: k, AtS: at, Benchmark: benchmarkAt(seed, i, r, k, at, spanDays)}
		}
	}
	return best
}

func NextRelease(seed uint64, i int, r *balance.Rival, clock float64) float64 {
	next := math.Inf(1)
	lo, hi := kRange(r, clock, clock+r.ReleaseEveryDays*Day)
	for k := lo; k <= hi; k++ {
		if at := releaseAt(seed, i, r, k); at > clock && at < next {
			next = at
		}
	}
	return next
}

func ReleasedIn(seed uint64, i int, r *balance.Rival, fromS, toS, spanDays float64) (Release, bool) {
	if !(toS > fromS) {
		return Release{}, false
	}
	found := false
	var rel Release
	lo, hi := kRange(r, fromS, toS)
	for k := lo; k <= hi; k++ {
		if at := releaseAt(seed, i, r, k); at > fromS && at <= toS && (!found || k > rel.K) {
			rel, found = Release{K: k, AtS: at, Benchmark: benchmarkAt(seed, i, r, k, at, spanDays)}, true
		}
	}
	return rel, found
}

func countReleases(seed uint64, i int, r *balance.Rival, fromS, toS float64) int {
	if !(toS > fromS) {
		return 0
	}
	n := 0
	lo, hi := kRange(r, fromS, toS)
	for k := lo; k <= hi; k++ {
		if at := releaseAt(seed, i, r, k); at > fromS && at <= toS {
			n++
		}
	}
	return n
}

func Pressure(seed uint64, rivals []balance.Rival, sinceS, clock, floor float64) float64 {
	p := 1.0
	for i := range rivals {
		r := &rivals[i]
		if r.PricePressure == 1 {
			continue
		}
		if n := countReleases(seed, i, r, sinceS, clock); n > 0 {
			p *= math.Pow(r.PricePressure, float64(n))
		}
	}
	return math.Max(floor, p)
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
	return ShareOf(player, RivalWeight(rivals, exp), exp)
}

func RivalWeight(rivals []float64, exp float64) float64 {
	sum := 0.0
	for _, r := range rivals {
		sum += math.Pow(math.Max(0, r), exp)
	}
	return sum
}

func ShareOf(player, rivalWeight, exp float64) float64 {
	p := math.Pow(math.Max(0, player), exp)
	if !(p+rivalWeight > 0) {
		return 0
	}
	return p / (p + rivalWeight)
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

type Quote struct {
	Share     float64
	Quality   float64
	Demand    float64
	Deflation float64
	Price     float64
}

func Price(bench, rivalWeight, median, hype, ageS, pressure float64, m *balance.Market) Quote {
	q := Quote{
		Share:     ShareOf(bench, rivalWeight, m.ShareExponent),
		Quality:   Quality(bench, median, m),
		Deflation: Deflation(ageS, m),
	}
	q.Demand = Demand(q.Share, hype, m)
	q.Price = m.PriceBase * q.Quality * q.Demand * q.Deflation * pressure
	return q
}

func SegmentMult(seg int, m *balance.Market) float64 {
	switch seg {
	case SegFree:
		return m.Segments.FreePrice
	case SegEnterprise:
		return m.Segments.EnterprisePrice
	}
	return m.Segments.ProPrice
}

func HypeGainPerS(freeAlloc, share float64, m *balance.Market) float64 {
	return freeAlloc * share * m.Segments.FreeHypePerShare / 3600
}

// Exact integral of dh/dt = gain - h*ln2/halfLife, independent of step length.
func HypeAfter(h, gainPerS, halfLifeS, dt float64) float64 {
	lambda := math.Ln2 / halfLifeS
	eq := gainPerS / lambda
	return eq + (h-eq)*math.Exp(-lambda*dt)
}

func SteadyHype(gainPerS, halfLifeS float64) float64 {
	return gainPerS * halfLifeS / math.Ln2
}

func Rank(player float64, rivals []float64) int {
	r := 1
	for _, x := range rivals {
		if x > player {
			r++
		}
	}
	return r
}
