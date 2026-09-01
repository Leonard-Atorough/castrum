package benchmark

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// ============================================================================
// Hierarchy Operations
// ============================================================================

// BenchmarkSetParent measures the time to set a parent-child relationship.
func BenchmarkSetParent(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create entities
	parents := make([]*core.Entity, 10000)
	children := make([]*core.Entity, 10000)
	for i := range 10000 {
		parents[i] = world.Create("Generic")
		children[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		world.SetParent(children[i%10000].ID, parents[i%10000].ID)
	}
}

// BenchmarkChildrenOf measures the time to query children of an entity.
func BenchmarkChildrenOf(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create hierarchy - one parent with many children
	parent := world.Create("Generic")
	children := make([]*core.Entity, 10000)
	for i := range 10000 {
		children[i] = world.Create("Generic")
		world.SetParent(children[i].ID, parent.ID)
	}

	for i := 0; b.Loop(); i++ {
		world.ChildrenOf(parent.ID)
	}
}
