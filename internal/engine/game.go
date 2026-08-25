package engine

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/config"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/scene"
	"github.com/leonard-atorough/castrum/internal/system"
	"github.com/leonard-atorough/castrum/internal/timers"
)

type GameRuntime struct {
	world   ecs.World
	Systems *system.Manager
	Timers  *timers.Manager
	Scenes  *scene.Manager

	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int
	fpsCounter  float64
}

func NewGameRuntime(world ecs.World, config *config.Config) *GameRuntime {
	// TODO: Use the config parameter to configure the game runtime as needed.
	return &GameRuntime{
		world:   world,
		Systems: system.NewManager(),
		Timers:  timers.NewManager(),
		Scenes:  scene.NewManager(world),
	}
}

// Update implements the fixed-timestep game loop.
func (g *GameRuntime) Update() error {
	g.accumulator += g.fixedDelta

	for g.accumulator >= g.fixedDelta {
		g.tickSystems(g.fixedDelta)
		g.tickTimers(g.fixedDelta)
		g.accumulator -= g.fixedDelta
	}
	return nil
}

// Draw orchestrates rendering through the scene manager.
func (g *GameRuntime) Draw(screen *ebiten.Image) {
	// Implementation goes here.
	// TODO: Implement renderer package. This package will handle rendering and layout logic, including scaling and aspect ratio management.
}

// Layout delegates to renderer.
func (g *GameRuntime) Layout(w, h int) (int, int) {
	// Implementation goes here.
	// TODO: Implement renderer package. This package will handle rendering and layout logic, including scaling and aspect ratio management.
	return w, h
}

// tickSystems runs all registered systems in order.
func (g *GameRuntime) tickSystems(dt float64) {
	g.Systems.Update(g.world, dt)
}

// tickTimers advances the timer queue.
func (g *GameRuntime) tickTimers(dt float64) {
	g.Timers.Update(dt)
}

// measureFps calculates actual frames per second.
func (g *GameRuntime) measureFps() {
	// Implementation goes here.
}
