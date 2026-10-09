package benchmark

import (
	"runtime"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

// BenchmarkMemoryPerEntity reports the heap cost of one entity with
// a typical component set: Transform, Sprite, and the engine's
// paired PrevTransform. It runs once over a fixed population - the
// benchmark regression workflow skips it - and reports bytes per
// entity as a custom metric.
func BenchmarkMemoryPerEntity(b *testing.B) {
	const population = 10000
	runtime.GC()
	world := core.NewWorld()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range population {
		if _, err := world.NewEntity(
			core.Transform{Position: geom.Vector2{X: 1, Y: 2}},
			render.Sprite{Drawable: render.CircleShape{Radii: geom.Vector2{X: 4, Y: 4}}},
		); err != nil {
			b.Fatal(err)
		}
	}
	runtime.ReadMemStats(&after)
	perEntity := float64(after.TotalAlloc-before.TotalAlloc) / population
	b.ReportMetric(perEntity, "bytes/entity")
}
