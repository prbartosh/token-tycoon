// Package balance loads, merges and validates the game's balance constants.
//
// The default file is embedded (balance.toml). A local file at
// ~/.config/token-tycoon/balance.toml can override any subset of keys.
// Every section of the file has a typed struct here, including systems that
// later milestones implement, so those milestones only add logic.
package balance

// MaxBusinesses is the fixed number of business tiers. The simulation state
// uses fixed arrays of this size.
const MaxBusinesses = 10

// Balance is the whole, validated balance file plus derived lookup tables.
type Balance struct {
	Meta             Meta             `toml:"meta"`
	Engine           Engine           `toml:"engine"`
	Offline          Offline          `toml:"offline"`
	Start            Start            `toml:"start"`
	BusinessDefaults BusinessDefaults `toml:"business_defaults"`
	Businesses       []Business       `toml:"business"`
	Milestones       Milestones       `toml:"milestones"`
	Upgrades         []Upgrade        `toml:"upgrade"`
	Infra            Infra            `toml:"infra"`
	GPUGens          []GPUGen         `toml:"gpu_gen"`
	Energy           Energy           `toml:"energy"`
	Market           Market           `toml:"market"`
	Rivals           []Rival          `toml:"rival"`
	Regions          []Region         `toml:"region"`
	Training         Training         `toml:"training"`
	Data             Data             `toml:"data"`
	Talent           Talent           `toml:"talent"`
	TalentClasses    []TalentClass    `toml:"talent_class"`
	Contracts        Contracts        `toml:"contracts"`
	Safety           Safety           `toml:"safety"`
	Investors        Investors        `toml:"investors"`
	Prestige         Prestige         `toml:"prestige"`
	Research         Research         `toml:"research"`
	AGI              AGI              `toml:"agi"`
	AGIStages        []AGIStage       `toml:"agi_stage"`
	Crit             Crit             `toml:"crit"`
	Combo            Combo            `toml:"combo"`
	Streak           Streak           `toml:"streak"`
	Events           Events           `toml:"events"`
	Event            []Event          `toml:"event"`
	Cards            Cards            `toml:"cards"`
	Card             []Card           `toml:"card"`
	Rules            Rules            `toml:"rules"`
	UI               UI               `toml:"ui"`

	idx index `toml:"-"`
}

type Meta struct {
	Version    int `toml:"version"`
	TargetDays int `toml:"target_days"`
}

type Engine struct {
	SimStepMs     int     `toml:"sim_step_ms"`
	FPSAnimation  int     `toml:"fps_animation"`
	FPSNormal     int     `toml:"fps_normal"`
	FPSBackground int     `toml:"fps_background"`
	OfflineStepS  float64 `toml:"offline_step_s"`
	AutosaveS     float64 `toml:"autosave_s"`
	Backups       int     `toml:"backups"`
	UndoWindowS   float64 `toml:"undo_window_s"`
}

type Offline struct {
	EfficiencyStart float64 `toml:"efficiency_start"`
	EfficiencyMax   float64 `toml:"efficiency_max"`
	CapHoursStart   float64 `toml:"cap_hours_start"`
	CapHoursMax     float64 `toml:"cap_hours_max"`
}

type Start struct {
	Money          float64        `toml:"money"`
	FreeBusinesses map[string]int `toml:"free_businesses"`
}

type BusinessDefaults struct {
	ManagerMult         float64 `toml:"manager_mult"`
	ContinuousBarBelowS float64 `toml:"continuous_bar_below_s"`
	DrawCapPerS         float64 `toml:"draw_cap_per_s"`
}

type Business struct {
	ID             string  `toml:"id"`
	Cost           float64 `toml:"cost"`
	R              float64 `toml:"r"`
	CycleS         float64 `toml:"cycle_s"`
	Tokens         float64 `toml:"tokens"`
	Compute        float64 `toml:"compute"`
	ManagerCost    float64 `toml:"manager_cost"` // 0 = Cost * ManagerMult
	UnlockLifetime float64 `toml:"unlock_lifetime"`
}

