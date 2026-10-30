package sim

import "token-tycoon/internal/balance"

// civilDays converts a YYYY-MM-DD date to days since 1970-01-01 (Howard
// Hinnant's days_from_civil), so the streak works without a clock.
func civilDays(date string) (int, bool) {
	if len(date) != 10 || date[4] != '-' || date[7] != '-' {
		return 0, false
	}
	num := func(s string) (int, bool) {
		n := 0
		for i := range len(s) {
			if s[i] < '0' || s[i] > '9' {
				return 0, false
			}
			n = n*10 + int(s[i]-'0')
		}
		return n, true
	}
	y, ok1 := num(date[:4])
	m, ok2 := num(date[5:7])
	d, ok3 := num(date[8:])
	if !ok1 || !ok2 || !ok3 || m < 1 || m > 12 || d < 1 || d > daysIn(y, m) {
		return 0, false
	}
	if m <= 2 {
		y--
	}
	era := y / 400
	yoe := y - era*400
	mp := (m + 9) % 12
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468, true
}

func daysIn(y, m int) int {
	switch m {
	case 2:
		if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

type GrantInfo struct {
	Claimable bool
	Day       int
	Amount    float64
	Reason    Reject
}

func GrantPreview(s *State, b *balance.Balance, date string) GrantInfo {
	today, ok := civilDays(date)
	if !ok {
		return GrantInfo{Reason: RejectBadDate}
	}
	st := &s.Streak
	day := 1
	if last, ok := civilDays(st.LastClaim); ok {
		switch gap := today - last; {
		case gap <= 0:
			return GrantInfo{Day: st.Day, Reason: RejectAlreadyClaimed}
		case gap == 1:
			day = min(st.Day+1, b.Streak.MaxDay)
		default:
			day = max(1, st.Day-b.Streak.BreakPenalty)
		}
	}
	return GrantInfo{Claimable: true, Day: day, Amount: MinutesOfIncome(s, float64(day)*b.Streak.BonusPerDayMinutes)}
}

func claimGrant(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	g := GrantPreview(s, b, c.Text)
	if !g.Claimable {
		return reject(out, c, g.Reason, 0)
	}
	s.Streak.Day = g.Day
	s.Streak.LastClaim = c.Text
	s.Stats.Grants++
	earn(s, g.Amount)
	Recompute(s, b)
	return append(out, Event{Kind: EvGrant, Business: -1, Count: g.Day, Amount: g.Amount})
}
