package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// ============================================================================
// Memory Efficiency
// ============================================================================

// BenchmarkMemoryPerEntity measures memory footprint per entity (baseline, no components).
func BenchmarkMemoryPerEntity(b *testing.B) {
	// Measure memory footprint per entity (baseline, no components)
	b.ReportAllocs()
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; i < 10000; i++ {
		world.Create("Generic")
	}

	// Memory measured via benchmem flag
}

// BenchmarkMemoryPerEntityWithComponents measures memory footprint per entity with components.
func BenchmarkMemoryPerEntityWithComponents(b *testing.B) {
	// Measure memory footprint per entity with components
	b.ReportAllocs()
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; i < 1000; i++ {
		e := world.Create("Generic")
		world.AddComponent(e.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(e.ID, Velocity{X: 1.0, Y: 1.0})
	}
}
