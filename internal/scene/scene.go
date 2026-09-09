package scene

import (
	"fmt"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

type SceneHook func(world *ecs.World) error

// Scene is a lightweight descriptor for a group of entities and their lifecycle hooks.
// Entity membership is not tracked here; it lives on entities as a components.SceneTag
// component, queryable via world.NewQuery().InScene(id). This keeps membership as a
// single source of truth instead of a second bookkeeping structure that can drift.
type Scene struct {
	ID         string         // Unique scene identifier
	data       map[string]any // Scene-specific data/state
	loadHook   SceneHook      // Optional hook called when the scene is loaded
	unloadHook SceneHook      // Optional hook called when the scene is unloaded
}

func NewScene(id string) *Scene {
	return &Scene{
		ID:   id,
		data: make(map[string]any),
	}
}

func (s *Scene) Name() string {
	return s.ID
}

// AddToScene tags an entity as belonging to this scene.
// The entity must already exist in the world.
func (s *Scene) AddToScene(entityID ecs.EntityID, world *ecs.World) error {
	if !world.HasEntity(entityID) {
		return fmt.Errorf("entity %d does not exist in world", entityID)
	}
	if err := world.AddComponent(entityID, components.SceneTag{SceneID: s.ID}); err != nil {
		return fmt.Errorf("failed to add SceneTag to entity %d: %w", entityID, err)
	}
	return nil
}

// RemoveFromScene removes the SceneTag component so the entity is no longer part of this scene.
func (s *Scene) RemoveFromScene(entityID ecs.EntityID, world *ecs.World) error {
	if !world.HasEntity(entityID) {
		return fmt.Errorf("entity %d does not exist in world", entityID)
	}
	return world.RemoveComponent[components.SceneTag](entityID)
}

// Entities returns all entities currently tagged with this scene's ID.
func (s *Scene) Entities(world *ecs.World) []ecs.EntityID {
	return world.NewQuery().InScene(s.ID).EntityIDs()
}

func (s *Scene) SetLoadHook(hook SceneHook) {
	s.loadHook = hook
}

func (s *Scene) SetUnloadHook(hook SceneHook) {
	s.unloadHook = hook
}

// OnLoad is called when the scene becomes active.
func (s *Scene) OnLoad(world *ecs.World) error {
	if s.loadHook != nil {
		if err := s.loadHook(world); err != nil {
			return fmt.Errorf("failed to execute load hook for scene %s: %w", s.ID, err)
		}
	}
	return nil
}

// OnUnload is called when the scene is deactivated.
// This untags all entities belonging to this scene.
func (s *Scene) OnUnload(world *ecs.World) error {
	for _, entityID := range s.Entities(world) {
		// Suppress errors for entities that no longer exist.
		_ = s.RemoveFromScene(entityID, world)
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

// Manager tracks loaded scenes and a stack of active ones.
// It does not store a *ecs.World reference; callers pass the world explicitly
// to operations that need it (Push, Pop, TransitionTo, UnloadScene), mirroring
// how Bevy's OnEnter/OnExit systems receive World rather than a state machine owning it.
// This lets Manager be registered as a ecs.Resource on World without an import cycle.
type Manager struct {
	scenes  map[string]*Scene
	stack   []string
	builder *Builder
}

func NewManager() *Manager {
	return &Manager{
		scenes: make(map[string]*Scene),
	}
}

func (sm *Manager) Scenes() map[string]*Scene {
	return sm.scenes
}

// Current returns the scene at the top of the stack, or nil if the stack is empty.
func (sm *Manager) Current() *Scene {
	if len(sm.stack) == 0 {
		return nil
	}
	return sm.scenes[sm.stack[len(sm.stack)-1]]
}

// CurrentScene is an alias for Current, kept for readability at call sites.
func (sm *Manager) CurrentScene() *Scene {
	return sm.Current()
}

// Stack returns the IDs of active scenes, bottom to top.
func (sm *Manager) Stack() []string {
	out := make([]string, len(sm.stack))
	copy(out, sm.stack)
	return out
}

func (sm *Manager) SceneBuilder(sceneID string) *Builder {
	if sm.builder == nil {
		sm.builder = NewBuilder(sceneID)
	}
	return sm.builder
}

func (sm *Manager) LoadScene(name string, scene *Scene) error {
	if _, exists := sm.scenes[name]; exists {
		return fmt.Errorf("scene %s already loaded", name)
	}

	sm.scenes[name] = scene
	return nil
}

// UnloadScene removes a scene from the registry. If it is on the active stack,
// OnUnload runs first and it is removed from the stack.
func (sm *Manager) UnloadScene(world *ecs.World, name string) error {
	if _, exists := sm.scenes[name]; !exists {
		return fmt.Errorf("scene %s not found", name)
	}

	if idx := sm.stackIndex(name); idx != -1 {
		if err := sm.scenes[name].OnUnload(world); err != nil {
			return fmt.Errorf("failed to unload scene %s: %w", name, err)
		}
		sm.stack = append(sm.stack[:idx], sm.stack[idx+1:]...)
	}

	delete(sm.scenes, name)
	return nil
}

// Push loads and activates a scene on top of the stack, leaving scenes beneath it loaded.
// Use this for overlays such as a pause menu on top of gameplay.
func (sm *Manager) Push(world *ecs.World, name string) error {
	scene, exists := sm.scenes[name]
	if !exists {
		return fmt.Errorf("scene %s not found", name)
	}

	sm.stack = append(sm.stack, name)
	if err := scene.OnLoad(world); err != nil {
		sm.stack = sm.stack[:len(sm.stack)-1]
		return fmt.Errorf("failed to load scene %s: %w", name, err)
	}
	return nil
}

// Pop unloads and removes the top scene, resuming the one beneath it.
func (sm *Manager) Pop(world *ecs.World) error {
	if len(sm.stack) == 0 {
		return fmt.Errorf("scene stack is empty")
	}

	top := sm.stack[len(sm.stack)-1]
	if err := sm.scenes[top].OnUnload(world); err != nil {
		return fmt.Errorf("failed to unload scene %s: %w", top, err)
	}
	sm.stack = sm.stack[:len(sm.stack)-1]
	return nil
}

// TransitionTo clears the entire stack (unloading top to bottom) and pushes name as
// the sole active scene.
func (sm *Manager) TransitionTo(world *ecs.World, name string) error {
	if _, exists := sm.scenes[name]; !exists {
		return fmt.Errorf("scene %s not found", name)
	}

	for len(sm.stack) > 0 {
		if err := sm.Pop(world); err != nil {
			return fmt.Errorf("failed to unload current scene during transition: %w", err)
		}
	}

	return sm.Push(world, name)
}

func (sm *Manager) stackIndex(name string) int {
	for i, id := range sm.stack {
		if id == name {
			return i
		}
	}
	return -1
}
