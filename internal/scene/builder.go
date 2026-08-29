package scene

import (
	"github.com/leonard-atorough/castrum/internal/core"
)

// Builder provides a fluent interface for creating scenes with entities.
// This helps connect the threads: create scene -> add entities -> bind to world.
type Builder struct {
	scene     *Scene
	world     *core.World
	entityIDs []core.EntityID
}

// NewBuilder creates a new scene builder.
func NewBuilder(sceneID string, world *core.World) *Builder {
	return &Builder{
		scene:     NewScene(sceneID),
		world:     world,
		entityIDs: []core.EntityID{},
	}
}

// WithEntity stages an entity to be added to the scene.
func (b *Builder) WithEntity(entityID core.EntityID) *Builder {
	b.entityIDs = append(b.entityIDs, entityID)
	return b
}

// Build creates the scene and registers all staged entities.
// Returns the scene and any error encountered during entity registration.
func (b *Builder) Build() (*Scene, error) {
	for _, entityID := range b.entityIDs {
		if err := b.scene.AddToScene(entityID, b.world); err != nil {
			return nil, err
		}
	}
	return b.scene, nil
}

// WithLoadHook sets a custom load hook for the scene.
// This hook is called when the scene is loaded and can be used to initialize scene state or perform setup tasks.
func (b *Builder) WithLoadHook(hook SceneHook) *Builder {
	b.scene.loadHook = hook
	return b
}

// WithUnloadHook sets a custom unload hook for the scene.
// This hook is called when the scene is unloaded and can be used to clean up resources or perform teardown tasks.
func (b *Builder) WithUnloadHook(hook SceneHook) *Builder {
	b.scene.unloadHook = hook
	return b
}

func (b *Builder) WithHooks(load, unload SceneHook) *Builder {
	b.scene.loadHook = load
	b.scene.unloadHook = unload
	return b
}

func (b *Builder) WithData(key string, value any) *Builder {
	if b.scene.data == nil {
		b.scene.data = make(map[string]any)
	}
	b.scene.data[key] = value
	return b
}

func (b *Builder) WithDataMap(data map[string]any) *Builder {
	if b.scene.data == nil {
		b.scene.data = make(map[string]any)
	}
	for k, v := range data {
		b.scene.data[k] = v
	}
	return b
}

// Scene returns the underlying scene without registering entities.
// Useful if you want to manually add entities later.
func (b *Builder) Scene() *Scene {
	return b.scene
}
