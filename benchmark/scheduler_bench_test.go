package benchmark

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// BenchmarkAdvanceEmptySystems reports one engine advance with n
// registered systems that do nothing - the scheduler's per-tick
// floor. This is Bevy's "empty systems" shape.
func BenchmarkAdvanceEmptySystems(b *testing.B) {
	for _, n := range []int{1, 10, 50} {
		b.Run(intName(n), func(b *testing.B) {
			g, err := castrum.New()
			if err != nil {
				b.Fatal(err)
			}
			empty := core.SystemFunc(func(*core.Context) error { return nil })
			for range n {
				if err := g.AddSystem(core.PhaseFixed, "bench.empty", empty); err != nil {
					b.Fatal(err)
				}
			}
			if err := g.Startup(); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := g.Advance(time.Second / 60); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAdvanceSimpleSystem reports one engine advance whose
// fixed system moves every transform in the world - Bevy's "simple
// system" shape at the suite's population sizes.
func BenchmarkAdvanceSimpleSystem(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			g, err := castrum.New()
			if err != nil {
				b.Fatal(err)
			}
			world := g.World()
			for range n {
				if _, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 1, Y: 1}}); err != nil {
					b.Fatal(err)
				}
			}
			var moveQuery *core.Query
			move := core.SystemFunc(func(ctx *core.Context) error {
				if moveQuery == nil {
					moveQuery = core.NewQuery(ctx.World).With(core.Transform{})
				}
				dt := ctx.DeltaTime.Seconds()
				for e := range moveQuery.Execute() {
					e.Update(func(t *core.Transform) {
						t.Position.X += 10 * dt
					})
				}
				return nil
			})
			if err := g.AddSystem(core.PhaseFixed, "bench.move", move); err != nil {
				b.Fatal(err)
			}
			if err := g.Startup(); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := g.Advance(time.Second / 60); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
