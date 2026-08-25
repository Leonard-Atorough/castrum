package scene

import (
	"errors"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

func TestNewManager(t *testing.T) {
	world := &mockWorld{}
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

func TestManager_World(t *testing.T) {
	world := &mockWorld{}
	manager := NewManager(world)

	if manager.World() != world {
		t.Fatal("expected World() to return the manager's world")
	}
}

func TestManager_LoadScene(t *testing.T) {
	world := &mockWorld{}
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
	world := &mockWorld{}
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
	world := &mockWorld{}
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
	world := &mockWorld{}
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
	world := &mockWorld{}
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
	world := &mockWorld{}
	manager := NewManager(world)

	scene := NewScene("level-1")
	expectedErr := errors.New("unload error")
	scene.SetUnloadHook(func(w ecs.World) error {
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
	world := &mockWorld{}
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
	world := &mockWorld{}
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
	world := &mockWorld{}
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
	world := &mockWorld{}
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	expectedErr := errors.New("unload error")
	scene1.SetUnloadHook(func(w ecs.World) error {
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
	world := &mockWorld{}
	manager := NewManager(world)

	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	expectedErr := errors.New("load error")
	scene2.SetLoadHook(func(w ecs.World) error {
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
	world := &mockWorld{}
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
