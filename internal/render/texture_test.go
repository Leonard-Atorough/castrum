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

// textureFixture provides a Texture provider wired to a real atlas service and
// loader, so tests can exercise caching and invalidation without filesystem
// access.
type textureFixture struct {
	provider *Texture
	atlasSvc *atlas.Service
	loader   *assets.Loader
}

func newTextureFixture(t *testing.T) textureFixture {
	t.Helper()
	loader := assets.NewLoader(nil)
	atlasSvc := atlas.NewService(atlas.NewStore())
	return textureFixture{
		provider: NewTextureProvider(loader, atlasSvc),
		atlasSvc: atlasSvc,
		loader:   loader,
	}
}

// seedAtlas pre-caches a texture image and registers an atlas with the service,
// matching the production Builder.Build() path (stores *Atlas).
func (f textureFixture) seedAtlas(t *testing.T, atlasID, assetID string, w, h int, regions map[string]pubatlas.AtlasRegion) {
	t.Helper()
	f.provider.images[assets.ID(assetID)] = &textureResource{
		image:  ebiten.NewImage(w, h),
		width:  w,
		height: h,
	}
	f.atlasSvc.Set(atlasID, assetID, pubatlas.NewTextureAtlas(atlasID, assetID, w, h, regions))
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func TestNewTextureProvider(t *testing.T) {
	f := newTextureFixture(t)

	if f.provider == nil {
		t.Fatal("NewTextureProvider returned nil")
	}
	if f.provider.loader != f.loader {
		t.Error("loader not wired")
	}
	if f.provider.atlasSvc != f.atlasSvc {
		t.Error("atlasSvc not wired")
	}
	if f.provider.images == nil {
		t.Error("images map not initialized")
	}
	if f.provider.subimages == nil {
		t.Error("subimages map not initialized")
	}
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestTextureProviderLoadCaches(t *testing.T) {
	f := newTextureFixture(t)
	id := assets.ID("test.png")

	img := ebiten.NewImage(32, 32)
	f.provider.images[id] = &textureResource{image: img, width: 32, height: 32}

	got, w, h, err := f.provider.Load(context.Background(), id)
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
	f := newTextureFixture(t)

	_, _, _, err := f.provider.Load(context.Background(), assets.ID("nonexistent.png"))
	if err == nil {
		t.Error("Load() for nonexistent file error = nil, want error")
	}
}

// ---------------------------------------------------------------------------
// SubImage
// ---------------------------------------------------------------------------

func TestTextureProviderSubImage(t *testing.T) {
	f := newTextureFixture(t)
	f.seedAtlas(t, "test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})

	img, w, h, err := f.provider.SubImage(context.Background(), assets.ID("sprite.png"), pubatlas.ID("test_atlas"), "head")
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
	f := newTextureFixture(t)
	f.seedAtlas(t, "test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})

	img1, _, _, err := f.provider.SubImage(context.Background(), assets.ID("sprite.png"), pubatlas.ID("test_atlas"), "head")
	if err != nil {
		t.Fatalf("first SubImage() error = %v", err)
	}
	img2, _, _, err := f.provider.SubImage(context.Background(), assets.ID("sprite.png"), pubatlas.ID("test_atlas"), "head")
	if err != nil {
		t.Fatalf("second SubImage() error = %v", err)
	}
	if img1 != img2 {
		t.Error("SubImage() returned different instances for cached subimage")
	}
}

func TestTextureProviderSubImageAtlasNotFound(t *testing.T) {
	f := newTextureFixture(t)

	_, _, _, err := f.provider.SubImage(context.Background(), assets.ID("sprite.png"), pubatlas.ID("nonexistent"), "head")
	if err == nil {
		t.Error("SubImage() for nonexistent atlas error = nil, want error")
	}
}

func TestTextureProviderSubImageRegionNotFound(t *testing.T) {
	f := newTextureFixture(t)
	f.seedAtlas(t, "test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})

	_, _, _, err := f.provider.SubImage(context.Background(), assets.ID("sprite.png"), pubatlas.ID("test_atlas"), "nonexistent")
	if err == nil {
		t.Error("SubImage() for nonexistent region error = nil, want error")
	}
}

func TestTextureProviderSubImageDimensionMismatch(t *testing.T) {
	f := newTextureFixture(t)
	// Texture is 64x64 but atlas claims 32x32.
	f.provider.images[assets.ID("sprite.png")] = &textureResource{
		image:  ebiten.NewImage(64, 64),
		width:  64,
		height: 64,
	}
	f.atlasSvc.Set("test_atlas", "sprite.png", pubatlas.NewTextureAtlas("test_atlas", "sprite.png", 32, 32, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 16, H: 16},
	}))

	_, _, _, err := f.provider.SubImage(context.Background(), assets.ID("sprite.png"), pubatlas.ID("test_atlas"), "head")
	if err == nil {
		t.Error("SubImage() with dimension mismatch error = nil, want error")
	}
}

// ---------------------------------------------------------------------------
// Invalidate
// ---------------------------------------------------------------------------

func TestTextureProviderInvalidateRemovesTexture(t *testing.T) {
	f := newTextureFixture(t)
	id := assets.ID("test.png")
	f.provider.images[id] = &textureResource{image: ebiten.NewImage(32, 32), width: 32, height: 32}

	f.provider.Invalidate(id)

	if _, ok := f.provider.images[id]; ok {
		t.Error("texture still cached after invalidation")
	}
}

func TestTextureProviderInvalidateRemovesSubImages(t *testing.T) {
	f := newTextureFixture(t)
	f.seedAtlas(t, "test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
		"body": {Name: "body", X: 0, Y: 32, W: 32, H: 32},
	})

	textureID := assets.ID("sprite.png")
	atlasID := pubatlas.ID("test_atlas")
	_, _, _, _ = f.provider.SubImage(context.Background(), textureID, atlasID, "head")
	_, _, _, _ = f.provider.SubImage(context.Background(), textureID, atlasID, "body")

	if len(f.provider.subimages) != 2 {
		t.Fatalf("expected 2 subimages cached, got %d", len(f.provider.subimages))
	}

	f.provider.Invalidate(textureID)

	for key := range f.provider.subimages {
		if key.assetID == textureID {
			t.Errorf("subimage for invalidated texture still cached: %+v", key)
		}
	}
}

func TestTextureProviderInvalidateClearsAtlasServiceEntries(t *testing.T) {
	f := newTextureFixture(t)
	f.seedAtlas(t, "atlas_a", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})
	f.seedAtlas(t, "atlas_b", "other.png", 32, 32, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 16, H: 16},
	})

	// Invalidate sprite.png — only atlas_a should be removed from the service.
	f.provider.Invalidate(assets.ID("sprite.png"))

	if f.atlasSvc.Has("atlas_a", "sprite.png") {
		t.Error("atlas_a still exists after invalidating sprite.png")
	}
	if !f.atlasSvc.Has("atlas_b", "other.png") {
		t.Error("atlas_b was removed but uses a different texture")
	}
}

