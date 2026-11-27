package input

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// NewFrameSystem returns the pre-frame updater that resolves one frame of
// input into the frame view: it reads the snapshot resource and calls
// [ActionMap.Update]. castrum.New registers it first in the pre-frame
// schedule so user systems in that phase see fresh actions.
//
// It returns an error if the snapshot resource is not registered.
func NewFrameSystem(am *ActionMap) core.SystemFunc {
	return func(ctx *core.Context) error {
		snapshot, err := ctx.World.Resource[*Snapshot]()
		if err != nil {
			return fmt.Errorf("input: frame system: resolve snapshot: %w", err)
		}
		am.Update(snapshot, ctx.DeltaTime.Seconds())
		return nil
	}
}

// NewTickSystem returns the pre-fixed updater that delivers the press and
// release edges accumulated since the previous tick by calling
// [ActionMap.Tick]. castrum.New registers it first in the pre-fixed
// schedule so fixed systems read latched edges.
func NewTickSystem(am *ActionMap) core.SystemFunc {
	return func(*core.Context) error {
		am.Tick()
		return nil
	}
}
