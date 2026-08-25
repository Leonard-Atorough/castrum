package scene

import (
	"fmt"

	"github.com/leonard-atorough/castrum/ecs"
)

type Manager struct {
	scenes  map[string]Scene
	current string
	world   ecs.World
}

func NewManager(world ecs.World) *Manager {
	return &Manager{
		scenes: make(map[string]Scene),
		world:  world,
	}
}

func (sm *Manager) LoadScene(name string, scene Scene) error {
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
		if err := scene.OnUnload(sm.world); err != nil {
			return fmt.Errorf("failed to unload scene %s: %v", name, err)
		}
		sm.current = ""
	}

	delete(sm.scenes, name)
	return nil
}

func (sm *Manager) CurrentScene() Scene {
	if sm.current == "" {
		return Scene{}
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
		if err := currentScene.OnUnload(sm.world); err != nil {
			return fmt.Errorf("failed to unload current scene %s: %v", sm.current, err)
		}
	}

	sm.current = name
	if err := scene.OnLoad(sm.world); err != nil {
		return fmt.Errorf("failed to load scene %s: %v", name, err)
	}
	return nil
}
