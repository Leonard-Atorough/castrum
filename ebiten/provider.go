package ebitrun

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Leonard-Atorough/castrum/asset"
)

// TextureProvider is the runner's GPU half of the asset pipeline: it
// converts decoded textures to ebiten images at most once per texture
// path and materializes region subimages at most once per path and
// region. Registration already decoded each atlas texture through the
// server's cache, so the provider's loads are hits.
//
// [New] provides the provider as a resource; the engine renderer and
// user DrawFuncs resolve it through the locator.
type TextureProvider struct {
	server *asset.Server
	mu     sync.RWMutex
	images map[asset.ID]*ebiten.Image // one GPU image per texture path
	subs   map[subKey]*ebiten.Image   // one subimage per texture path + region
}

type subKey struct {
	texturePath asset.ID
	region      string
}

func newTextureProvider(server *asset.Server) *TextureProvider {
	return &TextureProvider{
		server: server,
		images: make(map[asset.ID]*ebiten.Image),
		subs:   make(map[subKey]*ebiten.Image),
	}
}

// SubImage resolves an atlas region to its GPU subimage. The atlas
// texture is converted on first use and shared by all its regions;
// errors name the atlas, region, and texture path.
func (p *TextureProvider) SubImage(atlasID asset.AtlasID, regionName string) (*ebiten.Image, error) {
	atlas, err := p.server.Store().Atlas(atlasID)
	if err != nil {
		return nil, err
	}
	region, err := atlas.Region(regionName)
	if err != nil {
		return nil, err
	}
	texture, err := p.texture(atlas.TexturePath())
	if err != nil {
		return nil, fmt.Errorf("atlas %q: %w", atlasID, err)
	}

	key := subKey{texturePath: atlas.TexturePath(), region: regionName}
	p.mu.RLock()
	sub, ok := p.subs[key]
	p.mu.RUnlock()
	if ok {
		return sub, nil
	}

	sub = texture.SubImage(region.Rect()).(*ebiten.Image)
	p.mu.Lock()
	if existing, ok := p.subs[key]; ok {
		return existing, nil
	}
	p.subs[key] = sub
	p.mu.Unlock()
	return sub, nil
}

// texture returns the GPU image for a texture path, converting it from
// the server's decoded cache on first use.
func (p *TextureProvider) texture(path asset.ID) (*ebiten.Image, error) {
	p.mu.RLock()
	image, ok := p.images[path]
	p.mu.RUnlock()
	if ok {
		return image, nil
	}

	data, err := p.server.Load[asset.TextureData](string(path))
	if err != nil {
		return nil, err
	}
	// Atlas validation guarantees positive dimensions, so
	// NewImageFromImage cannot panic here.
	image = ebiten.NewImageFromImage(data.Image)

	p.mu.Lock()
	if existing, ok := p.images[path]; ok {
		return existing, nil
	}
	p.images[path] = image
	p.mu.Unlock()
	return image, nil
}
