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
	"time"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/signal"
	spr "github.com/Leonard-Atorough/castrum/sprite"
)

var unitScale = geom.Vector2{X: 1, Y: 1}

// --- harness ---------------------------------------------------------------

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

// scene is a test harness for collector tests: a world with the test asset
// filesystem and window, plus fluent spawners so the prev/curr transform
// pair rule lives in one place.
type scene struct {
	t   *testing.T
	w   *core.World
	ctx *core.Context
}

// newScene returns a scene with the test asset loader.
func newScene(t *testing.T) *scene {
	t.Helper()
	w := newCollectorWorld(t)
	return &scene{t: t, w: w, ctx: drawContext(w)}
}

// newBareScene returns a scene without an asset loader, for loader-less paths.
func newBareScene(t *testing.T) *scene {
	t.Helper()
	w := core.NewWorld()
	return &scene{t: t, w: w, ctx: drawContext(w)}
}

func (s *scene) camera(curr geom.Vector2, zoom float64) *scene {
	return s.cameraFrom(curr, curr, zoom, true)
}

func (s *scene) cameraFrom(prev, curr geom.Vector2, zoom float64, primary bool) *scene {
	s.t.Helper()
	zoomVec := geom.Vector2{X: zoom, Y: zoom}
	if _, err := s.w.NewEntity(
		Camera{Primary: primary},
		core.Transform{Position: curr, Scale: zoomVec},
		core.PrevTransform{Position: prev, Scale: zoomVec},
	); err != nil {
		s.t.Fatalf("spawn camera: %v", err)
	}
	return s
}

func (s *scene) addCamera(cam Camera, curr core.Transform, prev core.PrevTransform) *scene {
	s.t.Helper()
	if _, err := s.w.NewEntity(cam, curr, prev); err != nil {
		s.t.Fatalf("spawn camera: %v", err)
	}
	return s
}

func (s *scene) spawn(tr core.Transform, prev core.PrevTransform, visuals ...any) core.Entity {
	s.t.Helper()
	comps := make([]any, 0, len(visuals)+2)
	comps = append(comps, visuals...)
	comps = append(comps, tr, prev)
	e, err := s.w.NewEntity(comps...)
	if err != nil {
		s.t.Fatalf("spawn %T: %v", visuals, err)
	}
	return e
}

func (s *scene) add(visual any, tr core.Transform) *scene {
	s.addFrom(visual, tr.Position, tr)
	return s
}

func (s *scene) addFrom(visual any, prevPos geom.Vector2, tr core.Transform) *scene {
	if tr.Scale == (geom.Vector2{}) {
		tr.Scale = unitScale
	}
	s.spawn(tr, core.PrevTransform{Position: prevPos, Rotation: tr.Rotation, Scale: tr.Scale}, visual)
	return s
}

func (s *scene) texture(id asset.ID, at geom.Vector2, style spr.Style) *scene {
	return s.add(spr.Sprite{Asset: id, Style: style}, core.Transform{Position: at})
}

func (s *scene) text(str string, at geom.Vector2, fill *color.RGBA) *scene {
	return s.add(spr.Text{Asset: "fonts/go.ttf", Text: str, TextSize: 16, Style: spr.Style{Color: fill}},
		core.Transform{Position: at})
}

func (s *scene) withAnimatorSets() *scene {
	sets, err := animation.NewSets(map[string]animation.Set{
		"test": {Name: "test", States: map[string]animation.State{
			"play": {
				Name: "play", Loop: animation.Loop,
				Tracks: []animation.Track{
					{Name: "body", Source: "sprites.json", Frames: []string{"player", "enemy"}, FPS: 10},
					{Name: "overlay", Source: "sprites.json", Frames: []string{"enemy"}, FPS: 10},
				},
			},
		}},
	})
	if err != nil {
		s.t.Fatal(err)
	}
	if err := s.w.InsertResource(sets); err != nil {
		s.t.Fatal(err)
	}
	return s
}

func (s *scene) startAnimation(dt time.Duration) *scene {
	if err := s.w.Provide(func(*core.World) (*signal.Hub, error) { return signal.NewHub(), nil }); err != nil {
		s.t.Fatal(err)
	}
	ctx := &core.Context{World: s.w, DeltaTime: dt, Tick: 1}
	if err := animation.Animate(ctx); err != nil {
		s.t.Fatal(err)
	}
	return s
}

