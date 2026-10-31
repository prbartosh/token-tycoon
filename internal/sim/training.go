package sim

import (
	"math"
	"strconv"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/econ"
	"token-tycoon/internal/market"
	"token-tycoon/internal/training"
)

type TrainView struct {
	ClusterFLOPS  float64
	TrainFLOPS    float64
	SynthFLOPS    float64
	TrainShare    float64
	SynthShare    float64
	EnergyFrac    float64
	ParamsLevel   int
	MaxParams     float64
	ArchMult      float64
	BestBenchmark float64
	SynthOK       bool
	RevealRatio   bool
	ETA           float64
	QueueETA      float64
	DataPerSec    float64
	SynthPerSec   float64
	ScraperPaused bool
	LicenseCost   float64
	LicenseTokens float64
	MaxModels     int
}

type TrainEstimate struct {
	Benchmark float64
	FLOPs     float64
	TimeS     float64
	Real      float64
	Synthetic float64
	SynthFrac float64
	ArchMult  float64
	DataOK    bool
	Optimal   float64
}

func computeReserve(s *State, b *balance.Balance) (train, synth float64) {
	if !s.IsUnlocked(UnlockTraining) {
		return 0, 0
	}
	if len(s.Training.Queue) > 0 {
		train = math.Min(1, math.Max(0, s.Training.Queue[0].ClusterShare))
	}
	if synthAllowed(s, b) {
		synth = math.Min(1-train, math.Max(0, s.Training.SynthShare))
	}
	return train, synth
}

func synthAllowed(s *State, b *balance.Balance) bool {
	i := bestModel(s)
	return i >= 0 && s.Training.Models[i].Benchmark >= b.Training.SyntheticMinBenchmark
}

func MaxModels(s *State, b *balance.Balance) int {
	return b.Training.MaxModels + s.ResMods.MaxModels
}

func ArchMult(s *State, b *balance.Balance) float64 {
	m := 1.0
	if x := s.Derived.Talent.LossMult; x > 0 {
		m *= x
	}
	for _, a := range b.Archs() {
		if a.Card >= 0 && s.Cards[b.Card[a.Card].ID] > 0 {
			m *= a.Mult
		}
	}
	return m
}

func archList(s *State, b *balance.Balance) []string {
	var out []string
	for _, a := range b.Archs() {
		if a.Card >= 0 && s.Cards[b.Card[a.Card].ID] > 0 {
			out = append(out, a.ID)
		}
	}
	return out
}

func finishTrainView(s *State, b *balance.Balance) {
	tv := &s.Derived.Train
	t := &b.Training
	tv.ClusterFLOPS, tv.TrainFLOPS, tv.SynthFLOPS = 0, 0, 0
	if s.IsUnlocked(UnlockInfra) {
		orbital := 1 + float64(s.Infra.Energy.Orbital)*b.Energy.OrbitalComputeBonus
		tv.ClusterFLOPS = training.ClusterFLOPS(s.Infra.Racks[:], b.GPUGens, t.ClusterRackExponent) * orbital
		speed := tv.EnergyFrac * (1 + s.CardMods.TrainingSpeed) * s.Derived.Talent.Mult
		tv.TrainFLOPS = tv.ClusterFLOPS * tv.TrainShare * speed
		tv.SynthFLOPS = tv.ClusterFLOPS * tv.SynthShare * speed
	}
	tv.ParamsLevel = training.ParamsLevel(t, s.Lifetime.Total)
	tv.MaxParams = training.MaxParams(t, tv.ParamsLevel) * s.ResMods.MaxParamsMult
	tv.MaxModels = MaxModels(s, b)
	tv.ArchMult = ArchMult(s, b)
	tv.BestBenchmark = BestBenchmark(s, b)
	tv.SynthOK = synthAllowed(s, b)
	tv.RevealRatio = s.CardMods.RevealRatio
	tv.ETA, tv.QueueETA = 0, 0
	for k := range s.Training.Queue {
		j := &s.Training.Queue[k]
		left := training.FLOPs(j.Params, j.Tokens, t) * math.Max(0, 1-j.Progress)
		rate := plannedRate(s, j.ClusterShare)
		if k == 0 {
			rate = tv.TrainFLOPS
		}
		eta := math.Inf(1)
		if rate > 0 {
			eta = left / rate
		}
		if k == 0 {
			tv.ETA = eta
		}
		tv.QueueETA += eta
	}
	tv.ScraperPaused = s.Training.ScraperPauseS > s.Clock
	tv.DataPerSec = 0
	if s.IsUnlocked(UnlockData) && !tv.ScraperPaused {
		tv.DataPerSec = float64(s.Training.Scrapers) * b.Data.ScraperTokensPerS
	}
	tv.SynthPerSec = 0
	if i := bestModel(s); i >= 0 && tv.SynthFLOPS > 0 {
		tv.SynthPerSec = tv.SynthFLOPS / (t.SyntheticFlopsPerParam * s.Training.Models[i].Params)
	}
	tv.LicenseCost = MinutesOfIncome(s, b.Data.LicenseCostMinutes)
	tv.LicenseTokens = training.LicenseTokens(&b.Data, tv.ParamsLevel)
}

