// Package balance loads, merges and validates the game's balance constants.
// The default file is embedded (balance.toml); a local file in the config dir can override any subset of keys.
package balance

const MaxBusinesses = 10

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
	TalentTraits     []TalentTrait    `toml:"talent_trait"`
	Contracts        Contracts        `toml:"contracts"`
	Safety           Safety           `toml:"safety"`
	Incidents        []Incident       `toml:"safety_incident"`
	Investors        Investors        `toml:"investors"`
	Prestige         Prestige         `toml:"prestige"`
	Research         Research         `toml:"research"`
	ResearchNodes    []ResearchNode   `toml:"research_node"`
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
	Pacing           Pacing           `toml:"pacing"`

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
	ManagerCost    float64 `toml:"manager_cost"`
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

const (
	EffectProfit  = "profit"
	EffectCost    = "cost"
	EffectCycle   = "cycle"
	EffectCompute = "compute"
)

type Upgrade struct {
	ID          string   `toml:"id"`
	Cost        float64  `toml:"cost"`
	Target      []string `toml:"target"`
	Effect      string   `toml:"effect"`
	Value       float64  `toml:"value"`
	ComputeMult float64  `toml:"compute_mult"`
}

type Infra struct {
	UnlockLifetime    float64 `toml:"unlock_lifetime"`
	StarterHeadroom   float64 `toml:"starter_headroom"`
	RackR             float64 `toml:"rack_r"`
	RackSellback      float64 `toml:"rack_sellback"`
	CloudRevenueShare float64 `toml:"cloud_revenue_share"`
}

type GPUGen struct {
	Gen            int     `toml:"gen"`
	RackCost       float64 `toml:"rack_cost"`
	Compute        float64 `toml:"compute"`
	MW             float64 `toml:"mw"`
	FlopsPerRack   float64 `toml:"flops_per_rack"`
	UnlockLifetime float64 `toml:"unlock_lifetime"`
	Research       bool    `toml:"research"`
}

type Energy struct {
	GridCostCoef          float64   `toml:"grid_cost_coef"`
	GridMWCost            float64   `toml:"grid_mw_cost"`
	PPAMWCost             float64   `toml:"ppa_mw_cost"`
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
	RivalSpanDays         float64   `toml:"rival_span_days"`
	BaseBenchmark         float64   `toml:"base_benchmark"`
	ReleaseHype           float64   `toml:"release_hype"`
	PressureFloor         float64   `toml:"pressure_floor"`
	Segments              Segments  `toml:"segments"`
}

type Segments struct {
	FreePrice              float64 `toml:"free_price"`
	FreeHypePerShare       float64 `toml:"free_hype_per_share"`
	ProPrice               float64 `toml:"pro_price"`
	EnterprisePrice        float64 `toml:"enterprise_price"`
	EnterpriseMinBenchmark float64 `toml:"enterprise_min_benchmark"`
	EnterpriseMaxAlloc     float64 `toml:"enterprise_max_alloc"`
}

const MaxRivals = 8

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
	ID                string  `toml:"id"`
	MarketShare       float64 `toml:"market_share"`
	Entry             string  `toml:"entry"`
	EntryCostMinutes  float64 `toml:"entry_cost_minutes"`
	EnterpriseCostMin float64 `toml:"enterprise_cost_minutes"`
}

const (
	EntryNone       = "none"
	EntryCompliance = "compliance"
	EntryDatacenter = "datacenter"
)

const MaxRegions = 8

type Training struct {
	UnlockLifetime         float64            `toml:"unlock_lifetime"`
	LossE                  float64            `toml:"loss_E"`
	LossA                  float64            `toml:"loss_A"`
	LossB                  float64            `toml:"loss_B"`
	Alpha                  float64            `toml:"alpha"`
	Beta                   float64            `toml:"beta"`
	BenchmarkK             float64            `toml:"benchmark_k"`
	FlopsPerParamToken     float64            `toml:"flops_per_param_token"`
	ClusterRackExponent    float64            `toml:"cluster_rack_exponent"`
	MaxModels              int                `toml:"max_models"`
	MaxQueue               int                `toml:"max_queue"`
	MinParams              float64            `toml:"min_params"`
	MaxParamsLevels        []float64          `toml:"max_params_levels"`
	MaxParamsUnlock        []float64          `toml:"max_params_unlock"`
	SyntheticMinBenchmark  float64            `toml:"synthetic_min_benchmark"`
	SyntheticPenaltyAbove  float64            `toml:"synthetic_penalty_above"`
	SyntheticPenalty       float64            `toml:"synthetic_penalty"`
	SyntheticFlopsPerParam float64            `toml:"synthetic_flops_per_param"`
	Architecture           map[string]float64 `toml:"architecture"`
	ArchitectureCard       map[string]string  `toml:"architecture_card"`
}

