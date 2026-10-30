package sim

import (
	"math"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/market"
)

type Cond struct {
	Metric string  `json:"metric"`
	Op     string  `json:"op"`
	Value  float64 `json:"value"`
}

type Action struct {
	Kind  string  `json:"action"`
	Index int     `json:"index"`
	Qty   int     `json:"qty"`
	Value float64 `json:"value"`
}

type Rule struct {
	If      [MaxConds]Cond `json:"if"`
	Then    Action         `json:"then"`
	Enabled bool           `json:"enabled"`
	Fired   int            `json:"fired"`
	LastS   float64        `json:"last_s"`
}

const MaxConds = 2

const (
	OpGreater = ">"
	OpLess    = "<"
)

type MetricKind uint8

const (
	KindTimes MetricKind = iota
	KindMinutes
	KindRate
	KindPercent
	KindIndex
	KindBool
	KindSeconds
)

type MetricDef struct {
	ID      string
	Kind    MetricKind
	Presets []float64
}

var Metrics = [...]MetricDef{
	{"money_x_best", KindTimes, []float64{1, 2, 5, 10, 100}},
	{"money_minutes", KindMinutes, []float64{1, 5, 15, 60}},
	{"income", KindRate, nil},
	{"gpu_util", KindPercent, []float64{0.5, 0.8, 0.9, 0.95, 1}},
	{"energy_util", KindPercent, []float64{0.5, 0.8, 0.9, 0.95, 1}},
	{"energy_index", KindIndex, []float64{20, 35, 50, 65, 80}},
	{"share", KindPercent, []float64{0.1, 0.25, 0.5, 0.75}},
	{"hype", KindTimes, []float64{1.2, 1.5, 2, 3}},
	{"training_idle", KindBool, nil},
	{"event_in", KindSeconds, []float64{10, 30, 60, 120}},
}

func MetricIndex(id string) int {
	for i := range Metrics {
		if Metrics[i].ID == id {
			return i
		}
	}
	return -1
}

type ParamKind uint8

const (
	ParamNone ParamKind = iota
	ParamQty
	ParamBusinessQty
	ParamFree
)

type ActionDef struct {
	ID    string
	Param ParamKind
}

var Actions = [...]ActionDef{
	{"buy_best", ParamQty},
	{"buy_business", ParamBusinessQty},
	{"buy_rack", ParamQty},
	{"buy_grid", ParamNone},
	{"buy_ppa", ParamNone},
	{"buy_smr", ParamNone},
	{"buy_scraper", ParamQty},
	{"buy_license", ParamNone},
	{"train", ParamNone},
	{"set_free", ParamFree},
	{"prestige1", ParamNone},
}

var RuleQtys = [...]int{Qty1, Qty10, Qty100, QtyMax}

var FreePresets = [...]float64{0, 0.1, 0.2, 0.3}

func ActionIndex(id string) int {
	for i := range Actions {
		if Actions[i].ID == id {
			return i
		}
	}
	return -1
}

const (
	ruleTrainShare   = 0.3
	ruleTrainTargetS = 7200
)

func RuleSlots(s *State, b *balance.Balance) int {
	return min(b.Rules.SlotsMax, b.Rules.SlotsStart+s.ResMods.RuleSlots)
}

func ValidRule(r *Rule, b *balance.Balance) bool {
	if MetricIndex(r.If[0].Metric) < 0 {
		return false
	}
	for k := range r.If {
		c := &r.If[k]
		if k > 0 && c.Metric == "" {
			continue
		}
		if MetricIndex(c.Metric) < 0 || (c.Op != OpGreater && c.Op != OpLess) || math.IsNaN(c.Value) || math.IsInf(c.Value, 0) {
			return false
		}
	}
	a := ActionIndex(r.Then.Kind)
	if a < 0 {
		return false
	}
	switch Actions[a].Param {
	case ParamQty:
		return validQty(r.Then.Qty)
	case ParamBusinessQty:
		return validBusiness(r.Then.Index) && validQty(r.Then.Qty)
	case ParamFree:
		return r.Then.Value >= 0 && r.Then.Value <= 1
	}
	return true
}

func validQty(q int) bool { return q == QtyMax || q >= 1 }