func normalizeTraining(s *State, b *balance.Balance) {
	tr := &s.Training
	tr.Scrapers = max(0, tr.Scrapers)
	tr.Licenses = max(0, tr.Licenses)
	tr.Released = max(tr.Released, len(tr.Models))
	if !(tr.DataTokens >= 0) || math.IsInf(tr.DataTokens, 0) {
		tr.DataTokens = 0
	}
	if !(tr.SynthTokens >= 0) || math.IsInf(tr.SynthTokens, 0) {
		tr.SynthTokens = 0
	}
	tr.SynthShare = math.Min(1, math.Max(0, tr.SynthShare))
	if !(tr.SynthShare >= 0) {
		tr.SynthShare = 0
	}
	for k := range tr.Queue {
		j := &tr.Queue[k]
		if !(j.ArchMult > 0) {
			j.ArchMult = 1
		}
		if !(j.ClusterShare > 0) || j.ClusterShare > 1 {
			j.ClusterShare = 1
		}
		j.Progress = math.Min(1, math.Max(0, j.Progress))
	}
	normalizeArchive(s, b)
}

func EstimateTraining(s *State, b *balance.Balance, n, d, share float64) TrainEstimate {
	t := &b.Training
	e := TrainEstimate{FLOPs: training.FLOPs(n, d, t), ArchMult: ArchMult(s, b)}
	e.Real = math.Min(s.Training.DataTokens, d)
	e.Synthetic = math.Max(0, d-e.Real)
	e.DataOK = e.Synthetic <= s.Training.SynthTokens
	if d > 0 {
		e.SynthFrac = e.Synthetic / d
	}
	e.Benchmark = training.Benchmark(n, d, e.ArchMult*synthMult(s, b, e.SynthFrac), t)
	rate := plannedRate(s, share)
	e.TimeS = math.Inf(1)
	if rate > 0 {
		e.TimeS = e.FLOPs / rate
	}
	if s.CardMods.RevealRatio {
		e.Optimal = training.OptimalTokens(n, t)
	}
	return e
}

func plannedRate(s *State, share float64) float64 {
	tv := &s.Derived.Train
	energy := tv.EnergyFrac
	if !(energy > 0) {
		energy = 1
	}
	return tv.ClusterFLOPS * math.Min(1, math.Max(0, share)) * (1 + s.CardMods.TrainingSpeed) * s.Derived.Talent.Mult * energy
}

func SuggestTraining(s *State, b *balance.Balance, share, targetS float64) (n, d, bench float64) {
	rate := plannedRate(s, share)
	maxD := s.Training.DataTokens + math.Min(s.Training.SynthTokens, s.Training.DataTokens)
	return training.BestSplit(rate*targetS, s.Derived.Train.MaxParams, maxD, ArchMult(s, b), &b.Training)
}

func ScraperCost(s *State, b *balance.Balance, k int) float64 {
	return econ.BulkCost(b.Data.ScraperCost, b.Data.ScraperR, s.Training.Scrapers, k)
}

