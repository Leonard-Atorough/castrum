package benchmark

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/signal"
)

// colliderWorld builds n circle colliders on a grid.
// Spacing keeps pairs from overlapping, so the system
// pays its broad phase without narrow-phase hits.
func colliderWorld(b *testing.B, n int) (*core.World, []*core.Entity) {
	b.Helper()
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*signal.Hub, error) {
		return signal.NewHub(), nil
	}); err != nil {
		b.Fatal(err)
	}
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

func BenchmarkCollisionTickMoving(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, entities := colliderWorld(b, n)
			collisionTicks(b, world, entities, false)
		})
	}
}

// BenchmarkCollisionTickStatic reports one tick where nothing moved:
// the system's floor cost over a settled world.
func BenchmarkCollisionTickStatic(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, _ := colliderWorld(b, n)
			collisionTicks(b, world, nil, false)
		})
	}
}

// BenchmarkCollisionTickSparse moves one percent of the collider population.
func BenchmarkCollisionTickSparse(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, entities := colliderWorld(b, n)
			collisionTicks(b, world, entities[:n/100], false)
		})
	}
}

// BenchmarkCollisionTickContacts measures settled, persistent overlaps.
func BenchmarkCollisionTickContacts(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, entities := colliderWorld(b, n)
			for _, e := range entities {
				if err := e.Update(world, func(t *core.Transform) {
					t.Position = t.Position.Mul(0.5)
				}); err != nil {
					b.Fatal(err)
				}
			}
			collisionTicks(b, world, nil, false)
		})
	}
}

// BenchmarkCollisionDetectMoving excludes gameplay's transform writes.
func BenchmarkCollisionDetectMoving(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world, entities := colliderWorld(b, n)
			collisionTicks(b, world, entities, true)
		})
	}
}

func collisionTicks(b *testing.B, world *core.World, moving []*core.Entity, detectorOnly bool) {
	b.Helper()
	system := collision.NewSystem()
	ctx := drawCtx(world, 0)
	// Settle attachment, the index, and both contact history buffers.
	for range 3 {
		ctx.Tick++
		if err := system.Update(ctx); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if detectorOnly {
			b.StopTimer()
		}
		for _, e := range moving {
			if err := e.Update(world, func(t *core.Transform) {
				t.Position.X += 0.5
			}); err != nil {
				b.Fatal(err)
			}
		}
		if detectorOnly {
			b.StartTimer()
		}
		ctx.Tick++
		if err := system.Update(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