func TestTextureProviderInvalidateEmptyIDClearsAll(t *testing.T) {
	f := newTextureFixture(t)
	id := assets.ID("test.png")
	f.provider.images[id] = &textureResource{image: ebiten.NewImage(1, 1), width: 1, height: 1}
	f.seedAtlas(t, "test_atlas", "test.png", 1, 1, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 1, H: 1},
	})

	f.provider.Invalidate("")

	if len(f.provider.images) != 0 {
		t.Errorf("expected 0 images after Invalidate(\"\"), got %d", len(f.provider.images))
	}
	if len(f.provider.subimages) != 0 {
		t.Errorf("expected 0 subimages after Invalidate(\"\"), got %d", len(f.provider.subimages))
	}
	if f.atlasSvc.Has("test_atlas", "test.png") {
		t.Error("atlas service entry survived Invalidate(\"\")")
	}
}

func TestTextureProviderInvalidationListenerRemovesImage(t *testing.T) {
	f := newTextureFixture(t)
	id := assets.ID("texture.png")
	f.provider.images[id] = &textureResource{image: ebiten.NewImage(1, 1), width: 1, height: 1}

	f.loader.Invalidate(id)

	if _, ok := f.provider.images[id]; ok {
		t.Fatal("texture provider image survived loader invalidation")
	}
}

// ---------------------------------------------------------------------------
// Invalidate("") — full cache clear
// ---------------------------------------------------------------------------

