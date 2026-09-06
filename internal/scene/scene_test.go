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

	if scene.entities == nil {
		t.Fatal("expected entities map to be initialized")
	}

	if len(scene.entities) != 0 {
		t.Fatalf("expected empty entities map, got %d items", len(scene.entities))
	}

	if scene.data == nil {
		t.Fatal("expected data map to be initialized")
	}

	if len(scene.data) != 0 {
		t.Fatalf("expected empty data map, got %d items", len(scene.data))
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

	// Check that the entity is now tracked in the scene
	entities := scene.Entities(world)
	found := false
	for _, id := range entities {
		if id == entityID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected entity to be in scene after AddToScene")
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

	// Check that the entity is no longer in the scene
	entities := scene.Entities(world)
	for _, id := range entities {
		if id == entityID {
			t.Fatal("expected entity to not be in scene after RemoveFromScene")
		}
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

func TestNewManager(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	if manager.world != world {
		t.Fatal("expected manager to have the provided world")
	}

	if manager.scenes == nil {
		t.Fatal("expected scenes map to be initialized")
	}

	if len(manager.scenes) != 0 {
		t.Fatalf("expected empty scenes map, got %d scenes", len(manager.scenes))
	}

	if manager.current != "" {
		t.Fatalf("expected empty current scene, got %q", manager.current)
	}
}

func TestManager_LoadScene(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene := NewScene("level-1")

	err := manager.LoadScene("level-1", scene)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manager.scenes) != 1 {
		t.Fatalf("expected 1 scene loaded, got %d", len(manager.scenes))
	}

	if manager.scenes["level-1"] != scene {
		t.Fatal("expected loaded scene to be retrievable")
	}
}

func TestManager_LoadScene_Duplicate(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-1")

	_ = manager.LoadScene("level-1", scene1)

	err := manager.LoadScene("level-1", scene2)
	if err == nil {
		t.Fatal("expected error when loading duplicate scene")
	}

	if err.Error() != "scene level-1 already loaded" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestManager_UnloadScene(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)

	err := manager.UnloadScene("level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manager.scenes) != 0 {
		t.Fatalf("expected 0 scenes after unload, got %d", len(manager.scenes))
	}
}

func TestManager_UnloadScene_NotFound(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	err := manager.UnloadScene("nonexistent")
	if err == nil {
		t.Fatal("expected error when unloading non-existent scene")
	}

	if err.Error() != "scene nonexistent not found" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestManager_UnloadScene_CurrentScene(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo("level-1")

	if manager.current != "level-1" {
		t.Fatal("expected level-1 to be current scene")
	}

	// Unload the current scene
	err := manager.UnloadScene("level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Current scene should be cleared
	if manager.current != "" {
		t.Fatalf("expected current scene to be empty after unloading, got %q", manager.current)
	}
}

func TestManager_UnloadScene_CurrentSceneWithUnloadError(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene := NewScene("level-1")
	expectedErr := errors.New("unload error")
	scene.SetUnloadHook(func(w *core.World) error {
		return expectedErr
	})

	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo("level-1")

	err := manager.UnloadScene("level-1")
	if err == nil {
		t.Fatal("expected error when unloading current scene with failing hook")
	}
}

func TestManager_CurrentScene(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	// No current scene
	if manager.CurrentScene() != nil {
		t.Fatal("expected nil for current scene when none is set")
	}

	// Load and transition to a scene
	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo("level-1")

	if manager.CurrentScene() != scene {
		t.Fatal("expected CurrentScene() to return the current scene")
	}
}

func TestManager_TransitionTo(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	// Transition to level-1
	err := manager.TransitionTo("level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.current != "level-1" {
		t.Fatalf("expected current scene to be 'level-1', got %q", manager.current)
	}

	// Transition to level-2
	err = manager.TransitionTo("level-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.current != "level-2" {
		t.Fatalf("expected current scene to be 'level-2', got %q", manager.current)
	}
}

func TestManager_TransitionTo_NotFound(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	err := manager.TransitionTo("nonexistent")
	if err == nil {
		t.Fatal("expected error when transitioning to non-existent scene")
	}

	if err.Error() != "scene nonexistent not found" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestManager_TransitionTo_UnloadCurrentError(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	expectedErr := errors.New("unload error")
	scene1.SetUnloadHook(func(w *core.World) error {
		return expectedErr
	})

	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)
	_ = manager.TransitionTo("level-1")

	// Transition to level-2 should fail because level-1 unload fails
	err := manager.TransitionTo("level-2")
	if err == nil {
		t.Fatal("expected error when transitioning with failing unload")
	}
}

func TestManager_TransitionTo_LoadError(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	expectedErr := errors.New("load error")
	scene2.SetLoadHook(func(w *core.World) error {
		return expectedErr
	})

	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	// Transition to level-2 should fail because load fails
	err := manager.TransitionTo("level-2")
	if err == nil {
		t.Fatal("expected error when transitioning with failing load")
	}
}

func TestManager_TransitionTo_FromEmpty(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)

	// Transition from no current scene to level-1
	err := manager.TransitionTo("level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.current != "level-1" {
		t.Fatalf("expected current scene to be 'level-1', got %q", manager.current)
	}
}

func TestManager_Scenes(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")
	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	scenes := manager.Scenes()
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
	if scenes["level-1"] != scene1 || scenes["level-2"] != scene2 {
		t.Fatal("expected scenes map to contain loaded scenes")
	}
}

func TestManager_Current(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	if manager.Current() != nil {
		t.Fatal("expected nil for current scene when none is set")
	}

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo("level-1")

	if manager.Current() != scene {
		t.Fatal("expected Current() to return the current scene")
	}
}

func TestManager_SceneBuilder(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	builder := manager.SceneBuilder("test-scene")
	if builder == nil {
		t.Fatal("expected SceneBuilder to return a builder")
	}

	// SceneBuilder should return the same builder instance on subsequent calls
	builder2 := manager.SceneBuilder("test-scene")
	if builder != builder2 {
		t.Fatal("expected SceneBuilder to return the same builder instance")
	}
}
