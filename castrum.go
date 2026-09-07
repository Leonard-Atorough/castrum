package castrum

import (
	"image/color"
	"io/fs"
	"math"
	"reflect"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/animation"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/camera"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/input"
	"github.com/leonard-atorough/castrum/internal/physics"
	"github.com/leonard-atorough/castrum/internal/render"
	"github.com/leonard-atorough/castrum/internal/scene"
	"github.com/leonard-atorough/castrum/internal/spatial"
	"github.com/leonard-atorough/castrum/internal/timers"
)

type (
	World        = core.World
	Entity       = core.Entity
	Timer        = timers.Timer
	System       = core.System
	Scene        = scene.Scene
	SceneBuilder = scene.Builder
	SceneTag     = scene.SceneTag
	Component    = core.Component
	Input        = input.InputHandler
	Animation    = animation.Manager
	Collision    = physics.Manager
	Spatial      = spatial.SpatialIndexHandler
	Camera       = camera.Camera
	EntityID     = core.EntityID
	TimerID      = timers.TimerID
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
	World  *World
	Config *Config

	Systems *core.Manager
	Timers  *timers.TimerSystem
	Camera  *camera.System
	Scenes  *scene.Manager
	Render  *render.Renderer
	Spatial *spatial.SpatialIndexHandler

	Input     *Input
	Animation *Animation
	Collision *Collision

	Assets            *assets.Assets
	ComponentRegistry *core.ComponentRegistry

	// Camera entity ID for updating screen size and accessing camera state
	CameraEntityID EntityID

	// Timestep state
	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int

	// pause and speed
	Paused bool
	Speed  float64
}

func NewGame(config *Config, filesystem fs.FS) (*Game, error) {
	ValidateConfig(config)

	newWorld := core.NewWorld()

	scenes := scene.NewManager(newWorld)
	systems := core.NewManager()
	timers := timers.NewTimerSystem(timersToRemove)
	cameraSystem := camera.NewSystem()
	spatial, err := spatial.NewManager(config.World.GridCellSize)
	if err != nil {
		return nil, err
	}
	input := input.New()
	animation := animation.NewManager()
	collisionMgr := physics.NewManager(spatial.Index, physics.DefaultConfig())

	assets := assets.NewAssets(filesystem)
	renderer := render.New(assets.Textures)

	// Create primary camera as an entity
	cameraEntity, err := newWorld.CreateWithComponents(
		"PrimaryCamera",
		camera.Camera{
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
		World:             newWorld,
		Config:            config,
		Systems:           systems,
		Timers:            timers,
		Camera:            cameraSystem,
		Scenes:            scenes,
		Assets:            assets,
		ComponentRegistry: core.GlobalRegistry, // Use the global component registry instance
		Render:            renderer,            // Initialize the renderer
		CameraEntityID:    cameraEntity.ID,
		Spatial:           spatial,
		Input:             input,
		Animation:         animation,
		Collision:         collisionMgr,
		fixedDelta:        1.0 / float64(config.Engine.TicksPerSecond),
		Speed:             config.Engine.TimeScale,
	}, nil
}

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
		g.Timers.Update(g.World, g.fixedDelta)
		g.Camera.Update(g.World, g.fixedDelta)
		if err := g.Spatial.Update(g.World, g.fixedDelta); err != nil {
			return err
		}
		g.Collision.Update(g.World, g.fixedDelta)
		g.Animation.Update(g.World, g.fixedDelta)
		// Systems run last
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

// Layout reports the engine's virtual resolution and keeps the camera's
// screen size in sync with it.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.Config.Graphics.VirtualWidth, g.Config.Graphics.VirtualHeight

	// Update camera screen size
	camComp, err := g.World.GetComponent(g.CameraEntityID, reflect.TypeFor[camera.Camera]())
	if err == nil {
		cam := camComp.(camera.Camera)
		cam.ScreenSize = geom.Vector2I{X: w, Y: h}
		g.World.SetComponent(g.CameraEntityID, reflect.TypeFor[camera.Camera](), cam)
	}

	return w, h
}

// GetCamera returns the primary camera component from the world.
func (g *Game) GetCamera() (camera.Camera, error) {
	camComp, err := g.World.GetComponent(g.CameraEntityID, reflect.TypeFor[camera.Camera]())
	if err != nil {
		return camera.Camera{}, err
	}
	return camComp.(camera.Camera), nil
}

// SetCamera updates the primary camera component in the world.
func (g *Game) SetCamera(cam camera.Camera) error {
	return g.World.SetComponent(g.CameraEntityID, reflect.TypeFor[camera.Camera](), cam)
}

// Generic component accessors — forward to typed.go helpers
func GetComponent[T Component](w *core.World, entityID EntityID) (T, error) {
	return core.GetComponent[T](w, entityID)
}

func SetComponent[T Component](w *core.World, entityID EntityID, comp T) error {
	return core.SetComponent(w, entityID, comp)
}

func HasComponent[T Component](w *core.World, entityID EntityID) bool {
	return core.HasComponent[T](w, entityID)
}

func QueryFor[T Component](w *core.World) []EntityID {
	return core.QueryFor[T](w)
}

func Types(comps ...Component) []reflect.Type {
	return core.Types(comps...)
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
