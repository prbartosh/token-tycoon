package market

import (
	"math"
	"testing"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

func TestFormulas(t *testing.T) {
	m := &balance.Default().Market
	rivals := []float64{50, 60, 70, 80}
	sum := Share(65, rivals, 4)
	for i, r := range rivals {
		others := append([]float64{65}, rivals[:i]...)
		others = append(others, rivals[i+1:]...)
		sum += Share(r, others, 4)
	}
	if math.Abs(sum-1) > 1e-12 {
		t.Fatalf("shares sum to %v", sum)
	}
	if Median(rivals) != 65 || Median([]float64{3, 1, 2}) != 2 {
		t.Fatal("median")
	}
	if q := Quality(100, 0, m); q != m.QualityClamp[1] {
		t.Fatalf("quality clamp %v", q)
	}
	if q := Quality(60, 50, m); math.Abs(q-1.2) > 1e-12 {
		t.Fatalf("quality %v", q)
	}
	if d := Demand(0.25, 0, m); d != 1 {
		t.Fatalf("neutral demand %v", d)
	}
	if d := Demand(1, 100, m); d != m.DemandClamp[1] {
		t.Fatalf("demand clamp %v", d)
	}
	if f := Deflation(2*Day, m); math.Abs(f-0.95*0.95) > 1e-12 {
		t.Fatalf("deflation %v", f)
	}
	if Rank(70, rivals) != 2 {
		t.Fatal("rank")
	}
}

func TestHypeStepIndependent(t *testing.T) {
	h1 := HypeAfter(1, 1e-4, 6*3600, 3600)
	h2 := 1.0
	for range 36000 {
		h2 = HypeAfter(h2, 1e-4, 6*3600, 0.1)
	}
	if math.Abs(h1-h2) > 1e-9 {
		t.Fatalf("hype depends on step: %v vs %v", h1, h2)
	}
	if h := HypeAfter(1, 0, 6*3600, 6*3600); math.Abs(h-0.5) > 1e-12 {
		t.Fatalf("half-life: %v", h)
	}
	if s := SteadyHype(1e-4, 6*3600); math.Abs(HypeAfter(s, 1e-4, 6*3600, 999)-s) > 1e-12 {
		t.Fatal("steady state moves")
	}
}

func TestRivalSchedule(t *testing.T) {
	b := balance.Default()
	m := &b.Market
	for i := range b.Rivals {
		r := &b.Rivals[i]
		prev := Release{K: -1}
		for day := 0.0; day <= 30; day += 0.25 {
			rel := LastRelease(42, i, r, day*Day, m.RivalSpanDays)
			if rel.K < prev.K || rel.AtS > day*Day {
				t.Fatalf("%s: release order broken at day %v", r.ID, day)
			}
			if rel.K != prev.K {
				got, ok := ReleasedIn(42, i, r, (day-0.25)*Day, day*Day, m.RivalSpanDays)
				if !ok || got.K != rel.K || got.Benchmark != rel.Benchmark {
					t.Fatalf("%s: ReleasedIn misses release %d at day %v", r.ID, rel.K, day)
				}
			}
			prev = rel
		}
		last := LastRelease(42, i, r, 40*Day, m.RivalSpanDays)
		if math.Abs(last.Benchmark-r.EndBenchmark) > 7 {
			t.Errorf("%s: day 40 benchmark %v, want about %v", r.ID, last.Benchmark, r.EndBenchmark)
		}
		if LastRelease(42, i, r, 30*Day, m.RivalSpanDays) != LastRelease(42, i, r, 30*Day, m.RivalSpanDays) {
			t.Fatal("not deterministic")
		}
	}
	if LastRelease(1, 2, &b.Rivals[2], 20*Day, m.RivalSpanDays) == LastRelease(2, 2, &b.Rivals[2], 20*Day, m.RivalSpanDays) {
		t.Error("jumpy rival ignores the seed")
	}
}

func TestPressure(t *testing.T) {
	b := balance.Default()
	m := &b.Market
	if p := Pressure(1, b.Rivals, 0, 0.5*Day, m.PressureFloor); p != 1 {
		t.Fatalf("pressure before any release %v", p)
	}
	ow := 1
	n := countReleases(1, ow, &b.Rivals[ow], 0, 4.5*Day)
	if n != 2 {
		t.Fatalf("open-weights releases in 4.5 days: %d", n)
	}
	want := math.Pow(b.Rivals[ow].PricePressure, 2)
	if p := Pressure(1, b.Rivals, 0, 4.5*Day, m.PressureFloor); math.Abs(p-want) > 1e-12 {
		t.Fatalf("pressure %v want %v", p, want)
	}
	if p := Pressure(1, b.Rivals, 0, 1000*Day, m.PressureFloor); p != m.PressureFloor {
		t.Fatalf("pressure floor %v", p)
	}
}
