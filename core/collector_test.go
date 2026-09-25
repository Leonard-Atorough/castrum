package core

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

var unitScale = geom.Vector2{X: 1, Y: 1}

// newCollectorWorld wires a world the way a runner wires it: an asset
// server over a filesystem holding two 4x4 textures and a two-region
// atlas sidecar over the first, both registered.
func newCollectorWorld(t *testing.T) *World {
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
		"atlas.json": &fstest.MapFile{Data: []byte(
			`{"regions":[{"name":"player","x":0,"y":0,"w":2,"h":2},{"name":"enemy","x":2,"y":2,"w":2,"h":2}]}`)},
	}
	w := NewWorld()
	if err := w.Provide(func(*World) (*asset.Server, error) {
		return asset.New(files), nil
	}); err != nil {
		t.Fatalf("provide asset server: %v", err)
	}
	server, err := w.Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("resolve asset server: %v", err)
	}
	if err := server.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err != nil {
		t.Fatalf("register atlas: %v", err)
	}
	return w
}

// drawContext builds a draw context against a 100x100 logical target,
// the resolution every culling test's viewport math assumes.
func drawContext(w *World, alpha float64) *Context {
	return &Context{World: w, Alpha: alpha, LogicalWidth: 100, LogicalHeight: 100}
}

func spawnCamera(t *testing.T, w *World, prev, curr geom.Vector2, zoom float64, primary bool) {
	t.Helper()
	if _, err := w.NewEntity(
		Camera{Zoom: zoom, Primary: primary},
		Transform{Position: curr, Scale: unitScale},
		PrevTransform{Position: prev},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
}

func spawnTextureSprite(t *testing.T, w *World, texture asset.ID, prev, curr geom.Vector2, sprite Sprite) *Entity {
	t.Helper()
	e, err := w.NewEntity(
		TextureSprite{Texture: texture},
		sprite,
		Transform{Position: curr, Scale: unitScale},
		PrevTransform{Position: prev},
	)
	if err != nil {
		t.Fatalf("spawn texture sprite: %v", err)
	}
	return e
}

// Without a primary camera the collector has nothing to draw: empty
// Items, no error — overlays still run. The unresolvable sprite proves
// sprite resolution never happens on this path.
func TestCollectWithoutPrimaryCameraReturnsEmptyList(t *testing.T) {
	w := newCollectorWorld(t)
	spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, Sprite{})

	list, err := NewCollector(w).Collect(drawContext(w, 0.5))
	if err != nil {
		t.Fatalf("Collect without primary camera: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("Items = %d, want 0", len(list.Items))
	}
}

func TestCollectIgnoresNonPrimaryCameras(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, false)

	list, err := NewCollector(w).Collect(drawContext(w, 0.5))
	if err != nil {
		t.Fatalf("Collect with non-primary camera: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("Items = %d, want 0", len(list.Items))
	}
}

// The zero value shows; Hidden opts out. A hidden sprite never
// collects — no resolution, no bounds, no draw.
func TestCollectSkipsHiddenSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, Sprite{})
	spawnTextureSprite(t, w, "tex2.png", geom.Vector2{}, geom.Vector2{}, Sprite{Hidden: true})

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with hidden sprite: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("Items = %d, want 1 — the hidden sprite must not collect", len(list.Items))
	}
	if list.Items[0].Texture != "tex.png" {
		t.Errorf("Items[0].Texture = %q, want tex.png", list.Items[0].Texture)
	}
}

// The camera position interpolates from PrevTransform to Transform by
// alpha; the zoom comes straight from the camera component.
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
		list, err := collector.Collect(drawContext(w, tc.alpha))
		if err != nil {
			t.Fatalf("Collect at alpha %v: %v", tc.alpha, err)
		}
		if !list.Camera.Position.AlmostEqual(tc.want, 1e-9) {
			t.Errorf("alpha %v: camera position = %v, want %v", tc.alpha, list.Camera.Position, tc.want)
		}
		if list.Camera.Zoom != 2 {
			t.Errorf("alpha %v: camera zoom = %v, want 2", tc.alpha, list.Camera.Zoom)
		}
	}
}

