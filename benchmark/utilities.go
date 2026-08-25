package benchmark

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
)

// BenchmarkConfig holds configuration for benchmark scenarios
type BenchmarkConfig struct {
	EntityCount      int
	ComponentTypes   []reflect.Type
	TagNames         []string
	TemplateNames    []string
	HierarchyDepth   int
	HierarchyBreadth int
}

// DefaultBenchmarkConfig returns a sensible default configuration
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		EntityCount:      10000,
		ComponentTypes:   []reflect.Type{reflect.TypeFor[Position](), reflect.TypeFor[Velocity](), reflect.TypeFor[Health](), reflect.TypeFor[Sprite]()},
		TagNames:         []string{"Player", "Enemy", "Neutral", "Environment"},
		TemplateNames:    []string{"Generic", "Province", "City"},
		HierarchyDepth:   3,
		HierarchyBreadth: 10,
	}
}

// CreateBenchmarkWorld creates a world populated with entities for benchmarking
func CreateBenchmarkWorld(config BenchmarkConfig) *core.World {
	world := core.NewWorld()

	// Pre-compute component types for comparison
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()
	healthType := reflect.TypeFor[Health]()
	spriteType := reflect.TypeFor[Sprite]()

	// Create entities with various configurations
	for i := 0; i < config.EntityCount; i++ {
		template := config.TemplateNames[i%len(config.TemplateNames)]
		id := world.Create(template)

		// Add components
		for _, compType := range config.ComponentTypes {
			var comp ecs.Component
			switch compType {
			case posType:
				comp = Position{X: float64(i), Y: float64(i)}
			case velType:
				comp = Velocity{X: 1.0, Y: 1.0}
			case healthType:
				comp = Health{Value: 100 + (i % 100)}
			case spriteType:
				comp = Sprite{TextureID: "test", Width: 32, Height: 32}
			}
			if comp != nil {
				world.AddComponent(id, comp)
			}
		}

		// Add tags
		if len(config.TagNames) > 0 {
			tag := config.TagNames[i%len(config.TagNames)]
			world.AddTag(id, tag)
		}
	}

	// Create hierarchy if configured
	if config.HierarchyDepth > 0 && config.HierarchyBreadth > 0 {
		CreateBenchmarkHierarchy(world, config)
	}

	return world
}

// CreateBenchmarkHierarchy creates a hierarchical structure for benchmarking
func CreateBenchmarkHierarchy(world *core.World, config BenchmarkConfig) {
	var rootID ecs.EntityID

	// Create root entity
	rootID = world.Create("Root")
	world.AddTag(rootID, "Root")

	// Create hierarchical structure
	var currentLevel []ecs.EntityID
	currentLevel = append(currentLevel, rootID)

	for depth := 0; depth < config.HierarchyDepth; depth++ {
		var nextLevel []ecs.EntityID
		for _, parentID := range currentLevel {
			for breadth := 0; breadth < config.HierarchyBreadth; breadth++ {
				childID := world.Create("HierarchyNode")
				world.SetParent(childID, parentID)
				world.AddTag(childID, "HierarchyNode")
				nextLevel = append(nextLevel, childID)
			}
		}
		currentLevel = nextLevel
	}
}

// BenchmarkWorker represents a worker for parallel benchmarking
type BenchmarkWorker struct {
	ID       int
	World    *core.World
	Entities []ecs.EntityID
}

// CreateBenchmarkWorkers creates multiple worker worlds for parallel benchmarking
func CreateBenchmarkWorkers(workerCount int, entitiesPerWorker int) []*BenchmarkWorker {
	workers := make([]*BenchmarkWorker, workerCount)

	for i := range workerCount {
		world := core.NewWorld()
		entities := make([]ecs.EntityID, entitiesPerWorker)

		for j := range entitiesPerWorker {
			id := world.Create("WorkerEntity")
			world.AddComponent(id, Position{X: float64(j), Y: float64(j)})
			world.AddComponent(id, Velocity{X: 1.0, Y: 1.0})
			entities[j] = id
		}

		workers[i] = &BenchmarkWorker{
			ID:       i,
			World:    world,
			Entities: entities,
		}
	}

	return workers
}

// ParallelBenchmarkHelper helps run benchmarks in parallel with proper setup
func ParallelBenchmarkHelper(b *testing.B, workerCount int, entitiesPerWorker int, benchmarkFunc func(*testing.B, *BenchmarkWorker)) {
	workers := CreateBenchmarkWorkers(workerCount, entitiesPerWorker)
	if len(workers) == 0 {
		b.Fatalf("workerCount must be greater than zero")
	}

	var workerIdx uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		worker := workers[int(atomic.AddUint64(&workerIdx, 1)-1)%len(workers)]
		for pb.Next() {
			benchmarkFunc(b, worker)
		}
	})
}

// MeasureOperationTime measures the time taken for a single operation
func MeasureOperationTime(operation func()) float64 {
	return testing.AllocsPerRun(1, func() {
		operation()
	})
}

