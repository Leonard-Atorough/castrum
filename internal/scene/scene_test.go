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

func TestScene_Entities_MultipleScenesDoNotOverlap(t *testing.T) {
	world := core.NewWorld()
	sceneA := NewScene("a")
	sceneB := NewScene("b")

	entity1 := world.Create("player")
	entity2 := world.Create("enemy")

	_ = sceneA.AddToScene(entity1.ID, world)
	_ = sceneB.AddToScene(entity2.ID, world)

	if entities := sceneA.Entities(world); len(entities) != 1 || entities[0] != entity1.ID {
		t.Fatalf("expected scene a to contain only entity1, got %v", entities)
	}
	if entities := sceneB.Entities(world); len(entities) != 1 || entities[0] != entity2.ID {
		t.Fatalf("expected scene b to contain only entity2, got %v", entities)
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
	// Entities are untagged BEFORE the unload hook is called.
	world := core.NewWorld()
	scene := NewScene("test")

	entity1 := world.Create("player")
	entity2 := world.Create("enemy")
	_ = scene.AddToScene(entity1.ID, world)
	_ = scene.AddToScene(entity2.ID, world)

	unloadCalled := false
	scene.SetUnloadHook(func(w *core.World) error {
		unloadCalled = true
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

	entities := scene.Entities(world)
	if len(entities) != 0 {
		t.Fatalf("expected 0 entities after unload, got %d", len(entities))
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager.scenes == nil {
		t.Fatal("expected scenes map to be initialized")
	}

	if len(manager.scenes) != 0 {
		t.Fatalf("expected empty scenes map, got %d scenes", len(manager.scenes))
	}

	if len(manager.stack) != 0 {
		t.Fatalf("expected empty stack, got %d entries", len(manager.stack))
	}
}

func TestManager_LoadScene(t *testing.T) {
	manager := NewManager()

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
	manager := NewManager()

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
	manager := NewManager()

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)

	err := manager.UnloadScene(world, "level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manager.scenes) != 0 {
		t.Fatalf("expected 0 scenes after unload, got %d", len(manager.scenes))
	}
}

func TestManager_UnloadScene_NotFound(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	err := manager.UnloadScene(world, "nonexistent")
	if err == nil {
		t.Fatal("expected error when unloading non-existent scene")
	}

	if err.Error() != "scene nonexistent not found" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestManager_UnloadScene_CurrentScene(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo(world, "level-1")

	if manager.CurrentScene() != scene {
		t.Fatal("expected level-1 to be current scene")
	}

	err := manager.UnloadScene(world, "level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.CurrentScene() != nil {
		t.Fatal("expected current scene to be nil after unloading")
	}
}

func TestManager_UnloadScene_CurrentSceneWithUnloadError(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene := NewScene("level-1")
	expectedErr := errors.New("unload error")
	scene.SetUnloadHook(func(w *core.World) error {
		return expectedErr
	})

	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo(world, "level-1")

	err := manager.UnloadScene(world, "level-1")
	if err == nil {
		t.Fatal("expected error when unloading current scene with failing hook")
	}
}

func TestManager_CurrentScene(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	if manager.CurrentScene() != nil {
		t.Fatal("expected nil for current scene when none is set")
	}

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo(world, "level-1")

	if manager.CurrentScene() != scene {
		t.Fatal("expected CurrentScene() to return the current scene")
	}
}

func TestManager_TransitionTo(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	err := manager.TransitionTo(world, "level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.CurrentScene() != scene1 {
		t.Fatal("expected current scene to be level-1")
	}

	err = manager.TransitionTo(world, "level-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.CurrentScene() != scene2 {
		t.Fatal("expected current scene to be level-2")
	}

	if len(manager.Stack()) != 1 {
		t.Fatalf("expected stack of size 1 after transition, got %d", len(manager.Stack()))
	}
}

func TestManager_TransitionTo_NotFound(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	err := manager.TransitionTo(world, "nonexistent")
	if err == nil {
		t.Fatal("expected error when transitioning to non-existent scene")
	}

	if err.Error() != "scene nonexistent not found" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestManager_TransitionTo_UnloadCurrentError(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	expectedErr := errors.New("unload error")
	scene1.SetUnloadHook(func(w *core.World) error {
		return expectedErr
	})

	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)
	_ = manager.TransitionTo(world, "level-1")

	err := manager.TransitionTo(world, "level-2")
	if err == nil {
		t.Fatal("expected error when transitioning with failing unload")
	}
}

func TestManager_TransitionTo_LoadError(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	expectedErr := errors.New("load error")
	scene2.SetLoadHook(func(w *core.World) error {
		return expectedErr
	})

	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	err := manager.TransitionTo(world, "level-2")
	if err == nil {
		t.Fatal("expected error when transitioning with failing load")
	}
}

func TestManager_TransitionTo_FromEmpty(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)

	err := manager.TransitionTo(world, "level-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.CurrentScene() != scene {
		t.Fatal("expected current scene to be level-1")
	}
}

func TestManager_Scenes(t *testing.T) {
	manager := NewManager()

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
	manager := NewManager()

	if manager.Current() != nil {
		t.Fatal("expected nil for current scene when none is set")
	}

	scene := NewScene("level-1")
	_ = manager.LoadScene("level-1", scene)
	_ = manager.TransitionTo(world, "level-1")

	if manager.Current() != scene {
		t.Fatal("expected Current() to return the current scene")
	}
}

func TestManager_SceneBuilder(t *testing.T) {
	manager := NewManager()

	builder := manager.SceneBuilder("test-scene")
	if builder == nil {
		t.Fatal("expected SceneBuilder to return a builder")
	}

	builder2 := manager.SceneBuilder("test-scene")
	if builder != builder2 {
		t.Fatal("expected SceneBuilder to return the same builder instance")
	}
}

func TestManager_Push_Pop(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	gameplay := NewScene("gameplay")
	pause := NewScene("pause")
	_ = manager.LoadScene("gameplay", gameplay)
	_ = manager.LoadScene("pause", pause)

	if err := manager.Push(world, "gameplay"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.Current() != gameplay {
		t.Fatal("expected gameplay to be current after push")
	}

	if err := manager.Push(world, "pause"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.Current() != pause {
		t.Fatal("expected pause to be current after push")
	}
	if len(manager.Stack()) != 2 {
		t.Fatalf("expected stack size 2, got %d", len(manager.Stack()))
	}

	if err := manager.Pop(world); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.Current() != gameplay {
		t.Fatal("expected gameplay to be current after popping pause")
	}
	if len(manager.Stack()) != 1 {
		t.Fatalf("expected stack size 1 after pop, got %d", len(manager.Stack()))
	}
}

func TestManager_Push_NotFound(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	err := manager.Push(world, "nonexistent")
	if err == nil {
		t.Fatal("expected error when pushing non-existent scene")
	}
}

func TestManager_Push_LoadErrorRollsBackStack(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	scene := NewScene("broken")
	expectedErr := errors.New("load error")
	scene.SetLoadHook(func(w *core.World) error {
		return expectedErr
	})
	_ = manager.LoadScene("broken", scene)

	err := manager.Push(world, "broken")
	if err == nil {
		t.Fatal("expected error when push load hook fails")
	}
	if len(manager.Stack()) != 0 {
		t.Fatalf("expected stack to remain empty after failed push, got %d", len(manager.Stack()))
	}
}

func TestManager_Pop_EmptyStack(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	err := manager.Pop(world)
	if err == nil {
		t.Fatal("expected error when popping empty stack")
	}
}

func TestManager_Push_KeepsUnderlyingSceneLoaded(t *testing.T) {
	// Pushing an overlay must not unload the scene beneath it (pause-menu pattern).
	world := core.NewWorld()
	manager := NewManager()

	gameplay := NewScene("gameplay")
	pause := NewScene("pause")

	gameplayUnloaded := false
	gameplay.SetUnloadHook(func(w *core.World) error {
		gameplayUnloaded = true
		return nil
	})

	_ = manager.LoadScene("gameplay", gameplay)
	_ = manager.LoadScene("pause", pause)

	_ = manager.Push(world, "gameplay")
	_ = manager.Push(world, "pause")

	if gameplayUnloaded {
		t.Fatal("expected gameplay scene to remain loaded while pause is pushed on top")
	}
}

func TestManager_TransitionTo_ClearsWholeStack(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager()

	gameplay := NewScene("gameplay")
	pause := NewScene("pause")
	menu := NewScene("menu")

	_ = manager.LoadScene("gameplay", gameplay)
	_ = manager.LoadScene("pause", pause)
	_ = manager.LoadScene("menu", menu)

	_ = manager.Push(world, "gameplay")
	_ = manager.Push(world, "pause")

	if err := manager.TransitionTo(world, "menu"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manager.Stack()) != 1 {
		t.Fatalf("expected stack size 1 after transition, got %d", len(manager.Stack()))
	}
	if manager.Current() != menu {
		t.Fatal("expected menu to be the sole active scene after transition")
	}
}
