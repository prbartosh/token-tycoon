// Package theme turns the balance [ui.colors] palette into Lip Gloss styles,
// once at startup, downsampled to the terminal's color profile.
package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"token-tycoon/internal/balance"
)

const (
	Tiers     = balance.MaxBusinesses
	FlowSteps = 16
	FadeSteps = 6
)

type Paint struct{ pre, post string }

func (p Paint) Render(s string) string {
	if p.pre == "" && p.post == "" {
		return s
	}
	return p.pre + s + p.post
}

func paintOf(s lipgloss.Style) Paint {
	r := s.Render("X")
	i := strings.IndexByte(r, 'X')
	if i < 0 {
		return Paint{}
	}
	return Paint{pre: r[:i], post: r[i+1:]}
}

type Theme struct {
	Profile colorprofile.Profile
	Plain   bool
	BG      color.Color

	Text, Dim, Bold, Title           Paint
	Money, Rate, Bad, Good           Paint
	Gold, Magenta, Params, Rare      Paint
	Border, Marker, Star, Locked     Paint
	TabActive, TabIdle, TabLocked    Paint
	Badge, Key, KeyText, Header      Paint
	FlashPurchase, FlashMilestone    Paint
	FlashCrit, FlashWarn, Selected   Paint
	ToastInfo, ToastGold, ToastMagic Paint
	ToastWarn, Panel, Modal          Paint
	Track                            Paint

	Tier     [Tiers]Paint
	Flow     [Tiers][FlowSteps]Paint
	Fade     [FadeSteps]Paint
	GoldFade [FadeSteps]Paint
}

type palette struct {
	p     colorprofile.Profile
	plain bool
}

func (pl palette) c(hex string) color.Color {
	if pl.plain {
		return nil
	}
	return pl.p.Convert(lipgloss.Color(hex))
}

func (pl palette) conv(c color.Color) color.Color {
	if pl.plain || c == nil {
		return nil
	}
	return pl.p.Convert(c)
}

func fg(s lipgloss.Style, c color.Color) lipgloss.Style {
	if c == nil {
		return s
	}
	return s.Foreground(c)
}

func bg(s lipgloss.Style, c color.Color) lipgloss.Style {
	if c == nil {
		return s
	}
	return s.Background(c)
}

func New(c balance.Colors, p colorprofile.Profile) *Theme {
	pl := palette{p: p, plain: p <= colorprofile.ASCII}
	t := &Theme{Profile: p, Plain: pl.plain}
	base := lipgloss.NewStyle()
	text, dim := pl.c(c.Text), pl.c(c.Dim)
	money, rate := pl.c(c.MoneyFrom), pl.c(c.MoneyTo)
	bad, gold, mag := pl.c(c.CostBad), pl.c(c.Gold), pl.c(c.Magenta)
	params, rare := pl.c(c.Params), pl.c(c.Rare)
	bgc := pl.c(c.BG)
	if p >= colorprofile.ANSI256 {
		t.BG = bgc
	}

	t.Text = paintOf(fg(base, text))
	t.Dim = paintOf(fg(base, dim))
	if pl.plain {
		t.Dim = paintOf(base.Faint(true))
	}
	t.Bold = paintOf(fg(base, text).Bold(true))
	t.Title = paintOf(fg(base, money).Bold(true))
	t.Money = paintOf(fg(base, money).Bold(true))
	t.Rate = paintOf(fg(base, rate))
	t.Bad = paintOf(fg(base, bad))
	t.Good = paintOf(fg(base, rate))
	t.Gold = paintOf(fg(base, gold).Bold(true))
	t.Magenta = paintOf(fg(base, mag).Bold(true))
	t.Params = paintOf(fg(base, params))
	t.Rare = paintOf(fg(base, rare))
	t.Border = paintOf(fg(base, dim))
	t.Marker = paintOf(fg(base, gold).Bold(true))
	t.Star = paintOf(fg(base, gold).Bold(true))
	t.Locked = t.Dim
	t.TabActive = paintOf(fg(base, money).Bold(true).Reverse(pl.plain))
	t.TabIdle = paintOf(fg(base, text))
	t.TabLocked = t.Dim
	t.Badge = paintOf(fg(base, mag).Bold(true))
	t.Key = paintOf(fg(base, money).Bold(true))
	t.KeyText = paintOf(fg(base, dim))
	t.Header = paintOf(fg(base, text).Bold(true))
	t.Selected = paintOf(fg(base, text).Bold(true))
	t.Track = paintOf(fg(base, dim))
	t.Panel = paintOf(fg(base, text))

	t.FlashPurchase = paintOf(bg(fg(base, pl.c(c.BG)), rate).Bold(true).Reverse(pl.plain))
	t.FlashMilestone = paintOf(bg(fg(base, pl.c(c.BG)), mag).Bold(true).Reverse(pl.plain))
	t.FlashCrit = paintOf(bg(fg(base, pl.c(c.BG)), gold).Bold(true).Reverse(pl.plain))
	t.FlashWarn = paintOf(fg(base, bad).Bold(true).Underline(pl.plain))

	t.ToastInfo = paintOf(bg(fg(base, text), pl.conv(lipgloss.Darken(lipgloss.Color(c.Rare), 0.45))).Bold(true).Reverse(pl.plain))
	t.ToastGold = paintOf(bg(fg(base, pl.c(c.BG)), gold).Bold(true).Reverse(pl.plain))
	t.ToastMagic = paintOf(bg(fg(base, pl.c(c.BG)), mag).Bold(true).Reverse(pl.plain))
	t.ToastWarn = paintOf(bg(fg(base, text), pl.conv(lipgloss.Darken(lipgloss.Color(c.CostBad), 0.4))).Bold(true).Reverse(pl.plain))
	t.Modal = paintOf(fg(base, text))

	for i := range Tiers {
		hex := c.MoneyFrom
		if i < len(c.TierGradient) {
			hex = c.TierGradient[i]
		}
		tc := lipgloss.Color(hex)
		t.Tier[i] = paintOf(fg(base, pl.conv(tc)))
		half := lipgloss.Blend1D(FlowSteps/2+1, lipgloss.Darken(tc, 0.35), lipgloss.Lighten(tc, 0.35))
		for k := range FlowSteps {
			j := k
			if j > FlowSteps/2 {
				j = FlowSteps - k
			}
			if j >= len(half) {
				j = len(half) - 1
			}
			t.Flow[i][k] = paintOf(fg(base, pl.conv(half[j])))
			if pl.plain && k%4 == 0 {
				t.Flow[i][k] = paintOf(base.Bold(true))
			}
		}
	}
	fade := lipgloss.Blend1D(FadeSteps, lipgloss.Color(c.MoneyTo), lipgloss.Color(c.Dim))
	goldFade := lipgloss.Blend1D(FadeSteps, lipgloss.Color(c.Gold), lipgloss.Color(c.Dim))
	for k := range FadeSteps {
		t.Fade[k] = paintOf(fg(base, pl.conv(fade[min(k, len(fade)-1)])).Bold(k < 2))
		t.GoldFade[k] = paintOf(fg(base, pl.conv(goldFade[min(k, len(goldFade)-1)])).Bold(true))
	}
	return t
}
