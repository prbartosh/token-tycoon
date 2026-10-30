package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"token-tycoon/internal/balance"
)

func TestProfiles(t *testing.T) {
	c := balance.Default().UI.Colors
	for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI, colorprofile.ASCII} {
		th := New(c, p)
		out := th.Tier[9].Render("x") + th.Flow[0][3].Render("x")
		want := map[colorprofile.Profile]string{colorprofile.TrueColor: "38;2;", colorprofile.ANSI256: "38;5;", colorprofile.ANSI: "\x1b["}[p]
		switch {
		case p == colorprofile.ASCII && out != "xx":
			t.Errorf("ascii output styled: %q", out)
		case want != "" && !strings.Contains(out, want):
			t.Errorf("%v: %q lacks %q", p, out, want)
		case p != colorprofile.TrueColor && strings.Contains(out, "38;2;"):
			t.Errorf("%v emitted truecolor: %q", p, out)
		}
		if (th.BG != nil) != (p >= colorprofile.ANSI256) {
			t.Errorf("%v: background", p)
		}
	}
}
