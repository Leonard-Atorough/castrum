package render

import (
	"context"
	"fmt"
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/assets"
	pubatlas "github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/internal/atlas"
)

type Texture struct {
	loader   *assets.Loader
	atlasSvc *atlas.Service

	mu        sync.RWMutex
	images    map[assets.ID]*textureResource
	subimages map[subImageKey]*subimageResource
}

type textureResource struct {
	image  *ebiten.Image
	width  int
	height int
}

type subImageKey struct {
	assetID    assets.ID
	atlasID    pubatlas.ID
	regionName string
}

func newSubImageKey(assetID assets.ID, atlasID pubatlas.ID, regionName string) subImageKey {
	return subImageKey{
		assetID:    assetID,
		atlasID:    atlasID,
		regionName: regionName,
	}
}

type subimageResource struct {
	image  *ebiten.Image
	width  int
	height int
}

func NewTextureProvider(loader *assets.Loader, atlasSvc *atlas.Service) *Texture {
	provider := &Texture{
		loader:    loader,
		atlasSvc:  atlasSvc,
		images:    make(map[assets.ID]*textureResource),
		subimages: make(map[subImageKey]*subimageResource),
	}
	loader.RegisterInvalidationListener(provider.Invalidate)
	return provider
}

func (p *Texture) Load(ctx context.Context, id assets.ID) (*ebiten.Image, int, int, error) {
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

func (p *Texture) SubImage(ctx context.Context, assetID assets.ID, atlasID pubatlas.ID, regionName string) (*ebiten.Image, int, int, error) {
	p.mu.RLock()
	cached, ok := p.subimages[newSubImageKey(assetID, atlasID, regionName)]
	p.mu.RUnlock()
	if ok {
		return cached.image, cached.width, cached.height, nil
	}

	res, err := p.atlasSvc.Get(string(atlasID), string(assetID))
	if err != nil {
		return nil, 0, 0, err
	}
	if res == nil {
		return nil, 0, 0, fmt.Errorf("atlas not found for id=%s, regionName=%s", atlasID, regionName)
	}

	texture, _, _, err := p.Load(ctx, assetID)
	if err != nil {
		return nil, 0, 0, err
	}

	atlasData := res.(pubatlas.Atlas)
	atlasRegion, ok := atlasData.Regions()[regionName]
	if !ok {
		return nil, 0, 0, fmt.Errorf("region %s not found in atlas for id=%s", regionName, atlasID)
	}

	rect := image.Rectangle{
		Min: image.Point{
			X: atlasRegion.X,
			Y: atlasRegion.Y,
		},
		Max: image.Point{
			X: atlasRegion.X + atlasRegion.W,
			Y: atlasRegion.Y + atlasRegion.H,
		},
	}
	subImage := texture.SubImage(rect).(*ebiten.Image)
	resource := &subimageResource{
		image:  subImage,
		width:  rect.Dx(),
		height: rect.Dy(),
	}

	p.mu.Lock()
	p.subimages[newSubImageKey(assetID, atlasID, regionName)] = resource
	p.mu.Unlock()

	return resource.image, resource.width, resource.height, nil
}

func (p *Texture) Invalidate(id assets.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id == "" {
		p.Clear()
		return
	}
	delete(p.images, id)

	// loop through subimages and delete any that reference this texture ID
	for key := range p.subimages {
		if key.assetID == id {
			delete(p.subimages, key)
		}
	}
}

func (p *Texture) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.images = make(map[assets.ID]*textureResource)
	p.subimages = make(map[subImageKey]*subimageResource)
}
