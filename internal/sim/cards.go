package sim

import (
	"token-tycoon/internal/balance"
	"token-tycoon/internal/cards"
)

func dropCard(s *State, b *balance.Balance, out []Event) []Event {
	i := cards.Draw(b, &s.RNG.Cards)
	if i < 0 {
		return out
	}
	before := s.CardMods.SetsComplete
	id := b.Card[i].ID
	s.Cards[id]++
	s.Stats.CardsDropped++
	cards.Compute(b, s.Cards, &s.CardMods)
	out = append(out, Event{Kind: EvCard, Business: -1, Index: i, Count: s.Cards[id]})
	for r, done := range s.CardMods.SetsComplete {
		if done && !before[r] {
			out = append(out, Event{Kind: EvCardSet, Business: -1, Index: r})
		}
	}
	return out
}

func CardLevel(s *State, b *balance.Balance, i int) int {
	if i < 0 || i >= len(b.Card) {
		return 0
	}
	return min(s.Cards[b.Card[i].ID], b.Cards.MaxLevel)
}

func CardBonus(s *State, b *balance.Balance, i int) float64 {
	if i < 0 || i >= len(b.Card) {
		return 0
	}
	return cards.LevelBonus(b, b.Card[i].Value, s.Cards[b.Card[i].ID])
}
