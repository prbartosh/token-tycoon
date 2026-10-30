// Package sim is the pure game simulation: state, commands, Step and offline
// catch-up. It does no I/O, never reads the wall clock and never imports UI
// code. The same engine drives the game, offline catch-up and the balance
// simulator.
package sim

import (
	"encoding/json"
	"math"
	"math/bits"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/rng"
)

// N is the number of businesses.
const N = balance.MaxBusinesses

// Bitset is a fixed 256-bit set, copyable by value.
type Bitset [4]uint64

// Has reports whether bit i is set.
func (b *Bitset) Has(i int) bool { return b[i>>6]&(1<<(uint(i)&63)) != 0 }

// Set sets bit i.
func (b *Bitset) Set(i int) { b[i>>6] |= 1 << (uint(i) & 63) }

// Clear clears bit i.
func (b *Bitset) Clear(i int) { b[i>>6] &^= 1 << (uint(i) & 63) }

// Count returns the number of set bits.
func (b *Bitset) Count() int {
	n := 0
	for _, w := range b {
		n += bits.OnesCount64(w)
	}
	return n
}

// State is the whole game state. It is designed for the full game: systems
// that arrive in later milestones already have their fields here (zero
// values until implemented). Fields tagged json:"-" are either derived or
// serialized by the save package by id (businesses, upgrades, unlocks).
type State struct {
	Seed     uint64      `json:"seed"`
	RNG      rng.Streams `json:"rng"`
	Clock    float64     `json:"clock_s"` // game seconds since the start, online and offline
	Money    float64     `json:"money"`
	Lifetime Lifetime    `json:"lifetime"`

	Businesses [N]Business `json:"-"` // save: list of {id, ...}
	Upgrades   Bitset      `json:"-"` // save: list of upgrade ids (balance order)
	Unlocks    Bitset      `json:"-"` // save: list of unlock names

	Infra        Infra          `json:"infra"`
	Market       Market         `json:"market"`
	Training     Training       `json:"training"`
	Talent       []Researcher   `json:"talent"`
	Contracts    []Contract     `json:"contracts"`
	Research     []string       `json:"research"`
	Cards        map[string]int `json:"cards"`
	Rules        []Rule         `json:"rules"`
	Prestige     Prestige       `json:"prestige"`
	AGI          AGI            `json:"agi"`
	Streak       Streak         `json:"streak"`
	RandomEvent  RandomEvent    `json:"random_event"`
	Achievements []string       `json:"achievements"`
	Settings     Settings       `json:"settings"`
	Stats        Stats          `json:"stats"`

	// Transient: not saved.
	Combo   Combo   `json:"-"`
	Undo    Undo    `json:"-"`
	Derived Derived `json:"-"`
}

// Lifetime tracks revenue. Run resets on prestige layer 1, Layer2Total on
// layer 2, Total never.
type Lifetime struct {
	Run         float64 `json:"run"`
	Total       float64 `json:"total"`
	Layer2Total float64 `json:"layer2_total"`
}

// Business is one generator tier.
type Business struct {
	ID       string  `json:"id"`
	Owned    int     `json:"owned"`
	Manager  bool    `json:"manager"`
	Progress float64 `json:"progress"` // 0..1 of the current cycle
	Running  bool    `json:"running"`  // a manual cycle is in progress (no manager)
	// DrawBudget is the number of cycles that may still be rolled for crits
	// and cards (DECISIONS 14); refills at draw_cap_per_s. It is saved so a
	// loaded game continues exactly like the uninterrupted one.
	DrawBudget float64 `json:"draw_budget"`
}

// Infra is the compute and energy side (M2).
type Infra struct {
	GPUGen       int                     `json:"gpu_gen"`
	Racks        [balance.MaxGPUGens]int `json:"racks"` // index gen-1
	Energy       EnergyState             `json:"energy"`
	CloudEnabled bool                    `json:"cloud_enabled"`
	StarterGiven bool                    `json:"starter_given"`
}

// EnergyState holds energy sources (M2).
type EnergyState struct {
	GridMW  float64 `json:"grid_mw"`
	PPA     PPA     `json:"ppa"`
	SMR     int     `json:"smr"`
	Orbital int     `json:"orbital"`
}

// PPA is a fixed-price power contract.
type PPA struct {
	MW       float64 `json:"mw"`
	Index    float64 `json:"index"`
	ExpiresS float64 `json:"expires_s"`
}

// Market is pricing, share and rivals (M3).
type Market struct {
	UnlockedRegions []string `json:"unlocked_regions"`
	Segments        Segments `json:"segments"`
	Hype            float64  `json:"hype"`
	Reputation      float64  `json:"reputation"`
	PricePressure   float64  `json:"price_pressure"` // cumulative rival price pressure (1 = none)
}

// Segments is the token allocation between market segments.
type Segments struct {
	Free       float64 `json:"free"`
	Pro        float64 `json:"pro"`
	Enterprise float64 `json:"enterprise"`
}

// Training holds models and the training queue (M3).
type Training struct {
	Models     []Model    `json:"models"`
	Queue      []TrainJob `json:"queue"`
	DataTokens float64    `json:"data_tokens"`
}

// Model is a released model.
type Model struct {
	Name      string  `json:"name"`
	Params    float64 `json:"params"`
	Tokens    float64 `json:"tokens"`
	Benchmark float64 `json:"benchmark"`
	ReleasedS float64 `json:"released_s"`
	Segment   string  `json:"segment"`
}

