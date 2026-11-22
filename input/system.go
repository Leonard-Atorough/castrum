package input

import (
	"github.com/Leonard-Atorough/castrum/core"
)

// NewFrameSystem returns the pre-frame updater that resolves one frame of
// input into the frame view: it reads the snapshot resource and calls
// [ActionMap.Update]. castrum.New registers it first in the pre-frame
// schedule so user systems in that phase see fresh actions.
//
// A missing snapshot resource reads as one frame with no input.
func NewFrameSystem(am *ActionMap) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		snapshot, _ := ctx.World.Resource[*Snapshot]()
		am.Update(snapshot, ctx.DeltaTime.Seconds())
		return nil
	})
}

// NewTickSystem returns the pre-fixed updater that delivers the press and
// release edges accumulated since the previous tick by calling
// [ActionMap.Tick]. castrum.New registers it first in the pre-fixed
// schedule so fixed systems read latched edges.
func NewTickSystem(am *ActionMap) core.System {
	return core.SystemFunc(func(*core.Context) error {
		am.Tick()
		return nil
	})
}
