package ecs

import (
	"testing"

	"github.com/leonard-atorough/castrum/pkg/component"
)

type testComponentA struct {
	value int
}

func (t testComponentA) Name() string               { return "testComponentA" }
func (t testComponentA) Clone() component.Component { return testComponentA{value: t.value} }

type testComponentB struct {
	value int
}

func (t testComponentB) Name() string               { return "testComponentB" }
func (t testComponentB) Clone() component.Component { return testComponentB{value: t.value} }

func TestComponentStore_SetAndGetAll(t *testing.T) {
	store := NewComponentStore()
	store.Set(10, testComponentA{value: 1})
	store.Set(10, testComponentB{value: 2})

	got := store.GetAll(10)
	if len(got) != 2 {
		t.Fatalf("expected 2 components, got %d", len(got))
	}
	if _, ok := got[0].(testComponentA); !ok {
		t.Fatalf("expected first component to be testComponentA, got %#v", got[0])
	}
	if _, ok := got[1].(testComponentB); !ok {
		t.Fatalf("expected second component to be testComponentB, got %#v", got[1])
	}
}

func TestComponentStore_GetTypedComponent(t *testing.T) {
	store := NewComponentStore()
	store.Set(1, testComponentA{value: 99})
	store.Set(1, testComponentB{value: 42})

	got, err := Get[testComponentA](store, 1)
	if err != nil {
		t.Fatalf("expected Get[testComponentA] to succeed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil component pointer")
	}
	if got.value != 99 {
		t.Fatalf("expected value 99, got %d", got.value)
	}

	missing, err := Get[testComponentB](store, 2)
	if err == nil {
		t.Fatal("expected missing component type to return an error")
	}
	if missing != nil {
		t.Fatalf("expected nil pointer when component type is missing, got %#v", missing)
	}
}

func TestComponentStore_RemoveAndRemoveAll(t *testing.T) {
	store := NewComponentStore()
	a := testComponentA{value: 7}
	b := testComponentB{value: 8}
	store.Set(1, a)
	store.Set(1, b)

	store.Remove(1, a)
	got := store.GetAll(1)
	if len(got) != 1 {
		t.Fatalf("expected 1 component after Remove, got %d", len(got))
	}
	if _, ok := got[0].(testComponentB); !ok {
		t.Fatalf("expected remaining component to be testComponentB, got %#v", got[0])
	}

	store.RemoveAll(1)
	if got := store.GetAll(1); len(got) != 0 {
		t.Fatalf("expected no components after RemoveAll, got %#v", got)
	}
}

func TestComponentStore_MissingEntity(t *testing.T) {
	store := NewComponentStore()

	if got := store.GetAll(123); len(got) != 0 {
		t.Fatalf("missing entity should return no components, got %#v", got)
	}

	got, err := Get[testComponentA](store, 123)
	if err == nil {
		t.Fatal("missing entity should return an error")
	}
	if got != nil {
		t.Fatalf("missing entity should return nil pointer, got %#v", got)
	}
}
