package sim

import (
	"math"

	"token-tycoon/internal/balance"
)

type OfflineResult struct {
	Elapsed    float64
	Simulated  float64
	Earned     float64
	Efficiency float64
	Steps      int
}

func OfflineCapS(s *State, b *balance.Balance) float64 {
	return b.Offline.CapHoursStart * 3600
}

func OfflineEfficiency(s *State, b *balance.Balance) float64 {
	return b.Offline.EfficiencyStart
}

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
