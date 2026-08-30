package castrum

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/internal/blueprint"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/scene"
	"github.com/leonard-atorough/castrum/internal/system"
	"github.com/leonard-atorough/castrum/internal/timers"
)

type (
	Entity       = core.Entity
	EntityID     = core.EntityID
	Timer        = timers.Timer
	TimerID      = timers.TimerID
	System       = system.System
	Layer        = system.Layer
	Scene        = scene.Scene
	SceneBuilder = scene.Builder
	Component    = core.Component
)

type Game struct {
	World  *core.World
	Config *Config

	Systems *system.Manager
	Timers  *timers.Manager
	Scenes  *scene.Manager

	BlueprintLoader *blueprint.Loader

	// Timestep state
	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int
}

func NewGame(config *Config) *Game {
	newWorld := core.NewWorld()

	scenes := scene.NewManager(newWorld)
	systems := system.NewManager()
	timers := timers.NewManager()

	return &Game{
		World:           newWorld,
		Config:          config,
		Systems:         systems,
		Timers:          timers,
		Scenes:          scenes,
		BlueprintLoader: blueprint.NewLoader(newWorld.Registry()),
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
