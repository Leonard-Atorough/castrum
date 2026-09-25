package core

import (
	"bytes"
	"image"
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
	spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, Sprite{Visible: true})

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
		Sprite{Visible: true, Opacity: 0.5, FlipH: true})

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
		if !item.FlipH || item.Opacity != 0.5 {
			t.Errorf("flip/opacity = %v/%v, want true/0.5", item.FlipH, item.Opacity)
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
			Sprite{Visible: true},
			Transform{Position: geom.Vector2{}, Scale: unitScale},
			PrevTransform{},
		); err != nil {
			t.Fatalf("spawn atlas sprite %q: %v", name, err)
		}
		// A second sprite off-screen: the region is 2x2, so at x=60 its
		// bounds end at 61, well outside the 100x100 viewport's +50.
		if _, err := w.NewEntity(
			AtlasSprite{Atlas: "sprites", Region: name},
			Sprite{Visible: true},
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
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, Sprite{Visible: true})           // dead center: kept
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 48}, geom.Vector2{X: 48}, Sprite{Visible: true}) // 4x4 bounds touch +50: kept
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 54}, geom.Vector2{X: 54}, Sprite{Visible: true}) // bounds [52,56]: culled
	spawnTextureSprite(t, w, "tex.png", geom.Vector2{X: 60}, geom.Vector2{X: 60}, Sprite{Visible: true}) // far out: culled

	// Scale grows bounds: the same off-screen position survives at 10x.
	if _, err := w.NewEntity(
		TextureSprite{Texture: "tex.png"},
		Sprite{Visible: true},
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
	spawnTextureSprite(t, zoomed, "tex.png", geom.Vector2{X: 48}, geom.Vector2{X: 48}, Sprite{Visible: true})
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
			spawnTextureSprite(t, w, "missing.png", geom.Vector2{}, geom.Vector2{}, Sprite{Visible: true})
		}, []string{"missing.png"}},
		{"unregistered atlas", func(t *testing.T, w *World) {
			if _, err := w.NewEntity(
				AtlasSprite{Atlas: "ghost", Region: "player"},
				Sprite{Visible: true},
				Transform{Position: geom.Vector2{}, Scale: unitScale},
				PrevTransform{},
			); err != nil {
				t.Fatalf("spawn sprite: %v", err)
			}
		}, []string{"ghost"}},
		{"unknown region", func(t *testing.T, w *World) {
			if _, err := w.NewEntity(
				AtlasSprite{Atlas: "sprites", Region: "boss"},
				Sprite{Visible: true},
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

	at("tex.png", -10, Sprite{Visible: true, Layer: 0})               // A
	at("tex.png", 10, Sprite{Visible: true, Layer: 0})                // B
	at("tex.png", -10, Sprite{Visible: true, Layer: 1})               // C
	at("tex.png", -10, Sprite{Visible: true, Layer: 0, SortOrder: 1}) // D
	at("tex.png", 10, Sprite{Visible: true, Layer: 0, SortOrder: -1}) // E
	at("tex.png", 0, Sprite{Visible: true, Layer: 2})                 // tie 1
	at("tex2.png", 0, Sprite{Visible: true, Layer: 2})                // tie 2

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
	firstSprite := spawnTextureSprite(t, w, "tex.png", geom.Vector2{}, geom.Vector2{}, Sprite{Visible: true})
	spawnTextureSprite(t, w, "tex2.png", geom.Vector2{}, geom.Vector2{}, Sprite{Visible: true})

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
