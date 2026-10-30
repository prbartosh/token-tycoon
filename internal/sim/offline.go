package sim

import (
	"math"

	"token-tycoon/internal/balance"
)

// OfflineResult summarizes an offline catch-up for the "while you were away"
// screen.
type OfflineResult struct {
	Elapsed    float64 // seconds the player was away
	Simulated  float64 // seconds actually simulated (capped)
	Earned     float64 // $ earned while away
	Efficiency float64 // revenue multiplier applied
	Steps      int
}

// OfflineCapS is the current offline cap in seconds.
func OfflineCapS(s *State, b *balance.Balance) float64 {
	// Research raises the cap towards cap_hours_max in M4.
	return b.Offline.CapHoursStart * 3600
}

// OfflineEfficiency is the current offline revenue multiplier.
func OfflineEfficiency(s *State, b *balance.Balance) float64 {
	// Research raises it towards efficiency_max in M4.
	return b.Offline.EfficiencyStart
}

// Offline catches up `elapsed` seconds of absence (ARCHITECTURE 5): capped to
// the offline limit, simulated in offline_step_s steps with expected-value
// crits, no combo and revenue times the offline efficiency. Unlock events are
// appended to out; cycle and crit events are not emitted.
func Offline(s *State, b *balance.Balance, elapsed float64, out []Event) (OfflineResult, []Event) {
	res := OfflineResult{Elapsed: math.Max(0, elapsed), Efficiency: OfflineEfficiency(s, b)}
	res.Simulated = math.Min(res.Elapsed, OfflineCapS(s, b))
	md := mode{offline: true, efficiency: res.Efficiency}
	before := s.Lifetime.Total
	stepS := b.Engine.OfflineStepS
	for left := res.Simulated; left > 1e-9; {
		dt := math.Min(stepS, left)
		out = step(s, b, dt, nil, out, md)
		left -= dt
		res.Steps++
	}
	res.Earned = s.Lifetime.Total - before
	return res, out
}
