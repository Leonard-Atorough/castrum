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
	if err := g.world.InsertResource[render.Window](render.NewStaticWindow(render.Viewport{
		LogicalWidth: 1280, LogicalHeight: 720,
		WindowWidth: 1280, WindowHeight: 720,
	})); err != nil {
		return fmt.Errorf("castrum: set static window: %w", err)
	}

	sim := gametime.NewSimulation(options.fixedDT)
	if err := g.world.InsertResource(sim); err != nil {
		return fmt.Errorf("castrum: set simulation time: %w", err)
	}
	g.sim = sim

	// The runner overwrites the snapshot resource before each Advance. The
	// zero value reads as no input until the first poll.
	if err := g.world.InsertResource(&input.Snapshot{}); err != nil {
		return fmt.Errorf("castrum: insert input snapshot: %w", err)
	}

	real := gametime.NewReal(options.MaxFrameTime)
	if err := g.world.InsertResource(real); err != nil {
		return fmt.Errorf("castrum: insert real time: %w", err)
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
	if err := g.world.InsertResource(loader); err != nil {
		return fmt.Errorf("castrum: insert asset loader: %w", err)
	}
	if err := g.world.InsertResource(codecs); err != nil {
		return fmt.Errorf("castrum: insert asset codecs: %w", err)
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
	if err := g.world.InsertResource(reg); err != nil {
		return fmt.Errorf("castrum: insert blueprint registry: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFixed, core.PrevTransformSystemName, core.NewPrevTransformCapture()); err != nil {
		return err
	}

	scheduler := gametime.NewScheduler(g.world)
	if err := g.world.InsertResource(scheduler); err != nil {
		return fmt.Errorf("castrum: insert scheduler: %w", err)
	}
	if err := g.addEngineFirst(core.PhasePreFixed, gametime.SchedulerSystemName, scheduler.System()); err != nil {
		return err
	}

	if err := g.world.InsertResource(gametime.NewGroups()); err != nil {
		return fmt.Errorf("castrum: insert timer groups: %w", err)
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
	if err := g.world.InsertResource(hub); err != nil {
		return fmt.Errorf("castrum: insert signal hub: %w", err)
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
	if err := g.world.InsertResource(panel); err != nil {
		return fmt.Errorf("castrum: insert debug panel: %w", err)
	}

	mixer := audio.NewMixer()
	if err := g.world.InsertResource(mixer); err != nil {
		return fmt.Errorf("castrum: insert audio mixer: %w", err)
	}
	if err := g.world.InsertResource(&audio.Listener{}); err != nil {
		return fmt.Errorf("castrum: insert audio listener: %w", err)
	}
	backend := &audio.Backend{}
	if err := g.world.InsertResource(backend); err != nil {
		return fmt.Errorf("castrum: insert audio backend: %w", err)
	}
	reconciler := audio.NewAudioSystem(nil, mixer)
	attached := false
	// Capture before other frame systems so mid-phase attachments wait for the next pass.
	if err := g.addEngineFirst(core.PhaseFrame, "engine.audio-prepare", func(*core.Context) error {
		if controller := backend.Controller(); !attached && controller != nil {
			reconciler = audio.NewAudioSystem(controller, mixer)
			attached = true
		}
		return nil
	}); err != nil {
		return err
	}
	if err := g.schedule(core.PhaseFrame).AddLast(runtime.Entry{
		Name: audio.SystemName,
		System: func(ctx *core.Context) error {
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

	if err := g.world.InsertResource[input.Actions](am); err != nil {
		return fmt.Errorf("castrum: insert action map: %w", err)
	}
	if err := g.world.InsertResource[input.Edges](am); err != nil {
		return fmt.Errorf("castrum: insert input edges: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFrame, input.FrameSystemName, input.NewFrameSystem(am)); err != nil {
		return err
	}
	return g.addEngineFirst(core.PhasePreFixed, input.TickSystemName, input.NewTickSystem(am))
}
