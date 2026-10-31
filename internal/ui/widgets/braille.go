package widgets

import (
	"math"
	"strings"
)

var brailleBit = [4][2]uint8{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

type Plot struct {
	cells []uint8
	w, h  int
}

func (p *Plot) Reset(w, h int) {
	w, h = max(0, w), max(0, h)
	n := w * h
	if cap(p.cells) < n {
		p.cells = make([]uint8, n)
	}
	p.cells = p.cells[:n]
	clear(p.cells)
	p.w, p.h = w, h
}

func (p *Plot) Width() int  { return p.w }
func (p *Plot) Height() int { return p.h }

func (p *Plot) Set(x, y int) {
	if x < 0 || y < 0 || x >= 2*p.w || y >= 4*p.h {
		return
	}
	p.cells[(y/4)*p.w+x/2] |= brailleBit[y%4][x%2]
}

func (p *Plot) Line(x0, y0, x1, y1 int) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	e := dx + dy
	for {
		p.Set(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (p *Plot) Series(vals []float64, lo, hi float64, slots int) {
	if p.w == 0 || p.h == 0 || len(vals) == 0 {
		return
	}
	slots = max(slots, len(vals))
	xs, ys := 2*p.w-1, 4*p.h-1
	span := hi - lo
	px, py := -1, -1
	for i, v := range vals {
		x := 0
		if slots > 1 {
			x = int(math.Round(float64(i) * float64(xs) / float64(slots-1)))
		}
		f := 0.5
		if span > 0 {
			f = (v - lo) / span
		}
		if !(f >= 0) {
			f = 0
		}
		y := ys - int(math.Round(math.Min(1, f)*float64(ys)))
		if px >= 0 {
			p.Line(px, py, x, y)
		} else {
			p.Set(x, y)
		}
		px, py = x, y
	}
}

func (p *Plot) Row(sb *strings.Builder, r int) {
	if r < 0 || r >= p.h {
		return
	}
	for _, c := range p.cells[r*p.w : (r+1)*p.w] {
		if c == 0 {
			sb.WriteByte(' ')
			continue
		}
		sb.WriteRune(rune(0x2800 + int(c)))
	}
}
