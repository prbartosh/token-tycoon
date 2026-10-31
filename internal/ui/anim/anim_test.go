package anim

import (
	"math"
	"testing"
)

func TestTweenConvergesIndependentOfFPS(t *testing.T) {
	for _, fps := range []float64{30, 10} {
		tw := Tween{}
		tw.Set(1000)
		for range int(0.3 * fps) {
			tw.Advance(1 / fps)
		}
		if tw.Value < 850 || tw.Value > 1000 {
			t.Fatalf("fps %v: %v after 300ms", fps, tw.Value)
		}
		for range int(2 * fps) {
			tw.Advance(1 / fps)
		}
		if tw.Value != 1000 || tw.Moving() {
			t.Fatalf("fps %v: did not snap: %v", fps, tw.Value)
		}
	}
}

func TestFlashFloatToastLifecycle(t *testing.T) {
	var a Animator
	a.Flash.Start(3, KindCrit, 0.3)
	a.Float.Spawn(3, "+1K", KindMoney, 0.3, 3)
	a.Toast.Push("hello", KindInfo, 1)
	if !a.Active() {
		t.Fatal("should be active")
	}
	if k, f := a.Flash.At(3); k != KindCrit || f != 1 {
		t.Fatal("flash")
	}
	a.Advance(0.15)
	if f := a.Float.Items()[0]; f.Offset() < 1 || f.Offset() > 2 {
		t.Fatalf("floater offset %d", f.Offset())
	}
	for range 60 {
		a.Advance(1.0 / 30)
	}
	if a.Flash.Active() || a.Float.Active() {
		t.Fatal("flash and floater should be gone")
	}
	if len(a.Toast.Items()) != 0 || a.Active() {
		t.Fatalf("toast should have left: %+v", a.Toast.Items())
	}
	for range MaxFloaters + 5 {
		a.Float.Spawn(0, "x", KindMoney, 1, 1)
	}
	for range MaxToasts + 2 {
		a.Toast.Push("t", KindInfo, 1)
	}
	if len(a.Toast.Items()) != MaxToasts {
		t.Fatal("toast cap")
	}
}

func TestToastSlidesIn(t *testing.T) {
	var ts Toasts
	ts.Push("x", KindInfo, 3)
	for range 8 {
		ts.Advance(1.0 / 30)
	}
	if p := ts.Items()[0].Pos; math.Abs(p) > 0.15 {
		t.Fatalf("toast not in after 250ms: %v", p)
	}
}

func TestToastQueueKeepsMessages(t *testing.T) {
	var ts Toasts
	for i := range MaxToasts + 3 {
		ts.Push(string(rune('a'+i)), KindInfo, 10)
	}
	if len(ts.Items()) != MaxToasts || ts.Pending() != 3 {
		t.Fatalf("visible %d pending %d", len(ts.Items()), ts.Pending())
	}
	for range int((MinShow - 0.5) * 30) {
		ts.Advance(1.0 / 30)
	}
	if ts.Items()[0].Text != "a" || ts.Items()[0].Age < MinShow-0.6 {
		t.Fatal("first toast left before the minimum show time")
	}
	seen := map[string]bool{}
	for range 30 * 60 {
		for _, it := range ts.Items() {
			seen[it.Text] = true
		}
		ts.Advance(1.0 / 30)
	}
	if len(seen) != MaxToasts+3 || ts.Pending() != 0 {
		t.Fatalf("seen %d pending %d", len(seen), ts.Pending())
	}
}

func TestLongFloaterHoldsAndRests(t *testing.T) {
	var fs Floaters
	fs.Replace(1, "burst", KindCrit, 5, 1)
	fs.Replace(1, "burst 2", KindCrit, 5, 1)
	n := 0
	for _, f := range fs.Items() {
		if f.Alive() {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("replace spawned %d", n)
	}
	for range 15 {
		fs.Advance(1.0 / 30)
	}
	f := fs.Items()[0]
	if fs.Active() || !fs.Alive() || f.Fade() != 0 || f.Offset() != 1 || f.Text != "burst 2" {
		t.Fatalf("hold phase: active %v fade %v offset %d", fs.Active(), f.Fade(), f.Offset())
	}
}
