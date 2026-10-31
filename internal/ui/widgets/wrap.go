package widgets

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const sgrReset = "\x1b[m"

func Wrap(s string, width int) []string {
	return WrapHang(s, width, 0)
}

func WrapHang(s string, width, indent int) []string {
	width = max(1, width)
	indent = max(0, min(indent, width-1))
	if !strings.Contains(s, "\n") && ansi.StringWidth(s) <= width {
		return []string{s}
	}
	raw := strings.Split(ansi.Wrap(s, width, ""), "\n")
	if indent == 0 || len(raw) == 1 {
		return carryStyles(raw, "")
	}
	first := carryStyles(raw[:1], "")
	rest := sgrState("", raw[0]) + strings.Join(raw[1:], " ")
	pad := strings.Repeat(" ", indent)
	tail := carryStyles(strings.Split(ansi.Wrap(rest, width-indent, ""), "\n"), "")
	out := make([]string, 0, 1+len(tail))
	out = append(out, first...)
	for _, l := range tail {
		out = append(out, pad+l)
	}
	return out
}

func carryStyles(parts []string, active string) []string {
	for i, p := range parts {
		next := sgrState(active, p)
		if active != "" {
			p = active + p
		}
		if next != "" {
			p += sgrReset
		}
		parts[i] = p
		active = next
	}
	return parts
}

func sgrState(active, s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b || i+1 >= len(s) || s[i+1] != '[' {
			continue
		}
		j := i + 2
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
			j++
		}
		if j >= len(s) {
			break
		}
		if s[j] == 'm' {
			params := s[i+2 : j]
			if params == "" || params == "0" {
				active = ""
			} else {
				active += s[i : j+1]
			}
		}
		i = j
	}
	return active
}
