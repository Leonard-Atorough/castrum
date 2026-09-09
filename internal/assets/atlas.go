package assets

import (
	"fmt"
	"image"
	"io/fs"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"go.yaml.in/yaml/v3"
)

type atlasDef struct {
	Path    string            `yaml:"path"`
	Regions map[string][4]int `yaml:"regions"`
}

type SubTexture struct {
	Name          string
	Image         *ebiten.Image
	Width, Height int
}

type TextureAtlas struct {
	Path    string   // .atlas.yaml files
	Texture *Texture // loaded from/using texture store
	Regions map[string]*SubTexture
}

type atlasStore struct {
	fs      fs.FS
	mu      sync.RWMutex
	atlases map[string]*TextureAtlas // keyed by atlas path
}

func newAtlasStore(filesystem fs.FS) *atlasStore {
	return &atlasStore{
		fs:      filesystem,
		atlases: make(map[string]*TextureAtlas),
	}
}

func (s *atlasStore) Load(path string) (*TextureAtlas, error) {
	s.mu.RLock()
	if atlas, ok := s.atlases[path]; ok {
		s.mu.RUnlock()
		return atlas, nil
	}
	s.mu.RUnlock()

	if s.fs == nil {
		return nil, fmt.Errorf("atlas store has no filesystem configured")
	}

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open atlas file: %w", err)
	}
	defer file.Close()

	var def atlasDef
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&def); err != nil {
		return nil, fmt.Errorf("failed to decode atlas file: %w", err)
	}

	atlas := TextureAtlas{
		Path:    def.Path,
		Texture: &Texture{},
		Regions: make(map[string]*SubTexture),
	}

	//eagerly load texture
	texture, generic, err := loadImageFromFS(s.fs, def.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load atlas texture: %w", err)
	}
	atlas.Texture.Path = def.Path
	atlas.Texture.Image = texture
	atlas.Texture.Width = generic.Bounds().Dx()
	atlas.Texture.Height = generic.Bounds().Dy()

	// slice using atlas def (transform to rect using geom.rect)
	for name, r := range def.Regions {
		rect := image.Rect(r[0], r[1], r[2], r[3])
		if !rect.In(texture.Bounds()) {
			return nil, fmt.Errorf("region %s is out of bounds of the texture", name)
		}
		atlas.Regions[name] = &SubTexture{
			Name:   name,
			Image:  atlas.Texture.Image.SubImage(rect).(*ebiten.Image),
			Width:  rect.Dx(),
			Height: rect.Dy(),
		}
	}

	s.mu.Lock()
	s.atlases[path] = &atlas
	s.mu.Unlock()

	return &atlas, nil
}
