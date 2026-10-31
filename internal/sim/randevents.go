package sim

import (
	"math"
	"strings"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

func activeEvent(s *State, b *balance.Balance) int {
	if s.RandomEvent.ActiveID == "" {
		return -1
	}
	i, ok := b.EventIndex(s.RandomEvent.ActiveID)
	if !ok {
		return -1
	}
	return i
}

func timedEvent(s *State, b *balance.Balance) int {
	i := activeEvent(s, b)
	if i < 0 || b.Event[i].Effect == balance.EventReaction {
		return -1
	}
	return i
}

func ActiveEvent(s *State, b *balance.Balance) (index int, remainingS float64) {
	i := activeEvent(s, b)
	if i < 0 {
		return -1, 0
	}
	return i, math.Max(0, s.RandomEvent.EndsAtS-s.Clock)
}

func eventEligible(s *State, ev *balance.Event, offline bool) bool {
	switch ev.Effect {
	case balance.EventReaction:
		return !offline
	case balance.EventHype:
		return s.IsUnlocked(UnlockMarket)
	case balance.EventRackCost:
		return s.IsUnlocked(UnlockInfra)
	}
	return true
}

func scheduleNext(s *State, b *balance.Balance, from float64) {
	iv := b.Events.IntervalS
	s.RandomEvent.NextAtS = from + s.RNG.Events.Range(iv[0], iv[1])
}

func endEvent(s *State, b *balance.Balance, at float64) {
	s.RandomEvent.ActiveID = ""
	scheduleNext(s, b, at)
}

func tickEvents(s *State, b *balance.Balance, dt float64, out []Event, md mode) ([]Event, float64) {
	if !s.IsUnlocked(UnlockEvents) || len(b.Event) == 0 {
		return out, 1
	}
	re := &s.RandomEvent
	now := s.Clock
	changed := false
	if i := activeEvent(s, b); i >= 0 {
		ev := &b.Event[i]
		switch {
		case ev.Effect == balance.EventReaction && md.offline:
			endEvent(s, b, now)
			changed = true
		case now >= re.EndsAtS && ev.Effect == balance.EventReaction:
			loss := math.Min(s.Money, MinutesOfIncome(s, ev.PenaltyMinutes)*s.ResMods.PenaltyMult)
			s.Money -= loss
			s.Stats.EventsMissed++
			endEvent(s, b, re.EndsAtS)
			out = append(out, Event{Kind: EvReactMiss, Business: -1, Index: i, Amount: loss})
		case now >= re.EndsAtS:
			endEvent(s, b, re.EndsAtS)
			out = append(out, Event{Kind: EvEventEnd, Business: -1, Index: i})
			changed = true
		}
	} else if re.ActiveID != "" {
		re.ActiveID = ""
	}
	if re.ActiveID == "" {
		switch {
		case re.NextAtS <= 0:
			scheduleNext(s, b, now)
		case now >= re.NextAtS:
			if i := pickEvent(s, b, md.offline); i >= 0 {
				ev := &b.Event[i]
				re.ActiveID = ev.ID
				d := ev.DurationS
				if ev.Effect == balance.EventReaction {
					d = ev.WindowS
				}
				re.EndsAtS = now + d
				s.Stats.EventsSeen++
				out = append(out, Event{Kind: EvEventStart, Business: -1, Index: i, Value: d})
				changed = true
			} else {
				scheduleNext(s, b, now)
			}
		}
	}
	if changed {
		Recompute(s, b)
	}
	frac := 1.0
	if timedEvent(s, b) >= 0 && re.EndsAtS < now+dt {
		frac = math.Max(0, (re.EndsAtS-now)/dt)
	}
	return out, frac
}

func pickEvent(s *State, b *balance.Balance, offline bool) int {
	total := 0.0
	for i := range b.Event {
		if eventEligible(s, &b.Event[i], offline) {
			total += b.Event[i].Weight
		}
	}
	if !(total > 0) {
		return -1
	}
	x := s.RNG.Events.Float64() * total
	pick := -1
	for i := range b.Event {
		if !eventEligible(s, &b.Event[i], offline) {
			continue
		}
		pick = i
		x -= b.Event[i].Weight
		if x < 0 {
			break
		}
	}
	return pick
}

func ReactionReward(s *State, b *balance.Balance, i int) float64 {
	if i < 0 || i >= len(b.Event) {
		return 0
	}
	return MinutesOfIncome(s, b.Event[i].RewardMinutes) * (1 + s.CardMods.ReactionReward)
}

func react(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i := activeEvent(s, b)
	if i < 0 {
		return reject(out, c, RejectNoEvent, 0)
	}
	ev := &b.Event[i]
	if ev.Effect != balance.EventReaction || !strings.EqualFold(ev.Key, c.Text) || s.Clock >= s.RandomEvent.EndsAtS {
		return reject(out, c, RejectNoEvent, 0)
	}
	reward := ReactionReward(s, b, i)
	earn(s, reward)
	s.Stats.EventsCaught++
	endEvent(s, b, s.Clock)
	return append(out, Event{Kind: EvReact, Business: -1, Index: i, Amount: reward})
}