func newTestAnimator(t *testing.T) animation.Animator {
	t.Helper()
	a, err := animation.NewAnimator("test", "play")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func (s *scene) collector() *Collector { return NewCollector(s.w) }

func (s *scene) collect(alpha float64) DrawList {
	return s.collectWith(s.collector(), alpha)
}

func (s *scene) collectWith(c *Collector, alpha float64) DrawList {
	s.t.Helper()
	list, err := c.Collect(s.ctx, alpha)
	if err != nil {
		s.t.Fatalf("Collect: %v", err)
	}
	return list
}

func (s *scene) tryCollect(alpha float64) (DrawList, error) {
	return s.collector().Collect(s.ctx, alpha)
}

func wantItems(t *testing.T, list DrawList, n int) []DrawItem {
	t.Helper()
	if len(list.Items) != n {
		t.Fatalf("Items = %d, want %d", len(list.Items), n)
	}
	return list.Items
}

func texturePayload(t *testing.T, item DrawItem) TextureDraw {
	t.Helper()
	p, ok := item.Payload().(TextureDraw)
	if !ok {
		t.Fatalf("payload = %T, want TextureDraw", item.Payload())
	}
	return p
}

func shapePayload(t *testing.T, item DrawItem) ShapeDraw {
	t.Helper()
	p, ok := item.Payload().(ShapeDraw)
	if !ok {
		t.Fatalf("payload = %T, want ShapeDraw", item.Payload())
	}
	return p
}

func textPayload(t *testing.T, item DrawItem) TextDraw {
	t.Helper()
	p, ok := item.Payload().(TextDraw)
	if !ok {
		t.Fatalf("payload = %T, want TextDraw", item.Payload())
	}
	return p
}

// --- camera resolution -----------------------------------------------------

func TestCollectCameraResolution(t *testing.T) {
	t.Run("no primary camera yields an empty list and resolves no assets", func(t *testing.T) {
		s := newScene(t)
		// The missing texture is deliberate: sprite resolution must not
		// happen on a no-camera path.
		s.texture("missing.png", geom.Vector2{}, spr.Style{})
		if list := s.collect(0.5); len(list.Items) != 0 {
			t.Errorf("Items = %d, want 0", len(list.Items))
		}
	})

	t.Run("a non-primary camera yields an empty list", func(t *testing.T) {
		s := newScene(t).cameraFrom(geom.Vector2{}, geom.Vector2{}, 1, false)
		s.texture("tex.png", geom.Vector2{}, spr.Style{})
		if list := s.collect(0); len(list.Items) != 0 {
			t.Errorf("Items = %d, want 0", len(list.Items))
		}
	})

	t.Run("the first primary camera wins", func(t *testing.T) {
		s := newScene(t).
			cameraFrom(geom.Vector2{}, geom.Vector2{X: 10, Y: 0}, 1, true).
			cameraFrom(geom.Vector2{X: 100, Y: 0}, geom.Vector2{X: 200, Y: 0}, 1, true)
		list := s.collect(0.5)
		if want := (geom.Vector2{X: 5, Y: 0}); !list.Camera.Position.AlmostEqual(want, 1e-9) {
			t.Errorf("camera position = %v, want %v from the first primary", list.Camera.Position, want)
		}
	})

	t.Run("the engine camera fills in only without a user primary", func(t *testing.T) {
		s := newScene(t)
		if list := s.collect(0); len(list.Items) != 0 || list.Camera != (CameraView{}) {
			t.Fatalf("no-camera list = %+v, want empty", list)
		}
		if _, err := SpawnEngineCamera(s.w); err != nil {
			t.Fatalf("spawn engine camera: %v", err)
		}
		list := s.collect(0)
		if list.Camera.Position != (geom.Vector2{}) || list.Camera.Zoom != 1 {
			t.Fatalf("engine camera view = %+v, want origin zoom 1", list.Camera)
		}
		s.camera(geom.Vector2{X: 30, Y: 30}, 2)
		list = s.collect(0)
		if list.Camera.Position != (geom.Vector2{X: 30, Y: 30}) || list.Camera.Zoom != 2 {
			t.Fatalf("user camera should win: view = %+v, want (30, 30) zoom 2", list.Camera)
		}
		win, err := s.w.Resource[Window]()
		if err != nil {
			t.Fatal(err)
		}
		vp := win.Viewport()
		view, ok, err := NewCameraResolver(s.w).Resolve(0, vp.LogicalWidth, vp.LogicalHeight)
		if err != nil || !ok || view != list.Camera {
			t.Fatalf("resolved camera = %+v, %v, %v; want collected view %+v", view, ok, err, list.Camera)
		}
	})

	t.Run("invalid zoom is rejected", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			scale geom.Vector2
		}{
			{"negative zoom", geom.Vector2{X: -1, Y: -1}},
			{"non-uniform zoom", geom.Vector2{X: 1, Y: 2}},
			{"infinite zoom", geom.Vector2{X: math.Inf(1), Y: math.Inf(1)}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := newScene(t)
				if _, err := s.w.NewEntity(
					Camera{Primary: true},
					core.Transform{Scale: tc.scale},
					core.PrevTransform{Scale: tc.scale},
				); err != nil {
					t.Fatalf("spawn camera: %v", err)
				}
				if _, err := s.tryCollect(0); err == nil || !strings.Contains(err.Error(), "camera zoom") {
					t.Errorf("Collect error = %v, want it to name the camera zoom", err)
				}
			})
		}
	})
}

// --- camera interpolation and clamping -------------------------------------

func TestCollectCameraInterpolation(t *testing.T) {
	t.Run("position and zoom", func(t *testing.T) {
		s := newScene(t).cameraFrom(geom.Vector2{}, geom.Vector2{X: 10, Y: 4}, 2, true)
		c := s.collector()
		for _, tc := range []struct {
			alpha float64
			want  geom.Vector2
		}{
			{0, geom.Vector2{}},
			{0.5, geom.Vector2{X: 5, Y: 2}},
			{1, geom.Vector2{X: 10, Y: 4}},
		} {
			list := s.collectWith(c, tc.alpha)
			if !list.Camera.Position.AlmostEqual(tc.want, 1e-9) {
				t.Errorf("alpha %v: camera position = %v, want %v", tc.alpha, list.Camera.Position, tc.want)
			}
			if list.Camera.Zoom != 2 {
				t.Errorf("alpha %v: camera zoom = %v, want 2", tc.alpha, list.Camera.Zoom)
			}
		}
	})

	t.Run("rotation", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			prev    float64
			curr    float64
			wantMid float64
		}{
			{"negative angles", -math.Pi / 2, -math.Pi / 4, -3 * math.Pi / 8},
			{"same angle across the wrap", 3 * math.Pi / 2, -math.Pi / 2, math.Pi / 2},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := newScene(t)
				if _, err := s.w.NewEntity(
					Camera{Primary: true},
					core.Transform{Rotation: tc.curr, Scale: unitScale},
					core.PrevTransform{Rotation: tc.prev, Scale: unitScale},
				); err != nil {
					t.Fatalf("spawn camera: %v", err)
				}
				c := s.collector()
				for _, a := range []struct {
					alpha float64
					want  float64
				}{
					{0, tc.prev},
					{0.5, tc.wantMid},
					{1, tc.curr},
				} {
					list := s.collectWith(c, a.alpha)
					if math.Abs(list.Camera.Rotation-a.want) > 1e-9 {
						t.Errorf("alpha %v: camera rotation = %v, want %v", a.alpha, list.Camera.Rotation, a.want)
					}
				}
			})
		}
	})
}

