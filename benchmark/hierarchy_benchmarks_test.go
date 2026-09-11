package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// ============================================================================
// Hierarchy Operations
// ============================================================================

// BenchmarkSetParent measures the time to set a parent-child relationship.
func BenchmarkSetParent(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create entities
	parents := make([]*ecs.Entity, BenchmarkEntityPoolSize)
	children := make([]*ecs.Entity, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		parents[i] = world.Create("Generic")
		children[i] = world.Create("Generic")
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		world.SetParent(children[i%BenchmarkEntityPoolSize].ID, parents[i%BenchmarkEntityPoolSize].ID)
		i++
	}
}

// BenchmarkChildrenOf measures the time to query children of an entity.
func BenchmarkChildrenOf(b *testing.B) {
	world := ecs.NewWorld()

	// Pre-Create hierarchy - one parent with many children
	parent := world.Create("Generic")
	children := make([]*ecs.Entity, BenchmarkEntityPoolSize)
	for i := range BenchmarkEntityPoolSize {
		children[i] = world.Create("Generic")
		world.SetParent(children[i].ID, parent.ID)
	}

	b.ResetTimer()
	for b.Loop() {
		world.ChildrenOf(parent.ID)
	}
}
