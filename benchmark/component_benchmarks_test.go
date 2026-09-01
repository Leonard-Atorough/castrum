package benchmark

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// ============================================================================
// Component Operations
// ============================================================================

// BenchmarkAddComponent measures the time to add a component to an entity.
func BenchmarkAddComponent(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range entityPool {
		entityPool[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		// Reuse entities from pool, remove old component if exists
		entity := entityPool[i%10000]
		world.RemoveComponent(entity.ID, reflect.TypeFor[Position]())
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}
}

// BenchmarkGetComponent measures the time to retrieve a component from an entity.
func BenchmarkGetComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		world.GetComponent(entityPool[i%10000], posType)
	}
}

// BenchmarkHasComponent measures the time to check for component existence.
func BenchmarkHasComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		world.HasComponent(entityPool[i%10000], posType)
	}
}

// BenchmarkRemoveComponent measures the time to remove a component from an entity.
func BenchmarkRemoveComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-Create a fixed pool of entities with components
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		entityID := entityPool[i%10000]
		world.RemoveComponent(entityID, posType)
		// Re-add component for next iteration
		world.AddComponent(entityID, Position{X: float64(i), Y: float64(i)})
	}
}

// BenchmarkDestroyEntityWithCleanup measures entity destruction with batched cleanup.
func BenchmarkDestroyEntityWithCleanup(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range 10000 {
		entityPool[i] = world.Create("Generic")
	}

	b.ResetTimer()
	destroyCount := 0
	for i := 0; b.Loop(); i++ {
		entity := entityPool[i%10000]
		world.DestroyEntity(entity.ID, false)
		destroyCount++

		// Batch cleanup calls (every 100 operations)
		if destroyCount%100 == 0 {
			world.Cleanup()
		}

		// Recreate entity for next iteration
		entityPool[i%10000] = world.Create("Generic")
	}

	// Final cleanup
	world.Cleanup()
}

// ============================================================================
// Component Migrations (Archetype Movement)
// ============================================================================

// BenchmarkComponentAddMigration measures the cost of adding a component to an entity (archetype migration).
func BenchmarkComponentAddMigration(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entity with no components (empty archetype)
	entity := world.Create("Generic")

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		// Remove and re-add to test archetype migration
		world.RemoveComponent(entity.ID, reflect.TypeFor[Position]())
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}
}

// BenchmarkComponentRemoveMigration measures the cost of removing a component from an entity (archetype migration).
func BenchmarkComponentRemoveMigration(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entity with multiple components
	entity := world.Create("Generic")
	world.AddComponent(entity.ID, Position{X: 1, Y: 1})
	world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
	world.AddComponent(entity.ID, Health{Value: 100})

	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	healthType := reflect.TypeFor[Health]()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		// Cycle through removing each component type
		compType := []reflect.Type{posType, velType, healthType}[i%3]
		world.RemoveComponent(entity.ID, compType)
		// Re-add for next iteration
		switch i % 3 {
		case 0:
			world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		case 1:
			world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
		case 2:
			world.AddComponent(entity.ID, Health{Value: 100})
		}
	}
}
