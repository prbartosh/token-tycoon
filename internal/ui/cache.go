package ui

import (
	"math"

	"github.com/prbartosh/token-tycoon/internal/format"
)

const numSlots = 512

type numEntry struct {
	v     float64
	n     format.Notation
	money bool
	ok    bool
	s     string
}

type numCache [numSlots]numEntry

func (c *numCache) get(v float64, n format.Notation, money bool) string {
	h := math.Float64bits(v) * 0x9E3779B97F4A7C15
	h ^= uint64(n)<<1 | b2u(money)
	e := &c[(h>>40)%numSlots]
	if e.ok && e.v == v && e.n == n && e.money == money {
		return e.s
	}
	s := ""
	if money {
		s = format.Money(v, n)
	} else {
		s = format.Number(v, n)
	}
	*e = numEntry{v: v, n: n, money: money, ok: true, s: s}
	return s
}

func b2u(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

type logCache struct {
	seq, w, h int
	lines     []string
}

type tabBarCache struct {
	key tabBarKey
	ok  bool
	s   string
}
