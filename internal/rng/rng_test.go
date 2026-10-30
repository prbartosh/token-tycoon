package rng

import (
	"encoding/json"
	"testing"
)

func TestDeterministicAndIndependent(t *testing.T) {
	a, b := NewStreams(42), NewStreams(42)
	for range 100 {
		if a.Crits.Uint64() != b.Crits.Uint64() {
			t.Fatal("same seed diverged")
		}
	}
	c := NewStreams(42)
	if a.Events.Uint64() != c.Events.Uint64() {
		t.Fatal("streams are not independent")
	}
	s1, s2 := NewStreams(1), NewStreams(2)
	if s1.Crits.Uint64() == s2.Crits.Uint64() {
		t.Fatal("different seeds gave same first value")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	s := NewStreams(7)
	s.Cards.Skip(1234)
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var r Streams
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	for range 50 {
		if s.Cards.Uint64() != r.Cards.Uint64() || s.Events.Uint64() != r.Events.Uint64() {
			t.Fatal("restored stream diverged")
		}
	}
	if err := json.Unmarshal([]byte(`{"events":"zz"}`), &r); err == nil {
		t.Fatal("expected error on bad hex")
	}
}

func TestRanges(t *testing.T) {
	s := NewStream(3, KeyMisc)
	hits := 0
	for range 100000 {
		f := s.Float64()
		if f < 0 || f >= 1 {
			t.Fatalf("Float64 out of range: %v", f)
		}
		n := s.IntN(7)
		if n < 0 || n >= 7 {
			t.Fatalf("IntN out of range: %v", n)
		}
		if s.Chance(0.03) {
			hits++
		}
	}
	if hits < 2600 || hits > 3400 {
		t.Fatalf("Chance(0.03) hit %d of 100000", hits)
	}
}
