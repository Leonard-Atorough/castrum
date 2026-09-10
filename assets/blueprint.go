package assets

import (
	"fmt"
	"io/fs"
	"os"
	"sync"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
	"go.yaml.in/yaml/v3"
)

type Blueprint struct {
	Name       string          `yaml:"name"`
	Components []ComponentData `yaml:"components"`
	Version    string          `yaml:"version"`
}

type ComponentData struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

type blueprintStore struct {
	fs         fs.FS
	mu         sync.RWMutex
	Blueprints map[string]*Blueprint
}

func newBlueprintStore(filesystem fs.FS) *blueprintStore {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &blueprintStore{
		fs:         filesystem,
		Blueprints: make(map[string]*Blueprint),
	}
}

func (s *blueprintStore) Load(path string) (*Blueprint, error) {
	// Check cache with read lock first
	s.mu.RLock()
	if bp, ok := s.Blueprints[path]; ok {
		s.mu.RUnlock()
		return bp, nil
	}
	s.mu.RUnlock()

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blueprint Blueprint
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&blueprint); err != nil {
		return nil, err
	}

	// Store with write lock
	s.mu.Lock()
	s.Blueprints[path] = &blueprint // cache by path
	s.mu.Unlock()

	return &blueprint, nil
}

var (
	// ErrComponentTypeNotRegistered is returned when a component type is not registered in the registry.
	ErrComponentTypeNotRegistered = &blueprintError{Message: "component type is not registered"}
	// ErrComponentTypeAlreadyRegistered is returned when a component type is already registered in the registry.
	ErrComponentTypeAlreadyRegistered = &blueprintError{Message: "component type is already registered"}
	// ErrBlueprintNotFound is returned when a blueprint is not found in the manager.
	ErrBlueprintNotFound = &blueprintError{Message: "blueprint not found"}
)

type blueprintError struct {
	Message string
	Err     error
}

func (e *blueprintError) Error() string {
	return fmt.Sprintf("blueprint error: %s: %v", e.Message, e.Err)
}

func (e *blueprintError) Unwrap() error {
	return e.Err
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

type testComponent struct {
	Value int
}

func TestCreateFromBlueprint(t *testing.T) {
	ecs.Register[testComponent]()

	t.Run("spawns an entity with resolved components", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "Goblin",
			Components: []ComponentData{
				{Type: "testComponent", Properties: map[string]any{"Value": 5}},
			},
		}

		entity, err := CreateFromBlueprint(world, bp)
		if err != nil {
			t.Fatalf("CreateFromBlueprint failed: %v", err)
		}

		// ecs.Resolve returns the resolved value (not a pointer), matching
		// GetComponent/SetComponent's value semantics used everywhere else.
		comp, err := world.GetComponent[testComponent](entity.ID)
		if err != nil {
			t.Fatalf("GetComponent failed: %v", err)
		}
		if comp.Value != 5 {
			t.Fatalf("component Value = %d, want 5", comp.Value)
		}
	})

	t.Run("an unregistered component type fails the spawn", func(t *testing.T) {
		world := ecs.NewWorld()
		bp := &Blueprint{
			Name: "Broken",
			Components: []ComponentData{
				{Type: "doesNotExist", Properties: nil},
			},
		}

		if _, err := CreateFromBlueprint(world, bp); err == nil {
			t.Fatal("expected an error for an unregistered component type")
		}
	})
}
