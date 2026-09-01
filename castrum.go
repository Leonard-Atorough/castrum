package castrum

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
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

type Game struct {
	World  *core.World
	Config *Config

	Systems *core.Manager
	Timers  *timers.Manager
	Scenes  *scene.Manager

	Assets            *assets.Assets
	ComponentRegistry *core.ComponentRegistry

	// Timestep state
	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int
}

func NewGame(config *Config) *Game {
	newWorld := core.NewWorld()

	scenes := scene.NewManager(newWorld)
	systems := core.NewManager()
	timers := timers.NewManager()

	return &Game{
		World:             newWorld,
		Config:            config,
		Systems:           systems,
		Timers:            timers,
		Scenes:            scenes,
		Assets:            assets.GlobalAssets, // Use the global assets instance
		ComponentRegistry: core.GlobalRegistry, // Use the global component registry instance
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
	// Render current scene + systems
	if scene := g.Scenes.Current(); scene != nil {
		// delegate to scene/renderer
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	// Your resolution logic
	return w, h
}
