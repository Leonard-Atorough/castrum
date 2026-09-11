package ecs

import "testing"

func TestEntity_BasicLifecycle(t *testing.T) {
	e := NewEntity(42, "player")

	if e.ID != 42 {
		t.Errorf("expected id 42, got %d", e.ID)
	}
	if e.Template() != "player" {
		t.Errorf("expected template %q, got %q", "player", e.Template())
	}
	if !e.IsAlive() {
		t.Error("new entity should be alive")
	}
	if e.Version() != 0 {
		t.Errorf("expected version 0, got %d", e.Version())
	}

	e.Destroy()
	if e.IsAlive() {
		t.Error("destroyed entity should not be alive")
	}
}

func TestEntity_Clone(t *testing.T) {
	src := NewEntity(5, "enemy")
	src.Destroy()

	clone := src.Clone(99)
	if clone.ID != 99 {
		t.Errorf("expected cloned id 99, got %d", clone.ID)
	}
	if clone.Template() != "enemy" {
		t.Errorf("expected template enemy, got %q", clone.Template())
	}
	if clone.IsAlive() != src.IsAlive() {
		t.Error("clone alive state should match source")
	}
	if clone.Version() != src.Version() {
		t.Errorf("expected version %d, got %d", src.Version(), clone.Version())
	}
}

func TestEntity_CloneIsIndependent(t *testing.T) {
	src := NewEntity(1, "npc")
	clone := src.Clone(2)

	clone.Destroy()
	if src.IsAlive() != true {
		t.Error("source entity should remain unaffected by clone mutation")
	}
	if clone.ID == src.ID {
		t.Error("clone should have a distinct ID from the source")
	}
}
