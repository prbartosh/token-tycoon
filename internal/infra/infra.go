// Package infra holds the pure formulas of the infrastructure layer: the seeded
// energy price index, peak hours, energy dispatch and compute throttling.
package infra

import (
	"math"

	"token-tycoon/internal/balance"
)

const day = 86400.0

func HourOfDay(clock float64) float64 {
	return math.Mod(clock, day) / 3600
}

func IsPeak(clock float64, e *balance.Energy) bool {
	h := HourOfDay(clock)
	return h >= e.GridPeakHours[0] && h < e.GridPeakHours[1]
}

// EnergyIndex depends only on seed and game clock (DESIGN 16.6: never saved):
// a daily sine peaking mid peak window plus interpolated hourly hash noise.
func EnergyIndex(seed uint64, clock float64, e *balance.Energy) float64 {
	peakMid := (e.GridPeakHours[0] + e.GridPeakHours[1]) / 2
	h := HourOfDay(clock)
	wave := math.Sin(2 * math.Pi * (h - peakMid + 6) / 24)
	hours := clock / 3600
	k := math.Floor(hours)
	f := hours - k
	n0 := noise(seed, uint64(k))
	n1 := noise(seed, uint64(k)+1)
	n := n0 + (n1-n0)*f
	v := e.IndexMean + e.IndexDailyAmplitude*wave + e.IndexNoise*n
	return math.Min(100, math.Max(0, v))
}

func noise(seed, k uint64) float64 {
	x := seed ^ (k * 0x9E3779B97F4A7C15) ^ 0x5EED_E4E7
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	x ^= x >> 31
	return float64(x>>11)*(2.0/(1<<53)) - 1
}

type Supply struct {
	FreeMW float64
	PPAMW  float64
	GridMW float64
	Peak   bool
}

type Load struct {
	EnergyAvail float64
	EnergyUsed  float64
	FreeUsed    float64
	PPAUsed     float64
	GridUsed    float64
}

func Dispatch(needMW float64, s Supply, peakAvailability float64) Load {
	grid := s.GridMW
	if s.Peak {
		grid *= peakAvailability
	}
	l := Load{EnergyAvail: s.FreeMW + s.PPAMW + grid}
	left := needMW
	l.FreeUsed = math.Min(left, s.FreeMW)
	left -= l.FreeUsed
	l.PPAUsed = math.Min(left, s.PPAMW)
	left -= l.PPAUsed
	l.GridUsed = math.Min(left, grid)
	l.EnergyUsed = l.FreeUsed + l.PPAUsed + l.GridUsed
	return l
}

func Throttle(computeNeed, computeCap, energyNeed, energyAvail float64) float64 {
	t := 1.0
	if computeNeed > 0 {
		t = math.Min(t, computeCap/computeNeed)
	}
	if energyNeed > 0 {
		t = math.Min(t, energyAvail/energyNeed)
	}
	return math.Max(0, t)
}

func EnergyCostShare(l Load, gridIndex, ppaIndex, coef float64) float64 {
	if !(l.EnergyUsed > 0) {
		return 0
	}
	return coef * (l.GridUsed*gridIndex/50 + l.PPAUsed*ppaIndex/50) / l.EnergyUsed
}
