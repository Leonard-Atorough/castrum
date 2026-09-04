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

	b.ResetTimer()
	for b.Loop() {
		core.QueryFor[Position](world)
	}
}

// BenchmarkQueryMultipleComponents measures query performance for multiple component types.
func BenchmarkQueryMultipleComponents(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with Position + Velocity
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 0.1, Y: 0.1})
	}

	b.ResetTimer()
	for b.Loop() {
		world.NewQuery().WithRequiredComponents(Position{}, Velocity{}).EntityIDs()
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

	b.ResetTimer()
	for b.Loop() {
		world.NewQuery().WithRequiredComponents(Position{}).EntityIDs()
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

	b.ResetTimer()
	for b.Loop() {
		world.NewQuery().WithRequiredComponents(Position{}).EntityIDs()
	}
}

// BenchmarkIterateQueryResults measures cost of iterating query results and accessing components.
func BenchmarkIterateQueryResults(b *testing.B) {
	world := core.NewWorld()
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}

	b.ResetTimer()
	for b.Loop() {

		for entry := range world.NewQuery().WithRequiredComponents(Position{}).Execute() {
			world.GetComponent(entry.EntityID, reflect.TypeFor[Position]())
		}
	}
}
