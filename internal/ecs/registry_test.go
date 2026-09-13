package ecs

import (
	"errors"
	"reflect"
	"testing"
)

type registryPosition struct{ X int }
type registryVelocity struct{ X int }

func registryFactory(props map[string]any) (any, error) {
	return registryPosition{X: props["X"].(int)}, nil
}

func TestRegistryRegisterResolveAndLookup(t *testing.T) {
	registry := NewRegistry()
	typ := reflect.TypeFor[registryPosition]()

	if err := registry.Register("game.position", typ, registryFactory); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !registry.Has("game.position") {
		t.Fatal("registered component should be present")
	}

	definition, ok := registry.LookupByName("game.position")
	if !ok || definition.Type != typ || definition.Size != typ.Size() {
		t.Fatalf("unexpected definition: %#v, %v", definition, ok)
	}
	definition, ok = registry.LookupByType(typ)
	if !ok || definition.Name != "game.position" {
		t.Fatalf("unexpected type lookup: %#v, %v", definition, ok)
	}

	value, err := registry.Resolve("game.position", map[string]any{"X": 4})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got := value.(registryPosition); got.X != 4 {
		t.Fatalf("unexpected resolved value: %#v", got)
	}
}

func TestRegistryRejectsInvalidAndConflictingRegistrations(t *testing.T) {
	registry := NewRegistry()
	typ := reflect.TypeFor[registryPosition]()
	factory := func(map[string]any) (any, error) { return registryPosition{}, nil }

	invalid := []struct {
		name string
		typ  reflect.Type
		fn   ComponentFactory
	}{
		{"", typ, factory},
		{"   ", typ, factory},
		{"game.nil-type", nil, factory},
		{"game.nil-factory", typ, nil},
	}
	for _, test := range invalid {
		if err := registry.Register(test.name, test.typ, test.fn); err == nil {
			t.Fatalf("Register(%q, %v) should fail", test.name, test.typ)
		}
	}

	if err := registry.Register("game.position", typ, factory); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}
	if err := registry.Register("game.position", typ, factory); err != nil {
		t.Fatalf("identical registration should be idempotent: %v", err)
	}
	if err := registry.Register("game.position", reflect.TypeFor[registryVelocity](), factory); err == nil {
		t.Fatal("same name with a different type should fail")
	}
	if err := registry.Register("game.velocity", typ, factory); err == nil {
		t.Fatal("same type with a different name should fail")
	}
}

func TestRegistryResolveErrorsAndFactoryErrors(t *testing.T) {
	registry := NewRegistry()
	factoryErr := errors.New("factory failed")
	if err := registry.Register("game.failure", reflect.TypeFor[registryPosition](), func(map[string]any) (any, error) {
		return nil, factoryErr
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if _, err := registry.Resolve("game.missing", nil); err == nil {
		t.Fatal("unknown component should fail to resolve")
	}
	if _, err := registry.Resolve("game.failure", nil); !errors.Is(err, factoryErr) {
		t.Fatalf("factory error was not preserved: %v", err)
	}
	if _, ok := registry.LookupByName("game.missing"); ok {
		t.Fatal("unknown component should not be found")
	}
	if _, ok := registry.LookupByType(reflect.TypeFor[registryVelocity]()); ok {
		t.Fatal("unknown component type should not be found")
	}
}
