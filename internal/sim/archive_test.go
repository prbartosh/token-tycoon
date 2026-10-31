package sim

import (
	"math"
	"testing"
)

func fullModels(t *testing.T) (*State, func(manual bool)) {
	t.Helper()
	b := bal(t)
	st := m3State(t, b)
	s := &st
	s.Training.Models = []Model{{Name: "a", Benchmark: 30, Segment: "pro"}, {Name: "b", Benchmark: 20}, {Name: "c", Benchmark: 40}}
	s.Training.Released = 3
	Recompute(s, b)
	queue := func(manual bool) {
		s.Training.Queue = append(s.Training.Queue, TrainJob{Params: 1e9, Tokens: 2e10, ClusterShare: 1, Progress: 0.999999, ArchMult: 1, Manual: manual})
		Recompute(s, b)
	}
	return s, queue
}

func TestManualTrainMarksJob(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Training.DataTokens = 1e12
	Step(&s, b, 0, []Command{ManualTrain(1e7, 2e8, 0.5), Train(1e7, 2e8, 0.5)}, nil)
	if len(s.Training.Queue) != 2 || !s.Training.Queue[0].Manual || s.Training.Queue[1].Manual {
		t.Fatalf("queue %+v", s.Training.Queue)
	}
}

func TestOnlineReleaseWaitsForChoice(t *testing.T) {
	b := bal(t)
	s, queue := fullModels(t)
	queue(true)
	evs := Step(s, b, 1, nil, nil)
	p, ok := hasEvent(evs, EvReleasePending)
	if !ok || len(s.Training.Pending) != 1 || len(s.Training.Models) != 3 || len(s.Training.Archive) != 0 {
		t.Fatalf("pending: %+v models %+v", evs, s.Training.Models)
	}
	if _, ok := hasEvent(evs, EvModelRelease); ok {
		t.Fatal("released before the player chose")
	}
	md, ok := PendingRelease(s)
	if !ok || md.Name != "TT-4" || p.Value != md.Benchmark || s.Stats.Models != 1 {
		t.Fatalf("pending model %+v", md)
	}
	for range 50 {
		Step(s, b, 1, nil, nil)
	}
	if len(s.Training.Pending) != 1 || len(s.Training.Models) != 3 {
		t.Fatal("pending model resolved without the player")
	}
	evs = Step(s, b, 0, []Command{ResolveRelease(0)}, nil)
	if _, ok := hasEvent(evs, EvModelRelease); !ok {
		t.Fatalf("no release after the choice: %+v", evs)
	}
	if len(s.Training.Pending) != 0 || len(s.Training.Models) != 3 || s.Training.Models[2].Name != "TT-4" ||
		len(s.Training.Archive) != 1 || s.Training.Archive[0].Name != "a" || s.Training.Archive[0].Segment != "" {
		t.Fatalf("after choice: models %+v archive %+v", s.Training.Models, s.Training.Archive)
	}
}

func TestOnlineReleaseRejectNew(t *testing.T) {
	b := bal(t)
	s, queue := fullModels(t)
	queue(true)
	Step(s, b, 1, nil, nil)
	evs := Step(s, b, 0, []Command{ResolveRelease(-1)}, nil)
	a, ok := hasEvent(evs, EvModelArchived)
	if !ok || a.Count != ArchivedNew || len(s.Training.Archive) != 1 || s.Training.Archive[0].Name != "TT-4" ||
		s.Training.Models[0].Name != "a" || s.Training.Models[1].Name != "b" {
		t.Fatalf("reject new: %+v %+v", s.Training.Models, s.Training.Archive)
	}
	if evs = Step(s, b, 0, []Command{ResolveRelease(0)}, nil); evs[0].Reason != RejectInvalid {
		t.Fatal("resolve without pending accepted")
	}
}

