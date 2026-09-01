package castrum

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/render"
	"github.com/leonard-atorough/castrum/internal/scene"
	"github.com/leonard-atorough/castrum/internal/timers"
)

type (
	Entity       = core.Entity
	EntityID     = core.EntityID
	Timer        = timers.Timer
	TimerID      = timers.TimerID
	System       = core.System
	Scene        = scene.Scene
	SceneBuilder = scene.Builder
	Component    = core.Component
)

// re-export utility functions
var (
	GetComponent      = core.GetComponent[Component]
	SetComponent      = core.SetComponent[Component]
	HasComponent      = core.HasComponent[Component]
	QueryForComponent = core.QueryFor[Component]
)

type Game struct {
	World  *core.World
	Config *Config

	Systems *core.Manager
	Timers  *timers.Manager
	Scenes  *scene.Manager
	Render  *render.Renderer
	Camera  *render.Camera

	Assets            *assets.Assets
	ComponentRegistry *core.ComponentRegistry

	// Timestep state
	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int
}

func NewGame(config *Config) *Game {
	ValidateConfig(config)

	newWorld := core.NewWorld()

	scenes := scene.NewManager(newWorld)
	systems := core.NewManager()
	timers := timers.NewManager()

	camera := render.NewCamera()
	camera.SetScreenSize(config.Graphics.VirtualWidth, config.Graphics.VirtualHeight)

	return &Game{
		World:             newWorld,
		Config:            config,
		Systems:           systems,
		Timers:            timers,
		Scenes:            scenes,
		Assets:            assets.GlobalAssets,                     // Use the global assets instance
		ComponentRegistry: core.GlobalRegistry,                     // Use the global component registry instance
		Render:            render.NewRenderer(assets.GlobalAssets), // Initialize the renderer
		Camera:            camera,
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
	g.lastTime = time.Now()
	g.accumulator += delta

	for g.accumulator >= g.fixedDelta {
		g.Systems.Update(g.World, g.fixedDelta)
		g.Timers.Update(g.fixedDelta)
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
