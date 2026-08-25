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

type gameRuntime struct {
	world   *ecs.World
	systems *system.Manager
	timers  *timers.Manager
	scenes  *scene.Manager

	accumulator float64
	fixedDelta  float64
	lastTime    time.Time
	fpsTarget   int
	fpsCounter  float64
}

func newGameRuntime(world *ecs.World, config *config.Config) *gameRuntime {
	// TODO: Use the config parameter to configure the game runtime as needed.
	return &gameRuntime{
		world:   world,
		systems: system.NewManager(),
		timers:  timers.NewManager(),
		scenes:  scene.NewManager(*world),
	}
}

// Update implements the fixed-timestep game loop.
func (g *gameRuntime) Update() error {
	g.accumulator += g.fixedDelta

	for g.accumulator >= g.fixedDelta {
		g.tickSystems(g.fixedDelta)
		g.tickTimers(g.fixedDelta)
		g.accumulator -= g.fixedDelta
	}
	return nil
}

// Draw orchestrates rendering through the scene manager.
func (g *gameRuntime) Draw(screen *ebiten.Image) {
	// Implementation goes here.
	// TODO: Implement renderer package. This package will handle rendering and layout logic, including scaling and aspect ratio management.
}

// Layout delegates to renderer.
func (g *gameRuntime) Layout(w, h int) (int, int) {
	// Implementation goes here.
	// TODO: Implement renderer package. This package will handle rendering and layout logic, including scaling and aspect ratio management.
	return w, h
}

// tickSystems runs all registered systems in order.
func (g *gameRuntime) tickSystems(dt float64) {
	g.systems.Update(*g.world, dt)
}

// tickTimers advances the timer queue.
func (g *gameRuntime) tickTimers(dt float64) {
	g.timers.Update(dt)
}

// measureFps calculates actual frames per second.
func (g *gameRuntime) measureFps() {
	// Implementation goes here.
}
