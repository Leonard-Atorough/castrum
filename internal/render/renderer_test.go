package render

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/texture"
)

// These are smoke tests: ebiten images can't be read back outside a running
// ebiten.RunGame loop, so we can only assert DrawScene doesn't panic - which
// is still a real regression guard (e.g. against the nil-Color type-assertion
// panic this package used to have).
func newTestRenderer() *Renderer {
	store := texture.NewStore(nil)
	// Manually populate store with a test texture (we're not testing texture loading,
	// just that rendering doesn't panic). Create a 1x1 ebiten.Image as a minimal sprite.
	testImage := ebiten.NewImage(1, 1)
	testImage.Fill(color.White)
	store.Textures["square"] = &texture.Texture{
		Path:   "square",
		Image:  testImage,
		Width:  1,
		Height: 1,
	}
	return New(&assets.Assets{Textures: store})
}

func TestRenderer_DrawScene(t *testing.T) {
	renderer := newTestRenderer()
	camera := NewCamera()
	camera.SetScreenSize(200, 200)
	screen := ebiten.NewImage(200, 200)

	t.Run("empty world draws nothing and does not panic", func(t *testing.T) {
		world := core.NewWorld()
		renderer.DrawScene(screen, camera, world)
	})

	t.Run("primitive entities of every kind draw without panicking", func(t *testing.T) {
		world := core.NewWorld()
		kinds := []components.PrimitiveKind{
			components.PrimitiveKindRectangle,
			components.PrimitiveKindCircle,
			components.PrimitiveKindLine,
		}
		for _, kind := range kinds {
			_, err := world.CreateWithComponents("shape",
				components.Transform{Scale: geom.Vector2{X: 10, Y: 10}},
				components.Renderable{Primitive: kind, Visible: true},
			)
			if err != nil {
				t.Fatalf("CreateWithComponents failed: %v", err)
			}
		}
		renderer.DrawScene(screen, camera, world)
	})

	t.Run("a Transform with a nil Color does not panic (regression)", func(t *testing.T) {
		world := core.NewWorld()
		_, err := world.CreateWithComponents("shape",
			components.Transform{Scale: geom.Vector2{X: 10, Y: 10}, Color: nil},
			components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: true},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, camera, world)
	})

	t.Run("sprite entities with a registered texture draw without panicking", func(t *testing.T) {
		world := core.NewWorld()
		_, err := world.CreateWithComponents("sprite",
			components.Transform{Scale: geom.Vector2{X: 1, Y: 1}, Color: color.White},
			components.Renderable{TexturePath: "square", Visible: true},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, camera, world)
	})

	t.Run("sprite entities with a missing texture are silently skipped", func(t *testing.T) {
		world := core.NewWorld()
		_, err := world.CreateWithComponents("sprite",
			components.Transform{},
			components.Renderable{TexturePath: "does-not-exist", Visible: true},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, camera, world)
	})

	t.Run("invisible entities are skipped", func(t *testing.T) {
		world := core.NewWorld()
		id, err := world.CreateWithComponents("shape",
			components.Transform{Scale: geom.Vector2{X: 10, Y: 10}},
			components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: false},
		)
		if err != nil {
			t.Fatalf("CreateWithComponents failed: %v", err)
		}
		renderer.DrawScene(screen, camera, world)
		_ = id
	})

	t.Run("draws entities in ascending layer order without panicking", func(t *testing.T) {
		world := core.NewWorld()
		for _, layer := range []components.RenderLayer{components.LayerDebug, components.Layer0, components.Layer10} {
			_, err := world.CreateWithComponents("shape",
				components.Transform{Scale: geom.Vector2{X: 10, Y: 10}},
				components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: layer},
			)
			if err != nil {
				t.Fatalf("CreateWithComponents failed: %v", err)
			}
		}
		renderer.DrawScene(screen, camera, world)
	})
}

func TestRenderer_DrawDebugInfo(t *testing.T) {
	renderer := newTestRenderer()
	camera := NewCamera()
	camera.SetScreenSize(200, 200)
	screen := ebiten.NewImage(200, 200)
	world := core.NewWorld()

	renderer.DrawDebugInfo(screen, camera, world)
}

func TestRenderer_Clear(t *testing.T) {
	renderer := newTestRenderer()
	screen := ebiten.NewImage(10, 10)
	renderer.Clear(screen, color.Black)
}
