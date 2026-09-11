package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// ============================================================================
// Game Loop Scenarios
// ============================================================================

// BenchmarkGameLoopSimple measures a simple game loop: query and read only.
func BenchmarkGameLoopSimple(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create entities
	for i := range DefaultEntityCount {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}
	query := world.NewQuery().WithRequiredComponents(Position{}, Velocity{})

	b.ResetTimer()
	for b.Loop() {
		// Simulate a simple game frame: query and read
		for entry := range query.Execute() {
			pos, _ := entry.Get[Position]()
			vel, _ := entry.Get[Velocity]()
			_ = pos // Use to prevent optimization
			_ = vel
		}
	}
}

// BenchmarkGameLoopWithUpdates measures game loop with query, read, and update operations.
func BenchmarkGameLoopWithUpdates(b *testing.B) {
	// Query, read, and update components (modify and store)
	world := ecs.NewWorld()

	for i := range DefaultEntityCount {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}
	query := world.NewQuery().WithRequiredComponents(Position{}, Velocity{})

	b.ResetTimer()
	for b.Loop() {
		for entry := range query.Execute() {
			id := entry.EntityID
			posComp, _ := entry.Get[Position]()
			velComp, _ := entry.Get[Velocity]()
			p := posComp
			v := velComp
			p.X += v.X
			p.Y += v.Y
			world.SetComponent(id, p)
		}
	}
}

// BenchmarkGameLoopWithSpawning measures fixed-population spawn/despawn churn
// while updating the existing entities.
func BenchmarkGameLoopWithSpawning(b *testing.B) {
	// Query, update, and replace 1% of the population with new entities.
	world := ecs.NewWorld()

	for i := range DefaultEntityCount {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}
	query := world.NewQuery().WithRequiredComponents(Position{}, Velocity{})
	entities := make([]ecs.EntityID, 0, DefaultEntityCount)

	b.ResetTimer()
	for b.Loop() {
		// Query and update
		entities = entities[:0]
		for entry := range query.Execute() {
			entities = append(entities, entry.EntityID)
			pos, _ := entry.Get[Position]()
			vel, _ := entry.Get[Velocity]()
			p := pos
			v := vel
			p.X += v.X
			p.Y += v.Y
			world.SetComponent(entry.EntityID, p)
		}

		// Retire and replace 1% of the population to keep the workload bounded.
		spawnCount := len(entities) / 100
		for i := 0; i < spawnCount; i++ {
			world.DestroyEntity(entities[i], false)
			e := world.Create("Spawned")
			world.AddComponent(e.ID, Position{X: 0, Y: 0})
			world.AddComponent(e.ID, Velocity{X: 1.0, Y: 1.0})
		}
	}
}

// BenchmarkGameLoopWithDestruction measures fixed-population destruction and
// replacement while updating the existing entities.
func BenchmarkGameLoopWithDestruction(b *testing.B) {
	// Query, update, and replace 0.5% of the population per frame.
	world := ecs.NewWorld()

	for i := range DefaultEntityCount {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}
	query := world.NewQuery().WithRequiredComponents(Position{}, Velocity{})
	entities := make([]ecs.EntityID, 0, DefaultEntityCount)

	b.ResetTimer()
	cleanupCount := 0
	for b.Loop() {
		// Query and update
		entities = entities[:0]
		for entry := range query.Execute() {
			entities = append(entities, entry.EntityID)
			pos, _ := entry.Get[Position]()
			vel, _ := entry.Get[Velocity]()
			p := pos
			v := vel
			p.X += v.X
			p.Y += v.Y
			world.SetComponent(entry.EntityID, p)
		}

		// Destroy 0.5% of entities per frame
		destroyCount := len(entities) / 200
		for i := 0; i < destroyCount; i++ {
			world.DestroyEntity(entities[i], false)
		}
		for i := 0; i < destroyCount; i++ {
			e := world.Create("Replacement")
			world.AddComponent(e.ID, Position{X: 0, Y: 0})
			world.AddComponent(e.ID, Velocity{X: 1.0, Y: 1.0})
		}

		// Batch cleanup (every 50 frames)
		cleanupCount++
		if cleanupCount%50 == 0 {
			world.Cleanup()
		}
	}
}

// BenchmarkGameLoopMixed measures a complete fixed-population game loop with
// updates and balanced spawning/destruction.
func BenchmarkGameLoopMixed(b *testing.B) {
	// Complete game loop: query, update, and replace 0.5% of the population.
	world := ecs.NewWorld()

	for i := range DefaultEntityCount {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}
	query := world.NewQuery().WithRequiredComponents(Position{}, Velocity{})
	entities := make([]ecs.EntityID, 0, DefaultEntityCount)

	b.ResetTimer()
	cleanupCount := 0
	for b.Loop() {
		// Query and update
		entities = entities[:0]
		for entry := range query.Execute() {
			entities = append(entities, entry.EntityID)
			pos, _ := entry.Get[Position]()
			vel, _ := entry.Get[Velocity]()
			p := pos
			v := vel
			p.X += v.X
			p.Y += v.Y
			world.SetComponent(entry.EntityID, p)
		}

		// Destroy and replace 0.5% per frame to keep population stable.
		replaceCount := len(entities) / 200
		for i := 0; i < replaceCount; i++ {
			world.DestroyEntity(entities[i], false)
		}
		for i := 0; i < replaceCount; i++ {
			e := world.Create("Replacement")
			world.AddComponent(e.ID, Position{X: 0, Y: 0})
			world.AddComponent(e.ID, Velocity{X: 1.0, Y: 1.0})
		}

		// Batch cleanup every 50 frames
		cleanupCount++
		if cleanupCount%50 == 0 {
			world.Cleanup()
		}
	}
}
