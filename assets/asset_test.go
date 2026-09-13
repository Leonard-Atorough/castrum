package assets

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

type testComponent struct {
	Value int
}

func TestCreateFromBlueprint(t *testing.T) {
	registry := NewComponentRegistry()
	if err := registry.RegisterComponent[testComponent]("testComponent"); err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	t.Run("spawns an entity with resolved components", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "Goblin",
			Components: []ComponentData{
				{Type: "testComponent", Properties: map[string]any{"Value": 5}},
			},
		}

		entity, err := CreateFromBlueprint(world, registry, bp)
		if err != nil {
			t.Errorf("CreateFromBlueprint failed: %v", err)
		}

		comp, err := world.GetComponent[testComponent](entity.ID)
		if err != nil {
			t.Errorf("GetComponent failed: %v", err)
		}
		if comp.Value != 5 {
			t.Errorf("component Value = %d, want 5", comp.Value)
		}
	})

	t.Run("an unregistered component type fails the spawn", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "Broken",
			Components: []ComponentData{
				{Type: "doesNotExist", Properties: nil},
			},
		}

		if _, err := CreateFromBlueprint(world, registry, bp); err == nil {
			t.Errorf("expected an error for an unregistered component type")
		}
	})

	t.Run("registrations are scoped to a world", func(t *testing.T) {
		first := NewComponentRegistry()
		second := NewComponentRegistry()
		if err := first.RegisterComponent("testComponent", func(props map[string]any) (testComponent, error) {
			return testComponent{Value: 42}, nil
		}); err != nil {
			t.Fatalf("RegisterComponent failed: %v", err)
		}

		resolved, err := first.Resolve("testComponent", nil)
		if err != nil || resolved.(testComponent).Value != 42 {
			t.Fatalf("world registration did not resolve correctly: %#v, %v", resolved, err)
		}
		if _, err := second.Resolve("testComponent", nil); err == nil {
			t.Fatal("component registration leaked between registries")
		}
	})
}