func TestOfflineAndPolicyReleaseAuto(t *testing.T) {
	b := bal(t)
	for _, tc := range []struct {
		name    string
		manual  bool
		offline bool
	}{{"offline", true, true}, {"policy", false, false}} {
		s, queue := fullModels(t)
		queue(tc.manual)
		var evs []Event
		if tc.offline {
			_, evs = Offline(s, b, 120, nil)
		} else {
			evs = Step(s, b, 1, nil, nil)
		}
		if len(s.Training.Pending) != 0 || len(s.Training.Models) != 3 || len(s.Training.Archive) != 1 || s.Training.Archive[0].Name != "b" {
			t.Fatalf("%s better: models %+v archive %+v", tc.name, s.Training.Models, s.Training.Archive)
		}
		if _, ok := hasEvent(evs, EvModelRelease); !ok {
			t.Fatalf("%s: better model not released", tc.name)
		}
		s.Training.Queue = append(s.Training.Queue, TrainJob{Params: 1e6, Tokens: 1e6, ClusterShare: 1, Progress: 0.999999, ArchMult: 1, Manual: tc.manual})
		Recompute(s, b)
		if tc.offline {
			Offline(s, b, 120, nil)
		} else {
			Step(s, b, 1, nil, nil)
		}
		if len(s.Training.Models) != 3 || len(s.Training.Archive) != 2 || s.Training.Archive[1].Name != "TT-5" {
			t.Fatalf("%s worse: models %+v archive %+v", tc.name, s.Training.Models, s.Training.Archive)
		}
	}
}

func TestFreeSlotReleasesDirectly(t *testing.T) {
	b := bal(t)
	s, queue := fullModels(t)
	s.Training.Models = s.Training.Models[:2]
	queue(true)
	evs := Step(s, b, 1, nil, nil)
	if _, ok := hasEvent(evs, EvModelRelease); !ok || len(s.Training.Models) != 3 || len(s.Training.Pending) != 0 {
		t.Fatalf("free slot: %+v", s.Training)
	}
}

func TestArchiveSwapAndDelete(t *testing.T) {
	b := bal(t)
	s, _ := fullModels(t)
	s.Training.Archive = []Model{{Name: "z", Benchmark: 80, ReleasedS: 5}, {Name: "y", Benchmark: 10}}
	hype := s.Market.Hype
	evs := Step(s, b, 0, []Command{SwapArchived(0, 0)}, nil)
	sw, ok := hasEvent(evs, EvArchiveSwap)
	if !ok || sw.Index != 0 || s.Training.Models[0].Name != "z" || s.Training.Models[0].Segment != "pro" ||
		s.Training.Archive[0].Name != "a" || s.Training.Archive[0].Segment != "" || s.Training.Models[0].ReleasedS != 5 {
		t.Fatalf("swap: %+v %+v", s.Training.Models, s.Training.Archive)
	}
	if s.Market.Hype != hype {
		t.Fatal("swapping a model in adds hype")
	}
	if BestBenchmark(s, b) != 80 {
		t.Fatalf("best benchmark after swap %v", BestBenchmark(s, b))
	}
	if evs = Step(s, b, 0, []Command{SwapArchived(1, -1)}, nil); evs[0].Reason != RejectTeamFull {
		t.Fatal("swap into full slots without a target accepted")
	}
	Step(s, b, 0, []Command{DeleteArchived(1)}, nil)
	if len(s.Training.Archive) != 1 || s.Training.Archive[0].Name != "a" {
		t.Fatalf("delete: %+v", s.Training.Archive)
	}
	s.Training.Models = s.Training.Models[:2]
	Step(s, b, 0, []Command{SwapArchived(0, -1)}, nil)
	if len(s.Training.Models) != 3 || len(s.Training.Archive) != 0 || s.Training.Models[2].Name != "a" {
		t.Fatalf("swap into a free slot: %+v %+v", s.Training.Models, s.Training.Archive)
	}
	for _, c := range []Command{SwapArchived(3, 0), DeleteArchived(-1)} {
		if evs = Step(s, b, 0, []Command{c}, nil); evs[0].Reason != RejectInvalid {
			t.Fatalf("bad index accepted: %+v", c)
		}
	}
}