func MaxScrapers(s *State, b *balance.Balance) int {
	return econ.MaxAffordable(s.Money, b.Data.ScraperCost, b.Data.ScraperR, s.Training.Scrapers)
}

func train(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	t := &b.Training
	if !s.IsUnlocked(UnlockTraining) {
		return reject(out, c, RejectLocked, t.UnlockLifetime)
	}
	n, d, share := c.Params, c.Tokens, c.Share
	if !(n >= t.MinParams) || !(d > 0) || math.IsInf(d, 0) || !(share > 0) || share > 1 {
		return reject(out, c, RejectInvalid, 0)
	}
	if maxN := s.Derived.Train.MaxParams; n > maxN {
		return reject(out, c, RejectTooLarge, maxN)
	}
	if len(s.Training.Queue) >= t.MaxQueue {
		return reject(out, c, RejectQueueFull, 0)
	}
	tr := &s.Training
	fromReal := math.Min(tr.DataTokens, d)
	syn := d - fromReal
	if syn > tr.SynthTokens {
		return reject(out, c, RejectNoData, syn-tr.SynthTokens)
	}
	tr.DataTokens -= fromReal
	tr.SynthTokens -= syn
	tr.Queue = append(tr.Queue, TrainJob{
		Params: n, Tokens: d, ClusterShare: share, Arch: archList(s, b),
		ArchMult: ArchMult(s, b), Synthetic: syn / d, Manual: c.On,
	})
	Recompute(s, b)
	return append(out, Event{Kind: EvTrainQueued, Business: -1, Index: len(tr.Queue) - 1, Value: n, Amount: d})
}

func cancelTrain(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	tr := &s.Training
	i := c.Index
	if i < 0 || i >= len(tr.Queue) {
		return reject(out, c, RejectInvalid, 0)
	}
	j := tr.Queue[i]
	left := j.Tokens * math.Max(0, 1-j.Progress)
	tr.DataTokens += left * (1 - j.Synthetic)
	tr.SynthTokens += left * j.Synthetic
	tr.Queue = append(tr.Queue[:i], tr.Queue[i+1:]...)
	Recompute(s, b)
	return append(out, Event{Kind: EvTrainCancel, Business: -1, Index: i, Value: left})
}

func setTrainShare(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	tr := &s.Training
	i := c.Index
	if i < 0 || i >= len(tr.Queue) || !(c.Share > 0) || c.Share > 1 {
		return reject(out, c, RejectInvalid, 0)
	}
	tr.Queue[i].ClusterShare = c.Share
	Recompute(s, b)
	return append(out, Event{Kind: EvTrainShare, Business: -1, Index: i, Value: c.Share})
}

func setSynthetic(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockData) {
		return reject(out, c, RejectLocked, b.Data.UnlockLifetime)
	}
	if !(c.Share >= 0) || c.Share > 1 {
		return reject(out, c, RejectInvalid, 0)
	}
	if c.Share > 0 && !synthAllowed(s, b) {
		return reject(out, c, RejectNeedBenchmark, b.Training.SyntheticMinBenchmark)
	}
	s.Training.SynthShare = c.Share
	Recompute(s, b)
	return append(out, Event{Kind: EvSynthetic, Business: -1, Value: c.Share})
}

func buyScraper(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockData) {
		return reject(out, c, RejectLocked, b.Data.UnlockLifetime)
	}
	k := c.Qty
	if k == QtyMax {
		k = MaxScrapers(s, b)
	}
	k = max(1, k)
	cost := ScraperCost(s, b, k)
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Training.Scrapers += k
	s.Stats.Purchases++
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvScraper, Business: -1, Count: k, Amount: cost})
}

func buyLicense(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockData) {
		return reject(out, c, RejectLocked, b.Data.UnlockLifetime)
	}
	cost := s.Derived.Train.LicenseCost
	tokens := s.Derived.Train.LicenseTokens
	if !(cost > 0) || !(tokens > 0) {
		return reject(out, c, RejectInvalid, 0)
	}
	if cost > s.Money {
		return reject(out, c, RejectCannotAfford, cost-s.Money)
	}
	s.Money -= cost
	s.Training.DataTokens += tokens
	s.Training.Licenses++
	s.Stats.Purchases++
	s.Undo = Undo{}
	Recompute(s, b)
	return append(out, Event{Kind: EvLicense, Business: -1, Amount: cost, Value: tokens})
}

