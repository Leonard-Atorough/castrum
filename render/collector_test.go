package render

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	spr "github.com/Leonard-Atorough/castrum/sprite"
)

var unitScale = geom.Vector2{X: 1, Y: 1}

func newCollectorWorld(t *testing.T) *core.World {
	t.Helper()
	encode := func() []byte {
		var buf bytes.Buffer
		if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
			t.Fatalf("encode test png: %v", err)
		}
		return buf.Bytes()
	}
	files := fstest.MapFS{
		"tex.png":  &fstest.MapFile{Data: encode()},
		"tex2.png": &fstest.MapFile{Data: encode()},
		"sprites.json": &fstest.MapFile{Data: []byte(
			`{"texture":"./tex.png","regions":{"player":{"x":0,"y":0,"w":2,"h":2},"enemy":{"x":2,"y":2,"w":2,"h":2}}}`)},
		"oversized.json": &fstest.MapFile{Data: []byte(
			`{"texture":"tex.png","regions":{"player":{"x":0,"y":0,"w":99,"h":2}}}`)},
		"missing-texture.json": &fstest.MapFile{Data: []byte(
			`{"texture":"missing.png","regions":{"player":{"x":0,"y":0,"w":2,"h":2}}}`)},
	}
	w := core.NewWorld()
	if err := w.Provide(func(*core.World) (*asset.Loader, error) {
		return asset.NewLoader(files, asset.LoaderOptions{})
	}); err != nil {
		t.Fatalf("provide asset loader: %v", err)
	}
	return w
}

func newFontWorld(t *testing.T) *core.World {
	t.Helper()
	files := fstest.MapFS{
		"fonts/go.ttf": &fstest.MapFile{Data: goregular.TTF},
	}
	w := core.NewWorld()
	if err := w.Provide(func(*core.World) (*asset.Loader, error) {
		return asset.NewLoader(files, asset.LoaderOptions{})
	}); err != nil {
		t.Fatalf("provide asset loader: %v", err)
	}
	return w
}

func drawContext(w *core.World) *core.Context {
	if _, err := w.Resource[Window](); err != nil {
		_ = w.Provide(func(*core.World) (Window, error) {
			return NewStaticWindow(Viewport{LogicalWidth: 100, LogicalHeight: 100, WindowWidth: 100, WindowHeight: 100}), nil
		})
	}
	return &core.Context{World: w}
}

func collect(t *testing.T, c *Collector, ctx *core.Context, alpha float64) DrawList {
	t.Helper()
	list, err := c.Collect(ctx, alpha)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return list
}

func spawnCamera(t *testing.T, w *core.World, prev, curr geom.Vector2, zoom float64, primary bool) {
	t.Helper()

	if _, err := w.NewEntity(
		Camera{Primary: primary},
		core.Transform{Position: curr, Scale: geom.Vector2{X: zoom, Y: zoom}},
		core.PrevTransform{Position: prev, Scale: geom.Vector2{X: zoom, Y: zoom}},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
}

func spawnSprite(t *testing.T, w *core.World, prev, curr geom.Vector2, visual any, transform core.Transform) core.Entity {
	t.Helper()
	transform.Position = curr
	if transform.Scale == (geom.Vector2{}) {
		transform.Scale = unitScale
	}

	e, err := w.NewEntity(visual, transform,
		core.PrevTransform{Position: prev, Rotation: transform.Rotation, Scale: transform.Scale})
	if err != nil {
		t.Fatalf("spawn visual: %v", err)
	}
	return e
}

func spawnTextureSprite(t *testing.T, w *core.World, texture asset.ID, prev, curr geom.Vector2, sprite spr.Sprite) core.Entity {
	t.Helper()
	sprite.Asset = texture
	return spawnSprite(t, w, prev, curr, sprite, core.Transform{})
}

func spawnTextSprite(t *testing.T, w *core.World, text string, fill *color.RGBA, pos geom.Vector2) core.Entity {
	t.Helper()
	entity, err := w.NewEntity(
		spr.Text{Asset: "fonts/go.ttf", Text: text, TextSize: 16, Style: spr.Style{Color: fill}},
		core.Transform{Position: pos, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	if err != nil {
		t.Fatalf("spawn text sprite: %v", err)
	}
	return entity
}

func TestCollectWithoutPrimaryCameraReturnsEmptyList(t *testing.T) {
	w := newCollectorWorld(t)
	// The missing texture is deliberate: sprite resolution must not
	// happen on a no-camera path.
	spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})

	if list := collect(t, NewCollector(w), drawContext(w), 0.5); len(list.Items) != 0 {
		t.Errorf("Items = %d, want 0", len(list.Items))
	}
}

func TestCollectNeverDraws(t *testing.T) {
	cases := []struct {
		name       string
		world      func(*testing.T) *core.World
		setup      func(*testing.T, *core.World)
		want       int
		survivor   asset.ID
		survivorIs spr.DrawShape
	}{
		{"non-primary camera", newCollectorWorld, func(t *testing.T, w *core.World) {
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, false)
		}, 0, "", nil},
		{"hidden texture sprite", newCollectorWorld, func(t *testing.T, w *core.World) {
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})
			spawnTextureSprite(t, w, "tex2.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{Style: spr.Style{Hidden: true}})
		}, 1, "tex.png", nil},
		{"hidden shape", newCollectorWorld, func(t *testing.T, w *core.World) {
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
				spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
			spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
				spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}, Style: spr.Style{Hidden: true}}, core.Transform{})
		}, 1, "", spr.RectShape{Size: geom.Vector2{X: 10, Y: 10}}},
		// Style without a picture is legal and must not draw.
		{"nil drawable", newCollectorWorld, func(t *testing.T, w *core.World) {
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			spawnSprite(t, w, geom.Vector2{}, geom.Vector2{}, spr.Sprite{Style: spr.Style{Layer: 3}}, core.Transform{})
			spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
				spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
		}, 1, "", spr.RectShape{Size: geom.Vector2{X: 10, Y: 10}}},
		{"culled text", newFontWorld, func(t *testing.T, w *core.World) {
			spawnCamera(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50}, 1, true)
			spawnTextSprite(t, w, "off screen", nil, geom.Vector2{X: 500, Y: 500})
		}, 0, "", nil},
		{"empty text", newFontWorld, func(t *testing.T, w *core.World) {
			spawnCamera(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50}, 1, true)
			spawnTextSprite(t, w, "", nil, geom.Vector2{X: 50, Y: 50})
		}, 0, "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.world(t)
			tc.setup(t, w)
			list := collect(t, NewCollector(w), drawContext(w), 0)
			if len(list.Items) != tc.want {
				t.Fatalf("Items = %d, want %d", len(list.Items), tc.want)
			}
			if tc.survivor != "" && list.Items[0].Texture != tc.survivor {
				t.Errorf("survivor texture = %q, want %q", list.Items[0].Texture, tc.survivor)
			}
			if tc.survivorIs != nil && list.Items[0].Shape != tc.survivorIs {
				t.Errorf("survivor shape = %v, want %v", list.Items[0].Shape, tc.survivorIs)
			}
		})
	}
}

