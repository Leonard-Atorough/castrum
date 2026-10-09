package benchmark

import (
	"strconv"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

// benchSizes are the population sizes the go-ecs-benchmark suite
// reports against.
var benchSizes = []int{100, 1000, 10000}

// BenchmarkSpawnEntity reports the cost of one entity entering the
// world with one Transform component.
func BenchmarkSpawnEntity(b *testing.B) {
	world := core.NewWorld()
	b.ReportAllocs()

	for b.Loop() {
		if _, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 1, Y: 2}}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSpawnEntityThreeComponents is the same spawn with a
// three-component signature: Transform, Sprite, and a marker.
func BenchmarkSpawnEntityThreeComponents(b *testing.B) {
	world := core.NewWorld()
	b.ReportAllocs()

	for b.Loop() {
		if _, err := world.NewEntity(
			core.Transform{Position: geom.Vector2{X: 1, Y: 2}},
			render.Sprite{Drawable: render.CircleShape{Radii: geom.Vector2{X: 4, Y: 4}}},
			benchTag{},
		); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSpawnEntitiesBatch reports batched spawn: the world's
// NewEntities in batches of one hundred.
func BenchmarkSpawnEntitiesBatch(b *testing.B) {
	world := core.NewWorld()
	b.ReportAllocs()

	for b.Loop() {
		if _, err := world.NewEntities(100, core.Transform{Position: geom.Vector2{X: 1, Y: 2}}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDestroyEntity reports the cost of one entity leaving the
// world. The pool refills between timings so the measured work is
// destruction alone.
func BenchmarkDestroyEntity(b *testing.B) {
	world := core.NewWorld()
	pool := make([]*core.Entity, 0, 10000)
	next := 0
	refill := func() {
		pool = pool[:0]
		next = 0
		for range cap(pool) {
			e, err := world.NewEntity(core.Transform{})
			if err != nil {
				b.Fatal(err)
			}
			pool = append(pool, e)
		}
	}
	refill()
	b.ReportAllocs()

	for b.Loop() {
		if next >= len(pool) {
			b.StopTimer()
			refill()
			b.StartTimer()
		}
		if err := world.DestroyEntity(pool[next]); err != nil {
			b.Fatal(err)
		}
		next++
	}
}

// BenchmarkComponentRead reports a handle's component read through
// the world, the per-tick cost systems pay for state access.
func BenchmarkComponentRead(b *testing.B) {
	world := core.NewWorld()
	pool := make([]*core.Entity, 0, 10000)
	for range 10000 {
		e, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 3, Y: 4}})
		if err != nil {
			b.Fatal(err)
		}
		pool = append(pool, e)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		t, _ := pool[i%len(pool)].Component[core.Transform](world)
		sinkF += t.Position.X
	}
}

// BenchmarkComponentAddRemove reports one migration pair: adding
// and removing a component moves the entity across archetypes both
// ways.
func BenchmarkComponentAddRemove(b *testing.B) {
	world := core.NewWorld()
	e, err := world.NewEntity(core.Transform{})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()

	for b.Loop() {
		if err := e.AddComponent(world, render.Sprite{}); err != nil {
			b.Fatal(err)
		}
		if err := e.RemoveComponent[render.Sprite](world); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQuery1 iterates a one-component query over n entities.
func BenchmarkQuery1(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world := core.NewWorld()
			for range n {
				if _, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 1, Y: 1}}); err != nil {
					b.Fatal(err)
				}
			}
			query := core.NewQuery(world).With(core.Transform{})
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for e := range query.Execute() {
					t, _ := e.Component[core.Transform]()
					sinkF += t.Position.X
				}
			}
		})
	}
}

// BenchmarkQuery2 iterates a two-component query over n entities.
func BenchmarkQuery2(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world := core.NewWorld()
			for range n {
				if _, err := world.NewEntity(
					core.Transform{},
					render.Sprite{Drawable: render.CircleShape{Radii: geom.Vector2{X: 2, Y: 2}}},
				); err != nil {
					b.Fatal(err)
				}
			}
			query := core.NewQuery(world).With(core.Transform{}, render.Sprite{})
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for e := range query.Execute() {
					t, _ := e.Component[core.Transform]()
					sinkF += t.Position.X
				}
			}
		})
	}
}

// BenchmarkQuery3 iterates a three-component query over n entities,
// with the marker narrowing the match.
func BenchmarkQuery3(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(intName(n), func(b *testing.B) {
			world := core.NewWorld()
			for range n {
				if _, err := world.NewEntity(
					core.Transform{},
					render.Sprite{Drawable: render.CircleShape{Radii: geom.Vector2{X: 2, Y: 2}}},
					benchTag{},
				); err != nil {
					b.Fatal(err)
				}
			}
			query := core.NewQuery(world).With(core.Transform{}, render.Sprite{}, benchTag{})
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for e := range query.Execute() {
					t, _ := e.Component[core.Transform]()
					sinkF += t.Position.X
				}
			}
		})
	}
}

// BenchmarkQueryWhere adds a per-entry predicate to the iteration,
// the cost of a Where filter at one thousand entities.
func BenchmarkQueryWhere(b *testing.B) {
	world := core.NewWorld()
	for range 1000 {
		if _, err := world.NewEntity(
			core.Transform{Position: geom.Vector2{X: 1, Y: 1}},
			render.Sprite{},
		); err != nil {
			b.Fatal(err)
		}
	}
	query := core.NewQuery(world).With(core.Transform{}).Where(func(e core.Entry) bool {
		s, _ := e.Component[render.Sprite]()
		return s.Drawable != nil
	})
	b.ReportAllocs()

	for b.Loop() {
		for e := range query.Execute() {
			t, _ := e.Component[core.Transform]()
			sinkF += t.Position.X
		}
	}
}

// BenchmarkQueryConstruct reports the cost of building a query and
// running its first iteration, the engine's per-system lazy setup.
func BenchmarkQueryConstruct(b *testing.B) {
	world := core.NewWorld()
	for range 1000 {
		if _, err := world.NewEntity(core.Transform{}, render.Sprite{}); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()

	for b.Loop() {
		query := core.NewQuery(world).With(core.Transform{}, render.Sprite{})
		for e := range query.Execute() {
			sinkEnt = e.ID()
		}
	}
}

// intName renders a population size as a sub-benchmark name.
func intName(n int) string {
	return strconv.Itoa(n)
}
