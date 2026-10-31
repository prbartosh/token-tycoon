package sim

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

type OfflineResult struct {
	Elapsed    float64
	Simulated  float64
	Earned     float64
	Efficiency float64
	Steps      int
}

func OfflineCapS(s *State, b *balance.Balance) float64 {
	return math.Min(b.Offline.CapHoursMax, b.Offline.CapHoursStart+s.ResMods.OfflineCapH) * 3600
}

func OfflineEfficiency(s *State, b *balance.Balance) float64 {
	return math.Min(b.Offline.EfficiencyMax, b.Offline.EfficiencyStart+s.CardMods.Offline+s.ResMods.OfflineEff)
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
