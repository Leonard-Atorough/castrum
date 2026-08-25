package castrum

import (
	"github.com/leonard-atorough/castrum/internal/scene"
)

type Scene = scene.Scene

type SceneAPI struct {
	manager *scene.Manager
}

func (s *SceneAPI) LoadScene(name string, scene Scene) error {
	return s.manager.LoadScene(name, scene)
}

func (s *SceneAPI) UnloadScene(name string) error {
	return s.manager.UnloadScene(name)
}

func (s *SceneAPI) CurrentScene() Scene {
	return s.manager.CurrentScene()
}

func (s *SceneAPI) TransitionTo(name string) error {
	return s.manager.TransitionTo(name)
}
