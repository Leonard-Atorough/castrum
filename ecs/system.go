package ecs

type System interface {
	// Init is called once when the system is registered.
	// Use this to initialize state, spawn entities, or validate preconditions.
	Init(world World) error

	// Update is called every frame for this system.
	// deltaTime is the time elapsed since the last frame in seconds.
	Update(world World, deltaTime float64) error

	// Shutdown is called when the system is unregistered.
	// Use this to clean up resources and remove temporary entities.
	Shutdown(world World) error
}
