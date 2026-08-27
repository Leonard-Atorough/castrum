package scene

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
)

// Integration tests using the real core.World implementation

func TestScene_IntegrationWithRealWorld(t *testing.T) {
	// Use the real core.World implementation
	world := core.NewWorld()
	scene := NewScene("integration-test")

	// Create entities
	entity1 := world.CreateEntity("player")
	entity2 := world.CreateEntity("enemy")

	// Add entities to scene
	_ = scene.AddToScene(entity1, world)
	_ = scene.AddToScene(entity2, world)

	// Verify entities are in scene
	entities := scene.Entities(world)
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities in scene, got %d", len(entities))
	}

	// Verify entities have the scene tag
	for _, id := range entities {
		hasTag, err := world.HasTag(id, "scene:integration-test")
		if err != nil {
			t.Fatalf("error checking tag for entity %d: %v", id, err)
		}
		if !hasTag {
			t.Fatalf("entity %d should have scene tag", id)
		}
	}

	// Remove from scene
	_ = scene.RemoveFromScene(entity1, world)
	entities = scene.Entities(world)
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity in scene after removal, got %d", len(entities))
	}
}

func TestManager_IntegrationWithRealWorld(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	// Create scenes with entities
	scene1 := NewScene("level-1")
	scene2 := NewScene("level-2")

	entity1 := world.CreateEntity("player")
	entity2 := world.CreateEntity("enemy")

	_ = scene1.AddToScene(entity1, world)
	_ = scene2.AddToScene(entity2, world)

	// Load scenes
	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	// Transition to level-1
	_ = manager.TransitionTo("level-1")

	if manager.CurrentScene() != scene1 {
		t.Fatal("expected current scene to be level-1")
	}

	// Transition to level-2
	_ = manager.TransitionTo("level-2")

	if manager.CurrentScene() != scene2 {
		t.Fatal("expected current scene to be level-2")
	}

	// Verify entity1 was removed from scene1 during transition
	entities := scene1.Entities(world)
	if len(entities) != 0 {
		t.Fatalf("expected scene1 to have 0 entities after transition, got %d", len(entities))
	}
}

func TestBuilder_IntegrationWithRealWorld(t *testing.T) {
	world := core.NewWorld()

	// Create entities
	entity1 := world.CreateEntity("player")
	entity2 := world.CreateEntity("enemy")

	// Build scene with entities
	builder := NewBuilder("builder-test")
	builder.WithEntity(entity1).WithEntity(entity2)

	scene, err := builder.Build(world)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify scene has entities
	entities := scene.Entities(world)
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities in scene, got %d", len(entities))
	}

	// Test with hooks
	loadCalled := false
	builder2 := NewBuilder("builder-test-2")
	builder2.WithEntity(entity1).WithLoadHook(func(w ecs.World) error {
		loadCalled = true
		return nil
	})

	scene2, _ := builder2.Build(world)
	_ = scene2.OnLoad(world)

	if !loadCalled {
		t.Fatal("expected load hook to be called")
	}
}

func TestSceneManager_IntegrationWithRealWorld(t *testing.T) {
	world := core.NewWorld()
	manager := NewManager(world)

	// Create entities
	player := world.CreateEntity("player")
	enemy := world.CreateEntity("enemy")
	boss := world.CreateEntity("boss")

	// Create scenes using builder
	builder1 := NewBuilder("level-1")
	builder1.WithEntity(player).WithEntity(enemy)

	builder2 := NewBuilder("level-2")
	builder2.WithEntity(boss)

	scene1, _ := builder1.Build(world)
	scene2, _ := builder2.Build(world)

	// Load scenes
	_ = manager.LoadScene("level-1", scene1)
	_ = manager.LoadScene("level-2", scene2)

	// Verify entities are in their respective scenes
	if len(scene1.Entities(world)) != 2 {
		t.Fatalf("expected 2 entities in level-1, got %d", len(scene1.Entities(world)))
	}
	if len(scene2.Entities(world)) != 1 {
		t.Fatalf("expected 1 entity in level-2, got %d", len(scene2.Entities(world)))
	}

	// Transition to level-1
	_ = manager.TransitionTo("level-1")
	if manager.CurrentScene() != scene1 {
		t.Fatal("expected current scene to be level-1")
	}

	// Transition to level-2
	_ = manager.TransitionTo("level-2")
	if manager.CurrentScene() != scene2 {
		t.Fatal("expected current scene to be level-2")
	}

	// Verify level-1 entities were cleaned up
	if len(scene1.Entities(world)) != 0 {
		t.Fatalf("expected level-1 to have 0 entities after transition, got %d", len(scene1.Entities(world)))
	}

	// Unload level-2
	_ = manager.UnloadScene("level-2")
	if manager.CurrentScene() != nil {
		t.Fatal("expected no current scene after unloading level-2")
	}
}

func TestScene_Lifecycle_IntegrationWithRealWorld(t *testing.T) {
	world := core.NewWorld()

	// Create a scene with load and unload hooks
	scene := NewScene("lifecycle-test")

	loadCalled := false
	unloadCalled := false

	scene.SetLoadHook(func(w ecs.World) error {
		loadCalled = true
		return nil
	})

	scene.SetUnloadHook(func(w ecs.World) error {
		unloadCalled = true
		return nil
	})

	// Create and add entities
	entity1 := world.CreateEntity("test-entity")
	_ = scene.AddToScene(entity1, world)

	// Test OnLoad
	err := scene.OnLoad(world)
	if err != nil {
		t.Fatalf("unexpected error during OnLoad: %v", err)
	}
	if !loadCalled {
		t.Fatal("expected load hook to be called")
	}

	// Verify entity is in scene
	if len(scene.Entities(world)) != 1 {
		t.Fatalf("expected 1 entity in scene after load, got %d", len(scene.Entities(world)))
	}

	// Test OnUnload
	err = scene.OnUnload(world)
	if err != nil {
		t.Fatalf("unexpected error during OnUnload: %v", err)
	}
	if !unloadCalled {
		t.Fatal("expected unload hook to be called")
	}

	// Verify entities are removed
	if len(scene.Entities(world)) != 0 {
		t.Fatalf("expected 0 entities in scene after unload, got %d", len(scene.Entities(world)))
	}
}

func TestScene_Data_IntegrationWithRealWorld(t *testing.T) {
	world := core.NewWorld()
	scene := NewScene("data-test")

	// Set various types of data
	scene.SetData("score", 1000)
	scene.SetData("name", "test-level")
	scene.SetData("completed", true)
	scene.SetData("settings", map[string]int{"difficulty": 5})

	// Verify data retrieval
	val, ok := scene.GetData("score")
	if !ok || val != 1000 {
		t.Fatal("expected score to be 1000")
	}

	val, ok = scene.GetData("name")
	if !ok || val != "test-level" {
		t.Fatal("expected name to be 'test-level'")
	}

	val, ok = scene.GetData("completed")
	if !ok || val != true {
		t.Fatal("expected completed to be true")
	}

	// Verify non-existent key
	_, ok = scene.GetData("nonexistent")
	if ok {
		t.Fatal("expected nonexistent key to not exist")
	}

	// Test with entities
	entity := world.CreateEntity("data-entity")
	_ = scene.AddToScene(entity, world)

	// Data should still be accessible
	val, ok = scene.GetData("score")
	if !ok || val != 1000 {
		t.Fatal("expected score to still be 1000 after adding entity")
	}
}
