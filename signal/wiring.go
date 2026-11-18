package signal

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/engine"
)

const (
	// BeginFrameSystemName is the registered name of the pre-frame signal
	// boundary system.
	BeginFrameSystemName = "engine.signal-begin-frame"
	// EndFrameSystemName is the registered name of the post-frame signal
	// boundary system.
	EndFrameSystemName = "engine.signal-end-frame"
	// EndTickSystemName is the registered name of the post-fixed signal
	// boundary system.
	EndTickSystemName = "engine.signal-end-tick"
)

// Wiring returns the engine registration for the signal subsystem: the [Hub]
// resource and the frame/tick boundary systems.
func Wiring() engine.Registration {
	boundary := func(name string, phase core.Phase, slot engine.Slot, run func(*Hub)) engine.SystemSpec {
		return engine.SystemSpec{
			Name:  name,
			Phase: phase,
			Slot:  slot,
			New: func() core.System {
				return core.SystemFunc(func(ctx *core.Context) error {
					hub, err := ctx.World.Resource[*Hub]()
					if err != nil {
						return err
					}
					run(hub)
					return nil
				})
			},
		}
	}
	return engine.Registration{
		Name: "signal",
		Resources: []func(*core.World) error{
			func(w *core.World) error {
				return w.Provide(func(*core.World) (*Hub, error) { return NewHub(), nil })
			},
		},
		Systems: []engine.SystemSpec{
			boundary(BeginFrameSystemName, core.PhasePreFrame, engine.Opening, func(h *Hub) { h.BeginFrame() }),
			boundary(EndFrameSystemName, core.PhasePostFrame, engine.Closing, func(h *Hub) { h.EndFrame() }),
			boundary(EndTickSystemName, core.PhasePostFixed, engine.Closing, func(h *Hub) { h.EndTick() }),
		},
	}
}