// Several primary cameras: the first in deterministic query order wins.
func TestCollectFirstPrimaryCameraWins(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{X: 10, Y: 0}, 1, true)
	spawnCamera(t, w, geom.Vector2{X: 100, Y: 0}, geom.Vector2{X: 200, Y: 0}, 1, true)

	list, err := NewCollector(w).Collect(drawContext(w, 0.5))
	if err != nil {
		t.Fatalf("Collect with two primary cameras: %v", err)
	}
	want := geom.Vector2{X: 5, Y: 0}
	if !list.Camera.Position.AlmostEqual(want, 1e-9) {
		t.Errorf("camera position = %v, want %v from the first primary", list.Camera.Position, want)
	}
}

// Sprite positions interpolate by alpha; rotation and scale snap to the
// current transform. A texture sprite carries an empty Rect — the blit
// draws the whole texture.
func TestCollectInterpolatesTextureSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png",
		geom.Vector2{}, geom.Vector2{X: 10, Y: 20},
		Sprite{FlipH: true, Transparency: 0.5})

	collector := NewCollector(w)
	for _, tc := range []struct {
		alpha float64
		want  geom.Vector2
	}{
		{0, geom.Vector2{}},
		{0.5, geom.Vector2{X: 5, Y: 10}},
		{1, geom.Vector2{X: 10, Y: 20}},
	} {
		list, err := collector.Collect(drawContext(w, tc.alpha))
		if err != nil {
			t.Fatalf("Collect at alpha %v: %v", tc.alpha, err)
		}
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
		if !item.FlipH || item.Transparency != 0.5 {
			t.Errorf("flip/transparency = %v/%v, want true/0.5", item.FlipH, item.Transparency)
		}
	}
}

// An atlas sprite resolves to its atlas texture and region rect at
// collection, and culls against the region's dimensions.
func TestCollectResolvesAtlasSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	for name := range map[string]image.Rectangle{
		"player": image.Rect(0, 0, 2, 2),
		"enemy":  image.Rect(2, 2, 4, 4),
	} {
		if _, err := w.NewEntity(
			AtlasSprite{Atlas: "sprites", Region: name},
			Sprite{},
			Transform{Position: geom.Vector2{}, Scale: unitScale},
			PrevTransform{},
		); err != nil {
			t.Fatalf("spawn atlas sprite %q: %v", name, err)
		}
		// A second sprite off-screen: the region is 2x2, so at x=60 its
		// bounds end at 61, well outside the 100x100 viewport's +50.
		if _, err := w.NewEntity(
			AtlasSprite{Atlas: "sprites", Region: name},
			Sprite{},
			Transform{Position: geom.Vector2{X: 60}, Scale: unitScale},
			PrevTransform{Position: geom.Vector2{X: 60}},
		); err != nil {
			t.Fatalf("spawn culled atlas sprite %q: %v", name, err)
		}
	}

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with atlas sprites: %v", err)
	}
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

// Culling: the viewport is the logical resolution divided by zoom in
// world space, centered on the interpolated camera. Sprites whose
// scaled bounds touch the viewport survive.
func TestCollectCullsAgainstViewport(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, Sprite{})           // dead center: kept
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 48}, geom.Vector2{X: 48}, Sprite{}) // 4x4 bounds touch +50: kept
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 54}, geom.Vector2{X: 54}, Sprite{}) // bounds [52,56]: culled
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 60}, geom.Vector2{X: 60}, Sprite{}) // far out: culled

	// Scale grows bounds: the same off-screen position survives at 10x.
	if _, err := w.NewEntity(
		TextureSprite{Texture: "tex.png"},
		Sprite{},
		Transform{Position: geom.Vector2{X: 60}, Scale: geom.Vector2{X: 10, Y: 10}},
		PrevTransform{Position: geom.Vector2{X: 60}},
	); err != nil {
		t.Fatalf("spawn scaled sprite: %v", err)
	}

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with culling: %v", err)
	}
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
	spawnTextureSprite(t, zoomed, "tex.png", geom.Vector2{X: 48}, geom.Vector2{X: 48}, Sprite{})
	list, err = NewCollector(zoomed).Collect(drawContext(zoomed, 0))
	if err != nil {
		t.Fatalf("Collect with zoom: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("Items = %d, want 0 under 2x zoom", len(list.Items))
	}
}

