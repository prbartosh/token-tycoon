package simulate

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseDuration(s string) (float64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) < 2 {
		return 0, fmt.Errorf("bad duration %q", s)
	}
	mult := map[byte]float64{'s': 1, 'm': 60, 'h': 3600, 'd': 86400}[s[len(s)-1]]
	if mult == 0 {
		return 0, fmt.Errorf("bad duration unit in %q", s)
	}
	v, err := strconv.ParseFloat(s[:len(s)-1], 64)
	if err != nil || !(v > 0) {
		return 0, fmt.Errorf("bad duration %q", s)
	}
	return v * mult, nil
}

func ParseSessions(s string) (float64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if i := strings.IndexByte(s, '/'); i >= 0 {
		switch s[i+1:] {
		case "day", "d":
		default:
			return 0, fmt.Errorf("bad sessions %q", s)
		}
		s = s[:i]
	}
	v, err := ParseDuration(s)
	if err != nil || v > 86400 {
		return 0, fmt.Errorf("bad sessions %q", s)
	}
	return v, nil
}
