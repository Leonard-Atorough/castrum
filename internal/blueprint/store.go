package blueprint

import (
	"io/fs"
	"os"

	"go.yaml.in/yaml/v3"
)

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
