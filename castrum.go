package castrum

import (
	"image/color"
	"io/fs"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/input"
	"github.com/leonard-atorough/castrum/physics"
	"github.com/leonard-atorough/castrum/render"
	"github.com/leonard-atorough/castrum/timers"
)

const (
	maxDelta              = 0.25
	maxIterationsPerFrame = 5
	timersToRemove        = 60
)

func unboundedRect() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: math.Inf(-1), Y: math.Inf(-1)},
		Max: geom.Vector2{X: math.Inf(1), Y: math.Inf(1)},
	}
}

type Game struct {
	// World is the ECS container managing all entities and components.
	// Use World to create/destroy entities, add/remove components, and query entities.
	World *ecs.World

	// Config is the engine configuration (graphics, audio, input, engine settings).
	// Generally immutable after NewGame.
	Config *Config

	// Systems is the system manager (lifecycle: init, update, shutdown).
	// Use Systems.Register to add custom game logic systems.
	// Note: ecs systems (physics, animation, rendering) are auto-registered.
	Systems *ecs.Manager

	// renderer is the rendering backend (advanced use only).
	// Most games should not interact with this directly; rendering is automatic.
	// Exposed for custom rendering (e.g., debug overlays, post-processing).
	renderer *render.Renderer

	// Input is the input handler for keyboard, mouse, and gamepad state.
	// Use Input.KeyPressed, MouseHeld, etc. to poll input state.
	Input *input.InputHandler

	// Assets is the asset manager for loading and caching textures, animations, and blueprints.
	// Use Assets to load game resources.
	Assets *assets.AssetLoader

	// CameraEntityID is the entity ID of the primary camera (internal, do not modify).
	CameraEntityID ecs.EntityID

	// Timestep state (internal, do not modify).
	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int

	// Paused controls whether the game loop advances (game logic stops, but rendering continues).
	Paused bool

	// Speed is the time scale multiplier (1.0 = normal speed, 0.5 = half speed, etc.).
	Speed float64
}

func NewGame(config *Config, filesystem fs.FS) (*Game, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	newWorld := ecs.NewWorld()
	input := input.New()
	systems := ecs.NewManager()

	newWorld.SetResource(animation.NewAnimationClipStore())
	newWorld.SetResource(events.NewEventBus())

	result, err := registerCoreSystems(systems, newWorld)
	if err != nil {
		return result, err
	}

	assets := assets.NewAssetLoader(filesystem)
	renderer := render.New(assets.Textures)

	// Create primary camera as an entity
	cameraEntity, err := newWorld.CreateWithComponents(
		"PrimaryCamera",
		components.Camera{
			Zoom:       1.0,
			Primary:    true,
			Bounds:     unboundedRect(),
			ScreenSize: geom.Vector2I{X: config.Graphics.VirtualWidth, Y: config.Graphics.VirtualHeight},
		},
	)
	if err != nil {
		return nil, err
	}

	return &Game{
		World:          newWorld,
		Config:         config,
		Systems:        systems,
		Assets:         assets,
		renderer:       renderer,
		CameraEntityID: cameraEntity.ID,
		Input:          input,
		fixedDelta:     1.0 / float64(config.Engine.TicksPerSecond),
		Speed:          config.Engine.TimeScale,
	}, nil
}


// Scenes returns the scene manager registered on this game's world.

