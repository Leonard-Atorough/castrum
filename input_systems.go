package castrum

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

func (g *Game) wireInput(bindings input.Bindings, deadzone float64) error {
	am, err := input.New(bindings, deadzone)
	if err != nil {
		return fmt.Errorf("castrum: input bindings: %w", err)
	}

	if err := g.world.Provide(func(*core.World) (*input.ActionMap, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide action map: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFrame, input.FrameSystemName, input.NewFrameSystem(am)); err != nil {
		return err
	}
	return g.addEngineFirst(core.PhasePreFixed, input.TickSystemName, input.NewTickSystem(am))
}
