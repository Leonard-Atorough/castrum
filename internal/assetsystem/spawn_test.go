package assetsystem

import (
	"testing"

	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

type testComponent struct {
	Value int
}

func TestCreateFromBlueprint(t *testing.T) {
	ecs.Register[testComponent]()

	t.Run("spawns an entity with resolved components", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &assets.Blueprint{
			Name: "Goblin",
			Components: []assets.ComponentData{
				{Type: "testComponent", Properties: map[string]any{"Value": 5}},
			},
		}

		entity, err := CreateFromBlueprint(world, bp)
		if err != nil {
			t.Fatalf("CreateFromBlueprint failed: %v", err)
		}

		// ecs.Resolve returns the resolved value (not a pointer), matching
		// GetComponent/SetComponent's value semantics used everywhere else.
		comp, err := world.GetComponent[testComponent](entity.ID)
		if err != nil {
			t.Fatalf("GetComponent failed: %v", err)
		}
		if comp.Value != 5 {
			t.Fatalf("component Value = %d, want 5", comp.Value)
		}
	})

	t.Run("an unregistered component type fails the spawn", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &assets.Blueprint{
			Name: "Broken",
			Components: []assets.ComponentData{
				{Type: "doesNotExist", Properties: nil},
			},
		}

		if _, err := CreateFromBlueprint(world, bp); err == nil {
			t.Fatal("expected an error for an unregistered component type")
		}
	})
}
