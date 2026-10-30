package sim

import (
	"math"
	"reflect"
	"testing"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/market"
	"token-tycoon/internal/training"
)

func m3State(t testing.TB, b *balance.Balance) State {
	s := infraState(t, b)
	s.Lifetime.Total = b.Training.UnlockLifetime
	s.Clock = 2 * market.Day
	Step(&s, b, 0.1, nil, nil)
	if !s.IsUnlocked(UnlockMarket) || !s.IsUnlocked(UnlockTraining) || !s.IsUnlocked(UnlockData) {
		t.Fatal("M3 systems not unlocked")
	}
	s.Infra.Racks[3] = 16
	s.Infra.Energy.GridMW = 1e4
	s.Money = 1e30
	Recompute(&s, b)
	return s
}

func hasEvent(evs []Event, k EventKind) (Event, bool) {
	for _, e := range evs {
		if e.Kind == k {
			return e, true
		}
	}
	return Event{}, false
}

func TestPriceBeforeMarket(t *testing.T) {
	b := bal(t)
	s := infraState(t, b)
	s.Market.Hype = 5
	Recompute(&s, b)
	if s.Derived.Price != b.Market.PriceBase || s.Derived.Market.On {
		t.Fatalf("price before market %v", s.Derived.Price)
	}
	evs := Step(&s, b, 0, []Command{SetSegments(1, 0, 0)}, nil)
	if evs[0].Reason != RejectLocked {
		t.Fatal("segments before market")
	}
}

func TestMarketPrice(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	mv := &s.Derived.Market
	if !mv.On || mv.Rivals != len(b.Rivals) || mv.Model[market.SegPro] != -1 {
		t.Fatalf("market view %+v", mv)
	}
	q := mv.Quote[market.SegPro]
	want := b.Market.PriceBase * q.Quality * q.Demand * q.Deflation * mv.Pressure
	if math.Abs(s.Derived.Price-want) > 1e-12*want || q.Deflation != 1 {
		t.Fatalf("price %v want %v", s.Derived.Price, want)
	}
	before := s.Derived.PerSec[0]
	Step(&s, b, 0, []Command{SetSegments(0.5, 0.5, 0)}, nil)
	if r := s.Derived.PerSec[0] / before; math.Abs(r-0.5) > 1e-9 {
		t.Fatalf("half free must halve revenue, got x%v", r)
	}
	if !(mv.HypeGain > 0) {
		t.Fatal("free segment gives no hype")
	}
	h0 := s.Market.Hype
	for range 100 {
		Step(&s, b, 60, nil, nil)
	}
	if !(s.Market.Hype > h0) {
		t.Fatal("hype did not grow")
	}
	evs := Step(&s, b, 0, []Command{SetSegments(0, 0.5, 0.5)}, nil)
	if evs[0].Reason != RejectNeedBenchmark {
		t.Fatal("enterprise without a 70 pts model")
	}
}

func TestEnterpriseAndModels(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 80, ReleasedS: s.Clock}}
	Recompute(&s, b)
	pro := s.Derived.Price
	Step(&s, b, 0, []Command{SetSegments(0, 0, 1)}, nil)
	limit := b.Market.Segments.EnterpriseMaxAlloc
	if s.Market.Segments.Enterprise != limit {
		t.Fatalf("enterprise cap: %+v", s.Market.Segments)
	}
	want := pro * (1 - limit + limit*b.Market.Segments.EnterprisePrice)
	if math.Abs(s.Derived.Price-want) > 1e-9*want {
		t.Fatalf("enterprise price %v want %v", s.Derived.Price, want)
	}
	s.Training.Models = append(s.Training.Models, Model{Name: "B", Params: 1e8, Benchmark: 40, ReleasedS: s.Clock})
	Recompute(&s, b)
	if s.Derived.Market.EnterpriseOK {
		t.Fatal("latest 40 pts model serves enterprise")
	}
	Step(&s, b, 0, []Command{AssignModel(0, "enterprise")}, nil)
	if !s.Derived.Market.EnterpriseOK || s.Derived.Market.Model[market.SegEnterprise] != 0 || s.Derived.Market.Model[market.SegPro] != 1 {
		t.Fatalf("assignment: %+v", s.Derived.Market.Model)
	}
	if evs := Step(&s, b, 0, []Command{AssignModel(1, "enterprise")}, nil); evs[0].Reason != RejectNeedBenchmark {
		t.Fatal("40 pts model assigned to enterprise")
	}
	Step(&s, b, 0, []Command{RetireModel(1)}, nil)
	if len(s.Training.Models) != 1 {
		t.Fatal("retire")
	}
}

