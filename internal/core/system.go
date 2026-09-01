package core

// System is a unit of game logic run by a Manager. World is passed explicitly
// rather than held by the system so the same System implementation can be
// reused across multiple worlds and stays trivially inspectable/testable.
type System interface {
	Init(world *World) error
	Update(world *World, delta float64) error
	Shutdown(world *World) error
}
