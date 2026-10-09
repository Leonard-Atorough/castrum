package benchmark

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// colliderWorld builds n circle colliders on a grid with the given
// jitter seed. Spacing keeps pairs from overlapping, so the system
// pays its broad phase without narrow-phase hits.
func colliderWorld(b *testing.B, n int) (*core.World, []*core.Entity) {
	b.Helper()
	world := core.NewWorld()
	entities := make([]*core.Entity, 0, n)
	for i := range n {
		collider, err := collision.NewCollider(collision.Circle{Radius: 8})
		if err != nil {
			b.Fatal(err)
		}
		collider.Layers = collision.Layers(1)
		collider.Mask = collision.Mask(1)
		e, err := world.NewEntity(
			core.Transform{Position: geom.Vector2{X: float64(i%64) * 24, Y: float64(i/64) * 24}},
			collider,
		)
		if err != nil {
			b.Fatal(err)
		}
		entities = append(entities, e)
	}
	return world, entities
}

// BenchmarkCollisionTickMoving reports one collision system tick
// where every collider moved: the dirty-proxy broad phase plus
// candidate narrow-phase tests.
func BenchmarkCollisionTickMoving(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, entities := colliderWorld(b, n)
			system := collision.NewSystem()
			ctx := drawCtx(world, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				ctx.Tick = uint64(i) + 1
				for _, e := range entities {
					if err := e.Update(world, func(t *core.Transform) {
						t.Position.X += 0.5
					}); err != nil {
						b.Fatal(err)
					}
				}
				if err := system.Update(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCollisionTickStatic reports one tick where nothing moved:
// the system's floor cost over a settled world.
func BenchmarkCollisionTickStatic(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, _ := colliderWorld(b, n)
			system := collision.NewSystem()
			ctx := drawCtx(world, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				ctx.Tick = uint64(i) + 1
				if err := system.Update(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