func TestDeflationAndPressure(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 50, ReleasedS: s.Clock}}
	Recompute(&s, b)
	p0 := s.Derived.Price
	s.Clock += market.Day
	Recompute(&s, b)
	q := s.Derived.Market.Quote[market.SegPro]
	if math.Abs(q.Deflation-b.Market.DeflationPerDay) > 1e-12 {
		t.Fatalf("deflation after a day %v", q.Deflation)
	}
	if !(s.Derived.Price < p0) {
		t.Fatal("price did not fall")
	}
	s.Clock += 10 * market.Day
	Recompute(&s, b)
	if !(s.Derived.Market.Pressure < 1) {
		t.Fatal("open-weights releases do not press the price")
	}
}

func TestRivalReleaseEvents(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	seen := map[int]bool{}
	var evs []Event
	for range 5 * 24 * 60 {
		evs = Step(&s, b, 60, nil, evs[:0])
		for _, e := range evs {
			if e.Kind == EvRivalRelease {
				seen[e.Index] = true
				if e.Value != s.Derived.Market.RivalBench[e.Index] {
					t.Fatalf("release value %v vs view %v", e.Value, s.Derived.Market.RivalBench[e.Index])
				}
			}
		}
	}
	if len(seen) != len(b.Rivals) {
		t.Fatalf("rivals released in 5 days: %v", seen)
	}
}

func TestTrainingLifecycle(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	n, d := 1e8, 2e9
	if evs := Step(&s, b, 0, []Command{Train(n, d, 0.5)}, nil); evs[0].Reason != RejectNoData {
		t.Fatalf("training without data: %+v", evs[0])
	}
	if evs := Step(&s, b, 0, []Command{Train(1e15, d, 0.5)}, nil); evs[0].Reason != RejectTooLarge {
		t.Fatal("max params")
	}
	s.Training.DataTokens = 3 * d
	capBefore := s.Derived.ComputeCap
	evs := Step(&s, b, 0, []Command{Train(n, d, 0.5)}, nil)
	if _, ok := hasEvent(evs, EvTrainQueued); !ok || s.Training.DataTokens != 2*d {
		t.Fatalf("queue: %+v", evs)
	}
	if math.Abs(s.Derived.ComputeCap-capBefore*0.5) > 1e-9*capBefore {
		t.Fatalf("training share not taken from production: %v of %v", s.Derived.ComputeCap, capBefore)
	}
	tv := s.Derived.Train
	if !(tv.TrainFLOPS > 0) || math.Abs(tv.ETA-training.FLOPs(n, d, &b.Training)/tv.TrainFLOPS) > 1e-6*tv.ETA {
		t.Fatalf("train view %+v", tv)
	}
	Step(&s, b, 0, []Command{Train(n, d, 0.5), Train(n, d, 0.5)}, nil)
	if evs := Step(&s, b, 0, []Command{Train(n, d, 0.5)}, nil); evs[0].Reason != RejectNoData && evs[0].Reason != RejectQueueFull {
		t.Fatal("queue limit")
	}
	Step(&s, b, 0, []Command{CancelTrain(2)}, nil)
	if len(s.Training.Queue) != 2 || s.Training.DataTokens != d {
		t.Fatalf("cancel: queue %d data %v", len(s.Training.Queue), s.Training.DataTokens)
	}
	var rel []Event
	for k := 0; k < 100000 && len(rel) < 2; k++ {
		for _, e := range Step(&s, b, 1, nil, nil) {
			if e.Kind == EvModelRelease {
				rel = append(rel, e)
			}
		}
	}
	if len(rel) != 2 || len(s.Training.Models) != 2 || len(s.Training.Queue) != 0 {
		t.Fatalf("releases %d models %d", len(rel), len(s.Training.Models))
	}
	want := training.Benchmark(n, d, 1, &b.Training)
	if math.Abs(rel[0].Value-want) > 1e-9 || s.Training.Models[0].Name != "TT-1" {
		t.Fatalf("benchmark %v want %v", rel[0].Value, want)
	}
	if s.Derived.ComputeCap != capBefore || s.Derived.Train.TrainFLOPS != 0 {
		t.Fatal("cluster not returned to production")
	}
	if !(s.Market.Hype > 0) {
		t.Fatal("release adds no hype")
	}
}

