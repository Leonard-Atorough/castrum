package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// ============================================================================
// Component Operations
// ============================================================================

// BenchmarkAddComponent measures the time to add a component to an entity.
func BenchmarkAddComponent(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*ecs.Entity, BenchmarkEntityPoolSize)
	for i := range entityPool {
		entityPool[i] = world.Create("Generic")
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		// Reuse entities from pool, remove old component if exists
		entity := entityPool[i%BenchmarkEntityPoolSize]
		world.RemoveComponent[Position](entity.ID)
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		i++
	}
}

// BenchmarkGetComponent measures the time to retrieve a component from an entity.
func BenchmarkGetComponent(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]ecs.EntityID, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		world.GetComponent[Position](entityPool[i%BenchmarkEntityPoolSize])
		i++
	}
}

// BenchmarkHasComponent measures the time to check for component existence.
func BenchmarkHasComponent(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]ecs.EntityID, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		world.HasComponent[Position](entityPool[i%BenchmarkEntityPoolSize])
		i++
	}
}

// BenchmarkRemoveComponent measures the time to remove a component from an entity.
func BenchmarkRemoveComponent(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create a fixed pool of entities with components
	entityPool := make([]ecs.EntityID, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		entityID := entityPool[i%BenchmarkEntityPoolSize]
		world.RemoveComponent[Position](entityID)
		// Re-add component for next iteration
		world.AddComponent(entityID, Position{X: float64(i), Y: float64(i)})
		i++
	}
}

// BenchmarkDestroyEntityWithCleanup measures entity destruction with batched cleanup.
func BenchmarkDestroyEntityWithCleanup(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*ecs.Entity, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		entityPool[i] = world.Create("Generic")
	}

	b.ResetTimer()
	destroyCount := 0
	i := 0
	for b.Loop() {
		entity := entityPool[i%BenchmarkEntityPoolSize]
		world.DestroyEntity(entity.ID, false)
		destroyCount++

		// Batch cleanup calls (every 100 operations)
		if destroyCount%100 == 0 {
			world.Cleanup()
		}

		// Recreate entity for next iteration
		entityPool[i%BenchmarkEntityPoolSize] = world.Create("Generic")
		i++
	}

	// Final cleanup
	world.Cleanup()
}

// ============================================================================
// Component Migrations (Archetype Movement)
// ============================================================================

// BenchmarkComponentAddMigration measures the cost of adding a component to an entity (archetype migration).
func BenchmarkComponentAddMigration(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-create entity with no components (empty archetype)
	entity := world.Create("Generic")

	b.ResetTimer()
	i := 0
	for b.Loop() {
		// Remove and re-add to test archetype migration
		world.RemoveComponent[Position](entity.ID)
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		i++
	}
}

// BenchmarkComponentRemoveMigration measures the cost of removing a component from an entity (archetype migration).
func BenchmarkComponentRemoveMigration(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-create entity with multiple components
	entity := world.Create("Generic")
	world.AddComponent(entity.ID, Position{X: 1, Y: 1})
	world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
	world.AddComponent(entity.ID, Health{Value: 100})

	b.ResetTimer()
	i := 0
	for b.Loop() {
		// Cycle through removing each component type
		switch i % 3 {
		case 0:
			world.RemoveComponent[Position](entity.ID)
		case 1:
			world.RemoveComponent[Velocity](entity.ID)
		case 2:
			world.RemoveComponent[Health](entity.ID)
		}
		// Re-add for next iteration
		switch i % 3 {
		case 0:
			world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		case 1:
			world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
		case 2:
			world.AddComponent(entity.ID, Health{Value: 100})
		}
		i++
	}
}
