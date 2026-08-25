package castrum

import (
	"github.com/leonard-atorough/castrum/internal/scene"
)

type Scene = scene.Scene

type SceneBuilder = scene.Builder

type SceneAPI struct {
	manager *scene.Manager
}

func (s *SceneAPI) LoadScene(name string, scene *Scene) error {
	return s.manager.LoadScene(name, scene)
}

func (s *SceneAPI) UnloadScene(name string) error {
	return s.manager.UnloadScene(name)
}

func (s *SceneAPI) CurrentScene() *Scene {
	return s.manager.CurrentScene().(*Scene)
}

func (s *SceneAPI) TransitionTo(name string) error {
	return s.manager.TransitionTo(name)
}

func (s *SceneAPI) NewSceneBuilder(sceneID string) *SceneBuilder {
	return scene.NewBuilder(sceneID)
}

func (s *SceneAPI) NewScene(sceneID string) *Scene {
	return scene.NewScene(sceneID)
}

func (s *SceneAPI) AddEntityToScene(entityID EntityID, scene *Scene) error {
	return scene.AddToScene(entityID, s.manager.World())
}

func (s *SceneAPI) RemoveEntityFromScene(entityID EntityID, scene *Scene) error {
	return scene.RemoveFromScene(entityID, s.manager.World())
}

func (s *SceneAPI) GetEntitiesInScene(scene *Scene) []EntityID {
	return scene.Entities(s.manager.World())
}

func (s *SceneAPI) GetSceneData(scene *Scene, key string) (any, bool) {
	value, exists := scene.GetData(key)
	return value, exists
}

func (s *SceneAPI) SetSceneData(scene *Scene, key string, value any) {
	scene.SetData(key, value)
}