func TestCollectCameraClamping(t *testing.T) {
	bounds := geom.RectFromMinMax(geom.Vector2{}, geom.Vector2{X: 200, Y: 100})

	t.Run("clamps into bounds", func(t *testing.T) {
		s := newScene(t)
		tr := core.Transform{Position: geom.Vector2{X: 10, Y: 50}, Scale: unitScale}
		prev := core.PrevTransform{Position: tr.Position, Scale: tr.Scale}
		s.addCamera(Camera{Primary: true, Bounds: bounds}, tr, prev)
		list := s.collect(1)
		// A 100x100 view in 200x100 bounds: x clamps into [50, 150], y pins
		// to the only fitting value.
		if want := (geom.Vector2{X: 50, Y: 50}); list.Camera.Position != want {
			t.Errorf("clamped position = %v, want %v", list.Camera.Position, want)
		}
	})

	t.Run("pins to the bounds center when the view cannot fit", func(t *testing.T) {
		s := newScene(t)
		// Zoom 0.5 doubles the view to 200x200: neither axis fits.
		tr := core.Transform{Position: geom.Vector2{X: 10, Y: 50}, Scale: geom.Vector2{X: 0.5, Y: 0.5}}
		prev := core.PrevTransform{Position: tr.Position, Scale: tr.Scale}
		s.addCamera(Camera{Primary: true, Bounds: bounds}, tr, prev)
		list := s.collect(1)
		if want := (geom.Vector2{X: 100, Y: 50}); list.Camera.Position != want {
			t.Errorf("pinned position = %v, want %v", list.Camera.Position, want)
		}
	})

	t.Run("clamps a rotated view", func(t *testing.T) {
		s := newScene(t)
		tr := core.Transform{Rotation: math.Pi / 4, Scale: unitScale}
		prev := core.PrevTransform{Rotation: tr.Rotation, Scale: tr.Scale}
		s.addCamera(Camera{Primary: true, Bounds: bounds}, tr, prev)
		list := s.collect(1)
		// At 45 degrees the 100x100 view occupies ~141 world units per
		// axis: x clamps into the shrunk window while y, which cannot fit
		// the 100-unit bounds, pins to the bounds center.
		want := geom.Vector2{X: 50 * (math.Cos(math.Pi/4) + math.Sin(math.Pi/4)), Y: 50}
		if !list.Camera.Position.AlmostEqual(want, 1e-9) {
			t.Errorf("rotated clamped position = %v, want %v", list.Camera.Position, want)
		}
	})
}

func TestCollectCameraViewResource(t *testing.T) {
	t.Run("a successful collect publishes and updates the view", func(t *testing.T) {
		s := newScene(t).cameraFrom(geom.Vector2{X: 10, Y: 10}, geom.Vector2{X: 20, Y: 20}, 1, true)
		c := s.collector()
		s.collectWith(c, 0)
		view, err := s.w.Resource[*CameraView]()
		if err != nil {
			t.Fatalf("resolve view resource: %v", err)
		}
		if want := (geom.Vector2{X: 10, Y: 10}); view.Position != want {
			t.Errorf("resource position = %v, want %v", view.Position, want)
		}
		s.collectWith(c, 1)
		if want := (geom.Vector2{X: 20, Y: 20}); view.Position != want {
			t.Errorf("resource position after update = %v, want %v", view.Position, want)
		}
		// A second collector adopts the same resource instead of failing.
		s.collect(1)
	})

	t.Run("a failed collect keeps the previous view and registers none on first failure", func(t *testing.T) {
		s := newScene(t).cameraFrom(geom.Vector2{}, geom.Vector2{X: 10, Y: 0}, 1, true)
		s.texture("tex.png", geom.Vector2{}, spr.Style{})
		c := s.collector()
		s.collectWith(c, 0)
		view, err := s.w.Resource[*CameraView]()
		if err != nil {
			t.Fatalf("resolve view resource: %v", err)
		}
		if want := (geom.Vector2{}); view.Position != want {
			t.Fatalf("first view position = %v, want %v", view.Position, want)
		}

		s.texture("missing.png", geom.Vector2{}, spr.Style{})
		if _, err := c.Collect(s.ctx, 1); err == nil {
			t.Fatal("second Collect should fail on the missing texture")
		}
		if view.Position != (geom.Vector2{}) {
			t.Errorf("failed Collect published the failed frame's view %v; want it to keep the drawn frame's",
				view.Position)
		}

		broken := newScene(t).camera(geom.Vector2{}, 1)
		broken.texture("missing.png", geom.Vector2{}, spr.Style{})
		if _, err := broken.tryCollect(0); err == nil {
			t.Fatal("broken Collect should fail")
		}
		if _, err := broken.w.Resource[*CameraView](); !errors.Is(err, core.ErrResourceNotRegistered) {
			t.Errorf("failed first Collect registered a view: %v, want ErrResourceNotRegistered", err)
		}
	})
}

// --- culling ---------------------------------------------------------------

