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
