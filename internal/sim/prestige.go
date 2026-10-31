package sim

import (
	"math"
	"strconv"
	"strings"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
)

type PrestigePreview struct {
	Allowed      bool
	NeedRun      float64
	Gain         float64
	RawGain      float64
	ParamsNow    float64
	ParamsAfter  float64
	BonusNow     float64
	BonusAfter   float64
	RunS         float64
	ReturnS      float64
	VersionNow   int
	VersionAfter int
}

func Layer1Raw(s *State, b *balance.Balance) float64 {
	p := &b.Prestige.Layer1
	return econ.Layer1Params(s.Lifetime.Run, p.ParamCoef, p.ParamDivisor, p.ParamExponent)
}

func Layer1Gain(s *State, b *balance.Balance) float64 {
	p := &b.Prestige.Layer1
	return econ.Layer1Leap(Layer1Raw(s, b), s.Prestige.Params, p.ParamCoef, s.Clock-s.Prestige.RunStartS, p.LeapS)
}

// Preview estimates the time to return to the current run revenue after the
// reset. With roughly exponential growth, run revenue R and income I give the
// growth rate g = I / R; a permanent multiplier k shifts the curve by ln(k) / g.
// The linear-growth case, run time / k, is the lower bound.
func Preview(s *State, b *balance.Balance) PrestigePreview {
	p := &b.Prestige.Layer1
	pv := PrestigePreview{
		NeedRun:    p.UnlockRunRevenue,
		Gain:       Layer1Gain(s, b),
		RawGain:    Layer1Raw(s, b),
		ParamsNow:  s.Prestige.Params,
		RunS:       math.Max(0, s.Clock-s.Prestige.RunStartS),
		VersionNow: s.Prestige.VersionIndex,
	}
	pv.Allowed = s.IsUnlocked(UnlockPrestige1) && s.Lifetime.Run >= p.UnlockRunRevenue && pv.Gain > 0
	pv.ParamsAfter = pv.ParamsNow + pv.Gain
	pv.BonusNow = econ.Layer1Bonus(pv.ParamsNow, p.BonusPerParam)
	pv.BonusAfter = econ.Layer1Bonus(pv.ParamsAfter, p.BonusPerParam)
	pv.VersionAfter = pv.VersionNow + 1
	pv.ReturnS = pv.RunS
	if inc := s.Derived.BaseIncome; inc > 0 && s.Lifetime.Run > 0 && pv.BonusNow > 0 {
		g := inc / s.Lifetime.Run
		k := pv.BonusAfter / pv.BonusNow
		pv.ReturnS = math.Max(pv.RunS/k, pv.RunS-math.Log(k)/g)
	}
	return pv
}

func VersionName(b *balance.Balance, idx int) string {
	vs := b.Prestige.Layer1.Versions
	if idx < 0 {
		idx = 0
	}
	if idx < len(vs) {
		return vs[idx]
	}
	return "v" + strconv.Itoa(versionMajor(vs[len(vs)-1])+idx-len(vs)+1) + ".0"
}

func versionMajor(v string) int {
	v = strings.TrimPrefix(v, "v")
	end := 0
	for end < len(v) && v[end] >= '0' && v[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(v[:end])
	return n
}

func prestige(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	pv := Preview(s, b)
	if !pv.Allowed {
		return reject(out, c, RejectLocked, pv.NeedRun)
	}
	s.Prestige.Params += pv.Gain
	s.Prestige.ParamsLifetime += pv.Gain
	s.Prestige.VersionIndex++
	s.Prestige.ResetsL1++
	if pv.RunS > 0 && (s.Stats.FastestRunS == 0 || pv.RunS < s.Stats.FastestRunS) {
		s.Stats.FastestRunS = pv.RunS
	}
	s.Prestige.RunStartS = s.Clock
	s.Money = b.Start.Money
	s.Lifetime.Run = 0
	resetBusinesses(s, b)
	s.Upgrades = Bitset{}
	s.GlobalMilestones = 0
	s.Infra = Infra{GPUGen: s.Infra.GPUGen}
	s.Combo = Combo{Mult: 1}
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvPrestige, Business: -1, Index: s.Prestige.VersionIndex, Amount: pv.Gain, Value: s.Prestige.Params})
}
