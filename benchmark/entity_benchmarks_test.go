package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// ============================================================================
// Entity Operations
// ============================================================================

// BenchmarkEntityCreation measures the time to create an empty entity.
func BenchmarkEntityCreation(b *testing.B) {
	world := ecs.NewWorld()

	b.ResetTimer()
	for b.Loop() {
		world.Create("Generic")
	}

	b.SetBytes(int64(b.N))
}

// BenchmarkEntityCreationWithComponents measures the time to create an entity with multiple components.
func BenchmarkEntityCreationWithComponents(b *testing.B) {
	world := ecs.NewWorld()

	components := []ecs.Component{
		Position{X: 0, Y: 0},
		Velocity{X: 1, Y: 1},
		Health{Value: 100},
		Sprite{TextureID: "test", Width: 32, Height: 32},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := world.CreateWithComponents("Generic", components...)
		if err != nil {
			panic(err)
		}
	}
}

// BenchmarkDestroyAndRecreateEntity measures the time to destroy and recreate entities.
func BenchmarkDestroyAndRecreateEntity(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*ecs.Entity, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		entityPool[i] = world.Create("Generic")
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		entity := entityPool[i%BenchmarkEntityPoolSize]
		world.DestroyEntity(entity.ID, false)
		// Recreate entity for next iteration
		entityPool[i%BenchmarkEntityPoolSize] = world.Create("Generic")
		i++
	}

	// Cleanup after benchmark
	world.Cleanup()
}
