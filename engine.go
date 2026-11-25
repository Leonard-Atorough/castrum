package castrum

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/gametime"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/internal/runtime"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/serialization"
	"github.com/Leonard-Atorough/castrum/signal"
)

// wireEngine provides the engine-owned resources and registers the engine
// systems in their schedules.
func (g *Game) wireEngine(options *Options) error {
	// Provide a default static window until a runner replaces it.
	g.world.SetResource[render.Window](render.NewStaticWindow(render.Viewport{
		LogicalWidth: 1280, LogicalHeight: 720,
		WindowWidth: 1280, WindowHeight: 720,
	}))

	sim := gametime.NewSimulation(options.fixedDT)
	if err := g.world.ProvideEager(func(*core.World) (*gametime.Simulation, error) {
		return sim, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide simulation time: %w", err)
	}
	g.sim = sim

	// The runner overwrites the snapshot resource before each Advance. The
	// zero value reads as no input until the first poll.
	if err := g.world.Provide(func(*core.World) (*input.Snapshot, error) {
		return &input.Snapshot{}, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide input snapshot: %w", err)
	}

	real := gametime.NewReal(options.MaxFrameTime)
	if err := g.world.ProvideEager(func(*core.World) (*gametime.Real, error) {
		return real, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide real time: %w", err)
	}
	g.real = real

	codecs := options.AssetCodecs
	if codecs == nil {
		codecs = NewAssetCodecs()
	}
	loader, err := asset.NewLoader(builtin.WithFilesystem(options.Filesystem), asset.LoaderOptions{
		Codecs: codecs,
	})
	if err != nil {
		return fmt.Errorf("castrum: create asset loader: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*asset.Loader, error) {
		return loader, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide asset loader: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*asset.Codecs, error) {
		return codecs, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide asset codecs: %w", err)
	}

	reg := serialization.NewRegistry(g.world)
	if err := gametime.RegisterBlueprints(reg); err != nil {
		return fmt.Errorf("castrum: engine blueprints: %w", err)
	}
	if err := registerEngine(reg); err != nil {
		return fmt.Errorf("castrum: engine components: %w", err)
	}
	if err := registerAssets(reg); err != nil {
		return fmt.Errorf("castrum: engine assets: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*serialization.Registry, error) {
		return reg, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide blueprint registry: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFixed, core.PrevTransformSystemName, core.NewPrevTransformCapture()); err != nil {
		return err
	}

	scheduler := gametime.NewScheduler(g.world)
	if err := g.world.Provide(func(*core.World) (*gametime.Scheduler, error) {
		return scheduler, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide scheduler: %w", err)
	}
	if err := g.addEngineFirst(core.PhasePreFixed, gametime.SchedulerSystemName, scheduler.System()); err != nil {
		return err
	}

	if err := g.world.Provide(func(*core.World) (*gametime.Groups, error) {
		return gametime.NewGroups(), nil
	}); err != nil {
		return fmt.Errorf("castrum: provide timer groups: %w", err)
	}
	if err := g.addEngineFirst(core.PhaseFixed, gametime.TimerSystemName, gametime.NewTimerSystem()); err != nil {
		return err
	}
	if err := g.addEngineFirst(core.PhaseFixed, animation.SystemName, animation.Animate); err != nil {
		return err
	}
	if err := g.world.ProvideEager(func(*core.World) (*collision.Queries, error) {
		return collision.NewQueries()
	}); err != nil {
		return fmt.Errorf("castrum: provide collision queries: %w", err)
	}
	if err := g.addEngineFirst(core.PhaseFixed, collision.SystemName, collision.NewSystem()); err != nil {
		return err
	}

	hub := signal.NewHub()
	if err := g.world.Provide(func(*core.World) (*signal.Hub, error) {
		return hub, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide signal hub: %w", err)
	}
	g.hub = hub
	if err := g.schedule(core.PhasePreFrame).SetOpening(runtime.Entry{
		Name: "engine.signal-begin-frame",
		System: func(*core.Context) error {
			hub.BeginFrame()
			return nil
		},
	}); err != nil {
		return fmt.Errorf("castrum: register signal begin-frame system: %w", err)
	}
	if err := g.schedule(core.PhasePostFrame).SetClosing(runtime.Entry{
		Name: "engine.signal-end-frame",
		System: func(*core.Context) error {
			hub.EndFrame()
			return nil
		},
	}); err != nil {
		return fmt.Errorf("castrum: register signal end-frame system: %w", err)
	}
	if err := g.schedule(core.PhasePostFixed).SetClosing(runtime.Entry{
		Name: "engine.signal-end-tick",
		System: func(*core.Context) error {
			hub.EndTick()
			return nil
		},
	}); err != nil {
		return fmt.Errorf("castrum: register signal end-tick system: %w", err)
	}

	panel := debug.NewPanel()
	if err := g.world.Provide(func(*core.World) (*debug.Panel, error) {
		return panel, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide debug panel: %w", err)
	}

	mixer := audio.NewMixer()
	if err := g.world.Provide(func(*core.World) (*audio.Mixer, error) {
		return mixer, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide audio mixer: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*audio.Listener, error) {
		return &audio.Listener{}, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide audio listener: %w", err)
	}
	backend := &audio.Backend{}
	if err := g.world.Provide(func(*core.World) (*audio.Backend, error) {
		return backend, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide audio backend: %w", err)
	}
	var reconciler core.SystemFunc
	// Capture before other frame systems so mid-phase attachments wait for the next pass.
	if err := g.addEngineFirst(core.PhaseFrame, "engine.audio-prepare", func(*core.Context) error {
		if controller := backend.Controller(); reconciler == nil && controller != nil {
			reconciler = audio.NewAudioSystem(controller, mixer)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := g.schedule(core.PhaseFrame).AddLast(runtime.Entry{
		Name: audio.SystemName,
		System: func(ctx *core.Context) error {
			if reconciler == nil {
				return nil
			}
			return reconciler(ctx)
		},
	}); err != nil {
		return fmt.Errorf("castrum: register audio system: %w", err)
	}
	return nil
}

// wireInput provides the action map and registers the engine's input
// systems first in their schedules, so feature and user systems in those
// phases see fresh actions.
func (g *Game) wireInput(bindings input.Bindings, deadzone float64) error {
	am, err := input.New(bindings, deadzone)
	if err != nil {
		return fmt.Errorf("castrum: input bindings: %w", err)
	}

	if err := g.world.Provide(func(*core.World) (input.Actions, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide action map: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (input.Edges, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide input edges: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFrame, input.FrameSystemName, input.NewFrameSystem(am)); err != nil {
		return err
	}
	return g.addEngineFirst(core.PhasePreFixed, input.TickSystemName, input.NewTickSystem(am))
}
