package blueprint

import (
	"io/fs"
	"os"

	"github.com/leonard-atorough/castrum/internal/core"
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

func (b *Blueprint) Spawn(world *core.World) (*core.Entity, error) {
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

type Store struct {
	fs         fs.FS
	Blueprints map[string]*Blueprint
}

func NewStore(filesystem fs.FS) *Store {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &Store{
		fs:         filesystem,
		Blueprints: make(map[string]*Blueprint),
	}
}

func (s *Store) Load(path string) (*Blueprint, error) {
	// Check cache first
	if bp, ok := s.Blueprints[path]; ok {
		return bp, nil
	}

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

	s.Blueprints[path] = &blueprint // cache by path
	return &blueprint, nil
}
