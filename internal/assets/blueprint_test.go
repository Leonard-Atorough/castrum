package assets

import (
	"testing"
	"testing/fstest"

	"github.com/leonard-atorough/castrum/internal/ecs"
)

type testComponent struct {
	Value int
}

func TestBlueprint_Spawn(t *testing.T) {
	ecs.Register[testComponent]()

	t.Run("spawns an entity with resolved components", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &blueprint{
			Name: "Goblin",
			Components: []componentData{
				{Type: "testComponent", Properties: map[string]any{"Value": 5}},
			},
		}

		entity, err := bp.spawn(world)
		if err != nil {
			t.Fatalf("Spawn failed: %v", err)
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
		bp := &blueprint{
			Name: "Broken",
			Components: []componentData{
				{Type: "doesNotExist", Properties: nil},
			},
		}

		if _, err := bp.spawn(world); err == nil {
			t.Fatal("expected an error for an unregistered component type")
		}
	})
}

const validBlueprintYAML = `name: Goblin
version: "1.0"
components:
  - type: testComponent
    properties:
      Value: 5
`

const invalidBlueprintYAML = "not: [valid"

func TestStore_Load(t *testing.T) {
	t.Run("parses a valid blueprint and caches it by path", func(t *testing.T) {
		fs := fstest.MapFS{
			"goblin.yaml": {Data: []byte(validBlueprintYAML)},
		}
		s := newBlueprintStore(fs)

		bp, err := s.Load("goblin.yaml")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if bp.Name != "Goblin" {
			t.Fatalf("Name = %q, want %q", bp.Name, "Goblin")
		}
		if len(bp.Components) != 1 || bp.Components[0].Type != "testComponent" {
			t.Fatalf("unexpected components: %#v", bp.Components)
		}

		// Verify caching by path
		got, exists := s.Blueprints["goblin.yaml"]
		if !exists || got != bp {
			t.Fatal("expected Load to cache the blueprint under its path")
		}
	})

	t.Run("invalid YAML returns an error", func(t *testing.T) {
		fs := fstest.MapFS{
			"broken.yaml": {Data: []byte(invalidBlueprintYAML)},
		}
		s := newBlueprintStore(fs)

		if _, err := s.Load("broken.yaml"); err == nil {
			t.Fatal("expected an error for malformed YAML")
		}
	})

	t.Run("a missing file returns an error", func(t *testing.T) {
		fs := fstest.MapFS{}
		s := newBlueprintStore(fs)

		if _, err := s.Load("missing.yaml"); err == nil {
			t.Fatal("expected an error for a missing file")
		}
	})

	t.Run("caches result on repeated calls", func(t *testing.T) {
		fs := fstest.MapFS{
			"hero.yaml": {Data: []byte(validBlueprintYAML)},
		}
		s := newBlueprintStore(fs)

		bp1, _ := s.Load("hero.yaml")
		bp2, _ := s.Load("hero.yaml")

		if bp1 != bp2 {
			t.Fatal("expected Load to return cached blueprint on second call")
		}
	})
}
