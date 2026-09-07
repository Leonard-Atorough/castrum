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
	Systems      = core.Manager
	Scene        = scene.Scene
	SceneBuilder = scene.Builder
	SceneTag     = components.SceneTag
	Component    = core.Component
	Input        = input.InputHandler
	Animation    = animation.System
	Collision    = physics.System
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
	Render  *render.Renderer
	Spatial *spatial.SpatialIndexHandler

	Input *Input

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

	// SceneManager is registered as a World resource (not a typed struct field) so core
	// never needs to import the scene package; see internal/core/resource.go.
	core.SetResource(newWorld, scene.NewManager())

	// all core systems are allowed a priority of -1 for now. Better to have a field for core system priorities in the future.
	input := input.New()

	systems := core.NewManager()

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
		Assets:            assets,
		ComponentRegistry: core.GlobalRegistry, // Use the global component registry instance
		Render:            renderer,            // Initialize the renderer
		CameraEntityID:    cameraEntity.ID,
		Spatial:           spatial,
		Input:             input,
		fixedDelta:        1.0 / float64(config.Engine.TicksPerSecond),
		Speed:             config.Engine.TimeScale,
	}, nil
}

// Scenes returns the scene manager registered on this game's world.
func (g *Game) Scenes() *scene.Manager {
	mgr, _ := core.GetResource[*scene.Manager](g.World)
	return mgr
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

// Layout reports the engine's virtual resolution and keeps the camera's
// screen size in sync with it.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.Config.Graphics.VirtualWidth, g.Config.Graphics.VirtualHeight

	// Update camera screen size
	camComp, err := g.World.GetComponent[camera.Camera](g.CameraEntityID)
	if err == nil {
		cam := camComp
		cam.ScreenSize = geom.Vector2I{X: w, Y: h}
		g.World.SetComponent(g.CameraEntityID, cam)
	}

	return w, h
}

// GetCamera returns the primary camera component from the world.
func (g *Game) GetCamera() (camera.Camera, error) {
	cam, err := g.World.GetComponent[camera.Camera](g.CameraEntityID)
	if err != nil {
		return camera.Camera{}, err
	}
	return cam, nil
}

// SetCamera updates the primary camera component in the world.
func (g *Game) SetCamera(cam camera.Camera) error {
	return g.World.SetComponent(g.CameraEntityID, cam)
}

func QueryFor[T Component](w *core.World) []EntityID {
	return core.QueryFor[T](w)
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