func BestROI(s *State, b *balance.Balance) (int, float64) {
	h := HypoOf(s)
	var d Derived
	Evaluate(b, &h, &d)
	r0 := managedRate(s, &d)
	best, bestScore, bestCost := -1, math.Inf(1), 0.0
	for i := range N {
		if !h.Unlocked.Has(i) {
			continue
		}
		cost := BusinessCost(s, b, i, 1)
		h.Owned[i]++
		Evaluate(b, &h, &d)
		h.Owned[i]--
		gain := managedRate(s, &d) - r0
		if s.Businesses[i].Owned == 0 || !s.Businesses[i].Manager {
			gain = d.PerSec[i] * 0.5
		}
		if !(gain > 0) {
			continue
		}
		if sc := cost / gain; sc < bestScore {
			best, bestScore, bestCost = i, sc, cost
		}
	}
	return best, bestCost
}

func managedRate(s *State, d *Derived) float64 {
	r := 0.0
	for i := range N {
		if s.Businesses[i].Manager {
			r += d.PerSec[i]
		}
	}
	return r
}

func MetricValue(s *State, b *balance.Balance, id string) float64 {
	d := &s.Derived
	switch id {
	case "money_x_best":
		if _, cost := BestROI(s, b); cost > 0 {
			return s.Money / cost
		}
		return 0
	case "money_minutes":
		if m := MinutesOfIncome(s, 1); m > 0 {
			return s.Money / m
		}
		return 0
	case "income":
		return d.BaseIncome
	case "gpu_util":
		if d.ComputeCap > 0 {
			return d.ComputeNeed / d.ComputeCap
		}
		return 0
	case "energy_util":
		if d.EnergyAvail > 0 {
			return d.EnergyNeed / d.EnergyAvail
		}
		return 0
	case "energy_index":
		return d.EnergyIndex
	case "share":
		return d.Market.Share
	case "hype":
		return 1 + d.Market.Hype
	case "training_idle":
		if s.IsUnlocked(UnlockTraining) && len(s.Training.Queue) == 0 {
			return 1
		}
		return 0
	case "event_in":
		if !s.IsUnlocked(UnlockEvents) || s.RandomEvent.ActiveID != "" {
			return 0
		}
		return math.Max(0, s.RandomEvent.NextAtS-s.Clock)
	}
	return math.NaN()
}

func condTrue(s *State, b *balance.Balance, c *Cond) bool {
	v := MetricValue(s, b, c.Metric)
	if math.IsNaN(v) {
		return false
	}
	if c.Op == OpLess {
		return v < c.Value
	}
	return v > c.Value
}

func ruleMatches(s *State, b *balance.Balance, r *Rule) bool {
	for k := range r.If {
		if k > 0 && r.If[k].Metric == "" {
			continue
		}
		if !condTrue(s, b, &r.If[k]) {
			return false
		}
	}
	return true
}

func runAction(s *State, b *balance.Balance, a *Action, out []Event) []Event {
	var c Command
	switch a.Kind {
	case "buy_best":
		i, _ := BestROI(s, b)
		if i < 0 {
			return reject(out, &Command{Kind: CmdBuyBusiness, Business: -1}, RejectInvalid, 0)
		}
		c = Buy(i, a.Qty)
	case "buy_business":
		c = Buy(a.Index, a.Qty)
	case "buy_rack":
		gen := s.Infra.GPUGen
		q := a.Qty
		if q == QtyMax {
			q = max(1, MaxRacks(s, b, gen))
		}
		if validGen(b, gen) {
			mw := GridNeededMW(s, b, float64(q)*b.GPUGens[gen-1].MW)
			if RackCost(s, b, gen, q)+GridCost(b, mw) > s.Money && q > 1 {
				q--
				mw = GridNeededMW(s, b, float64(q)*b.GPUGens[gen-1].MW)
			}
			c = BuyRack(gen, q)
			n := len(out)
			out = apply(s, b, &c, out)
			if rejected(out[n:]) || !(mw > 0) {
				return out
			}
			c = BuyGrid(mw)
			n = len(out)
			out = apply(s, b, &c, out)
			if rejected(out[n:]) {
				out = out[:n]
			}
			return out
		} else {
			c = BuyRack(gen, q)
		}
	case "buy_grid":
		c = BuyGrid(GridNeededMW(s, b, 0))
	case "buy_ppa":
		c = BuyPPA(math.Max(1, math.Ceil(s.Derived.GridUsed)))
	case "buy_smr":
		c = BuySMR()
	case "buy_scraper":
		c = BuyScraper(a.Qty)
	case "buy_license":
		c = BuyLicense()
	case "train":
		n, d, bench := SuggestTraining(s, b, ruleTrainShare, ruleTrainTargetS)
		if !(bench > 0) || (bestModel(s) >= 0 && bench <= BestBenchmark(s, b)) {
			return reject(out, &Command{Kind: CmdTrain, Business: -1}, RejectInvalid, 0)
		}
		c = Train(n, d, ruleTrainShare)
	case "set_free":
		ent := s.Derived.Market.Alloc[market.SegEnterprise]
		f := math.Min(a.Value, 1-ent)
		cur := s.Market.Segments
		if math.Abs(cur.Free-f) < 1e-9 && math.Abs(cur.Enterprise-ent) < 1e-9 {
			return reject(out, &Command{Kind: CmdSetSegments, Business: -1}, RejectAlreadyOwned, 0)
		}
		c = SetSegments(f, 1-f-ent, ent)
	case "prestige1":
		c = PrestigeReset()
	default:
		return reject(out, &Command{Business: -1}, RejectInvalid, 0)
	}
	return apply(s, b, &c, out)
}

