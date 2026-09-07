package assets

import (
	"io/fs"
	"os"
)

type fileExtension string

var (
	FileExtensionTexture    []fileExtension = []fileExtension{".png", ".jpg", ".jpeg"}
	FileExtensionBlueprint  fileExtension   = ".yaml"
	FileExtensionAnimation  fileExtension   = ".anim.yaml"
)

type Assets struct {
	Textures    *textureStore
	Blueprints  *blueprintStore
	Animations  *animationStore
}

func NewAssets(filesystem fs.FS) *Assets {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}

	return &Assets{
		Textures:    newTextureStore(filesystem),
		Blueprints:  newBlueprintStore(filesystem),
		Animations:  newAnimationStore(filesystem),
	}
}

// Load is the single gateway for loading any asset type: it infers the
// asset kind from path's extension, routes to the matching store, and
// returns the cached, concretely-typed result as any. Callers that know
// which type they want up front (e.g. render.TextureLoader) should depend
// on the specific store instead of going through this router.
func (a *Assets) Load(path string) (res any, err error) {
	switch {
	case hasTextureExtension(path):
		res, err = a.Textures.Load(path)
	case hasAnimationExtension(path):
		res, err = a.Animations.Load(path)
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

func hasAnimationExtension(path string) bool {
	ext := FileExtensionAnimation
	return len(path) >= len(ext) && path[len(path)-len(ext):] == string(ext)
}

func hasBlueprintExtension(path string) bool {
	ext := FileExtensionBlueprint
	return len(path) >= len(ext) && path[len(path)-len(ext):] == string(ext)
}
