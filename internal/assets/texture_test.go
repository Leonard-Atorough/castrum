//go:build !linux
// +build !linux

package assets

import (
	"testing"
	"testing/fstest"
)

func TestTextureStore_Load(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		fs := fstest.MapFS{}
		s := newTextureStore(fs)
		if _, err := s.Load("missing.png"); err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid image data returns error", func(t *testing.T) {
		fs := fstest.MapFS{
			"bad.png": {Data: []byte("not an image")},
		}
		s := newTextureStore(fs)
		if _, err := s.Load("bad.png"); err == nil {
			t.Fatal("expected error for invalid image data")
		}
	})
}

func TestTextureStore_NoFilesystem(t *testing.T) {
	s := &textureStore{fs: nil, Textures: make(map[string]*Texture)}
	if _, err := s.Load("test.png"); err == nil {
		t.Fatal("expected error when filesystem is nil")
	}
}

func TestNewTextureStore(t *testing.T) {
	fs := fstest.MapFS{}
	s := newTextureStore(fs)
	if s.Textures == nil {
		t.Fatal("expected Textures map to be initialized")
	}
}
