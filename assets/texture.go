package assets

import (
	"fmt"
	"io/fs"
	"sync"

	_ "image/jpeg"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Texture struct {
	Path   string        `yaml:"path"`
	Image  *ebiten.Image `yaml:"-"`
	Height int           `yaml:"height"`
	Width  int           `yaml:"width"`
}

type textureStore struct {
	fs       fs.FS
	mu       sync.RWMutex
	Textures map[string]*Texture
}

func newTextureStore(filesystem fs.FS) *textureStore {
	return &textureStore{
		fs:       filesystem,
		Textures: make(map[string]*Texture),
	}
}

func (s *textureStore) Load(path string) (*Texture, error) {
	// Check cache with read lock first
	s.mu.RLock()
	if tex, ok := s.Textures[path]; ok {
		s.mu.RUnlock()
		return tex, nil
	}
	s.mu.RUnlock()

	if s.fs == nil {
		return nil, fmt.Errorf("texture store has no filesystem configured")
	}

	img, generic, err := ebitenutil.NewImageFromFileSystem(s.fs, path)
	if err != nil {
		return nil, err
	}

	bounds := generic.Bounds() // Get the bounds of the generic image. .
	tex := &Texture{
		Path:   path,
		Image:  img,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}

	// Store with write lock
	s.mu.Lock()
	s.Textures[path] = tex
	s.mu.Unlock()

	return tex, nil
}