func TestCollectCulling(t *testing.T) {
	t.Run("viewport edges", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.texture("tex.png", geom.Vector2{}, spr.Style{})      // dead center: kept
		s.texture("tex.png", geom.Vector2{X: 48}, spr.Style{}) // 4x4 bounds touch +50: kept
		s.texture("tex.png", geom.Vector2{X: 54}, spr.Style{}) // bounds [52, 56]: culled
		s.texture("tex.png", geom.Vector2{X: 60}, spr.Style{}) // far out: culled
		// Scale grows bounds: the same off-screen position survives at 10x.
		s.add(spr.Sprite{Asset: "tex.png"}, core.Transform{Position: geom.Vector2{X: 60}, Scale: geom.Vector2{X: 10, Y: 10}})

		items := wantItems(t, s.collect(0), 3)
		for _, item := range items {
			if item.Position.X > 60 {
				t.Errorf("culled sprite leaked into the list at %v", item.Position)
			}
		}
	})

	t.Run("zoom shrinks the world-space viewport", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 2)
		// The x=48 sprite's bounds touch +50 at zoom 1, but 2x zoom halves
		// the viewport to the +25 edge.
		s.texture("tex.png", geom.Vector2{X: 48}, spr.Style{})
		wantItems(t, s.collect(0), 0)
	})

	t.Run("shapes", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		// Viewport is [-50, 50]^2 at zoom 1.
		s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{Position: geom.Vector2{X: 40, Y: 40}})
		s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{Position: geom.Vector2{X: 56, Y: 56}})
		s.add(spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 10, Y: 10}}, core.Transform{Position: geom.Vector2{X: 61, Y: 61}})
		// The offset proof: position (75, 0) is outside the viewport, but
		// the segment reaches back to (45, 0) - its bounds are [45, 75],
		// which overlaps. Centered bounds [60, 90] would have culled it.
		s.add(spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{X: -30, Y: 0}}, core.Transform{Position: geom.Vector2{X: 75, Y: 0}})
		s.add(spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{X: 20, Y: 0}}, core.Transform{Position: geom.Vector2{X: 60, Y: 0}})
		// The inverse proof: this line's position is dead center, but its
		// segment spans (80, 0) to (110, 0), entirely outside - the AABB
		// bounds cull it despite the visible position.
		s.add(spr.Primitive{Shape: spr.ShapeLine, From: geom.Vector2{X: 80, Y: 0}, To: geom.Vector2{X: 110, Y: 0}}, core.Transform{})

		items := wantItems(t, s.collect(0), 2)
		var gotRect, gotLine bool
		for _, item := range items {
			switch shapePayload(t, item).Shape.(type) {
			case spr.RectShape:
				gotRect = true
			case spr.LineShape:
				gotLine = true
			}
		}
		if !gotRect || !gotLine {
			t.Fatalf("survivors = rect:%v line:%v, want both", gotRect, gotLine)
		}
	})

	t.Run("rotated and stroked geometry", func(t *testing.T) {
		for _, tc := range []struct {
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
			t.Run(tc.name, func(t *testing.T) {
				s := newScene(t).camera(geom.Vector2{}, 1)
				s.add(tc.visual, core.Transform{Position: tc.position, Rotation: tc.rotation, Scale: tc.scale})
				if got := s.collect(1); len(got.Items) != tc.want {
					t.Fatalf("collected %d items, want %d", len(got.Items), tc.want)
				}
			})
		}
	})

	t.Run("interpolated scale", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		// Static position at x=54: the 4x4 texture's scale-1 bounds
		// [52, 56] sit outside the viewport's +50 edge.
		s.spawn(core.Transform{Position: geom.Vector2{X: 54}},
			core.PrevTransform{Position: geom.Vector2{X: 54}, Scale: geom.Vector2{X: 10, Y: 10}},
			spr.Sprite{Asset: "tex.png"})
		c := s.collector()

		list := s.collectWith(c, 0.5)
		item := wantItems(t, list, 1)[0]
		if !item.Scale.AlmostEqual(geom.Vector2{X: 5.5, Y: 5.5}, 1e-9) {
			t.Errorf("alpha 0.5: scale = %v, want 5.5 (the interpolated scale keeps it)", item.Scale)
		}
		if list := s.collectWith(c, 1); len(list.Items) != 0 {
			t.Fatalf("alpha 1: Items = %d, want 0 (the current scale culls it)", len(list.Items))
		}
	})

	t.Run("interpolated drawable rotation", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		position := geom.Vector2{X: 60}
		s.spawn(core.Transform{Position: position, Rotation: math.Pi, Scale: unitScale},
			core.PrevTransform{Position: position, Scale: unitScale},
			spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 40}})
		c := s.collector()
		for _, tc := range []struct {
			alpha float64
			want  int
		}{{0, 0}, {0.5, 1}, {1, 0}} {
			if got := s.collectWith(c, tc.alpha); len(got.Items) != tc.want {
				t.Fatalf("alpha %v: collected %d, want %d", tc.alpha, len(got.Items), tc.want)
			}
		}
	})

	t.Run("camera anchor shifts the viewport", func(t *testing.T) {
		s := newScene(t)
		tr := core.Transform{Scale: unitScale}
		s.addCamera(Camera{Primary: true, Anchor: geom.Vector2{X: -0.5, Y: -0.5}}, tr, core.PrevTransform{Scale: unitScale})
		s.texture("tex.png", geom.Vector2{X: 90, Y: 90}, spr.Style{})
		s.texture("tex.png", geom.Vector2{X: -10, Y: 50}, spr.Style{})
		s.texture("tex.png", geom.Vector2{X: 150, Y: 50}, spr.Style{})

		items := wantItems(t, s.collect(0), 1)
		if items[0].Position.X != 90 {
			t.Errorf("kept sprite = %v, want the one at (90, 90)", items[0].Position)
		}
	})

	t.Run("rotated camera culls conservatively", func(t *testing.T) {
		s := newScene(t)
		s.addCamera(Camera{Primary: true},
			core.Transform{Rotation: math.Pi / 2, Scale: unitScale},
			core.PrevTransform{Scale: unitScale})
		// Alpha 0.5 resolves rotation to pi/4; the circumscribed half extents
		// are 50*(cos45+sin45) ~ 70.7, so (65, 65) culls-in conservatively
		// even though the exact rotated view excludes it.
		s.texture("tex.png", geom.Vector2{X: 65, Y: 65}, spr.Style{})
		s.texture("tex.png", geom.Vector2{X: 90, Y: 90}, spr.Style{})

		list := s.collect(0.5)
		if want := math.Pi / 4; math.Abs(list.Camera.Rotation-want) > 1e-9 {
			t.Errorf("Camera.Rotation = %v, want %v", list.Camera.Rotation, want)
		}
		items := wantItems(t, list, 1)
		if items[0].Position.X != 65 {
			t.Fatalf("kept item = %v, want only the conservatively-visible (65, 65)", items[0].Position)
		}
	})
}

// --- drawable interpolation --------------------------------------------------

