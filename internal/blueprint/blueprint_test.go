package blueprint

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

type testComponent struct {
	Value int
}

func TestBlueprint_Spawn(t *testing.T) {
	core.Register[testComponent]()

	t.Run("spawns an entity with resolved components and tags", func(t *testing.T) {
		world := core.NewWorld()
		bp := &Blueprint{
			Name: "Goblin",
			Components: []ComponentData{
				{Type: "testComponent", Properties: map[string]any{"Value": 5}},
			},
			Tags: []string{"enemy", "hostile"},
		}

		entity, err := bp.Spawn(world)
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
		}

		// core.Resolve returns the resolved value (not a pointer), matching
		// GetComponent/SetComponent's value semantics used everywhere else.
		comp, err := core.GetComponent[testComponent](world, entity.ID)
		if err != nil {
			t.Fatalf("GetComponent failed: %v", err)
		}
		if comp.Value != 5 {
			t.Fatalf("component Value = %d, want 5", comp.Value)
		}
		if !entity.HasTag("enemy") || !entity.HasTag("hostile") {
			t.Fatalf("expected both tags on spawned entity, got %#v", entity.Tags())
		}
	})

	t.Run("an unregistered component type fails the spawn", func(t *testing.T) {
		world := core.NewWorld()
		bp := &Blueprint{
			Name: "Broken",
			Components: []ComponentData{
				{Type: "doesNotExist", Properties: nil},
			},
		}

		if _, err := bp.Spawn(world); err == nil {
			t.Fatal("expected an error for an unregistered component type")
		}
	})
}
