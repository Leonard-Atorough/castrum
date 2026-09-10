package render

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

// Test constants for magic values
const (
	screenWidth  = 200
	screenHeight = 200
	entityScale = 10
	textureScale = 1
	cameraZoom  = 1.0
)

// These are smoke tests: ebiten images can't be read back outside a running
// ebiten.RunGame loop, so we can only assert DrawScene doesn't panic - which
// is still a real regression guard (e.g. against the nil-Color type-assertion
// panic this package used to have).

// mockTextureLoader implements TextureLoader without any filesystem access.
type mockTextureLoader struct {
	textures map[string]*assets.Texture
}

func (m *mockTextureLoader) Load(path string) (*assets.Texture, error) {
	tex, ok := m.textures[path]
	if !ok {
		return nil, fmt.Errorf("texture not found: %s", path)
	}
	return tex, nil
}

func newTestRenderer() *Renderer {
	// Create a 1x1 ebiten.Image as a minimal sprite (we're not testing texture
	// loading, just that rendering doesn't panic).
	testImage := ebiten.NewImage(1, 1)
	testImage.Fill(color.White)
	textureLoader := &mockTextureLoader{textures: map[string]*assets.Texture{
		"square": {Path: "square", Image: testImage, Width: 1, Height: 1},
	}}
	animationMgr := animation.NewAnimationClipStore()
	return New(textureLoader, animationMgr)
}

// setupTestWorldWithCamera creates a world with a primary camera entity.
func setupTestWorldWithCamera(world *ecs.World, width, height int) error {
	_, err := world.CreateWithComponents("camera",
		components.Camera{
			Zoom:       1.0,
			Primary:    true,
			Bounds:     unboundedRect(),
			ScreenSize: geom.Vector2I{X: width, Y: height},
		},
	)
	return err
}

func unboundedRect() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: 1e-9, Y: 1e-9},
		Max: geom.Vector2{X: 1e9, Y: 1e9},
	}
}

// testContext holds common test fixtures
type testContext struct {
	renderer *Renderer
	screen   *ebiten.Image
	world    *ecs.World
}

// setupTestContext creates a reusable test context with renderer, screen, and world
func setupTestContext(t *testing.T) testContext {
	t.Helper()
	ctx := testContext{
		renderer: newTestRenderer(),
		screen:   ebiten.NewImage(screenWidth, screenHeight),
		world:    ecs.NewWorld(),
	}
	if err := setupTestWorldWithCamera(ctx.world, screenWidth, screenHeight); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}
	return ctx
}

// createEntity is a helper to create an entity with components, failing on error
func createEntity(t *testing.T, world *ecs.World, name string, components ...ecs.Component) *ecs.Entity {
	t.Helper()
	entity, err := world.CreateWithComponents(name, components...)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}
	return entity
}