func TestCollectInterpolation(t *testing.T) {
	t.Run("sprite position carries its texture payload", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.addFrom(spr.Sprite{Asset: "tex.png", Style: spr.Style{FlipH: true, Color: &color.RGBA{R: 128, A: 64}}},
			geom.Vector2{}, core.Transform{Position: geom.Vector2{X: 10, Y: 20}})
		c := s.collector()
		for _, tc := range []struct {
			alpha float64
			want  geom.Vector2
		}{
			{0, geom.Vector2{}},
			{0.5, geom.Vector2{X: 5, Y: 10}},
			{1, geom.Vector2{X: 10, Y: 20}},
		} {
			item := wantItems(t, s.collectWith(c, tc.alpha), 1)[0]
			if !item.Position.AlmostEqual(tc.want, 1e-9) {
				t.Errorf("alpha %v: position = %v, want %v", tc.alpha, item.Position, tc.want)
			}
			p := texturePayload(t, item)
			if p.Texture != "tex.png" {
				t.Errorf("texture = %q, want tex.png", p.Texture)
			}
			if p.Rect != (image.Rectangle{}) {
				t.Errorf("rect = %v, want empty for a whole-texture sprite", p.Rect)
			}
			// The color flows whole: its alpha channel is the sprite's
			// opacity, so the item carries the color as declared.
			if !p.FlipH || item.Color != color.Color(color.RGBA{R: 128, A: 64}) {
				t.Errorf("flip/color = %v/%v, want true and the declared color", p.FlipH, item.Color)
			}
		}
	})

	t.Run("rotation and scale", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.spawn(core.Transform{Rotation: 1, Scale: geom.Vector2{X: 3, Y: 5}},
			core.PrevTransform{Scale: unitScale},
			spr.Sprite{Asset: "tex.png"})
		c := s.collector()
		for _, tc := range []struct {
			alpha    float64
			rotation float64
			scale    geom.Vector2
		}{
			{0, 0, geom.Vector2{X: 1, Y: 1}},
			{0.5, 0.5, geom.Vector2{X: 2, Y: 3}},
			{1, 1, geom.Vector2{X: 3, Y: 5}},
		} {
			item := wantItems(t, s.collectWith(c, tc.alpha), 1)[0]
			if item.Rotation != tc.rotation {
				t.Errorf("alpha %v: rotation = %v, want %v", tc.alpha, item.Rotation, tc.rotation)
			}
			if !item.Scale.AlmostEqual(tc.scale, 1e-9) {
				t.Errorf("alpha %v: scale = %v, want %v", tc.alpha, item.Scale, tc.scale)
			}
		}
	})

	t.Run("rotation across angle boundaries", func(t *testing.T) {
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
			t.Run(tc.name, func(t *testing.T) {
				s := newScene(t).camera(geom.Vector2{}, 1)
				s.spawn(core.Transform{Rotation: tc.curr, Scale: unitScale},
					core.PrevTransform{Rotation: tc.prev, Scale: unitScale},
					spr.Sprite{Asset: "tex.png"})
				c := s.collector()
				for _, a := range []struct {
					alpha float64
					want  float64
				}{
					{0, tc.prev},
					{0.5, tc.wantMid},
					{1, tc.curr},
				} {
					got := wantItems(t, s.collectWith(c, a.alpha), 1)[0].Rotation
					if math.Abs(got-a.want) > 1e-9 {
						t.Errorf("alpha %v: rotation = %v, want %v", a.alpha, got, a.want)
					}
				}
			})
		}
	})

	t.Run("non-uniform scale", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.spawn(core.Transform{Scale: geom.Vector2{X: 4, Y: 2}},
			core.PrevTransform{Scale: geom.Vector2{X: 2, Y: 10}},
			spr.Sprite{Asset: "tex.png"})
		c := s.collector()
		for _, tc := range []struct {
			alpha float64
			want  geom.Vector2
		}{
			{0, geom.Vector2{X: 2, Y: 10}},
			{0.25, geom.Vector2{X: 2.5, Y: 8}},
			{0.5, geom.Vector2{X: 3, Y: 6}},
			{1, geom.Vector2{X: 4, Y: 2}},
		} {
			got := wantItems(t, s.collectWith(c, tc.alpha), 1)[0].Scale
			if !got.AlmostEqual(tc.want, 1e-9) {
				t.Errorf("alpha %v: scale = %v, want %v", tc.alpha, got, tc.want)
			}
		}
	})
}

// --- textures and sheets -----------------------------------------------------

func TestCollectTextures(t *testing.T) {
	t.Run("sheet sprites resolve their atlas regions", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		for name := range map[string]image.Rectangle{
			"player": image.Rect(0, 0, 2, 2),
			"enemy":  image.Rect(2, 2, 4, 4),
		} {
			s.add(spr.Sprite{Asset: "sprites.json", Region: name}, core.Transform{})
			// The off-screen copy is culled.
			s.add(spr.Sprite{Asset: "sprites.json", Region: name}, core.Transform{Position: geom.Vector2{X: 60}})
		}

		items := wantItems(t, s.collect(0), 2)
		rects := map[image.Rectangle]bool{}
		for _, item := range items {
			p := texturePayload(t, item)
			if p.Texture != "tex.png" {
				t.Errorf("texture = %q, want the atlas texture tex.png", p.Texture)
			}
			rects[p.Rect] = true
		}
		for name, rect := range map[string]image.Rectangle{
			"player": image.Rect(0, 0, 2, 2),
			"enemy":  image.Rect(2, 2, 4, 4),
		} {
			if !rects[rect] {
				t.Errorf("region %q: rect %v missing from the list", name, rect)
			}
		}
	})

	t.Run("zero scale reads as unit", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		// No explicit prev: the pair rule snapshots the zero scale as
		// unit at spawn, so both ends of the interpolation read unscaled.
		s.add(spr.Sprite{Asset: "sprites.json", Region: "player"}, core.Transform{})
		item := wantItems(t, s.collect(0), 1)[0]
		if item.Scale != unitScale {
			t.Errorf("Scale = %v, want the zero scale to read as unit", item.Scale)
		}
		if p := texturePayload(t, item); p.Rect != image.Rect(0, 0, 2, 2) {
			t.Errorf("Rect = %v, want the player region", p.Rect)
		}
	})

	t.Run("the sheet loader is resolved lazily", func(t *testing.T) {
		s := newBareScene(t)
		s.add(spr.Sprite{Asset: "sprites.json", Region: "player"}, core.Transform{})
		c := s.collector()
		if list, err := c.Collect(s.ctx, 1); err != nil || len(list.Items) != 0 {
			t.Fatal("no-camera collection should not resolve the sheet")
		}
		s.camera(geom.Vector2{}, 1)
		if _, err := c.Collect(s.ctx, 1); err == nil || !strings.Contains(err.Error(), "asset loader") {
			t.Fatalf("missing loader error = %v", err)
		}
		loader := newCollectorWorld(t).MustResource[*asset.Loader]()
		if err := s.w.Provide(func(*core.World) (*asset.Loader, error) { return loader, nil }); err != nil {
			t.Fatal(err)
		}
		item := wantItems(t, s.collectWith(c, 1), 1)[0]
		if p := texturePayload(t, item); p.Texture != "tex.png" || p.Rect != image.Rect(0, 0, 2, 2) {
			t.Fatalf("sheet item = %+v, want tex.png player region", p)
		}
	})

	t.Run("shapes stage without an asset loader", func(t *testing.T) {
		s := newBareScene(t).camera(geom.Vector2{}, 1)
		s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}},
			core.Transform{Position: geom.Vector2{X: 50, Y: 50}})
		wantItems(t, s.collect(1), 1)
	})

	t.Run("a texture without an asset loader errors", func(t *testing.T) {
		s := newBareScene(t).camera(geom.Vector2{}, 1)
		s.texture("tex.png", geom.Vector2{X: 50, Y: 50}, spr.Style{})
		if _, err := s.tryCollect(1); err == nil || !strings.Contains(err.Error(), "asset loader") {
			t.Errorf("error = %v, want it to name the asset loader", err)
		}
	})
}

