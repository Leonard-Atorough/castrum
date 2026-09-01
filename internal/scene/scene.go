package scene

import (
	"fmt"

	"github.com/leonard-atorough/castrum/internal/core"
)

type SceneHook func(world *core.World) error

// Scene manages a collection of entities and scene-specific state.
// Entities are tracked directly within the scene; use AddToScene/RemoveFromScene to manage membership.
type Scene struct {
	ID         string                 // Unique scene identifier
	entities   map[core.EntityID]bool // Tracks entities in this scene
	data       map[string]any         // Scene-specific data/state
	loadHook   SceneHook              // Optional hook called when the scene is loaded
	unloadHook SceneHook              // Optional hook called when the scene is unloaded
}

func NewScene(id string) *Scene {
	return &Scene{
		ID:       id,
		entities: make(map[core.EntityID]bool),
		data:     make(map[string]any),
	}
}

func (s *Scene) Name() string {
	return s.ID
}

// AddToScene adds an entity to this scene.
// The entity must already exist in the world.
func (s *Scene) AddToScene(entityID core.EntityID, world *core.World) error {
	if !world.HasEntity(entityID) {
		return fmt.Errorf("entity %d does not exist in world", entityID)
	}
	s.entities[entityID] = true
	return nil
}

// RemoveFromScene removes an entity from this scene.
func (s *Scene) RemoveFromScene(entityID core.EntityID, world *core.World) error {
	if !world.HasEntity(entityID) {
		return fmt.Errorf("entity %d does not exist in world", entityID)
	}
	delete(s.entities, entityID)
	return nil
}

// Entities returns all entities currently in this scene.
func (s *Scene) Entities(world *core.World) []core.EntityID {
	if len(s.entities) == 0 {
		return nil
	}
	entities := make([]core.EntityID, 0, len(s.entities))
	for entityID := range s.entities {
		entities = append(entities, entityID)
	}
	return entities
}

func (s *Scene) SetLoadHook(hook SceneHook) {
	s.loadHook = hook
}

func (s *Scene) SetUnloadHook(hook SceneHook) {
	s.unloadHook = hook
}

// OnLoad is called when the scene becomes active.
// Override this to set up initial scene state.
func (s *Scene) OnLoad(world *core.World) error {
	if s.loadHook != nil {
		if err := s.loadHook(world); err != nil {
			return fmt.Errorf("failed to execute load hook for scene %s: %w", s.ID, err)
		}
	}
	return nil
}

// OnUnload is called when the scene is unloaded.
// This cleans up all entities belonging to this scene.
func (s *Scene) OnUnload(world *core.World) error {
	// Get all entities in this scene (make a copy of the list since RemoveFromScene modifies it)
	entitiesList := make([]core.EntityID, 0, len(s.entities))
	for entityID := range s.entities {
		entitiesList = append(entitiesList, entityID)
	}

	for _, entityID := range entitiesList {
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
