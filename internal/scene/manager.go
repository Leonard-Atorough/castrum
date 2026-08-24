package scene

import (
	"fmt"

	"github.com/leonard-atorough/castrum/ecs"
	castrum "github.com/leonard-atorough/castrum/engine"
)

type Manager struct {
	scenes  map[string]castrum.Scene
	current string
	world   ecs.World
}

func NewManager(world ecs.World) *Manager {
	return &Manager{
		scenes: make(map[string]castrum.Scene),
		world:  world,
	}
}

func (sm *Manager) LoadScene(name string, scene castrum.Scene) error {
	if _, exists := sm.scenes[name]; exists {
		return fmt.Errorf("scene %s already loaded", name)
	}

	sm.scenes[name] = scene
	return nil
}

func (sm *Manager) UnloadScene(name string) error {
	if _, exists := sm.scenes[name]; !exists {
		return fmt.Errorf("scene %s not found", name)
	}

	scene := sm.scenes[name]
	if sm.current == name {
		if err := scene.Shutdown(sm.world); err != nil {
			return fmt.Errorf("failed to shutdown scene %s: %v", name, err)
		}
		sm.current = ""
	}

	delete(sm.scenes, name)
	return nil
}

func (sm *Manager) CurrentScene() castrum.Scene {
	if sm.current == "" {
		return nil
	}
	return sm.scenes[sm.current]
}

func (sm *Manager) TransitionTo(name string) error {
	scene, exists := sm.scenes[name]
	if !exists {
		return fmt.Errorf("scene %s not found", name)
	}

	if sm.current != "" {
		currentScene := sm.scenes[sm.current]
		if err := currentScene.Shutdown(sm.world); err != nil {
			return fmt.Errorf("failed to shutdown current scene %s: %v", sm.current, err)
		}
	}

	sm.current = name
	if err := scene.Init(sm.world); err != nil {
		return fmt.Errorf("failed to initialize scene %s: %v", name, err)
	}
	return nil
}