// --- shapes ------------------------------------------------------------------

func TestCollectShapes(t *testing.T) {
	s := newScene(t).camera(geom.Vector2{}, 1)
	s.addFrom(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 20}, Style: spr.Style{Layer: 1}},
		geom.Vector2{}, core.Transform{Position: geom.Vector2{X: 10, Y: 0}})
	s.add(spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}, Style: spr.Style{Color: &color.RGBA{R: 255, A: 255}}},
		core.Transform{Rotation: 0.5, Scale: geom.Vector2{X: 2, Y: 2}})
	s.add(spr.Primitive{Shape: spr.ShapeLine, From: geom.Vector2{X: -10, Y: 0}, To: geom.Vector2{X: 30, Y: 0}}, core.Transform{})
	s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 6, Y: 6}, Style: spr.Style{Outline: true, StrokeWidth: 2}}, core.Transform{})

	items := wantItems(t, s.collect(0.5), 4)
	var gotRect, gotCircle, gotLine, gotOutlinedRect bool
	for _, item := range items {
		p := shapePayload(t, item)
		switch shape := p.Shape.(type) {
		case spr.RectShape:
			if shape.Size == (geom.Vector2{X: 10, Y: 20}) {
				if !item.Position.AlmostEqual(geom.Vector2{X: 5, Y: 0}, 1e-9) {
					t.Errorf("rect position = %v, want interpolated (5, 0)", item.Position)
				}
				if item.Color != color.Black {
					t.Errorf("rect color = %v, want the black default for a nil color", item.Color)
				}
				if p.Outline || p.StrokeWidth != 0 {
					t.Errorf("filled rect outline/width = %v/%v, want false/0", p.Outline, p.StrokeWidth)
				}
				gotRect = true
			} else {
				if !p.Outline || p.StrokeWidth != 2 {
					t.Errorf("outlined rect outline/width = %v/%v, want true/2", p.Outline, p.StrokeWidth)
				}
				gotOutlinedRect = true
			}
		case spr.CircleShape:
			if shape.Radii != (geom.Vector2{X: 5, Y: 5}) {
				t.Errorf("circle shape = %v, want radii (5, 5)", p.Shape)
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
				t.Errorf("line shape = %v, want From (-10, 0) To (30, 0)", p.Shape)
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

// --- text --------------------------------------------------------------------

func TestCollectText(t *testing.T) {
	t.Run("stages the text payload with its measured bounds", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{X: 50, Y: 50}, 1)
		s.text("score: 1", geom.Vector2{X: 60, Y: 50}, nil)

		item := wantItems(t, s.collect(0.5), 1)[0]
		p := textPayload(t, item)
		if p.Text != "score: 1" || p.Font != "fonts/go.ttf" || p.Size != 16 {
			t.Errorf("item text fields = %q %q %v", p.Text, p.Font, p.Size)
		}
		if p.Bounds.X <= 0 || p.Bounds.Y <= 0 {
			t.Errorf("measured bounds = %v, want positive", p.Bounds)
		}
		if item.Position != (geom.Vector2{X: 60, Y: 50}) {
			t.Errorf("position = %v, want (60, 50)", item.Position)
		}
		if item.Color != color.White {
			t.Errorf("text color = %v, want the white default", item.Color)
		}
	})

	t.Run("keeps the sprite color", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{X: 50, Y: 50}, 1)
		s.text("tinted", geom.Vector2{X: 50, Y: 50}, &color.RGBA{R: 200, G: 40, B: 40, A: 255})

		item := wantItems(t, s.collect(0.5), 1)[0]
		tinted, ok := item.Color.(color.RGBA)
		if !ok || tinted != (color.RGBA{R: 200, G: 40, B: 40, A: 255}) {
			t.Errorf("text color = %v, want the sprite's own color", item.Color)
		}
	})

	t.Run("a missing font fails and names the font", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{X: 50, Y: 50}, 1)
		s.add(spr.Text{Asset: "fonts/ghost.ttf", Text: "haunted", TextSize: 16},
			core.Transform{Position: geom.Vector2{X: 50, Y: 50}})
		if _, err := s.tryCollect(0.5); err == nil || !strings.Contains(err.Error(), "fonts/ghost.ttf") {
			t.Errorf("missing font error = %v, want an error naming the font", err)
		}
	})
}

// --- sorting -----------------------------------------------------------------