// Fail-fast: any unresolvable sprite source errors the frame naming the
// handles, whichever variant it came from.
func TestCollectFailsFastNamingHandles(t *testing.T) {
	cases := []struct {
		name   string
		sprite func(t *testing.T, w *World)
		want   []string
	}{
		{"unloadable texture", func(t *testing.T, w *World) {
			spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, Sprite{})
		}, []string{"missing.png"}},
		{"unregistered atlas", func(t *testing.T, w *World) {
			if _, err := w.NewEntity(
				AtlasSprite{Atlas: "ghost", Region: "player"},
				Sprite{},
				Transform{Position: geom.Vector2{}, Scale: unitScale},
				PrevTransform{},
			); err != nil {
				t.Fatalf("spawn sprite: %v", err)
			}
		}, []string{"ghost"}},
		{"unknown region", func(t *testing.T, w *World) {
			if _, err := w.NewEntity(
				AtlasSprite{Atlas: "sprites", Region: "boss"},
				Sprite{},
				Transform{Position: geom.Vector2{}, Scale: unitScale},
				PrevTransform{},
			); err != nil {
				t.Fatalf("spawn sprite: %v", err)
			}
		}, []string{"boss", "sprites"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newCollectorWorld(t)
			spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
			tc.sprite(t, w)

			list, err := NewCollector(w).Collect(drawContext(w, 0))
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

// Ordering: layer first, then SortOrder, then the interpolated world Y
// — higher Y draws later, on top. Full ties keep spawn order.
func TestCollectSortsLayerThenOrderThenY(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	at := func(texture asset.ID, y float64, sprite Sprite) {
		spawnTextureSprite(t, w, texture, geom.Vector2{Y: y}, geom.Vector2{Y: y}, sprite)
	}

	at("tex.png", -10, Sprite{Layer: 0})               // A
	at("tex.png", 10, Sprite{Layer: 0})                // B
	at("tex.png", -10, Sprite{Layer: 1})               // C
	at("tex.png", -10, Sprite{Layer: 0, SortOrder: 1}) // D
	at("tex.png", 10, Sprite{Layer: 0, SortOrder: -1}) // E
	at("tex.png", 0, Sprite{Layer: 2})                 // tie 1
	at("tex2.png", 0, Sprite{Layer: 2})                // tie 2

	want := []asset.ID{
		"tex.png",  // E: layer 0, sort -1 — beats every sort-0 sprite
		"tex.png",  // A: layer 0, sort 0, y -10
		"tex.png",  // B: layer 0, sort 0, y 10
		"tex.png",  // D: layer 0, sort 1
		"tex.png",  // C: layer 1
		"tex.png",  // tie 1: layer 2, spawn order
		"tex2.png", // tie 2
	}

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with ordering: %v", err)
	}
	if len(list.Items) != len(want) {
		t.Fatalf("Items = %d, want %d", len(list.Items), len(want))
	}
	for i, item := range list.Items {
		if item.Texture != want[i] {
			t.Errorf("Items[%d].Texture = %q, want %q", i, item.Texture, want[i])
		}
	}
}

// DrawList.Items views the collector's reused buffers: the next collect
// overwrites that buffer underneath a retained list, which is the
// do-not-retain contract.
func TestCollectReusesBuffers(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	firstSprite := spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, Sprite{})
	spawnTextureSprite(t, w, "tex2.png", geom.Vector2{}, geom.Vector2{}, Sprite{})

	collector := NewCollector(w)
	first, err := collector.Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("first Collect: %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].Texture != "tex.png" {
		t.Fatalf("first Collect: Items = %v, want [tex.png tex2.png]", first.Items)
	}

	if err := w.DestroyEntity(firstSprite); err != nil {
		t.Fatalf("destroy first sprite: %v", err)
	}
	second, err := collector.Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("second Collect: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("second Collect: Items = %d, want 1 with no stale entry", len(second.Items))
	}
	if second.Items[0].Texture != "tex2.png" {
		t.Errorf("second Collect: Items[0].Texture = %q, want tex2.png", second.Items[0].Texture)
	}
	// The retained list views the same buffer: the second collect
	// overwrote its first item. A stale length is exactly why DrawLists
	// must not be retained across frames.
	if first.Items[0].Texture != "tex2.png" {
		t.Errorf("retained list Items[0].Texture = %q, want tex2.png — the buffer was reused underneath it",
			first.Items[0].Texture)
	}
}

func spawnRectPrimitive(t *testing.T, w *World, size geom.Vector2, prev, curr geom.Vector2, prim Primitive) {
	t.Helper()
	if _, err := w.NewEntity(
		RectPrimitive{Size: size},
		prim,
		Transform{Position: curr, Scale: unitScale},
		PrevTransform{Position: prev},
	); err != nil {
		t.Fatalf("spawn rect primitive: %v", err)
	}
}

func spawnCirclePrimitive(t *testing.T, w *World, radius float64, prev, curr geom.Vector2, prim Primitive, transform Transform) {
	t.Helper()
	transform.Position = curr
	if transform.Scale == (geom.Vector2{}) {
		transform.Scale = unitScale
	}
	if _, err := w.NewEntity(
		CirclePrimitive{Radius: radius},
		prim,
		transform,
		PrevTransform{Position: prev},
	); err != nil {
		t.Fatalf("spawn circle primitive: %v", err)
	}
}

func spawnLinePrimitive(t *testing.T, w *World, to geom.Vector2, prev, curr geom.Vector2, prim Primitive) {
	t.Helper()
	if _, err := w.NewEntity(
		LinePrimitive{To: to},
		prim,
		Transform{Position: curr, Scale: unitScale},
		PrevTransform{Position: prev},
	); err != nil {
		t.Fatalf("spawn line primitive: %v", err)
	}
}

// All three geometry variants stage: variant geometry becomes the
// Shape, style flows (nil color defaults to black), rotation and scale
// snap from the transform, and positions interpolate like sprites.
func TestCollectStagesShapePrimitives(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnRectPrimitive(t, w, geom.Vector2{X: 10, Y: 20},
		geom.Vector2{}, geom.Vector2{X: 10, Y: 0},
		Primitive{Layer: 1, Transparency: 0.5})
	spawnCirclePrimitive(t, w, 5, geom.Vector2{}, geom.Vector2{},
		Primitive{Color: color.RGBA{R: 255, A: 255}},
		Transform{Rotation: 0.5, Scale: geom.Vector2{X: 2, Y: 2}})
	spawnLinePrimitive(t, w, geom.Vector2{X: 30, Y: 0}, geom.Vector2{}, geom.Vector2{}, Primitive{})

	list, err := NewCollector(w).Collect(drawContext(w, 0.5))
	if err != nil {
		t.Fatalf("Collect with shapes: %v", err)
	}
	if len(list.Items) != 3 {
		t.Fatalf("Items = %d, want 3", len(list.Items))
	}

	shapes := map[string]DrawItem{}
	for _, item := range list.Items {
		switch shape := item.Shape.(type) {
		case RectShape:
			if shape.Size != (geom.Vector2{X: 10, Y: 20}) {
				t.Errorf("rect shape size = %v, want (10, 20)", shape.Size)
			}
			if !item.Position.AlmostEqual(geom.Vector2{X: 5, Y: 0}, 1e-9) {
				t.Errorf("rect position = %v, want interpolated (5, 0)", item.Position)
			}
			if item.Tint != color.Black {
				t.Errorf("rect tint = %v, want the black default for a nil color", item.Tint)
			}
			if item.Transparency != 0.5 {
				t.Errorf("rect transparency = %v, want 0.5", item.Transparency)
			}
			shapes["rect"] = item
		case CircleShape:
			if shape.Radius != 5 {
				t.Errorf("circle shape radius = %v, want 5", shape.Radius)
			}
			if item.Rotation != 0.5 || item.Scale != (geom.Vector2{X: 2, Y: 2}) {
				t.Errorf("circle rotation/scale = %v/%v, want snapped 0.5/(2, 2)", item.Rotation, item.Scale)
			}
			if item.Tint != color.Color(color.RGBA{R: 255, A: 255}) {
				t.Errorf("circle tint = %v, want the declared color", item.Tint)
			}
			shapes["circle"] = item
		case LineShape:
			if shape.To != (geom.Vector2{X: 30, Y: 0}) {
				t.Errorf("line shape To = %v, want (30, 0)", shape.To)
			}
			shapes["line"] = item
		default:
			t.Errorf("unexpected shape %T in the draw list", item.Shape)
		}
	}
	if len(shapes) != 3 {
		t.Fatalf("collected shapes = %v, want one of each kind", shapes)
	}
}

// Shapes and sprites sort together in one list: same layer → sort
// order → world Y, with full ties keeping pass order (sprites before
// shapes).
func TestCollectOrdersShapesWithSprites(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{Y: 10}, geom.Vector2{Y: 10}, Sprite{Layer: 1})
	spawnRectPrimitive(t, w, geom.Vector2{X: 4, Y: 4}, geom.Vector2{Y: -10}, geom.Vector2{Y: -10}, Primitive{}) // layer 0
	spawnCirclePrimitive(t, w, 2, geom.Vector2{Y: -10}, geom.Vector2{Y: -10}, Primitive{Layer: 1}, Transform{})
	spawnLinePrimitive(t, w, geom.Vector2{X: 8, Y: 0}, geom.Vector2{Y: 10}, geom.Vector2{Y: 10}, Primitive{Layer: 1})

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with mixed drawables: %v", err)
	}
	if len(list.Items) != 4 {
		t.Fatalf("Items = %d, want 4", len(list.Items))
	}

	// Layer 0 rect first; layer 1: circle (y -10), then the y-10 tie
	// resolved by pass order - the sprite pass runs before the line
	// pass.
	want := []struct {
		texture asset.ID
		shape   Shape
	}{
		{"", RectShape{Size: geom.Vector2{X: 4, Y: 4}}}, // rect, layer 0
		{"", CircleShape{Radius: 2}},                    // circle, layer 1, y -10
		{"tex.png", nil},                                // sprite, layer 1, y 10 (pass order)
		{"", LineShape{To: geom.Vector2{X: 8, Y: 0}}},   // line, layer 1, y 10
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

// Per-shape culling, including the line's offset bounds: a segment
// whose position is outside the viewport survives because its bounds
// center halfway along the segment - the case a centered-bounds
// assumption would get wrong.
func TestCollectCullsShapes(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	// Viewport is [-50, 50]^2 at zoom 1.
	spawnRectPrimitive(t, w, geom.Vector2{X: 10, Y: 10}, geom.Vector2{X: 40}, geom.Vector2{X: 40}, Primitive{}) // kept
	spawnRectPrimitive(t, w, geom.Vector2{X: 10, Y: 10}, geom.Vector2{X: 56}, geom.Vector2{X: 56}, Primitive{}) // bounds [51,61]: culled
	spawnCirclePrimitive(t, w, 10, geom.Vector2{X: 61}, geom.Vector2{X: 61}, Primitive{}, Transform{})          // bounds [51,71]: culled
	// The offset proof: position (75, 0) is outside the viewport, but
	// the segment reaches back to (45, 0) - its bounds are [45, 75],
	// which overlaps. Centered bounds [60, 90] would have culled it.
	spawnLinePrimitive(t, w, geom.Vector2{X: -30, Y: 0}, geom.Vector2{X: 75}, geom.Vector2{X: 75}, Primitive{})
	spawnLinePrimitive(t, w, geom.Vector2{X: 20, Y: 0}, geom.Vector2{X: 60}, geom.Vector2{X: 60}, Primitive{}) // bounds [60,80]: culled

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with culled shapes: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("Items = %d, want 2 (the near rect and the reaching line)", len(list.Items))
	}
	var gotRect, gotLine bool
	for _, item := range list.Items {
		switch item.Shape.(type) {
		case RectShape:
			gotRect = true
		case LineShape:
			gotLine = true
		}
	}
	if !gotRect || !gotLine {
		t.Fatalf("survivors = rect:%v line:%v, want both", gotRect, gotLine)
	}
}

// The zero value shows; Hidden opts out for shapes exactly as for
// sprites.
func TestCollectSkipsHiddenShapes(t *testing.T) {
	w := newCollectorWorld(t)
	spawnCamera(t, w, geom.Vector2{}, geom.Vector2{}, 1, true)
	spawnRectPrimitive(t, w, geom.Vector2{X: 10, Y: 10}, geom.Vector2{}, geom.Vector2{}, Primitive{})
	spawnCirclePrimitive(t, w, 5, geom.Vector2{}, geom.Vector2{}, Primitive{Hidden: true}, Transform{})

	list, err := NewCollector(w).Collect(drawContext(w, 0))
	if err != nil {
		t.Fatalf("Collect with hidden shape: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("Items = %d, want 1 - the hidden circle must not collect", len(list.Items))
	}
	if _, ok := list.Items[0].Shape.(RectShape); !ok {
		t.Errorf("Items[0].Shape = %T, want RectShape", list.Items[0].Shape)
	}
}
