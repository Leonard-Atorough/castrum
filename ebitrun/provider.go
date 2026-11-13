package ebitrun

import (
	"fmt"
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Leonard-Atorough/castrum/asset"
)

// TextureProvider is the runner's GPU half of the asset pipeline: it
// converts decoded textures to ebiten images at most once per texture
// path and materializes subimages at most once per path and rect.
// Sheet files resolve through the same loader as their textures; memory
// references retain sheet metadata directly.
//
// Texture and SubImageRect map one-to-one onto the DrawItem source
// variants the collector produces: an empty Rect routes to
// Texture, a populated one to SubImageRect.
//
// [New] provides the provider as a resource; the engine renderer and
// user DrawFuncs resolve it through the locator.
type TextureProvider struct {
	loader *asset.Loader
	mu     sync.RWMutex
	images map[asset.ID]*ebiten.Image // one GPU image per texture path
	subs   map[subKey]*ebiten.Image   // one subimage per texture path and rect
}

type subKey struct {
	texturePath asset.ID
	rect        image.Rectangle
}

func newTextureProvider(loader *asset.Loader) *TextureProvider {
	return &TextureProvider{
		loader: loader,
		images: make(map[asset.ID]*ebiten.Image),
		subs:   make(map[subKey]*ebiten.Image),
	}
}

// Texture returns the whole GPU image for a texture path, converting it
// from the loader's decoded cache on first use. Errors name the path.
func (p *TextureProvider) Texture(path asset.ID) (*ebiten.Image, error) {
	return p.texture(path)
}

// SubImageRect returns the GPU subimage of texturePath at rect, in
// pixel coordinates. The rect must be non-empty and lie within the
// texture; all rectangles are validated here. Errors name the texture path
// and the rect.
func (p *TextureProvider) SubImageRect(texturePath asset.ID, rect image.Rectangle) (*ebiten.Image, error) {
	texture, err := p.texture(texturePath)
	if err != nil {
		return nil, err
	}
	bounds := texture.Bounds()
	if rect.Empty() {
		return nil, fmt.Errorf("texture %q: rect %v is empty; use Texture for the whole image", texturePath, rect)
	}
	if !rect.In(bounds) {
		return nil, fmt.Errorf("texture %q: rect %v falls outside texture bounds %v", texturePath, rect, bounds)
	}

	key := subKey{texturePath: texturePath, rect: rect}
	p.mu.RLock()
	sub, ok := p.subs[key]
	p.mu.RUnlock()
	if ok {
		return sub, nil
	}

	sub = texture.SubImage(rect).(*ebiten.Image)
	p.mu.Lock()
	if existing, ok := p.subs[key]; ok {
		return existing, nil
	}
	p.subs[key] = sub
	p.mu.Unlock()
	return sub, nil
}

// SubImage resolves a sheet region to its GPU subimage. The sheet
// texture is converted on first use and shared by all its regions;
// errors name the document, missing region, or texture path.
func (p *TextureProvider) SubImage(source asset.SheetRef, regionName string) (*ebiten.Image, error) {
	sheet, err := source.Resolve(p.loader)
	if err != nil {
		return nil, err
	}
	region, err := sheet.Region(regionName)
	if err != nil {
		return nil, err
	}
	sub, err := p.SubImageRect(sheet.Texture, region)
	if err != nil {
		return nil, fmt.Errorf("sheet region %q: %w", regionName, err)
	}
	return sub, nil
}

// texture returns the GPU image for a texture path, converting it from
// the loader's decoded cache on first use.
func (p *TextureProvider) texture(path asset.ID) (*ebiten.Image, error) {
	p.mu.RLock()
	image, ok := p.images[path]
	p.mu.RUnlock()
	if ok {
		return image, nil
	}

	data, err := p.loader.Load[asset.TextureData](string(path))
	if err != nil {
		return nil, err
	}
	image = ebiten.NewImageFromImage(data.Image)

	p.mu.Lock()
	if existing, ok := p.images[path]; ok {
		return existing, nil
	}
	p.images[path] = image
	p.mu.Unlock()
	return image, nil
}
