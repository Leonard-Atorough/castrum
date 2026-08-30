package scene

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	if builder.scene.ID != "test-scene" {
		t.Fatalf("expected scene ID 'test-scene', got %q", builder.scene.ID)
	}

	if len(builder.entityIDs) != 0 {
		t.Fatalf("expected empty entity IDs, got %d", len(builder.entityIDs))
	}
}

func TestBuilder_WithEntity(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	entityID := core.EntityID(1)
	result := builder.WithEntity(entityID)

	if result != builder {
		t.Fatal("WithEntity should return the same builder for chaining")
	}

	if len(builder.entityIDs) != 1 {
		t.Fatalf("expected 1 entity ID, got %d", len(builder.entityIDs))
	}

	if builder.entityIDs[0] != entityID {
		t.Fatalf("expected entity ID %d, got %d", entityID, builder.entityIDs[0])
	}
}

func TestBuilder_WithEntity_Multiple(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	builder.WithEntity(1).WithEntity(2).WithEntity(3)

	if len(builder.entityIDs) != 3 {
		t.Fatalf("expected 3 entity IDs, got %d", len(builder.entityIDs))
	}
}

func TestBuilder_Build(t *testing.T) {
	world := core.NewWorld()
	builder := NewBuilder("test-scene", world)

	entity1 := world.CreateEntity("player")
	entity2 := world.CreateEntity("enemy")

	builder.WithEntity(entity1.ID).WithEntity(entity2.ID)

	scene, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if scene.ID != "test-scene" {
		t.Fatalf("expected scene ID 'test-scene', got %q", scene.ID)
	}

	// Verify entities are in the scene
	entities := scene.Entities(world)
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities in scene, got %d", len(entities))
	}
}

func TestBuilder_Build_Empty(t *testing.T) {
	world := core.NewWorld()
	builder := NewBuilder("empty-scene", world)

	scene, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if scene.ID != "empty-scene" {
		t.Fatalf("expected scene ID 'empty-scene', got %q", scene.ID)
	}

	entities := scene.Entities(world)
	if len(entities) != 0 {
		t.Fatalf("expected 0 entities in empty scene, got %d", len(entities))
	}
}

func TestBuilder_Build_WithNonExistentEntity(t *testing.T) {
	world := core.NewWorld()
	builder := NewBuilder("test-scene", world)

	builder.WithEntity(999) // Non-existent entity

	_, err := builder.Build()
	if err == nil {
		t.Fatal("expected error when building with non-existent entity")
	}
}

func TestBuilder_WithLoadHook(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	loadCalled := false
	builder.WithLoadHook(func(w *core.World) error {
		loadCalled = true
		return nil
	})

	if builder.scene.loadHook == nil {
		t.Fatal("expected load hook to be set")
	}

	// Verify the hook works
	world := core.NewWorld()
	scene, _ := builder.Build()
	_ = scene.OnLoad(world)

	if !loadCalled {
		t.Fatal("expected load hook to be called")
	}
}

func TestBuilder_WithUnloadHook(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	unloadCalled := false
	builder.WithUnloadHook(func(w *core.World) error {
		unloadCalled = true
		return nil
	})

	if builder.scene.unloadHook == nil {
		t.Fatal("expected unload hook to be set")
	}

	// Verify the hook works
	world := core.NewWorld()
	scene, _ := builder.Build()
	_ = scene.OnUnload(world)

	if !unloadCalled {
		t.Fatal("expected unload hook to be called")
	}
}

func TestBuilder_WithHooks(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	loadCalled, unloadCalled := false, false
	builder.WithHooks(
		func(w *core.World) error {
			loadCalled = true
			return nil
		},
		func(w *core.World) error {
			unloadCalled = true
			return nil
		},
	)

	world := core.NewWorld()
	scene, _ := builder.Build()

	_ = scene.OnLoad(world)
	if !loadCalled {
		t.Fatal("expected load hook to be called")
	}

	_ = scene.OnUnload(world)
	if !unloadCalled {
		t.Fatal("expected unload hook to be called")
	}
}

func TestBuilder_WithData(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	builder.WithData("score", 100)
	builder.WithData("level", 5)

	scene := builder.Scene()

	val, ok := scene.GetData("score")
	if !ok || val != 100 {
		t.Fatal("expected score to be 100")
	}

	val, ok = scene.GetData("level")
	if !ok || val != 5 {
		t.Fatal("expected level to be 5")
	}
}

func TestBuilder_WithDataMap(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	data := map[string]any{
		"health": 100,
		"mana":   50,
		"name":   "player",
	}
	builder.WithDataMap(data)

	scene := builder.Scene()

	for k, v := range data {
		val, ok := scene.GetData(k)
		if !ok {
			t.Fatalf("expected key %q to exist", k)
		}
		if val != v {
			t.Fatalf("expected value %v for key %q, got %v", v, k, val)
		}
	}
}

func TestBuilder_Scene(t *testing.T) {
	builder := NewBuilder("test-scene", core.NewWorld())

	scene := builder.Scene()

	if scene.ID != "test-scene" {
		t.Fatalf("expected scene ID 'test-scene', got %q", scene.ID)
	}

	// Verify we can still build after getting the scene
	builtScene, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if builtScene != scene {
		t.Fatal("expected Scene() to return the same scene instance")
	}
}
