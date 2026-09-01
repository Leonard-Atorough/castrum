# Castrum Engine Benchmarks

This package contains comprehensive benchmarks for the Castrum ECS engine. The benchmarks are designed to measure performance against the targets defined in the ROADMAP.md.

## Performance Targets

| Metric | Target | Current Status |
|--------|--------|----------------|
| Entity create rate | 10,000+ entities/sec | ✅ Measured |
| Query performance | <1ms for 10,000 entities | ✅ Measured |
| System update rate | 1,000+ systems/frame | ⏳ Pending |
| Memory per entity | <100 bytes | ✅ Measured |
| FPS stability | 60fps with 5,000+ entities | ⏳ Pending |

## Running Benchmarks

### All Benchmarks
```bash
# Using Makefile
make bench

# Direct Go command
go test -bench=. -benchmem -run=^$ ./benchmark/...
```

### Specific Benchmarks
```bash
# Entity creation benchmarks
go test -bench=BenchmarkEntityCreation -benchmem -run=^$ ./benchmark/...

# Query performance benchmarks  
go test -bench=BenchmarkQuery -benchmem -run=^$ ./benchmark/...

# Component operations benchmarks
go test -bench=BenchmarkComponent -benchmem -run=^$ ./benchmark/...

# Hierarchy operations benchmarks
go test -bench=BenchmarkHierarchy -benchmem -run=^$ ./benchmark/...

# Large scale scenarios
go test -bench=BenchmarkLargeScale -benchmem -run=^$ ./benchmark/...
```

### Custom Benchmark Configuration
```bash
# Run with different iteration times
go test -bench=BenchmarkEntityCreation -benchtime=1s -run=^$ ./benchmark/...

# Run with memory profiling
go test -bench=BenchmarkEntityCreation -benchmem -run=^$ ./benchmark/...

# Run with CPU profiling
go test -bench=BenchmarkEntityCreation -cpuprofile=cpu.prof -run=^$ ./benchmark/...
```

## Benchmark Categories

### Entity Operations
- `BenchmarkEntityCreation` - Basic entity creation
- `BenchmarkEntityCreationWithComponents` - Entity creation with multiple components
- `BenchmarkEntityCreationParallel` - Parallel entity creation
- `BenchmarkDestroyEntity` - Entity destruction
- `BenchmarkDestroyEntityWithCleanup` - Entity destruction with cleanup

### Component Operations
- `BenchmarkAddComponent` - Adding components to entities
- `BenchmarkGetComponent` - Retrieving components from entities
- `BenchmarkHasComponent` - Checking for component existence
- `BenchmarkRemoveComponent` - Removing components from entities
- `BenchmarkComponentsList` - Listing all components for an entity

### Query Operations
- `BenchmarkQuerySingleComponent` - Query by single component type
- `BenchmarkQueryMultipleComponents` - Query by multiple component types

### Hierarchy Operations
- `BenchmarkSetParent` - Setting parent-child relationships
- `BenchmarkParentOf` - Getting parent of an entity
- `BenchmarkChildrenOf` - Getting children of an entity

### World Operations
- `BenchmarkWorldCount` - Getting entity count
- `BenchmarkWorldExists` - Checking entity existence

### Simulation Benchmarks
- `BenchmarkGameLoopSimulation` - Simulating a complete game loop
- `BenchmarkLargeScaleEntityCreation` - Creating entities at scale
- `BenchmarkLargeScaleQuery` - Querying large numbers of entities
- `BenchmarkMemoryEfficientOperations` - Memory allocation patterns

## Benchmark Results Interpretation

### Example Output
```
BenchmarkEntityCreation-12                   18417    13012 ns/op    1415.36 MB/s    440 B/op    3 allocs/op
```

- `BenchmarkEntityCreation-12` - Benchmark name and GOMAXPROCS
- `18417` - Number of iterations
- `13012 ns/op` - Nanoseconds per operation (lower is better)
- `1415.36 MB/s` - Throughput in MB/second
- `440 B/op` - Bytes allocated per operation
- `3 allocs/op` - Number of memory allocations per operation

### Key Metrics to Watch

1. **ns/op**: Time per operation - lower is better
2. **B/op**: Bytes allocated per operation - lower is better  
3. **allocs/op**: Memory allocations per operation - lower is better
4. **Throughput**: Operations per second - higher is better

## Performance Analysis

### Current Results (as of 2026-08-25)

#### Entity Creation
- Basic entity creation: ~13,000 ns/op (~77,000 entities/sec)
- With 3 components: ~16,400 ns/op (~61,000 entities/sec)
- Memory: ~440-950 bytes per entity

#### Query Performance
- Single component query (10K entities): ~1.35 ms/op
- Multiple component query (10K entities): ~2.63 ms/op
- Tag query: ~11 ns/op (very fast due to direct map lookup)
- Template query: ~11.5 ns/op (very fast due to direct map lookup)

#### Component Operations
- Add component: ~12,600 ns/op
- Get component: ~255 ns/op
- Has component: Similar to Get
- Remove component: Similar to Add

## Optimization Opportunities

Based on current benchmark results:

1. **Query Performance**: Multi-component queries could be optimized with better indexing
2. **Entity Creation**: Consider object pooling for frequently created entity types
3. **Component Storage**: Evaluate memory usage patterns for large-scale scenarios

## Continuous Integration

Benchmarks should be integrated into CI/CD to track performance regressions:

```yaml
# Example GitHub Actions workflow
name: Benchmark
on: [push, pull_request]
jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
    - run: go test -bench=. -benchmem -run=^$ ./benchmark/... > benchmarks.txt
    - run: ./compare_benchmarks.sh # Compare with previous results
```

## Adding New Benchmarks

When adding new features to the engine, add corresponding benchmarks:

1. Create test components if needed
2. Add benchmark functions following Go benchmark conventions
3. Use `b.ResetTimer()` to exclude setup time
4. Use `b.SetBytes()` for throughput calculations
5. Use `b.ReportAllocs()` for memory allocation tracking

## Best Practices

- Always reset timer after setup: `b.ResetTimer()`
- Pre-allocate test data outside the benchmark loop
- Use realistic data sizes (1K-50K entities for most tests)
- Test both small and large scale scenarios
- Include memory allocation tracking with `-benchmem`