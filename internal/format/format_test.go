package format

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

// goldenValues covers 1 to 1e300: every power of ten plus a few mantissas.
func goldenValues() []float64 {
	var vs []float64
	for e := 0; e <= 300; e++ {
		p := math.Pow(10, float64(e))
		for _, m := range []float64{1, 1.5, 2.345, 9.999} {
			vs = append(vs, m*p)
		}
	}
	return vs
}

func TestGolden(t *testing.T) {
	for _, nt := range []Notation{Letters, Scientific} {
		var b strings.Builder
		for _, v := range goldenValues() {
			fmt.Fprintf(&b, "%-24.6g %s\n", v, Number(v, nt))
		}
		path := filepath.Join("testdata", "numbers_"+nt.String()+".golden")
		if *update {
			if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test ./internal/format -update)", err)
		}
		if string(want) != b.String() {
			t.Errorf("%s mismatch; run with -update and review the diff", path)
		}
	}
}

func TestNumber(t *testing.T) {
	cases := []struct {
		v    float64
		n    Notation
		want string
	}{
		{0, Letters, "0"},
		{1, Letters, "1"},
		{0.5, Letters, "0.5"},
		{12.345, Letters, "12.3"},
		{999, Letters, "999"},
		{999.6, Letters, "1K"},
		{1000, Letters, "1K"},
		{1234, Letters, "1.23K"},
		{12345, Letters, "12.3K"},
		{123456, Letters, "123K"},
		{999500, Letters, "1M"},
		{1.28e9, Letters, "1.28B"},
		{1e12, Letters, "1T"},
		{1e15, Letters, "1aa"},
		{4.4e21, Letters, "4.4ac"},
		{1e90, Letters, "1az"},
		{1e93, Letters, "1ba"},
		{-1500, Letters, "-1.5K"},
		{1.5e15, Scientific, "1.50e15"},
		{999, Scientific, "999"},
		{math.Inf(1), Letters, "inf"},
	}
	for _, c := range cases {
		if got := Number(c.v, c.n); got != c.want {
			t.Errorf("Number(%g, %v) = %q, want %q", c.v, c.n, got, c.want)
		}
	}
}

func TestSuffixMonotonic(t *testing.T) {
	seen := map[string]bool{}
	for g := 0; g < 103; g++ {
		s := Suffix(g)
		if seen[s] {
			t.Fatalf("duplicate suffix %q at group %d", s, g)
		}
		seen[s] = true
	}
}

func TestDuration(t *testing.T) {
	cases := []struct {
		s    float64
		want string
	}{
		{0, "0s"}, {-3, "0s"}, {0.25, "0.3s"}, {0.2, "0.2s"}, {9.94, "9.9s"}, {45, "45s"},
		{134, "2m14s"}, {3900, "1h05m"}, {273600, "3d04h"}, {math.Inf(1), "--"},
	}
	for _, c := range cases {
		if got := Duration(c.s); got != c.want {
			t.Errorf("Duration(%v) = %q, want %q", c.s, got, c.want)
		}
	}
}

func BenchmarkNumber(b *testing.B) {
	buf := make([]byte, 0, 32)
	for i := 0; b.Loop(); i++ {
		buf = AppendNumber(buf[:0], 1.2345e45*float64(i%7+1), Letters)
	}
}
