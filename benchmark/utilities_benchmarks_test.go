package benchmark

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// BenchmarkComponentTypeRegistry benchmarks component type lookups.
func BenchmarkComponentTypeRegistry(b *testing.B) {
	for b.Loop() {
		_ = reflect.TypeFor[Position]()
		_ = reflect.TypeFor[Velocity]()
		_ = reflect.TypeFor[Health]()
		_ = reflect.TypeFor[Sprite]()
	}
}

// BenchmarkEntityReuse benchmarks adding and removing components on a fixed entity pool.
func BenchmarkEntityReuse(b *testing.B) {
	world := ecs.NewWorld()

	entityPool := make([]ecs.EntityID, 1000)
	for i := range entityPool {
		entityPool[i] = world.Create("PooledEntity").ID
	}

	for i := 0; b.Loop(); i++ {
		entityID := entityPool[i%len(entityPool)]
		world.AddComponent(entityID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entityID, Velocity{X: 1.0, Y: 1.0})
		world.RemoveComponent[Position](entityID)
		world.RemoveComponent[Velocity](entityID)
	}
}