// TrainJob is a queued or running training.
type TrainJob struct {
	Params       float64  `json:"params"`
	Tokens       float64  `json:"tokens"`
	ClusterShare float64  `json:"cluster_share"`
	Progress     float64  `json:"progress"`
	Arch         []string `json:"arch"`
}

// Researcher is a hired talent (M4).
type Researcher struct {
	Class string `json:"class"`
	Name  string `json:"name"`
	Trait string `json:"trait,omitempty"`
}

// Contract is an enterprise contract (M4).
type Contract struct {
	ID            string  `json:"id"`
	Volume        float64 `json:"volume"`
	Delivered     float64 `json:"delivered"`
	MinBenchmark  float64 `json:"min_benchmark"`
	DeadlineS     float64 `json:"deadline_s"`
	RewardMinutes float64 `json:"reward_minutes"`
}

// Rule is an automation policy (M4). Condition and action are kept as raw
// JSON until the rules engine defines their types.
type Rule struct {
	If      json.RawMessage `json:"if"`
	Then    json.RawMessage `json:"then"`
	Enabled bool            `json:"enabled"`
}

// Prestige holds both prestige layers.
type Prestige struct {
	Params         float64 `json:"params"`
	ParamsLifetime float64 `json:"params_lifetime"`
	Capital        float64 `json:"capital"`
	VersionIndex   int     `json:"version_index"`
	RoundIndex     int     `json:"round_index"`
	ResetsL1       int     `json:"resets_l1"`
	ResetsL2       int     `json:"resets_l2"`
}

// AGI is the final megaproject (M5).
type AGI struct {
	Stage    int     `json:"stage"`
	Invested float64 `json:"invested"`
}

// Streak is the daily grant streak (M6). LastClaim is a date (YYYY-MM-DD)
// supplied by the UI; the simulation never reads the wall clock.
type Streak struct {
	Day       int    `json:"day"`
	LastClaim string `json:"last_claim"`
}

// RandomEvent is the active random event (M2).
type RandomEvent struct {
	ActiveID string  `json:"active_id"`
	EndsAtS  float64 `json:"ends_at_s"`
	NextAtS  float64 `json:"next_at_s"`
}

// Settings are player preferences stored in the save.
type Settings struct {
	Lang       string `json:"lang"`     // "", "pl" or "en"; "" = from LANG
	Notation   string `json:"notation"` // letters | scientific
	Advisor    bool   `json:"advisor"`
	Colorblind bool   `json:"colorblind"`
	Bell       bool   `json:"bell"`
}

// Stats are counters for the statistics tab and achievements.
type Stats struct {
	PlayS        float64 `json:"play_s"`
	OfflineS     float64 `json:"offline_s"`
	Crits        int     `json:"crits"`
	BestCombo    float64 `json:"best_combo"`
	EventsCaught int     `json:"events_caught"`
	Purchases    int     `json:"purchases"`
	ManualStarts int     `json:"manual_starts"`
	BoostPresses int     `json:"boost_presses"`
}

// Combo is the Space boost (DECISIONS 16).
type Combo struct {
	Mult  float64 // 1 = no boost
	IdleS float64 // seconds since the last press
}

// PurchaseKind identifies what the last purchase bought.
type PurchaseKind uint8

const (
	PurchaseNone PurchaseKind = iota
	PurchaseBusiness
	PurchaseManager
	PurchaseUpgrade
)

// Undo remembers the last purchase for the undo window.
type Undo struct {
	Kind     PurchaseKind
	Business int
	Upgrade  int
	Qty      int
	Cost     float64
	AtS      float64 // State.Clock at purchase
}

// NewState starts a new game: start money, free businesses, seeded RNG.
func NewState(b *balance.Balance, seed uint64) State {
	s := State{
		Seed:     seed,
		RNG:      rng.NewStreams(seed),
		Money:    b.Start.Money,
		Cards:    map[string]int{},
		Settings: Settings{Notation: "letters", Advisor: b.UI.AdvisorDefault},
		Market:   Market{Segments: Segments{Pro: 1}, PricePressure: 1, UnlockedRegions: []string{}},
		Combo:    Combo{Mult: 1},
	}
	free := b.FreeOwned()
	for i := range s.Businesses {
		s.Businesses[i].ID = b.Businesses[i].ID
		s.Businesses[i].Owned = free[i]
		s.Businesses[i].DrawBudget = b.BusinessDefaults.DrawCapPerS
	}
	checkUnlocks(&s, b, nil)
	Recompute(&s, b)
	return s
}

// Normalize repairs a loaded state: transient fields, combo, budgets and
// derived values. Call it after loading a save.
func Normalize(s *State, b *balance.Balance) {
	if s.Combo.Mult < 1 {
		s.Combo.Mult = 1
	}
	if s.Cards == nil {
		s.Cards = map[string]int{}
	}
	if s.Market.PricePressure == 0 {
		s.Market.PricePressure = 1
	}
	for i := range s.Businesses {
		s.Businesses[i].ID = b.Businesses[i].ID
		if s.Businesses[i].Owned < 0 {
			s.Businesses[i].Owned = 0
		}
		if s.Businesses[i].Manager {
			s.Businesses[i].Running = false
		}
		// Older saves have no budget (0): it refills within 1/draw_cap_per_s s.
		bz := &s.Businesses[i]
		if !(bz.DrawBudget >= 0) {
			bz.DrawBudget = 0
		}
		bz.DrawBudget = math.Min(bz.DrawBudget, b.BusinessDefaults.DrawCapPerS)
	}
	checkUnlocks(s, b, nil)
	Recompute(s, b)
}
