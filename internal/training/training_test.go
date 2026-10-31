package training

import (
	"math"
	"testing"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

func TestDesignExamples(t *testing.T) {
	tr := &balance.Default().Training
	cases := []struct{ n, d, want, tol float64 }{
		{1e7, 2e8, 13, 1.5},
		{1e9, 2e10, 60, 2},
		{1e12, 2e13, 94, 1},
	}
	for _, c := range cases {
		if got := Benchmark(c.n, c.d, 1, tr); math.Abs(got-c.want) > c.tol {
			t.Errorf("N=%g D=%g: benchmark %.2f, want about %v", c.n, c.d, got, c.want)
		}
	}
	if FLOPs(1e9, 2e10, tr) != 1.2e20 {
		t.Error("FLOPs = 6ND")
	}
	if Benchmark(1e9, 2e10, 0.8, tr) <= Benchmark(1e9, 2e10, 1, tr) {
		t.Error("architecture bonus must raise the benchmark")
	}
}

func TestOptimalTokensIsOptimal(t *testing.T) {
	tr := &balance.Default().Training
	for _, n := range []float64{1e7, 1e9, 1e11} {
		d := OptimalTokens(n, tr)
		c := FLOPs(n, d, tr)
		best := Benchmark(n, d, 1, tr)
		for _, k := range []float64{0.5, 0.8, 1.25, 2} {
			n2 := n * k
			if b := Benchmark(n2, c/(tr.FlopsPerParamToken*n2), 1, tr); b > best+1e-9 {
				t.Errorf("N=%g: split x%v beats the optimum (%v > %v)", n, k, b, best)
			}
		}
	}
}

func TestBestSplitLimits(t *testing.T) {
	tr := &balance.Default().Training
	n, d, b := BestSplit(1e22, 1e9, 1e10, 1, tr)
	if n > 1e9*(1+1e-9) || d > 1e10 || b <= 0 {
		t.Fatalf("limits ignored: N=%g D=%g", n, d)
	}
	if n2, _, b2 := BestSplit(1e22, 1e12, 1e15, 1, tr); n2 <= n || b2 <= b {
		t.Fatal("more room must not give a worse model")
	}
	if _, _, b := BestSplit(0, 1e9, 1e10, 1, tr); b != 0 {
		t.Fatal("zero budget")
	}
}

func TestLevelsAndSynthetic(t *testing.T) {
	tr := &balance.Default().Training
	if ParamsLevel(tr, 0) != 0 || ParamsLevel(tr, tr.MaxParamsUnlock[2]) != 2 || ParamsLevel(tr, 1e300) != len(tr.MaxParamsLevels)-1 {
		t.Fatal("params level")
	}
	if MaxParams(tr, 99) != tr.MaxParamsLevels[len(tr.MaxParamsLevels)-1] {
		t.Fatal("max params clamp")
	}
	if SyntheticMult(0.5, tr) != 1 || SyntheticMult(1, tr) != 1+tr.SyntheticPenalty {
		t.Fatal("synthetic penalty")
	}
	if m := SyntheticMult(0.75, tr); math.Abs(m-(1+tr.SyntheticPenalty/2)) > 1e-12 {
		t.Fatalf("synthetic penalty at 75%%: %v", m)
	}
}
