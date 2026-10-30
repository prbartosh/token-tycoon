package sim

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/market"
)

const NumSegments = market.NumSegments

type MarketView struct {
	On           bool
	Price        float64
	Rivals       int
	RivalBench   [balance.MaxRivals]float64
	RivalLastS   [balance.MaxRivals]float64
	Median       float64
	Benchmark    float64
	Share        float64
	Rank         int
	Hype         float64
	HypeGain     float64
	Pressure     float64
	EnterpriseOK bool
	RegionMult   float64
	EntShare     float64
	Alloc        [NumSegments]float64
	Model        [NumSegments]int
	Quote        [NumSegments]market.Quote
	SegPrice     [NumSegments]float64
}

type rivalCache struct {
	ok      bool
	seed    uint64
	fromS   float64
	untilS  float64
	bench   [balance.MaxRivals]float64
	last    [balance.MaxRivals]float64
	median  float64
	weight  float64
	since   [NumSegments]float64
	press   [NumSegments]float64
	pressOK [NumSegments]bool
}

func rivalsAt(s *State, b *balance.Balance) *rivalCache {
	c := &s.rivals
	if c.ok && c.seed == s.Seed && s.Clock >= c.fromS && s.Clock < c.untilS {
		return c
	}
	m := &b.Market
	*c = rivalCache{ok: true, seed: s.Seed, fromS: math.Inf(-1), untilS: math.Inf(1)}
	n := min(len(b.Rivals), balance.MaxRivals)
	for i := range n {
		r := &b.Rivals[i]
		rel := market.LastRelease(s.Seed, i, r, s.Clock, m.RivalSpanDays)
		c.bench[i], c.last[i] = rel.Benchmark, rel.AtS
		c.fromS = math.Max(c.fromS, rel.AtS)
		c.untilS = math.Min(c.untilS, market.NextRelease(s.Seed, i, r, s.Clock))
	}
	c.median = market.Median(c.bench[:n])
	c.weight = market.RivalWeight(c.bench[:n], m.ShareExponent)
	return c
}

func (c *rivalCache) pressure(s *State, b *balance.Balance, seg int, since float64) float64 {
	if !c.pressOK[seg] || c.since[seg] != since {
		c.since[seg], c.pressOK[seg] = since, true
		c.press[seg] = market.Pressure(s.Seed, b.Rivals, since, s.Clock, b.Market.PressureFloor)
	}
	return c.press[seg]
}

func bestModel(s *State) int {
	best := -1
	for i := range s.Training.Models {
		if best < 0 || s.Training.Models[i].Benchmark > s.Training.Models[best].Benchmark {
			best = i
		}
	}
	return best
}

func latestModel(s *State) int {
	last := -1
	for i := range s.Training.Models {
		if last < 0 || s.Training.Models[i].ReleasedS >= s.Training.Models[last].ReleasedS {
			last = i
		}
	}
	return last
}

func BestBenchmark(s *State, b *balance.Balance) float64 {
	if i := bestModel(s); i >= 0 {
		return s.Training.Models[i].Benchmark
	}
	return baseBenchmark(s, b)
}

func baseBenchmark(s *State, b *balance.Balance) float64 {
	return math.Max(b.Market.BaseBenchmark, s.Derived.Market.Median)
}

func segmentModel(s *State, seg int) int {
	name := market.SegmentNames[seg]
	for i := range s.Training.Models {
		if s.Training.Models[i].Segment == name {
			return i
		}
	}
	return latestModel(s)
}

