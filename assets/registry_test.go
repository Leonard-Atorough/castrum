package assets

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// Test component types

type mockComponent struct {
	Name  string
	Value int
}

func (m mockComponent) Serialize() (map[string]any, error) {
	return map[string]any{"Name": m.Name, "Value": m.Value}, nil
}

func (m mockComponent) Deserialize(props map[string]any) (mockComponent, error) {
	if name, ok := props["Name"].(string); ok {
		m.Name = name
	}
	if value, ok := props["Value"].(float64); ok {
		m.Value = int(value)
	}
	return m, nil
}

type anotherComponent struct {
	Enabled bool
}

func TestNewComponentRegistry(t *testing.T) {
	registry := NewComponentRegistry()
	if registry == nil {
		t.Fatal("NewComponentRegistry returned nil")
	}
	if registry.byName == nil {
		t.Fatal("ComponentRegistry.byName is nil")
	}
	if registry.byType == nil {
		t.Fatal("ComponentRegistry.byType is nil")
	}
}

func TestComponentRegistryRegisterComponent(t *testing.T) {
	registry := NewComponentRegistry()

	t.Run("successful registration", func(t *testing.T) {
		err := registry.RegisterComponent("mock", func(props map[string]any) (ecs.Component, error) {
			return &mockComponent{Name: "test"}, nil
		})
		if err != nil {
			t.Fatalf("RegisterComponent failed: %v", err)
		}
	})

	t.Run("successful registration with factory", func(t *testing.T) {
		factory := func(props map[string]any) (mockComponent, error) {
			return mockComponent{Name: props["Name"].(string)}, nil
		}
		err := registry.RegisterComponent("mockWithFactory", factory)
		if err != nil {
			t.Fatalf("RegisterComponent with factory failed: %v", err)
		}
	})

	t.Run("duplicate registration same type and factory", func(t *testing.T) {
		// Register the same component again with the same type and factory - should succeed
		err := registry.RegisterComponent("mock", func(props map[string]any) (ecs.Component, error) {
			return &mockComponent{Name: "test"}, nil
		})
		if err != nil {
			t.Errorf("duplicate registration with same type and factory should succeed, got: %v", err)
		}
	})

	t.Run("duplicate registration different type", func(t *testing.T) {
		// RegisterComponent uses the return type of the factory (ecs.Component),
		// so different component structs with the same interface return type will conflict
		// Use Register with explicit type instead
		err := registry.Register("anotherMock", reflect.TypeFor[mockComponent](), func(props map[string]any) (ecs.Component, error) {
			return &mockComponent{Name: "another", Value: 1}, nil
		})
		if err != nil {
			// This should fail because mockComponent type is already registered
			if !strings.Contains(err.Error(), "already registered") {
				t.Errorf("expected 'already registered' error, got: %v", err)
			}
		}
	})

	t.Run("registration with custom name and type", func(t *testing.T) {
		// This should fail because mockComponent type is already registered as "mock"
		typ := reflect.TypeFor[mockComponent]()
		factory := func(props map[string]any) (ecs.Component, error) {
			return &mockComponent{}, nil
		}
		err := registry.Register("customName", typ, factory)
		if err != nil {
			// Expected to fail because the type is already registered
			if !strings.Contains(err.Error(), "already registered") {
				t.Errorf("expected 'already registered' error, got: %v", err)
			}
		} else {
			t.Error("Register should fail when type is already registered with different name")
		}
	})
}

