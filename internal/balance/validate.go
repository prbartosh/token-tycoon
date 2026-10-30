package balance

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

type validator struct{ errs []error }

func (v *validator) add(path, format string, args ...any) {
	v.errs = append(v.errs, fmt.Errorf("%s: %s", path, fmt.Sprintf(format, args...)))
}

func (v *validator) positive(path string, x float64) {
	if !(x > 0) || math.IsInf(x, 0) {
		v.add(path, "must be > 0, got %v", x)
	}
}

func (v *validator) nonNeg(path string, x float64) {
	if !(x >= 0) || math.IsInf(x, 0) {
		v.add(path, "must be >= 0, got %v", x)
	}
}

func (v *validator) unit(path string, x float64) {
	if !(x >= 0 && x <= 1) {
		v.add(path, "must be in [0, 1], got %v", x)
	}
}

func (v *validator) rangePair(path string, p []float64) {
	if len(p) != 2 {
		v.add(path, "must have exactly 2 elements [min, max], got %d", len(p))
		return
	}
	if p[0] > p[1] {
		v.add(path, "min %v greater than max %v", p[0], p[1])
	}
}

func (v *validator) uniqueIDs(section string, ids []string) map[string]int {
	seen := make(map[string]int, len(ids))
	for i, id := range ids {
		if id == "" {
			v.add(fmt.Sprintf("%s[%d].id", section, i), "must not be empty")
			continue
		}
		if j, ok := seen[id]; ok {
			v.add(fmt.Sprintf("%s[%d].id", section, i), "duplicate id %q (also at index %d)", id, j)
			continue
		}
		seen[id] = i
	}
	return seen
}

func strictlyAscending(xs []int) bool {
	for i := 1; i < len(xs); i++ {
		if xs[i] <= xs[i-1] {
			return false
		}
	}
	return true
}

