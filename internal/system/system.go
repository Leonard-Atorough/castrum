package system

import "github.com/leonard-atorough/castrum/internal/core"

//NOTE: Not sure about having to pass world into all these methods. It might be better to have the system hold a reference to the world,
// but that could lead to issues with systems being used across different worlds. For now, we'll keep it simple and pass the world as a parameter.
// Since its a reference type, it should be fine to pass it around without performance issues. If we find that we need to optimize this later, we can revisit the design.

type System interface {
	Init(world *core.World) error
	Update(world *core.World, delta float64) error
	Shutdown(world *core.World) error
}
