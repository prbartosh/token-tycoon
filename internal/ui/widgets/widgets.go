// Package widgets draws the small text widgets of the UI: the 8-step cycle
// bar, the flowing gradient bar for continuous mode, sparklines and
// width-exact cells.
package widgets

import (
	"math"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"token-tycoon/internal/ui/theme"
)

var eighths = [8]string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

var sparks = [8]string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

const (
	Full  = "█"
	Shade = "░"
)

func Bar(sb *strings.Builder, frac float64, width int) {
	if width <= 0 {
		return
	}
	if !(frac > 0) {
		frac = 0
	}
	frac = math.Min(frac, 1)
	total := int(frac * float64(width) * 8)
	full, rem := total/8, total%8
	for range full {
		sb.WriteString(Full)
	}
	n := full
	if rem > 0 && n < width {
		sb.WriteString(eighths[rem-1])
		n++
	}
	for ; n < width; n++ {
		sb.WriteByte(' ')
	}
}

func Track(sb *strings.Builder, frac float64, width int, fill, track theme.Paint) {
	if width <= 0 {
		return
	}
	frac = math.Max(0, math.Min(frac, 1))
	full := int(math.Round(frac * float64(width)))
	if full > 0 {
		sb.WriteString(fill.Render(strings.Repeat(Full, full)))
	}
	if full < width {
		sb.WriteString(track.Render(strings.Repeat(Shade, width-full)))
	}
}

func Flow(sb *strings.Builder, width, phase int, styles []theme.Paint) {
	n := len(styles)
	if width <= 0 || n == 0 {
		return
	}
	for i := range width {
		k := ((i-phase)%n + n) % n
		sb.WriteString(styles[k].Render(Full))
	}
}

func Sparkline(sb *strings.Builder, vals []float64, width int) {
	if width <= 0 {
		return
	}
	if len(vals) == 0 {
		sb.WriteString(strings.Repeat(" ", width))
		return
	}
	per := max(1, int(math.Ceil(float64(len(vals))/float64(width))))
	cells := min(width, (len(vals)+per-1)/per)
	lo, hi := math.Inf(1), math.Inf(-1)
	var buf [64]float64
	agg := buf[:0]
	start := len(vals) - cells*per
	for c := range cells {
		sum, cnt := 0.0, 0
		for j := start + c*per; j < start+(c+1)*per; j++ {
			if j >= 0 {
				sum += vals[j]
				cnt++
			}
		}
		v := 0.0
		if cnt > 0 {
			v = sum / float64(cnt)
		}
		agg = append(agg, v)
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	for range width - cells {
		sb.WriteByte(' ')
	}
	for _, v := range agg {
		k := 0
		if hi > lo {
			k = int((v - lo) / (hi - lo) * 7.999)
		} else if hi > 0 {
			k = 3
		}
		sb.WriteString(sparks[max(0, min(7, k))])
	}
}

func Fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := ansi.StringWidth(s)
	switch {
	case w > width:
		s = ansi.Truncate(s, width, "…")
		w = ansi.StringWidth(s)
	case w == width:
		return s
	}
	return s + strings.Repeat(" ", max(0, width-w))
}

func FitRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w >= width {
		return ansi.Truncate(s, width, "")
	}
	return strings.Repeat(" ", width-w) + s
}

func Center(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return Fit(s, width)
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}

func Width(s string) int { return ansi.StringWidth(s) }
