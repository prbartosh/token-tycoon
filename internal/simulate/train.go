package simulate

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/market"
	"github.com/prbartosh/token-tycoon/internal/sim"
)

const (
	marketEveryS  = 60
	trainEveryS   = 30
	trainShare    = 0.3
	scraperBudget = 0.01
	licenseBudget = 0.05
	rackBudget    = 0.01
)

var trainTargets = [...]float64{1800, 7200, 28800}

var freeSteps = [...]float64{0, 0.1, 0.2, 0.3}

func (bt *bot) maintainMarket() {
	s := &bt.s
	if !s.IsUnlocked(sim.UnlockMarket) || s.Clock < bt.nextMkt {
		return
	}
	bt.nextMkt = s.Clock + marketEveryS
	mv := &s.Derived.Market
	m := &bt.b.Market
	ent := 0.0
	if mv.EnterpriseOK {
		ent = m.Segments.EnterpriseMaxAlloc
	}
	bestF, bestV := 0.0, -1.0
	for _, f := range freeSteps {
		h := sim.SteadyHype(s, bt.b, f)
		v := 0.0
		alloc := [sim.NumSegments]float64{f, 1 - f - ent, ent}
		for seg := range sim.NumSegments {
			q := mv.Quote[seg]
			if !(q.Demand > 0) {
				continue
			}
			v += alloc[seg] * mv.SegPrice[seg] / q.Demand * market.Demand(q.Share, h, m)
		}
		if v > bestV*1.01 {
			bestF, bestV = f, v
		}
	}
	cur := s.Market.Segments
	if math.Abs(cur.Free-bestF) > 1e-9 || math.Abs(cur.Enterprise-ent) > 1e-9 {
		bt.exec(sim.SetSegments(bestF, 1-bestF-ent, ent))
	}
	if !mv.EnterpriseOK {
		if i := bt.bestModel(); i >= 0 && s.Training.Models[i].Benchmark >= m.Segments.EnterpriseMinBenchmark {
			bt.exec(sim.AssignModel(i, market.SegmentNames[market.SegEnterprise]))
		}
	}
}

func (bt *bot) bestModel() int {
	best := -1
	for i, md := range bt.s.Training.Models {
		if best < 0 || md.Benchmark > bt.s.Training.Models[best].Benchmark {
			best = i
		}
	}
	return best
}

func (bt *bot) maintainTraining() {
	s := &bt.s
	b := bt.b
	if !s.IsUnlocked(sim.UnlockData) || s.Clock < bt.nextTrain {
		return
	}
	bt.nextTrain = s.Clock + trainEveryS
	for range 20 {
		if c := sim.ScraperCost(s, b, 1); !(c <= s.Money*scraperBudget) {
			break
		}
		bt.exec(sim.BuyScraper(1))
	}
	tv := &s.Derived.Train
	if c := tv.LicenseCost; c > 0 && c <= s.Money*licenseBudget {
		bt.exec(sim.BuyLicense())
	}
	if tv.SynthOK && s.Training.SynthShare == 0 {
		bt.exec(sim.SetSynthetic(0.05))
	}
	if !s.IsUnlocked(sim.UnlockTraining) {
		return
	}
	bt.trainingRacks()
	if len(s.Training.Queue) > 0 {
		return
	}
	cur, age := 0.0, math.Inf(1)
	if i := bt.bestModel(); i >= 0 {
		cur = s.Training.Models[i].Benchmark
	}
	if i := bt.latestModel(); i >= 0 {
		age = s.Clock - s.Training.Models[i].ReleasedS
	}
	for _, target := range trainTargets {
		n, d, bench := sim.SuggestTraining(s, b, trainShare, target)
		if bench > 0 && (bench > cur+1 || (age > market.Day && bench >= cur-1)) {
			bt.exec(sim.Train(n, d, trainShare))
			return
		}
	}
}

func (bt *bot) trainingRacks() {
	s := &bt.s
	b := bt.b
	gen := s.Infra.GPUGen
	if gen < 1 {
		return
	}
	budget := s.Money * rackBudget
	g := &b.GPUGens[gen-1]
	k := 0
	for k < 256 && sim.RackCost(s, b, gen, k+1)+sim.GridCost(b, sim.GridNeededMW(s, b, float64(k+1)*g.MW)) <= budget {
		k++
	}
	if k == 0 {
		return
	}
	mw := sim.GridNeededMW(s, b, float64(k)*g.MW)
	bt.exec(sim.BuyRack(gen, k))
	if mw > 0 {
		bt.exec(sim.BuyGrid(mw))
	}
	bt.res.Racks += k
}

func (bt *bot) latestModel() int {
	last := -1
	for i, md := range bt.s.Training.Models {
		if last < 0 || md.ReleasedS >= bt.s.Training.Models[last].ReleasedS {
			last = i
		}
	}
	return last
}
