package scene

import (
	"github.com/leonard-atorough/castrum/ecs"
)

// Builder provides a fluent interface for creating scenes with entities.
// This helps connect the threads: create scene -> add entities -> bind to world.
type Builder struct {
	scene     *Scene
	entityIDs []ecs.EntityID
}

// NewBuilder creates a new scene builder.
func NewBuilder(sceneID string) *Builder {
	return &Builder{
		scene:     NewScene(sceneID),
		entityIDs: []ecs.EntityID{},
	}
}

// WithEntity stages an entity to be added to the scene.
func (b *Builder) WithEntity(entityID ecs.EntityID) *Builder {
	b.entityIDs = append(b.entityIDs, entityID)
	return b
}

// Build creates the scene and registers all staged entities.
// Returns the scene and any error encountered during entity registration.
func (b *Builder) Build(world ecs.World) (*Scene, error) {
	for _, entityID := range b.entityIDs {
		if err := b.scene.AddToScene(entityID, world); err != nil {
			return nil, err
		}
	}
	return b.scene, nil
}

// Scene returns the underlying scene without registering entities.
// Useful if you want to manually add entities later.
func (b *Builder) Scene() *Scene {
	return b.scene
}
