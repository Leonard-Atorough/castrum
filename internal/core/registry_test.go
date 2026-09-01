package core

import (
	"reflect"
	"testing"
)

type registryTestComponent struct {
	Name  string
	Value int
}

type registrySerializableComponent struct {
	deserializeCalled bool
	Value             string
}

func (c *registrySerializableComponent) Serialize() (map[string]any, error) {
	return map[string]any{"Value": c.Value}, nil
}

func (c *registrySerializableComponent) Deserialize(props map[string]any) error {
	c.deserializeCalled = true
	if v, ok := props["Value"].(string); ok {
		c.Value = v
	}
	return nil
}

func TestRegister(t *testing.T) {
	info := Register[registryTestComponent]()

	if info.Name != "registryTestComponent" {
		t.Fatalf("expected name %q, got %q", "registryTestComponent", info.Name)
	}
	if info.Type != reflect.TypeFor[registryTestComponent]() {
		t.Fatal("registered type should match the requested type")
	}

	t.Run("registering the same type twice returns the cached info", func(t *testing.T) {
		again := Register[registryTestComponent]()
		if again != info {
			t.Fatal("expected Register to return the same *ComponentType on repeat calls")
		}
	})
}

func TestResolve(t *testing.T) {
	t.Run("plain struct fields are populated from props", func(t *testing.T) {
		Register[registryTestComponent]()

		comp, err := Resolve("registryTestComponent", map[string]any{"Name": "goblin", "Value": 7})
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		got, ok := comp.(registryTestComponent)
		if !ok {
			t.Fatalf("expected registryTestComponent, got %T", comp)
		}
		if got.Name != "goblin" || got.Value != 7 {
			t.Fatalf("unexpected component values: %#v", got)
		}
	})

	t.Run("Serializable components delegate to Deserialize", func(t *testing.T) {
		Register[registrySerializableComponent]()

		comp, err := Resolve("registrySerializableComponent", map[string]any{"Value": "poisoned"})
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		// Resolve returns the mutated value, not the pointer used internally to
		// call Deserialize - matches GetComponent/SetComponent's value semantics.
		got, ok := comp.(registrySerializableComponent)
		if !ok {
			t.Fatalf("expected registrySerializableComponent, got %T", comp)
		}
		if !got.deserializeCalled {
			t.Fatal("expected Deserialize to be called for a Serializable component")
		}
		if got.Value != "poisoned" {
			t.Fatalf("expected Deserialize to populate Value, got %q", got.Value)
		}
	})

	t.Run("unknown type name returns an error", func(t *testing.T) {
		if _, err := Resolve("nonexistentComponentType", nil); err == nil {
			t.Fatal("expected an error for an unregistered type name")
		}
	})
}

func TestGetTypeInfo(t *testing.T) {
	typ := reflect.TypeFor[registryTestComponent]()
	Register[registryTestComponent]()

	if info := GetTypeInfo(typ); info == nil || info.Type != typ {
		t.Fatalf("expected type info for %v, got %#v", typ, info)
	}

	type neverRegistered struct{}
	if info := GetTypeInfo(reflect.TypeFor[neverRegistered]()); info != nil {
		t.Fatalf("expected nil info for an unregistered type, got %#v", info)
	}
}
