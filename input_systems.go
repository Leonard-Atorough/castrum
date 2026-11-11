package castrum

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

func (g *Game) wireInput(bindings input.Bindings) error {
	am, err := input.New(bindings, 0)
	if err != nil {
		return fmt.Errorf("castrum: input bindings: %w", err)
	}

	if err := g.world.Provide(func(*core.World) (*input.ActionMap, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide action map: %w", err)
	}

	if err := g.AddSystem(core.PhasePreFrame, input.FrameSystemName, newInputUpdateSystem(am)); err != nil {
		return err
	}

	g.Context().Actions = am
	return g.AddSystem(core.PhaseFixed, input.TickSystemName, newInputTickSystem(am))
}

// Capturing the provided map avoids per-frame resource resolution.
// New registers this before systems added normally to the pre-frame phase.
func newInputUpdateSystem(am *input.ActionMap) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		am.Update(ctx.Input, ctx.DeltaTime.Seconds())
		return nil
	})
}

// New registers this before systems added normally to the fixed phase.
func newInputTickSystem(am *input.ActionMap) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		am.Tick()
		return nil
	})
}