func TestCollectInterpolatesCamera(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{X: 10, Y: 4}, 2, true)

	collector := NewCollector(w)
	for _, tc := range []struct {
		alpha float64
		want  geom.Vector2
	}{
		{0, geom.Vector2{}},
		{0.5, geom.Vector2{X: 5, Y: 2}},
		{1, geom.Vector2{X: 10, Y: 4}},
	} {
		list := collect(t, collector, drawContext(w), tc.alpha)
		if !list.Camera.Position.AlmostEqual(tc.want, 1e-9) {
			t.Errorf("alpha %v: camera position = %v, want %v", tc.alpha, list.Camera.Position, tc.want)
		}
		if list.Camera.Zoom != 2 {
			t.Errorf("alpha %v: camera zoom = %v, want 2", tc.alpha, list.Camera.Zoom)
		}
	}
}

func TestCollectFirstPrimaryCameraWins(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{X: 10, Y: 0}, 1, true)
	spawnCamera(t, w, geom.Vector2{X: 100, Y: 0}, geom.Vector2{X: 200, Y: 0}, 1, true)

	list := collect(t, NewCollector(w), drawContext(w), 0.5)
	want := geom.Vector2{X: 5, Y: 0}
	if !list.Camera.Position.AlmostEqual(want, 1e-9) {
		t.Errorf("camera position = %v, want %v from the first primary", list.Camera.Position, want)
	}
}

func TestCollectInterpolatesTextureSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png",
		geom.Vector2{}, geom.Vector2{X: 10, Y: 20},
		spr.Sprite{Style: spr.Style{FlipH: true, Color: &color.RGBA{R: 128, A: 64}}})

	collector := NewCollector(w)
	for _, tc := range []struct {
		alpha float64
		want  geom.Vector2
	}{
		{0, geom.Vector2{}},
		{0.5, geom.Vector2{X: 5, Y: 10}},
		{1, geom.Vector2{X: 10, Y: 20}},
	} {
		list := collect(t, collector, drawContext(w), tc.alpha)
		if len(list.Items) != 1 {
			t.Fatalf("alpha %v: Items = %d, want 1", tc.alpha, len(list.Items))
		}
		item := list.Items[0]
		if !item.Position.AlmostEqual(tc.want, 1e-9) {
			t.Errorf("alpha %v: position = %v, want %v", tc.alpha, item.Position, tc.want)
		}
		if item.Texture != "tex.png" {
			t.Errorf("texture = %q, want tex.png", item.Texture)
		}
		if item.Rect != (image.Rectangle{}) {
			t.Errorf("rect = %v, want empty for a whole-texture sprite", item.Rect)
		}
		// The half-alpha color flows whole: its alpha channel is the
		// sprite's opacity, so the item carries the color as declared.
		if !item.FlipH || item.Color != color.Color(color.RGBA{R: 128, A: 64}) {
			t.Errorf("flip/color = %v/%v, want true/(255, 128-alpha)", item.FlipH, item.Color)
		}
	}
}

func TestCollectInterpolatesRotationAndScale(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	if _, err := w.NewEntity(
		spr.Sprite{Asset: "tex.png"},
		core.Transform{Rotation: 1, Scale: geom.Vector2{X: 3, Y: 5}},
		core.PrevTransform{Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn sprite: %v", err)
	}

	collector := NewCollector(w)
	for _, tc := range []struct {
		alpha    float64
		rotation float64
		scale    geom.Vector2
	}{
		{0, 0, geom.Vector2{X: 1, Y: 1}},
		{0.5, 0.5, geom.Vector2{X: 2, Y: 3}},
		{1, 1, geom.Vector2{X: 3, Y: 5}},
	} {
		list := collect(t, collector, drawContext(w), tc.alpha)
		if len(list.Items) != 1 {
			t.Fatalf("alpha %v: Items = %d, want 1", tc.alpha, len(list.Items))
		}
		item := list.Items[0]
		if item.Rotation != tc.rotation {
			t.Errorf("alpha %v: rotation = %v, want %v", tc.alpha, item.Rotation, tc.rotation)
		}
		if !item.Scale.AlmostEqual(tc.scale, 1e-9) {
			t.Errorf("alpha %v: scale = %v, want %v", tc.alpha, item.Scale, tc.scale)
		}
	}
}

func TestCollectInterpolatesRotationAcrossAngleBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prev    float64
		curr    float64
		wantMid float64
	}{
		{"negative angles", -math.Pi, -math.Pi / 6, -7 * math.Pi / 12},
		{"angles beyond two turns", 2*math.Pi + math.Pi/6, 2*math.Pi + math.Pi/3, 2*math.Pi + math.Pi/4},
		{"same angle across the wrap", -math.Pi / 4, 7 * math.Pi / 4, 3 * math.Pi / 4},
		{"same angle back across the wrap", 3 * math.Pi / 2, -math.Pi / 2, math.Pi / 2},
	} {
		w := newCollectorWorld(t)
		spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
		if _, err := w.NewEntity(
			spr.Sprite{Asset: "tex.png"},
			core.Transform{Rotation: tc.curr, Scale: unitScale},
			core.PrevTransform{Rotation: tc.prev, Scale: unitScale},
		); err != nil {
			t.Fatalf("%s: spawn sprite: %v", tc.name, err)
		}

		collector := NewCollector(w)
		for _, alpha := range []struct {
			alpha float64
			want  float64
		}{
			{0, tc.prev},
			{0.5, tc.wantMid},
			{1, tc.curr},
		} {
			list := collect(t, collector, drawContext(w), alpha.alpha)
			if len(list.Items) != 1 {
				t.Fatalf("%s: alpha %v: Items = %d, want 1", tc.name, alpha.alpha, len(list.Items))
			}
			got := list.Items[0].Rotation
			if math.Abs(got-alpha.want) > 1e-9 {
				t.Errorf("%s: alpha %v: rotation = %v, want %v", tc.name, alpha.alpha, got, alpha.want)
			}
		}
	}
}

func TestCollectInterpolatesNonUniformScale(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	if _, err := w.NewEntity(
		spr.Sprite{Asset: "tex.png"},
		core.Transform{Scale: geom.Vector2{X: 4, Y: 2}},
		core.PrevTransform{Scale: geom.Vector2{X: 2, Y: 10}},
	); err != nil {
		t.Fatalf("spawn sprite: %v", err)
	}

	collector := NewCollector(w)
	for _, tc := range []struct {
		alpha float64
		want  geom.Vector2
	}{
		{0, geom.Vector2{X: 2, Y: 10}},
		{0.25, geom.Vector2{X: 2.5, Y: 8}},
		{0.5, geom.Vector2{X: 3, Y: 6}},
		{1, geom.Vector2{X: 4, Y: 2}},
	} {
		list := collect(t, collector, drawContext(w), tc.alpha)
		if len(list.Items) != 1 {
			t.Fatalf("alpha %v: Items = %d, want 1", tc.alpha, len(list.Items))
		}
		if !list.Items[0].Scale.AlmostEqual(tc.want, 1e-9) {
			t.Errorf("alpha %v: scale = %v, want %v", tc.alpha, list.Items[0].Scale, tc.want)
		}
	}
}

