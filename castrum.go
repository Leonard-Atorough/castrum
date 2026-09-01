package castrum

import (
	"image/color"
	"io/fs"
	"math"
	"reflect"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/internal/animation"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/input"
	"github.com/leonard-atorough/castrum/internal/render"
	"github.com/leonard-atorough/castrum/internal/scene"
	"github.com/leonard-atorough/castrum/internal/timers"
)

type (
	World        = core.World
	Entity       = core.Entity
	EntityID     = core.EntityID
	Timer        = timers.Timer
	TimerID      = timers.TimerID
	System       = core.System
	Scene        = scene.Scene
	SceneBuilder = scene.Builder
	SceneTag     = scene.SceneTag
	Component    = core.Component
	Input        = input.Manager
	Animation    = animation.Manager
	Camera       = render.Camera
)

const MaxDelta = 0.25

type Game struct {
	World  *World
	Config *Config

	Systems *core.Manager
	Timers  *timers.Manager
	Scenes  *scene.Manager
	Render  *render.Renderer
	Camera  *render.Camera

	Input     *Input
	Animation *Animation

	Assets            *assets.Assets
	ComponentRegistry *core.ComponentRegistry

	// Timestep state
	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int
}

func NewGame(config *Config, filesystem fs.FS) *Game {
	ValidateConfig(config)

	newWorld := core.NewWorld()

	scenes := scene.NewManager(newWorld)
	systems := core.NewManager()
	timers := timers.NewManager()

	camera := render.NewCamera()
	camera.SetScreenSize(config.Graphics.VirtualWidth, config.Graphics.VirtualHeight)

	assets := assets.NewAssets(filesystem)

	return &Game{
		World:             newWorld,
		Config:            config,
		Systems:           systems,
		Timers:            timers,
		Scenes:            scenes,
		Assets:            assets,              // Use the global assets instance
		ComponentRegistry: core.GlobalRegistry, // Use the global component registry instance
		Render:            render.New(assets),  // Initialize the renderer
		Camera:            camera,
		Input:             input.NewManager(),
		Animation:         animation.NewManager(),
		fixedDelta:        1.0 / float64(config.Engine.TicksPerSecond),
	}
}

// EBITEN INTERFACE IMPLEMENTATION

func (g *Game) Update() error {
	if g.lastTime.IsZero() {
		g.lastTime = time.Now()
	}

	// Fixed timestep accumulation
	delta := time.Since(g.lastTime).Seconds()
	delta = math.Min(delta, MaxDelta) // clamp delta to a maximum of 0.25 seconds

	g.lastTime = time.Now()
	g.accumulator += delta

	for g.accumulator >= g.fixedDelta {
		g.Timers.Update(g.fixedDelta)
		g.Input.Update(g.World, g.fixedDelta)
		g.Animation.Update(g.World, g.fixedDelta)
		// Systems run last
		g.Systems.Update(g.World, g.fixedDelta)
		g.accumulator -= g.fixedDelta
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.Render.Clear(screen, color.Black)
	g.Render.DrawScene(screen, g.Camera, g.World)

	if g.Config.Engine.EnableDebug {
		g.Render.DrawDebugInfo(screen, g.Camera, g.World)
	}
}

// Layout reports the engine's virtual resolution and keeps the camera's
// screen size in sync with it.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.Config.Graphics.VirtualWidth, g.Config.Graphics.VirtualHeight
	g.Camera.SetScreenSize(w, h)
	return w, h
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