func TestNormalizeKeepsEveryModel(t *testing.T) {
	b := bal(t)
	s, _ := fullModels(t)
	s.Training.Models = append(s.Training.Models, Model{Name: "d", Benchmark: 50})
	s.Training.Pending = []Model{{Name: "p", Benchmark: 60}}
	Normalize(s, b)
	if len(s.Training.Models) != 3 || len(s.Training.Archive) != 1 || s.Training.Archive[0].Name != "a" || len(s.Training.Pending) != 1 {
		t.Fatalf("over limit: %+v %+v", s.Training.Models, s.Training.Archive)
	}
	s.Training.Models = s.Training.Models[:2]
	Normalize(s, b)
	if len(s.Training.Models) != 3 || len(s.Training.Pending) != 0 || s.Training.Models[2].Name != "p" {
		t.Fatalf("pending into a free slot: %+v", s.Training)
	}
}

func TestContractDeliveryIsSteady(t *testing.T) {
	b := bal(t)
	s := m4State(t, b)
	for i := range 6 {
		s.Businesses[i].Owned, s.Businesses[i].Manager = 50, true
	}
	s.Contracts = []Contract{{ID: 1, Volume: 1e300, DurationS: 1e6, DeadlineS: s.Clock + 1e6}}
	Recompute(&s, b)
	want := SteadyTokensPerSec(&s)
	if !(want > 0) || !(s.Derived.CritEV > 1) {
		t.Fatalf("steady %v crit ev %v", want, s.Derived.CritEV)
	}
	for range 20 {
		Step(&s, b, 0, []Command{Boost(-1)}, nil)
	}
	forceEvent(&s, b, "viral_demo")
	Recompute(&s, b)
	if s.Combo.Mult <= 1 {
		t.Fatal("combo not active")
	}
	before := s.Lifetime.Total
	Step(&s, b, 1, nil, nil)
	got := s.Contracts[0].Delivered
	if math.Abs(got-want)/want > 1e-9 {
		t.Fatalf("delivered %v, steady %v (earned %v)", got, want, s.Lifetime.Total-before)
	}
	if s.Derived.TokensPerSec/s.Derived.CritEV != SteadyTokensPerSec(&s) {
		t.Fatal("steady rate not crit free")
	}
}

func TestAGIInvestPortion(t *testing.T) {
	b := bal(t)
	s := m5State(t, b)
	s.Money = b.AGIStages[0].Cost / 4
	m0 := s.Money
	Step(&s, b, 0, []Command{{Kind: CmdAGIInvest, Business: -1, Share: 0.1}}, nil)
	if math.Abs(s.AGI.Invested-m0*0.1) > 1e-9*m0 || math.Abs(s.Money-m0*0.9) > 1e-9*m0 {
		t.Fatalf("10%%: invested %v money %v", s.AGI.Invested, s.Money)
	}
	Step(&s, b, 0, []Command{{Kind: CmdAGIInvest, Business: -1, Share: 0.5}}, nil)
	if math.Abs(s.Money-m0*0.45) > 1e-9*m0 {
		t.Fatalf("50%%: money %v", s.Money)
	}
	s.Money = b.AGIStages[0].Cost * 3
	Step(&s, b, 0, []Command{{Kind: CmdAGIInvest, Business: -1, Share: 1}}, nil)
	if s.AGI.Stage != 1 || math.Abs(s.Money-(b.AGIStages[0].Cost*3-(b.AGIStages[0].Cost-m0*0.55))) > 1e-6*s.Money {
		t.Fatalf("missing amount: stage %d money %v", s.AGI.Stage, s.Money)
	}
}

func TestLateDecisionReleasesFresh(t *testing.T) {
	b := bal(t)
	s, queue := fullModels(t)
	queue(true)
	Step(s, b, 1, nil, nil)
	if len(s.Training.Pending) != 1 {
		t.Fatalf("pending %+v", s.Training.Pending)
	}
	for range 100 {
		Step(s, b, 60, nil, nil)
	}
	Step(s, b, 0, []Command{ResolveRelease(1)}, nil)
	md := s.Training.Models[len(s.Training.Models)-1]
	if md.ReleasedS != s.Clock {
		t.Fatalf("model waiting for a decision aged on the market: released %v, clock %v", md.ReleasedS, s.Clock)
	}
}