func TestCollectCameraRotationInterpolatesNumerically(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prev    float64
		curr    float64
		wantMid float64
	}{
		{"negative angles", -math.Pi / 2, -math.Pi / 4, -3 * math.Pi / 8},
		{"same angle across the wrap", 3 * math.Pi / 2, -math.Pi / 2, math.Pi / 2},
	} {
		w := newCollectorWorld(t)
		if _, err := w.NewEntity(
			Camera{Primary: true},
			core.Transform{Rotation: tc.curr, Scale: unitScale},
			core.PrevTransform{Rotation: tc.prev, Scale: unitScale},
		); err != nil {
			t.Fatalf("%s: spawn camera: %v", tc.name, err)
		}

		collector := NewCollector(w)
		for _, alpha := range []struct {
			alpha float64
			want  float64
		}{
			{0, tc.prev},
			{0.5, tc.wantMid},
			{1, tc.curr},
		} {
			list := collect(t, collector, drawContext(w), alpha.alpha)
			if math.Abs(list.Camera.Rotation-alpha.want) > 1e-9 {
				t.Errorf("%s: alpha %v: camera rotation = %v, want %v", tc.name, alpha.alpha, list.Camera.Rotation, alpha.want)
			}
		}
	}
}

func TestCollectCullsByInterpolatedScale(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	// Static position at x=54: the 4x4 texture's scale-1 bounds
	// [52, 56] sit outside the viewport's +50 edge.
	if _, err := w.NewEntity(
		spr.Sprite{Asset: "tex.png"},
		core.Transform{Position: geom.Vector2{X: 54}},
		core.PrevTransform{Position: geom.Vector2{X: 54}, Scale: geom.Vector2{X: 10, Y: 10}},
	); err != nil {
		t.Fatalf("spawn sprite: %v", err)
	}

	collector := NewCollector(w)

	list := collect(t, collector, drawContext(w), 0.5)
	if len(list.Items) != 1 {
		t.Fatalf("alpha 0.5: Items = %d, want 1 (the interpolated scale keeps it)", len(list.Items))
	}
	if !list.Items[0].Scale.AlmostEqual(geom.Vector2{X: 5.5, Y: 5.5}, 1e-9) {
		t.Errorf("alpha 0.5: scale = %v, want 5.5", list.Items[0].Scale)
	}

	if list := collect(t, collector, drawContext(w), 1); len(list.Items) != 0 {
		t.Fatalf("alpha 1: Items = %d, want 0 (the current scale culls it)", len(list.Items))
	}
}

func TestCollectResolvesSheetSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	for name := range map[string]image.Rectangle{
		"player": image.Rect(0, 0, 2, 2),
		"enemy":  image.Rect(2, 2, 4, 4),
	} {
		if _, err := w.NewEntity(
			spr.Sprite{Asset: "sprites.json", Region: name},
			core.Transform{Position: geom.Vector2{}, Scale: unitScale},
		); err != nil {
			t.Fatalf("spawn atlas sprite %q: %v", name, err)
		}
		if _, err := w.NewEntity(
			spr.Sprite{Asset: "sprites.json", Region: name},
			core.Transform{Position: geom.Vector2{X: 60}, Scale: unitScale},
		); err != nil {
			t.Fatalf("spawn culled atlas sprite %q: %v", name, err)
		}
	}

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 2 {
		t.Fatalf("Items = %d, want 2 (one per region; the off-screen copies culled)", len(list.Items))
	}
	rects := map[image.Rectangle]bool{}
	for _, item := range list.Items {
		if item.Texture != "tex.png" {
			t.Errorf("texture = %q, want the atlas texture tex.png", item.Texture)
		}
		rects[item.Rect] = true
	}
	for name, rect := range map[string]image.Rectangle{
		"player": image.Rect(0, 0, 2, 2),
		"enemy":  image.Rect(2, 2, 4, 4),
	} {
		if !rects[rect] {
			t.Errorf("region %q: rect %v missing from the list", name, rect)
		}
	}
}

func TestCollectZeroScaleReadsAsUnit(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	// No explicit prev: the pair rule snapshots the zero scale as
	// unit at spawn, so both ends of the interpolation read unscaled.
	if _, err := w.NewEntity(
		spr.Sprite{Asset: "sprites.json", Region: "player"},
		core.Transform{Position: geom.Vector2{}},
	); err != nil {
		t.Fatalf("spawn zero-scale sprite: %v", err)
	}

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 1 {
		t.Fatalf("Items = %d, want 1", len(list.Items))
	}
	item := list.Items[0]
	if item.Scale != unitScale {
		t.Errorf("Scale = %v, want the zero scale to read as unit", item.Scale)
	}
	if item.Rect != image.Rect(0, 0, 2, 2) {
		t.Errorf("Rect = %v, want the player region", item.Rect)
	}
}

func TestCollectCullsAgainstViewport(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})           // dead center: kept
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 48}, geom.Vector2{X: 48}, spr.Sprite{}) // 4x4 bounds touch +50: kept
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 54}, geom.Vector2{X: 54}, spr.Sprite{}) // bounds [52, 56]: culled
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 60}, geom.Vector2{X: 60}, spr.Sprite{}) // far out: culled

	// Scale grows bounds: the same off-screen position survives at
	// 10x. The pair rule snapshots the scale with the spawn.
	if _, err := w.NewEntity(
		spr.Sprite{Asset: "tex.png"},
		core.Transform{Position: geom.Vector2{X: 60}, Scale: geom.Vector2{X: 10, Y: 10}},
	); err != nil {
		t.Fatalf("spawn scaled sprite: %v", err)
	}

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 3 {
		t.Fatalf("Items = %d, want 3 (center, touching, and scaled; two culled)", len(list.Items))
	}
	for _, item := range list.Items {
		if item.Position.X > 60 {
			t.Errorf("culled sprite leaked into the list at %v", item.Position)
		}
	}

	// Zoom halves the world-space viewport: the x=48 sprite, whose
	// bounds touch +50, now falls outside the +25 edge.
	zoomed := newCollectorWorld(t)
	spawnCamera(t, zoomed, geom.Vector2{}, geom.Vector2{}, 2, true)
	spawnTextureSprite(t, zoomed, "tex.png", geom.Vector2{X: 48}, geom.Vector2{X: 48}, spr.Sprite{})
	if list := collect(t, NewCollector(zoomed), drawContext(zoomed), 0); len(list.Items) != 0 {
		t.Errorf("Items = %d, want 0 under 2x zoom", len(list.Items))
	}
}

