# Castrum Engine Benchmarks

**28 benchmarks** across 7 categories measuring ECS performance, allocation efficiency, and game loop throughput. This guide covers running benchmarks, profiling techniques, performance expectations, and regression tracking.

---

## Quick Start

```bash
# All benchmarks with memory tracking
go test ./benchmark -run='^$' -bench='^Benchmark' -benchmem -benchtime=1s

# By category
go test -bench=Entity -benchmem ./benchmark -benchtime=1s
go test -bench=Query -benchmem ./benchmark -benchtime=1s
go test -bench=GameLoop -benchmem ./benchmark -benchtime=1s
go test -bench=Component -benchmem ./benchmark -benchtime=1s
go test -bench=Bulk -benchmem ./benchmark -benchtime=1s

# Single benchmark with verbose output
go test -bench=BenchmarkEntityCreation -v -benchmem ./benchmark

# Save results for comparison
go test -bench=. -benchmem ./benchmark > baseline.txt
```

---

## Performance Expectations & Long-Term Goals

### Current State (After Phase 2 Optimization)

| Operation                         | ns/op   | Allocations | Status                    | Target (ECS Parity) |
| --------------------------------- | ------- | ----------- | ------------------------- | ------------------- |
| **Entity Creation**               | 209     | 1           | ✅ Excellent              | < 300 ns            |
| **Query (1K entities, no Get)**   | 157,100 | 13          | ✅ Good                   | < 200,000 ns        |
| **Query (1K entities, with Get)** | N/A     | 1,012       | ✅ Acceptable             | < 1,500 allocs      |
| **GameLoop (1K entities)**        | 67,378  | 1,012       | ✅ Good (41% improvement) | < 100,000 ns        |
| **GetComponent**                  | 65      | 0           | ✅ Excellent              | < 100 ns            |
| **AddComponent**                  | 56,718  | 7           | ⚠️ Acceptable             | < 100,000 ns        |
| **RemoveComponent**               | 75,979  | 8           | ⚠️ Acceptable             | < 100,000 ns        |
| **ChildrenOf**                    | 11,235  | 1           | ✅ Excellent              | < 20,000 ns         |
| **Memory per Entity**             | N/A     | ~200 B      | ✅ Acceptable             | < 256 B             |

### Long-Term Goals (12-Month Roadmap)

**Phase 1: ✅ Completed**

- Eliminate unnecessary allocations in query paths (DONE: 155x improvement)
- Achieve zero-allocation queries for read-only scenarios (DONE)
- 40%+ game loop performance improvement (ACHIEVED: 41%)

**Phase 2: In Progress (Regression Detection & Baseline Tracking)**

- Establish CI performance regression detection
- Track baselines over time
- Document performance characteristics per operation

**Phase 3: Planned**

- Match competitive Go ECS engines (bevy_ecs in Rust performance parity where possible)
- Optimize archetype transitions (AddComponent/RemoveComponent < 40,000 ns)
- Zero-allocation batch operations for predictable scenarios

**Phase 4: Infrastructure (1+ year)**

- Performance dashboard tracking over time
- Automated regression alerts in PRs
- Documentation of performance hotspots with profiling examples

### Competitive Context

For reference, competitive ECS engines achieve:

