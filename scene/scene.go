package scene

import "github.com/leonard-atorough/castrum/ecs"

type Scene interface {
	Name() string
	Init(world ecs.World) error
	Update(world ecs.World, deltaTime float64) error
	Shutdown(world ecs.World) error
}

type SceneManager interface {
	LoadScene(name string, scene Scene) error
	UnloadScene(name string) error

	CurrentScene() Scene
	TransitionTo(name string) error

	Update(deltaTime float64) error
	Shutdown() error
}
