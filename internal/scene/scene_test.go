package scene

import (
	"errors"
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

func TestNewScene(t *testing.T) {
	scene := NewScene("test-scene")

	if scene.ID != "test-scene" {
		t.Fatalf("expected ID 'test-scene', got %q", scene.ID)
	}

	if scene.Name() != "test-scene" {
		t.Fatalf("expected Name() 'test-scene', got %q", scene.Name())
	}

	if scene.tag != "scene:test-scene" {
		t.Fatalf("expected internal tag 'scene:test-scene', got %q", scene.tag)
	}

	if scene.data == nil {
		t.Fatal("expected data map to be initialized")
	}

	if len(scene.data) != 0 {
		t.Fatalf("expected empty data map, got %d items", len(scene.data))
	}
}

func TestScene_Name(t *testing.T) {
	scene := NewScene("level-1")

	if scene.Name() != "level-1" {
		t.Fatalf("expected Name() to return 'level-1', got %q", scene.Name())
	}
}

func TestScene_AddToScene(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	entity := world.Create("player")
	entityID := entity.ID

	err := scene.AddToScene(entityID, world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasTag := entity.HasTag("scene:test")
	if !hasTag {
		t.Fatal("expected entity to have scene tag after AddToScene")
	}
}

func TestScene_AddToScene_NonExistentEntity(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	err := scene.AddToScene(999, world)
	if err == nil {
		t.Fatal("expected error when adding non-existent entity")
	}
}

func TestScene_RemoveFromScene(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	entity := world.Create("player")
	entityID := entity.ID
	_ = scene.AddToScene(entityID, world)

	err := scene.RemoveFromScene(entityID, world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasTag := entity.HasTag("scene:test")
	if hasTag {
		t.Fatal("expected entity to not have scene tag after RemoveFromScene")
	}
}

func TestScene_RemoveFromScene_NonExistentEntity(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	err := scene.RemoveFromScene(999, world)
	if err == nil {
		t.Fatal("expected error when removing non-existent entity")
	}
}

func TestScene_Entities(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	entity1 := world.Create("player")
	entity2 := world.Create("enemy")
	_ = world.Create("npc")

	_ = scene.AddToScene(entity1.ID, world)
	_ = scene.AddToScene(entity2.ID, world)

	entities := scene.Entities(world)
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}

	found1, found2 := false, false
	for _, id := range entities {
		if id == entity1.ID {
			found1 = true
		}
		if id == entity2.ID {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Fatal("expected both entities to be in the scene")
	}
}

func TestScene_Entities_Empty(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("empty-scene")

	entities := scene.Entities(world)
	if len(entities) != 0 {
		t.Fatalf("expected 0 entities for empty scene, got %d", len(entities))
	}
}

func TestScene_SetGetData(t *testing.T) {
	scene := NewScene("test")

	scene.SetData("score", 100)
	val, ok := scene.GetData("score")
	if !ok {
		t.Fatal("expected to find 'score' in data")
	}
	if val != 100 {
		t.Fatalf("expected score 100, got %v", val)
	}

	_, ok = scene.GetData("nonexistent")
	if ok {
		t.Fatal("expected 'nonexistent' key to not exist")
	}
}

func TestScene_OnLoad_WithHook(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	loadCalled := false
	scene.SetLoadHook(func(w *core.World) error {
		loadCalled = true
		return nil
	})

	err := scene.OnLoad(world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !loadCalled {
		t.Fatal("expected load hook to be called")
	}
}

func TestScene_OnLoad_HookError(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	expectedErr := errors.New("load failed")
	scene.SetLoadHook(func(w *core.World) error {
		return expectedErr
	})

	err := scene.OnLoad(world)
	if err == nil {
		t.Fatal("expected error from load hook")
	}
	if err.Error() != "failed to execute load hook for scene test: load failed" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestScene_OnLoad_NoHook(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	err := scene.OnLoad(world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScene_OnUnload_WithEntities(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	entity1 := world.Create("player")
	entity2 := world.Create("enemy")
	_ = scene.AddToScene(entity1.ID, world)
	_ = scene.AddToScene(entity2.ID, world)

	if len(scene.Entities(world)) != 2 {
		t.Fatal("expected 2 entities in scene before unload")
	}

	err := scene.OnUnload(world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(scene.Entities(world)) != 0 {
		t.Fatal("expected 0 entities in scene after unload")
	}
}

func TestScene_OnUnload_WithHook(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	unloadCalled := false
	scene.SetUnloadHook(func(w *core.World) error {
		unloadCalled = true
		return nil
	})

	err := scene.OnUnload(world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unloadCalled {
		t.Fatal("expected unload hook to be called")
	}
}

func TestScene_OnUnload_HookError(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("test")

	entity1 := world.Create("player")
	_ = scene.AddToScene(entity1.ID, world)

	expectedErr := errors.New("unload failed")
	scene.SetUnloadHook(func(w *core.World) error {
		return expectedErr
	})

	err := scene.OnUnload(world)
	if err == nil {
		t.Fatal("expected error from unload hook")
	}
	if err.Error() != "failed to execute unload hook for scene test: unload failed" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestScene_OnUnload_WithBothEntitiesAndHook(t *testing.T) {
	// Test that OnUnload properly removes entities AND calls the unload hook
	// Note: entities are removed BEFORE the unload hook is called
	world := core.NewWorld()
	scene := NewScene("test")

	entity1 := world.Create("player")
	entity2 := world.Create("enemy")
	_ = scene.AddToScene(entity1.ID, world)
	_ = scene.AddToScene(entity2.ID, world)

	unloadCalled := false
	scene.SetUnloadHook(func(w *core.World) error {
		unloadCalled = true
		// Verify entities are already removed from scene when hook is called
		// (this is the actual behavior - entities are cleaned up first)
		entities := scene.Entities(world)
		if len(entities) != 0 {
			t.Errorf("expected 0 entities during unload hook (entities removed before hook), got %d", len(entities))
		}
		return nil
	})

	err := scene.OnUnload(world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !unloadCalled {
		t.Fatal("expected unload hook to be called")
	}

	// Verify entities are removed from scene
	entities := scene.Entities(world)
	if len(entities) != 0 {
		t.Fatalf("expected 0 entities after unload, got %d", len(entities))
	}
}