func TestTextureProviderInvalidateAll(t *testing.T) {
	f := newTextureFixture(t)
	// Seed two textures.
	f.provider.images[assets.ID("a.png")] = &textureResource{image: ebiten.NewImage(32, 32), width: 32, height: 32}
	f.provider.images[assets.ID("b.png")] = &textureResource{image: ebiten.NewImage(64, 64), width: 64, height: 64}
	// Seed a subimage and an atlas entry.
	f.seedAtlas(t, "atlas", "a.png", 32, 32, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 16, H: 16},
	})
	_, _, _, _ = f.provider.SubImage(context.Background(), assets.ID("a.png"), pubatlas.ID("atlas"), "head")

	if len(f.provider.images) != 2 {
		t.Fatalf("expected 2 textures cached, got %d", len(f.provider.images))
	}
	if len(f.provider.subimages) != 1 {
		t.Fatalf("expected 1 subimage cached, got %d", len(f.provider.subimages))
	}

	f.provider.Invalidate("")

	if len(f.provider.images) != 0 {
		t.Errorf("expected 0 textures after Invalidate(\"\"), got %d", len(f.provider.images))
	}
	if len(f.provider.subimages) != 0 {
		t.Errorf("expected 0 subimages after Invalidate(\"\"), got %d", len(f.provider.subimages))
	}
	if f.atlasSvc.Has("atlas", "a.png") {
		t.Error("atlas service entry survived Invalidate(\"\")")
	}
}

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

func TestTextureProviderClear(t *testing.T) {
	f := newTextureFixture(t)
	f.provider.images[assets.ID("test.png")] = &textureResource{image: ebiten.NewImage(32, 32), width: 32, height: 32}

	f.provider.ClearImageCache()

	if len(f.provider.images) != 0 {
		t.Error("images not empty after Clear()")
	}
	if len(f.provider.subimages) != 0 {
		t.Error("subimages not empty after Clear()")
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestTextureProviderConcurrentLoad(t *testing.T) {
	f := newTextureFixture(t)
	id := assets.ID("test.png")
	img := ebiten.NewImage(32, 32)
	f.provider.images[id] = &textureResource{image: img, width: 32, height: 32}

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan *ebiten.Image, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loaded, _, _, _ := f.provider.Load(context.Background(), id)
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
	f := newTextureFixture(t)
	f.seedAtlas(t, "test_atlas", "sprite.png", 64, 64, map[string]pubatlas.AtlasRegion{
		"head": {Name: "head", X: 0, Y: 0, W: 32, H: 32},
	})

	textureID := assets.ID("sprite.png")
	atlasID := pubatlas.ID("test_atlas")

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan *ebiten.Image, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			img, _, _, _ := f.provider.SubImage(context.Background(), textureID, atlasID, "head")
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

// ---------------------------------------------------------------------------
// subImageKey
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Integration: Builder -> Service -> SubImage
// ---------------------------------------------------------------------------

// TestBuilderToSubImageIntegration exercises the full production path:
// atlas.NewBuilder stores *Atlas into the internal atlas Service via Build(),
// then Texture.SubImage retrieves and type-asserts *Atlas to extract a region.
// This is the integration point where the pointer-vs-value type assertion lives.
func TestBuilderToSubImageIntegration(t *testing.T) {
	f := newTextureFixture(t)

	builder, err := pubatlas.NewBuilder("player", "sprite.png", 64, 64, f.atlasSvc)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	_, err = builder.SliceRegion("idle", 0, 0, 32, 32)
	if err != nil {
		t.Fatalf("SliceRegion() error = %v", err)
	}
	_, err = builder.SliceRegion("walk", 32, 0, 32, 32)
	if err != nil {
		t.Fatalf("SliceRegion() error = %v", err)
	}

	builtAtlas, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if builtAtlas == nil {
		t.Fatal("Build() returned nil atlas")
	}

	// Pre-seed the texture image that SubImage will load.
	f.provider.images[assets.ID("sprite.png")] = &textureResource{
		image:  ebiten.NewImage(64, 64),
		width:  64,
		height: 64,
	}

	img, w, h, err := f.provider.SubImage(
		context.Background(),
		assets.ID("sprite.png"),
		pubatlas.ID("player"),
		"idle",
	)
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
