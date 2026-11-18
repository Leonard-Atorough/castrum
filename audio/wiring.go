package audio

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/engine"
)

// PrepareSystemName is the registered name of the frame-start audio system
// that builds the reconciliation system once the runner supplies a backend
// controller.
const PrepareSystemName = "engine.audio-prepare"

// Wiring returns the engine registration for the audio subsystem: the
// [Mixer], [Listener], and [Backend] resources and the two frame systems.
// The prepare system runs at the start of the frame phase so mid-phase
// attachments wait for the next pass; the reconciliation system runs at the
// end.
func Wiring() engine.Registration {
	var reconciler core.System
	return engine.Registration{
		Name: "audio",
		Resources: []func(*core.World) error{
			func(w *core.World) error {
				return w.Provide(func(*core.World) (*Mixer, error) { return NewMixer(), nil })
			},
			func(w *core.World) error {
				return w.Provide(func(*core.World) (*Listener, error) { return &Listener{}, nil })
			},
			func(w *core.World) error {
				return w.Provide(func(*core.World) (*Backend, error) { return &Backend{}, nil })
			},
		},
		Systems: []engine.SystemSpec{
			{
				Name:  PrepareSystemName,
				Phase: core.PhaseFrame,
				Slot:  engine.First,
				New: func() core.System {
					return core.SystemFunc(func(ctx *core.Context) error {
						if reconciler != nil {
							return nil
						}
						backend, err := ctx.World.Resource[*Backend]()
						if err != nil {
							return err
						}
						controller := backend.Controller()
						if controller == nil {
							return nil
						}
						mixer, err := ctx.World.Resource[*Mixer]()
						if err != nil {
							return err
						}
						reconciler = NewAudioSystem(controller, mixer)
						return nil
					})
				},
			},
			{
				Name:  SystemName,
				Phase: core.PhaseFrame,
				Slot:  engine.Last,
				New: func() core.System {
					return core.SystemFunc(func(ctx *core.Context) error {
						if reconciler == nil {
							return nil
						}
						return reconciler.Update(ctx)
					})
				},
			},
		},
	}
}
