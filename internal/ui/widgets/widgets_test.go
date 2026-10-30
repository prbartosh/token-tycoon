package widgets

import (
	"strings"
	"testing"

	"token-tycoon/internal/ui/theme"
)

func TestBar(t *testing.T) {
	cases := map[float64]string{0: "    ", 0.5: "██  ", 1: "████", 0.125: "▌   ", 0.0625: "▎   ", 2: "████"}
	for f, want := range cases {
		var sb strings.Builder
		Bar(&sb, f, 4)
		if sb.String() != want {
			t.Errorf("Bar(%v) = %q, want %q", f, sb.String(), want)
		}
	}
}

func TestSparklineAndFit(t *testing.T) {
	var sb strings.Builder
	Sparkline(&sb, []float64{0, 1, 2, 3, 4, 5, 6, 7}, 8)
	if sb.String() != "▁▂▃▄▅▆▇█" {
		t.Fatal(sb.String())
	}
	sb.Reset()
	Sparkline(&sb, []float64{1, 1}, 5)
	if Width(sb.String()) != 5 {
		t.Fatal("sparkline width")
	}
	if Fit("abcdef", 4) != "abc…" || Fit("ab", 4) != "ab  " || FitRight("ab", 4) != "  ab" {
		t.Fatal("fit")
	}
	sb.Reset()
	Flow(&sb, 6, 2, []theme.Paint{{}, {}})
	if Width(sb.String()) != 6 {
		t.Fatal("flow width")
	}
}
