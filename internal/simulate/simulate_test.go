package simulate

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"token-tycoon/internal/balance"
	"token-tycoon/internal/i18n"
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
		if st == Passive && (len(r.Sessions) != 2 || r.OfflineS != b.Offline.CapHoursStart*3600) {
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
