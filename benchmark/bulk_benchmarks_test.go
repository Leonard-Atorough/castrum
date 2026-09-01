package benchmark

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// ============================================================================
// Batch Operations
// ============================================================================

// BenchmarkCreateMany measures the performance of batch entity creation.
func BenchmarkCreateMany(b *testing.B) {
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		world.CreateMany("Generic", 100)
	}
}

// BenchmarkCreateManyVsIndividual compares batch creation vs individual entity creation.
func BenchmarkCreateManyVsIndividual(b *testing.B) {
	// Compare CreateMany(100) vs 100 individual Create() calls
	// to verify batch optimization works
	b.Run("CreateMany", func(b *testing.B) {
		world := core.NewWorld()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			world.CreateMany("Generic", 100)
		}
	})

	b.Run("IndividualCreate", func(b *testing.B) {
		world := core.NewWorld()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			for j := 0; j < 100; j++ {
				world.Create("Generic")
			}
		}
	})
}

// BenchmarkBulkAddComponents measures performance of adding multiple components rapidly.
func BenchmarkBulkAddComponents(b *testing.B) {
	// Add multiple components to same entity in rapid succession
	world := core.NewWorld()
	entity := world.Create("Generic")

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		// Remove existing components
		world.RemoveComponent(entity.ID, reflect.TypeFor[Position]())
		world.RemoveComponent(entity.ID, reflect.TypeFor[Velocity]())
		world.RemoveComponent(entity.ID, reflect.TypeFor[Health]())

		// Add all components
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
		world.AddComponent(entity.ID, Health{Value: 100})
	}
}

// BenchmarkBulkRemoveComponents measures performance of removing multiple components rapidly.
func BenchmarkBulkRemoveComponents(b *testing.B) {
	// Remove multiple components in rapid succession
	world := core.NewWorld()
	entity := world.Create("Generic")
	world.AddComponent(entity.ID, Position{X: 1, Y: 1})
	world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
	world.AddComponent(entity.ID, Health{Value: 100})

	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	healthType := reflect.TypeFor[Health]()

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		world.RemoveComponent(entity.ID, posType)
		world.RemoveComponent(entity.ID, velType)
		world.RemoveComponent(entity.ID, healthType)

		// Re-add for next iteration
		world.AddComponent(entity.ID, Position{X: 1, Y: 1})
		world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
		world.AddComponent(entity.ID, Health{Value: 100})
	}
}
