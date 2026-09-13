package assets

import (
	"testing"
	"testing/fstest"

	"github.com/leonard-atorough/castrum/ecs"
)

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
			t.Errorf("Load failed: %v", err)
		}
		if bp.Name != "Goblin" {
			t.Errorf("Name = %q, want %q", bp.Name, "Goblin")
		}
		if len(bp.Components) != 1 || bp.Components[0].Type != "testComponent" {
			t.Errorf("unexpected components: %#v", bp.Components)
		}

		// Verify caching by path
		got, exists := s.Blueprints["goblin.yaml"]
		if !exists || got != bp {
			t.Errorf("expected Load to cache the blueprint under its path")
		}
	})

	t.Run("invalid YAML returns an error", func(t *testing.T) {
		fs := fstest.MapFS{
			"broken.yaml": {Data: []byte(invalidBlueprintYAML)},
		}
		s := newBlueprintStore(fs)

		if _, err := s.Load("broken.yaml"); err == nil {
			t.Errorf("expected an error for malformed YAML")
		}
	})

	t.Run("a missing file returns an error", func(t *testing.T) {
		fs := fstest.MapFS{}
		s := newBlueprintStore(fs)

		if _, err := s.Load("missing.yaml"); err == nil {
			t.Errorf("expected an error for a missing file")
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
			t.Errorf("expected Load to return cached blueprint on second call")
		}
	})
}

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
		if err := first.RegisterComponent[testComponent]("testComponent", func(props map[string]any) (testComponent, error) {
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