// Validate checks the balance for values the simulation cannot work with and
// returns all problems joined, each prefixed with the key path.
func (b *Balance) Validate() error {
	v := &validator{}

	if b.Meta.Version <= 0 {
		v.add("meta.version", "must be > 0")
	}
	e := b.Engine
	if e.SimStepMs <= 0 {
		v.add("engine.sim_step_ms", "must be > 0")
	}
	if e.FPSAnimation <= 0 || e.FPSNormal <= 0 || e.FPSBackground <= 0 {
		v.add("engine.fps_*", "all frame rates must be > 0")
	}
	v.positive("engine.offline_step_s", e.OfflineStepS)
	v.positive("engine.autosave_s", e.AutosaveS)
	if e.Backups < 0 {
		v.add("engine.backups", "must be >= 0")
	}
	v.nonNeg("engine.undo_window_s", e.UndoWindowS)

	o := b.Offline
	v.unit("offline.efficiency_start", o.EfficiencyStart)
	v.unit("offline.efficiency_max", o.EfficiencyMax)
	if o.EfficiencyStart > o.EfficiencyMax {
		v.add("offline.efficiency_start", "greater than efficiency_max")
	}
	v.positive("offline.cap_hours_start", o.CapHoursStart)
	if o.CapHoursStart > o.CapHoursMax {
		v.add("offline.cap_hours_start", "greater than cap_hours_max")
	}

	// Businesses.
	if len(b.Businesses) != MaxBusinesses {
		v.add("business", "need exactly %d [[business]] entries, got %d", MaxBusinesses, len(b.Businesses))
	}
	ids := make([]string, len(b.Businesses))
	for i, bs := range b.Businesses {
		p := fmt.Sprintf("business[%d](%s)", i, bs.ID)
		ids[i] = bs.ID
		v.positive(p+".cost", bs.Cost)
		if !(bs.R > 1) {
			v.add(p+".r", "must be > 1, got %v", bs.R)
		}
		v.positive(p+".cycle_s", bs.CycleS)
		v.positive(p+".tokens", bs.Tokens)
		v.nonNeg(p+".compute", bs.Compute)
		v.nonNeg(p+".manager_cost", bs.ManagerCost)
		v.nonNeg(p+".unlock_lifetime", bs.UnlockLifetime)
	}
	bizIDs := v.uniqueIDs("business", ids)
	bd := b.BusinessDefaults
	v.positive("business_defaults.manager_mult", bd.ManagerMult)
	v.positive("business_defaults.continuous_bar_below_s", bd.ContinuousBarBelowS)
	v.positive("business_defaults.draw_cap_per_s", bd.DrawCapPerS)

	// Start.
	v.nonNeg("start.money", b.Start.Money)
	if len(b.Start.FreeBusinesses) == 0 && b.Start.Money <= 0 {
		v.add("start", "a new game needs either money > 0 or at least one free business")
	}
	for id, n := range b.Start.FreeBusinesses {
		if _, ok := bizIDs[id]; !ok {
			v.add("start.free_businesses."+id, "unknown business id")
		}
		if n < 0 {
			v.add("start.free_businesses."+id, "must be >= 0")
		}
	}

	// Milestones.
	m := b.Milestones
	if !strictlyAscending(m.SpeedThresholds) || (len(m.SpeedThresholds) > 0 && m.SpeedThresholds[0] <= 0) {
		v.add("milestones.speed_thresholds", "must be positive and strictly ascending")
	}
	if !(m.SpeedMult > 0 && m.SpeedMult <= 1) {
		v.add("milestones.speed_mult", "must be in (0, 1], got %v", m.SpeedMult)
	}
	if m.ProfitStart <= 0 || m.ProfitEvery <= 0 {
		v.add("milestones.profit_start/profit_every", "must be > 0")
	}
	if !(m.ProfitMult >= 1) {
		v.add("milestones.profit_mult", "must be >= 1")
	}
	if len(m.GlobalThresholds) == 0 || !strictlyAscending(m.GlobalThresholds) || m.GlobalThresholds[0] <= 0 {
		v.add("milestones.global_thresholds", "must be non-empty, positive and strictly ascending")
	}
	if m.GlobalEvery <= 0 {
		v.add("milestones.global_every", "must be > 0 (milestones continue forever)")
	}
	if !(m.GlobalMult >= 1) {
		v.add("milestones.global_mult", "must be >= 1")
	}

	// Upgrades.
	uids := make([]string, len(b.Upgrades))
	for i, u := range b.Upgrades {
		p := fmt.Sprintf("upgrade[%d](%s)", i, u.ID)
		uids[i] = u.ID
		v.positive(p+".cost", u.Cost)
		v.positive(p+".value", u.Value)
		if len(u.Target) == 0 {
			v.add(p+".target", "must not be empty")
		}
		for _, t := range u.Target {
			if t == "all" {
				if len(u.Target) != 1 {
					v.add(p+".target", `"all" must be the only target`)
				}
				continue
			}
			if _, ok := bizIDs[t]; !ok {
				v.add(p+".target", "unknown business id %q", t)
			}
		}
		switch u.Effect {
		case EffectProfit:
			if u.Value < 1 {
				v.add(p+".value", "profit multiplier must be >= 1")
			}
		case EffectCost, EffectCycle, EffectCompute:
			if u.Value > 1 {
				v.add(p+".value", "%s multiplier must be <= 1", u.Effect)
			}
		default:
			v.add(p+".effect", "unknown effect %q (profit|cost|cycle|compute)", u.Effect)
		}
	}
	v.uniqueIDs("upgrade", uids)
	if len(b.Upgrades) > 256 {
		v.add("upgrade", "at most 256 upgrades supported")
	}

	// Infra and GPU generations.
	v.nonNeg("infra.unlock_lifetime", b.Infra.UnlockLifetime)
	if !(b.Infra.StarterHeadroom >= 1) {
		v.add("infra.starter_headroom", "must be >= 1")
	}
	if !(b.Infra.RackR > 1) {
		v.add("infra.rack_r", "must be > 1")
	}
	v.unit("infra.rack_sellback", b.Infra.RackSellback)
	v.unit("infra.cloud_revenue_share", b.Infra.CloudRevenueShare)
	if len(b.GPUGens) == 0 {
		v.add("gpu_gen", "need at least one generation")
	}
	for i, g := range b.GPUGens {
		p := fmt.Sprintf("gpu_gen[%d]", i)
		if g.Gen != i+1 {
			v.add(p+".gen", "generations must be numbered 1..n in order, got %d", g.Gen)
		}
		v.positive(p+".rack_cost", g.RackCost)
		v.positive(p+".compute", g.Compute)
		v.positive(p+".mw", g.MW)
		v.positive(p+".flops_per_rack", g.FlopsPerRack)
	}
	if len(b.GPUGens) > MaxGPUGens {
		v.add("gpu_gen", "at most %d generations supported", MaxGPUGens)
	}

	en := b.Energy
	v.nonNeg("energy.grid_cost_coef", en.GridCostCoef)
	v.unit("energy.grid_peak_availability", en.GridPeakAvailability)
	if len(en.GridPeakHours) != 2 || en.GridPeakHours[0] < 0 || en.GridPeakHours[1] > 24 || en.GridPeakHours[0] >= en.GridPeakHours[1] {
		v.add("energy.grid_peak_hours", "must be [start, end] within 0..24")
	}
	v.positive("energy.ppa_days", en.PPADays)
	v.nonNeg("energy.orbital_unlock_lifetime", en.OrbitalUnlockLifetime)

	// Market.
	mk := b.Market
	v.nonNeg("market.unlock_lifetime", mk.UnlockLifetime)
	v.nonNeg("market.regions_unlock_lifetime", mk.RegionsUnlockLifetime)
	v.positive("market.price_base", mk.PriceBase)
	v.positive("market.share_exponent", mk.ShareExponent)
	v.unit("market.share_neutral", mk.ShareNeutral)
	v.rangePair("market.demand_clamp", mk.DemandClamp)
	v.rangePair("market.quality_clamp", mk.QualityClamp)
	v.positive("market.quality_divisor", mk.QualityDivisor)
	if !(mk.DeflationPerDay > 0 && mk.DeflationPerDay <= 1) {
		v.add("market.deflation_per_day", "must be in (0, 1]")
	}
	v.positive("market.hype_half_life_h", mk.HypeHalfLifeH)
	rids := make([]string, len(b.Rivals))
	for i, r := range b.Rivals {
		rids[i] = r.ID
		p := fmt.Sprintf("rival[%d](%s)", i, r.ID)
		v.positive(p+".release_every_days", r.ReleaseEveryDays)
		v.nonNeg(p+".release_jitter_days", r.ReleaseJitterDays)
		v.positive(p+".price_pressure", r.PricePressure)
		if r.StartBenchmark > r.EndBenchmark {
			v.add(p, "start_benchmark greater than end_benchmark")
		}
	}
	v.uniqueIDs("rival", rids)
	gids := make([]string, len(b.Regions))
	shareSum := 0.0
	for i, r := range b.Regions {
		gids[i] = r.ID
		shareSum += r.MarketShare
		switch r.Entry {
		case "none", "compliance", "datacenter":
		default:
			v.add(fmt.Sprintf("region[%d].entry", i), "unknown entry %q", r.Entry)
		}
	}
	v.uniqueIDs("region", gids)
	if len(b.Regions) > 0 && math.Abs(shareSum-1) > 1e-9 {
		v.add("region.market_share", "shares must sum to 1, got %v", shareSum)
	}

	// Training and data.
	t := b.Training
	v.nonNeg("training.unlock_lifetime", t.UnlockLifetime)
	v.positive("training.loss_A", t.LossA)
	v.positive("training.loss_B", t.LossB)
	v.positive("training.alpha", t.Alpha)
	v.positive("training.beta", t.Beta)
	v.positive("training.benchmark_k", t.BenchmarkK)
	if t.MaxModels <= 0 {
		v.add("training.max_models", "must be > 0")
	}
	for k, x := range t.Architecture {
		if !(x > 0 && x <= 1) {
			v.add("training.architecture."+k, "must be in (0, 1]")
		}
	}
	v.nonNeg("data.unlock_lifetime", b.Data.UnlockLifetime)
	if !(b.Data.ScraperR > 1) {
		v.add("data.scraper_r", "must be > 1")
	}

	v.nonNeg("talent.unlock_lifetime", b.Talent.UnlockLifetime)
	cids := make([]string, len(b.TalentClasses))
	for i, c := range b.TalentClasses {
		cids[i] = c.ID
		v.unit(fmt.Sprintf("talent_class[%d].salary_share", i), c.SalaryShare)
	}
	v.uniqueIDs("talent_class", cids)

	c := b.Contracts
	v.nonNeg("contracts.unlock_lifetime", c.UnlockLifetime)
	v.rangePair("contracts.duration_min", c.DurationMin)
	v.rangePair("contracts.volume_mult", c.VolumeMult)
	v.rangePair("contracts.reward_minutes", c.RewardMinutes)

	v.nonNeg("safety.unlock_lifetime", b.Safety.UnlockLifetime)
	if b.Safety.IncidentEveryHBase > b.Safety.IncidentEveryHMax {
		v.add("safety.incident_every_h_base", "greater than incident_every_h_max")
	}
	v.nonNeg("investors.unlock_lifetime", b.Investors.UnlockLifetime)
	v.rangePair("investors.target_growth", b.Investors.TargetGrowth)

	// Prestige.
	p1 := b.Prestige.Layer1
	v.positive("prestige.layer1.unlock_run_revenue", p1.UnlockRunRevenue)
	v.positive("prestige.layer1.param_coef", p1.ParamCoef)
	v.positive("prestige.layer1.param_divisor", p1.ParamDivisor)
	v.positive("prestige.layer1.param_exponent", p1.ParamExponent)
	v.nonNeg("prestige.layer1.bonus_per_param", p1.BonusPerParam)
	if len(p1.Versions) == 0 {
		v.add("prestige.layer1.versions", "must not be empty")
	}
	if p1.UnlockRunRevenue < b.Infra.UnlockLifetime {
		v.add("prestige.layer1.unlock_run_revenue", "must not be below infra.unlock_lifetime (infrastructure comes first)")
	}
	p2 := b.Prestige.Layer2
	v.positive("prestige.layer2.unlock_lifetime", p2.UnlockLifetime)
	v.positive("prestige.layer2.capital_coef", p2.CapitalCoef)
	v.positive("prestige.layer2.capital_exponent", p2.CapitalExponent)
	if len(p2.Rounds) == 0 {
		v.add("prestige.layer2.rounds", "must not be empty")
	}
	v.nonNeg("research.unlock_lifetime", b.Research.UnlockLifetime)
	v.nonNeg("agi.unlock_lifetime", b.AGI.UnlockLifetime)
	aids := make([]string, len(b.AGIStages))
	for i, s := range b.AGIStages {
		aids[i] = s.ID
		v.positive(fmt.Sprintf("agi_stage[%d].cost", i), s.Cost)
		if i > 0 && s.Cost <= b.AGIStages[i-1].Cost {
			v.add(fmt.Sprintf("agi_stage[%d].cost", i), "stage costs must increase")
		}
	}
	v.uniqueIDs("agi_stage", aids)

	// Dopamine.
	v.unit("crit.chance", b.Crit.Chance)
	if !(b.Crit.Mult >= 1) {
		v.add("crit.mult", "must be >= 1")
	}
	if !(b.Combo.MaxMult >= 1) {
		v.add("combo.max_mult", "must be >= 1")
	}
	v.positive("combo.step", b.Combo.Step)
	v.positive("combo.decay_after_s", b.Combo.DecayAfterS)
	if b.Streak.MaxDay <= 0 {
		v.add("streak.max_day", "must be > 0")
	}
	v.nonNeg("events.unlock_lifetime", b.Events.UnlockLifetime)
	v.rangePair("events.interval_s", b.Events.IntervalS)
	eids := make([]string, len(b.Event))
	for i, ev := range b.Event {
		eids[i] = ev.ID
		p := fmt.Sprintf("event[%d](%s)", i, ev.ID)
		v.positive(p+".weight", ev.Weight)
		switch ev.Effect {
		case "profit", "card_chance", "rack_cost", "hype":
			v.positive(p+".duration_s", ev.DurationS)
		case "profit_tiers":
			v.positive(p+".duration_s", ev.DurationS)
			for _, tier := range ev.Tiers {
				if tier < 1 || tier > MaxBusinesses {
					v.add(p+".tiers", "tier %d out of 1..%d", tier, MaxBusinesses)
				}
			}
		case "reaction":
			v.positive(p+".window_s", ev.WindowS)
			if ev.Key == "" {
				v.add(p+".key", "reaction events need a key")
			}
		default:
			v.add(p+".effect", "unknown effect %q", ev.Effect)
		}
	}
	v.uniqueIDs("event", eids)

	cd := b.Cards
	v.nonNeg("cards.unlock_lifetime", cd.UnlockLifetime)
	v.unit("cards.drop_chance_per_cycle", cd.DropChancePerCycle)
	rarities := []string{"common", "rare", "epic", "legendary"}
	for _, r := range rarities {
		if _, ok := cd.RarityWeights[r]; !ok {
			v.add("cards.rarity_weights", "missing rarity %q", r)
		}
	}
	kids := make([]string, len(b.Card))
	for i, cc := range b.Card {
		kids[i] = cc.ID
		p := fmt.Sprintf("card[%d](%s)", i, cc.ID)
		if _, ok := cd.RarityWeights[cc.Rarity]; !ok {
			v.add(p+".rarity", "unknown rarity %q", cc.Rarity)
		}
		if cc.Effect == "profit_business" {
			if _, ok := bizIDs[cc.Target]; !ok {
				v.add(p+".target", "unknown business id %q", cc.Target)
			}
		}
	}
	v.uniqueIDs("card", kids)

	v.nonNeg("rules.unlock_lifetime", b.Rules.UnlockLifetime)
	if b.Rules.SlotsStart <= 0 || b.Rules.SlotsStart > b.Rules.SlotsMax {
		v.add("rules.slots_start", "must be in 1..slots_max")
	}

	if len(b.UI.Colors.TierGradient) != MaxBusinesses {
		v.add("ui.colors.tier_gradient", "need %d colours, got %d", MaxBusinesses, len(b.UI.Colors.TierGradient))
	}

	if len(v.errs) == 0 {
		return nil
	}
	sort.SliceStable(v.errs, func(i, j int) bool { return v.errs[i].Error() < v.errs[j].Error() })
	return fmt.Errorf("invalid balance (%d problems):\n%w", len(v.errs), errors.Join(v.errs...))
}
