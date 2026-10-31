package sim

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
)

type FundingPreview struct {
	Allowed      bool
	Need         float64
	Have         float64
	Gain         float64
	CapitalNow   float64
	CapitalAfter float64
	RoundNow     int
	RoundAfter   int
	Valuation    float64
	RoundS       float64
	Bonus        float64
	BaseGain     float64
	RoundMult    float64
	RoundMultNew float64
}

const yearS = 365 * 86400

func baseFundingGain(s *State, b *balance.Balance) float64 {
	p := &b.Prestige.Layer2
	return econ.Layer2Capital(s.Lifetime.Layer2Total, p.CapitalCoef, p.CapitalLogOffset, p.CapitalExponent)
}

func FundingGain(s *State, b *balance.Balance) float64 {
	g := baseFundingGain(s, b)
	if g > 0 && s.Investors.Bonus > 0 {
		g = math.Floor(g * (1 + s.Investors.Bonus))
	}
	return g
}

func RoundBonus(s *State) float64 { return math.Max(1, s.Prestige.RoundMult) }

func Funding(s *State, b *balance.Balance) FundingPreview {
	p := &b.Prestige.Layer2
	pv := FundingPreview{
		Need: p.UnlockLifetime, Have: s.Lifetime.Layer2Total, Gain: FundingGain(s, b),
		CapitalNow: s.Prestige.Capital, RoundNow: s.Prestige.RoundIndex, RoundAfter: s.Prestige.RoundIndex + 1,
		RoundS: math.Max(0, s.Clock-s.Prestige.RoundStartS),
		Bonus:  s.Investors.Bonus, BaseGain: baseFundingGain(s, b),
	}
	pv.Allowed = s.IsUnlocked(UnlockLayer2) && pv.Have >= pv.Need && pv.Gain > 0
	pv.CapitalAfter = pv.CapitalNow + pv.Gain
	pv.Valuation = Valuation(s, b).Valuation
	pv.RoundMult = RoundBonus(s)
	pv.RoundMultNew = econ.RoundBank(pv.RoundMult, econ.Layer1Bonus(s.Prestige.Params, b.Prestige.Layer1.BonusPerParam), p.RoundBankShare)
	return pv
}

func RoundID(b *balance.Balance, idx int) (id string, mega int) {
	rs := b.Prestige.Layer2.Rounds
	idx = max(0, idx)
	if idx < len(rs) {
		return rs[idx], 0
	}
	return "", idx - len(rs) + 1
}

func fundingRound(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	pv := Funding(s, b)
	if !pv.Allowed {
		return reject(out, c, RejectLocked, pv.Need)
	}
	pr := &s.Prestige
	pr.Capital += pv.Gain
	pr.CapitalLifetime += pv.Gain
	round := pr.RoundIndex
	pr.RoundIndex++
	pr.ResetsL2++
	pr.RoundMult = pv.RoundMultNew
	s.Investors.Bonus = 0
	pr.Params = 0
	pr.VersionIndex = 0
	pr.RunStartS, pr.RoundStartS = s.Clock, s.Clock
	s.Money = b.Start.Money
	s.Lifetime.Run = 0
	s.Lifetime.Layer2Total = 0
	resetBusinesses(s, b)
	s.Upgrades = Bitset{}
	s.GlobalMilestones = 0
	s.Infra = Infra{GPUGen: s.Infra.GPUGen}
	s.Combo = Combo{Mult: 1}
	s.Undo = Undo{}
	s.Contracts = s.Contracts[:0]
	s.Offers = s.Offers[:0]
	s.NextOfferS = 0
	Recompute(s, b)
	return append(out, Event{Kind: EvFunding, Business: -1, Index: round, Amount: pv.Gain, Value: pv.Valuation})
}
