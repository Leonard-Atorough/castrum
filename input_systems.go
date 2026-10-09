package castrum

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

// newInputUpdateSystem returns the engine system that resolves one
// frame of input into the action map. It closes over the map, so no
// per-frame resource lookup runs and no resolution error path
// exists: wireInput provides that same instance to the world, and
// game systems reach it through the ActionMap resource.
//
// It must run FIRST in the frame phase - before any system reads the
// map's frame view. Registered after a reader, the reader sees the
// previous frame's view: one frame stale, silently. That is why
// castrum.New registers it under [input.FrameSystemName] before any
// user system can exist.
//
// castrum.New registers this system; games never register it themselves.
func newInputUpdateSystem(am *input.ActionMap) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		am.Update(ctx.Input, ctx.DeltaTime.Seconds())
		return nil
	})
}

// newInputTickSystem returns the engine system that delivers the
// press and release events accumulated since the last tick to the
// tick view, so every system running in that tick reads the same
// events.
//
// It must run FIRST in the fixed phase - before gameplay systems
// read Pressed or Released. Registered after a gameplay system, that
// system reads the previous tick's delivered events: one tick stale,
// silently. castrum.New registers it under [input.TickSystemName]
// right after the prev-transform capture, both ahead of every user
// system.
//
// castrum.New registers this system; games never register it themselves.
func newInputTickSystem(am *input.ActionMap) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		am.Tick()
		return nil
	})
}
