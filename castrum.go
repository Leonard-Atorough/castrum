package castrum

import (
	"image/color"
	"io/fs"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/animation"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/camera"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/events"
	"github.com/leonard-atorough/castrum/internal/input"
	"github.com/leonard-atorough/castrum/internal/physics"
	"github.com/leonard-atorough/castrum/internal/render"
	"github.com/leonard-atorough/castrum/internal/scene"
	"github.com/leonard-atorough/castrum/internal/spatial"
	"github.com/leonard-atorough/castrum/internal/timers"
)

// Core components (data attached to entities)
type (
	Transform  = components.Transform
	Renderable = components.Renderable
	Collider   = components.Collider
	Animation  = components.Animation
	SceneTag   = components.SceneTag
	Timer      = components.Timer
	TimerID    = components.TimerID
	Camera     = components.Camera
)

type (
	// World is the entity-component system container
	World       = core.World
	Entity      = core.Entity
	Component   = core.Component
	EntityID    = core.EntityID
	Query       = core.Query
	QueryResult = core.ResultEntry
)

type (
	// System is the interface for game logic systems
	System          = core.System
	Systems         = core.Manager
	AnimationSystem = animation.System
	CollisionSystem = physics.CollisionSystem
	SpatialIndex    = spatial.SpatialIndexHandler
	InputHandler    = input.InputHandler
)

type (
	Scene        = scene.Scene
	SceneBuilder = scene.Builder
)

type Input = InputHandler

type (
	// TimerCompletedEvent is emitted when a timer fires
	TimerCompletedEvent = timers.TimerCompletedEvent
	// AnimationEvent is emitted when an animation completes or loops
	AnimationEvent = animation.AnimationEvent
	// CollisionEvent is emitted when a collision occurs between two entities.
	CollisionEvent = physics.CollisionEvent
)

// Sentinel errors returned by engine operations. Use errors.Is() for checking.
var (
	ErrEntityNotFound          = core.ErrEntityNotFound
	ErrInvalidEntity           = core.ErrInvalidEntity
	ErrComponentNotFound       = core.ErrComponentNotFound
	ErrSystemNotFound          = core.ErrSystemNotFound
	ErrSystemAlreadyRegistered = core.ErrSystemAlreadyRegistered
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
	World *World

	// Config is the engine configuration (graphics, audio, input, engine settings).
	// Generally immutable after NewGame.
	Config *Config

	// Systems is the system manager (lifecycle: init, update, shutdown).
	// Use Systems.Register to add custom game logic systems.
	// Note: Core systems (physics, animation, rendering) are auto-registered.
	Systems *core.Manager

	// Render is the rendering backend (advanced use only).
	// Most games should not interact with this directly; rendering is automatic.
	// Exposed for custom rendering (e.g., debug overlays, post-processing).
	Render *render.Renderer

	// Spatial is the spatial index for efficient entity queries by position (advanced use only).
	// Most games should not interact with this directly; it's managed by the collision system.
	Spatial *SpatialIndex

	// Input is the input handler for keyboard, mouse, and gamepad state.
	// Use Input.KeyPressed, MouseHeld, etc. to poll input state.
	Input *InputHandler

	// Assets is the asset manager for loading and caching textures, animations, and blueprints.
	// Use Assets to load game resources.
	Assets *assets.Assets

	// CameraEntityID is the entity ID of the primary camera (internal, do not modify).
	CameraEntityID EntityID

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

	newWorld := core.NewWorld()

	// SceneManager is registered as a World resource (not a typed struct field) so core
	// never needs to import the scene package; see internal/core/resource.go.
	core.SetResource(newWorld, scene.NewManager())
	core.SetResource(newWorld, events.NewEventBus())

	input := input.New()
	
	systems := core.NewManager()
	
	// all core systems are allowed a priority of -1 for now. Better to have a field for core system priorities in the future.
	var err error
	if err = systems.Register("timer", -1, &timers.TimerSystem{Capacity: timersToRemove}, newWorld); err != nil {
		return nil, err
	}
	if err = systems.Register("camera", -1, &camera.System{}, newWorld); err != nil {
		return nil, err
	}
	if err = systems.Register("animation", -1, &animation.System{}, newWorld); err != nil {
		return nil, err
	}
	spatial, err := spatial.NewManager(config.World.GridCellSize)
	if err != nil {
		return nil, err
	}
	if err = systems.Register("collision", -1, physics.NewSystem(spatial.Index, physics.DefaultConfig()), newWorld); err != nil {
		return nil, err
	}

	assets := assets.NewAssets(filesystem)
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
		Render:         renderer,
		CameraEntityID: cameraEntity.ID,
		Spatial:        spatial,
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
		if err := g.Spatial.Update(g.World, g.fixedDelta); err != nil {
			return err
		}
		g.Systems.Update(g.World, g.fixedDelta)
		g.accumulator -= g.fixedDelta
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.Render.Clear(screen, color.Black)
	g.Render.DrawScene(screen, g.World)

	if g.Config.Engine.EnableDebug {
		g.Render.DrawDebugInfo(screen, g.World)
	}
}
func (g *Game) Scenes() *scene.Manager {
	mgr, _ := core.GetResource[*scene.Manager](g.World)
	return mgr
}

// GetResource retrieves a typed resource from the world's resource store.
// Returns a zero value and false if the resource is not registered.
func GetResource[T any](world *World) (T, bool) {
	return core.GetResource[T](world)
}

// PushScene activates a pre-loaded scene on top of the stack (useful for overlays/pause menus).
// The scene must have been loaded via Scenes().LoadScene() first.
func (g *Game) PushScene(id string) error {
	return g.Scenes().Push(g.World, id)
}

// PopScene deactivates and removes the top scene from the stack.
func (g *Game) PopScene() error {
	return g.Scenes().Pop(g.World)
}

// TransitionToScene unloads all active scenes and loads a new one.
// The scene must have been loaded via Scenes().LoadScene() first.
func (g *Game) TransitionToScene(id string) error {
	return g.Scenes().TransitionTo(g.World, id)
}

// UnloadScene removes a scene from the registry.
func (g *Game) UnloadScene(id string) error {
	return g.Scenes().UnloadScene(g.World, id)
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

// FindAll returns all entities that have at least a component of type T.
// Each QueryResult includes component access via result.Get[ComponentType]().
func FindAll[T Component](w *World) []QueryResult {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).All()
}

// FindOne returns the first entity with a component of type T, or false if none found.
func FindOne[T Component](w *World) (QueryResult, bool) {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).First()
}

// CountWith returns the number of entities that have a component of type T.
func CountWith[T Component](w *World) int {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).Count()
}

// RegisterSystem registers a system with the game.
// Priority controls execution order: lower values run first. Use negative values for systems
// that should run before core systems (e.g., shader prep), 0 for most game logic, positive for post-processing.
func (g *Game) RegisterSystem(name string, priority int, system System) error {
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
