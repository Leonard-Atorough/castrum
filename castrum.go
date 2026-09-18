package castrum

import (
	"context"
	"image/color"
	"io/fs"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/input"
	internalanimation "github.com/leonard-atorough/castrum/internal/animation"
	internalatlas "github.com/leonard-atorough/castrum/internal/atlas"
	internalaudio "github.com/leonard-atorough/castrum/internal/audio"
	internalinput "github.com/leonard-atorough/castrum/internal/input"
	"github.com/leonard-atorough/castrum/internal/render"
	"github.com/leonard-atorough/castrum/internal/timingscheduler"
	"github.com/leonard-atorough/castrum/physics"
	"github.com/leonard-atorough/castrum/timers"
)

type Game struct {
	world        *ecs.World
	assetsSaver  *assets.Saver
	assetsLoader *assets.Loader
	atlasSvc     *internalatlas.Service
	clipStore    *internalanimation.ClipStore
	audioService *internalaudio.AudioService
	eventBus     *events.EventBus
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
	newWorld.SetResource[input.Reader](inputHandler)

	assetSys := assets.NewAssets(filesystem)
	assetsLoader := assetSys.AssetLoader()
	assetsSaver := assetSys.AssetSaver()

	clipStore := internalanimation.NewClipStore()
	newWorld.SetResource(clipStore)

	atlasStore := internalatlas.NewStore()
	atlasService := internalatlas.NewService(atlasStore)
	textureProvider := render.NewTextureProvider(assetsLoader, atlasService)
	renderer := render.New(textureProvider, newWorld, render.RenderConfig{DrawDebugInfo: config.Engine.EnableDebug})

	eventBus := events.NewEventBus()
	newWorld.SetResource(eventBus)

	audioCtx := audio.NewContext(config.Audio.SampleRate)
	audioConfig := internalaudio.Config{
		SampleRate:   config.Audio.SampleRate,
		MasterVolume: config.Audio.MasterVolume,
		GroupVolumes: map[internalaudio.Group]float64{
			internalaudio.GroupMusic: config.Audio.MusicVolume,
			internalaudio.GroupSFX:   config.Audio.SFXVolume,
		},
	}
	audioService := internalaudio.NewService(audioCtx, assetsLoader, audioConfig, context.Background())
	audioSys := internalaudio.NewSystem(audioService)

	var err error
	if err = newWorld.RegisterSystem("timer", -1, &timers.TimerSystem{Capacity: 60}); err != nil {
		return nil, err
	}

	if err = newWorld.RegisterSystem("animation", -1, internalanimation.NewSystem()); err != nil {
		return nil, err
	}
	if err = newWorld.RegisterSystem("audio", -1, audioSys); err != nil {
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
		assetsSaver:  assetsSaver,
		assetsLoader: assetsLoader,
		atlasSvc:     atlasService,
		clipStore:    clipStore,
		eventBus:     eventBus,
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

// AssetsLoader provides methods for loading assets from storage or cache.
func (g *Game) AssetsLoader() *assets.Loader {
	return g.assetsLoader
}

// AssetsSaver provides methods for saving assets to storage or cache.
func (g *Game) AssetsSaver() *assets.Saver {
	return g.assetsSaver
}

// NewAtlas returns a [atlas.Builder] for creating a new texture atlas with
// the specified ID, asset path, and tile dimensions. The atlas is registered
// with the engine when [atlas.Builder.Build] is called.
func (g *Game) NewAtlas(id string, assetPath string, width, height int) *atlas.Builder {
	return g.atlasSvc.NewBuilder(id, assetPath, width, height)
}

// NewClip returns a [animation.ClipBuilder] for defining an animation clip
// with the given ID and atlas. The clip is registered with the engine's
// internal clip store when [animation.ClipBuilder.Build] is called.
func (g *Game) NewClip(id string, atlas *atlas.Atlas) *animation.ClipBuilder {
	return g.clipStore.NewBuilder(id, atlas)
}

// Input returns the resolved action reader used by gameplay systems.
func (g *Game) Input() input.Reader {
	return g.input
}

func (g *Game) EventBus() *events.EventBus {
	return g.eventBus
}

func (g *Game) AudioService() *internalaudio.AudioService {
	return g.audioService
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
	ctx := context.Background()
	g.renderer.Clear(screen, color.Black)
	g.renderer.DrawScene(ctx, screen)
}

type WindowSize geom.Vector2I

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	windowSize := geom.Vector2I{X: outsideWidth, Y: outsideHeight}
	g.world.SetResource(WindowSize(windowSize))

	return g.config.Graphics.VirtualWidth, g.config.Graphics.VirtualHeight
}