func TestCollectFailsFastNamingHandles(t *testing.T) {
	cases := []struct {
		name   string
		sprite func(t *testing.T, w *core.World)
		want   []string
	}{
		{"unloadable texture", func(t *testing.T, w *core.World) {
			spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})
		}, []string{"missing.png"}},
		{"missing sheet", func(t *testing.T, w *core.World) {
			if _, err := w.NewEntity(
				spr.Sprite{Asset: "ghost.json", Region: "player"},
				core.Transform{Position: geom.Vector2{}, Scale: unitScale},
				core.PrevTransform{},
			); err != nil {
				t.Fatalf("spawn sprite: %v", err)
			}
		}, []string{"ghost"}},
		{"unknown region", func(t *testing.T, w *core.World) {
			if _, err := w.NewEntity(
				spr.Sprite{Asset: "sprites.json", Region: "boss"},
				core.Transform{Position: geom.Vector2{}, Scale: unitScale},
				core.PrevTransform{},
			); err != nil {
				t.Fatalf("spawn sprite: %v", err)
			}
		}, []string{"boss", "tex.png"}},
		{"out-of-bounds sheet", func(t *testing.T, w *core.World) {
			spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
				spr.Sprite{Asset: "oversized.json", Region: "player"}, core.Transform{})
		}, []string{"player", "tex.png", "outside"}},
		{"missing sheet texture", func(t *testing.T, w *core.World) {
			spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
				spr.Sprite{Asset: "missing-texture.json", Region: "player"}, core.Transform{})
		}, []string{"player", "missing.png"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newCollectorWorld(t)
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			tc.sprite(t, w)

			list, err := NewCollector(w).Collect(drawContext(w), 0)
			if err == nil {
				t.Fatalf("Collect should fail fast on %s", tc.name)
			}
			for _, handle := range tc.want {
				if !strings.Contains(err.Error(), handle) {
					t.Errorf("error %q should name %q", err, handle)
				}
			}
			if len(list.Items) != 0 {
				t.Errorf("failed Collect returned %d items, want 0", len(list.Items))
			}
		})
	}
}

func TestCollectSortsLayerThenOrderThenY(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	at := func(x, y float64, sprite spr.Sprite) {
		pos := geom.Vector2{X: x, Y: y}
		spawnTextureSprite(t, w, "tex.png", pos, pos, sprite)
	}

	at(1, -10, spr.Sprite{Style: spr.Style{SortOrder: 1}})
	at(2, 0, spr.Sprite{Style: spr.Style{Layer: 2}})
	at(3, -10, spr.Sprite{Style: spr.Style{Layer: 1}})
	at(4, -10, spr.Sprite{})
	at(5, 10, spr.Sprite{Style: spr.Style{SortOrder: -1}})
	at(6, 10, spr.Sprite{})
	at(7, 0, spr.Sprite{Style: spr.Style{Layer: 2}})

	want := []float64{5, 4, 6, 1, 3, 2, 7}

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != len(want) {
		t.Fatalf("Items = %d, want %d", len(list.Items), len(want))
	}
	for i, item := range list.Items {
		if item.Position.X != want[i] {
			t.Errorf("Items[%d].Position.X = %v, want %v", i, item.Position.X, want[i])
		}
	}
}

func TestCollectReusesBuffers(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	firstSprite := spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})
	spawnTextureSprite(t, w, "tex2.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})

	collector := NewCollector(w)
	first := collect(t, collector, drawContext(w), 0)
	if len(first.Items) != 2 || first.Items[0].Texture != "tex.png" {
		t.Fatalf("first Collect: Items = %v, want [tex.png tex2.png]", first.Items)
	}

	if err := w.DestroyEntity(firstSprite); err != nil {
		t.Fatalf("destroy first sprite: %v", err)
	}
	second := collect(t, collector, drawContext(w), 0)
	if len(second.Items) != 1 {
		t.Fatalf("second Collect: Items = %d, want 1 with no stale entry", len(second.Items))
	}
	if second.Items[0].Texture != "tex2.png" {
		t.Errorf("second Collect: Items[0].Texture = %q, want tex2.png", second.Items[0].Texture)
	}
	// The retained list views the same buffer: the second collect
	// overwrote its first item.
	if first.Items[0].Texture != "tex2.png" {
		t.Errorf("retained list Items[0].Texture = %q, want tex2.png — the buffer was reused underneath it",
			first.Items[0].Texture)
	}
}

func TestCollectStagesShapeSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnSprite(t, w, geom.Vector2{}, geom.Vector2{X: 10, Y: 0},
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 20}, Style: spr.Style{Layer: 1}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
		spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}, Style: spr.Style{Color: &color.RGBA{R: 255, A: 255}}},
		core.Transform{Rotation: 0.5, Scale: geom.Vector2{X: 2, Y: 2}})
	spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
		spr.Primitive{Shape: spr.ShapeLine, From: geom.Vector2{X: -10, Y: 0}, To: geom.Vector2{X: 30, Y: 0}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 6, Y: 6}, Style: spr.Style{Outline: true, StrokeWidth: 2}},
		core.Transform{})

	list := collect(t, NewCollector(w), drawContext(w), 0.5)
	if len(list.Items) != 4 {
		t.Fatalf("Items = %d, want 4", len(list.Items))
	}

	var gotRect, gotCircle, gotLine, gotOutlinedRect bool
	for _, item := range list.Items {
		switch shape := item.Shape.(type) {
		case spr.RectShape:
			if shape.Size == (geom.Vector2{X: 10, Y: 20}) {
				if !item.Position.AlmostEqual(geom.Vector2{X: 5, Y: 0}, 1e-9) {
					t.Errorf("rect position = %v, want interpolated (5, 0)", item.Position)
				}
				if item.Color != color.Black {
					t.Errorf("rect color = %v, want the black default for a nil color", item.Color)
				}
				if item.Outline || item.StrokeWidth != 0 {
					t.Errorf("filled rect outline/width = %v/%v, want false/0",
						item.Outline, item.StrokeWidth)
				}
				gotRect = true
			} else {
				if !item.Outline || item.StrokeWidth != 2 {
					t.Errorf("outlined rect outline/width = %v/%v, want true/2",
						item.Outline, item.StrokeWidth)
				}
				gotOutlinedRect = true
			}
		case spr.CircleShape:
			if shape.Radii != (geom.Vector2{X: 5, Y: 5}) {
				t.Errorf("circle shape = %v, want radii (5, 5)", item.Shape)
			}
			if item.Rotation != 0.5 || item.Scale != (geom.Vector2{X: 2, Y: 2}) {
				t.Errorf("circle rotation/scale = %v/%v, want 0.5/(2, 2) (unchanged between ticks)", item.Rotation, item.Scale)
			}
			if item.Color != color.Color(color.RGBA{R: 255, A: 255}) {
				t.Errorf("circle color = %v, want the declared color", item.Color)
			}
			gotCircle = true
		case spr.LineShape:
			if shape.From != (geom.Vector2{X: -10, Y: 0}) || shape.To != (geom.Vector2{X: 30, Y: 0}) {
				t.Errorf("line shape = %v, want From (-10, 0) To (30, 0)", item.Shape)
			}
			gotLine = true
		default:
			t.Errorf("unexpected shape %T in the draw list", shape)
		}
	}
	if !gotRect || !gotCircle || !gotLine || !gotOutlinedRect {
		t.Fatalf("collected shapes: rect=%v circle=%v line=%v outlined=%v, want all four",
			gotRect, gotCircle, gotLine, gotOutlinedRect)
	}
}

func TestCollectOrdersShapesWithSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{Y: 10}, geom.Vector2{Y: 10}, spr.Sprite{Style: spr.Style{Layer: 1}})
	spawnSprite(t, w, geom.Vector2{Y: -10}, geom.Vector2{Y: -10},
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 4}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{Y: -10}, geom.Vector2{Y: -10},
		spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 2, Y: 2}, Style: spr.Style{Layer: 1}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{Y: 10}, geom.Vector2{Y: 10},
		spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{X: 8, Y: 0}, Style: spr.Style{Layer: 1}}, core.Transform{})

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 4 {
		t.Fatalf("Items = %d, want 4", len(list.Items))
	}

	want := []struct {
		texture asset.ID
		shape   spr.DrawShape
	}{
		{"", spr.RectShape{Size: geom.Vector2{X: 4, Y: 4}}},
		{"", spr.CircleShape{Radii: geom.Vector2{X: 2, Y: 2}}},
		{"tex.png", nil},
		{"", spr.LineShape{To: geom.Vector2{X: 8, Y: 0}}},
	}
	for i := range list.Items {
		if list.Items[i].Texture != want[i].texture {
			t.Errorf("Items[%d].Texture = %q, want %q", i, list.Items[i].Texture, want[i].texture)
		}
		switch {
		case want[i].shape == nil:
			if list.Items[i].Shape != nil {
				t.Errorf("Items[%d].Shape = %T, want a texture sprite", i, list.Items[i].Shape)
			}
		default:
			if list.Items[i].Shape != want[i].shape {
				t.Errorf("Items[%d].Shape = %v, want %v", i, list.Items[i].Shape, want[i].shape)
			}
		}
	}
}

func TestCollectCullsShapes(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	// Viewport is [-50, 50]^2 at zoom 1.
	spawnSprite(t, w, geom.Vector2{X: 40}, geom.Vector2{X: 40},
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{X: 56}, geom.Vector2{X: 56},
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{X: 61}, geom.Vector2{X: 61},
		spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
	// The offset proof: position (75, 0) is outside the viewport, but
	// the segment reaches back to (45, 0) - its bounds are [45, 75],
	// which overlaps. Centered bounds [60, 90] would have culled it.
	spawnSprite(t, w, geom.Vector2{X: 75}, geom.Vector2{X: 75},
		spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{X: -30, Y: 0}}, core.Transform{})
	spawnSprite(t, w, geom.Vector2{X: 60}, geom.Vector2{X: 60},
		spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{X: 20, Y: 0}}, core.Transform{})
	// The inverse proof: this line's position is dead center, but its
	// segment spans (80, 0) to (110, 0), entirely outside - the AABB
	// bounds cull it despite the visible position.
	spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
		spr.Primitive{Shape: spr.ShapeLine, From: geom.Vector2{X: 80, Y: 0}, To: geom.Vector2{X: 110, Y: 0}},
		core.Transform{})

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 2 {
		t.Fatalf("Items = %d, want 2 (the near rect and the reaching line)", len(list.Items))
	}
	var gotRect, gotLine bool
	for _, item := range list.Items {
		switch item.Shape.(type) {
		case spr.RectShape:
			gotRect = true
		case spr.LineShape:
			gotLine = true
		}
	}
	if !gotRect || !gotLine {
		t.Fatalf("survivors = rect:%v line:%v, want both", gotRect, gotLine)
	}
}

func TestCollectKeepsRotatedAndStrokedGeometry(t *testing.T) {
	for _, test := range []struct {
		name     string
		visual   any
		position geom.Vector2
		rotation float64
		scale    geom.Vector2
		want     int
	}{
		{"rotated rect",
			spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 40}}, geom.Vector2{X: 60}, math.Pi / 2, unitScale, 1},
		{"rotated ellipse", spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 2, Y: 20}}, geom.Vector2{X: 60}, math.Pi / 2, unitScale, 1},
		{"rotated offset line", spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{Y: 20}, Style: spr.Style{StrokeWidth: 2}}, geom.Vector2{X: 60}, math.Pi / 2, unitScale, 1},
		{"outlined rect", spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}, Style: spr.Style{Outline: true, StrokeWidth: 4}}, geom.Vector2{X: 56}, 0, unitScale, 1},
		{"thick line", spr.Primitive{Shape: spr.ShapeLine, From: geom.Vector2{Y: -5}, To: geom.Vector2{Y: 5}, Style: spr.Style{StrokeWidth: 6}}, geom.Vector2{X: 52}, 0, unitScale, 1},
		{"reflected texture", spr.Sprite{Asset: "tex.png"}, geom.Vector2{X: 60}, math.Pi / 2, geom.Vector2{X: -1, Y: 10}, 1},
		{"rotated sheet", spr.Sprite{Asset: "sprites.json", Region: "player"}, geom.Vector2{X: 60}, math.Pi / 2, geom.Vector2{X: 1, Y: 20}, 1},
		{"outside", spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 40}}, geom.Vector2{X: 80}, math.Pi / 2, unitScale, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := newCollectorWorld(t)
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			spawnSprite(t, w, test.position, test.position, test.visual,
				core.Transform{Rotation: test.rotation, Scale: test.scale})
			if got := collect(t, NewCollector(w), drawContext(w), 1); len(got.Items) != test.want {
				t.Fatalf("collected %d items, want %d", len(got.Items), test.want)
			}
		})
	}
}

func TestCollectCullsUsingInterpolatedDrawableRotation(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	position := geom.Vector2{X: 60}
	entity := spawnSprite(t, w, position, position,
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 40}},
		core.Transform{Rotation: math.Pi, Scale: unitScale})
	if err := entity.SetComponent(w, core.PrevTransform{Position: position, Scale: unitScale}); err != nil {
		t.Fatal(err)
	}
	collector := NewCollector(w)
	for _, test := range []struct {
		alpha float64
		want  int
	}{{0, 0}, {0.5, 1}, {1, 0}} {
		if got := collect(t, collector, drawContext(w), test.alpha); len(got.Items) != test.want {
			t.Fatalf("alpha %v: collected %d, want %d", test.alpha, len(got.Items), test.want)
		}
	}
}

func TestEngineCameraPreference(t *testing.T) {
	w := newCollectorWorld(t)
	ctx := drawContext(w)

	if list := collect(t, NewCollector(w), ctx, 0); len(list.Items) != 0 || list.Camera != (CameraView{}) {
		t.Fatalf("no-camera list = %+v, want empty", list)
	}

	if _, err := SpawnEngineCamera(w); err != nil {
		t.Fatalf("spawn engine camera: %v", err)
	}
	list := collect(t, NewCollector(w), ctx, 0)
	if list.Camera.Position != (geom.Vector2{}) || list.Camera.Zoom != 1 {
		t.Fatalf("engine camera view = %+v, want origin zoom 1", list.Camera)
	}

	spawnCamera(t, w, geom.Vector2{X: 30, Y: 30}, geom.Vector2{X: 30, Y: 30}, 2, true)
	list = collect(t, NewCollector(w), ctx, 0)
	if list.Camera.Position != (geom.Vector2{X: 30, Y: 30}) || list.Camera.Zoom != 2 {
		t.Fatalf("user camera should win: view = %+v, want (30, 30) zoom 2", list.Camera)
	}
	win, err := w.Resource[Window]()
	if err != nil {
		t.Fatal(err)
	}
	vp := win.Viewport()
	view, ok, err := NewCameraResolver(w).Resolve(0, vp.LogicalWidth, vp.LogicalHeight)
	if err != nil || !ok || view != list.Camera {
		t.Fatalf("resolved camera = %+v, %v, %v; want collected view %+v", view, ok, err, list.Camera)
	}
}

