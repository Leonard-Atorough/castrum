package assets

import (
	"io/fs"
	"os"

	"github.com/leonard-atorough/castrum/internal/blueprint"
	"github.com/leonard-atorough/castrum/internal/texture"
)

type fileExtension string

var (
	FileExtensionTexture   []fileExtension = []fileExtension{".png", ".jpg", ".jpeg"}
	FileExtensionBlueprint fileExtension   = ".yaml"
)

type Assets struct {
	Textures   *texture.Store
	Blueprints *blueprint.Store
}

func NewAssets(filesystem fs.FS) *Assets {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}

	return &Assets{
		Textures:   texture.NewStore(filesystem),
		Blueprints: blueprint.NewStore(filesystem),
	}
}

func (a *Assets) Load(path string) (res any, err error) {
	switch {
	case hasTextureExtension(path):
		res, err = a.Textures.Load(path)
	case hasBlueprintExtension(path):
		res, err = a.Blueprints.Load(path)
	default:
		return nil, nil
	}
	return res, err
}

func hasTextureExtension(path string) bool {
	for _, ext := range FileExtensionTexture {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == string(ext) {
			return true
		}
	}
	return false
}

func hasBlueprintExtension(path string) bool {
	ext := FileExtensionBlueprint
	return len(path) >= len(ext) && path[len(path)-len(ext):] == string(ext)
}
