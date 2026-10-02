package core

import (
	"github.com/Leonard-Atorough/castrum/geom"
)

// Transform is the 2d spatial state of an entity: position, rotation,
// and scale, relative to a position offset.
//
// A Transform always comes paired with a [PrevTransform]: spawn and
// AddComponent attach one automatically, snapshotting the position
// so the renderer can interpolate motion between fixed ticks.
type Transform struct {
	// Position represents the position of the transform in 2D space.
	Position geom.Vector2
	// Rotation represents the rotation of the transform in radians.
	Rotation float64
	// Scale represents the scale of the transform in 2D space. A zero
	// axis - the zero value - reads as unscaled: collection treats it
	// as 1.
	Scale geom.Vector2
	// Offset represents the offset of the transform relative to its position.
	Offset geom.Vector2
}

// PrevTransform is the engine-managed snapshot of an entity's
// position at the start of each fixed tick. Renderers interpolate
// between PrevTransform and the Transform by an alpha value to
// display smooth motion between ticks.
//
// The capture system writes it: game code reads it but never writes
// or removes it. A Transform without a PrevTransform never renders.
type PrevTransform struct {
	Position geom.Vector2
}

// NewPrevTransformCapture returns the engine system that snapshots
// every entity's Transform into its PrevTransform, keeping the
// previous tick's state available so renderers can interpolate
// motion between fixed ticks.
//
// It must run first in the fixed phase, before any gameplay system
// moves an entity - registered later, it snapshots the moved state
// and interpolation silently stops. Game.New registers this
// system; games never register it themselves.
func NewPrevTransformCapture() System {
	var update *Query
	return SystemFunc(func(ctx *Context) error {
		if update == nil {
			update = NewQuery(ctx.World).With(Transform{}, PrevTransform{})
		}

		// Snapshot entities that already carry a previous state:
		// zero-copy writes through the prefetched columns.
		for e := range update.Execute() {
			t, _ := e.Component[Transform]()
			e.Update(func(p *PrevTransform) { p.Position = t.Position })
		}
		return nil
	})
}