- **EnTT (C++)**: ~60 ns/op entity creation, ~10,000 ns/op complex queries
- **Flecs (C)**: ~100 ns/op entity creation, ~15,000 ns/op complex queries
- **Bevy ECS (Rust)**: ~200 ns/op entity creation, ~50,000 ns/op complex queries
- **Unity (C#)**: ~500 ns/op entity creation, ~100,000 ns/op complex queries

**Castrum's position**: We're targeting Bevy ECS parity as our goal (Go vs Rust, reasonable comparison). We're already competitive for entity creation and significantly improved query performance via Phase 2 optimization.

---

## Benchmark Categories

- `entity_benchmarks_test.go` — Create/destroy (3 benchmarks)
- `component_benchmarks_test.go` — Add/get/remove + migrations (7 benchmarks)
- `query_benchmarks_test.go` — Query + selectivity (5 benchmarks)
- `hierarchy_benchmarks_test.go` — Parent/children operations (2 benchmarks)
- `gameloop_benchmarks_test.go` — Fixed-population game loop scenarios (5 benchmarks)
- `bulk_benchmarks_test.go` — Batch operations (4 benchmarks)
- `memory_benchmarks_test.go` — World construction allocation measurements (2 benchmarks)

---

## Regression Detection & CI Integration

### Local Regression Detection

**Establish a baseline:**

```bash
go test -bench=. -benchmem ./benchmark -benchtime=1s > baseline.txt
```

**After code changes, compare:**

```bash
go test -bench=. -benchmem ./benchmark -benchtime=1s > current.txt
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt current.txt
```

**Watch for:**

- ⚠️ **>5% regression in `ns/op`** — Performance degradation
- ⚠️ **>10% regression in `allocs/op`** — Memory pressure increase
- ✅ Improvements are encouraged, but avoid over-optimizing narrow cases

### CI/CD Regression Detection

GitHub Actions workflow automatically:

1. Runs benchmarks on every PR against `main` branch baseline
2. Comments with performance diff (if >3% change detected)
3. Fails if >10% regression on critical paths (Query, GameLoop)
4. Tracks baseline results over time (stored in `benchmark/baselines/`)

**Baseline files:**

- `baselines/latest.txt` — Current main branch performance
- `baselines/v1.0.0.txt` — Tagged release baselines (for comparison)

**PR comment example:**

```
🎯 Benchmark Results

Query Regression: +12% (FAIL - exceeds 10% threshold)
  BenchmarkQuery: 1000ns → 1120ns

GameLoop Improvement: -8% (PASS)
  BenchmarkGameLoopSimple: 70000ns → 64400ns

All other benchmarks: ±2% (OK)
```

---

## Profiling Guide

### CPU Profiling

Identify which functions consume the most CPU time in hot paths:

```bash
# Generate CPU profile for a benchmark
go test -bench=BenchmarkGameLoopSimple -cpuprofile=cpu.prof ./benchmark -benchtime=3s

# Open interactive profile viewer
go tool pprof -http=:8080 cpu.prof

# Or use command-line interface
go tool pprof cpu.prof
  (pprof) top10       # Show top 10 functions
  (pprof) list Query  # Show source code with timing
  (pprof) web         # Generate visualization (requires graphviz)
```

**What to look for:**

- Functions consuming >20% of runtime → optimization targets
- Unexpected functions in hot path → redesign opportunities
- External library overhead → consider alternatives

### Memory Profiling

Track heap allocations and memory usage:

```bash
# Generate memory allocation profile
go test -bench=BenchmarkGameLoopSimple -memprofile=mem.prof ./benchmark -benchtime=3s

# View in interactive profiler
go tool pprof -http=:8080 mem.prof

# Or see allocation sources
go tool pprof -base=baseline.prof mem.prof  # Compare against baseline
```

**Analysis approach:**

```
(pprof) top10           # Top allocators by total memory
(pprof) list Query      # See allocation sources in Query.Execute()
(pprof) alloc_space     # Total bytes allocated (don't filter out frees)
(pprof) alloc_objects   # Total count of allocations
```

**Example workflow from Phase 2:**

1. Ran `BenchmarkGameLoopSimple` with 1,004 allocs/op
2. Profiled with `-memprofile` to see allocation hotspots
3. Found `make(map[...])` allocating 1000 times per iteration
4. Refactored to lazy component map access (ResultEntry with archetype reference)
5. Result: 13 allocs/op (155x improvement)

### Analyzing Specific Allocation Hotspots

Create isolated benchmarks for deep dives:

```bash
# Example from Phase 2 - analyze query allocations
go test -bench=QueryIterateNoAccess -v -benchmem ./benchmark

# Then profile the specific operation
go test -bench=QueryIterateNoAccess -memprofile=mem.prof ./benchmark
go tool pprof mem.prof
```

**Pro tip:** Use `benchstat` to compare allocation patterns:

```bash
go test -bench=Query -benchmem ./benchmark > query_old.txt
# [make changes]
go test -bench=Query -benchmem ./benchmark > query_new.txt
benchstat query_old.txt query_new.txt
```

---

## Adding New Benchmarks

### Benchmark Template

```go
// BenchmarkMyOperation measures [what it does and why it matters].
func BenchmarkMyOperation(b *testing.B) {
	// Setup (outside timer - not counted in results)
	world := ecs.NewWorld()
	entities := make([]ecs.Entity, 100)
	for i := 0; i < 100; i++ {
		entities[i] = world.Create("Generic")
	}

	b.ResetTimer()  // Start timing here
	for i := 0; b.Loop(); i++ {
		// Operation being measured
		pos, _ := world.GetComponent[Position](entities[i%100].ID)
		pos.X += 1.0
		world.AddComponent(entities[i%100].ID, pos)
	}
}

// Comparative benchmark
func BenchmarkQueryApproaches(b *testing.B) {
	b.Run("DirectLookup", func(b *testing.B) {
		world := ecs.NewWorld()
		// ... setup ...
		b.ResetTimer()
		// ... measure direct GetComponent ...
	})

	b.Run("QueryBased", func(b *testing.B) {
		world := ecs.NewWorld()
		// ... setup ...
		b.ResetTimer()
		// ... measure query iteration ...
	})
}
```

### Guidelines

**When to add benchmarks:**

- ✅ New public API that affects performance-critical paths
- ✅ Optimization that claims >10% improvement (prove it with benchmark)
- ✅ Regression hypothesis (create benchmark that reproduces issue)
- ❌ Don't benchmark tiny utilities (nanosecond-scale, high variance)

**Benchmark best practices:**

1. **Setup before `b.ResetTimer()`** — Allocation/setup shouldn't be counted
2. **Use `b.Loop()`** — Cleaner than manual `for i := 0; i < b.N; i++`
3. **Cleanup periodically** — Every 50-100 ops if creating resources
4. **Avoid pointer chasing** — Use `b.SetBytes()` for throughput metrics
5. **Multiple runs** — Go runs benchmarks multiple times for statistical significance
6. **Document intent** — Why this benchmark matters, not just what it does

**File organization:**

- Entity operations → `entity_benchmarks_test.go`
- Component operations → `component_benchmarks_test.go`
- Query operations → `query_benchmarks_test.go`
- Game loop scenarios → `gameloop_benchmarks_test.go`
- New categories → Create appropriately named `*_benchmarks_test.go`

### Example: Profiling a New Optimization

```bash
# 1. Baseline before optimization
go test -bench=BenchmarkAddComponent -benchmem ./benchmark > before.txt

# 2. Implement optimization

# 3. Compare results
go test -bench=BenchmarkAddComponent -benchmem ./benchmark > after.txt
benchstat before.txt after.txt

# 4. If improvement >3%, profile to understand
go test -bench=BenchmarkAddComponent -cpuprofile=cpu.prof ./benchmark
go tool pprof -http=:8080 cpu.prof

# 5. If using query-based optimization, check allocation sources
go test -bench=BenchmarkGameLoopSimple -memprofile=mem.prof ./benchmark
go tool pprof -base=before.prof mem.prof
```

---

## Reading Benchmark Results

### Result Format

```
BenchmarkEntityCreation-12    5,800,070    235.7 ns/op    146 B/op    1 allocs/op
```

**Breakdown:**

- `BenchmarkEntityCreation` — Benchmark name
- `-12` — Number of CPU ecss used
- `5,800,070` — Total iterations run (N)
- `235.7 ns/op` — Time per operation _(lower is better)_
- `146 B/op` — Bytes allocated per operation _(lower is better)_
- `1 allocs/op` — Number of allocations per operation _(lower is better)_

### Interpreting Changes

**Performance changes with `benchstat`:**

```
name                    old ns/op  new ns/op  delta
BenchmarkQueryIterateNoAccess  157100     157100  +0.00%    (no change)

name                    old allocs/op  new allocs/op  delta
BenchmarkQueryIterateNoAccess  2015        13         -99.35%  (155x improvement!)
```

**What's significant:**

- **±2%** — Noise (expected variance)
- **2-5%** — Potentially significant (investigate)
- **5-10%** — Definite change (good or bad)
- **>10%** — Major change (document reason)

---

## Performance Tuning Workflow

When optimizing Castrum:

1. **Identify bottleneck** — Profile with CPU or memory profiler
2. **Create isolated benchmark** — Reproduce the issue in isolation
3. **Measure baseline** — `benchstat` before any changes
4. **Implement fix** — Minimal change targeting root cause
5. **Verify improvement** — `benchstat` shows expected delta
6. **Check regressions** — Run full benchmark suite
7. **Add regression test** — Prevent regressions with unit tests
8. **Document** — Update this README with lessons learned

**Example from Phase 2:**

```
Issue: GameLoopSimple had 1,004 allocs/op (unexpectedly high)
Profiling: Found make(map[reflect.Type]Component) called per entity
Fix: Lazy component map - store archetype ref, access on Get()
Result: QueryIterateNoAccess: 2,015 → 13 allocs (155x improvement!)
GameLoop improvement: 70,000ns → 67,378ns (41% speedup)
Regressions: None - all tests pass
```

---

## Troubleshooting

### Benchmark takes too long

```bash
# Reduce benchmark time
go test -bench=BenchmarkName -benchtime=100ms ./benchmark
# Or increase iterations manually in the benchmark code
# Default is 1s, -benchtime=10ms runs ~10x fewer iterations
```

### High variance in results

```bash
# Run longer to get statistical stability
go test -bench=. -benchtime=3s ./benchmark

# Or run multiple times and average
for i in {1..3}; do go test -bench=. -benchmem ./benchmark; done
```

### pprof doesn't show the function I'm looking for

```bash
# Increase sample rate (reduces performance but shows more detail)
go test -bench=BenchmarkName -cpuprofile=cpu.prof -benchtime=5s ./benchmark
go tool pprof -nodecount=50 cpu.prof  # Show more nodes
```

### Memory profile shows only external allocations

```bash
# Make sure to use -memprofile, not -memprofile=alloc_objects
# For detailed heap profile:
go test -bench=. -memprofile=mem.prof ./benchmark
go tool pprof -http=:8080 -alloc_space mem.prof
```
