# Castrum Engine Benchmarks

28 benchmarks in 7 categories: entity operations, component operations, queries, hierarchy, batch operations, game loops, and memory.

## Quick Start

```bash
# All benchmarks
go test -bench=. -benchmem -run=^$ .

# By category
go test -bench=Entity -benchmem -run=^$ .
go test -bench=Query -benchmem -run=^$ .
go test -bench=GameLoop -benchmem -run=^$ .

# With profiling
go test -bench=. -cpuprofile=cpu.prof -benchmem -run=^$ .
go tool pprof cpu.prof
```

## Performance Baseline

| Metric | Result | Status |
|--------|--------|--------|
| Entity creation | 2.94M ops/sec | ✅ 3x target |
| Query (10K entities) | 53.4 μs | ✅ 18x faster than 1ms |
| Memory per entity | ~200 bytes | ✅ Acceptable |
| Sparse query (1%) | 1.2 μs | ✅ Highly efficient |
| Dense query (100%) | 95.9 μs | ✅ Acceptable |

## Key Insights

- **Query Selectivity**: 95x variance between sparse (1%) and dense (100%) results. Allocation dominates.
- **Game Loop**: Spawning is 36x slower than simple query — entity creation is the bottleneck.
- **Component Operations**: Remove is 2x more expensive than add.
- **GetComponent**: Very fast (65 ns) — direct lookup, no allocations.

## Benchmark Organization

- `entity_benchmarks_test.go` — Create/destroy (3)
- `component_benchmarks_test.go` — Add/get/remove + migrations (7)
- `query_benchmarks_test.go` — Query + selectivity (5)
- `hierarchy_benchmarks_test.go` — Parent/children (2)
- `gameloop_benchmarks_test.go` — Realistic scenarios (5)
- `bulk_benchmarks_test.go` — Batch operations (4)
- `memory_benchmarks_test.go` — Memory overhead (2)

## Adding New Benchmarks

```go
// BenchmarkMyOperation measures [what it does].
func BenchmarkMyOperation(b *testing.B) {
	// Setup (outside timer)
	world := core.NewWorld()
	entity := world.Create("Generic")
	
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		// Operation being measured
		world.AddComponent(entity.ID, Position{X: float64(i), Y: 0})
	}
}

// Compare approaches
func BenchmarkComparison(b *testing.B) {
	b.Run("Approach1", func(b *testing.B) { /*...*/ })
	b.Run("Approach2", func(b *testing.B) { /*...*/ })
}
```

**Rules:**
- Setup before `b.ResetTimer()`
- Use `b.Loop()` for iterations
- Batch cleanup every 50-100 ops if needed
- Place in appropriate `*_benchmarks_test.go` file

## Regression Detection

```bash
# Baseline
go test -bench=. -benchmem -run=^$ . > baseline.txt

# After changes
go test -bench=. -benchmem -run=^$ . > current.txt

# Compare
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt current.txt
```

Watch for >5% change in `ns/op` or `allocs/op`.

## Profiling

```bash
# CPU profile
go test -bench=BenchmarkName -cpuprofile=cpu.prof -run=^$ .
go tool pprof -http=:8080 cpu.prof

# Memory profile  
go test -bench=BenchmarkName -memprofile=mem.prof -run=^$ .
go tool pprof -http=:8080 mem.prof
```

## Result Format

```
BenchmarkEntityCreation-12    514358    340.0 ns/op    165 B/op    1 allocs/op
```

- `340.0 ns/op` — Time per operation (lower is better)
- `165 B/op` — Bytes allocated per operation (lower is better)
- `1 allocs/op` — Number of allocations (lower is better)

## CI/CD Integration

```yaml
- name: Benchmarks
  run: go test -bench=. -benchmem -run=^$ ./benchmark | tee results.txt

- name: Check Regressions
  run: |
    go install golang.org/x/perf/cmd/benchstat@latest
    benchstat baseline.txt results.txt
```
