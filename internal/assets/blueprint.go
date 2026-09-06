package assets

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/leonard-atorough/castrum/internal/core"
	"go.yaml.in/yaml/v3"
)

type blueprint struct {
	Name       string          `yaml:"name"`
	Components []componentData `yaml:"components"`
	Version    string          `yaml:"version"`
}

type componentData struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

func (b *blueprint) spawn(world *core.World) (*core.Entity, error) {
	components := make([]core.Component, len(b.Components))
	for i, comp := range b.Components {
		instance, err := core.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		components[i] = instance
	}

	entity, err := world.CreateWithComponents(b.Name, components...)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

type blueprintStore struct {
	fs         fs.FS
	Blueprints map[string]*blueprint
}

func newBlueprintStore(filesystem fs.FS) *blueprintStore {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &blueprintStore{
		fs:         filesystem,
		Blueprints: make(map[string]*blueprint),
	}
}

func (s *blueprintStore) Load(path string) (*blueprint, error) {
	// Check cache first
	if bp, ok := s.Blueprints[path]; ok {
		return bp, nil
	}

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blueprint blueprint
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&blueprint); err != nil {
		return nil, err
	}

	s.Blueprints[path] = &blueprint // cache by path
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
