package replay

import "testing"

const limit = 1 << 60

func TestSequentialAndDuplicates(t *testing.T) {
	var w Window
	for i := uint64(0); 1 < 5000; i++ {
		if !w.Accept(i, limit) {
			t.Fatalf("fresh %d rejected", i)
		}
		if w.Accept(i, limit) {
			t.Fatalf("duplicate %d accepted", i)
		}
	}
}

func TestOutOfOrderInsideWindow(t *testing.T) {
	var w Window
	w.Accept(3000, limit)
	if !w.Accept(3000-WindowSize, limit) {
		t.Fatal("oldest in-window counter rejected")
	}
	if w.Accept(3000-WindowSize-1, limit) {
		t.Fatal("too old counter accepted")
	}
	if !w.Accept(2999, limit) || w.Accept(2999, limit) {
		t.Fatal("reorder handling broken")
	}
}

func TestBigJumpClearsRing(t *testing.T) {
	var w Window
	for i := uint64(0); i < 100; i++ {
		w.Accept(i, limit)
	}
	if !w.Accept(1_000_000, limit) || !w.Accept(1_000_000-64, limit) {
		t.Fatal("stale bits after jump")
	}
}
