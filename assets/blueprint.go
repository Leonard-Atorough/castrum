package assets

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/leonard-atorough/castrum/ecs"
	"go.yaml.in/yaml/v3"
)

// Blueprint represents a reusable template for creating entities with predefined components.
type Blueprint struct {
	Name       string          `yaml:"name"`
	Components []ComponentData `yaml:"components"`
	Version    string          `yaml:"version"`
}

// ComponentData represents the data required to instantiate a component for an entity.
type ComponentData struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

// blueprintStore manages the caching and loading of blueprints from the filesystem.
type blueprintStore struct {
	fs         fs.FS
	mu         sync.RWMutex
	Blueprints map[string]*Blueprint
}

// newBlueprintStore creates a new blueprint store with the given filesystem.
// If the provided filesystem is nil, it defaults to the current directory.
func newBlueprintStore(filesystem fs.FS) *blueprintStore {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &blueprintStore{
		fs:         filesystem,
		Blueprints: make(map[string]*Blueprint),
	}
}

// Load retrieves a blueprint from the store by its path.
// It first checks the cache and then loads from the filesystem if not cached.
func (s *blueprintStore) Load(path string) (*Blueprint, error) {
	s.mu.RLock()
	if bp, ok := s.Blueprints[path]; ok {
		s.mu.RUnlock()
		return bp, nil
	}
	s.mu.RUnlock()

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, &BlueprintError{Message: "failed to open blueprint file", Err: err}
	}
	defer file.Close()

	var blueprint Blueprint
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&blueprint); err != nil {
		return nil, &BlueprintError{Message: "failed to decode blueprint file", Err: err}
	}

	s.mu.Lock()
	s.Blueprints[path] = &blueprint
	s.mu.Unlock()

	return &blueprint, nil
}

// CreateFromBlueprint creates a new entity from a blueprint's component data.
func CreateFromBlueprint(world *ecs.World, bp *Blueprint) (*ecs.Entity, error) {
	components := make([]ecs.Component, len(bp.Components))
	for i, comp := range bp.Components {
		instance, err := ecs.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		components[i] = instance
	}

	return world.CreateWithComponents(bp.Name, components...)
}

var (
	// ErrComponentTypeNotRegistered is returned when a component type is not registered in the registry.
	ErrComponentTypeNotRegistered = &BlueprintError{Message: "component type is not registered"}
	// ErrComponentTypeAlreadyRegistered is returned when a component type is already registered in the registry.
	ErrComponentTypeAlreadyRegistered = &BlueprintError{Message: "component type is already registered"}
	// ErrBlueprintNotFound is returned when a blueprint is not found in the manager.
	ErrBlueprintNotFound = &BlueprintError{Message: "blueprint not found"}
)

type BlueprintError struct {
	Message string
	Err     error
}

func (e *BlueprintError) Error() string {
	return fmt.Sprintf("blueprint error: %s: %v", e.Message, e.Err)
}

func (e *BlueprintError) Unwrap() error {
	return e.Err
}
