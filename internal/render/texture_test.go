package render

import (
	"context"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/assets"
	pubatlas "github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/internal/atlas"
)

func TestNewTextureProvider(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	if provider == nil {
		t.Fatal("NewTextureProvider returned nil")
	}
	if provider.loader != loader {
		t.Error("NewTextureProvider did not set loader")
	}
	if provider.atlasSvc != atlasSvc {
		t.Error("NewTextureProvider did not set atlasSvc")
	}
	if provider.images == nil {
		t.Error("NewTextureProvider did not initialize images map")
	}
	if provider.subimages == nil {
		t.Error("NewTextureProvider did not initialize subimages map")
	}
}

func TestTextureProviderLoadCaches(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	id := assets.ID("test.png")

	img := ebiten.NewImage(32, 32)
	provider.images[id] = &textureResource{image: img, width: 32, height: 32}

	got, w, h, err := provider.Load(ctx, id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != img {
		t.Error("Load() returned different image than cached")
	}
	if w != 32 || h != 32 {
		t.Errorf("Load() dimensions = (%d, %d), want (32, 32)", w, h)
	}
}

func TestTextureProviderLoadNotFound(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	_, _, _, err := provider.Load(ctx, assets.ID("nonexistent.png"))
	if err == nil {
		t.Error("Load() for nonexistent file error = nil, want error")
	}
}

func TestTextureProviderSubImage(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	textureID := assets.ID("sprite.png")
	atlasID := pubatlas.ID("test_atlas")

	fullImg := ebiten.NewImage(64, 64)
	provider.images[textureID] = &textureResource{image: fullImg, width: 64, height: 64}

	atlasData := pubatlas.NewTextureAtlas("test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})
	atlasSvc.Set("test_atlas", "sprite.png", atlasData)

	img, w, h, err := provider.SubImage(ctx, textureID, atlasID, "head")
	if err != nil {
		t.Fatalf("SubImage() error = %v", err)
	}
	if img == nil {
		t.Fatal("SubImage() returned nil image")
	}
	if w != 32 || h != 32 {
		t.Errorf("SubImage() dimensions = (%d, %d), want (32, 32)", w, h)
	}
}

func TestTextureProviderSubImageCaches(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	textureID := assets.ID("sprite.png")
	atlasID := pubatlas.ID("test_atlas")

	fullImg := ebiten.NewImage(64, 64)
	provider.images[textureID] = &textureResource{image: fullImg, width: 64, height: 64}

	atlasData := pubatlas.NewTextureAtlas("test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})
	atlasSvc.Set("test_atlas", "sprite.png", atlasData)

	img1, _, _, err := provider.SubImage(ctx, textureID, atlasID, "head")
	if err != nil {
		t.Fatalf("first SubImage() error = %v", err)
	}

	img2, _, _, err := provider.SubImage(ctx, textureID, atlasID, "head")
	if err != nil {
		t.Fatalf("second SubImage() error = %v", err)
	}

	if img1 != img2 {
		t.Error("SubImage() returned different instances for cached subimage")
	}
}

func TestTextureProviderSubImageAtlasNotFound(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	_, _, _, err := provider.SubImage(ctx, assets.ID("sprite.png"), pubatlas.ID("nonexistent"), "head")
	if err == nil {
		t.Error("SubImage() for nonexistent atlas error = nil, want error")
	}
}

func TestTextureProviderSubImageRegionNotFound(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	textureID := assets.ID("sprite.png")

	fullImg := ebiten.NewImage(64, 64)
	provider.images[textureID] = &textureResource{image: fullImg, width: 64, height: 64}

	atlasData := pubatlas.NewTextureAtlas("test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})
	atlasSvc.Set("test_atlas", "sprite.png", atlasData)

	_, _, _, err := provider.SubImage(ctx, textureID, pubatlas.ID("test_atlas"), "nonexistent")
	if err == nil {
		t.Error("SubImage() for nonexistent region error = nil, want error")
	}
}

func TestTextureProviderInvalidate(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	id := assets.ID("test.png")

	img := ebiten.NewImage(32, 32)
	provider.images[id] = &textureResource{image: img, width: 32, height: 32}

	provider.mu.RLock()
	_, ok := provider.images[id]
	provider.mu.RUnlock()
	if !ok {
		t.Fatal("texture not cached before invalidation")
	}

	provider.Invalidate(id)

	provider.mu.RLock()
	_, ok = provider.images[id]
	provider.mu.RUnlock()
	if ok {
		t.Error("texture still cached after invalidation")
	}
}

func TestTextureProviderInvalidationListenerRemovesImage(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	id := assets.ID("texture.png")
	provider.images[id] = &textureResource{image: ebiten.NewImage(1, 1), width: 1, height: 1}

	loader.Invalidate(id)

	provider.mu.RLock()
	_, ok := provider.images[id]
	provider.mu.RUnlock()
	if ok {
		t.Fatal("texture provider image survived loader invalidation")
	}
}

func TestTextureProviderInvalidateRemovesSubImages(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	textureID := assets.ID("sprite.png")
	atlasID := pubatlas.ID("test_atlas")

	fullImg := ebiten.NewImage(64, 64)
	provider.images[textureID] = &textureResource{image: fullImg, width: 64, height: 64}

	atlasData := pubatlas.NewTextureAtlas("test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
		"body": {Name: "body", X: 0, Y: 32, W: 32, H: 32},
	})
	atlasSvc.Set("test_atlas", "sprite.png", atlasData)

	_, _, _, _ = provider.SubImage(ctx, textureID, atlasID, "head")
	_, _, _, _ = provider.SubImage(ctx, textureID, atlasID, "body")

	provider.mu.RLock()
	if len(provider.subimages) != 2 {
		t.Fatalf("expected 2 subimages cached, got %d", len(provider.subimages))
	}
	provider.mu.RUnlock()

	provider.Invalidate(textureID)

	provider.mu.RLock()
	for key := range provider.subimages {
		if key.assetID == textureID {
			t.Errorf("subimage for invalidated texture still cached: %+v", key)
		}
	}
	provider.mu.RUnlock()
}

func TestTextureProviderInvalidateAll(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	img1 := ebiten.NewImage(32, 32)
	img2 := ebiten.NewImage(64, 64)
	provider.images[assets.ID("test1.png")] = &textureResource{image: img1, width: 32, height: 32}
	provider.images[assets.ID("test2.png")] = &textureResource{image: img2, width: 64, height: 64}

	provider.mu.RLock()
	if len(provider.images) != 2 {
		t.Fatalf("expected 2 textures cached, got %d", len(provider.images))
	}
	provider.mu.RUnlock()

	provider.InvalidateAll()

	provider.mu.RLock()
	if len(provider.images) != 0 {
		t.Errorf("expected 0 textures after Invalidate(\"\"), got %d", len(provider.images))
	}
	if len(provider.subimages) != 0 {
		t.Errorf("expected 0 subimages after Invalidate(\"\"), got %d", len(provider.subimages))
	}
	provider.mu.RUnlock()
}

func TestTextureProviderClear(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	img := ebiten.NewImage(32, 32)
	provider.images[assets.ID("test.png")] = &textureResource{image: img, width: 32, height: 32}

	provider.Clear()

	provider.mu.RLock()
	if len(provider.images) != 0 {
		t.Error("images not empty after Clear()")
	}
	if len(provider.subimages) != 0 {
		t.Error("subimages not empty after Clear()")
	}
	provider.mu.RUnlock()
}

func TestTextureProviderConcurrentLoad(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	id := assets.ID("test.png")

	img := ebiten.NewImage(32, 32)
	provider.images[id] = &textureResource{image: img, width: 32, height: 32}

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan *ebiten.Image, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loaded, _, _, _ := provider.Load(ctx, id)
			results <- loaded
		}()
	}

	wg.Wait()
	close(results)

	var first *ebiten.Image
	for img := range results {
		if first == nil {
			first = img
		}
		if img != first {
			t.Error("concurrent Load() returned different instances")
		}
	}
}

func TestTextureProviderConcurrentSubImage(t *testing.T) {
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	provider := NewTextureProvider(loader, atlasSvc)

	ctx := context.Background()
	textureID := assets.ID("sprite.png")
	atlasID := pubatlas.ID("test_atlas")

	fullImg := ebiten.NewImage(64, 64)
	provider.images[textureID] = &textureResource{image: fullImg, width: 64, height: 64}

	atlasData := pubatlas.NewTextureAtlas("test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})
	atlasSvc.Set("test_atlas", "sprite.png", atlasData)

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan *ebiten.Image, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			img, _, _, _ := provider.SubImage(ctx, textureID, atlasID, "head")
			results <- img
		}()
	}

	wg.Wait()
	close(results)

	var first *ebiten.Image
	for img := range results {
		if first == nil {
			first = img
		}
		if img != first {
			t.Error("concurrent SubImage() returned different instances")
		}
	}
}

func TestSubImageKey(t *testing.T) {
	key := newSubImageKey(assets.ID("tex.png"), pubatlas.ID("atlas"), "region")

	if key.assetID != assets.ID("tex.png") {
		t.Errorf("assetID = %v, want %v", key.assetID, assets.ID("tex.png"))
	}
	if key.atlasID != pubatlas.ID("atlas") {
		t.Errorf("atlasID = %v, want %v", key.atlasID, pubatlas.ID("atlas"))
	}
	if key.regionName != "region" {
		t.Errorf("regionName = %v, want %v", key.regionName, "region")
	}
}
