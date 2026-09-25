package ebitrun

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/hajimehoshi/ebiten/v2"
)

// Headless proof limits, verified 2026-09-24: ebiten.NewImage and
// DrawImage run without a running game, but pixel reads panic
// ("ReadPixels cannot be called before the game starts"). So the blit
// is smoke-testable and error-path testable, never pixel-assertable;
// worldToScreen is where the camera math is proven.

var spriteScale = geom.Vector2{X: 1, Y: 1}

// newSpriteGame wires a game the way a real launch does - runner
// constructor provides the asset pipeline over the test filesystem and
// publishes the logical resolution - and spawns a primary camera at
// the origin. Sprite spawning is left to each test.
func newSpriteGame(t *testing.T) (*castrum.Game, *Runner) {
	t.Helper()
	g, err := castrum.New()
	if err != nil {
		t.Fatalf("castrum.New: %v", err)
	}
	r, err := New(g, WithFilesystem(newAssetTestFS(t)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := g.World().NewEntity(
		core.Camera{Zoom: 1, Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: spriteScale},
		core.PrevTransform{},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	return g, r
}

func TestWorldToScreen(t *testing.T) {
	for _, tc := range []struct {
		name          string
		world         geom.Vector2
		camera        core.CameraView
		width, height int
		want          geom.Vector2
	}{
		{
			name:   "camera at origin",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: core.CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  100, height: 100,
			want: geom.Vector2{X: 50, Y: 50},
		},
		{
			name:   "camera offset",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: core.CameraView{Zoom: 1, Position: geom.Vector2{X: 10, Y: 20}},
			width:  100, height: 100,
			want: geom.Vector2{X: 40, Y: 30},
		},
		{
			name:   "camera offset with zoom",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: core.CameraView{Zoom: 2, Position: geom.Vector2{X: 10, Y: 20}},
			width:  100, height: 100,
			want: geom.Vector2{X: 30, Y: 10},
		},
		{
			name:   "world equals camera",
			world:  geom.Vector2{X: 10, Y: 20},
			camera: core.CameraView{Zoom: 3, Position: geom.Vector2{X: 10, Y: 20}},
			width:  100, height: 100,
			want: geom.Vector2{X: 50, Y: 50},
		},
		{
			name:   "non-square target",
			world:  geom.Vector2{X: 0, Y: 0},
			camera: core.CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  200, height: 100,
			want: geom.Vector2{X: 100, Y: 50},
		},
		// Fractional world positions snap to whole pixels: the
		// pixel-grid policy against nearest-filter shimmer.
		{
			name:   "fractional position rounds to whole pixels",
			world:  geom.Vector2{X: 10.4, Y: 20.4},
			camera: core.CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  100, height: 100,
			want: geom.Vector2{X: 60, Y: 70},
		},
		{
			name:   "halves round away from zero",
			world:  geom.Vector2{X: 10.5, Y: -0.5},
			camera: core.CameraView{Zoom: 1, Position: geom.Vector2{X: 0, Y: 0}},
			width:  100, height: 100,
			want: geom.Vector2{X: 61, Y: 50},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := worldToScreen(tc.world, tc.camera, tc.width, tc.height)
			if !got.AlmostEqual(tc.want, 1e-9) {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// Fail-fast propagates: a sprite whose texture will not load makes
// Collect error before the first DrawImage, so the engine draw func
// returns the error without ever touching the screen.
func TestEngineDrawFailsFast(t *testing.T) {
	g, r := newSpriteGame(t)
	if _, err := g.World().NewEntity(
		core.Sprite{Drawable: core.TextureSource{Texture: "missing.png"}},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
		core.PrevTransform{Position: geom.Vector2{X: 0, Y: 0}},
	); err != nil {
		t.Fatalf("spawn sprite: %v", err)
	}

	err := r.engine(g.Context(), ebiten.NewImage(100, 100))
	if err == nil {
		t.Fatal("engine draw should fail fast on an unloadable texture")
	}
	if !strings.Contains(err.Error(), "missing.png") {
		t.Errorf("error %q should name the texture", err)
	}
}

// The happy path is a smoke test only: with a camera, a whole-texture
// sprite, and a registered atlas sprite, the engine draw runs against
// an offscreen image and returns nil. DrawImage works headless; pixel
// reads do not, so assert no error and no panic - nothing visual.
func TestEngineDrawSmoke(t *testing.T) {
	g, r := newSpriteGame(t)
	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("resolve asset server: %v", err)
	}
	if err := server.RegisterAtlasFromSidecar("sprites", "tex.png", "atlas.json"); err != nil {
		t.Fatalf("register atlas: %v", err)
	}

	if _, err := g.World().NewEntity(
		core.Sprite{Drawable: core.TextureSource{Texture: "tex.png"}},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
		core.PrevTransform{Position: geom.Vector2{X: 0, Y: 0}},
	); err != nil {
		t.Fatalf("spawn texture sprite: %v", err)
	}
	if _, err := g.World().NewEntity(
		core.Sprite{Drawable: core.AtlasSource{Atlas: "sprites", Region: "player"}},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
		core.PrevTransform{Position: geom.Vector2{X: 0, Y: 0}},
	); err != nil {
		t.Fatalf("spawn atlas sprite: %v", err)
	}

	if err := r.engine(g.Context(), ebiten.NewImage(100, 100)); err != nil {
		t.Fatalf("engine draw smoke: %v", err)
	}
}

// Runner wiring: the engine draw runs before user draws, its errors are
// stored with the "engine draw" prefix and surface from Update, and a
// user draw still ran despite the engine failure. The screen can be
// nil here - the fail-fast error returns before any DrawImage call.
func TestDrawEngineErrorPrefixedAndUserDrawsRun(t *testing.T) {
	g, r := newSpriteGame(t)
	if _, err := g.World().NewEntity(
		core.Sprite{Drawable: core.TextureSource{Texture: "missing.png"}},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
		core.PrevTransform{Position: geom.Vector2{X: 0, Y: 0}},
	); err != nil {
		t.Fatalf("spawn sprite: %v", err)
	}

	ran := false
	r.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ran = true
		return nil
	})
	r.Draw(nil)
	if !ran {
		t.Fatal("user draws must still run when the engine draw fails")
	}

	err := r.Update()
	if err == nil {
		t.Fatal("the stored engine draw error must surface from Update")
	}
	if !strings.Contains(err.Error(), "engine draw") {
		t.Errorf("error %q should carry the engine draw prefix", err)
	}
}

// The shape projection helpers are the testable seam — pixel reads are
// impossible headless, so the math lives here and the vector calls
// stay smoke-only.
func TestRectCorners(t *testing.T) {
	center := geom.Vector2{X: 50, Y: 50}
	unit := geom.Vector2{X: 1, Y: 1}

	// Axis-aligned: corners at half the scaled size.
	got := rectCorners(center, geom.Vector2{X: 10, Y: 20}, unit, 1, 0)
	want := [4]geom.Vector2{
		{X: 45, Y: 40}, {X: 55, Y: 40}, {X: 55, Y: 60}, {X: 45, Y: 60},
	}
	for i := range want {
		if !got[i].AlmostEqual(want[i], 1e-9) {
			t.Errorf("corner %d = %v, want %v", i, got[i], want[i])
		}
	}

	// A quarter turn swaps the extents rigidly around the center.
	rotated := rectCorners(center, geom.Vector2{X: 10, Y: 20}, unit, 1, math.Pi/2)
	wantRotated := [4]geom.Vector2{
		{X: 60, Y: 45}, {X: 60, Y: 55}, {X: 40, Y: 55}, {X: 40, Y: 45},
	}
	for i := range wantRotated {
		if !rotated[i].AlmostEqual(wantRotated[i], 1e-9) {
			t.Errorf("rotated corner %d = %v, want %v", i, rotated[i], wantRotated[i])
		}
	}

	// Zoom scales the extents from the snapped center.
	zoomed := rectCorners(center, geom.Vector2{X: 10, Y: 10}, unit, 2, 0)
	if zoomed[1] != (geom.Vector2{X: 60, Y: 40}) {
		t.Errorf("zoomed corner = %v, want (60, 40)", zoomed[1])
	}
}

func TestLineEnd(t *testing.T) {
	start := geom.Vector2{X: 10, Y: 10}
	unit := geom.Vector2{X: 1, Y: 1}

	end := lineEnd(start, geom.Vector2{X: 30, Y: 0}, unit, 1, 0)
	if end != (geom.Vector2{X: 40, Y: 10}) {
		t.Errorf("line end = %v, want (40, 10)", end)
	}
	// A quarter turn points the segment down: screen Y grows.
	rotated := lineEnd(start, geom.Vector2{X: 30, Y: 0}, unit, 1, math.Pi/2)
	if !rotated.AlmostEqual(geom.Vector2{X: 10, Y: 40}, 1e-9) {
		t.Errorf("rotated line end = %v, want (10, 40)", rotated)
	}
	scaled := lineEnd(start, geom.Vector2{X: 30, Y: 0}, geom.Vector2{X: 2, Y: 2}, 1, 0)
	if scaled != (geom.Vector2{X: 70, Y: 10}) {
		t.Errorf("scaled line end = %v, want (70, 10)", scaled)
	}
}

func TestCircleRadius(t *testing.T) {
	if r := circleRadius(5, geom.Vector2{}.X, 1); r != 0 {
		t.Errorf("zero scale radius = %v, want 0", r)
	}
	if r := circleRadius(5, 2, 1); r != 10 {
		t.Errorf("scaled radius = %v, want 10", r)
	}
	if r := circleRadius(5, -2, 1); r != 10 {
		t.Errorf("mirrored scale radius = %v, want 10 (mirroring is not a size)", r)
	}
}

func TestShapeColor(t *testing.T) {
	black := shapeColor(color.Black, 0)
	if black != (color.RGBA{R: 0, G: 0, B: 0, A: 255}) {
		t.Errorf("black shape color = %v, want opaque black", black)
	}
	red := shapeColor(color.RGBA{R: 255, A: 255}, 0.5)
	if red != (color.RGBA{R: 255, G: 0, B: 0, A: 127}) {
		t.Errorf("half-transparent red = %v, want alpha 127", red)
	}
	// A fully transparent tint carries no hue: invisible.
	if empty := shapeColor(color.RGBA{}, 0); empty != (color.RGBA{}) {
		t.Errorf("transparent tint = %v, want the zero color", empty)
	}
}

// The shape blit is a smoke test: all three shapes, filled and
// outlined, on and off center, against an offscreen image — no error,
// no panic. Nothing pixel-assertable headless.
func TestEngineDrawShapeSmoke(t *testing.T) {
	g, r := newSpriteGame(t)
	spawns := []core.Sprite{
		{Drawable: core.RectShape{Size: geom.Vector2{X: 10, Y: 20}}},
		{Drawable: core.RectShape{Size: geom.Vector2{X: 6, Y: 6}}, Outline: true, StrokeWidth: 2},
		{Drawable: core.CircleShape{Radius: 5}},
		{Drawable: core.CircleShape{Radius: 5}, Outline: true, StrokeWidth: 1},
		{Drawable: core.LineShape{To: geom.Vector2{X: 30, Y: 0}}, Outline: true, StrokeWidth: 2},
	}
	for i, sprite := range spawns {
		sprite.Layer = uint8(i)
		if _, err := g.World().NewEntity(
			sprite,
			core.Transform{
				Position: geom.Vector2{X: 50, Y: 50},
				Scale:    spriteScale,
			},
			core.PrevTransform{Position: geom.Vector2{X: 50, Y: 50}},
		); err != nil {
			t.Fatalf("spawn shape %d: %v", i, err)
		}
	}

	if err := r.engine(g.Context(), ebiten.NewImage(100, 100)); err != nil {
		t.Fatalf("engine draw shape smoke: %v", err)
	}
}
