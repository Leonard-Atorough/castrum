package castrum

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/engine"
	"github.com/Leonard-Atorough/castrum/input"
)

// coreWiring declares the engine-owned registration for the system that
// lives in core and therefore cannot import engine itself.
func coreWiring() engine.Registration {
	return engine.Registration{
		Name: "engine.core",
		Systems: []engine.SystemSpec{
			{Name: core.PrevTransformSystemName, Phase: core.PhasePreFixed, Slot: engine.First, New: core.NewPrevTransformCapture},
		},
	}
}

// inputWiring declares the input registration in the root package because
// core imports input (for [core.Context.Input]), so input cannot import
// engine without a cycle.
func inputWiring(bindings input.Bindings, deadzone float64) engine.Registration {
	var am *input.ActionMap
	return engine.Registration{
		Name: "input",
		Resources: []func(*core.World) error{
			func(w *core.World) error {
				m, err := input.New(bindings, deadzone)
				if err != nil {
					return err
				}
				am = m
				return w.Provide(func(*core.World) (*input.ActionMap, error) { return am, nil })
			},
		},
		Systems: []engine.SystemSpec{
			{
				Name:  input.FrameSystemName,
				Phase: core.PhasePreFrame,
				Slot:  engine.First,
				New: func() core.System {
					return core.SystemFunc(func(ctx *core.Context) error {
						am.Update(ctx.Input, ctx.DeltaTime.Seconds())
						return nil
					})
				},
			},
			{
				Name:  input.TickSystemName,
				Phase: core.PhasePreFixed,
				Slot:  engine.First,
				New: func() core.System {
					return core.SystemFunc(func(ctx *core.Context) error {
						am.Tick()
						return nil
					})
				},
			},
		},
	}
}
