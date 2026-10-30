// Package sim is the pure game simulation: state, commands, Step and offline
// catch-up. It does no I/O, never reads the wall clock and never imports UI
// code. The same engine drives the game, offline catch-up and the simulator.
package sim

import (
	"encoding/json"
	"math"
	"math/bits"

	"token-tycoon/internal/balance"
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

	Businesses [N]Business `json:"-"`
	Upgrades   Bitset      `json:"-"`
	Unlocks    Bitset      `json:"-"`

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

	Combo   Combo   `json:"-"`
	Undo    Undo    `json:"-"`
	Derived Derived `json:"-"`
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
	Segments        Segments `json:"segments"`
	Hype            float64  `json:"hype"`
	Reputation      float64  `json:"reputation"`
	PricePressure   float64  `json:"price_pressure"`
}

type Segments struct {
	Free       float64 `json:"free"`
	Pro        float64 `json:"pro"`
	Enterprise float64 `json:"enterprise"`
}

type Training struct {
	Models     []Model    `json:"models"`
	Queue      []TrainJob `json:"queue"`
	DataTokens float64    `json:"data_tokens"`
}

type Model struct {
	Name      string  `json:"name"`
	Params    float64 `json:"params"`
	Tokens    float64 `json:"tokens"`
	Benchmark float64 `json:"benchmark"`
	ReleasedS float64 `json:"released_s"`
	Segment   string  `json:"segment"`
}

type TrainJob struct {
	Params       float64  `json:"params"`
	Tokens       float64  `json:"tokens"`
	ClusterShare float64  `json:"cluster_share"`
	Progress     float64  `json:"progress"`
	Arch         []string `json:"arch"`
}

type Researcher struct {
	Class string `json:"class"`
	Name  string `json:"name"`
	Trait string `json:"trait,omitempty"`
}

type Contract struct {
	ID            string  `json:"id"`
	Volume        float64 `json:"volume"`
	Delivered     float64 `json:"delivered"`
	MinBenchmark  float64 `json:"min_benchmark"`
	DeadlineS     float64 `json:"deadline_s"`
	RewardMinutes float64 `json:"reward_minutes"`
}

type Rule struct {
	If      json.RawMessage `json:"if"`
	Then    json.RawMessage `json:"then"`
	Enabled bool            `json:"enabled"`
}

type Prestige struct {
	Params         float64 `json:"params"`
	ParamsLifetime float64 `json:"params_lifetime"`
	Capital        float64 `json:"capital"`
	VersionIndex   int     `json:"version_index"`
	RoundIndex     int     `json:"round_index"`
	ResetsL1       int     `json:"resets_l1"`
	ResetsL2       int     `json:"resets_l2"`
}

type AGI struct {
	Stage    int     `json:"stage"`
	Invested float64 `json:"invested"`
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
	PlayS        float64 `json:"play_s"`
	OfflineS     float64 `json:"offline_s"`
	Crits        int     `json:"crits"`
	BestCombo    float64 `json:"best_combo"`
	EventsCaught int     `json:"events_caught"`
	Purchases    int     `json:"purchases"`
	ManualStarts int     `json:"manual_starts"`
	BoostPresses int     `json:"boost_presses"`
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
)

type Undo struct {
	Kind     PurchaseKind
	Business int
	Upgrade  int
	Qty      int
	Cost     float64
	AtS      float64
}

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
		bz := &s.Businesses[i]
		if !(bz.DrawBudget >= 0) {
			bz.DrawBudget = 0
		}
		bz.DrawBudget = math.Min(bz.DrawBudget, b.BusinessDefaults.DrawCapPerS)
	}
	checkUnlocks(s, b, nil)
	Recompute(s, b)
}