type Arch struct {
	ID   string
	Mult float64
	Card int
}

type Data struct {
	UnlockLifetime            float64   `toml:"unlock_lifetime"`
	ScraperCost               float64   `toml:"scraper_cost"`
	ScraperR                  float64   `toml:"scraper_r"`
	ScraperTokensPerS         float64   `toml:"scraper_tokens_per_s"`
	ScraperIncidentChancePerH float64   `toml:"scraper_incident_chance_per_h"`
	ScraperIncidentPauseS     float64   `toml:"scraper_incident_pause_s"`
	ScraperIncidentFineMin    float64   `toml:"scraper_incident_fine_minutes"`
	LicenseCostMinutes        float64   `toml:"license_cost_minutes"`
	LicenseTokens             []float64 `toml:"license_tokens"`
}

type Talent struct {
	UnlockLifetime   float64  `toml:"unlock_lifetime"`
	PoachWindowS     float64  `toml:"poach_window_s"`
	PoachCounterMult float64  `toml:"poach_counter_mult"`
	PoachCounterDays float64  `toml:"poach_counter_days"`
	MaxStaff         int      `toml:"max_staff"`
	PoachChancePerH  float64  `toml:"poach_chance_per_h"`
	Names            []string `toml:"names"`
}

type TalentClass struct {
	ID          string  `toml:"id"`
	SalaryShare float64 `toml:"salary_share"`
	Speed       float64 `toml:"speed"`
	Trait       bool    `toml:"trait"`
	HireMinutes float64 `toml:"hire_minutes"`
}

type TalentTrait struct {
	ID          string  `toml:"id"`
	Speed       float64 `toml:"speed"`
	LossMult    float64 `toml:"loss_mult"`
	HypePerH    float64 `toml:"hype_per_h"`
	ComputeMult float64 `toml:"compute_mult"`
}

type Contracts struct {
	UnlockLifetime float64   `toml:"unlock_lifetime"`
	MaxActive      int       `toml:"max_active"`
	DurationMin    []float64 `toml:"duration_min"`
	VolumeMult     []float64 `toml:"volume_mult"`
	RewardMinutes  []float64 `toml:"reward_minutes"`
	ReputationGain float64   `toml:"reputation_gain"`
	ReputationLoss float64   `toml:"reputation_loss"`
	Offers         int       `toml:"offers"`
	OfferEveryS    float64   `toml:"offer_every_s"`
	OfferLifeS     float64   `toml:"offer_life_s"`
	BenchOffset    []float64 `toml:"benchmark_offset"`
	MinBenchFloor  float64   `toml:"min_benchmark_floor"`
	RepRewardBonus float64   `toml:"reputation_reward_bonus"`
	ReputationMax  float64   `toml:"reputation_max"`
}

type Safety struct {
	UnlockLifetime     float64 `toml:"unlock_lifetime"`
	IncidentEveryHBase float64 `toml:"incident_every_h_base"`
	IncidentEveryHMax  float64 `toml:"incident_every_h_max"`
	HypeLoss           float64 `toml:"hype_loss"`
	ReactionWindowS    float64 `toml:"reaction_window_s"`
	Key                string  `toml:"key"`
	ReputationBonus    float64 `toml:"reputation_bonus"`
}

type Incident struct {
	ID       string  `toml:"id"`
	Weight   float64 `toml:"weight"`
	Severity float64 `toml:"severity"`
}

type Investors struct {
	UnlockLifetime   float64   `toml:"unlock_lifetime"`
	QuarterH         float64   `toml:"quarter_h"`
	TargetGrowth     []float64 `toml:"target_growth"`
	BeatBonusCapital float64   `toml:"beat_bonus_capital"`
	GrowthWeight     float64   `toml:"growth_weight"`
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
	ValuationMult    float64  `toml:"valuation_mult"`
}

type Research struct {
	UnlockLifetime    float64 `toml:"unlock_lifetime"`
	AutoPrestigeRatio float64 `toml:"auto_prestige_ratio"`
}

