package castrum

import "github.com/leonard-atorough/castrum/ecs"

type Scene interface {
	Name() string
	Init(world ecs.World) error
	Update(world ecs.World, deltaTime float64) error
	Shutdown(world ecs.World) error
}
