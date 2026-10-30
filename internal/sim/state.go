// Package sim is the pure game simulation: state, commands, Step and offline
// catch-up. It does no I/O, never reads the wall clock and never imports UI
// code. The same engine drives the game, offline catch-up and the simulator.
package sim

import (
	"math"
	"math/bits"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/cards"
	"token-tycoon/internal/rng"
)

const N = balance.MaxBusinesses

type Bitset [4]uint64

func (b *Bitset) Has(i int) bool { return b[i>>6]&(1<<(uint(i)&63)) != 0 }

func (b *Bitset) Set(i int) { b[i>>6] |= 1 << (uint(i) & 63) }

func (b *Bitset) Clear(i int) { b[i>>6] &^= 1 << (uint(i) & 63) }

func (b *Bitset) Count() int {
	n := 0
	for _, w := range b {
		n += bits.OnesCount64(w)
	}
	return n
}

type State struct {
	Seed     uint64      `json:"seed"`
	RNG      rng.Streams `json:"rng"`
	Clock    float64     `json:"clock_s"`
	Money    float64     `json:"money"`
	Lifetime Lifetime    `json:"lifetime"`

	GlobalMilestones int `json:"global_milestones"`

	Businesses [N]Business `json:"-"`
	Upgrades   Bitset      `json:"-"`
	Unlocks    Bitset      `json:"-"`

	Infra        Infra          `json:"infra"`
	Market       Market         `json:"market"`
	Training     Training       `json:"training"`
	Talent       []Researcher   `json:"talent"`
	Poach        Poach          `json:"poach"`
	Incident     Incident       `json:"incident"`
	Investors    Investors      `json:"investors"`
	Contracts    []Contract     `json:"contracts"`
	Offers       []Contract     `json:"contract_offers"`
	ContractSeq  int            `json:"contract_seq"`
	NextOfferS   float64        `json:"next_offer_s"`
	Research     ResearchState  `json:"-"`
	Cards        map[string]int `json:"cards"`
	Rules        []Rule         `json:"rules"`
	Prestige     Prestige       `json:"prestige"`
	AGI          AGI            `json:"agi"`
	Streak       Streak         `json:"streak"`
	RandomEvent  RandomEvent    `json:"random_event"`
	Achievements []string       `json:"achievements"`
	Settings     Settings       `json:"settings"`
	Stats        Stats          `json:"stats"`

	Achieved Bitset       `json:"-"`
	CardMods cards.Mods   `json:"-"`
	ResMods  ResearchMods `json:"-"`
	Combo    Combo        `json:"-"`
	Undo     Undo         `json:"-"`
	Derived  Derived      `json:"-"`

	rivals rivalCache
}

type Lifetime struct {
	Run         float64 `json:"run"`
	Total       float64 `json:"total"`
	Layer2Total float64 `json:"layer2_total"`
}

type Business struct {
	ID         string  `json:"id"`
	Owned      int     `json:"owned"`
	Manager    bool    `json:"manager"`
	Progress   float64 `json:"progress"`
	Running    bool    `json:"running"`
	DrawBudget float64 `json:"draw_budget"`
}

type Infra struct {
	GPUGen       int                     `json:"gpu_gen"`
	Racks        [balance.MaxGPUGens]int `json:"racks"`
	Energy       EnergyState             `json:"energy"`
	CloudEnabled bool                    `json:"cloud_enabled"`
	StarterGiven bool                    `json:"starter_given"`
}

type EnergyState struct {
	GridMW  float64 `json:"grid_mw"`
	PPA     PPA     `json:"ppa"`
	SMR     int     `json:"smr"`
	Orbital int     `json:"orbital"`
}

type PPA struct {
	MW       float64 `json:"mw"`
	Index    float64 `json:"index"`
	ExpiresS float64 `json:"expires_s"`
}

type Market struct {
	UnlockedRegions []string `json:"unlocked_regions"`
	Compliance      []string `json:"compliance"`
	Segments        Segments `json:"segments"`
	Hype            float64  `json:"hype"`
	Reputation      float64  `json:"reputation"`
	OpenedS         float64  `json:"opened_s"`
}

