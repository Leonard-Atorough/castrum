package engine

import (
	"github.com/leonard-atorough/castrum/internal/ecs"
	"github.com/leonard-atorough/castrum/internal/system"
)

// Alias for ecs.EntityID to simplify usage in the engine package.
type EntityID = ecs.EntityID

// Alias for system.Layer to simplify usage in the engine package.
type Layer = system.Layer

// Alias for system.System to simplify usage in the engine package.
type System = system.System

type Game struct {
	world   *ecs.World
	manager *system.Manager

	accumulator float64
	fixedDelta  float64
}

func NewGame(fixedDelta float64) *Game {
	return &Game{
		accumulator: 0,
		fixedDelta:  fixedDelta,
		world:       ecs.NewWorld(),
		manager:     system.NewSystemManager(),
	}
}

func (g *Game) World() *ecs.World {
	if g.world == nil {
		g.world = ecs.NewWorld()
	}
	return g.world
}

func (g *Game) Manager() *system.Manager {
	if g.manager == nil {
		g.manager = system.NewSystemManager()
	}
	return g.manager
}

func (g *Game) RegisterSystem(name string, sys System) error {
	return g.Manager().Register(system.Player, name, sys, g.World())
}

func (g *Game) UnregisterSystem(name string) error {
	return g.Manager().Unregister(name, g.World())
}

func (g *Game) Update() error {
	g.accumulator += g.fixedDelta

	for g.accumulator >= g.fixedDelta {
		g.accumulator -= g.fixedDelta
		g.Manager().Update(g.World(), g.fixedDelta)
	}

	return nil
}

func (*Game) Draw() error {
	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// registerCoreSystem registers a system in the Core layer of the game.
// This is typically used for essential systems that should always be active, such as rendering, physics, or input handling.
func (g *Game) registerCoreSystem(name string, sys System) error {
	return g.Manager().Register(system.Core, name, sys, g.World())
}