func TestRenderer_DrawScene(t *testing.T) {
	ctx := setupTestContext(t)

	t.Run("empty world draws nothing and does not panic", func(t *testing.T) {
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("primitive entities of every kind draw without panicking", func(t *testing.T) {
		kinds := []components.PrimitiveType{
			components.PrimitiveKindRectangle,
			components.PrimitiveKindCircle,
			components.PrimitiveKindLine,
		}
		for _, kind := range kinds {
			createEntity(t, ctx.world, "shape",
				components.Transform{Scale: geom.Vector2{X: entityScale, Y: entityScale}},
				components.Sprite{Primitive: kind, Visible: true},
			)
		}
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("a Transform with a nil Color does not panic (regression)", func(t *testing.T) {
		createEntity(t, ctx.world, "shape",
			components.Transform{Scale: geom.Vector2{X: entityScale, Y: entityScale}, Color: nil},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true},
		)
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("sprite entities with a registered texture draw without panicking", func(t *testing.T) {
		createEntity(t, ctx.world, "sprite",
			components.Transform{Scale: geom.Vector2{X: textureScale, Y: textureScale}, Color: color.White},
			components.Sprite{TexturePath: "square", Visible: true},
		)
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("sprite entities with a missing texture are silently skipped", func(t *testing.T) {
		createEntity(t, ctx.world, "sprite",
			components.Transform{},
			components.Sprite{TexturePath: "does-not-exist", Visible: true},
		)
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("invisible entities are skipped", func(t *testing.T) {
		createEntity(t, ctx.world, "shape",
			components.Transform{Scale: geom.Vector2{X: entityScale, Y: entityScale}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: false},
		)
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("draws entities in ascending layer order without panicking", func(t *testing.T) {
		for _, layer := range []uint8{31, 0, 10} {
			createEntity(t, ctx.world, "shape",
				components.Transform{Scale: geom.Vector2{X: entityScale, Y: entityScale}},
				components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: layer},
			)
		}
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("depth sorting within same layer and Y position", func(t *testing.T) {
		// Create three entities at same layer, same Y, different depths.
		// Expected render order (first to last): depth 10, 50, 100
		// Higher depth renders on top (drawn last).
		depths := []int8{100, 10, 50}
		for i, depth := range depths {
			createEntity(t, ctx.world, "shape",
				components.Transform{Position: geom.Vector2{X: float64(i*20) - 20, Y: 50}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
				components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: depth},
			)
		}
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("depth takes priority over Y position within same layer", func(t *testing.T) {
		// Create two entities at same layer but different Y positions and depths.
		// Higher depth should render on top regardless of Y.
		// Entity 1: Y=100, Depth=50 (should render first)
		// Entity 2: Y=50, Depth=100 (should render second, on top)
		createEntity(t, ctx.world, "back",
			components.Transform{Position: geom.Vector2{X: 0, Y: 100}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 50},
		)
		createEntity(t, ctx.world, "front",
			components.Transform{Position: geom.Vector2{X: 0, Y: 50}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 100},
		)
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})

	t.Run("Y position is fallback when layer and depth are equal", func(t *testing.T) {
		// Create two entities at same layer, same depth, different Y.
		// Should sort by Y (smaller Y renders first).
		createEntity(t, ctx.world, "lower",
			components.Transform{Position: geom.Vector2{X: 0, Y: 30}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 50},
		)
		createEntity(t, ctx.world, "higher",
			components.Transform{Position: geom.Vector2{X: 0, Y: 70}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 50},
		)
		ctx.renderer.DrawScene(ctx.screen, ctx.world)
	})
}

func TestRenderer_DrawDebugInfo(t *testing.T) {
	ctx := setupTestContext(t)
	ctx.renderer.DrawDebugInfo(ctx.screen, ctx.world)
}

func TestRenderer_Clear(t *testing.T) {
	ctx := setupTestContext(t)
	// Use a smaller screen for Clear test
	smallScreen := ebiten.NewImage(10, 10)
	ctx.renderer.Clear(smallScreen, color.Black)
}

func TestRenderer_DrawScene_WithPolygonPrimitive(t *testing.T) {
	ctx := setupTestContext(t)

	// Create a polygon entity with valid polygon data
	triangle := &geom.Polygon{
		Points: []geom.Vector2{
			{X: 0, Y: -10},
			{X: -10, Y: 10},
			{X: 10, Y: 10},
		},
	}

	createEntity(t, ctx.world, "polygon",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 1, Y: 1}},
		components.Sprite{Primitive: components.PrimitiveKindPolygon, Visible: true, Data: triangle},
	)

	// Should not panic
	ctx.renderer.DrawScene(ctx.screen, ctx.world)
}

func TestRenderer_DrawScene_WithInvalidPolygon(t *testing.T) {
	ctx := setupTestContext(t)

	// Create a polygon entity with insufficient points (< 3)
	invalidPolygon := &geom.Polygon{
		Points: []geom.Vector2{
			{X: 0, Y: 0},
			{X: 1, Y: 1},
		},
	}

	createEntity(t, ctx.world, "invalid_polygon",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 1, Y: 1}},
		components.Sprite{Primitive: components.PrimitiveKindPolygon, Visible: true, Data: invalidPolygon},
	)

	// Should not panic (should silently skip)
	ctx.renderer.DrawScene(ctx.screen, ctx.world)
}

func TestRenderer_DrawScene_WithTexturedSprite(t *testing.T) {
	ctx := setupTestContext(t)

	// Create a textured sprite entity
	createEntity(t, ctx.world, "textured_sprite",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
		components.Sprite{TexturePath: "square", Visible: true},
	)

	// Should not panic
	ctx.renderer.DrawScene(ctx.screen, ctx.world)
}

func TestRenderer_DrawScene_WithAnimatedSprite(t *testing.T) {
	ctx := setupTestContext(t)

	// Create an animated sprite entity
	anim := components.Animation{
		ClipPath:      "idle",
		FrameIndex:    0,
		FrameTime:     0.0,
		Playing:       true,
		PlaybackSpeed: 1.0,
	}

	createEntity(t, ctx.world, "animated_sprite",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
		components.Sprite{TexturePath: "square", Visible: true},
		anim,
	)

	// Should not panic
	ctx.renderer.DrawScene(ctx.screen, ctx.world)
}

func TestRenderer_DrawScene_WithFrustumCulling(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(screenWidth, screenHeight)
	world := ecs.NewWorld()

	// Create camera with limited bounds
	createEntity(t, world, "camera",
		components.Camera{
			Zoom:       cameraZoom,
			Primary:    true,
			Bounds:     geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: screenWidth, Y: screenHeight}},
			ScreenSize: geom.Vector2I{X: screenWidth, Y: screenHeight},
		},
	)

	// Create entity outside camera bounds
	createEntity(t, world, "outside",
		components.Transform{Position: geom.Vector2{X: 500, Y: 500}, Scale: geom.Vector2{X: entityScale, Y: entityScale}},
		components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true},
	)

	// Should not panic, entity should be culled
	renderer.DrawScene(screen, world)
}

func TestRenderer_DrawDebugInfo_WithCulledEntities(t *testing.T) {
	ctx := setupTestContext(t)

	// Create entity with transform but no other components
	createEntity(t, ctx.world, "debug_target",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 5, Y: 5}},
	)

	// Should not panic
	ctx.renderer.DrawDebugInfo(ctx.screen, ctx.world)
}