type Segments struct {
	Free       float64 `json:"free"`
	Pro        float64 `json:"pro"`
	Enterprise float64 `json:"enterprise"`
}

type Training struct {
	Models        []Model    `json:"models"`
	Queue         []TrainJob `json:"queue"`
	DataTokens    float64    `json:"data_tokens"`
	SynthTokens   float64    `json:"synthetic_tokens"`
	SynthShare    float64    `json:"synthetic_share"`
	Scrapers      int        `json:"scrapers"`
	Licenses      int        `json:"licenses"`
	Released      int        `json:"released"`
	ScraperPauseS float64    `json:"scraper_pause_until_s"`
}

type Model struct {
	Name      string  `json:"name"`
	Params    float64 `json:"params"`
	Tokens    float64 `json:"tokens"`
	Benchmark float64 `json:"benchmark"`
	ReleasedS float64 `json:"released_s"`
	Segment   string  `json:"segment"`
	Synthetic float64 `json:"synthetic"`
}

type TrainJob struct {
	Params       float64  `json:"params"`
	Tokens       float64  `json:"tokens"`
	ClusterShare float64  `json:"cluster_share"`
	Progress     float64  `json:"progress"`
	Arch         []string `json:"arch"`
	ArchMult     float64  `json:"arch_mult"`
	Synthetic    float64  `json:"synthetic"`
}

type Researcher struct {
	Class       string  `json:"class"`
	Name        string  `json:"name"`
	Trait       string  `json:"trait,omitempty"`
	RaiseUntilS float64 `json:"raise_until_s"`
}

type Prestige struct {
	Params          float64 `json:"params"`
	ParamsLifetime  float64 `json:"params_lifetime"`
	Capital         float64 `json:"capital"`
	CapitalLifetime float64 `json:"capital_lifetime"`
	RoundStartS     float64 `json:"round_start_s"`
	VersionIndex    int     `json:"version_index"`
	RoundIndex      int     `json:"round_index"`
	ResetsL1        int     `json:"resets_l1"`
	ResetsL2        int     `json:"resets_l2"`
	RunStartS       float64 `json:"run_start_s"`
}

type AGI struct {
	Stage      int                          `json:"stage"`
	Invested   float64                      `json:"invested"`
	TrainEndsS float64                      `json:"train_ends_s"`
	EndedS     float64                      `json:"ended_s"`
	SILevel    int                          `json:"si_level"`
	SIRecords  [balance.MaxSILevels]float64 `json:"si_records"`
}

type Streak struct {
	Day       int    `json:"day"`
	LastClaim string `json:"last_claim"`
}

type RandomEvent struct {
	ActiveID string  `json:"active_id"`
	EndsAtS  float64 `json:"ends_at_s"`
	NextAtS  float64 `json:"next_at_s"`
}

type Settings struct {
	Lang       string `json:"lang"`
	Notation   string `json:"notation"`
	Advisor    bool   `json:"advisor"`
	Colorblind bool   `json:"colorblind"`
	Bell       bool   `json:"bell"`
}

type Stats struct {
	PlayS           float64 `json:"play_s"`
	OfflineS        float64 `json:"offline_s"`
	Crits           int     `json:"crits"`
	BestCombo       float64 `json:"best_combo"`
	EventsCaught    int     `json:"events_caught"`
	Purchases       int     `json:"purchases"`
	ManualStarts    int     `json:"manual_starts"`
	BoostPresses    int     `json:"boost_presses"`
	EventsSeen      int     `json:"events_seen"`
	EventsMissed    int     `json:"events_missed"`
	CardsDropped    int     `json:"cards_dropped"`
	Grants          int     `json:"grants"`
	BestIncome      float64 `json:"best_income"`
	Models          int     `json:"models"`
	BestBench       float64 `json:"best_benchmark"`
	Incidents       int     `json:"scraper_incidents"`
	Hired           int     `json:"hired"`
	Poached         int     `json:"poached"`
	Retained        int     `json:"retained"`
	ContractsDone   int     `json:"contracts_done"`
	ContractsFailed int     `json:"contracts_failed"`
	ResearchDone    int     `json:"research_done"`
	RulesFired      int     `json:"rules_fired"`
	IncidentsSeen   int     `json:"incidents_seen"`
	IncidentsOK     int     `json:"incidents_handled"`
	IncidentsMissed int     `json:"incidents_missed"`
	Quarters        int     `json:"quarters"`
	QuartersBeaten  int     `json:"quarters_beaten"`
	FastestRunS     float64 `json:"fastest_run_s"`
	AGIInvested     float64 `json:"agi_invested"`
}

