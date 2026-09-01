package texture

import (
	"fmt"
	"io"
)

type Store struct {
	Textures map[string]*Texture
}

func NewStore() *Store {
	return &Store{
		Textures: make(map[string]*Texture),
	}
}

func (s *Store) AddTexture(name string, texture *Texture) {
	s.Textures[name] = texture
}

func (s *Store) GetTexture(name string) (*Texture, error) {
	texture, exists := s.Textures[name]
	if !exists {
		return nil, fmt.Errorf("texture %s not found", name)
	}
	return texture, nil
}

func (s *Store) Load(reader io.Reader) (*Texture, error) {
	// Implement the loading logic here, for example using render.LoadTexture
	// This is a placeholder implementation
	return nil, nil
}

func (s *Store) LoadTexture(name string, reader io.Reader) (*Texture, error) {
	texture, err := s.Load(reader)
	if err != nil {
		return nil, err
	}
	s.AddTexture(name, texture)
	return texture, nil
}
