package blueprint

import (
	"bytes"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Store struct {
	Blueprints map[string]*Blueprint
}

func NewStore() *Store {
	return &Store{
		Blueprints: make(map[string]*Blueprint),
	}
}

func (s *Store) Load(reader io.Reader) (*Blueprint, error) {
	var blueprint Blueprint
	decoder := yaml.NewDecoder(reader)
	err := decoder.Decode(&blueprint)
	if err != nil {
		return nil, err
	}
	s.AddBlueprint(blueprint.Name, &blueprint)
	return &blueprint, nil
}

func (s *Store) LoadFromPath(path string) (*Blueprint, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return s.Load(file)
}

func (s *Store) LoadFromString(data string) (*Blueprint, error) {
	return s.Load(strings.NewReader(data))
}

func (s *Store) LoadFromBytes(data []byte) (*Blueprint, error) {
	return s.Load(bytes.NewReader(data))
}



func (s *Store) AddBlueprint(name string, blueprint *Blueprint) {
	s.Blueprints[name] = blueprint
}

func (s *Store) GetBlueprint(name string) (*Blueprint, bool) {
	blueprint, exists := s.Blueprints[name]
	return blueprint, exists
}
