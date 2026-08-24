package scene

import (
	"fmt"

	"github.com/leonard-atorough/castrum/ecs"
)

// Scene represents a logical grouping of entities within the world.
// A scene owns entities via scene tags and manages its own lifecycle.
// Multiple scenes can be active simultaneously (entities can have multiple scene tags).
type Scene struct {
	ID   string         // Unique scene identifier
	tag  string         // Internal tag used to track entities in this scene
	data map[string]any // Scene-specific data/state
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

// Init is called when the scene becomes active.
// Override this to set up initial scene state.
func (s *Scene) Init(world ecs.World) error {
	return nil
}

// Update is called every frame for the active scene.
// Override this to implement scene-specific logic.
func (s *Scene) Update(world ecs.World, deltaTime float64) error {
	return nil
}

// Shutdown is called when the scene is unloaded.
// This cleans up all entities belonging to this scene.
func (s *Scene) Shutdown(world ecs.World) error {
	// Get all entities in this scene
	entities := s.Entities(world)

	for _, entityID := range entities {
		if err := s.RemoveFromScene(entityID, world); err != nil {
			return fmt.Errorf("failed to remove entity %d from scene %s: %w", entityID, s.ID, err)
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
