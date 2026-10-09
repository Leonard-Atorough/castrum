package benchmark

import (
	"testing"
)

// BenchmarkCollectShapes reports the collector's per-frame floor
// over shape sprites: collect, interpolate, cull, and sort, with no
// asset resolution in the loop.
func BenchmarkCollectShapes(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, collector := spriteWorld(b, n, true, 1)
			ctx := drawCtx(world, 0.5)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				list, err := collector.Collect(ctx)
				if err != nil {
					b.Fatal(err)
				}
				sinkI += len(list.Items)
			}
		})
	}
}

// BenchmarkCollectTextures adds texture sprites: each item resolves
// through the asset server's warm cache, the realistic per-frame
// path for sprite-heavy scenes.
func BenchmarkCollectTextures(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, collector := spriteWorld(b, n, false, 1)
			ctx := drawCtx(world, 0.5)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				list, err := collector.Collect(ctx)
				if err != nil {
					b.Fatal(err)
				}
				sinkI += len(list.Items)
			}
		})
	}
}

// BenchmarkCollectCulled reports collection when three quarters of
// the sprites sit outside the viewport: off-screen sprites still
// cost collection but never reach the draw list.
func BenchmarkCollectCulled(b *testing.B) {
	world, collector := spriteWorld(b, 1000, false, 0.25)
	ctx := drawCtx(world, 0.5)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		list, err := collector.Collect(ctx)
		if err != nil {
			b.Fatal(err)
		}
		sinkI += len(list.Items)
	}
}