func evaluateMarket(s *State, b *balance.Balance, ev int) {
	mv := &s.Derived.Market
	m := &b.Market
	*mv = MarketView{Price: m.PriceBase, Pressure: 1, Rank: 1, RegionMult: 1, EntShare: 1}
	for seg := range mv.Model {
		mv.Model[seg] = -1
	}
	if !s.IsUnlocked(UnlockMarket) {
		return
	}
	mv.On = true
	mv.Rivals = min(len(b.Rivals), balance.MaxRivals)
	rc := rivalsAt(s, b)
	mv.RivalBench, mv.RivalLastS, mv.Median = rc.bench, rc.last, rc.median
	rivals := mv.RivalBench[:mv.Rivals]
	mv.Hype = math.Max(0, s.Market.Hype)
	if ev >= 0 && b.Event[ev].Effect == balance.EventHype {
		mv.Hype += b.Event[ev].Value
	}
	mv.Benchmark = BestBenchmark(s, b)
	mv.Share = market.ShareOf(mv.Benchmark, rc.weight, m.ShareExponent)
	mv.Rank = market.Rank(mv.Benchmark, rivals)
	for seg := range NumSegments {
		mi := segmentModel(s, seg)
		mv.Model[seg] = mi
		bench, age, since := baseBenchmark(s, b), 0.0, s.Market.OpenedS
		if mi >= 0 {
			md := &s.Training.Models[mi]
			bench, age, since = md.Benchmark, s.Clock-md.ReleasedS, md.ReleasedS
		}
		p := rc.pressure(s, b, seg, since)
		if seg == market.SegPro {
			mv.Pressure = p
		}
		mv.Quote[seg] = market.Price(bench, rc.weight, mv.Median, mv.Hype, age, p, m)
		mv.SegPrice[seg] = mv.Quote[seg].Price * market.SegmentMult(seg, m)
	}
	ent := mv.Model[market.SegEnterprise]
	mv.EnterpriseOK = ent >= 0 && s.Training.Models[ent].Benchmark >= m.Segments.EnterpriseMinBenchmark
	mv.Alloc = effectiveAlloc(s.Market.Segments, mv.EnterpriseOK, EnterpriseCap(s, b))
	mv.RegionMult, mv.EntShare = regionFactors(s, b)
	mv.Price = marketPrice(mv, mv.RegionMult, mv.EntShare)
	mv.HypeGain = market.HypeGainPerS(mv.Alloc[market.SegFree], mv.Quote[market.SegFree].Share, m) + s.Derived.Talent.HypePerS
}

func marketPrice(mv *MarketView, mult, ent float64) float64 {
	p := 0.0
	for seg := range NumSegments {
		sp := mv.SegPrice[seg]
		if seg == market.SegEnterprise {
			sp = ent*sp + (1-ent)*mv.SegPrice[market.SegPro]
		}
		p += mv.Alloc[seg] * sp
	}
	return p * mult
}

func EnterpriseCap(s *State, b *balance.Balance) float64 {
	return math.Min(1, b.Market.Segments.EnterpriseMaxAlloc+s.ResMods.EnterpriseCap)
}

func effectiveAlloc(sg Segments, entOK bool, limit float64) [NumSegments]float64 {
	f, p, e := math.Max(0, sg.Free), math.Max(0, sg.Pro), math.Max(0, sg.Enterprise)
	sum := f + p + e
	if !(sum > 0) || math.IsInf(sum, 0) {
		return [NumSegments]float64{0, 1, 0}
	}
	f, p, e = f/sum, p/sum, e/sum
	if !entOK {
		p, e = p+e, 0
	}
	if e > limit {
		p, e = p+e-limit, limit
	}
	return [NumSegments]float64{f, p, e}
}

func normalizeMarket(s *State, b *balance.Balance) {
	m := &s.Market
	if m.UnlockedRegions == nil {
		m.UnlockedRegions = []string{}
	}
	if !(m.Hype >= 0) || math.IsInf(m.Hype, 0) {
		m.Hype = 0
	}
	m.OpenedS = math.Min(math.Max(0, m.OpenedS), s.Clock)
	sg := &m.Segments
	if !(sg.Free >= 0) || !(sg.Pro >= 0) || !(sg.Enterprise >= 0) || !(sg.Free+sg.Pro+sg.Enterprise > 0) {
		*sg = Segments{Pro: 1}
	}
}

