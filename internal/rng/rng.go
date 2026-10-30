// Package rng provides deterministic, named random streams whose full state is
// serializable, so a saved game resumes the exact same random sequence.
//
// Each stream is a PCG generator from math/rand/v2 held by value: copying a
// Streams value copies the generator state, and drawing never allocates.
package rng

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
)

// Stream is one named PCG generator.
type Stream struct {
	pcg rand.PCG
}

// NewStream returns a stream seeded from seed and a stream key.
func NewStream(seed, key uint64) Stream {
	var s Stream
	s.pcg.Seed(splitmix(seed^key), splitmix(seed+key*0x9E3779B97F4A7C15))
	return s
}

// Uint64 returns the next raw value.
func (s *Stream) Uint64() uint64 { return s.pcg.Uint64() }

// Float64 returns a uniform value in [0, 1).
func (s *Stream) Float64() float64 { return float64(s.pcg.Uint64()>>11) * (1.0 / (1 << 53)) }

// Chance reports true with probability p.
func (s *Stream) Chance(p float64) bool { return s.Float64() < p }

// IntN returns a uniform value in [0, n). n must be > 0.
func (s *Stream) IntN(n int) int {
	// Lemire's multiply-shift; the tiny bias is irrelevant for a game.
	hi, _ := mul64(s.pcg.Uint64(), uint64(n))
	return int(hi)
}

// Range returns a uniform value in [lo, hi).
func (s *Stream) Range(lo, hi float64) float64 { return lo + (hi-lo)*s.Float64() }

// Skip advances the stream by n draws.
func (s *Stream) Skip(n uint64) {
	for range n {
		s.pcg.Uint64()
	}
}

// MarshalJSON encodes the generator state as a hex string.
func (s Stream) MarshalJSON() ([]byte, error) {
	b, err := s.pcg.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return json.Marshal(hex.EncodeToString(b))
}

// UnmarshalJSON restores a state written by MarshalJSON.
func (s *Stream) UnmarshalJSON(data []byte) error {
	var h string
	if err := json.Unmarshal(data, &h); err != nil {
		return fmt.Errorf("rng: stream state must be a hex string: %w", err)
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return fmt.Errorf("rng: bad hex: %w", err)
	}
	if err := s.pcg.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("rng: %w", err)
	}
	return nil
}

// Stream keys. Fixed forever: changing them changes every seeded game.
const (
	KeyEvents uint64 = iota + 1
	KeyCrits
	KeyCards
	KeyRivals
	KeyEnergy
	KeyMisc
)

// Streams is the set of named streams kept in the game state. Separate
// streams keep systems independent: drawing more crits never shifts which
// events happen.
type Streams struct {
	Events Stream `json:"events"`
	Crits  Stream `json:"crits"`
	Cards  Stream `json:"cards"`
	Rivals Stream `json:"rivals"`
	Energy Stream `json:"energy"`
	Misc   Stream `json:"misc"`
}

// NewStreams seeds every named stream from one game seed.
func NewStreams(seed uint64) Streams {
	return Streams{
		Events: NewStream(seed, KeyEvents),
		Crits:  NewStream(seed, KeyCrits),
		Cards:  NewStream(seed, KeyCards),
		Rivals: NewStream(seed, KeyRivals),
		Energy: NewStream(seed, KeyEnergy),
		Misc:   NewStream(seed, KeyMisc),
	}
}

func splitmix(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

func mul64(x, y uint64) (hi, lo uint64) {
	const mask32 = 1<<32 - 1
	x0, x1 := x&mask32, x>>32
	y0, y1 := y&mask32, y>>32
	w0 := x0 * y0
	t := x1*y0 + w0>>32
	w1, w2 := t&mask32, t>>32
	w1 += x0 * y1
	hi = x1*y1 + w2 + w1>>32
	lo = x * y
	return
}
