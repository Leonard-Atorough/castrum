package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// ============================================================================
// Batch Operations
// ============================================================================

// BenchmarkCreateMany measures the performance of batch entity creation.
func BenchmarkCreateMany(b *testing.B) {
	world := ecs.NewWorld()

	b.ResetTimer()
	for b.Loop() {
		world.CreateMany("Generic", 100)
	}
}

// BenchmarkCreateManyVsIndividual compares batch creation vs individual entity creation.
func BenchmarkCreateManyVsIndividual(b *testing.B) {
	// Compare CreateMany(100) vs 100 individual Create() calls
	// to verify batch optimization works
	b.Run("CreateMany", func(b *testing.B) {
		world := ecs.NewWorld()
		b.ResetTimer()
		for b.Loop() {
			world.CreateMany("Generic", 100)
		}
	})

	b.Run("IndividualCreate", func(b *testing.B) {
		world := ecs.NewWorld()
		b.ResetTimer()
		for b.Loop() {
			for j := 0; j < 100; j++ {
				world.Create("Generic")
			}
		}
	})
}

// BenchmarkBulkAddAndRemoveComponents measures performance of adding and removing multiple components rapidly.
func BenchmarkBulkAddAndRemoveComponents(b *testing.B) {
	// Add and remove multiple components to same entity in rapid succession
	world := ecs.NewWorld()
	entity := world.Create("Generic")

	b.ResetTimer()
	i := 0
	for b.Loop() {
		// Remove existing components
		world.RemoveComponent[Position](entity.ID)
		world.RemoveComponent[Velocity](entity.ID)
		world.RemoveComponent[Health](entity.ID)

		// Add all components
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
		world.AddComponent(entity.ID, Health{Value: 100})
		i++
	}
}

// BenchmarkBulkRemoveAndAddComponents measures performance of removing and adding multiple components rapidly.
func BenchmarkBulkRemoveAndAddComponents(b *testing.B) {
	// Remove and add multiple components in rapid succession
	world := ecs.NewWorld()
	entity := world.Create("Generic")
	world.AddComponent(entity.ID, Position{X: 1, Y: 1})
	world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
	world.AddComponent(entity.ID, Health{Value: 100})

	b.ResetTimer()
	for b.Loop() {
		world.RemoveComponent[Position](entity.ID)
		world.RemoveComponent[Velocity](entity.ID)
		world.RemoveComponent[Health](entity.ID)

		// Re-add for next iteration
		world.AddComponent(entity.ID, Position{X: 1, Y: 1})
		world.AddComponent(entity.ID, Velocity{X: 1, Y: 1})
		world.AddComponent(entity.ID, Health{Value: 100})
	}
}