type Milestones struct {
	SpeedThresholds  []int   `toml:"speed_thresholds"`
	SpeedMult        float64 `toml:"speed_mult"`
	ProfitStart      int     `toml:"profit_start"`
	ProfitEvery      int     `toml:"profit_every"`
	ProfitMult       float64 `toml:"profit_mult"`
	GlobalThresholds []int   `toml:"global_thresholds"`
	GlobalEvery      int     `toml:"global_every"`
	GlobalMult       float64 `toml:"global_mult"`
}

// Upgrade effects.
const (
	EffectProfit  = "profit"
	EffectCost    = "cost"
	EffectCycle   = "cycle"
	EffectCompute = "compute"
)

type Upgrade struct {
	ID     string   `toml:"id"`
	Cost   float64  `toml:"cost"`
	Target []string `toml:"target"` // business ids or ["all"]
	Effect string   `toml:"effect"`
	Value  float64  `toml:"value"`
}

type Infra struct {
	UnlockLifetime    float64 `toml:"unlock_lifetime"`
	StarterHeadroom   float64 `toml:"starter_headroom"`
	RackR             float64 `toml:"rack_r"`
	RackSellback      float64 `toml:"rack_sellback"`
	CloudRevenueShare float64 `toml:"cloud_revenue_share"`
}

type GPUGen struct {
	Gen          int     `toml:"gen"`
	RackCost     float64 `toml:"rack_cost"`
	Compute      float64 `toml:"compute"`
	MW           float64 `toml:"mw"`
	FlopsPerRack float64 `toml:"flops_per_rack"`
}

type Energy struct {
	GridCostCoef          float64   `toml:"grid_cost_coef"`
	GridPeakAvailability  float64   `toml:"grid_peak_availability"`
	GridPeakHours         []float64 `toml:"grid_peak_hours"`
	IndexMean             float64   `toml:"index_mean"`
	IndexDailyAmplitude   float64   `toml:"index_daily_amplitude"`
	IndexNoise            float64   `toml:"index_noise"`
	PPADays               float64   `toml:"ppa_days"`
	SMRCostMinutes        float64   `toml:"smr_cost_minutes"`
	SMRMW                 float64   `toml:"smr_mw"`
	OrbitalUnlockLifetime float64   `toml:"orbital_unlock_lifetime"`
	OrbitalCostMinutes    float64   `toml:"orbital_cost_minutes"`
	OrbitalMW             float64   `toml:"orbital_mw"`
	OrbitalComputeBonus   float64   `toml:"orbital_compute_bonus"`
}

type Market struct {
	UnlockLifetime        float64   `toml:"unlock_lifetime"`
	RegionsUnlockLifetime float64   `toml:"regions_unlock_lifetime"`
	PriceBase             float64   `toml:"price_base"`
	ShareExponent         float64   `toml:"share_exponent"`
	ShareNeutral          float64   `toml:"share_neutral"`
	DemandClamp           []float64 `toml:"demand_clamp"`
	QualityClamp          []float64 `toml:"quality_clamp"`
	QualityDivisor        float64   `toml:"quality_divisor"`
	DeflationPerDay       float64   `toml:"deflation_per_day"`
	HypeHalfLifeH         float64   `toml:"hype_half_life_h"`
	Segments              Segments  `toml:"segments"`
}

type Segments struct {
	FreePrice              float64 `toml:"free_price"`
	FreeHypePerShare       float64 `toml:"free_hype_per_share"`
	ProPrice               float64 `toml:"pro_price"`
	EnterprisePrice        float64 `toml:"enterprise_price"`
	EnterpriseMinBenchmark float64 `toml:"enterprise_min_benchmark"`
}

type Rival struct {
	ID                string  `toml:"id"`
	Style             string  `toml:"style"`
	ReleaseEveryDays  float64 `toml:"release_every_days"`
	ReleaseJitterDays float64 `toml:"release_jitter_days"`
	StartBenchmark    float64 `toml:"start_benchmark"`
	EndBenchmark      float64 `toml:"end_benchmark"`
	PricePressure     float64 `toml:"price_pressure"`
}

