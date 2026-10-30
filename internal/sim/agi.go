package sim

import (
	"math"

	"token-tycoon/internal/balance"
)

type AGIView struct {
	Stage      int
	Stages     int
	Cost       float64
	Invested   float64
	Training   bool
	TrainLeftS float64
	Ended      bool
	SILevel    int
	NextSI     float64
}

func AGIDone(s *State, b *balance.Balance, effect string) bool {
	for i := 0; i < s.AGI.Stage && i < len(b.AGIStages); i++ {
		if b.AGIStages[i].Effect == effect {
			return true
		}
	}
	return false
}

func AGIEnded(s *State, b *balance.Balance) bool {
	return len(b.AGIStages) > 0 && s.AGI.Stage >= len(b.AGIStages)
}

func AGIProfitMult(s *State, b *balance.Balance) float64 {
	m := 1.0
	if AGIDone(s, b, balance.AGIProfitX2) {
		m *= 2
	}
	if s.AGI.SILevel > 0 {
		m *= 1 + float64(s.AGI.SILevel)*b.AGI.SIBonusPerLevel
	}
	return m
}

func siBase(b *balance.Balance) float64 {
	if len(b.AGIStages) == 0 {
		return 0
	}
	return math.Log10(b.AGIStages[len(b.AGIStages)-1].Cost)
}

func NextSIThreshold(s *State, b *balance.Balance) float64 {
	return math.Pow(10, siBase(b)+float64(s.AGI.SILevel+1)*b.AGI.SIEveryOOM)
}

func AGIStatus(s *State, b *balance.Balance) AGIView {
	a := &s.AGI
	v := AGIView{Stage: a.Stage, Stages: len(b.AGIStages), Invested: a.Invested, Ended: AGIEnded(s, b), SILevel: a.SILevel}
	if !v.Ended && a.Stage < len(b.AGIStages) {
		v.Cost = b.AGIStages[a.Stage].Cost
	}
	if a.TrainEndsS > 0 {
		v.Training = true
		v.TrainLeftS = math.Max(0, a.TrainEndsS-s.Clock)
	}
	if v.Ended && a.SILevel < b.AGI.SIMaxLevels {
		v.NextSI = NextSIThreshold(s, b)
	}
	return v
}

func agiInvest(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockAGI) {
		return reject(out, c, RejectLocked, b.AGI.UnlockLifetime)
	}
	a := &s.AGI
	if AGIEnded(s, b) {
		return reject(out, c, RejectFinished, 0)
	}
	if a.TrainEndsS > 0 {
		return reject(out, c, RejectBusy, 0)
	}
	st := &b.AGIStages[a.Stage]
	need := st.Cost - a.Invested
	if !(s.Money > 0) {
		return reject(out, c, RejectCannotAfford, need)
	}
	x := math.Min(s.Money, need)
	s.Money -= x
	a.Invested += x
	s.Stats.AGIInvested += x
	s.Undo = Undo{}
	out = append(out, Event{Kind: EvAGIInvest, Business: -1, Index: a.Stage, Amount: x, Value: a.Invested / st.Cost})
	if a.Invested < st.Cost*(1-1e-12) {
		return out
	}
	a.Invested = st.Cost
	if st.Effect == balance.AGIFinalTraining {
		a.TrainEndsS = s.Clock + b.AGI.FinalTrainingH*3600
		return append(out, Event{Kind: EvAGITraining, Business: -1, Index: a.Stage, Value: b.AGI.FinalTrainingH * 3600})
	}
	return completeStage(s, b, out)
}

func completeStage(s *State, b *balance.Balance, out []Event) []Event {
	a := &s.AGI
	i := a.Stage
	a.Stage++
	a.Invested, a.TrainEndsS = 0, 0
	if b.AGIStages[i].Effect == balance.AGINoIncidents {
		s.Incident = Incident{}
	}
	out = append(out, Event{Kind: EvAGIStage, Business: -1, Index: i})
	if b.AGIStages[i].Effect == balance.AGIEnding {
		a.EndedS = s.Clock
		out = append(out, Event{Kind: EvAGIEnding, Business: -1, Index: i, Value: s.Clock})
	}
	Recompute(s, b)
	return out
}

func tickAGI(s *State, b *balance.Balance, out []Event) []Event {
	a := &s.AGI
	if a.TrainEndsS > 0 && s.Clock >= a.TrainEndsS {
		out = addModel(s, b, Model{Name: "AGI-1", Benchmark: b.AGI.FinalBenchmark, ReleasedS: s.Clock}, out)
		out = completeStage(s, b, out)
	}
	if !AGIEnded(s, b) || a.SILevel >= b.AGI.SIMaxLevels || s.Lifetime.Total < NextSIThreshold(s, b) {
		return out
	}
	a.SIRecords[a.SILevel] = s.Clock - a.EndedS
	a.SILevel++
	Recompute(s, b)
	return append(out, Event{Kind: EvSILevel, Business: -1, Index: a.SILevel, Value: a.SIRecords[a.SILevel-1]})
}

func normalizeAGI(s *State, b *balance.Balance) {
	a := &s.AGI
	a.Stage = min(max(0, a.Stage), len(b.AGIStages))
	bad := func(x float64) bool { return !(x >= 0) || math.IsInf(x, 0) }
	if bad(a.Invested) {
		a.Invested = 0
	}
	if a.Stage < len(b.AGIStages) {
		a.Invested = math.Min(a.Invested, b.AGIStages[a.Stage].Cost)
		if b.AGIStages[a.Stage].Effect != balance.AGIFinalTraining {
			a.TrainEndsS = 0
		}
	} else {
		a.Invested, a.TrainEndsS = 0, 0
	}
	if bad(a.TrainEndsS) {
		a.TrainEndsS = 0
	}
	if bad(a.EndedS) {
		a.EndedS = 0
	}
	a.SILevel = min(max(0, a.SILevel), b.AGI.SIMaxLevels)
	if !AGIEnded(s, b) {
		a.SILevel = 0
	}
	for k := range a.SIRecords {
		if k >= a.SILevel || bad(a.SIRecords[k]) {
			a.SIRecords[k] = 0
		}
	}
	if bad(s.Stats.AGIInvested) {
		s.Stats.AGIInvested = 0
	}
}