func tickMarket(s *State, b *balance.Balance, from float64, dt float64, out []Event) []Event {
	if !s.IsUnlocked(UnlockMarket) {
		return out
	}
	mk := &b.Market
	s.Market.Hype = market.HypeAfter(s.Market.Hype, s.Derived.Market.HypeGain, mk.HypeHalfLifeH*3600, dt)
	if from+dt < rivalsAt(s, b).untilS {
		return out
	}
	for i := range min(len(b.Rivals), balance.MaxRivals) {
		if rel, ok := market.ReleasedIn(s.Seed, i, &b.Rivals[i], from, from+dt, mk.RivalSpanDays); ok {
			out = append(out, Event{Kind: EvRivalRelease, Business: -1, Index: i, Value: rel.Benchmark})
		}
	}
	return out
}

func setSegments(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockMarket) {
		return reject(out, c, RejectLocked, b.Market.UnlockLifetime)
	}
	sg := c.Segments
	if !(sg.Free >= 0) || !(sg.Pro >= 0) || !(sg.Enterprise >= 0) {
		return reject(out, c, RejectInvalid, 0)
	}
	sum := sg.Free + sg.Pro + sg.Enterprise
	if !(sum > 0) || math.IsInf(sum, 0) {
		return reject(out, c, RejectInvalid, 0)
	}
	sg = Segments{Free: sg.Free / sum, Pro: sg.Pro / sum, Enterprise: sg.Enterprise / sum}
	if sg.Enterprise > 0 && !s.Derived.Market.EnterpriseOK {
		return reject(out, c, RejectNeedBenchmark, b.Market.Segments.EnterpriseMinBenchmark)
	}
	if limit := EnterpriseCap(s, b); sg.Enterprise > limit {
		sg.Pro += sg.Enterprise - limit
		sg.Enterprise = limit
	}
	s.Market.Segments = sg
	Recompute(s, b)
	return append(out, Event{Kind: EvSegments, Business: -1, Value: sg.Free, Amount: sg.Enterprise})
}

func assignModel(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockMarket) {
		return reject(out, c, RejectLocked, b.Market.UnlockLifetime)
	}
	i := c.Index
	if i < 0 || i >= len(s.Training.Models) {
		return reject(out, c, RejectInvalid, 0)
	}
	seg := -1
	if c.Text != "" {
		if seg = market.SegmentIndex(c.Text); seg < 0 {
			return reject(out, c, RejectInvalid, 0)
		}
	}
	if seg == market.SegEnterprise && s.Training.Models[i].Benchmark < b.Market.Segments.EnterpriseMinBenchmark {
		return reject(out, c, RejectNeedBenchmark, b.Market.Segments.EnterpriseMinBenchmark)
	}
	for k := range s.Training.Models {
		if seg >= 0 && s.Training.Models[k].Segment == c.Text {
			s.Training.Models[k].Segment = ""
		}
	}
	s.Training.Models[i].Segment = c.Text
	Recompute(s, b)
	return append(out, Event{Kind: EvAssign, Business: -1, Index: i, Count: seg})
}

func retireModel(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := c.Index
	if i < 0 || i >= len(s.Training.Models) {
		return reject(out, c, RejectInvalid, 0)
	}
	s.Training.Models = append(s.Training.Models[:i], s.Training.Models[i+1:]...)
	Recompute(s, b)
	return append(out, Event{Kind: EvModelRetired, Business: -1, Index: i})
}

func SegmentAlloc(s *State) [NumSegments]float64 { return s.Derived.Market.Alloc }

func RivalRelease(s *State, b *balance.Balance, i int) market.Release {
	if i < 0 || i >= len(b.Rivals) {
		return market.Release{K: -1}
	}
	return market.LastRelease(s.Seed, i, &b.Rivals[i], s.Clock, b.Market.RivalSpanDays)
}

func SteadyHype(s *State, b *balance.Balance, freeAlloc float64) float64 {
	m := &b.Market
	gain := market.HypeGainPerS(freeAlloc, s.Derived.Market.Quote[market.SegFree].Share, m)
	return market.SteadyHype(gain, m.HypeHalfLifeH*3600)
}
