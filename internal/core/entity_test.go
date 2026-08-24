package core

import "testing"

func TestEntity_BasicLifecycle(t *testing.T) {
	e := NewEntity(42, "player")

	if e.ID() != 42 {
		t.Fatalf("expected id 42, got %d", e.ID())
	}
	if e.Template() != "player" {
		t.Fatalf("expected template %q, got %q", "player", e.Template())
	}
	if !e.IsAlive() {
		t.Fatal("new entity should be alive")
	}
	if e.Version() != 0 {
		t.Fatalf("expected version 0, got %d", e.Version())
	}

	e.Destroy()
	if e.IsAlive() {
		t.Fatal("destroyed entity should not be alive")
	}
}

func TestEntity_Clone(t *testing.T) {
	src := NewEntity(5, "enemy")
	src.AddTag("hostile")
	src.Destroy()

	clone := src.Clone(99)
	if clone.ID() != 99 {
		t.Fatalf("expected cloned id 99, got %d", clone.ID())
	}
	if clone.Template() != "enemy" {
		t.Fatalf("expected template enemy, got %q", clone.Template())
	}
	if clone.IsAlive() != src.IsAlive() {
		t.Fatal("clone alive state should match source")
	}
	if clone.Version() != src.Version() {
		t.Fatalf("expected version %d, got %d", src.Version(), clone.Version())
	}
	if !clone.HasTag("hostile") {
		t.Fatal("clone should have copied tags from source")
	}
}

func TestEntity_CloneIsIndependent(t *testing.T) {
	src := NewEntity(1, "npc")
	clone := src.Clone(2)

	clone.Destroy()
	if src.IsAlive() != true {
		t.Fatal("source entity should remain unaffected by clone mutation")
	}
	if clone.ID() == src.ID() {
		t.Fatal("clone should have a distinct ID from the source")
	}
}

func TestEntity_Tags(t *testing.T) {
	e := NewEntity(7, "city")

	if e.HasTag("player") {
		t.Fatal("entity should not have an unset tag")
	}

	e.AddTag("player")
	e.AddTag("player")
	e.AddTag("enemy")

	if !e.HasTag("player") {
		t.Fatal("entity should have added tag")
	}

	got := e.Tags()
	if len(got) != 2 {
		t.Fatalf("expected 2 tags, got %d: %#v", len(got), got)
	}

	e.RemoveTag("player")
	if e.HasTag("player") {
		t.Fatal("entity should not keep removed tag")
	}
	if !e.HasTag("enemy") {
		t.Fatal("entity should keep unrelated tags")
	}
}