type Region struct {
	ID               string  `toml:"id"`
	MarketShare      float64 `toml:"market_share"`
	Entry            string  `toml:"entry"` // none | compliance | datacenter
	EntryCostMinutes float64 `toml:"entry_cost_minutes"`
}

type Training struct {
	UnlockLifetime        float64            `toml:"unlock_lifetime"`
	LossE                 float64            `toml:"loss_E"`
	LossA                 float64            `toml:"loss_A"`
	LossB                 float64            `toml:"loss_B"`
	Alpha                 float64            `toml:"alpha"`
	Beta                  float64            `toml:"beta"`
	BenchmarkK            float64            `toml:"benchmark_k"`
	FlopsPerParamToken    float64            `toml:"flops_per_param_token"`
	ClusterRackExponent   float64            `toml:"cluster_rack_exponent"`
	MaxModels             int                `toml:"max_models"`
	MaxParamsLevels       []float64          `toml:"max_params_levels"`
	SyntheticMinBenchmark float64            `toml:"synthetic_min_benchmark"`
	SyntheticPenaltyAbove float64            `toml:"synthetic_penalty_above"`
	Architecture          map[string]float64 `toml:"architecture"`
}

type Data struct {
	UnlockLifetime            float64   `toml:"unlock_lifetime"`
	ScraperCost               float64   `toml:"scraper_cost"`
	ScraperR                  float64   `toml:"scraper_r"`
	ScraperTokensPerS         float64   `toml:"scraper_tokens_per_s"`
	ScraperIncidentChancePerH float64   `toml:"scraper_incident_chance_per_h"`
	LicenseCostMinutes        float64   `toml:"license_cost_minutes"`
	LicenseTokens             []float64 `toml:"license_tokens"`
}

type Talent struct {
	UnlockLifetime   float64 `toml:"unlock_lifetime"`
	PoachWindowS     float64 `toml:"poach_window_s"`
	PoachCounterMult float64 `toml:"poach_counter_mult"`
	PoachCounterDays float64 `toml:"poach_counter_days"`
}

type TalentClass struct {
	ID          string  `toml:"id"`
	SalaryShare float64 `toml:"salary_share"`
	Speed       float64 `toml:"speed"`
	Trait       bool    `toml:"trait"`
}

type Contracts struct {
	UnlockLifetime float64   `toml:"unlock_lifetime"`
	MaxActive      int       `toml:"max_active"`
	DurationMin    []float64 `toml:"duration_min"`
	VolumeMult     []float64 `toml:"volume_mult"`
	RewardMinutes  []float64 `toml:"reward_minutes"`
	ReputationGain float64   `toml:"reputation_gain"`
	ReputationLoss float64   `toml:"reputation_loss"`
}

type Safety struct {
	UnlockLifetime     float64 `toml:"unlock_lifetime"`
	IncidentEveryHBase float64 `toml:"incident_every_h_base"`
	IncidentEveryHMax  float64 `toml:"incident_every_h_max"`
	HypeLoss           float64 `toml:"hype_loss"`
	ReactionWindowS    float64 `toml:"reaction_window_s"`
}

type Investors struct {
	UnlockLifetime   float64   `toml:"unlock_lifetime"`
	QuarterH         float64   `toml:"quarter_h"`
	TargetGrowth     []float64 `toml:"target_growth"`
	BeatBonusCapital float64   `toml:"beat_bonus_capital"`
}

type Prestige struct {
	Layer1 PrestigeLayer1 `toml:"layer1"`
	Layer2 PrestigeLayer2 `toml:"layer2"`
}

type PrestigeLayer1 struct {
	UnlockRunRevenue float64  `toml:"unlock_run_revenue"`
	ParamCoef        float64  `toml:"param_coef"`
	ParamDivisor     float64  `toml:"param_divisor"`
	ParamExponent    float64  `toml:"param_exponent"`
	BonusPerParam    float64  `toml:"bonus_per_param"`
	Versions         []string `toml:"versions"`
}

