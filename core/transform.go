package core

import (
	"github.com/Leonard-Atorough/castrum/geom"
)

// Transform stores an entity's position, rotation, scale, and offset in 2D
// space.
//
// When [World.NewEntity] or [Entity.AddComponent] adds a [Transform], the
// world also adds a [PrevTransform] if one is not already present.
type Transform struct {
	// Position is the entity's position in 2D space.
	Position geom.Vector2
	// Rotation is the entity's rotation in radians.
	Rotation float64
	// Scale is the entity's 2D scale. A zero component is treated as 1.
	Scale geom.Vector2
	// Offset is the transform's position offset.
	Offset geom.Vector2
}

// PrevTransform stores an entity's transform at the start of a fixed tick.
// Renderers interpolate between it and [Transform] to smooth motion between
// ticks.
//
// Rotation is interpolated as a numeric value, not along the shortest arc.
//
// The capture system updates this component; game code should treat it as
// read-only. A [Transform] without a [PrevTransform] is not rendered.
type PrevTransform struct {
	// Position is the snapshot position.
	Position geom.Vector2
	// Rotation is the snapshot rotation in radians.
	Rotation float64
	// Scale is the snapshot scale, with zero components normalized to 1.
	Scale geom.Vector2
}

// snapshot returns t's motion state for a PrevTransform. A zero
// scale axis normalizes to 1: the snapshot must read the same
// zero-value rule as the transform, or interpolation lerps from 0
// and the entity pops at spawn.
func (t Transform) snapshot() PrevTransform {
	scale := t.Scale
	if scale.X == 0 {
		scale.X = 1
	}
	if scale.Y == 0 {
		scale.Y = 1
	}
	return PrevTransform{Position: t.Position, Rotation: t.Rotation, Scale: scale}
}

// PrevTransformSystemName is the name under which castrum.New registers the
// previous-transform capture system in the fixed schedule.
const PrevTransformSystemName = "engine.prev-transform"

// NewPrevTransformCapture returns the system that updates each existing
// [PrevTransform] from its entity's [Transform] for rendering interpolation.
//
// Run it before systems that change transforms so the snapshot represents
// the start of the tick. castrum.New registers it first in the fixed phase
// under [PrevTransformSystemName].
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
			e.Update(func(p *PrevTransform) { *p = t.snapshot() })
		}
		return nil
	})
}
