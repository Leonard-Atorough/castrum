package benchmark

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

// benchTracked is a marker component that opts into change tracking.
type benchTracked struct{ Value float64 }

func (benchTracked) TrackChanges() {}

// BenchmarkSpawnEntityTrackedComponent reports the cost of one entity
// entering the world with one tracked component. Compare against
// BenchmarkSpawnEntity for the untracked cost.
func BenchmarkSpawnEntityTrackedComponent(b *testing.B) {
	world := core.NewWorld()
	b.ReportAllocs()

	for b.Loop() {
		if _, err := world.NewEntity(benchTracked{Value: 1}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComponentSetTracked reports one SetComponent write on a tracked
// component, including the change-tick stamp.
func BenchmarkComponentSetTracked(b *testing.B) {
	world := core.NewWorld()
	e, err := world.NewEntity(benchTracked{})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()

	for b.Loop() {
		if err := e.SetComponent(world, benchTracked{Value: float64(sinkI)}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryChanged iterates a Changed-filtered query over n tracked
// entities in steady state: after the first pass no row matches, so each
// pass pays the full per-row tick scan. Compare against BenchmarkQuery1 for
// the unfiltered scan cost.
func BenchmarkQueryChanged(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world := core.NewWorld()
			for range n {
				if _, err := world.NewEntity(benchTracked{Value: 1}); err != nil {
					b.Fatal(err)
				}
			}
			query := core.NewQuery(world).With(benchTracked{}).Changed()
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for range query.Execute() {
					sinkI++
				}
			}
		})
	}
}

// BenchmarkQueryChangedWithWrite reports one fixed tick that writes one
// tracked component and then runs the Changed-filtered pass, so the pass
// both scans and matches.
func BenchmarkQueryChangedWithWrite(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world := core.NewWorld()
			pool := make([]core.Entity, 0, n)
			for range n {
				e, err := world.NewEntity(benchTracked{Value: 1})
				if err != nil {
					b.Fatal(err)
				}
				pool = append(pool, e)
			}
			query := core.NewQuery(world).With(benchTracked{}).Changed()
			for range query.Execute() {
				sinkI++
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				world.AdvanceTick()
				if err := pool[i%n].SetComponent(world, benchTracked{Value: float64(i)}); err != nil {
					b.Fatal(err)
				}
				for range query.Execute() {
					sinkI++
				}
			}
		})
	}
}
