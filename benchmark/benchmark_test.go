package benchmark

import (
	"reflect"
	"testing"

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

	components := []core.Component{
		Position{X: 0, Y: 0},
		Velocity{X: 1, Y: 1},
		Health{Value: 100},
		Sprite{TextureID: "test", Width: 32, Height: 32},
	}

	for i := 0; b.Loop(); i++ {
		_, err := world.CreateWithComponents("Generic", components...)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkEntityCreationParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			world := core.NewWorld()
			world.CreateMany("Generic", 1000)
		}
	})
}

// Benchmark component operations
func BenchmarkAddComponent(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range entityPool {
		entityPool[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		// Reuse entities from pool, remove old component if exists
		entity := entityPool[i%10000]
		world.RemoveComponent(entity.ID, reflect.TypeFor[Position]())
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}
}

func BenchmarkGetComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		world.GetComponent(entityPool[i%10000], posType)
	}
}

func BenchmarkHasComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		world.HasComponent(entityPool[i%10000], posType)
	}
}

func BenchmarkRemoveComponent(b *testing.B) {
	world := core.NewWorld()
	posType := reflect.TypeFor[Position]()

	// Pre-Create a fixed pool of entities with components
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		entityID := entityPool[i%10000]
		world.RemoveComponent(entityID, posType)
		// Re-add component for next iteration
		world.AddComponent(entityID, Position{X: float64(i), Y: float64(i)})
	}
}

// Benchmark query operations
func BenchmarkArchetypeQuery(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with Position
	for i := 0; i < 10000; i++ {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}

	posType := reflect.TypeFor[Position]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Query(posType)
	}
}

func BenchmarkArchetypeQueryMultiple(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entities with Position + Velocity
	for i := 0; i < 10000; i++ {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 0.1, Y: 0.1})
	}

	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()

	
	for b.Loop() {
		world.Query(posType, velType)
	}
}

func BenchmarkArchetypeGetComponent(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entity with Position
	entity := world.Create("Generic")
	world.AddComponent(entity.ID, Position{X: 1, Y: 2})

	posType := reflect.TypeFor[Position]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.GetComponent(entity.ID, posType)
	}
}

func BenchmarkArchetypeAddComponent(b *testing.B) {
	world := core.NewWorld()

	// Pre-create entity
	entity := world.Create("Generic")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Remove and re-add to test migration
		world.RemoveComponent(entity.ID, reflect.TypeFor[Position]())
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
	}
}

func BenchmarkArchetypeEntityCreation(b *testing.B) {
	world := core.NewWorld()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 0.1, Y: 0.1})
	}
}

func BenchmarkQueryByTag(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create entities with tags
	for range 10000 {
		entity := world.Create("Province")
		world.AddTag(entity.ID, "Province")
	}

	for b.Loop() {
		world.QueryByTag("Province")
	}
}

func BenchmarkQueryByTemplate(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create entities with specific template
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

func BenchmarkParentOf(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create hierarchy
	parents := make([]*core.Entity, 10000)
	children := make([]*core.Entity, 10000)
	for i := range 10000 {
		parents[i] = world.Create("Generic")
		children[i] = world.Create("Generic")
		world.SetParent(children[i].ID, parents[i].ID)
	}

	for i := 0; b.Loop(); i++ {
		world.ParentOf(children[i%10000].ID)
	}
}

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

// Benchmark entity destruction
func BenchmarkDestroyEntity(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range 10000 {
		entityPool[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		entity := entityPool[i%10000]
		world.DestroyEntity(entity.ID, false)
		// Recreate entity for next iteration
		entityPool[i%10000] = world.Create("Generic")
		entityPool[i%10000] = entityPool[i%10000]
	}

	// Cleanup after benchmark
	world.Cleanup()
}

func BenchmarkDestroyEntityWithCleanup(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range 10000 {
		entityPool[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		entity := entityPool[i%10000]
		world.DestroyEntity(entity.ID, false)
		world.Cleanup()
		// Recreate entity for next iteration
		entityPool[i%10000] = world.Create("Generic")
	}
}

// Benchmark mixed operations (simulating game loop)
func BenchmarkGameLoopSimulation(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	world := core.NewWorld()

	// Pre-Create some entities
	for i := range 1000 {
		entity := world.Create("Generic")
		world.AddComponent(entity.ID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entity.ID, Velocity{X: 1.0, Y: 1.0})
	}

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

	// Pre-Create a fixed pool of entities to avoid setup time dominating
	entityPool := make([]*core.Entity, 10000)
	for i := range 10000 {
		entityPool[i] = world.Create("Generic")
	}

	for i := 0; b.Loop(); i++ {
		entityID := entityPool[i%10000].ID
		world.AddTag(entityID, "TestTag")
		// Remove tag for next iteration
		if i > 0 && i%10000 == 0 {
			world.RemoveTag(entityID, "TestTag")
		}
	}
}

func BenchmarkRemoveTag(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create a fixed pool of entities with tags
	entityPool := make([]core.EntityID, 10000)
	for i := range 10000 {
		entity := world.Create("Generic")
		world.AddTag(entity.ID, "TestTag")
		entityPool[i] = entity.ID
	}

	for i := 0; b.Loop(); i++ {
		entityID := entityPool[i%10000]
		world.RemoveTag(entityID, "TestTag")
		// Re-add tag for next iteration
		world.AddTag(entityID, "TestTag")
	}
}

// Benchmark world operations
func BenchmarkWorldCount(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create entities
	for range 10000 {
		world.Create("Generic")
	}

	for b.Loop() {
		world.Count()
	}
}

func BenchmarkWorldExists(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create entities
	entities := make([]core.EntityID, 10000)
	for i := range 10000 {
		entities[i] = world.Create("Generic").ID
	}

	for i := 0; b.Loop(); i++ {
		world.HasEntity(entities[i%10000])
	}
}

// Benchmark component storage operations
func BenchmarkComponentsList(b *testing.B) {
	world := core.NewWorld()

	// Pre-Create entities with multiple components
	entities := make([]core.EntityID, 1000)
	for i := range 1000 {
		id := world.Create("Generic").ID
		world.AddComponent(id, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
		world.AddComponent(id, Health{Value: 100})
		world.AddComponent(id, Sprite{TextureID: "test", Width: 32, Height: 32})
		entities[i] = id
	}

	for i := 0; b.Loop(); i++ {
		world.Components(entities[i%1000])
	}
}

// Benchmark large scale scenarios
func BenchmarkLargeScaleEntityCreation(b *testing.B) {
	world := core.NewWorld()

	b.ResetTimer()
	for b.Loop() {
		for j := range 100 {
			id := world.Create("Generic").ID
			world.AddComponent(id, Position{X: float64(j), Y: float64(j)})
		}
	}
}

func BenchmarkLargeScaleQuery(b *testing.B) {
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	world := core.NewWorld()

	// Pre-Create 50,000 entities
	for i := range 50000 {
		id := world.Create("Generic").ID
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

	// Pre-Create entities
	for i := range 10000 {
		id := world.Create("Generic").ID
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