func TestMaxModelsRetiresWorst(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Training.Models = []Model{{Name: "a", Benchmark: 30}, {Name: "b", Benchmark: 20}, {Name: "c", Benchmark: 40}}
	s.Training.Released = 3
	s.Training.Queue = []TrainJob{{Params: 1e6, Tokens: 1e6, ClusterShare: 1, Progress: 0.999999, ArchMult: 1}}
	Recompute(&s, b)
	evs := Step(&s, b, 1, nil, nil)
	r, ok := hasEvent(evs, EvModelRetired)
	if !ok || r.Index != 1 || len(s.Training.Models) != 3 || s.Training.Models[2].Name != "TT-4" {
		t.Fatalf("retire worst: %+v %+v", evs, s.Training.Models)
	}
}

func TestDataSources(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	cost := ScraperCost(&s, b, 3)
	m0 := s.Money
	Step(&s, b, 0, []Command{BuyScraper(3)}, nil)
	if s.Training.Scrapers != 3 || s.Money != m0-cost {
		t.Fatal("scrapers")
	}
	s.Training.ScraperPauseS = 0
	d0 := s.Training.DataTokens
	Offline(&s, b, 3600, nil)
	if got := s.Training.DataTokens - d0; math.Abs(got-3*b.Data.ScraperTokensPerS*3600) > 1 {
		t.Fatalf("scraped %v", got)
	}
	if s.Stats.Incidents != 0 {
		t.Fatal("incident offline")
	}
	lc := s.Derived.Train.LicenseCost
	if math.Abs(lc-MinutesOfIncome(&s, b.Data.LicenseCostMinutes)) > 1e-9*lc {
		t.Fatal("license cost")
	}
	d1 := s.Training.DataTokens
	Step(&s, b, 0, []Command{BuyLicense()}, nil)
	if s.Training.DataTokens-d1 != b.Data.LicenseTokens[0] {
		t.Fatal("license tokens")
	}
	if evs := Step(&s, b, 0, []Command{SetSynthetic(0.2)}, nil); evs[0].Reason != RejectNeedBenchmark {
		t.Fatal("synthetic without a 60 pts model")
	}
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 65}}
	Recompute(&s, b)
	Step(&s, b, 0, []Command{SetSynthetic(0.2)}, nil)
	if s.Derived.Train.SynthShare != 0.2 || !(s.Derived.Train.SynthPerSec > 0) {
		t.Fatalf("synthetic view %+v", s.Derived.Train)
	}
	syn0 := s.Training.SynthTokens
	Step(&s, b, 10, nil, nil)
	if !(s.Training.SynthTokens > syn0) {
		t.Fatal("no synthetic data")
	}
	s.Training.DataTokens, s.Training.SynthTokens = 1e9, 3e9
	clean := EstimateTraining(&s, b, 1e8, 2e9, 0.5)
	dirty := EstimateTraining(&s, b, 1e8, 4e9, 0.5)
	if clean.SynthFrac != 0.5 || dirty.SynthFrac != 0.75 || !(dirty.Benchmark < training.Benchmark(1e8, 4e9, 1, &b.Training)) {
		t.Fatalf("synthetic penalty: %+v %+v", clean, dirty)
	}
	if clean.Optimal != 0 {
		t.Fatal("optimal ratio shown without the card")
	}
	s.Cards["chinchilla"] = 1
	Normalize(&s, b)
	if e := EstimateTraining(&s, b, 1e8, 2e9, 0.5); !(e.Optimal > 0) {
		t.Fatal("chinchilla card does not reveal")
	}
}

