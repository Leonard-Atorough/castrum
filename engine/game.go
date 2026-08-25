package castrum

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/config"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/engine"
	"github.com/leonard-atorough/castrum/internal/system"
	"github.com/leonard-atorough/castrum/internal/timers"
)

type Layer = system.Layer
type EntityID = ecs.EntityID
type Component = ecs.Component
type System = ecs.System
type TimerID = timers.TimerID

type Game struct {
	world   *core.World
	config  *config.Config
	runtime *engine.GameRuntime

	systems *SystemAPI
	timers  *TimersAPI
	scenes  *SceneAPI
}

func NewGame(config *config.Config) *Game {
	newWorld := core.NewWorld()
	runtime := engine.NewGameRuntime(newWorld, config)

	return &Game{
		world:   newWorld,
		config:  config,
		runtime: runtime,
		systems: &SystemAPI{mgr: runtime.Systems, wrld: newWorld},
		timers:  &TimersAPI{mgr: runtime.Timers},
	}
}

func (g *Game) World() ecs.World {
	if g.world == nil {
		g.world = core.NewWorld()
		g.runtime = engine.NewGameRuntime(g.world, g.config)
	}
	return g.world
}

// Systems returns the system manager API.
func (g *Game) Systems() *SystemAPI {
	return g.systems
}

// Timers returns the timer manager API.
func (g *Game) Timers() *TimersAPI {
	return g.timers
}

func (g *Game) Scenes() *SceneAPI {
	if g.scenes == nil {
		g.scenes = &SceneAPI{manager: g.runtime.Scenes}
	}
	return g.scenes
}

// CORE EBITEN COMPATIBLE FUNCTIONS

func (g *Game) Update() error {
	return g.runtime.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.runtime.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.runtime.Layout(outsideWidth, outsideHeight)
}

// INTERNAL FUNCTIONS

// registerCoreSystem registers a system in the Core layer of the game.
// This is typically used for essential systems that should always be active, such as rendering, physics, or input handling.
// func (g *Game) registerCoreSystem(name string, sys ecs.System) error {
// 	return g.Manager().mgr.Register(system.Core, name, sys, g.World())
// }
