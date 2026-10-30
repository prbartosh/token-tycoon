// Package format renders game numbers and durations for the UI.
package format

import (
	"math"
	"strconv"
	"strings"
)

type Notation uint8

const (
	Letters Notation = iota
	Scientific
)

func ParseNotation(s string) Notation {
	if s == "scientific" || s == "sci" {
		return Scientific
	}
	return Letters
}

func (n Notation) String() string {
	if n == Scientific {
		return "scientific"
	}
	return "letters"
}

var named = [...]string{"", "K", "M", "B", "T"}

func Suffix(group int) string {
	if group < len(named) {
		return named[group]
	}
	i := group - len(named)
	if i < 26*26 {
		return string([]byte{byte('a' + i/26), byte('a' + i%26)})
	}
	return "e" + strconv.Itoa(group*3)
}

func Number(v float64, n Notation) string {
	return string(AppendNumber(nil, v, n))
}

func AppendNumber(dst []byte, v float64, n Notation) []byte {
	switch {
	case math.IsNaN(v):
		return append(dst, "NaN"...)
	case math.IsInf(v, 1):
		return append(dst, "inf"...)
	case math.IsInf(v, -1):
		return append(dst, "-inf"...)
	}
	if v < 0 {
		dst = append(dst, '-')
		v = -v
	}
	if v < 1000 {
		return appendSmall(dst, v)
	}
	var buf [32]byte
	s := strconv.AppendFloat(buf[:0], v, 'e', 2, 64)
	mant, exp := splitE(s)
	if n == Scientific {
		dst = append(dst, mant...)
		dst = append(dst, 'e')
		return strconv.AppendInt(dst, int64(exp), 10)
	}
	group := exp / 3
	intDigits := exp%3 + 1
	digits := [3]byte{mant[0], mant[2], mant[3]}
	dst = append(dst, digits[:intDigits]...)
	frac := digits[intDigits:]
	for len(frac) > 0 && frac[len(frac)-1] == '0' {
		frac = frac[:len(frac)-1]
	}
	if len(frac) > 0 {
		dst = append(dst, '.')
		dst = append(dst, frac...)
	}
	return append(dst, Suffix(group)...)
}

func appendSmall(dst []byte, v float64) []byte {
	if v == math.Trunc(v) {
		return strconv.AppendFloat(dst, v, 'f', 0, 64)
	}
	prec := 2
	if v >= 100 {
		prec = 0
	} else if v >= 10 {
		prec = 1
	}
	r := math.Pow(10, float64(prec))
	rv := math.Round(v*r) / r
	if rv >= 1000 {
		return append(dst, "1K"...)
	}
	dst = strconv.AppendFloat(dst, rv, 'f', prec, 64)
	if prec > 0 {
		for dst[len(dst)-1] == '0' {
			dst = dst[:len(dst)-1]
		}
		if dst[len(dst)-1] == '.' {
			dst = dst[:len(dst)-1]
		}
	}
	return dst
}

func splitE(s []byte) ([]byte, int) {
	i := 0
	for i < len(s) && s[i] != 'e' {
		i++
	}
	exp, _ := strconv.Atoi(string(s[i+1:]))
	return s[:i], exp
}

func Money(v float64, n Notation) string { return "$ " + Number(v, n) }

func Rate(v float64, n Notation) string {
	var b strings.Builder
	if v >= 0 {
		b.WriteByte('+')
	}
	b.WriteString(Number(v, n))
	b.WriteString("/s")
	return b.String()
}

func Duration(sec float64) string {
	if math.IsInf(sec, 0) || sec > 1e15 {
		return "--"
	}
	if !(sec > 0) {
		return "0s"
	}
	if sec < 10 {
		s := strconv.FormatFloat(math.Round(sec*10)/10, 'f', -1, 64)
		return s + "s"
	}
	t := int64(math.Round(sec))
	d, t := t/86400, t%86400
	h, t := t/3600, t%3600
	m, s := t/60, t%60
	switch {
	case d > 0:
		return strconv.FormatInt(d, 10) + "d" + pad2(h) + "h"
	case h > 0:
		return strconv.FormatInt(h, 10) + "h" + pad2(m) + "m"
	case m > 0:
		return strconv.FormatInt(m, 10) + "m" + pad2(s) + "s"
	default:
		return strconv.FormatInt(s, 10) + "s"
	}
}

func pad2(x int64) string {
	if x < 10 {
		return "0" + strconv.FormatInt(x, 10)
	}
	return strconv.FormatInt(x, 10)
}
