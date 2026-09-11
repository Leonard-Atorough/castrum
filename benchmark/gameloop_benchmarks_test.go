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

	// Pre-Create some entities
	for i := range 1000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}

	b.ResetTimer()
	for b.Loop() {
		// Simulate a simple game frame: query and read
		for entry := range world.NewQuery().WithRequiredComponents(Position{}, Velocity{}).Execute() {
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

	for i := range 5000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}

	b.ResetTimer()
	for b.Loop() {
		for entry := range world.NewQuery().WithRequiredComponents(Position{}, Velocity{}).Execute() {
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

// BenchmarkGameLoopWithSpawning measures game loop with spawning (1% per frame).
func BenchmarkGameLoopWithSpawning(b *testing.B) {
	// Query, update, and spawn new entities (1% spawn rate per frame)
	world := ecs.NewWorld()

	for i := range 5000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}

	b.ResetTimer()
	for b.Loop() {
		// Query and update
		var entities []ecs.EntityID
		for entry := range world.NewQuery().WithRequiredComponents(Position{}, Velocity{}).Execute() {
			entities = append(entities, entry.EntityID)
			pos, _ := entry.Get[Position]()
			vel, _ := entry.Get[Velocity]()
			p := pos
			v := vel
			p.X += v.X
			p.Y += v.Y
			world.SetComponent(entry.EntityID, p)
		}

		// Spawn new entities (1% of current count per frame)
		spawnCount := len(entities) / 100
		for i := 0; i < spawnCount; i++ {
			e := world.Create("Spawned")
			world.AddComponent(e.ID, Position{X: 0, Y: 0})
			world.AddComponent(e.ID, Velocity{X: 1.0, Y: 1.0})
		}
	}
}

// BenchmarkGameLoopWithDestruction measures game loop with destruction (0.5% per frame).
func BenchmarkGameLoopWithDestruction(b *testing.B) {
	// Query, update, and destroy entities (0.5% destruction rate per frame)
	world := ecs.NewWorld()

	for i := range 5000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}

	b.ResetTimer()
	cleanupCount := 0
	for b.Loop() {
		// Query and update
		var entities []ecs.EntityID
		for entry := range world.NewQuery().WithRequiredComponents(Position{}, Velocity{}).Execute() {
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
			if i < len(entities) {
				world.DestroyEntity(entities[i], false)
			}
		}

		// Batch cleanup (every 50 frames)
		cleanupCount++
		if cleanupCount%50 == 0 {
			world.Cleanup()
		}
	}
}

// BenchmarkGameLoopMixed measures complete game loop with updates, spawning, and destruction.
func BenchmarkGameLoopMixed(b *testing.B) {
	// Complete realistic game loop: query, update, spawn, destroy
	world := ecs.NewWorld()

	for i := range 5000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}

	b.ResetTimer()
	cleanupCount := 0
	for b.Loop() {
		// Query and update
		var entities []ecs.EntityID
		for entry := range world.NewQuery().WithRequiredComponents(Position{}, Velocity{}).Execute() {
			entities = append(entities, entry.EntityID)
			pos, _ := entry.Get[Position]()
			vel, _ := entry.Get[Velocity]()
			p := pos
			v := vel
			p.X += v.X
			p.Y += v.Y
			world.SetComponent(entry.EntityID, p)
		}

		// Spawn 1% per frame
		spawnCount := len(entities) / 100
		for i := 0; i < spawnCount; i++ {
			e := world.Create("Spawned")
			world.AddComponent(e.ID, Position{X: 0, Y: 0})
			world.AddComponent(e.ID, Velocity{X: 1.0, Y: 1.0})
		}

		// Destroy 0.5% per frame
		destroyCount := len(entities) / 200
		for i := 0; i < destroyCount; i++ {
			if i < len(entities) {
				world.DestroyEntity(entities[i], false)
			}
		}

		// Batch cleanup every 50 frames
		cleanupCount++
		if cleanupCount%50 == 0 {
			world.Cleanup()
		}
	}
}
