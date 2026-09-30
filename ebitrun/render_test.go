package ebitrun

import (
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
// the camera projection math lives in core and is proven there.

var spriteScale = geom.Vector2{X: 1, Y: 1}

// newSpriteGame wires a game the way a real launch does - runner
// constructor provides the asset pipeline over the test filesystem and
// publishes the logical resolution - and spawns a primary camera at
// the origin. Sprite spawning is left to each test.
func newSpriteGame(t *testing.T) (*castrum.Game, *Runner) {
	t.Helper()
	g, err := castrum.New(castrum.WithFilesystem(newAssetTestFS(t)))
	if err != nil {
		t.Fatalf("castrum.New: %v", err)
	}
	r, err := New(g)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := g.World().NewEntity(
		core.Camera{Zoom: 1, Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: spriteScale},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	return g, r
}

func TestEngineDrawFailsFast(t *testing.T) {
	g, r := newSpriteGame(t)
	if _, err := g.World().NewEntity(
		core.Sprite{Drawable: core.TextureSource{Texture: "missing.png"}},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
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
	); err != nil {
		t.Fatalf("spawn texture sprite: %v", err)
	}
	if _, err := g.World().NewEntity(
		core.Sprite{Drawable: core.AtlasSource{Atlas: "sprites", Region: "player"}},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
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

func TestOffsetPoint(t *testing.T) {
	start := geom.Vector2{X: 10, Y: 10}
	unit := geom.Vector2{X: 1, Y: 1}

	end := offsetPoint(start, geom.Vector2{X: 30, Y: 0}, unit, 1, 0)
	if end != (geom.Vector2{X: 40, Y: 10}) {
		t.Errorf("offset point = %v, want (40, 10)", end)
	}
	// A quarter turn points the offset down: screen Y grows.
	rotated := offsetPoint(start, geom.Vector2{X: 30, Y: 0}, unit, 1, math.Pi/2)
	if !rotated.AlmostEqual(geom.Vector2{X: 10, Y: 40}, 1e-9) {
		t.Errorf("rotated offset point = %v, want (10, 40)", rotated)
	}
	scaled := offsetPoint(start, geom.Vector2{X: 30, Y: 0}, geom.Vector2{X: 2, Y: 2}, 1, 0)
	if scaled != (geom.Vector2{X: 70, Y: 10}) {
		t.Errorf("scaled offset point = %v, want (70, 10)", scaled)
	}
	// A line's From maps symmetrically: both endpoints route through
	// the same projection from one snapped position.
	from := offsetPoint(start, geom.Vector2{X: -30, Y: 0}, unit, 1, 0)
	if from != (geom.Vector2{X: -20, Y: 10}) {
		t.Errorf("negative offset point = %v, want (-20, 10)", from)
	}
}

func TestEllipsePoints(t *testing.T) {
	center := geom.Vector2{X: 50, Y: 50}
	unit := geom.Vector2{X: 1, Y: 1}
	circle := geom.Vector2{X: 10, Y: 10}
	k := 4 * (math.Sqrt(2) - 1) / 3

	// Equal radii, uniform scale: the anchors sit on the axes at the
	// radius, the first controls at the kappa offsets.
	got := ellipsePoints(center, circle, unit, 1, 0)
	anchors := [4]geom.Vector2{{X: 60, Y: 50}, {X: 50, Y: 60}, {X: 40, Y: 50}, {X: 50, Y: 40}}
	for i, want := range anchors {
		if !got[3*i].AlmostEqual(want, 1e-9) {
			t.Errorf("anchor %d = %v, want %v", i, got[3*i], want)
		}
	}
	if !got[1].AlmostEqual(geom.Vector2{X: 60, Y: 50 + 10*k}, 1e-9) {
		t.Errorf("first control = %v, want (60, %v)", got[1], 50+10*k)
	}

	// Unequal radii draw an ellipse: each axis anchor carries its own
	// radius - the geometry the shape declares.
	widened := ellipsePoints(center, geom.Vector2{X: 20, Y: 10}, unit, 1, 0)
	if !widened[0].AlmostEqual(geom.Vector2{X: 70, Y: 50}, 1e-9) {
		t.Errorf("wide X anchor = %v, want (70, 50)", widened[0])
	}
	if !widened[3].AlmostEqual(geom.Vector2{X: 50, Y: 60}, 1e-9) {
		t.Errorf("narrow Y anchor = %v, want (50, 60)", widened[3])
	}

	// The transform's scale composes with the radii per axis.
	scaled := ellipsePoints(center, circle, geom.Vector2{X: 2, Y: 1}, 1, 0)
	if !scaled[0].AlmostEqual(geom.Vector2{X: 70, Y: 50}, 1e-9) {
		t.Errorf("scaled X anchor = %v, want (70, 50)", scaled[0])
	}

	// Mirrored scale is not a size: the extents stay positive.
	mirrored := ellipsePoints(center, geom.Vector2{X: 20, Y: 10}, geom.Vector2{X: -2, Y: 1}, 1, 0)
	if !mirrored[0].AlmostEqual(geom.Vector2{X: 90, Y: 50}, 1e-9) {
		t.Errorf("mirrored X anchor = %v, want (90, 50)", mirrored[0])
	}

	// A quarter turn swaps the axes rigidly around the center.
	rotated := ellipsePoints(center, circle, unit, 1, math.Pi/2)
	if !rotated[0].AlmostEqual(geom.Vector2{X: 50, Y: 60}, 1e-9) {
		t.Errorf("rotated X anchor = %v, want (50, 60)", rotated[0])
	}
	if !rotated[3].AlmostEqual(geom.Vector2{X: 40, Y: 50}, 1e-9) {
		t.Errorf("rotated Y anchor = %v, want (40, 50)", rotated[3])
	}

	// Zoom scales the extents from the snapped center.
	zoomed := ellipsePoints(center, circle, unit, 2, 0)
	if !zoomed[0].AlmostEqual(geom.Vector2{X: 70, Y: 50}, 1e-9) {
		t.Errorf("zoomed X anchor = %v, want (70, 50)", zoomed[0])
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
		{Drawable: core.CircleShape{Radii: geom.Vector2{X: 5, Y: 5}}},
		{Drawable: core.CircleShape{Radii: geom.Vector2{X: 5, Y: 5}}, Outline: true, StrokeWidth: 1},
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
		); err != nil {
			t.Fatalf("spawn shape %d: %v", i, err)
		}
	}

	// A rotated ellipse declares its own unequal radii and rides the
	// Béziers.
	if _, err := g.World().NewEntity(
		core.Sprite{
			Drawable: core.CircleShape{Radii: geom.Vector2{X: 15, Y: 5}}, Outline: true, StrokeWidth: 1, Layer: 5,
		},
		core.Transform{
			Position: geom.Vector2{X: 50, Y: 50},
			Scale:    spriteScale,
			Rotation: math.Pi / 4,
		},
	); err != nil {
		t.Fatalf("spawn rotated ellipse: %v", err)
	}

	if err := r.engine(g.Context(), ebiten.NewImage(100, 100)); err != nil {
		t.Fatalf("engine draw shape smoke: %v", err)
	}
}