func tickData(s *State, b *balance.Balance, dt float64, out []Event, md mode) []Event {
	tr := &s.Training
	dd := &b.Data
	if s.IsUnlocked(UnlockData) && tr.Scrapers > 0 && tr.ScraperPauseS <= s.Clock {
		tr.DataTokens += float64(tr.Scrapers) * dd.ScraperTokensPerS * dt
		if rate := dd.ScraperIncidentChancePerH * float64(tr.Scrapers) / 3600 * s.ResMods.IncidentMult; !md.offline && rate > 0 {
			if s.RNG.Misc.Chance(1 - math.Exp(-rate*dt)) {
				fine := math.Min(s.Money, MinutesOfIncome(s, dd.ScraperIncidentFineMin)*s.ResMods.IncidentMult)
				s.Money -= fine
				tr.ScraperPauseS = s.Clock + dd.ScraperIncidentPauseS
				s.Stats.Incidents++
				out = append(out, Event{Kind: EvScraperIncident, Business: -1, Amount: fine, Value: dd.ScraperIncidentPauseS})
			}
		}
	}
	tv := &s.Derived.Train
	if tv.SynthFLOPS > 0 {
		if i := bestModel(s); i >= 0 {
			tr.SynthTokens += tv.SynthFLOPS * dt / (b.Training.SyntheticFlopsPerParam * tr.Models[i].Params)
		}
	}
	return out
}

func tickTraining(s *State, b *balance.Balance, dt float64, out []Event, md mode) []Event {
	tr := &s.Training
	if !s.IsUnlocked(UnlockTraining) || len(tr.Queue) == 0 {
		return out
	}
	rate := s.Derived.Train.TrainFLOPS
	if !(rate > 0) {
		return out
	}
	j := &tr.Queue[0]
	j.Progress += rate * dt / training.FLOPs(j.Params, j.Tokens, &b.Training)
	if j.Progress < 1 {
		return out
	}
	return release(s, b, out, j.Manual && !md.offline)
}

func synthMult(s *State, b *balance.Balance, frac float64) float64 {
	if AGIDone(s, b, balance.AGISynthetic) {
		return 1
	}
	return training.SyntheticMult(frac, &b.Training)
}

func release(s *State, b *balance.Balance, out []Event, ask bool) []Event {
	tr := &s.Training
	j := tr.Queue[0]
	tr.Queue = append(tr.Queue[:0], tr.Queue[1:]...)
	bench := training.Benchmark(j.Params, j.Tokens, j.ArchMult*synthMult(s, b, j.Synthetic), &b.Training)
	return placeModel(s, b, Model{Params: j.Params, Tokens: j.Tokens, Benchmark: bench, ReleasedS: s.Clock, Synthetic: j.Synthetic}, ask, out)
}

func addModel(s *State, b *balance.Balance, md Model, out []Event) []Event {
	return placeModel(s, b, md, false, out)
}

func finishModel(s *State, md *Model) {
	tr := &s.Training
	tr.Released++
	if md.Name == "" {
		md.Name = "TT-" + strconv.Itoa(tr.Released)
	}
	s.Stats.Models++
	s.Stats.BestBench = math.Max(s.Stats.BestBench, md.Benchmark)
}

func enterModel(s *State, b *balance.Balance, md Model, out []Event) []Event {
	tr := &s.Training
	tr.Models = append(tr.Models, md)
	bench := md.Benchmark
	mv := &s.Derived.Market
	rivals := mv.RivalBench[:mv.Rivals]
	share := market.Share(bench, rivals, b.Market.ShareExponent)
	if mv.On {
		s.Market.Hype += b.Market.ReleaseHype * market.Quality(bench, mv.Median, &b.Market)
	}
	return append(out, Event{Kind: EvModelRelease, Business: -1, Index: len(tr.Models) - 1, Value: bench,
		Amount: share, Count: market.Rank(bench, rivals)})
}