type PrestigeLayer2 struct {
	UnlockLifetime   float64  `toml:"unlock_lifetime"`
	CapitalCoef      float64  `toml:"capital_coef"`
	CapitalLogOffset float64  `toml:"capital_log_offset"`
	CapitalExponent  float64  `toml:"capital_exponent"`
	Rounds           []string `toml:"rounds"`
}

type Research struct {
	UnlockLifetime float64 `toml:"unlock_lifetime"`
}

type AGI struct {
	UnlockLifetime float64 `toml:"unlock_lifetime"`
}

type AGIStage struct {
	ID     string  `toml:"id"`
	Cost   float64 `toml:"cost"`
	Effect string  `toml:"effect"`
}

type Crit struct {
	Chance float64 `toml:"chance"`
	Mult   float64 `toml:"mult"`
}

type Combo struct {
	MaxMult     float64 `toml:"max_mult"`
	Step        float64 `toml:"step"`
	DecayAfterS float64 `toml:"decay_after_s"`
}

type Streak struct {
	MaxDay             int     `toml:"max_day"`
	BonusPerDayMinutes float64 `toml:"bonus_per_day_minutes"`
	BreakPenalty       int     `toml:"break_penalty"`
}

type Events struct {
	UnlockLifetime float64   `toml:"unlock_lifetime"`
	IntervalS      []float64 `toml:"interval_s"`
}

// Event is a random event definition; fields not used by an effect stay zero.
type Event struct {
	ID             string  `toml:"id"`
	Weight         float64 `toml:"weight"`
	Effect         string  `toml:"effect"` // profit | card_chance | rack_cost | reaction | profit_tiers | hype
	Value          float64 `toml:"value"`
	DurationS      float64 `toml:"duration_s"`
	Key            string  `toml:"key"`
	WindowS        float64 `toml:"window_s"`
	RewardMinutes  float64 `toml:"reward_minutes"`
	PenaltyMinutes float64 `toml:"penalty_minutes"`
	Tiers          []int   `toml:"tiers"`
}

type Cards struct {
	UnlockLifetime     float64            `toml:"unlock_lifetime"`
	DropChancePerCycle float64            `toml:"drop_chance_per_cycle"`
	LevelBonusMult     float64            `toml:"level_bonus_mult"`
	RarityWeights      map[string]float64 `toml:"rarity_weights"`
	SetBonus           map[string]float64 `toml:"set_bonus"`
}

type Card struct {
	ID     string  `toml:"id"`
	Rarity string  `toml:"rarity"`
	Effect string  `toml:"effect"`
	Target string  `toml:"target"`
	Value  float64 `toml:"value"`
}

type Rules struct {
	UnlockLifetime float64 `toml:"unlock_lifetime"`
	SlotsStart     int     `toml:"slots_start"`
	SlotsMax       int     `toml:"slots_max"`
	EvalEveryS     float64 `toml:"eval_every_s"`
}

type UI struct {
	NumberSuffixes      []string `toml:"number_suffixes"`
	LetterSuffixesAfter string   `toml:"letter_suffixes_after"`
	AdvisorDefault      bool     `toml:"advisor_default"`
	MinWidth            int      `toml:"min_width"`
	TwoPanelWidth       int      `toml:"two_panel_width"`
	ThreePanelWidth     int      `toml:"three_panel_width"`
	SparklineSeconds    float64  `toml:"sparkline_seconds"`
	Colors              Colors   `toml:"colors"`
}

type Colors struct {
	BG           string   `toml:"bg"`
	Text         string   `toml:"text"`
	Dim          string   `toml:"dim"`
	MoneyFrom    string   `toml:"money_from"`
	MoneyTo      string   `toml:"money_to"`
	CostBad      string   `toml:"cost_bad"`
	Gold         string   `toml:"gold"`
	Magenta      string   `toml:"magenta"`
	Params       string   `toml:"params"`
	Capital      string   `toml:"capital"`
	Rare         string   `toml:"rare"`
	Epic         string   `toml:"epic"`
	TierGradient []string `toml:"tier_gradient"`
}
