package simulate

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/i18n"
	"github.com/prbartosh/token-tycoon/internal/sim"
)

func TestParse(t *testing.T) {
	cases := map[string]float64{"90s": 90, "10m": 600, "1h": 3600, "1d": 86400, "1.5d": 129600, "30D": 30 * 86400}
	for in, want := range cases {
		if got, err := ParseDuration(in); err != nil || got != want {
			t.Errorf("ParseDuration(%q) = %v, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "h", "10", "-1h", "1w", "xh"} {
		if _, err := ParseDuration(bad); err == nil {
			t.Errorf("ParseDuration(%q) should fail", bad)
		}
	}
	if v, err := ParseSessions("10m/day"); err != nil || v != 600 {
		t.Errorf("ParseSessions = %v, %v", v, err)
	}
	for _, bad := range []string{"10m/week", "25h/day"} {
		if _, err := ParseSessions(bad); err == nil {
			t.Errorf("ParseSessions(%q) should fail", bad)
		}
	}
}

func TestRunDeterministicAndReport(t *testing.T) {
	b := balance.Default()
	cfg := Config{Duration: 900, Strategy: Greedy, Seed: 5}
	r1, err := Run(b, cfg)
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := Run(b, cfg)
	if !reflect.DeepEqual(r1.Final, r2.Final) || len(r1.Purchases) != len(r2.Purchases) {
		t.Fatal("simulator is not deterministic")
	}
	if r1.FirstManagerT < 0 || r1.Final.Lifetime.Total <= 0 || len(r1.Samples) == 0 {
		t.Fatalf("greedy bot made no progress: %+v", r1.Final.Stats)
	}
	var out bytes.Buffer
	Report(&out, r1, b, i18n.New(i18n.EN))
	if !strings.Contains(out.String(), "Acceptance criteria") {
		t.Fatal(out.String())
	}
	var csvb bytes.Buffer
	if err := WriteCSV(&csvb, r1, b); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(csvb.String()), "\n")
	if !strings.HasPrefix(lines[0], "t_s,income_per_s,money,params,capital,benchmark,share,unlocks") || len(lines) != len(r1.Samples)+1 {
		t.Fatalf("csv header %q, %d lines", lines[0], len(lines))
	}
}

func TestStrategies(t *testing.T) {
	b := balance.Default()
	for _, st := range []string{Passive, Random} {
		r, err := Run(b, Config{Duration: 2 * 86400, Strategy: st, Seed: 1, SessionS: 300})
		if err != nil {
			t.Fatal(err)
		}
		if r.Final.Lifetime.Total <= 0 {
			t.Fatalf("%s made no money", st)
		}
		if st == Passive && (len(r.Sessions) != 2 || r.OfflineS != 2*b.Offline.CapHoursStart*3600) {
			t.Fatalf("passive sessions %d offline %v", len(r.Sessions), r.OfflineS)
		}
	}
	r, err := Run(b, Config{Duration: 120, Strategy: Greedy, Seed: 1, ComboRate: 4})
	if err != nil || r.Final.Stats.BestCombo != b.Combo.MaxMult {
		t.Fatalf("combo bot: %v best %v", err, r.Final.Stats.BestCombo)
	}
	if _, err := Run(b, Config{Duration: 1, Strategy: "nope"}); err == nil {
		t.Fatal("unknown strategy accepted")
	}
}

func TestGreedyUsesM2Systems(t *testing.T) {
	b := balance.Default()
	r, err := Run(b, Config{Duration: 2 * 3600, Strategy: Greedy, Seed: 42, Continuous: true})
	if err != nil {
		t.Fatal(err)
	}
	s := &r.Final
	if r.InfraT < 0 || !s.Infra.StarterGiven || len(r.Prestiges) == 0 || s.Prestige.Params <= 0 {
		t.Fatalf("infra %v prestiges %d params %v", r.InfraT, len(r.Prestiges), s.Prestige.Params)
	}
	if r.Racks == 0 || s.Stats.EventsSeen == 0 || r.Reactions == 0 || s.Stats.Grants != 1 || s.Stats.CardsDropped == 0 {
		t.Fatalf("racks %d events %d reactions %d grants %d cards %d", r.Racks, s.Stats.EventsSeen, r.Reactions, s.Stats.Grants, s.Stats.CardsDropped)
	}
	if r.ThrottleS/r.InfraS < 0.9 {
		t.Fatalf("bot leaves production throttled: %v", r.ThrottleS/r.InfraS)
	}
	var out bytes.Buffer
	Report(&out, r, b, i18n.New(i18n.PL))
	if strings.Contains(out.String(), "!report") {
		t.Fatal(out.String())
	}
}

func TestGreedyUsesM3Systems(t *testing.T) {
	b := balance.Default()
	r, err := Run(b, Config{Duration: 6 * 3600, Strategy: Greedy, Seed: 42, Continuous: true})
	if err != nil {
		t.Fatal(err)
	}
	s := &r.Final
	if r.MarketT < 0 || r.TrainingT < 0 || s.Stats.Models == 0 || len(r.Models) == 0 || r.Models[0].T < 0 {
		t.Fatalf("market %v training %v models %d", r.MarketT, r.TrainingT, s.Stats.Models)
	}
	if s.Training.Scrapers == 0 || r.TrainS == 0 || !(s.Derived.Market.Benchmark > b.Market.BaseBenchmark) {
		t.Fatalf("scrapers %d train %v bench %v", s.Training.Scrapers, r.TrainS, s.Derived.Market.Benchmark)
	}
	last := r.Samples[len(r.Samples)-1]
	if !(last.Benchmark > 0) || !(last.Share > 0) {
		t.Fatalf("samples miss market data: %+v", last)
	}
	var out bytes.Buffer
	Report(&out, r, b, i18n.New(i18n.EN))
	if strings.Contains(out.String(), "!report") || !strings.Contains(out.String(), "TT-") {
		t.Fatal(out.String())
	}
}

func TestGreedyUsesM4Systems(t *testing.T) {
	over := `
[talent]
unlock_lifetime = 1e13
[contracts]
unlock_lifetime = 1e13
[prestige.layer2]
unlock_lifetime = 1e16
capital_log_offset = 15
[research]
unlock_lifetime = 1e16
[rules]
unlock_lifetime = 1e16
`
	b, err := balance.Parse(balance.DefaultTOML(), []byte(over))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Run(b, Config{Duration: 86400, Strategy: Greedy, Seed: 42, Continuous: true})
	if err != nil {
		t.Fatal(err)
	}
	s := &r.Final
	if s.Stats.Hired == 0 || s.Stats.ContractsDone+s.Stats.ContractsFailed == 0 {
		t.Fatalf("talent %d contracts %d/%d", s.Stats.Hired, s.Stats.ContractsDone, s.Stats.ContractsFailed)
	}
	if len(r.Rounds) == 0 || s.Prestige.ResetsL2 != len(r.Rounds) || s.Stats.ResearchDone == 0 {
		t.Fatalf("rounds %d research %d", len(r.Rounds), s.Stats.ResearchDone)
	}
	if len(s.Rules) == 0 || s.Stats.RulesFired == 0 {
		t.Fatalf("rules %d fired %d", len(s.Rules), s.Stats.RulesFired)
	}
	var out bytes.Buffer
	Report(&out, r, b, i18n.New(i18n.PL))
	if strings.Contains(out.String(), "!report") || !strings.Contains(out.String(), "Rundy finansowania") {
		t.Fatal(out.String())
	}
	t.Logf("hired %d poached %d kept %d contracts %d/%d rounds %d research %d rules fired %d",
		s.Stats.Hired, s.Stats.Poached, s.Stats.Retained, s.Stats.ContractsDone, s.Stats.ContractsFailed,
		len(r.Rounds), s.Stats.ResearchDone, s.Stats.RulesFired)
}

func TestGreedyFinishesGame(t *testing.T) {
	over := `
[market]
regions_unlock_lifetime = 1e14
[[region]]
id = "eu"
entry_cost_minutes = 15
[[region]]
id = "asia"
entry_cost_minutes = 20
[investors]
unlock_lifetime = 1e13
[safety]
incident_every_h_base = 0.5
[agi]
unlock_lifetime = 1e30
si_every_oom = 1
[[agi_stage]]
id = "alignment"
cost = 1e31
[[agi_stage]]
id = "compute"
cost = 1e32
[[agi_stage]]
id = "data"
cost = 1e33
[[agi_stage]]
id = "training"
cost = 1e34
[[agi_stage]]
id = "evaluation"
cost = 1e35
`
	b, err := balance.Parse(balance.DefaultTOML(), []byte(over))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Run(b, Config{Duration: 2 * 86400, Strategy: Greedy, Seed: 42, Continuous: true})
	if err != nil {
		t.Fatal(err)
	}
	s := &r.Final
	if r.EndingT < 0 || len(r.AGIStages) != len(b.AGIStages) || !sim.AGIEnded(s, b) {
		t.Fatalf("ending not reached: stages %v lifetime %v", r.AGIStages, s.Lifetime.Total)
	}
	if sim.RegionsEntered(s, b) < 2 || s.Stats.IncidentsOK == 0 || s.Stats.Quarters == 0 || s.AGI.SILevel == 0 {
		t.Fatalf("regions %d incidents %d/%d quarters %d si %d agi %v stages %v", sim.RegionsEntered(s, b), s.Stats.IncidentsOK, s.Stats.IncidentsSeen, s.Stats.Quarters, s.AGI.SILevel, r.AGIT, r.AGIStages)
	}
	var out bytes.Buffer
	Report(&out, r, b, i18n.New(i18n.EN))
	if strings.Contains(out.String(), "!report") || !strings.Contains(out.String(), "Ending reached on day") {
		t.Fatal(out.String())
	}
	t.Logf("ending at %.1f h, regions %d, incidents %d/%d, quarters %d/%d, achievements %d, SI %d",
		r.EndingT/3600, sim.RegionsEntered(s, b), s.Stats.IncidentsOK, s.Stats.IncidentsSeen,
		s.Stats.QuartersBeaten, s.Stats.Quarters, s.Achieved.Count(), s.AGI.SILevel)
}

func TestChecksPacing(t *testing.T) {
	b := balance.Default()
	c := i18n.New(i18n.EN)
	day := 86400.0
	r := &Result{
		Config:       Config{Duration: 30 * day, Strategy: Greedy},
		ManagerPlayS: 45, InfraPlayS: 25 * 60, PrestigePlayS: 60 * 60, InfraT: 1500, PrestigeT: 3600,
		Layer2T: 6.2 * day, AGIT: 18.5 * day, EndingT: 28.3 * day, LongestFlat: 60,
		Sessions: []Session{{StartT: 0, FirstUnlockT: -1}, {StartT: day, FirstUnlockT: 10}},
	}
	r.Final.Stats.PlayS = 30 * 3600
	log := []float64{9, 13, 17, 24, 29, 34, 39, 44, 49, 54, 58, 62, 64, 70, 76, 82, 88, 92, 96, 100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140}
	for d, l := range log {
		r.DayIncome = append(r.DayIncome, math.Pow(10, l))
		r.Goals = append(r.Goals, GoalAt{T: float64(d)*day + 100})
	}
	fails := func() []string {
		var out []string
		for _, ch := range Checks(r, b, c) {
			if !ch.Pass && !ch.NA {
				out = append(out, ch.Text)
			}
		}
		return out
	}
	if f := fails(); len(f) != 0 {
		t.Fatalf("expected all criteria to pass: %q", f)
	}
	r.DayIncome[2] = 1e25
	r.Goals = r.Goals[:5]
	r.EndingT = 20 * day
	r.Sessions[1].FirstUnlockT = 400
	if f := fails(); len(f) != 4 {
		t.Fatalf("expected bands, goals, ending and session failures: %q", f)
	}
	r.Config.Strategy = Passive
	r.EndingT = 45 * day
	r.Sessions[1].FirstUnlockT = 0
	if f := fails(); len(f) != 0 {
		t.Fatalf("passive bot is judged only by its own criteria: %q", f)
	}
}
