package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// ============================================================================
// Memory Efficiency
// ============================================================================

// BenchmarkMemoryPerEntity measures memory footprint per entity (baseline, no components).
func BenchmarkMemoryPerEntity(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		world := ecs.NewWorld()
		for range DefaultEntityCount {
			world.Create("Generic")
		}
	}
}

// BenchmarkMemoryPerEntityWithComponents measures the allocation cost of
// constructing a world whose entities start in their final component archetype.
func BenchmarkMemoryPerEntityWithComponents(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		world := ecs.NewWorld()
		for i := range DefaultEntityCount {
			_, err := world.CreateWithComponents(
				"Generic",
				Position{X: float64(i), Y: float64(i)},
				Velocity{X: 1.0, Y: 1.0},
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