func (g *Game) Update() error {
	if g.lastTime.IsZero() {
		g.lastTime = time.Now()
	}

	// Fixed timestep accumulation
	delta := time.Since(g.lastTime).Seconds()
	g.lastTime = time.Now()

	g.Input.Snapshot()
	
	if g.Paused {
		return nil
	}

	delta *= g.Speed
	delta = math.Min(delta, maxDelta) // clamp delta to a maximum of 0.25 seconds
	g.accumulator += delta

	iterator := 0
	for g.accumulator >= g.fixedDelta && iterator < maxIterationsPerFrame {
		iterator++
		g.Systems.Update(g.World, g.fixedDelta)
		g.accumulator -= g.fixedDelta
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.renderer.Clear(screen, color.Black)
	g.renderer.DrawScene(screen, g.World)

	if g.Config.Engine.EnableDebug {
		g.renderer.DrawDebugInfo(screen, g.World)
	}
}

// Layout reports the engine's virtual resolution and keeps the camera's
// screen size in sync with it.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.Config.Graphics.VirtualWidth, g.Config.Graphics.VirtualHeight

	// Update camera screen size
	camComp, err := g.World.GetComponent[components.Camera](g.CameraEntityID)
	if err == nil {
		cam := camComp
		cam.ScreenSize = geom.Vector2I{X: w, Y: h}
		g.World.SetComponent(g.CameraEntityID, cam)
	}
	
	return w, h
}

// GetCamera returns the primary camera component from the world.
func (g *Game) GetCamera() (components.Camera, error) {
	cam, err := g.World.GetComponent[components.Camera](g.CameraEntityID)
	if err != nil {
		return components.Camera{}, err
	}
	return cam, nil
}

// SetCamera updates the primary camera component in the world.
func (g *Game) SetCamera(cam components.Camera) error {
	return g.World.SetComponent(g.CameraEntityID, cam)
}

// GetCameraViewport returns the visible rectangle in world coordinates (what the camera can see).
// Useful for culling, spawning entities at screen edges, etc.
func (g *Game) GetCameraViewport() (geom.Rect, error) {
	cam, err := g.GetCamera()
	if err != nil {
		return geom.Rect{}, err
	}
	return cam.ViewportBounds(), nil
}

// Spawn creates a new entity in the world from a blueprint's component data.
// Load the blueprint first via g.Assets.Blueprints.Load(path).
func (g *Game) Spawn(bp *assets.Blueprint) (*ecs.Entity, error) {
	return assets.CreateFromBlueprint(g.World, bp)
}

// FindAll returns all entities that have at least a component of type T.
// Each QueryResult includes component access via result.Get[ComponentType]().
func FindAll[T ecs.Component](w *ecs.World) []ecs.ResultEntry {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).All()
}

// FindOne returns the first entity with a component of type T, or false if none found.
func FindOne[T ecs.Component](w *ecs.World) (ecs.ResultEntry, bool) {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).First()
}

// CountWith returns the number of entities that have a component of type T.
func CountWith[T ecs.Component](w *ecs.World) int {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).Count()
}

// RegisterSystem registers a system with the game.
// Priority controls execution order: lower values run first. Use negative values for systems
// that should run before ecs systems (e.g., shader prep), 0 for most game logic, positive for post-processing.
func (g *Game) RegisterSystem(name string, priority int, system ecs.System) error {
	return g.Systems.Register(name, priority, system, g.World)
}

func (g *Game) SetPaused(paused bool) {
	g.Paused = paused
}

func (g *Game) IsPaused() bool {
	return g.Paused
}

func (g *Game) SetTimeScale(scale float64) {
	if scale < 0 {
		scale = 0 // Clamp to 0
	}
	g.Speed = scale
}

func (g *Game) GetTimeScale() float64 {
	return g.Speed
}
func registerCoreSystems(systems *ecs.Manager, newWorld *ecs.World) (*Game, error) {
	var err error
	if err = systems.Register("timer", -1, &timers.TimerSystem{Capacity: timersToRemove}, newWorld); err != nil {
		return nil, err
	}
	if err = systems.Register("camera", -1, &render.CameraSystem{}, newWorld); err != nil {
		return nil, err
	}

	if err = systems.Register("animation", -1, animation.NewSystem(), newWorld); err != nil {
		return nil, err
	}

	if err = systems.Register("physics", -1, physics.NewSystem(physics.DefaultConfig()), newWorld); err != nil {
		return nil, err
	}
	return nil, nil
}