func TestCollectTextItem(t *testing.T) {
	w := newFontWorld(t)
	spawnCamera(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50}, 1, true)
	spawnTextSprite(t, w, "score: 1", nil, geom.Vector2{X: 60, Y: 50})

	list := collect(t, NewCollector(w), drawContext(w), 0.5)
	if len(list.Items) != 1 {
		t.Fatalf("collected %d items, want 1", len(list.Items))
	}
	item := list.Items[0]
	if item.Text != "score: 1" || item.Font != "fonts/go.ttf" || item.TextSize != 16 {
		t.Errorf("item text fields = %q %q %v", item.Text, item.Font, item.TextSize)
	}
	if item.TextWidth <= 0 || item.TextHeight <= 0 {
		t.Errorf("measured size = %v x %v, want positive", item.TextWidth, item.TextHeight)
	}
	if item.Position != (geom.Vector2{X: 60, Y: 50}) {
		t.Errorf("position = %v, want (60, 50)", item.Position)
	}
	if item.Color != color.White {
		t.Errorf("text color = %v, want the white default", item.Color)
	}
}

func TestCollectTextKeepsSpriteColor(t *testing.T) {
	w := newFontWorld(t)
	spawnCamera(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50}, 1, true)
	spawnTextSprite(t, w, "tinted", &color.RGBA{R: 200, G: 40, B: 40, A: 255}, geom.Vector2{X: 50, Y: 50})

	list := collect(t, NewCollector(w), drawContext(w), 0.5)
	if len(list.Items) != 1 {
		t.Fatalf("collected %d items, want 1", len(list.Items))
	}
	tinted, ok := list.Items[0].Color.(color.RGBA)
	if !ok || tinted != (color.RGBA{R: 200, G: 40, B: 40, A: 255}) {
		t.Errorf("text color = %v, want the sprite's own color", list.Items[0].Color)
	}
}

func TestCollectMissingFontFails(t *testing.T) {
	w := newFontWorld(t)
	spawnCamera(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50}, 1, true)
	if _, err := w.NewEntity(
		spr.Text{Asset: "fonts/ghost.ttf", Text: "haunted", TextSize: 16},
		core.Transform{Position: geom.Vector2{X: 50, Y: 50}, Scale: geom.Vector2{X: 1, Y: 1}},
	); err != nil {
		t.Fatal(err)
	}

	_, err := NewCollector(w).Collect(drawContext(w), 0.5)
	if err == nil || !strings.Contains(err.Error(), "fonts/ghost.ttf") {
		t.Errorf("missing font error = %v, want an error naming the font", err)
	}
}

func TestCollectDrawableAssetRequirements(t *testing.T) {
	for _, tc := range []struct {
		name  string
		world func(*testing.T) *core.World
		spawn func(*testing.T, *core.World)
	}{
		{"shape", func(*testing.T) *core.World { return core.NewWorld() }, func(t *testing.T, w *core.World) {
			spawnSprite(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50},
				spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
		}},
		{"texture", newCollectorWorld, func(t *testing.T, w *core.World) {
			spawnSprite(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50},
				spr.Sprite{Asset: "tex.png"}, core.Transform{})
		}},
		{"text", newFontWorld, func(t *testing.T, w *core.World) {
			spawnSprite(t, w, geom.Vector2{X: 50, Y: 50}, geom.Vector2{X: 50, Y: 50},
				spr.Text{Asset: "fonts/go.ttf", Text: "player", TextSize: 16}, core.Transform{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := tc.world(t)
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			tc.spawn(t, w)
			if list := collect(t, NewCollector(w), drawContext(w), 1); len(list.Items) != 1 {
				t.Fatalf("items = %d, want the %s drawable", len(list.Items), tc.name)
			}
		})
	}
}

func TestCollectSheetLoaderIsResolvedLazily(t *testing.T) {
	w := core.NewWorld()
	spawnSprite(t, w, geom.Vector2{}, geom.Vector2{},
		spr.Sprite{Asset: "sprites.json", Region: "player"}, core.Transform{})
	collector := NewCollector(w)
	if list := collect(t, collector, drawContext(w), 1); len(list.Items) != 0 {
		t.Fatal("no-camera collection should not resolve the sheet")
	}
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	if _, err := collector.Collect(drawContext(w), 1); err == nil || !strings.Contains(err.Error(), "asset loader") {
		t.Fatalf("missing loader error = %v", err)
	}
	loader := newCollectorWorld(t).MustResource[*asset.Loader]()
	if err := w.Provide(func(*core.World) (*asset.Loader, error) { return loader, nil }); err != nil {
		t.Fatal(err)
	}
	list := collect(t, collector, drawContext(w), 1)
	if len(list.Items) != 1 || list.Items[0].Texture != "tex.png" || list.Items[0].Rect != image.Rect(0, 0, 2, 2) {
		t.Fatalf("sheet world produced %+v", list.Items)
	}
}

func TestCollectMemoryClipAndFileShareDrawSource(t *testing.T) {
	// Sprite components reference sheets by file ID; memory sheets are
	// runtime-only and reach the renderer through animation clips.
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	sheet, err := asset.NewSheet("./tex.png", map[string]asset.SheetRegion{"player": {W: 2, H: 2}})
	if err != nil {
		t.Fatal(err)
	}
	memoryClip, err := animation.NewSpriteClip(asset.MemorySheet(sheet), []string{"player"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	fileClip, err := animation.NewSpriteClip(asset.FileSheet("sprites.json"), []string{"player"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, clip := range []animation.SpriteClip{memoryClip, fileClip} {
		animated, err := animation.NewAnimatedSprite(clip)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.NewEntity(animated, core.Transform{}); err != nil {
			t.Fatal(err)
		}
	}
	list := collect(t, NewCollector(w), drawContext(w), 1)
	if len(list.Items) != 2 || list.Items[0].Texture != list.Items[1].Texture || list.Items[0].Rect != list.Items[1].Rect {
		t.Fatalf("file and memory sources disagree: %+v", list.Items)
	}
}

func TestCollectTextureWithoutAssetLoaderErrors(t *testing.T) {
	w := core.NewWorld()
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	if _, err := w.NewEntity(
		core.Transform{Position: geom.Vector2{X: 50, Y: 50}},
		spr.Sprite{Asset: "tex.png"},
	); err != nil {
		t.Fatal(err)
	}

	_, err := NewCollector(w).Collect(drawContext(w), 1)
	if err == nil {
		t.Fatal("a texture drawable without an asset loader should error")
	}
	if !strings.Contains(err.Error(), "asset loader") {
		t.Errorf("error = %v, want it to name the asset loader", err)
	}
}

func TestCollectAnchorShiftsViewport(t *testing.T) {
	w := newCollectorWorld(t)
	if _, err := w.NewEntity(
		Camera{Primary: true, Anchor: geom.Vector2{X: -0.5, Y: -0.5}},
		core.Transform{Scale: unitScale},
		core.PrevTransform{Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 90, Y: 90}, geom.Vector2{X: 90, Y: 90}, spr.Sprite{})
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: -10, Y: 50}, geom.Vector2{X: -10, Y: 50}, spr.Sprite{})
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 150, Y: 50}, geom.Vector2{X: 150, Y: 50}, spr.Sprite{})

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 1 {
		t.Fatalf("Items = %d, want 1 (only the in-view sprite)", len(list.Items))
	}
	if list.Items[0].Position.X != 90 {
		t.Errorf("kept sprite = %v, want the one at (90, 90)", list.Items[0].Position)
	}
}

func TestCollectRotationInterpolatesAndCullsConservatively(t *testing.T) {
	w := newCollectorWorld(t)
	if _, err := w.NewEntity(
		Camera{Primary: true},
		core.Transform{Rotation: math.Pi / 2, Scale: unitScale},
		core.PrevTransform{Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	// Alpha 0.5 resolves rotation to pi/4; the circumscribed half extents
	// are 50*(cos45+sin45) ~ 70.7, so (65, 65) culls-in conservatively
	// even though the exact rotated view excludes it.
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 65, Y: 65}, geom.Vector2{X: 65, Y: 65}, spr.Sprite{})
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 90, Y: 90}, geom.Vector2{X: 90, Y: 90}, spr.Sprite{})

	list := collect(t, NewCollector(w), drawContext(w), 0.5)
	if want := math.Pi / 4; math.Abs(list.Camera.Rotation-want) > 1e-9 {
		t.Errorf("Camera.Rotation = %v, want %v", list.Camera.Rotation, want)
	}
	if len(list.Items) != 1 || list.Items[0].Position.X != 65 {
		t.Fatalf("Items = %v, want only the conservatively-visible (65, 65)", list.Items)
	}
}

