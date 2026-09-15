package render

import (
	"context"
	"fmt"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/assets"
	pubatlas "github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

// stubTextureProvider implements TextureProvider without filesystem or GPU
// access. It records every Load and SubImage call so tests can assert which
// textures were requested and in what order. Successful loads are cached so
// repeated calls (e.g. cull pass + render pass) don't inflate call counts,
// matching the real TextureProvider's behaviour.
type stubTextureProvider struct {
	textures      map[assets.ID]*ebiten.Image
	dimensions    map[assets.ID][2]int // optional: override returned (w,h) per texture
	loadCalls     []assets.ID
	subImageCalls []stubSubImageCall
	cache         map[assets.ID]*ebiten.Image
}

type stubSubImageCall struct {
	assetID    assets.ID
	atlasID    pubatlas.ID
	regionName string
}

func newStubTextureProvider() *stubTextureProvider {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return &stubTextureProvider{
		textures: map[assets.ID]*ebiten.Image{
			"square": img,
		},
		dimensions: make(map[assets.ID][2]int),
		cache:      make(map[assets.ID]*ebiten.Image),
	}
}

func (s *stubTextureProvider) Load(_ context.Context, id assets.ID) (*ebiten.Image, int, int, error) {
	if cached, ok := s.cache[id]; ok {
		w, h := s.textureDims(id)
		return cached, w, h, nil
	}
	s.loadCalls = append(s.loadCalls, id)
	tex, ok := s.textures[id]
	if !ok {
		return nil, 0, 0, fmt.Errorf("texture not found: %s", id)
	}
	s.cache[id] = tex
	w, h := s.textureDims(id)
	return tex, w, h, nil
}

func (s *stubTextureProvider) textureDims(id assets.ID) (int, int) {
	if dims, ok := s.dimensions[id]; ok {
		return dims[0], dims[1]
	}
	return 1, 1
}

func (s *stubTextureProvider) SubImage(_ context.Context, assetID assets.ID, atlasID pubatlas.ID, regionName string) (*ebiten.Image, int, int, error) {
	s.subImageCalls = append(s.subImageCalls, stubSubImageCall{
		assetID:    assetID,
		atlasID:    atlasID,
		regionName: regionName,
	})
	return ebiten.NewImage(1, 1), 1, 1, nil
}

// testRenderer bundles a Renderer with its world, screen, and stub provider so
// each test gets isolated state. The embedded *Renderer lets tests call
// DrawScene and inspect renderItems/renderErrors directly.
type testRenderer struct {
	*Renderer
	world    *ecs.World
	screen   *ebiten.Image
	provider *stubTextureProvider
}

func newTestRenderer(t *testing.T) testRenderer {
	t.Helper()
	provider := newStubTextureProvider()
	world := ecs.NewWorld()
	return testRenderer{
		Renderer: New(provider, world, RenderConfig{}),
		world:    world,
		screen:   ebiten.NewImage(200, 200),
		provider: provider,
	}
}

func (tr testRenderer) withCamera(t *testing.T) testRenderer {
	t.Helper()
	_, err := tr.world.CreateWithComponents("camera",
		components.Camera{
			Zoom:       1.0,
			Primary:    true,
			Bounds:     components.UnboundedRect(),
			ScreenSize: geom.Vector2I{X: 200, Y: 200},
		},
	)
	if err != nil {
		t.Fatalf("failed to create camera: %v", err)
	}
	return tr
}

// addEntity creates a sprite entity and returns its ID.
func (tr testRenderer) addEntity(t *testing.T, sprite components.Sprite, transform components.Transform) ecs.EntityID {
	t.Helper()
	entity, err := tr.world.CreateWithComponents("test", sprite, transform)
	if err != nil {
		t.Fatalf("failed to create entity: %v", err)
	}
	return entity.ID
}

func assertRenderOrder(t *testing.T, items []renderItem, want []ecs.EntityID) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("expected %d render items, got %d", len(want), len(items))
	}
	for i, item := range items {
		if item.entityID != want[i] {
			t.Errorf("render item %d: entityID = %d, want %d", i, item.entityID, want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// DrawScene: camera handling
// ---------------------------------------------------------------------------

func TestDrawScene_NoCameraReturnsEarly(t *testing.T) {
	tr := newTestRenderer(t)
	tr.addEntity(t,
		components.Sprite{TexturePath: "square", Visible: true},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderItems) != 0 {
		t.Errorf("expected 0 render items without camera, got %d", len(tr.renderItems))
	}
	if len(tr.provider.loadCalls) != 0 {
		t.Errorf("expected 0 Load calls without camera, got %d", len(tr.provider.loadCalls))
	}
}

func TestDrawScene_EmptyWorldRendersNothing(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderItems) != 0 {
		t.Errorf("expected 0 render items in empty world, got %d", len(tr.renderItems))
	}
	if len(tr.renderErrors) != 0 {
		t.Errorf("expected 0 render errors in empty world, got %d", len(tr.renderErrors))
	}
}

// ---------------------------------------------------------------------------
// DrawScene: rendering paths (smoke tests — ebiten images can't be read back
// outside a running game loop, so we assert "no panic" plus side effects)
// ---------------------------------------------------------------------------

func TestDrawScene_PrimitivesDoNotPanic(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	for _, kind := range []components.PrimitiveType{
		components.PrimitiveKindRectangle,
		components.PrimitiveKindCircle,
		components.PrimitiveKindLine,
	} {
		tr.addEntity(t,
			components.Sprite{Primitive: kind, Size: geom.Vector2{X: 10, Y: 10}, Visible: true},
			components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		)
	}

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderErrors) != 0 {
		t.Errorf("expected 0 render errors for primitives, got %d", len(tr.renderErrors))
	}
}

func TestDrawScene_NilColorDoesNotPanic(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	tr.addEntity(t,
		components.Sprite{Primitive: components.PrimitiveKindRectangle, Size: geom.Vector2{X: 10, Y: 10}, Visible: true, Color: nil},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderErrors) != 0 {
		t.Errorf("expected 0 render errors for nil color, got %d", len(tr.renderErrors))
	}
}

func TestDrawScene_TextureSpriteLoadsTexture(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	tr.addEntity(t,
		components.Sprite{TexturePath: "square", Visible: true},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.provider.loadCalls) != 1 {
		t.Fatalf("expected 1 Load call, got %d", len(tr.provider.loadCalls))
	}
	if tr.provider.loadCalls[0] != "square" {
		t.Errorf("Load called with %q, want %q", tr.provider.loadCalls[0], "square")
	}
	if len(tr.renderErrors) != 0 {
		t.Errorf("expected 0 render errors, got %d: %v", len(tr.renderErrors), tr.renderErrors)
	}
}

func TestDrawScene_MissingTextureCapturesError(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	id := tr.addEntity(t,
		components.Sprite{TexturePath: "missing", Visible: true},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderErrors) != 1 {
		t.Fatalf("expected 1 render error, got %d", len(tr.renderErrors))
	}
	if tr.renderErrors[0].EntityID != id {
		t.Errorf("error entity ID = %d, want %d", tr.renderErrors[0].EntityID, id)
	}
}

// ---------------------------------------------------------------------------
// DrawScene: visibility
// ---------------------------------------------------------------------------

func TestDrawScene_InvisibleSpriteSkipsTextureLoad(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	tr.addEntity(t,
		components.Sprite{TexturePath: "square", Visible: false},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.provider.loadCalls) != 0 {
		t.Errorf("expected 0 Load calls for invisible sprite, got %d", len(tr.provider.loadCalls))
	}
	if len(tr.renderErrors) != 0 {
		t.Errorf("expected 0 render errors for invisible sprite, got %d", len(tr.renderErrors))
	}
}

// ---------------------------------------------------------------------------
// DrawScene: viewport culling
// ---------------------------------------------------------------------------

func TestDrawScene_CullsEntitiesOutsideViewport(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	// Camera at (0,0), screen 200x200, zoom 1: viewport is (-100,-100)..(100,100)
	inside := tr.addEntity(t,
		components.Sprite{TexturePath: "square", Visible: true},
		components.Transform{
			Position: geom.Vector2{X: 50, Y: 50},
			Scale:    geom.Vector2{X: 10, Y: 10},
		},
	)
	tr.addEntity(t,
		components.Sprite{TexturePath: "square", Visible: true},
		components.Transform{
			Position: geom.Vector2{X: 500, Y: 500},
			Scale:    geom.Vector2{X: 10, Y: 10},
		},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderItems) != 1 {
		t.Fatalf("expected 1 render item (outside entity culled), got %d", len(tr.renderItems))
	}
	if tr.renderItems[0].entityID != inside {
		t.Errorf("expected inside entity %d, got %d", inside, tr.renderItems[0].entityID)
	}
}

// TestDrawScene_CullsPrimitivesBySpriteSize verifies that primitive culling
// uses Sprite.Size * Scale as the rendered dimensions, not Scale alone.
func TestDrawScene_CullsPrimitivesBySpriteSize(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	// Camera at (0,0), screen 200x200: viewport is (-100,-100)..(100,100)
	// A 20x20 primitive at (90,0) with Scale{1,1}: half-extents = 10, bounds
	// (80,-10)-(100,10) — just inside the viewport.
	inside := tr.addEntity(t,
		components.Sprite{Primitive: components.PrimitiveKindRectangle, Size: geom.Vector2{X: 20, Y: 20}, Visible: true},
		components.Transform{Position: geom.Vector2{X: 90, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	// Same size at (200,0): bounds (190,-10)-(210,10) — outside.
	tr.addEntity(t,
		components.Sprite{Primitive: components.PrimitiveKindRectangle, Size: geom.Vector2{X: 20, Y: 20}, Visible: true},
		components.Transform{Position: geom.Vector2{X: 200, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderItems) != 1 {
		t.Fatalf("expected 1 render item (outside primitive culled), got %d", len(tr.renderItems))
	}
	if tr.renderItems[0].entityID != inside {
		t.Errorf("expected inside entity %d, got %d", inside, tr.renderItems[0].entityID)
	}
}

// TestDrawScene_CullsTexturedSpriteByImageDimensions verifies that textured
// sprite culling uses image dimensions * Scale, not Scale alone. Under the
// old code, a 256x256 texture at (200,0) with Scale{1,1} was culled (bounds
// used Scale=1 as half-extents), even though the rendered sprite's half-width
// is 128 and it overlaps the viewport.
func TestDrawScene_CullsTexturedSpriteByImageDimensions(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	// Camera at (0,0), screen 200x200: viewport is (-100,-100)..(100,100)
	// Stub returns 256x256 for "big" texture. At (200,0) with Scale{1,1}:
	// half-extents = 128, bounds (72,-128)-(328,128) — overlaps viewport.
	tr.provider.textures["big"] = ebiten.NewImage(256, 256)
	tr.provider.dimensions["big"] = [2]int{256, 256}

	visible := tr.addEntity(t,
		components.Sprite{TexturePath: "big", Visible: true},
		components.Transform{Position: geom.Vector2{X: 200, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	if len(tr.renderItems) != 1 {
		t.Fatalf("expected 1 render item (large texture overlaps viewport), got %d", len(tr.renderItems))
	}
	if tr.renderItems[0].entityID != visible {
		t.Errorf("expected entity %d, got %d", visible, tr.renderItems[0].entityID)
	}
}

// ---------------------------------------------------------------------------
// DrawScene: sort order (layer -> sort order -> Y position)
// ---------------------------------------------------------------------------

func TestDrawScene_SortByLayer(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	// Create in non-sorted order.
	third := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 20},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)
	first := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 0},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)
	second := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 10},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	assertRenderOrder(t, tr.renderItems, []ecs.EntityID{first, second, third})
}

func TestDrawScene_SortBySortOrderWithinLayer(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	third := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 0, SortOrder: 100},
		components.Transform{Position: geom.Vector2{X: 0, Y: 50}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	first := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 0, SortOrder: 10},
		components.Transform{Position: geom.Vector2{X: 20, Y: 50}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	second := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 0, SortOrder: 50},
		components.Transform{Position: geom.Vector2{X: 40, Y: 50}, Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	assertRenderOrder(t, tr.renderItems, []ecs.EntityID{first, second, third})
}

func TestDrawScene_LayerTakesPriorityOverSortOrder(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	// Lower layer with higher sort order still renders first.
	first := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 5, SortOrder: 100},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)
	second := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 10, SortOrder: 1},
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	assertRenderOrder(t, tr.renderItems, []ecs.EntityID{first, second})
}

func TestDrawScene_YPositionFallbackWhenLayerAndSortOrderEqual(t *testing.T) {
	tr := newTestRenderer(t).withCamera(t)
	second := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 0, SortOrder: 50},
		components.Transform{Position: geom.Vector2{X: 0, Y: 70}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	first := tr.addEntity(t,
		components.Sprite{Size: geom.Vector2{X: 10, Y: 10}, Visible: true, RenderLayer: 0, SortOrder: 50},
		components.Transform{Position: geom.Vector2{X: 0, Y: 30}, Scale: geom.Vector2{X: 1, Y: 1}},
	)

	tr.DrawScene(context.Background(), tr.screen)

	assertRenderOrder(t, tr.renderItems, []ecs.EntityID{first, second})
}

// ---------------------------------------------------------------------------
// DrawScene: debug info
// ---------------------------------------------------------------------------

func TestDrawScene_DebugInfoDoesNotPanic(t *testing.T) {
	provider := newStubTextureProvider()
	world := ecs.NewWorld()
	_, err := world.CreateWithComponents("camera",
		components.Camera{
			Zoom:       1.0,
			Primary:    true,
			Bounds:     components.UnboundedRect(),
			ScreenSize: geom.Vector2I{X: 200, Y: 200},
		},
	)
	if err != nil {
		t.Fatalf("failed to create camera: %v", err)
	}
	renderer := New(provider, world, RenderConfig{DrawDebugInfo: true})

	renderer.DrawScene(context.Background(), ebiten.NewImage(200, 200))
}

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

func TestRenderer_Clear(t *testing.T) {
	tr := newTestRenderer(t)
	tr.Clear(tr.screen, color.Black)
}