func rejected(evs []Event) bool {
	for i := range evs {
		if evs[i].Kind == EvRejected {
			return true
		}
	}
	return false
}

func tickRules(s *State, b *balance.Balance, from float64, out []Event, md mode) []Event {
	every := b.Rules.EvalEveryS
	if !(every > 0) || math.Floor(s.Clock/every) == math.Floor(from/every) {
		return out
	}
	if md.offline && !s.ResMods.OfflineRules {
		return autoPrestige(s, b, out)
	}
	if s.IsUnlocked(UnlockRules) {
		slots := RuleSlots(s, b)
		for k := range s.Rules {
			if k >= slots {
				break
			}
			r := &s.Rules[k]
			if !r.Enabled || !ruleMatches(s, b, r) {
				continue
			}
			n := len(out)
			out = runAction(s, b, &r.Then, out)
			if rejected(out[n:]) {
				out = out[:n]
				continue
			}
			r = &s.Rules[k]
			r.Fired++
			r.LastS = s.Clock
			s.Stats.RulesFired++
			out = append(out, Event{Kind: EvRuleFired, Business: -1, Index: k, Count: ActionIndex(r.Then.Kind)})
		}
	}
	return autoPrestige(s, b, out)
}

func autoPrestige(s *State, b *balance.Balance, out []Event) []Event {
	if !s.ResMods.AutoPrestige {
		return out
	}
	pv := Preview(s, b)
	if !pv.Allowed || pv.BonusAfter < pv.BonusNow*b.Research.AutoPrestigeRatio {
		return out
	}
	c := PrestigeReset()
	n := len(out)
	out = apply(s, b, &c, out)
	if !rejected(out[n:]) {
		out = append(out, Event{Kind: EvAutoPrestige, Business: -1, Amount: pv.Gain})
	}
	return out
}

func addRule(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	if !s.IsUnlocked(UnlockRules) {
		return reject(out, c, RejectLocked, b.Rules.UnlockLifetime)
	}
	r := c.Rule
	r.Fired, r.LastS = 0, 0
	if r.Then.Kind != "set_free" {
		r.Then.Value = 0
	}
	if !ValidRule(&r, b) {
		return reject(out, c, RejectInvalid, 0)
	}
	if len(s.Rules) >= RuleSlots(s, b) {
		return reject(out, c, RejectTeamFull, 0)
	}
	s.Rules = append(s.Rules, r)
	return append(out, Event{Kind: EvRuleAdded, Business: -1, Index: len(s.Rules) - 1})
}

func editRule(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := c.Index
	if i < 0 || i >= len(s.Rules) {
		return reject(out, c, RejectInvalid, 0)
	}
	switch c.Kind {
	case CmdDeleteRule:
		s.Rules = append(s.Rules[:i], s.Rules[i+1:]...)
		return append(out, Event{Kind: EvRuleDeleted, Business: -1, Index: i})
	case CmdToggleRule:
		s.Rules[i].Enabled = !s.Rules[i].Enabled
		n := 0
		if s.Rules[i].Enabled {
			n = 1
		}
		return append(out, Event{Kind: EvRuleToggled, Business: -1, Index: i, Count: n})
	case CmdMoveRule:
		j := i + c.Qty
		if j < 0 || j >= len(s.Rules) || c.Qty == 0 {
			return reject(out, c, RejectInvalid, 0)
		}
		s.Rules[i], s.Rules[j] = s.Rules[j], s.Rules[i]
		return append(out, Event{Kind: EvRuleMoved, Business: -1, Index: j, Count: i})
	}
	return reject(out, c, RejectInvalid, 0)
}

func normalizeRules(s *State, b *balance.Balance) {
	keep := s.Rules[:0]
	for _, r := range s.Rules {
		if ValidRule(&r, b) {
			keep = append(keep, r)
		}
	}
	s.Rules = keep
	if s.Rules == nil {
		s.Rules = make([]Rule, 0, b.Rules.SlotsMax)
	}
}
