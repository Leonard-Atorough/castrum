package benchmark

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// ============================================================================
// Query Operations
// ============================================================================

// BenchmarkQuerySingleComponent measures query performance for single component type.
func BenchmarkQuerySingleComponent(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with Position
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}

	posType := reflect.TypeFor[Position]()

	b.ResetTimer()
	for b.Loop() {
		world.Query(posType)
	}
}

// BenchmarkQueryMultipleComponents measures query performance for multiple component types.
func BenchmarkQueryMultipleComponents(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with Position + Velocity
	for i := 0; i < 10000; i++ {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 0.1, Y: 0.1})
	}

	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()

	b.ResetTimer()
	for b.Loop() {
		world.Query(posType, velType)
	}
}

// ============================================================================
// Query Selectivity (Phase 2)
// ============================================================================

// BenchmarkQuerySparsely measures query performance with sparse results (1% match rate).
func BenchmarkQuerySparsely(b *testing.B) {
	// Create 10,000 entities, only 1% have Position (100 entities match)
	world := core.NewWorld()
	for i := range 10000 {
		entity := world.Create("Generic")
		if i%100 == 0 {
			world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		}
	}

	posType := reflect.TypeFor[Position]()

	b.ResetTimer()
	for b.Loop() {
		world.Query(posType)
	}
}

// BenchmarkQueryDensely measures query performance with dense results (100% match rate).
func BenchmarkQueryDensely(b *testing.B) {
	// Create 10,000 entities, all have Position (100% match)
	world := core.NewWorld()
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}

	posType := reflect.TypeFor[Position]()

	b.ResetTimer()
	for b.Loop() {
		world.Query(posType)
	}
}

// BenchmarkIterateQueryResults measures cost of iterating query results and accessing components.
func BenchmarkIterateQueryResults(b *testing.B) {
	// Measure cost of iterating query results and accessing components
	world := core.NewWorld()
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}

	posType := reflect.TypeFor[Position]()

	b.ResetTimer()
	for b.Loop() {
		entities := world.Query(posType)
		for _, id := range entities {
			world.GetComponent(id, posType)
		}
	}
}
