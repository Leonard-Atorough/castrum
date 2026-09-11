package ecs

import "testing"

func TestSetResource_GetResource(t *testing.T) {
	w := NewWorld()

	type Config struct {
		Name string
	}

	SetResource(w, Config{Name: "level-1"})

	got, ok := GetResource[Config](w)
	if !ok {
		t.Error("expected resource to be found")
	}
	if got.Name != "level-1" {
		t.Errorf("expected Name 'level-1', got %q", got.Name)
	}
}

func TestGetResource_NotSet(t *testing.T) {
	w := NewWorld()

	type Unset struct{}

	_, ok := GetResource[Unset](w)
	if ok {
		t.Error("expected ok=false for a resource that was never set")
	}
}

func TestSetResource_Overwrite(t *testing.T) {
	w := NewWorld()

	SetResource(w, 1)
	SetResource(w, 2)

	got, ok := GetResource[int](w)
	if !ok || got != 2 {
		t.Errorf("expected overwritten value 2, got %v (ok=%v)", got, ok)
	}
}

func TestSetResource_PointerType(t *testing.T) {
	w := NewWorld()

	type Manager struct {
		Count int
	}

	mgr := &Manager{Count: 5}
	SetResource(w, mgr)

	got, ok := GetResource[*Manager](w)
	if !ok {
		t.Error("expected pointer resource to be found")
	}
	if got != mgr {
		t.Error("expected the same pointer instance back")
	}
	got.Count = 10
	if mgr.Count != 10 {
		t.Error("expected mutations through the retrieved pointer to be visible on the original")
	}
}

func TestRemoveResource(t *testing.T) {
	w := NewWorld()

	SetResource(w, "hello")
	RemoveResource[string](w)

	_, ok := GetResource[string](w)
	if ok {
		t.Error("expected resource to be removed")
	}
}

func TestGetResource_WrongTypeAssertionMismatch(t *testing.T) {
	// Distinct types never collide, even with the same underlying kind.
	w := NewWorld()

	type A struct{ V int }
	type B struct{ V int }

	SetResource(w, A{V: 1})

	_, ok := GetResource[B](w)
	if ok {
		t.Error("expected no match for a different type with the same shape")
	}
}
