package castrum

import (
	"image/color"
	"io/fs"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/input"
	internalinput "github.com/leonard-atorough/castrum/internal/input"
	"github.com/leonard-atorough/castrum/internal/timingscheduler"
	"github.com/leonard-atorough/castrum/physics"
	"github.com/leonard-atorough/castrum/render"
	"github.com/leonard-atorough/castrum/timers"
)

type Game struct {
	world        *ecs.World
	config       *Config
	renderer     *render.Renderer
	input        input.Reader
	inputHandler *internalinput.Handler
	sched        *timingscheduler.Scheduler
	cameraEntity ecs.Entity
	lastTime     time.Time // the engine's ONLY wall-clock read
}

// worldRunner adapts the ECS SystemManager to the timingscheduler.Runnable interface.
// It converts time.Duration to float64 at the boundary, keeping system internals oblivious.
type worldRunner struct {
	world *ecs.World
}

// Run implements timingscheduler.Runnable. It ticks all registered systems in priority order,
// converting the fixed tick duration to float64 seconds once at this boundary.
// Fail-fast: stops at the first system that returns an error.
func (wr *worldRunner) Run(dt time.Duration) {
	_ = wr.world.SystemManager().Update(wr.world, float64(dt.Seconds()))
}

func NewGame(config *Config, filesystem fs.FS) (*Game, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	newWorld := ecs.NewWorld()

	inputHandler := internalinput.New(config.Input.Bindings)

	assets := assets.NewAssetLoader(filesystem)
	renderer := render.New(assets.Textures)

	newWorld.SetResource[input.Reader](inputHandler)
	newWorld.SetResource(animation.NewAnimationClipStore())
	newWorld.SetResource(events.NewEventBus())
	newWorld.SetResource(assets)

	var err error
	if err = newWorld.RegisterSystem("timer", -1, &timers.TimerSystem{Capacity: 60}); err != nil {
		return nil, err
	}

	if err = newWorld.RegisterSystem("animation", -1, animation.NewSystem()); err != nil {
		return nil, err
	}

	// Physics runs after gameplay systems update transforms and before systems
	// that consume collision events.
	if err = newWorld.RegisterSystem("physics", 1, physics.NewSystem(physics.PhysicsConfig{
		CellSize: config.Physics.CellSize,
		Enabled:  config.Physics.Enabled,
	})); err != nil {
		return nil, err
	}

	sched := timingscheduler.New(config.Engine.TimeScale)
	// Wire WorldRunner: the ECS SystemManager is the game logic Runnable for FixedUpdate phase
	sched.AddRunnable(timingscheduler.FixedUpdate, &worldRunner{world: newWorld})

	cameraEntity, err := newWorld.CreateWithComponents(
		"PrimaryCamera",
		components.NewCamera(
			uint32(config.Graphics.VirtualWidth),
			uint32(config.Graphics.VirtualHeight),
			true,
		),
	)
	if err != nil {
		return nil, err
	}

	return &Game{
		world:        newWorld,
		config:       config,
		renderer:     renderer,
		input:        inputHandler,
		inputHandler: inputHandler,
		sched:        sched,
		cameraEntity: *cameraEntity,
		lastTime:     time.Now(),
	}, nil
}

func (g *Game) CameraEntity() ecs.Entity {
	return g.cameraEntity
}

func (g *Game) World() *ecs.World {
	return g.world
}

// Input returns the resolved action reader used by gameplay systems.
func (g *Game) Input() input.Reader {
	return g.input
}

func (g *Game) SetPaused(paused bool) {
	// Implementation for pausing the game
}

func (g *Game) SetTimeScale(timeScale float64) {
	g.sched.SetTimeScale(timeScale)
}

func (g *Game) Update() error {
	now := time.Now()
	realDt := now.Sub(g.lastTime)
	g.lastTime = now

	g.inputHandler.Snapshot() // runs every frame, pause-agnostic
	g.sched.Update(realDt)    // 0..N fixed ticks inside
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.renderer.Clear(screen, color.Black)
	g.renderer.DrawScene(screen, g.world)
	if g.config.Engine.EnableDebug {
		g.renderer.DrawDebugInfo(screen, g.world)
	}
}

type WindowSize geom.Vector2I

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	windowSize := geom.Vector2I{X: outsideWidth, Y: outsideHeight}
	g.world.SetResource(WindowSize(windowSize))

	return g.config.Graphics.VirtualWidth, g.config.Graphics.VirtualHeight
}
