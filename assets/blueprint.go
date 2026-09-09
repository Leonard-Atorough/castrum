package assets

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

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
