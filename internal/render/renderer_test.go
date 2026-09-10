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

func TestRenderer_DrawScene(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)

	t.Run("empty world draws nothing and does not panic", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("primitive entities of every kind draw without panicking", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		kinds := []components.PrimitiveType{
			components.PrimitiveKindRectangle,
			components.PrimitiveKindCircle,
			components.PrimitiveKindLine,
		}
		for _, kind := range kinds {
			_, err := world.CreateWithComponents("shape",
				components.Transform{Scale: geom.Vector2{X: 10, Y: 10}},
				components.Sprite{Primitive: kind, Visible: true},
			)
			if err != nil {
				t.Fatalf("CreateWithComponents failed: %v", err)
			}
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("a Transform with a nil Color does not panic (regression)", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		_, err := world.CreateWithComponents("shape",
			components.Transform{Scale: geom.Vector2{X: 10, Y: 10}, Color: nil},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("sprite entities with a registered texture draw without panicking", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		_, err := world.CreateWithComponents("sprite",
			components.Transform{Scale: geom.Vector2{X: 1, Y: 1}, Color: color.White},
			components.Sprite{TexturePath: "square", Visible: true},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("sprite entities with a missing texture are silently skipped", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		_, err := world.CreateWithComponents("sprite",
			components.Transform{},
			components.Sprite{TexturePath: "does-not-exist", Visible: true},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("invisible entities are skipped", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		id, err := world.CreateWithComponents("shape",
			components.Transform{Scale: geom.Vector2{X: 10, Y: 10}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: false},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, world)
		_ = id
	})

	t.Run("draws entities in ascending layer order without panicking", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		for _, layer := range []uint8{31, 0, 10} {
			_, err := world.CreateWithComponents("shape",
				components.Transform{Scale: geom.Vector2{X: 10, Y: 10}},
				components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: layer},
			)
			if err != nil {
				t.Fatalf("CreateWithComponents failed: %v", err)
			}
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("depth sorting within same layer and Y position", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		// Create three entities at same layer, same Y, different depths.
		// Expected render order (first to last): depth 10, 50, 100
		// Higher depth renders on top (drawn last).
		depths := []int8{100, 10, 50}
		for i, depth := range depths {
			_, err := world.CreateWithComponents("shape",
				components.Transform{Position: geom.Vector2{X: float64(i*20) - 20, Y: 50}, Scale: geom.Vector2{X: 10, Y: 10}},
				components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: depth},
			)
			if err != nil {
				t.Fatalf("CreateWithComponents failed: %v", err)
			}
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("depth takes priority over Y position within same layer", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		// Create two entities at same layer but different Y positions and depths.
		// Higher depth should render on top regardless of Y.
		// Entity 1: Y=100, Depth=50 (should render first)
		// Entity 2: Y=50, Depth=100 (should render second, on top)
		_, err := world.CreateWithComponents("back",
			components.Transform{Position: geom.Vector2{X: 0, Y: 100}, Scale: geom.Vector2{X: 10, Y: 10}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 50},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		_, err = world.CreateWithComponents("front",
			components.Transform{Position: geom.Vector2{X: 0, Y: 50}, Scale: geom.Vector2{X: 10, Y: 10}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 100},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, world)
	})

	t.Run("Y position is fallback when layer and depth are equal", func(t *testing.T) {
		world := ecs.NewWorld()
		if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
			t.Fatalf("setupTestWorldWithCamera failed: %v", err)
		}
		// Create two entities at same layer, same depth, different Y.
		// Should sort by Y (smaller Y renders first).
		_, err := world.CreateWithComponents("lower",
			components.Transform{Position: geom.Vector2{X: 0, Y: 30}, Scale: geom.Vector2{X: 10, Y: 10}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 50},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		_, err = world.CreateWithComponents("higher",
			components.Transform{Position: geom.Vector2{X: 0, Y: 70}, Scale: geom.Vector2{X: 10, Y: 10}},
			components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0, SortOrder: 50},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, world)
	})
}

func TestRenderer_DrawDebugInfo(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()
	if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}

	renderer.DrawDebugInfo(screen, world)
}

func TestRenderer_Clear(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(10, 10)
	renderer.Clear(screen, color.Black)
}

func TestRenderer_DrawScene_WithPolygonPrimitive(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()
	if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}

	// Create a polygon entity with valid polygon data
	triangle := &geom.Polygon{
		Points: []geom.Vector2{
			{X: 0, Y: -10},
			{X: -10, Y: 10},
			{X: 10, Y: 10},
		},
	}

	_, err := world.CreateWithComponents("polygon",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 1, Y: 1}},
		components.Sprite{Primitive: components.PrimitiveKindPolygon, Visible: true, Data: triangle},
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Should not panic
	renderer.DrawScene(screen, world)
}

func TestRenderer_DrawScene_WithInvalidPolygon(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()
	if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}

	// Create a polygon entity with insufficient points (< 3)
	invalidPolygon := &geom.Polygon{
		Points: []geom.Vector2{
			{X: 0, Y: 0},
			{X: 1, Y: 1},
		},
	}

	_, err := world.CreateWithComponents("invalid_polygon",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 1, Y: 1}},
		components.Sprite{Primitive: components.PrimitiveKindPolygon, Visible: true, Data: invalidPolygon},
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Should not panic (should silently skip)
	renderer.DrawScene(screen, world)
}

func TestRenderer_DrawScene_WithTexturedSprite(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()
	if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}

	// Create a textured sprite entity
	_, err := world.CreateWithComponents("textured_sprite",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 10, Y: 10}},
		components.Sprite{TexturePath: "square", Visible: true},
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Should not panic
	renderer.DrawScene(screen, world)
}

func TestRenderer_DrawScene_WithAnimatedSprite(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()
	if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}

	// Create an animated sprite entity
	anim := components.Animation{
		ClipPath:      "idle",
		FrameIndex:    0,
		FrameTime:     0.0,
		Playing:       true,
		PlaybackSpeed: 1.0,
	}

	_, err := world.CreateWithComponents("animated_sprite",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 10, Y: 10}},
		components.Sprite{TexturePath: "square", Visible: true},
		anim,
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Should not panic
	renderer.DrawScene(screen, world)
}

func TestRenderer_DrawScene_WithFrustumCulling(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()

	// Create camera with limited bounds
	_, err := world.CreateWithComponents("camera",
		components.Camera{
			Zoom:       1.0,
			Primary:    true,
			Bounds:     geom.Rect{Min: geom.Vector2{X: 0, Y: 0}, Max: geom.Vector2{X: 200, Y: 200}},
			ScreenSize: geom.Vector2I{X: 200, Y: 200},
		},
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Create entity outside camera bounds
	_, err = world.CreateWithComponents("outside",
		components.Transform{Position: geom.Vector2{X: 500, Y: 500}, Scale: geom.Vector2{X: 10, Y: 10}},
		components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true},
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Should not panic, entity should be culled
	renderer.DrawScene(screen, world)
}

func TestRenderer_DrawDebugInfo_WithCulledEntities(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(200, 200)
	world := ecs.NewWorld()
	if err := setupTestWorldWithCamera(world, 200, 200); err != nil {
		t.Fatalf("setupTestWorldWithCamera failed: %v", err)
	}

	// Create entity with transform but no other components
	_, err := world.CreateWithComponents("debug_target",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}, Scale: geom.Vector2{X: 5, Y: 5}},
	)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Should not panic
	renderer.DrawDebugInfo(screen, world)
}
