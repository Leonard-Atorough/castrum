package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// ============================================================================
// Entity Operations
// ============================================================================

// BenchmarkEntityCreation measures the time to create an empty entity.
func BenchmarkEntityCreation(b *testing.B) {
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Create("Generic")
	}

	b.SetBytes(int64(b.N))
}

// BenchmarkEntityCreationWithComponents measures the time to create an entity with multiple components.
func BenchmarkEntityCreationWithComponents(b *testing.B) {
	world := core.NewWorld()

	components := []core.Component{
		Position{X: 0, Y: 0},
		Velocity{X: 1, Y: 1},
		Health{Value: 100},
		Sprite{TextureID: "test", Width: 32, Height: 32},
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		_, err := world.CreateWithComponents("Generic", components...)
		if err != nil {
			panic(err)
		}
	}
}

// BenchmarkDestroyEntity measures the time to destroy entities.
func BenchmarkDestroyEntity(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range 10000 {
		entityPool[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		entity := entityPool[i%10000]
		world.DestroyEntity(entity.ID, false)
		// Recreate entity for next iteration
		entityPool[i%10000] = world.Create("Generic")
	}

	// Cleanup after benchmark
	world.Cleanup()
}
