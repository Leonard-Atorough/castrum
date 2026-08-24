package system

import "github.com/leonard-atorough/castrum/internal/ecs"

type System interface {
	// Init is called once when the system is registered.
	// Use this to initialize state, spawn entities, or validate preconditions.
	Init(world *ecs.World) error

	// Update is called every frame for this system.
	// deltaTime is the time elapsed since the last frame in seconds.
	Update(world *ecs.World, deltaTime float64) error

	// Shutdown is called when the system is unregistered.
	// Use this to clean up resources and remove temporary entities.
	Shutdown(world *ecs.World) error
}
