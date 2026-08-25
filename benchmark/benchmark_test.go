package benchmark

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
)

// Benchmark entity creation
func BenchmarkEntityCreation(b *testing.B) {
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Create("Generic")
	}

	b.SetBytes(int64(b.N))
}

func BenchmarkEntityCreationWithComponents(b *testing.B) {
	world := core.NewWorld()

	for i := 0; b.Loop(); i++ {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
		world.AddComponent(id, Health{Value: 100})
	}
}

func BenchmarkEntityCreationParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			world := core.NewWorld()
			for range 1000 {
				world.Create("Generic")
			}
		}
	})
}

// Benchmark component operations
func BenchmarkAddComponent(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		entities[i] = world.Create("Generic")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.AddComponent(entities[i], Position{X: float64(i), Y: float64(i)})
	}
}

func BenchmarkGetComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-create entities with components
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		entities[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.GetComponent(entities[i], posType)
	}
}

func BenchmarkHasComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-create entities with components
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		entities[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.HasComponent(entities[i], posType)
	}
}

func BenchmarkRemoveComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-create entities with components
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		entities[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.RemoveComponent(entities[i], posType)
	}
}

// Benchmark query operations
func BenchmarkQuerySingleComponent(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	world := core.NewWorld()

	// Pre-create entities with Position component
	for i := range 10000 {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
	}

	b.ResetTimer()
	for b.Loop() {
		world.Query(posType)
	}
}

func BenchmarkQueryMultipleComponents(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	healthType := reflect.TypeFor[Health]()
	world := core.NewWorld()

	// Pre-create entities with multiple components
	for i := range 10000 {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
		world.AddComponent(id, Health{Value: 100})
	}

	for b.Loop() {
		world.Query(posType, velType, healthType)
	}
}

func BenchmarkQueryByTag(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with tags
	for range 10000 {
		id := world.Create("Province")
		world.AddTag(id, "Province")
	}

	for b.Loop() {
		world.QueryByTag("Province")
	}
}

func BenchmarkQueryByTemplate(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with specific template
	for range 10000 {
		world.Create("Province")
	}

	for b.Loop() {
		world.QueryByTemplate("Province")
	}
}

// Benchmark hierarchy operations
func BenchmarkSetParent(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	parents := make([]ecs.EntityID, b.N)
	children := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		parents[i] = world.Create("Generic")
		children[i] = world.Create("Generic")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.SetParent(children[i], parents[i])
	}
}

func BenchmarkParentOf(b *testing.B) {
	world := core.NewWorld()

	// Pre-create hierarchy
	parents := make([]ecs.EntityID, b.N)
	children := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		parents[i] = world.Create("Generic")
		children[i] = world.Create("Generic")
		world.SetParent(children[i], parents[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.ParentOf(children[i])
	}
}

func BenchmarkChildrenOf(b *testing.B) {
	world := core.NewWorld()

	// Pre-create hierarchy - one parent with many children
	parent := world.Create("Generic")
	children := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		children[i] = world.Create("Generic")
		world.SetParent(children[i], parent)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.ChildrenOf(parent)
	}
}

// Benchmark entity destruction
func BenchmarkDestroyEntity(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		entities[i] = world.Create("Generic")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Destroy(entities[i], false)
	}

	// Cleanup after benchmark
	world.Cleanup()
}

func BenchmarkDestroyEntityWithCleanup(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		entities[i] = world.Create("Generic")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Destroy(entities[i], false)
		world.Cleanup()
	}
}

// Benchmark mixed operations (simulating game loop)
func BenchmarkGameLoopSimulation(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	world := core.NewWorld()

	// Pre-create some entities
	for i := range 1000 {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
	}

	b.ResetTimer()
	for b.Loop() {
		// Simulate a game frame
		entities := world.Query(posType, velType)
		for _, id := range entities {
			pos, _ := world.GetComponent(id, posType)
			vel, _ := world.GetComponent(id, velType)
			if pos != nil && vel != nil {
				p := pos.(Position)
				v := vel.(Velocity)
				p.X += v.X
				p.Y += v.Y
				world.AddComponent(id, p)
			}
		}
	}
}

// Benchmark tag operations
func BenchmarkAddTag(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		entities[i] = world.Create("Generic")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.AddTag(entities[i], "TestTag")
	}
}

func BenchmarkHasTag(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with tags
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		id := world.Create("Generic")
		world.AddTag(id, "TestTag")
		entities[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.HasTag(entities[i], "TestTag")
	}
}

func BenchmarkRemoveTag(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with tags
	entities := make([]ecs.EntityID, b.N)
	for i := 0; i < b.N; i++ {
		id := world.Create("Generic")
		world.AddTag(id, "TestTag")
		entities[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.RemoveTag(entities[i], "TestTag")
	}
}

// Benchmark world operations
func BenchmarkWorldCount(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	for range 10000 {
		world.Create("Generic")
	}

	for b.Loop() {
		world.Count()
	}
}

func BenchmarkWorldExists(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities
	entities := make([]ecs.EntityID, 10000)
	for i := range 10000 {
		entities[i] = world.Create("Generic")
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		world.Exists(entities[i%10000])
	}
}

// Benchmark component storage operations
func BenchmarkComponentsList(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with multiple components
	entities := make([]ecs.EntityID, 1000)
	for i := range 1000 {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
		world.AddComponent(id, Health{Value: 100})
		world.AddComponent(id, Sprite{TextureID: "test", Width: 32, Height: 32})
		entities[i] = id
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		world.Components(entities[i%1000])
	}
}

// Benchmark large scale scenarios
func BenchmarkLargeScaleEntityCreation(b *testing.B) {
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range 100 {
			id := world.Create("Generic")
			world.AddComponent(id, Position{X: float64(j), Y: float64(j)})
		}
	}
}

func BenchmarkLargeScaleQuery(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	world := core.NewWorld()

	// Pre-create 50,000 entities
	for i := range 50000 {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
		if i%2 == 0 {
			world.AddTag(id, "Even")
		} else {
			world.AddTag(id, "Odd")
		}
	}

	b.ResetTimer()
	for b.Loop() {
		world.Query(posType, velType)
	}
}

// Benchmark memory allocation patterns
func BenchmarkMemoryEfficientOperations(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	world := core.NewWorld()

	// Pre-create entities
	for i := range 1000 {
		id := world.Create("Generic")
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		entities := world.Query(posType)
		for _, id := range entities {
			world.GetComponent(id, posType)
		}
	}
}