func TestScraperIncidents(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Training.Scrapers = 200
	var evs []Event
	got := false
	for range 36000 {
		evs = Step(&s, b, 0.1, nil, evs[:0])
		if e, ok := hasEvent(evs, EvScraperIncident); ok {
			got = true
			if !s.Derived.Train.ScraperPaused || e.Value != b.Data.ScraperIncidentPauseS {
				t.Fatal("incident does not pause scraping")
			}
			break
		}
	}
	if !got {
		t.Fatal("no incident in an hour with 200 scrapers")
	}
}

func TestArchitectureFromCards(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	if ArchMult(&s, b) != 1 {
		t.Fatal("arch without cards")
	}
	s.Cards["moe_paper"] = 1
	s.Cards["bpe"] = 2
	Normalize(&s, b)
	want := b.Training.Architecture["moe"] * b.Training.Architecture["tokenizer_v2"]
	if math.Abs(s.Derived.Train.ArchMult-want) > 1e-12 {
		t.Fatalf("arch %v want %v", s.Derived.Train.ArchMult, want)
	}
	s.Training.DataTokens = 1e10
	Step(&s, b, 0, []Command{Train(1e8, 1e9, 1)}, nil)
	if j := s.Training.Queue[0]; j.ArchMult != want || len(j.Arch) != 2 {
		t.Fatalf("job arch %+v", j)
	}
}

func TestTrainingSurvivesPrestige(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Lifetime.Run = 1e16
	s.Unlocks.Set(int(UnlockPrestige1))
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 50}}
	s.Training.Scrapers = 4
	s.Training.DataTokens = 5e9
	Step(&s, b, 0, []Command{Train(1e8, 2e9, 0.5)}, nil)
	Step(&s, b, 0, []Command{PrestigeReset()}, nil)
	if s.Prestige.ResetsL1 != 1 || len(s.Training.Models) != 1 || len(s.Training.Queue) != 1 || s.Training.Scrapers != 4 {
		t.Fatalf("training lost on prestige: %+v", s.Training)
	}
}

func TestStepNoAllocsM3(t *testing.T) {
	b := bal(t)
	s := m3State(t, b)
	s.Training.Models = []Model{{Name: "A", Params: 1e9, Benchmark: 70, ReleasedS: s.Clock}}
	s.Training.Scrapers = 5
	s.Training.DataTokens = 1e30
	Step(&s, b, 0, []Command{SetSegments(0.2, 0.8, 0), SetSynthetic(0.1), Train(1e8, 1e29, 0.3)}, nil)
	evs := make([]Event, 0, 256)
	cmds := []Command{Boost(0)}
	for range 3000 {
		evs = Step(&s, b, 0.1, cmds, evs[:0])
	}
	allocs := testing.AllocsPerRun(2000, func() {
		evs = Step(&s, b, 0.1, cmds, evs[:0])
	})
	if allocs > 0 {
		t.Fatalf("Step allocates %v times per call", allocs)
	}
}

func TestDeterminismM3(t *testing.T) {
	b := bal(t)
	runIt := func() State {
		s := m3State(t, b)
		s.Training.DataTokens = 1e12
		evs := make([]Event, 0, 64)
		for k := range 20000 {
			var cmds []Command
			switch k {
			case 0:
				cmds = []Command{BuyScraper(50), SetSegments(0.3, 0.7, 0), Train(1e8, 2e9, 0.4)}
			case 5000:
				cmds = []Command{Train(1e8, 4e9, 0.6)}
			}
			evs = Step(&s, b, 0.5, cmds, evs[:0])
		}
		return s
	}
	a, c := runIt(), runIt()
	if !reflect.DeepEqual(a, c) {
		t.Fatal("nondeterministic")
	}
	if a.Stats.Models == 0 {
		t.Fatal("nothing trained")
	}
}
