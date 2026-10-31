package sim

import (
	"math"
	"strings"

	"github.com/prbartosh/token-tycoon/internal/balance"
)

type Incident struct {
	ID      string  `json:"id"`
	EndsAtS float64 `json:"ends_at_s"`
}

func IncidentsOn(s *State, b *balance.Balance) bool {
	return s.IsUnlocked(UnlockSafety) && !AGIDone(s, b, balance.AGINoIncidents)
}

func IncidentEveryH(s *State, b *balance.Balance) float64 {
	sf := &b.Safety
	m := s.ResMods.IncidentMult
	if !(m > 0) {
		return sf.IncidentEveryHMax
	}
	return math.Min(math.Max(sf.IncidentEveryHBase, sf.IncidentEveryHBase/m), sf.IncidentEveryHMax)
}

func ActiveIncident(s *State, b *balance.Balance) (index int, leftS float64) {
	if s.Incident.ID == "" {
		return -1, 0
	}
	i, ok := b.IncidentIndex(s.Incident.ID)
	if !ok {
		return -1, 0
	}
	return i, math.Max(0, s.Incident.EndsAtS-s.Clock)
}

func IncidentLoss(s *State, b *balance.Balance, i int) float64 {
	if i < 0 || i >= len(b.Incidents) {
		return 0
	}
	return math.Max(0, s.Market.Hype) * math.Min(1, b.Safety.HypeLoss*b.Incidents[i].Severity)
}

func pickIncident(s *State, b *balance.Balance) int {
	total := 0.0
	for i := range b.Incidents {
		total += b.Incidents[i].Weight
	}
	if !(total > 0) {
		return -1
	}
	x := s.RNG.Safety.Float64() * total
	for i := range b.Incidents {
		x -= b.Incidents[i].Weight
		if x < 0 {
			return i
		}
	}
	return len(b.Incidents) - 1
}

func tickSafety(s *State, b *balance.Balance, dt float64, out []Event, md mode) []Event {
	in := &s.Incident
	if in.ID != "" {
		i, ok := b.IncidentIndex(in.ID)
		switch {
		case !ok || md.offline || !IncidentsOn(s, b):
			*in = Incident{}
		case s.Clock+dt >= in.EndsAtS:
			loss := IncidentLoss(s, b, i)
			s.Market.Hype = math.Max(0, s.Market.Hype-loss)
			s.Stats.IncidentsMissed++
			*in = Incident{}
			out = append(out, Event{Kind: EvIncidentMiss, Business: -1, Index: i, Amount: loss})
		}
		return out
	}
	if md.offline || !IncidentsOn(s, b) {
		return out
	}
	rate := 1 / (IncidentEveryH(s, b) * 3600)
	if !s.RNG.Safety.Chance(1 - math.Exp(-rate*dt)) {
		return out
	}
	i := pickIncident(s, b)
	if i < 0 {
		return out
	}
	w := b.Safety.ReactionWindowS
	*in = Incident{ID: b.Incidents[i].ID, EndsAtS: s.Clock + w}
	s.Stats.IncidentsSeen++
	return append(out, Event{Kind: EvIncident, Business: -1, Index: i, Value: w, Amount: IncidentLoss(s, b, i)})
}

func respond(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	i, left := ActiveIncident(s, b)
	if i < 0 || !(left > 0) || !strings.EqualFold(c.Text, b.Safety.Key) {
		return reject(out, c, RejectNoEvent, 0)
	}
	s.Incident = Incident{}
	s.Market.Reputation = math.Min(b.Contracts.ReputationMax, math.Max(0, s.Market.Reputation)+b.Safety.ReputationBonus)
	s.Stats.IncidentsOK++
	return append(out, Event{Kind: EvIncidentOK, Business: -1, Index: i, Value: s.Market.Reputation})
}