func TestCollectClampsToBounds(t *testing.T) {
	bounds := geom.RectFromMinMax(geom.Vector2{}, geom.Vector2{X: 200, Y: 100})

	w := newCollectorWorld(t)
	if _, err := w.NewEntity(
		Camera{Primary: true, Bounds: bounds},
		core.Transform{Position: geom.Vector2{X: 10, Y: 50}, Scale: unitScale},
		core.PrevTransform{Position: geom.Vector2{X: 10, Y: 50}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	list := collect(t, NewCollector(w), drawContext(w), 1)
	// A 100x100 view in 200x100 bounds: x clamps into [50, 150], y pins
	// to the only fitting value.
	if want := (geom.Vector2{X: 50, Y: 50}); list.Camera.Position != want {
		t.Errorf("clamped position = %v, want %v", list.Camera.Position, want)
	}

	// Zoom 0.5 doubles the view to 200x200: neither axis fits, so the
	// position pins to the bounds center.
	zoomed := newCollectorWorld(t)
	if _, err := zoomed.NewEntity(
		Camera{Primary: true, Bounds: bounds},
		core.Transform{Position: geom.Vector2{X: 10, Y: 50}, Scale: geom.Vector2{X: 0.5, Y: 0.5}},
		core.PrevTransform{Position: geom.Vector2{X: 10, Y: 50}, Scale: geom.Vector2{X: 0.5, Y: 0.5}},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	list = collect(t, NewCollector(zoomed), drawContext(zoomed), 1)
	if want := (geom.Vector2{X: 100, Y: 50}); list.Camera.Position != want {
		t.Errorf("pinned position = %v, want %v", list.Camera.Position, want)
	}
}

func TestCollectClampsRotatedViewToBounds(t *testing.T) {
	w := newCollectorWorld(t)
	if _, err := w.NewEntity(
		Camera{Primary: true, Bounds: geom.RectFromMinMax(geom.Vector2{}, geom.Vector2{X: 200, Y: 100})},
		core.Transform{Rotation: math.Pi / 4, Scale: unitScale},
		core.PrevTransform{Rotation: math.Pi / 4, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}

	list := collect(t, NewCollector(w), drawContext(w), 1)
	// At 45 degrees the 100x100 view occupies ~141 world units per
	// axis: x clamps into the shrunk window while y, which cannot fit
	// the 100-unit bounds, pins to the bounds center.
	want := geom.Vector2{X: 50 * (math.Cos(math.Pi/4) + math.Sin(math.Pi/4)), Y: 50}
	if !list.Camera.Position.AlmostEqual(want, 1e-9) {
		t.Errorf("rotated clamped position = %v, want %v", list.Camera.Position, want)
	}
}

func TestCollectRejectsInvalidCameraScale(t *testing.T) {
	for _, tc := range []struct {
		name  string
		scale geom.Vector2
	}{
		{"negative zoom", geom.Vector2{X: -1, Y: -1}},
		{"non-uniform zoom", geom.Vector2{X: 1, Y: 2}},
		{"infinite zoom", geom.Vector2{X: math.Inf(1), Y: math.Inf(1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newCollectorWorld(t)
			if _, err := w.NewEntity(
				Camera{Primary: true},
				core.Transform{Scale: tc.scale},
				core.PrevTransform{Scale: tc.scale},
			); err != nil {
				t.Fatalf("spawn camera: %v", err)
			}

			_, err := NewCollector(w).Collect(drawContext(w), 0)
			if err == nil {
				t.Fatalf("Collect should reject %s", tc.name)
			}
			if !strings.Contains(err.Error(), "camera zoom") {
				t.Errorf("error %q should name the camera zoom", err)
			}
		})
	}
}

func TestCollectPublishesViewResource(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{X: 10, Y: 10}, geom.Vector2{X: 20, Y: 20}, 1, true)

	collector := NewCollector(w)
	collect(t, collector, drawContext(w), 0)
	view, err := w.Resource[*CameraView]()
	if err != nil {
		t.Fatalf("resolve view resource: %v", err)
	}
	if want := (geom.Vector2{X: 10, Y: 10}); view.Position != want {
		t.Errorf("resource position = %v, want %v", view.Position, want)
	}

	collect(t, collector, drawContext(w), 1)
	if want := (geom.Vector2{X: 20, Y: 20}); view.Position != want {
		t.Errorf("resource position after update = %v, want %v", view.Position, want)
	}

	// A second collector adopts the same resource instead of failing.
	collect(t, NewCollector(w), drawContext(w), 1)
}

func TestCollectFailurePublishesNoView(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{X: 10, Y: 0}, 1, true)
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})
	collector := NewCollector(w)
	collect(t, collector, drawContext(w), 0)
	view, err := w.Resource[*CameraView]()
	if err != nil {
		t.Fatalf("resolve view resource: %v", err)
	}
	if want := (geom.Vector2{}); view.Position != want {
		t.Fatalf("first view position = %v, want %v", view.Position, want)
	}

	spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})
	if _, err := collector.Collect(drawContext(w), 1); err == nil {
		t.Fatal("second Collect should fail on the missing texture")
	}
	if view.Position != (geom.Vector2{}) {
		t.Errorf("failed Collect published the failed frame's view %v; want it to keep the drawn frame's",
			view.Position)
	}

	broken := newCollectorWorld(t)
	spawnCamera(t, broken, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, broken, "missing.png", geom.Vector2{}, geom.Vector2{}, spr.Sprite{})
	if _, err := NewCollector(broken).Collect(drawContext(broken), 0); err == nil {
		t.Fatal("broken Collect should fail")
	}
	if _, err := broken.Resource[*CameraView](); !errors.Is(err, core.ErrResourceNotRegistered) {
		t.Errorf("failed first Collect registered a view: %v, want ErrResourceNotRegistered", err)
	}
}

func TestCollectGathersAnimatedSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)

	clip, err := animation.NewSpriteClip(asset.FileSheet("sprites.json"), []string{"player", "enemy"}, 10)
	if err != nil {
		t.Fatalf("clip: %v", err)
	}
	animated, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		t.Fatalf("animated sprite: %v", err)
	}
	animated.Style.FlipH = true
	animated.Style.Color = &color.RGBA{R: 255, A: 255}
	if _, err := w.NewEntity(
		animated,
		core.Transform{Position: geom.Vector2{}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn animated sprite: %v", err)
	}

	// Hidden animated sprites stay excluded, like hidden sprites.
	hidden, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		t.Fatalf("hidden animated sprite: %v", err)
	}
	hidden.Style.Hidden = true
	if _, err := w.NewEntity(
		hidden,
		core.Transform{Position: geom.Vector2{}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn hidden animated sprite: %v", err)
	}

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 1 {
		t.Fatalf("Items = %d, want 1 (current frame plus a hidden sprite)", len(list.Items))
	}
	item := list.Items[0]
	if item.Texture != "tex.png" || item.Rect != (image.Rect(0, 0, 2, 2)) {
		t.Errorf("texture/rect = %q/%v, want tex.png frame zero of the player region", item.Texture, item.Rect)
	}
	if !item.FlipH || item.Color != color.Color(color.RGBA{R: 255, A: 255}) {
		t.Errorf("flip/color = %v/%v, want the animated sprite's own style", item.FlipH, item.Color)
	}
}

func TestCollectSkipsCliplessAnimatedSprite(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)

	if _, err := w.NewEntity(
		animation.AnimatedSprite{},
		core.Transform{Position: geom.Vector2{}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn clipless animated sprite: %v", err)
	}

	list, err := NewCollector(w).Collect(drawContext(w), 0)
	if err != nil {
		t.Fatalf("Collect = %v, want no error for a clipless sprite", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("Items = %d, want 0", len(list.Items))
	}
}

func TestCollectDrawsPausedAnimatedSpriteFrameZero(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)

	clip, err := animation.NewSpriteClip(asset.FileSheet("sprites.json"), []string{"player", "enemy"}, 10)
	if err != nil {
		t.Fatalf("clip: %v", err)
	}
	paused, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		t.Fatalf("animated sprite: %v", err)
	}
	paused.Paused = true
	if _, err := w.NewEntity(
		paused,
		core.Transform{Position: geom.Vector2{}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn paused animated sprite: %v", err)
	}

	list := collect(t, NewCollector(w), drawContext(w), 0)
	if len(list.Items) != 1 {
		t.Fatalf("Items = %d, want 1", len(list.Items))
	}
	if list.Items[0].Texture != "tex.png" || list.Items[0].Rect != (image.Rect(0, 0, 2, 2)) {
		t.Errorf("texture/rect = %q/%v, want frame zero drawn while paused", list.Items[0].Texture, list.Items[0].Rect)
	}
}

func TestCollectStagesSpriteAndAnimatedSpriteTogether(t *testing.T) {
	// The union-era conflict error is gone: sprite.Sprite and the
	// animated sprite stage as independent visuals, the static image
	// first and the animated frame after it.
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)

	clip, err := animation.NewSpriteClip(asset.FileSheet("sprites.json"), []string{"player"}, 10)
	if err != nil {
		t.Fatalf("clip: %v", err)
	}
	animated, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		t.Fatalf("animated sprite: %v", err)
	}
	if _, err := w.NewEntity(
		animated,
		spr.Sprite{Asset: "tex.png"},
		core.Transform{Position: geom.Vector2{}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn co-staged components: %v", err)
	}

	list, err := NewCollector(w).Collect(drawContext(w), 0)
	if err != nil {
		t.Fatalf("Collect = %v, want both visuals staged", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("Items = %d, want 2 (static image, animated frame)", len(list.Items))
	}
	if list.Items[0].Texture != "tex.png" {
		t.Errorf("first item texture = %q, want the static image first", list.Items[0].Texture)
	}
}

func TestCollectSurfacesInvalidAnimatedSpriteConfig(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)

	clip, err := animation.NewSpriteClip(asset.FileSheet("sprites.json"), []string{"player", "enemy"}, 10)
	if err != nil {
		t.Fatalf("clip: %v", err)
	}
	animated, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		t.Fatalf("animated sprite: %v", err)
	}
	if _, err := w.NewEntity(
		animated,
		core.Transform{Position: geom.Vector2{}, Scale: unitScale},
	); err != nil {
		t.Fatalf("spawn animated sprite: %v", err)
	}

	// The write path a system uses: read through the entry, mutate, write
	// back. Entry.SetComponent skips component validation.
	for e := range core.NewQuery(w).With(animation.AnimatedSprite{}).Execute() {
		a, _ := e.Component[animation.AnimatedSprite]()
		a.PlaybackMultiplier = -1
		e.SetComponent(a)
	}
	_, err = NewCollector(w).Collect(drawContext(w), 0)
	if err == nil || !strings.Contains(err.Error(), "playback multiplier") {
		t.Fatalf("Collect error = %v, want the invalid playback multiplier surfaced", err)
	}
}

// TestCollectStagesEveryVisualComponentClass is the composed-sprite
// experiment's probe: one entity carries Sprite, Primitive, and Text
// together, and each contributes its own item in the documented class
// order. The tagged-union design made this entity impossible.
func newEveryVisualWorld(t *testing.T) *core.World {
	t.Helper()
	encode := func() []byte {
		var buf bytes.Buffer
		if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
			t.Fatalf("encode test png: %v", err)
		}
		return buf.Bytes()
	}
	files := fstest.MapFS{
		"tex.png":      &fstest.MapFile{Data: encode()},
		"sprites.json": &fstest.MapFile{Data: []byte(`{"texture":"./tex.png","regions":{"player":{"x":0,"y":0,"w":2,"h":2}}}`)},
		"fonts/go.ttf": &fstest.MapFile{Data: goregular.TTF},
	}
	w := core.NewWorld()
	if err := w.Provide(func(*core.World) (*asset.Loader, error) {
		return asset.NewLoader(files, asset.LoaderOptions{})
	}); err != nil {
		t.Fatalf("provide loader: %v", err)
	}
	return w
}

func TestCollectStagesEveryVisualComponentClass(t *testing.T) {
	w := newEveryVisualWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	if _, err := w.NewEntity(
		core.Transform{Position: geom.Vector2{X: 20, Y: 20}, Scale: geom.Vector2{X: 1, Y: 1}},
		spr.Sprite{Asset: "tex.png"},
		spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 4}, Style: spr.Style{Layer: 1}},
		spr.Text{Asset: "fonts/go.ttf", Text: "badge", TextSize: 8, Style: spr.Style{Layer: 2}},
	); err != nil {
		t.Fatalf("multi-visual entity: %v", err)
	}

	list := collect(t, NewCollector(w), drawContext(w), 1)
	if len(list.Items) != 3 {
		t.Fatalf("Items = %d, want 3 (image, primitive, text)", len(list.Items))
	}
	if list.Items[0].Texture != "tex.png" {
		t.Errorf("first item texture = %q, want the image class first", list.Items[0].Texture)
	}
	if _, ok := list.Items[1].Shape.(spr.RectShape); !ok {
		t.Errorf("second item = %T, want the primitive class second", list.Items[1].Shape)
	}
	if list.Items[2].Text != "badge" {
		t.Errorf("third item text = %q, want the text class third", list.Items[2].Text)
	}
}
