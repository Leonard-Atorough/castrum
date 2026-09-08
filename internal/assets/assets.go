package assets

import (
	"io/fs"
	"os"
)

type fileExtension string

var (
	FileExtensionTexture   []fileExtension = []fileExtension{".png", ".jpg", ".jpeg"}
	FileExtensionBlueprint fileExtension   = ".yaml"
	FileExtensionAnimation fileExtension   = ".anim.yaml"
)

// Assets manages loading and caching of game resources (textures, animations, blueprints).
// All loading is synchronous (blocking). Load assets during initialization, scene setup,
// or dedicated loading screens—not during the active game loop frame.
type Assets struct {
	Textures   *textureStore
	Blueprints *blueprintStore
	Animations *animationStore
}

func NewAssets(filesystem fs.FS) *Assets {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}

	return &Assets{
		Textures:   newTextureStore(filesystem),
		Blueprints: newBlueprintStore(filesystem),
		Animations: newAnimationStore(filesystem),
	}
}

// Load is the gateway for loading any asset type (synchronous, blocking).
// It infers the asset kind from the path's extension, routes to the matching store,
// and returns the cached result. Results are cached after first load, so repeated
// calls for the same path are fast.
//
// For known types, prefer the specific stores directly:
//   - Assets.Textures.Load(path) for image assets
//   - Assets.Blueprints.Load(path) for entity blueprints
//   - Assets.Animations.Load(path) for animation clips
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
