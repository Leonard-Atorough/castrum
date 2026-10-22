package debug

import (
	"reflect"
	"testing"
)

func TestMessageOrderAndCapacity(t *testing.T) {
	p := NewPanel()
	p.Message("one")
	p.Message("two")
	if got := p.Lines(); !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("lines = %v, want [one two]", got)
	}

	for i := range messageCapacity {
		p.Message(string(rune('a' + i)))
	}
	want := make([]string, 0, messageCapacity)
	for i := range messageCapacity {
		want = append(want, string(rune('a'+i)))
	}
	if got := p.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}

	// Filling twice over drops the oldest first: the ring wraps.
	p.Message("new")
	want = append(want[1:], "new")
	if got := p.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
}

func TestClear(t *testing.T) {
	p := NewPanel()
	p.Message("one")
	p.Clear()
	if got := p.Lines(); len(got) != 0 {
		t.Fatalf("lines = %v, want empty", got)
	}
	p.Message("fresh")
	if got := p.Lines(); !reflect.DeepEqual(got, []string{"fresh"}) {
		t.Fatalf("lines = %v, want [fresh] after clear", got)
	}
}

func TestToggleFlipsSections(t *testing.T) {
	p := NewPanel()
	if !p.Enabled(FPS) || !p.Enabled(TPS) {
		t.Fatal("new panel should have every builtin enabled")
	}

	p.Toggle(FPS)
	if p.Enabled(FPS) || !p.Enabled(TPS) {
		t.Fatal("toggle should flip only its own section")
	}

	p.Toggle(FPS)
	if !p.Enabled(FPS) {
		t.Fatal("toggling back should re-enable the section")
	}
}

func TestUndefinedBuiltinIsInert(t *testing.T) {
	p := NewPanel()
	p.Toggle(Builtin(99))
	if !p.Enabled(FPS) || !p.Enabled(TPS) {
		t.Fatal("an undefined builtin must not disturb the sections")
	}
	if p.Enabled(Builtin(99)) {
		t.Fatal("an undefined builtin must report disabled")
	}
}

func TestStatusCreatesInWriteOrder(t *testing.T) {
	p := NewPanel()
	p.Status("controls", "SPACE: jump")
	p.Status("mixer", "master 50%")
	if got := p.StatusLines(); !reflect.DeepEqual(got, []string{"SPACE: jump", "master 50%"}) {
		t.Fatalf("status lines = %v, want [controls mixer] in write order", got)
	}
}

func TestStatusOverwritesInPlace(t *testing.T) {
	p := NewPanel()
	p.Status("controls", "SPACE: jump")
	p.Status("mixer", "master 50%")
	for range 100 {
		p.Status("mixer", "master 75%")
	}
	got := p.StatusLines()
	if !reflect.DeepEqual(got, []string{"SPACE: jump", "master 75%"}) {
		t.Fatalf("status lines = %v, want the mixer slot updated in place", got)
	}
	if len(got) != 2 {
		t.Fatalf("per-frame status writes must not create slots: %d lines", len(got))
	}
}

func TestStatusEmptyHidesButKeepsOrder(t *testing.T) {
	p := NewPanel()
	p.Status("controls", "SPACE: jump")
	p.Status("cooldown", "1.2s")
	p.Status("cooldown", "")
	if got := p.StatusLines(); !reflect.DeepEqual(got, []string{"SPACE: jump"}) {
		t.Fatalf("status lines = %v, want the empty cooldown hidden", got)
	}
	p.Status("cooldown", "0.8s")
	if got := p.StatusLines(); !reflect.DeepEqual(got, []string{"SPACE: jump", "0.8s"}) {
		t.Fatalf("status lines = %v, want the cooldown back in its slot", got)
	}
}

func TestClearRemovesStatuses(t *testing.T) {
	p := NewPanel()
	p.Status("controls", "SPACE: jump")
	p.Message("event")
	p.Clear()
	p.Status("controls", "later")
	if got := p.StatusLines(); !reflect.DeepEqual(got, []string{"later"}) {
		t.Fatalf("status lines = %v, want a fresh slot after Clear", got)
	}
}
