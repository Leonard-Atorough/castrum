package castrum

import (
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
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
	manager *system.Manager
	timers  *timers.TimerManager

	accumulator float64
	fixedDelta  float64
}

func NewGame(fixedDelta float64) *Game {
	return &Game{
		accumulator: 0,
		fixedDelta:  fixedDelta,
		world:       core.NewWorld(),
		manager:     system.NewSystemManager(),
		timers:      timers.NewTimerManager(),
	}
}

func (g *Game) World() ecs.World {
	if g.world == nil {
		g.world = core.NewWorld()
	}
	return g.world
}

func (g *Game) Manager() *SystemAPI {
	if g.manager == nil {
		g.manager = system.NewSystemManager()
	}
	return &SystemAPI{mgr: g.manager, wrld: g.World()}
}

func (g *Game) Timers() *TimersAPI {
	if g.timers == nil {
		g.timers = timers.NewTimerManager()
	}
	return &TimersAPI{mgr: g.timers}
}

// CORE EBITEN COMPATIBLE FUNCTIONS

func (g *Game) Update() error {
	g.accumulator += g.fixedDelta

	for g.accumulator >= g.fixedDelta {
		g.accumulator -= g.fixedDelta
		
		g.timers.Update(g.fixedDelta)
		g.Manager().Update(g.fixedDelta)
	}

	return nil
}

func (*Game) Draw() error {
	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// INTERNAL FUNCTIONS

// registerCoreSystem registers a system in the Core layer of the game.
// This is typically used for essential systems that should always be active, such as rendering, physics, or input handling.
func (g *Game) registerCoreSystem(name string, sys ecs.System) error {
	return g.Manager().mgr.Register(system.Core, name, sys, g.World())
}