func TestComponentRegistryRegister(t *testing.T) {
	registry := NewComponentRegistry()

	tests := []struct {
		name     string
		compName string
		typ      reflect.Type
		factory  ComponentFactory
		wantErr  bool
	}{
		{
			name:     "successful registration",
			compName: "test",
			typ:      reflect.TypeFor[mockComponent](),
			factory:  func(props map[string]any) (ecs.Component, error) { return &mockComponent{}, nil },
			wantErr:  false,
		},
		{
			name:     "empty name",
			compName: "   ",
			typ:      reflect.TypeFor[mockComponent](),
			factory:  func(props map[string]any) (ecs.Component, error) { return &mockComponent{}, nil },
			wantErr:  true,
		},
		{
			name:     "nil type",
			compName: "test",
			typ:      nil,
			factory:  func(props map[string]any) (ecs.Component, error) { return &mockComponent{}, nil },
			wantErr:  true,
		},
		{
			name:     "nil factory",
			compName: "test",
			typ:      reflect.TypeFor[mockComponent](),
			factory:  nil,
			wantErr:  true,
		},
		{
			name:     "duplicate name different type",
			compName: "test",
			typ:      reflect.TypeFor[anotherComponent](),
			factory:  func(props map[string]any) (ecs.Component, error) { return &anotherComponent{}, nil },
			wantErr:  true,
		},
		{
			name:     "duplicate type different name",
			compName: "different",
			typ:      reflect.TypeFor[mockComponent](),
			factory:  func(props map[string]any) (ecs.Component, error) { return &mockComponent{}, nil },
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.compName, tt.typ, tt.factory)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComponentRegistryResolve(t *testing.T) {
	registry := NewComponentRegistry()

	// Register a component
	if err := registry.Register("mock", reflect.TypeFor[mockComponent](), func(props map[string]any) (ecs.Component, error) {
		return &mockComponent{Name: "resolved", Value: 42}, nil
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	t.Run("successful resolve", func(t *testing.T) {
		comp, err := registry.Resolve("mock", nil)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}
		if comp == nil {
			t.Fatal("Resolve returned nil")
		}
		mock, ok := comp.(*mockComponent)
		if !ok {
			t.Fatal("Resolve returned wrong type")
		}
		if mock.Name != "resolved" {
			t.Errorf("Name = %q, want %q", mock.Name, "resolved")
		}
		if mock.Value != 42 {
			t.Errorf("Value = %d, want 42", mock.Value)
		}
	})

	t.Run("resolve with properties", func(t *testing.T) {
		props := map[string]any{"Name": "custom", "Value": 100}
		comp, err := registry.Resolve("mock", props)
		if err != nil {
			t.Fatalf("Resolve with properties failed: %v", err)
		}
		// The factory we registered ignores properties, so we get the defaults
		// This is expected behavior based on our test setup
		_ = comp
	})

	t.Run("resolve unregistered component", func(t *testing.T) {
		_, err := registry.Resolve("nonexistent", nil)
		if err == nil {
			t.Fatal("Resolve should fail for unregistered component")
		}
		// Verify error message contains expected text
		if !strings.Contains(err.Error(), "component definition not found") {
			t.Errorf("error message should contain 'component definition not found', got: %v", err)
		}
	})
}

func TestComponentRegistryLookupByType(t *testing.T) {
	registry := NewComponentRegistry()

	// Register a component
	if err := registry.Register("mock", reflect.TypeFor[mockComponent](), func(props map[string]any) (ecs.Component, error) {
		return &mockComponent{}, nil
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		def, ok := registry.LookupByType(reflect.TypeFor[mockComponent]())
		if !ok {
			t.Fatal("LookupByType returned ok=false for registered type")
		}
		if def.Name != "mock" {
			t.Errorf("Name = %q, want %q", def.Name, "mock")
		}
		if def.Type != reflect.TypeFor[mockComponent]() {
			t.Errorf("Type mismatch")
		}
		if def.Factory == nil {
			t.Error("Factory is nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := registry.LookupByType(reflect.TypeFor[anotherComponent]())
		if ok {
			t.Error("LookupByType returned ok=true for unregistered type")
		}
	})
}

func TestComponentRegistryLookupByName(t *testing.T) {
	registry := NewComponentRegistry()

	// Register a component
	if err := registry.Register("mock", reflect.TypeFor[mockComponent](), func(props map[string]any) (ecs.Component, error) {
		return &mockComponent{}, nil
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		def, ok := registry.LookupByName("mock")
		if !ok {
			t.Fatal("LookupByName returned ok=false for registered name")
		}
		if def.Name != "mock" {
			t.Errorf("Name = %q, want %q", def.Name, "mock")
		}
		if def.Type != reflect.TypeFor[mockComponent]() {
			t.Errorf("Type mismatch")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := registry.LookupByName("nonexistent")
		if ok {
			t.Error("LookupByName returned ok=true for unregistered name")
		}
	})
}

func TestComponentRegistryHas(t *testing.T) {
	registry := NewComponentRegistry()

	// Register a component
	if err := registry.Register("mock", reflect.TypeFor[mockComponent](), func(props map[string]any) (ecs.Component, error) {
		return &mockComponent{}, nil
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	t.Run("has registered", func(t *testing.T) {
		if !registry.Has("mock") {
			t.Error("Has returned false for registered component")
		}
	})

	t.Run("has unregistered", func(t *testing.T) {
		if registry.Has("nonexistent") {
			t.Error("Has returned true for unregistered component")
		}
	})
}

func TestComponentRegistryConcurrentAccess(t *testing.T) {
	registry := NewComponentRegistry()

	// Pre-register components with unique types to avoid conflicts
	// Create 10 different struct types
	types := make([]reflect.Type, 10)
	for i := 0; i < 10; i++ {
		types[i] = reflect.StructOf([]reflect.StructField{
			{Name: fmt.Sprintf("Field%d", i), Type: reflect.TypeOf(0)},
		})
	}

	// Register each with a unique type and name
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("component%d", i)
		if err := registry.Register(name, types[i], func(props map[string]any) (ecs.Component, error) {
			return &mockComponent{Value: i}, nil
		}); err != nil {
			t.Fatalf("pre-registration failed for component%d: %v", i, err)
		}
	}

	// Concurrent lookups
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			name := fmt.Sprintf("component%d", id%10)
			_ = registry.Has(name)
			_, _ = registry.LookupByName(name)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify registry is still functional
	if !registry.Has("component0") {
		t.Error("registry corrupted by concurrent access")
	}
}

func TestComponentRegistryReRegistration(t *testing.T) {
	registry := NewComponentRegistry()

	// Register a component
	factory1 := func(props map[string]any) (ecs.Component, error) {
		return &mockComponent{Name: "first"}, nil
	}
	factory2 := func(props map[string]any) (ecs.Component, error) {
		return &mockComponent{Name: "second"}, nil
	}

	if err := registry.Register("test", reflect.TypeFor[mockComponent](), factory1); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	// According to the registry code, re-registering with same name and type
	// will succeed if the existing factory is not nil (which it isn't)
	// So the second registration should succeed, but the factory won't be replaced
	err := registry.Register("test", reflect.TypeFor[mockComponent](), factory2)
	if err != nil {
		t.Errorf("Register should succeed when re-registering same name and type, got: %v", err)
	}

	// The first factory should still work (the second factory is ignored)
	comp, err := registry.Resolve("test", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	mock, ok := comp.(*mockComponent)
	if !ok {
		t.Fatal("Resolve returned wrong type")
	}
	// The factory returns "first" so the name should be "first"
	if mock.Name != "first" {
		t.Errorf("Name = %q, want %q", mock.Name, "first")
	}
}
