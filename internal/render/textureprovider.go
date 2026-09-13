package render

import (
	"context"
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/assets"
)

type TextureProvider struct {
	loader *assets.Loader

	mu        sync.RWMutex
	images    map[assets.ID]*textureResource
	subimages map[assets.ID]*subtextureResource
}

type textureResource struct {
	image  *ebiten.Image
	width  int
	height int
}

type subtextureResource struct {
	parent *textureResource
	image  *ebiten.Image
	width  int
	height int
}

func NewTextureProvider(loader *assets.Loader) *TextureProvider {
	provider := &TextureProvider{
		loader:    loader,
		images:    make(map[assets.ID]*textureResource),
		subimages: make(map[assets.ID]*subtextureResource),
	}
	loader.RegisterInvalidationListener(provider.Invalidate)
	return provider
}

func (p *TextureProvider) Load(
	ctx context.Context,
	id assets.ID,
) (*ebiten.Image, int, int, error) {
	p.mu.RLock()
	cached, ok := p.images[id]
	p.mu.RUnlock()
	if ok {
		return cached.image, cached.width, cached.height, nil
	}

	data, err := p.loader.Load[assets.TextureData](
		ctx,
		string(id),
		assets.WithID(id),
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("load texture %q: %w", id, err)
	}

	image := ebiten.NewImageFromImage(data.Image)
	resource := &textureResource{
		image:  image,
		width:  data.Width,
		height: data.Height,
	}

	p.mu.Lock()
	p.images[id] = resource
	p.mu.Unlock()

	return resource.image, resource.width, resource.height, nil
}

func (p *TextureProvider) Invalidate(id assets.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id == "" {
		p.images = make(map[assets.ID]*textureResource)
		p.subimages = make(map[assets.ID]*subtextureResource)
		return
	}
	delete(p.images, id)
	delete(p.subimages, id)
}

func (p *TextureProvider) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.images = make(map[assets.ID]*textureResource)
	p.subimages = make(map[assets.ID]*subtextureResource)
}
