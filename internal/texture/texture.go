package texture

import (
	"fmt"
	"io/fs"

	_ "image/jpeg"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Texture struct {
	Path          string
	Image         *ebiten.Image
	Height, Width int
}

type Store struct {
	fs       fs.FS
	Textures map[string]*Texture
}

func NewStore(filesystem fs.FS) *Store {
	return &Store{
		fs:       filesystem,
		Textures: make(map[string]*Texture),
	}
}

func (s *Store) Load(path string) (*Texture, error) {
	if tex, ok := s.Textures[path]; ok {
		return tex, nil
	}

	if s.fs == nil {
		return nil, fmt.Errorf("texture store has no filesystem configured")
	}

	// Load the texture from the filesystem
	file, err := s.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, generic, err := ebitenutil.NewImageFromFileSystem(s.fs, path)
	if err != nil {
		return nil, err
	}

	bounds := generic.Bounds() // Get the bounds of the generic image.
	tex := &Texture{
		Path:   path,
		Image:  img,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}
	s.Textures[path] = tex
	return tex, nil
}
