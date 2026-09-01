package benchmark

// Benchmark suite for Castrum ECS engine.
//
// Files:
//   entity_benchmarks_test.go - Entity creation/destruction (3)
//   component_benchmarks_test.go - Component add/get/remove + migrations (7)
//   query_benchmarks_test.go - Query operations and selectivity (5)
//   hierarchy_benchmarks_test.go - Parent/children relationships (2)
//   gameloop_benchmarks_test.go - Realistic game loop scenarios (5)
//   bulk_benchmarks_test.go - Batch operations (4)
//   memory_benchmarks_test.go - Memory overhead measurement (2)
//   components.go - Test component types
//   utilities.go - Helper functions
//
// Run: go test -bench=. -benchmem -run=^$ .
// See README.md for details on profiling and regression detection.
