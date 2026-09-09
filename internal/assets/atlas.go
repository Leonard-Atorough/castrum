package assets

import (
	"encoding/json"
	"fmt"
	"image"
	"io/fs"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Frame represents the coordinates and dimensions of a sprite frame.
type Frame struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// FrameData contains metadata for a sprite frame.
type FrameData struct {
	Frame Frame `json:"frame"`
}

// AtlasDef is the JSON structure for an atlas file.
type AtlasDef struct {
	FileName string               `json:"filename"`
	Frames   map[string]FrameData `json:"frames"`
}

type SubTexture struct {
	Name          string
	Image         *ebiten.Image
	Width, Height int
}

type TextureAtlas struct {
	Path    string   // .atlas.json files
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

	var def AtlasDef
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&def); err != nil {
		return nil, fmt.Errorf("failed to decode atlas file: %w", err)
	}

	atlas := TextureAtlas{
		Path:    def.FileName,
		Texture: &Texture{},
		Regions: make(map[string]*SubTexture),
	}

	// Eagerly load texture
	texture, generic, err := loadImageFromFS(s.fs, def.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to load atlas texture: %w", err)
	}
	atlas.Texture.Path = def.FileName
	atlas.Texture.Image = texture
	atlas.Texture.Width = generic.Bounds().Dx()
	atlas.Texture.Height = generic.Bounds().Dy()

	// Parse frames and create SubTextures
	for name, frameData := range def.Frames {
		f := frameData.Frame
		rect := image.Rect(f.X, f.Y, f.X+f.W, f.Y+f.H)
		if !rect.In(texture.Bounds()) {
			return nil, fmt.Errorf("region %s is out of bounds of the texture", name)
		}
		atlas.Regions[name] = &SubTexture{
			Name:   name,
			Image:  atlas.Texture.Image.SubImage(rect).(*ebiten.Image),
			Width:  f.W,
			Height: f.H,
		}
	}

	s.mu.Lock()
	s.atlases[path] = &atlas
	s.mu.Unlock()

	return &atlas, nil
}