const MaxResearchNodes = 192

var Branches = [...]string{"efficiency", "scale", "product", "automation", "safety", "luck"}

const (
	ResComputeMult    = "compute_mult"
	ResEnergyCost     = "energy_cost"
	ResMaxParamsMult  = "max_params_mult"
	ResGPUGen         = "gpu_gen"
	ResEnterprise     = "enterprise_alloc"
	ResContractSlots  = "contract_slots"
	ResContractReward = "contract_reward"
	ResMaxModels      = "max_models"
	ResRuleSlots      = "rule_slots"
	ResAutoPrestige   = "auto_prestige"
	ResOfflineEff     = "offline_eff"
	ResOfflineCapH    = "offline_cap_h"
	ResOfflineRules   = "offline_rules"
	ResIncidentMult   = "incident_mult"
	ResPenaltyMult    = "penalty_mult"
	ResCritChance     = "crit_chance"
	ResCardChance     = "card_chance"
	ResRareCards      = "rare_cards"
	ResComplianceCost = "compliance_cost"
)

type ResearchNode struct {
	ID       string   `toml:"id"`
	Branch   string   `toml:"branch"`
	Cost     float64  `toml:"cost"`
	TimeH    float64  `toml:"time_h"`
	Requires []string `toml:"requires"`
	Effect   string   `toml:"effect"`
	Value    float64  `toml:"value"`
}

type AGI struct {
	UnlockLifetime  float64 `toml:"unlock_lifetime"`
	FinalTrainingH  float64 `toml:"final_training_h"`
	FinalBenchmark  float64 `toml:"final_benchmark"`
	SIEveryOOM      float64 `toml:"si_every_oom"`
	SIBonusPerLevel float64 `toml:"si_bonus_per_level"`
	SIMaxLevels     int     `toml:"si_max_levels"`
}

const (
	AGINoIncidents   = "no_incidents"
	AGIProfitX2      = "all_profit_x2"
	AGISynthetic     = "synthetic_no_penalty"
	AGIFinalTraining = "final_training"
	AGIEnding        = "ending"
)

const MaxSILevels = 32

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

type Event struct {
	ID             string  `toml:"id"`
	Weight         float64 `toml:"weight"`
	Effect         string  `toml:"effect"`
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
	MaxLevel           int                `toml:"max_level"`
	RarityWeights      map[string]float64 `toml:"rarity_weights"`
	SetBonus           map[string]float64 `toml:"set_bonus"`
}

const (
	CardProfitAll      = "profit_all"
	CardProfitBusiness = "profit_business"
	CardComputeAll     = "compute_all"
	CardCycleAll       = "cycle_all"
	CardCostAll        = "cost_all"
	CardCritChance     = "crit_chance"
	CardCritMult       = "crit_mult"
	CardCardChance     = "card_chance"
	CardReactionReward = "reaction_reward"
	CardEnergyCost     = "energy_cost"
	CardRackCost       = "rack_cost"
	CardOffline        = "offline"
	CardTrainingSpeed  = "training_speed"
	CardRevealRatio    = "reveal_optimal_ratio"
)

var Rarities = [...]string{"common", "rare", "epic", "legendary"}

const MaxCards = 128

const (
	EventProfit      = "profit"
	EventProfitTiers = "profit_tiers"
	EventCardChance  = "card_chance"
	EventRackCost    = "rack_cost"
	EventHype        = "hype"
	EventReaction    = "reaction"
)

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

type Pacing struct {
	GreedySessionMin   float64     `toml:"greedy_session_min"`
	GreedySessionsDay  int         `toml:"greedy_sessions_per_day"`
	FlatMaxS           float64     `toml:"flat_max_s"`
	FirstManagerS      []float64   `toml:"first_manager_s"`
	InfraMin           []float64   `toml:"infra_min"`
	PrestigeMin        []float64   `toml:"prestige_min"`
	SessionNewS        float64     `toml:"session_new_s"`
	GoalUntilDay       int         `toml:"goal_until_day"`
	IncomeBands        [][]float64 `toml:"income_bands"`
	IncomeToleranceOOM float64     `toml:"income_tolerance_oom"`
	FundingDays        []float64   `toml:"funding_days"`
	AGIDays            []float64   `toml:"agi_days"`
	GreedyEndDays      []float64   `toml:"greedy_end_days"`
	PassiveEndDays     []float64   `toml:"passive_end_days"`
}
