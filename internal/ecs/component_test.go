package ecs

import (
	"reflect"
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

	// Check that both component types are present (map iteration order is non-deterministic)
	hasA := false
	hasB := false
	for _, comp := range got {
		if _, ok := comp.(testComponentA); ok {
			hasA = true
		}
		if _, ok := comp.(testComponentB); ok {
			hasB = true
		}
	}

	if !hasA {
		t.Fatalf("expected testComponentA to be in results, got %#v", got)
	}
	if !hasB {
		t.Fatalf("expected testComponentB to be in results, got %#v", got)
	}
}

func TestComponentStore_GetTypedComponent(t *testing.T) {
	store := NewComponentStore()
	store.Set(1, testComponentA{value: 99})
	store.Set(1, testComponentB{value: 42})

	got, err := store.Get(1, reflect.TypeFor[testComponentA]())
	if err != nil {
		t.Fatalf("expected Get to succeed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil component")
	}
	comp, ok := got.(testComponentA)
	if !ok {
		t.Fatalf("expected testComponentA, got %T", got)
	}
	if comp.value != 99 {
		t.Fatalf("expected value 99, got %d", comp.value)
	}

	missing, err := store.Get(2, reflect.TypeFor[testComponentB]())
	if err == nil {
		t.Fatal("expected missing component to return an error")
	}
	if missing != nil {
		t.Fatalf("expected nil component when not found, got %#v", missing)
	}
}

func TestComponentStore_GetByName(t *testing.T) {
	store := NewComponentStore()
	store.Set(1, testComponentA{value: 5})
	store.Set(1, testComponentB{value: 10})

	got, err := store.GetByString(1, "testComponentB")
	if err != nil {
		t.Fatalf("expected GetByString to succeed: %v", err)
	}

	if got == nil {
		t.Fatal("expected non-nil component")
	}
	comp, ok := got.(testComponentB)
	if !ok {
		t.Fatalf("expected testComponentB, got %T", got)
	}
	if comp.value != 10 {
		t.Fatalf("expected value 10, got %d", comp.value)
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

	got, err := store.Get(123, reflect.TypeFor[testComponentA]())
	if err == nil {
		t.Fatal("missing entity should return an error")
	}
	if got != nil {
		t.Fatalf("missing entity should return nil, got %#v", got)
	}
}
