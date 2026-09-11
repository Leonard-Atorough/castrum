package benchmark

// Benchmark configuration constants for consistency across all benchmarks
const (
	// DefaultEntityCount is the standard number of entities for benchmarks
	// that need a populated world. This ensures comparability across benchmarks.
	DefaultEntityCount = 10000

	// BenchmarkEntityPoolSize is the standard pool size for entity reuse benchmarks
	BenchmarkEntityPoolSize = 10000
)
