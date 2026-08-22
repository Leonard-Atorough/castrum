package ecs

import "testing"

func TestHierarchy_AddAndQuery(t *testing.T) {
	h := NewHierarchy()

	h.Add(1, 2)
	h.Add(1, 3)

	if !h.IsParent(1) {
		t.Fatal("entity 1 should be a parent")
	}
	if !h.IsChild(2) {
		t.Fatal("entity 2 should be a child")
	}
	if got := h.Children(1); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("unexpected children: got %#v", got)
	}
	if parent, ok := h.Parent(2); !ok || parent != 1 {
		t.Fatalf("expected parent 1 for child 2, got parent=%d ok=%v", parent, ok)
	}
	if root := h.Root(2); root != 1 {
		t.Fatalf("expected root 1, got %d", root)
	}
}

func TestHierarchy_Descendants(t *testing.T) {
	h := NewHierarchy()

	h.Add(1, 2)
	h.Add(1, 3)
	h.Add(2, 4)
	h.Add(2, 5)
	h.Add(5, 6)

	got := h.Descendants(1)
	want := []EntityID{2, 4, 5, 6, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %d descendants, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("descendant mismatch at index %d: got %d want %d; full=%#v", i, got[i], want[i], got)
		}
	}
}

func TestHierarchy_Remove(t *testing.T) {
	h := NewHierarchy()

	h.Add(1, 2)
	h.Add(1, 3)
	h.Add(2, 4)

	h.Remove(1, 2)
	if _, ok := h.Parent(2); ok {
		t.Fatal("child 2 should no longer have a parent")
	}
	if got := h.Children(1); len(got) != 1 || got[0] != 3 {
		t.Fatalf("unexpected child list after remove: %#v", got)
	}
	if !h.IsParent(2) {
		t.Fatal("entity 2 should still be a parent because it still owns child 4")
	}
}

func TestHierarchy_Reparent(t *testing.T) {
	h := NewHierarchy()
	h.Add(1, 2)
	h.Add(3, 2)

	if got := h.Children(1); len(got) != 0 {
		t.Fatalf("old parent should be cleaned up: %#v", got)
	}
	if parent, ok := h.Parent(2); !ok || parent != 3 {
		t.Fatalf("expected parent 3, got parent=%d ok=%v", parent, ok)
	}
	if got := h.Children(3); len(got) != 1 || got[0] != 2 {
		t.Fatalf("new parent should contain child: %#v", got)
	}
}

func TestHierarchy_EmptyAndUnknown(t *testing.T) {
	h := NewHierarchy()

	if h.IsParent(999) {
		t.Fatal("unknown entity should not be a parent")
	}
	if h.IsChild(999) {
		t.Fatal("unknown entity should not be a child")
	}
	if got := h.Children(999); len(got) != 0 {
		t.Fatalf("unknown parent should return no children, got %#v", got)
	}
	if parent, ok := h.Parent(999); ok || parent != 0 {
		t.Fatalf("unknown child should have no parent, got parent=%d ok=%v", parent, ok)
	}
	if got := h.Descendants(999); len(got) != 0 {
		t.Fatalf("unknown ancestor should have no descendants, got %#v", got)
	}
	if root := h.Root(999); root != 999 {
		t.Fatalf("root of an orphan should be itself, got %d", root)
	}
}
