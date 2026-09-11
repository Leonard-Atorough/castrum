package assets

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	_ "image/jpeg"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Texture represents an image asset loaded from the filesystem.
type Texture struct {
	Path   string        `yaml:"path"`
	Image  *ebiten.Image `yaml:"-"`
	Height int           `yaml:"height"`
	Width  int           `yaml:"width"`
}

// textureStore manages the caching and loading of textures from the filesystem.
type textureStore struct {
	fs       fs.FS
	mu       sync.RWMutex
	Textures map[string]*Texture
}

// newTextureStore creates a new texture store with the given filesystem.
// If the provided filesystem is nil, it defaults to the current directory.
func newTextureStore(filesystem fs.FS) *textureStore {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &textureStore{
		fs:       filesystem,
		Textures: make(map[string]*Texture),
	}
}

// Load retrieves a texture from the store by its path.
// It first checks the cache and then loads from the filesystem if not cached.
func (s *textureStore) Load(path string) (*Texture, error) {
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

	bounds := generic.Bounds()
	tex := &Texture{
		Path:   path,
		Image:  img,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}

	s.mu.Lock()
	s.Textures[path] = tex
	s.mu.Unlock()

	return tex, nil
}
