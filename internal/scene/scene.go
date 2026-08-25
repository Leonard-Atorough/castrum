package scene

import (
	"fmt"

	"github.com/leonard-atorough/castrum/ecs"
)

type SceneHook func(world ecs.World) error

// Scene is a ready-to-use scene with tag-based entity tracking.
// Embed this for default behavior, or implement Scene directly.
// A SceneBuilder is provided to help create scenes adhering to this struct in a fluent manner.
type Scene struct {
	ID         string         // Unique scene identifier
	tag        string         // Internal tag used to track entities in this scene
	data       map[string]any // Scene-specific data/state
	loadHook   SceneHook      // Optional hook called when the scene is loaded
	unloadHook SceneHook      // Optional hook called when the scene is unloaded
}

func NewScene(id string) *Scene {
	return &Scene{
		ID:   id,
		tag:  fmt.Sprintf("scene:%s", id), // Use scene-prefixed tag for internal tracking
		data: make(map[string]any),
	}
}

func (s *Scene) Name() string {
	return s.ID
}

// AddToScene adds an entity to this scene by tagging it.
// The entity must already exist in the world.
func (s *Scene) AddToScene(entityID ecs.EntityID, world ecs.World) error {
	return world.AddTag(entityID, s.tag)
}

// RemoveFromScene removes an entity from this scene.
func (s *Scene) RemoveFromScene(entityID ecs.EntityID, world ecs.World) error {
	return world.RemoveTag(entityID, s.tag)
}

// Entities returns all entities currently in this scene.
func (s *Scene) Entities(world ecs.World) []ecs.EntityID {
	return world.QueryByTag(s.tag)
}

func (s *Scene) SetLoadHook(hook SceneHook) {
	s.loadHook = hook
}

func (s *Scene) SetUnloadHook(hook SceneHook) {
	s.unloadHook = hook
}

// OnLoad is called when the scene becomes active.
// Override this to set up initial scene state.
func (s *Scene) OnLoad(world ecs.World) error {
	if s.loadHook != nil {
		if err := s.loadHook(world); err != nil {
			return fmt.Errorf("failed to execute load hook for scene %s: %w", s.ID, err)
		}
	}
	return nil
}

// OnUnload is called when the scene is unloaded.
// This cleans up all entities belonging to this scene.
func (s *Scene) OnUnload(world ecs.World) error {
	// Get all entities in this scene
	entities := s.Entities(world)

	for _, entityID := range entities {
		if err := s.RemoveFromScene(entityID, world); err != nil {
			return fmt.Errorf("failed to remove entity %d from scene %s: %w", entityID, s.ID, err)
		}
	}

	if s.unloadHook != nil {
		if err := s.unloadHook(world); err != nil {
			return fmt.Errorf("failed to execute unload hook for scene %s: %w", s.ID, err)
		}
	}

	return nil
}

// SetData stores scene-specific data (e.g., level config, state variables)
func (s *Scene) SetData(key string, value any) {
	s.data[key] = value
}

// GetData retrieves scene-specific data
func (s *Scene) GetData(key string) (any, bool) {
	val, ok := s.data[key]
	return val, ok
}
