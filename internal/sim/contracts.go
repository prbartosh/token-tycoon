package sim

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

type Contract struct {
	ID            int     `json:"id"`
	Volume        float64 `json:"volume"`
	Delivered     float64 `json:"delivered"`
	MinBenchmark  float64 `json:"min_benchmark"`
	DurationS     float64 `json:"duration_s"`
	DeadlineS     float64 `json:"deadline_s"`
	RewardMinutes float64 `json:"reward_minutes"`
	ExpiresS      float64 `json:"expires_s"`
}

func ContractSlots(s *State, b *balance.Balance) int {
	return b.Contracts.MaxActive + s.ResMods.ContractSlots
}

func ContractReward(s *State, b *balance.Balance, c *Contract) float64 {
	cc := &b.Contracts
	mult := (1 + math.Max(0, s.Market.Reputation)*cc.RepRewardBonus) * (1 + s.ResMods.ContractReward)
	return MinutesOfIncome(s, c.RewardMinutes) * mult
}

func newOffer(s *State, b *balance.Balance) Contract {
	cc := &b.Contracts
	r := &s.RNG.Misc
	dur := math.Round(r.Range(cc.DurationMin[0], cc.DurationMin[1])) * 60
	vol := SteadyTokensPerSec(s) * dur * r.Range(cc.VolumeMult[0], cc.VolumeMult[1])
	bench := math.Round(BestBenchmark(s, b) + r.Range(cc.BenchOffset[0], cc.BenchOffset[1]))
	bench = math.Min(99, math.Max(cc.MinBenchFloor, bench))
	s.ContractSeq++
	return Contract{
		ID: s.ContractSeq, Volume: vol, MinBenchmark: bench, DurationS: dur,
		RewardMinutes: math.Round(r.Range(cc.RewardMinutes[0], cc.RewardMinutes[1])),
		ExpiresS:      s.Clock + cc.OfferLifeS,
	}
}

func acceptContract(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockContracts) {
		return reject(out, c, RejectLocked, b.Contracts.UnlockLifetime)
	}
	i := c.Index
	if i < 0 || i >= len(s.Offers) {
		return reject(out, c, RejectInvalid, 0)
	}
	if len(s.Contracts) >= ContractSlots(s, b) {
		return reject(out, c, RejectTeamFull, 0)
	}
	o := s.Offers[i]
	if BestBenchmark(s, b) < o.MinBenchmark {
		return reject(out, c, RejectNeedBenchmark, o.MinBenchmark)
	}
	s.Offers = append(s.Offers[:i], s.Offers[i+1:]...)
	o.DeadlineS = s.Clock + o.DurationS
	o.ExpiresS = 0
	s.Contracts = append(s.Contracts, o)
	return append(out, Event{Kind: EvContractAccepted, Business: -1, Index: len(s.Contracts) - 1, Count: o.ID, Amount: o.Volume, Value: o.DurationS})
}

func declineContract(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := c.Index
	if i < 0 || i >= len(s.Offers) {
		return reject(out, c, RejectInvalid, 0)
	}
	id := s.Offers[i].ID
	s.Offers = append(s.Offers[:i], s.Offers[i+1:]...)
	return append(out, Event{Kind: EvContractDeclined, Business: -1, Index: i, Count: id})
}

func SteadyTokensPerSec(s *State) float64 {
	d := &s.Derived
	if !(d.CritEV > 0) {
		return d.TokensPerSec
	}
	return d.TokensPerSec / d.CritEV
}

func tickContracts(s *State, b *balance.Balance, dt, _ float64, out []Event, md mode) []Event {
	if !s.IsUnlocked(UnlockContracts) {
		return out
	}
	cc := &b.Contracts
	if md.offline {
		for k := range s.Contracts {
			s.Contracts[k].DeadlineS += dt
		}
		for k := range s.Offers {
			s.Offers[k].ExpiresS += dt
		}
		return out
	}
	d := &s.Derived
	tokens := SteadyTokensPerSec(s) * dt
	for k := 0; k < len(s.Contracts); {
		ct := &s.Contracts[k]
		ct.Delivered = math.Min(ct.Volume, ct.Delivered+tokens)
		switch {
		case ct.Delivered >= ct.Volume:
			reward := ContractReward(s, b, ct)
			earn(s, reward)
			s.Market.Reputation = math.Min(cc.ReputationMax, s.Market.Reputation+cc.ReputationGain)
			s.Stats.ContractsDone++
			out = append(out, Event{Kind: EvContractDone, Business: -1, Index: k, Count: ct.ID, Amount: reward, Value: s.Market.Reputation})
		case s.Clock+dt >= ct.DeadlineS:
			s.Market.Reputation = math.Max(0, s.Market.Reputation-cc.ReputationLoss)
			s.Stats.ContractsFailed++
			out = append(out, Event{Kind: EvContractFailed, Business: -1, Index: k, Count: ct.ID, Value: s.Market.Reputation})
		default:
			k++
			continue
		}
		s.Contracts = append(s.Contracts[:k], s.Contracts[k+1:]...)
	}
	for k := 0; k < len(s.Offers); {
		if s.Offers[k].ExpiresS <= s.Clock {
			s.Offers = append(s.Offers[:k], s.Offers[k+1:]...)
			continue
		}
		k++
	}
	if len(s.Offers) >= cc.Offers || !(d.TokensPerSec > 0) {
		return out
	}
	if s.NextOfferS <= 0 {
		s.NextOfferS = s.Clock
	}
	if s.Clock < s.NextOfferS {
		return out
	}
	s.NextOfferS = s.Clock + cc.OfferEveryS
	s.Offers = append(s.Offers, newOffer(s, b))
	return append(out, Event{Kind: EvContractOffer, Business: -1, Index: len(s.Offers) - 1, Count: s.ContractSeq})
}

func normalizeContracts(s *State, b *balance.Balance) {
	clean := func(cs []Contract) []Contract {
		keep := cs[:0]
		for _, c := range cs {
			if c.Volume > 0 && !math.IsInf(c.Volume, 0) && c.Delivered >= 0 {
				keep = append(keep, c)
			}
		}
		return keep
	}
	s.Contracts = clean(s.Contracts)
	s.Offers = clean(s.Offers)
	slots := b.Contracts.MaxActive + len(b.ResearchNodes)
	if s.Contracts == nil {
		s.Contracts = make([]Contract, 0, slots)
	}
	if s.Offers == nil {
		s.Offers = make([]Contract, 0, b.Contracts.Offers+1)
	}
	for _, c := range s.Contracts {
		s.ContractSeq = max(s.ContractSeq, c.ID)
	}
	for _, c := range s.Offers {
		s.ContractSeq = max(s.ContractSeq, c.ID)
	}
	if !(s.Market.Reputation >= 0) {
		s.Market.Reputation = 0
	}
	s.Market.Reputation = math.Min(b.Contracts.ReputationMax, s.Market.Reputation)
}