func TestCollectSorting(t *testing.T) {
	t.Run("layer, then sort order, then world Y", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		at := func(x, y float64, style spr.Style) {
			s.texture("tex.png", geom.Vector2{X: x, Y: y}, style)
		}
		at(1, -10, spr.Style{SortOrder: 1})
		at(2, 0, spr.Style{Layer: 2})
		at(3, -10, spr.Style{Layer: 1})
		at(4, -10, spr.Style{})
		at(5, 10, spr.Style{SortOrder: -1})
		at(6, 10, spr.Style{})
		at(7, 0, spr.Style{Layer: 2})

		want := []float64{5, 4, 6, 1, 3, 2, 7}
		items := wantItems(t, s.collect(0), len(want))
		for i, item := range items {
			if item.Position.X != want[i] {
				t.Errorf("Items[%d].Position.X = %v, want %v", i, item.Position.X, want[i])
			}
		}
	})

	t.Run("shapes interleave with sprites by the same keys", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.texture("tex.png", geom.Vector2{Y: 10}, spr.Style{Layer: 1})
		s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 4}}, core.Transform{Position: geom.Vector2{Y: -10}})
		s.add(spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 2, Y: 2}, Style: spr.Style{Layer: 1}},
			core.Transform{Position: geom.Vector2{Y: -10}})
		s.add(spr.Primitive{Shape: spr.ShapeLine, To: geom.Vector2{X: 8, Y: 0}, Style: spr.Style{Layer: 1}},
			core.Transform{Position: geom.Vector2{Y: 10}})

		want := []struct {
			texture asset.ID
			shape   spr.DrawShape
		}{
			{"", spr.RectShape{Size: geom.Vector2{X: 4, Y: 4}}},
			{"", spr.CircleShape{Radii: geom.Vector2{X: 2, Y: 2}}},
			{"tex.png", nil},
			{"", spr.LineShape{To: geom.Vector2{X: 8, Y: 0}}},
		}
		items := wantItems(t, s.collect(0), len(want))
		for i := range items {
			if want[i].texture != "" {
				if p := texturePayload(t, items[i]); p.Texture != want[i].texture {
					t.Errorf("Items[%d].Texture = %q, want %q", i, p.Texture, want[i].texture)
				}
				continue
			}
			if p := shapePayload(t, items[i]); p.Shape != want[i].shape {
				t.Errorf("Items[%d].Shape = %v, want %v", i, p.Shape, want[i].shape)
			}
		}
	})

	t.Run("one entity with every visual class stages in class order", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		tr := core.Transform{Position: geom.Vector2{X: 20, Y: 20}, Scale: unitScale}
		s.spawn(tr, core.PrevTransform{Position: tr.Position, Scale: unitScale},
			spr.Sprite{Asset: "tex.png"},
			spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 4, Y: 4}, Style: spr.Style{Layer: 1}},
			spr.Text{Asset: "fonts/go.ttf", Text: "badge", TextSize: 8, Style: spr.Style{Layer: 2}},
		)

		items := wantItems(t, s.collect(1), 3)
		if p := texturePayload(t, items[0]); p.Texture != "tex.png" {
			t.Errorf("first item texture = %q, want the image class first", p.Texture)
		}
		if p := shapePayload(t, items[1]); p.Shape != (spr.RectShape{Size: geom.Vector2{X: 4, Y: 4}}) {
			t.Errorf("second item shape = %v, want the primitive class second", p.Shape)
		}
		if p := textPayload(t, items[2]); p.Text != "badge" {
			t.Errorf("third item text = %q, want the text class third", p.Text)
		}
	})

	t.Run("sprite and animator layers on one entity", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1).withAnimatorSets()
		tr := core.Transform{Position: geom.Vector2{}, Scale: unitScale}
		s.spawn(tr, core.PrevTransform{Position: tr.Position, Scale: unitScale},
			newTestAnimator(s.t),
			spr.Sprite{Asset: "tex.png"},
		)
		s.startAnimation(time.Millisecond)

		// The static image stages first, the animator's layers after it.
		items := wantItems(t, s.collect(0), 3)
		if p := texturePayload(t, items[0]); p.Texture != "tex.png" || p.Rect != (image.Rectangle{}) {
			t.Errorf("first item = %q/%v, want the static image first", p.Texture, p.Rect)
		}
	})
}

// --- animators ---------------------------------------------------------------

func TestCollectAnimators(t *testing.T) {
	t.Run("draws each weighted layer", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1).withAnimatorSets()
		s.add(newTestAnimator(s.t), core.Transform{Scale: unitScale})
		s.startAnimation(time.Millisecond)

		items := wantItems(t, s.collect(0), 2)
		if body := texturePayload(t, items[0]); body.Texture != "tex.png" || body.Rect != image.Rect(0, 0, 2, 2) {
			t.Errorf("body = %q/%v, want tex.png player region", body.Texture, body.Rect)
		}
		if overlay := texturePayload(t, items[1]); overlay.Texture != "tex.png" || overlay.Rect != image.Rect(2, 2, 4, 4) {
			t.Errorf("overlay = %q/%v, want tex.png enemy region", overlay.Texture, overlay.Rect)
		}
	})

	t.Run("hidden animators stay excluded and the visible one keeps its style", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1).withAnimatorSets()
		visible := newTestAnimator(s.t)
		visible.Style.FlipH = true
		visible.Style.Color = &color.RGBA{R: 255, A: 255}
		s.add(visible, core.Transform{Scale: unitScale})
		hidden := newTestAnimator(s.t)
		hidden.Style.Hidden = true
		s.add(hidden, core.Transform{Scale: unitScale})
		s.startAnimation(time.Millisecond)

		items := wantItems(t, s.collect(0), 2)
		item := items[0]
		p := texturePayload(t, item)
		if p.Texture != "tex.png" || p.Rect != (image.Rect(0, 0, 2, 2)) {
			t.Errorf("texture/rect = %q/%v, want tex.png frame zero of the player region", p.Texture, p.Rect)
		}
		if !p.FlipH || item.Color != color.Color(color.RGBA{R: 255, A: 255}) {
			t.Errorf("flip/color = %v/%v, want the animator's own style", p.FlipH, item.Color)
		}
	})

	t.Run("a paused animator draws frame zero", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1).withAnimatorSets()
		paused := newTestAnimator(s.t)
		paused.Paused = true
		s.add(paused, core.Transform{Scale: unitScale})
		s.startAnimation(100 * time.Millisecond)

		items := wantItems(t, s.collect(0), 2)
		if p := texturePayload(t, items[0]); p.Texture != "tex.png" || p.Rect != (image.Rect(0, 0, 2, 2)) {
			t.Errorf("texture/rect = %q/%v, want frame zero drawn while paused", p.Texture, p.Rect)
		}
	})

	t.Run("an unstarted animator draws nothing", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		// No Sets resource and no system run: an unstarted animator draws nothing.
		s.add(newTestAnimator(s.t), core.Transform{Scale: unitScale})
		list, err := s.tryCollect(0)
		if err != nil {
			t.Fatalf("Collect = %v, want no error for an unstarted animator", err)
		}
		if len(list.Items) != 0 {
			t.Fatalf("Items = %d, want 0", len(list.Items))
		}
	})

	t.Run("an invalid animator config surfaces from Collect", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1).withAnimatorSets()
		s.add(newTestAnimator(s.t), core.Transform{Scale: unitScale})
		// The write path a system uses: read through the entry, mutate, write
		// back. Entry.SetComponent skips component validation.
		for e := range core.NewQuery(s.w).With(animation.Animator{}).Execute() {
			a, _ := e.Component[animation.Animator]()
			a.Speed = -1
			e.SetComponent(a)
		}
		if _, err := s.tryCollect(0); err == nil || !strings.Contains(err.Error(), "speed") {
			t.Fatalf("Collect error = %v, want the invalid speed surfaced", err)
		}
	})
}

