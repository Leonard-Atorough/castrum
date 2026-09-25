package core

import (
	"github.com/Leonard-Atorough/castrum/geom"
)

// PrevTransform is the engine-managed snapshot of an entity's position
// as it stood at the start of the current fixed tick — the end of the
// previous tick. Renderers interpolate between PrevTransform and
// Transform by alpha to display smooth motion between fixed ticks.
//
// The engine's capture system writes it; game code may read it but must
// never write it.
type PrevTransform struct {
	Position geom.Vector2
}

// NewPrevTransformCapture returns the engine system that snapshots every
// entity's Transform position into PrevTransform.
//
// It must run FIRST in the fixed phase — before gameplay systems move
// anything. The snapshot at tick start is the previous tick's end state;
// gameplay then advances Transform. The render pair is therefore
// (previous tick, current tick), which alpha interpolates between.
// Registering the capture last would snapshot the already-advanced
// Transform, making prev and curr identical and silently disabling
// interpolation.
//
// Entities spawned during a tick have no snapshot; the capture
// materializes their spawn position at the next tick start, so they
// first render at the beginning of the next tick. Game.New registers
// this system; games never register it themselves.
func NewPrevTransformCapture() System {
	var update, newborns *Query
	var pending []EntityID
	return SystemFunc(func(ctx *Context) error {
		if update == nil {
			update = NewQuery(ctx.World).
				With(Transform{}, PrevTransform{})
			newborns = NewQuery(ctx.World).
				With(Transform{}).
				Without(PrevTransform{})
		}

		// Snapshot entities that already carry a previous state:
		// zero-copy writes through the prefetched columns.
		for e := range update.Execute() {
			t, ok := e.Component[Transform]()
			if !ok {
				continue
			}
			e.SetComponent(PrevTransform{Position: t.Position})
		}

		// Entities spawned during the previous tick have no snapshot.
		// Collect their IDs, then attach PrevTransform after iteration —
		// structural changes are invalid during a pass. Their first
		// snapshot is their spawn position, so their first rendered
		// frame interpolates from where they spawned.
		pending = pending[:0]
		for e := range newborns.Execute() {
			pending = append(pending, e.ID())
		}
		for _, id := range pending {
			entity := NewEntity(id)
			t, ok := entity.Component[Transform](ctx.World)
			if !ok {
				continue
			}
			if err := entity.AddComponent(ctx.World, PrevTransform{Position: t.Position}); err != nil {
				return err
			}
		}
		return nil
	})
}