// ConcurrentBenchmark runs a benchmark with concurrent goroutines
func ConcurrentBenchmark(b *testing.B, goroutines int, operation func(int)) {
	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(goroutines)
		for j := range goroutines {
			go func(id int) {
				defer wg.Done()
				operation(id)
			}(j)
		}
		wg.Wait()
	}
}

// BenchmarkScenario defines a complete benchmarking scenario
type BenchmarkScenario struct {
	Name        string
	Setup       func() interface{}
	Benchmark   func(*testing.B, interface{})
	Teardown    func(interface{})
	Concurrency int
}

// RunScenario runs a complete benchmarking scenario
func RunScenario(b *testing.B, scenario BenchmarkScenario) {
	if scenario.Concurrency > 1 {
		b.Run(scenario.Name+"/Concurrent", func(b *testing.B) {
			data := scenario.Setup()
			defer scenario.Teardown(data)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					scenario.Benchmark(b, data)
				}
			})
		})
	} else {
		b.Run(scenario.Name, func(b *testing.B) {
			data := scenario.Setup()
			defer scenario.Teardown(data)

			b.ResetTimer()
			scenario.Benchmark(b, data)
		})
	}
}

// StressTestConfig configuration for stress testing
type StressTestConfig struct {
	Duration         int // in seconds
	InitialEntities  int
	OperationsPerSec int
	ComponentTypes   []reflect.Type
}

// RunStressTest runs a stress test on the ECS system
func RunStressTest(config StressTestConfig) (int, int, float64) {
	world := core.NewWorld()

	// Pre-compute component types for comparison
	posType := reflect.TypeFor[Position]()
	velType := reflect.TypeFor[Velocity]()

	// Create initial entities
	for i := 0; i < config.InitialEntities; i++ {
		id := world.Create("StressEntity")
		for _, compType := range config.ComponentTypes {
			var comp ecs.Component
			switch compType {
			case posType:
				comp = Position{X: float64(i), Y: float64(i)}
			case velType:
				comp = Velocity{X: 1.0, Y: 1.0}
			}
			if comp != nil {
				world.AddComponent(id, comp)
			}
		}
	}

	var operations int
	var errors int
	var totalTime float64

	// Run stress test
	// This is a simplified version - in practice you'd want to use proper timing
	for op := 0; op < config.OperationsPerSec*config.Duration; op++ {
		start := testing.AllocsPerRun(1, func() {
			// Perform various operations
			id := world.Create("StressEntity")
			world.AddComponent(id, Position{X: float64(op), Y: float64(op)})

			// Query some entities
			world.Query(reflect.TypeFor[Position]())

			// Remove some entities
			if op%100 == 0 && world.Count() > config.InitialEntities/2 {
				entities := world.Query(reflect.TypeFor[Position]())
				if len(entities) > 0 {
					world.Destroy(entities[0], false)
					world.Cleanup()
				}
			}
		})

		operations++
		totalTime += start
	}

	return operations, errors, totalTime
}

// BenchmarkComponentTypeRegistry benchmarks component type lookups
func BenchmarkComponentTypeRegistry(b *testing.B) {
	// This would be more relevant if we had a component registry
	// For now, just benchmark reflect.TypeFor

	for b.Loop() {
		_ = reflect.TypeFor[Position]()
		_ = reflect.TypeFor[Velocity]()
		_ = reflect.TypeFor[Health]()
		_ = reflect.TypeFor[Sprite]()
	}
}

// BenchmarkReflectionOverhead benchmarks the overhead of using reflection
func BenchmarkReflectionOverhead(b *testing.B) {
	world := core.NewWorld()

	// Create entity with components
	id := world.Create("Generic")
	world.AddComponent(id, Position{X: 1.0, Y: 2.0})

	// Direct access (no reflection)
	b.Run("Direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			comp, _ := world.GetComponent(id, reflect.TypeFor[Position]())
			if comp != nil {
				_ = comp.(Position)
			}
		}
	})

	// Cached reflection
	posType := reflect.TypeFor[Position]()
	b.Run("Cached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			comp, _ := world.GetComponent(id, posType)
			if comp != nil {
				_ = comp.(Position)
			}
		}
	})
}

// BenchmarkEntityReuse benchmarks entity reuse patterns
func BenchmarkEntityReuse(b *testing.B) {
	world := core.NewWorld()

	// Pre-create a pool of entities
	entityPool := make([]ecs.EntityID, 1000)
	for i := range entityPool {
		entityPool[i] = world.Create("PooledEntity")
	}

	for i := 0; b.Loop(); i++ {
		// Reuse entities from pool
		entityID := entityPool[i%1000]
		world.AddComponent(entityID, Position{X: float64(i), Y: float64(i)})
		world.AddComponent(entityID, Velocity{X: 1.0, Y: 1.0})

		// Clean up components for next use
		world.RemoveComponent(entityID, reflect.TypeFor[Position]())
		world.RemoveComponent(entityID, reflect.TypeFor[Velocity]())
	}
}