type Combo struct {
	Mult  float64
	IdleS float64
}

type PurchaseKind uint8

const (
	PurchaseNone PurchaseKind = iota
	PurchaseBusiness
	PurchaseManager
	PurchaseUpgrade
	PurchaseRack
	PurchaseGrid
)

type Undo struct {
	Kind       PurchaseKind
	Business   int
	Upgrade    int
	Gen        int
	Qty        int
	MW         float64
	Cost       float64
	AtS        float64
	PrevGlobal int
}

func NewState(b *balance.Balance, seed uint64) State {
	s := State{
		Seed:     seed,
		RNG:      rng.NewStreams(seed),
		Money:    b.Start.Money,
		Cards:    map[string]int{},
		Settings: Settings{Notation: "letters", Advisor: b.UI.AdvisorDefault},
		Market:   Market{Segments: Segments{Pro: 1}, UnlockedRegions: []string{}, Compliance: []string{}},
		Combo:    Combo{Mult: 1},
		Research: ResearchState{Active: -1},
	}
	normalizeTalent(&s, b)
	normalizeContracts(&s, b)
	normalizeRules(&s, b)
	normalizeM5(&s, b)
	resetBusinesses(&s, b)
	cards.Compute(b, s.Cards, &s.CardMods)
	checkUnlocks(&s, b, nil)
	Recompute(&s, b)
	return s
}

func resetBusinesses(s *State, b *balance.Balance) {
	free := b.FreeOwned()
	for i := range s.Businesses {
		s.Businesses[i] = Business{ID: b.Businesses[i].ID, Owned: free[i], DrawBudget: b.BusinessDefaults.DrawCapPerS}
	}
}

func Normalize(s *State, b *balance.Balance) {
	if s.Combo.Mult < 1 {
		s.Combo.Mult = 1
	}
	if s.Cards == nil {
		s.Cards = map[string]int{}
	}
	normalizeResearch(s, b)
	computeResearch(s, b)
	normalizeMarket(s, b)
	normalizeTraining(s, b)
	normalizeTalent(s, b)
	normalizeContracts(s, b)
	normalizeRules(s, b)
	normalizeM5(s, b)
	s.Prestige.RoundStartS = math.Min(math.Max(0, s.Prestige.RoundStartS), s.Clock)
	if !(s.Prestige.Capital >= 0) || math.IsInf(s.Prestige.Capital, 0) {
		s.Prestige.Capital = 0
	}
	for i := range s.Businesses {
		s.Businesses[i].ID = b.Businesses[i].ID
		if s.Businesses[i].Owned < 0 {
			s.Businesses[i].Owned = 0
		}
		if s.Businesses[i].Manager {
			s.Businesses[i].Running = false
		}
		bz := &s.Businesses[i]
		if !(bz.DrawBudget >= 0) {
			bz.DrawBudget = 0
		}
		bz.DrawBudget = math.Min(bz.DrawBudget, b.BusinessDefaults.DrawCapPerS)
	}
	for g := len(b.GPUGens); g < balance.MaxGPUGens; g++ {
		s.Infra.Racks[g] = 0
	}
	for g := range s.Infra.Racks {
		s.Infra.Racks[g] = max(0, s.Infra.Racks[g])
	}
	if s.RandomEvent.ActiveID != "" {
		if _, ok := b.EventIndex(s.RandomEvent.ActiveID); !ok {
			s.RandomEvent.ActiveID = ""
		}
	}
	s.GlobalMilestones = max(0, s.GlobalMilestones)
	s.Prestige.RunStartS = math.Min(math.Max(0, s.Prestige.RunStartS), s.Clock)
	cards.Compute(b, s.Cards, &s.CardMods)
	checkUnlocks(s, b, nil)
	updateGens(s, b, nil)
	Recompute(s, b)
}