// --- errors ------------------------------------------------------------------

func TestCollectErrors(t *testing.T) {
	cases := []struct {
		name  string
		spawn func(s *scene)
		want  []string
	}{
		{"unloadable texture", func(s *scene) {
			s.texture("missing.png", geom.Vector2{}, spr.Style{})
		}, []string{"missing.png"}},
		{"missing sheet", func(s *scene) {
			s.add(spr.Sprite{Asset: "ghost.json", Region: "player"}, core.Transform{})
		}, []string{"ghost"}},
		{"unknown region", func(s *scene) {
			s.add(spr.Sprite{Asset: "sprites.json", Region: "boss"}, core.Transform{})
		}, []string{"boss", "tex.png"}},
		{"out-of-bounds sheet region", func(s *scene) {
			s.add(spr.Sprite{Asset: "oversized.json", Region: "player"}, core.Transform{})
		}, []string{"player", "tex.png", "outside"}},
		{"missing sheet texture", func(s *scene) {
			s.add(spr.Sprite{Asset: "missing-texture.json", Region: "player"}, core.Transform{})
		}, []string{"player", "missing.png"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newScene(t).camera(geom.Vector2{}, 1)
			tc.spawn(s)

			list, err := s.tryCollect(0)
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

// --- buffers -----------------------------------------------------------------

func TestCollectBuffers(t *testing.T) {
	s := newScene(t).camera(geom.Vector2{}, 1)
	first := s.spawn(core.Transform{Scale: unitScale}, core.PrevTransform{Scale: unitScale}, spr.Sprite{Asset: "tex.png"})
	s.add(spr.Sprite{Asset: "tex2.png"}, core.Transform{})

	c := s.collector()
	firstList := s.collectWith(c, 0)
	if p := texturePayload(t, wantItems(t, firstList, 2)[0]); p.Texture != "tex.png" {
		t.Fatalf("first Collect: first texture = %q, want tex.png", p.Texture)
	}

	if err := s.w.DestroyEntity(first); err != nil {
		t.Fatalf("destroy first sprite: %v", err)
	}
	second := s.collectWith(c, 0)
	if p := texturePayload(t, wantItems(t, second, 1)[0]); p.Texture != "tex2.png" {
		t.Errorf("second Collect: texture = %q, want tex2.png with no stale entry", p.Texture)
	}
	// The retained list views the same buffer: the second collect
	// overwrote its first item.
	if p := texturePayload(t, firstList.Items[0]); p.Texture != "tex2.png" {
		t.Errorf("retained list texture = %q, want tex2.png — the buffer was reused underneath it", p.Texture)
	}
}

// --- hidden or empty drawables -------------------------------------------------

func TestCollectSkipsHiddenOrEmpty(t *testing.T) {
	t.Run("hidden texture sprite", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.texture("tex.png", geom.Vector2{}, spr.Style{})
		s.texture("tex2.png", geom.Vector2{}, spr.Style{Hidden: true})
		item := wantItems(t, s.collect(0), 1)[0]
		if p := texturePayload(t, item); p.Texture != "tex.png" {
			t.Errorf("survivor texture = %q, want tex.png", p.Texture)
		}
	})

	t.Run("hidden shape", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
		s.add(spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}, Style: spr.Style{Hidden: true}}, core.Transform{})
		item := wantItems(t, s.collect(0), 1)[0]
		if p := shapePayload(t, item); p.Shape != (spr.RectShape{Size: geom.Vector2{X: 10, Y: 10}}) {
			t.Errorf("survivor shape = %v, want the rect", p.Shape)
		}
	})

	t.Run("sprite without an asset is legal and not drawn", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{}, 1)
		s.add(spr.Sprite{Style: spr.Style{Layer: 3}}, core.Transform{})
		s.add(spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 10}}, core.Transform{})
		item := wantItems(t, s.collect(0), 1)[0]
		if p := shapePayload(t, item); p.Shape != (spr.RectShape{Size: geom.Vector2{X: 10, Y: 10}}) {
			t.Errorf("survivor shape = %v, want the rect", p.Shape)
		}
	})

	t.Run("culled text", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{X: 50, Y: 50}, 1)
		s.text("off screen", geom.Vector2{X: 500, Y: 500}, nil)
		wantItems(t, s.collect(0), 0)
	})

	t.Run("empty text", func(t *testing.T) {
		s := newScene(t).camera(geom.Vector2{X: 50, Y: 50}, 1)
		s.text("", geom.Vector2{X: 50, Y: 50}, nil)
		wantItems(t, s.collect(0), 0)
	})
}
