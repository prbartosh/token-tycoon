package sim

import (
	"math"

	"token-tycoon/internal/balance"
)

type ResearchState struct {
	Done     Bitset
	Active   int
	Progress float64
}

type ResearchMods struct {
	ComputeMult    float64
	EnergyCost     float64
	MaxParamsMult  float64
	GPUGen         int
	EnterpriseCap  float64
	ContractSlots  int
	ContractReward float64
	MaxModels      int
	RuleSlots      int
	AutoPrestige   bool
	OfflineEff     float64
	OfflineCapH    float64
	OfflineRules   bool
	IncidentMult   float64
	PenaltyMult    float64
	CritChance     float64
	CardChance     float64
	RareCards      float64
	ComplianceCost float64
}

func computeResearch(s *State, b *balance.Balance) {
	r := ResearchMods{ComputeMult: 1, EnergyCost: 1, MaxParamsMult: 1, IncidentMult: 1, PenaltyMult: 1, RareCards: 1, ComplianceCost: 1}
	for i := range b.ResearchNodes {
		if !s.Research.Done.Has(i) {
			continue
		}
		n := &b.ResearchNodes[i]
		switch n.Effect {
		case balance.ResComputeMult:
			r.ComputeMult *= n.Value
		case balance.ResEnergyCost:
			r.EnergyCost *= n.Value
		case balance.ResMaxParamsMult:
			r.MaxParamsMult *= n.Value
		case balance.ResGPUGen:
			r.GPUGen = max(r.GPUGen, int(n.Value))
		case balance.ResEnterprise:
			r.EnterpriseCap += n.Value
		case balance.ResContractSlots:
			r.ContractSlots += int(n.Value)
		case balance.ResContractReward:
			r.ContractReward += n.Value
		case balance.ResMaxModels:
			r.MaxModels += int(n.Value)
		case balance.ResRuleSlots:
			r.RuleSlots += int(n.Value)
		case balance.ResAutoPrestige:
			r.AutoPrestige = true
		case balance.ResOfflineEff:
			r.OfflineEff += n.Value
		case balance.ResOfflineCapH:
			r.OfflineCapH += n.Value
		case balance.ResOfflineRules:
			r.OfflineRules = true
		case balance.ResIncidentMult:
			r.IncidentMult *= n.Value
		case balance.ResPenaltyMult:
			r.PenaltyMult *= n.Value
		case balance.ResCritChance:
			r.CritChance += n.Value
		case balance.ResCardChance:
			r.CardChance += n.Value
		case balance.ResRareCards:
			r.RareCards *= n.Value
		case balance.ResComplianceCost:
			r.ComplianceCost *= n.Value
		}
	}
	s.ResMods = r
}

type NodeStatus uint8

const (
	NodeLocked NodeStatus = iota
	NodeAvailable
	NodeActive
	NodeDone
)

func NodeState(s *State, b *balance.Balance, i int) NodeStatus {
	switch {
	case i < 0 || i >= len(b.ResearchNodes):
		return NodeLocked
	case s.Research.Done.Has(i):
		return NodeDone
	case s.Research.Active == i:
		return NodeActive
	}
	for _, r := range b.NodeRequires(i) {
		if !s.Research.Done.Has(r) {
			return NodeLocked
		}
	}
	return NodeAvailable
}

func ResearchTimeS(s *State, b *balance.Balance, i int) float64 {
	if i < 0 || i >= len(b.ResearchNodes) {
		return 0
	}
	return b.ResearchNodes[i].TimeH * 3600 / s.Derived.Talent.Mult
}

func ResearchLeftS(s *State, b *balance.Balance) float64 {
	i := s.Research.Active
	if i < 0 {
		return 0
	}
	return ResearchTimeS(s, b, i) * math.Max(0, 1-s.Research.Progress)
}

func startResearch(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockResearch) {
		return reject(out, c, RejectLocked, b.Research.UnlockLifetime)
	}
	i := c.Index
	switch NodeState(s, b, i) {
	case NodeDone, NodeActive:
		return reject(out, c, RejectAlreadyOwned, 0)
	case NodeLocked:
		return reject(out, c, RejectNeedResearch, 0)
	}
	if s.Research.Active >= 0 {
		return reject(out, c, RejectBusy, 0)
	}
	cost := b.ResearchNodes[i].Cost
	if cost > s.Prestige.Capital {
		return reject(out, c, RejectNoCapital, cost-s.Prestige.Capital)
	}
	s.Prestige.Capital -= cost
	s.Research.Active, s.Research.Progress = i, 0
	out = append(out, Event{Kind: EvResearchStart, Business: -1, Index: i, Amount: cost})
	if !(b.ResearchNodes[i].TimeH > 0) {
		return finishResearch(s, b, out)
	}
	return out
}

func finishResearch(s *State, b *balance.Balance, out []Event) []Event {
	i := s.Research.Active
	s.Research.Done.Set(i)
	s.Research.Active, s.Research.Progress = -1, 0
	s.Stats.ResearchDone++
	computeResearch(s, b)
	out = updateGens(s, b, out)
	Recompute(s, b)
	return append(out, Event{Kind: EvResearchDone, Business: -1, Index: i})
}

func tickResearch(s *State, b *balance.Balance, dt float64, out []Event) []Event {
	i := s.Research.Active
	if i < 0 {
		return out
	}
	t := ResearchTimeS(s, b, i)
	if t > 0 {
		s.Research.Progress += dt / t
	}
	if t > 0 && s.Research.Progress < 1 {
		return out
	}
	return finishResearch(s, b, out)
}

func normalizeResearch(s *State, b *balance.Balance) {
	for i := len(b.ResearchNodes); i < len(s.Research.Done)*64; i++ {
		s.Research.Done.Clear(i)
	}
	r := &s.Research
	if r.Active < -1 || r.Active >= len(b.ResearchNodes) || (r.Active >= 0 && r.Done.Has(r.Active)) {
		r.Active = -1
	}
	if r.Active < 0 || !(r.Progress >= 0) {
		r.Progress = 0
	}
	r.Progress = math.Min(1, r.Progress)
}
