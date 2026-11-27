package ebitrun

import (
	"math"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	spr "github.com/Leonard-Atorough/castrum/sprite"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/goregular"
)

var spriteScale = geom.Vector2{X: 1, Y: 1}

// --- harness ---------------------------------------------------------------

// game is a test harness for engine-draw tests: a game wired the way a real
// launch wires it - runner constructor provides the asset pipeline over the
// test filesystem and publishes the logical resolution - with a primary
// camera at the origin. Spawning is left to each test.
type game struct {
	t *testing.T
	g *castrum.Game
	r *Runner
}

func newGame(t *testing.T) *game {
	t.Helper()
	files := newAssetTestFS(t)
	files["fonts/go.ttf"] = &fstest.MapFile{Data: goregular.TTF}
	g := castrum.New(castrum.WithFilesystem(files))
	r := mustRunner(t, g)
	if _, err := g.World().NewEntity(
		render.Camera{Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: spriteScale},
	); err != nil {
		t.Fatalf("spawn camera: %v", err)
	}
	return &game{t: t, g: g, r: r}
}

// newSpriteGame keeps the older two-value call shape for provider tests.
func newSpriteGame(t *testing.T) (*castrum.Game, *Runner) {
	h := newGame(t)
	return h.g, h.r
}

// spawn adds an entity with one visual component; a zero scale reads as unit.
func (h *game) spawn(visual any, tr core.Transform) *game {
	h.t.Helper()
	if tr.Scale == (geom.Vector2{}) {
		tr.Scale = spriteScale
	}
	if _, err := h.g.World().NewEntity(visual, tr); err != nil {
		h.t.Fatalf("spawn %T: %v", visual, err)
	}
	return h
}

func (h *game) sprite(id asset.ID, at geom.Vector2, style spr.Style) *game {
	return h.spawn(spr.Sprite{Asset: id, Style: style}, core.Transform{Position: at})
}

func (h *game) text(str string, at geom.Vector2, style spr.Style) *game {
	return h.spawn(spr.Text{Asset: "fonts/go.ttf", Text: str, TextSize: 16, Style: style},
		core.Transform{Position: at})
}

func (h *game) shape(p spr.Primitive, at geom.Vector2) *game {
	return h.spawn(p, core.Transform{Position: at})
}

// engine runs the engine draw against a fresh 100x100 offscreen image.
func (h *game) engine(alpha float64) error {
	return h.r.engine(h.g.Context(), ebiten.NewImage(100, 100), alpha)
}

// --- engine draw -------------------------------------------------------------

// Headless proof limits, verified 2026-09-24: ebiten.NewImage and
// DrawImage run without a running game, but pixel reads panic
// ("ReadPixels cannot be called before the game starts"). So the blit
// is smoke-testable and error-path testable, never pixel-assertable;
// the camera projection math lives in core and is proven there.
func TestEngineDrawSmoke(t *testing.T) {
	t.Run("every payload kind", func(t *testing.T) {
		h := newGame(t)
		h.sprite("tex.png", geom.Vector2{}, spr.Style{})
		h.spawn(spr.Sprite{Asset: "sprites.json", Region: "player"}, core.Transform{})
		h.text("score", geom.Vector2{}, spr.Style{})
		if err := h.engine(0); err != nil {
			t.Fatalf("engine draw smoke: %v", err)
		}
	})

	t.Run("shapes filled, outlined, and rotated", func(t *testing.T) {
		h := newGame(t)
		shapes := []spr.Primitive{
			{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 10, Y: 20}},
			{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 6, Y: 6}, Style: spr.Style{Outline: true, StrokeWidth: 2}},
			{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}},
			{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 5, Y: 5}, Style: spr.Style{Outline: true, StrokeWidth: 1}},
			{Shape: spr.ShapeLine, To: geom.Vector2{X: 30, Y: 0}, Style: spr.Style{Outline: true, StrokeWidth: 2}},
		}
		for i, p := range shapes {
			p.Style.Layer = uint8(i)
			h.shape(p, geom.Vector2{X: 50, Y: 50})
		}
		// A rotated ellipse declares its own unequal radii and rides the
		// Béziers.
		h.spawn(spr.Primitive{
			Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 15, Y: 5},
			Style: spr.Style{Outline: true, StrokeWidth: 1, Layer: 5},
		}, core.Transform{Position: geom.Vector2{X: 50, Y: 50}, Rotation: math.Pi / 4})

		if err := h.engine(0); err != nil {
			t.Fatalf("engine draw shape smoke: %v", err)
		}
	})
}

func TestEngineDrawErrors(t *testing.T) {
	t.Run("an unloadable texture fails fast and names the handle", func(t *testing.T) {
		h := newGame(t)
		h.sprite("missing.png", geom.Vector2{}, spr.Style{})
		err := h.engine(0)
		if err == nil {
			t.Fatal("engine draw should fail fast on an unloadable texture")
		}
		if !strings.Contains(err.Error(), "missing.png") {
			t.Errorf("error %q should name the texture", err)
		}
	})

	// The engine draw must visit every collected item, not stop after the
	// first. The good texture sprite sorts first and draws fine; the text
	// item second collects fine (the world loader has the font) but its
	// draw fails through a font provider wired to a font-less loader - so
	// the error only surfaces if the loop continues past the first item.
	t.Run("a later item's draw failure is still reported", func(t *testing.T) {
		h := newGame(t)
		h.sprite("tex.png", geom.Vector2{}, spr.Style{SortOrder: -1})
		h.text("score", geom.Vector2{}, spr.Style{SortOrder: 1})

		provider, err := h.g.World().Resource[*TextureProvider]()
		if err != nil {
			t.Fatal(err)
		}
		emptyLoader, err := asset.NewLoader(fstest.MapFS{}, asset.LoaderOptions{})
		if err != nil {
			t.Fatal(err)
		}
		fonts := newFontProvider(emptyLoader)

		draw := newEngineDrawFunc(render.NewCollector(h.g.World()), provider, fonts)
		err = draw(h.g.Context(), ebiten.NewImage(100, 100), 0)
		if err == nil {
			t.Fatal("engine draw should reach the second item and fail on its font")
		}
		if !strings.Contains(err.Error(), "fonts/go.ttf") {
			t.Errorf("error %q should name the font", err)
		}
	})

	// Runner wiring: the engine draw runs before user draws, its errors are
	// stored with the "engine draw" prefix and surface from Update, and a
	// user draw still ran despite the engine failure. The screen can be
	// nil here - the fail-fast error returns before any DrawImage call.
	t.Run("the engine error is prefixed and user draws still run", func(t *testing.T) {
		h := newGame(t)
		h.sprite("missing.png", geom.Vector2{}, spr.Style{})

		ran := false
		h.r.AddDraw(func(ctx *core.Context, screen *ebiten.Image, alpha float64) error {
			ran = true
			return nil
		})
		h.r.Draw(nil)
		if !ran {
			t.Fatal("user draws must still run when the engine draw fails")
		}

		err := h.r.Update()
		if err == nil {
			t.Fatal("the stored engine draw error must surface from Update")
		}
		if !strings.Contains(err.Error(), "engine draw") {
			t.Errorf("error %q should carry the engine draw prefix", err)
		}
	})
}

// --- shape projection --------------------------------------------------------
// The shape projection helpers are the testable seam — pixel reads are
// impossible headless, so the math lives here and the vector calls
// stay smoke-only.

func TestRectCorners(t *testing.T) {
	center := geom.Vector2{X: 50, Y: 50}
	unit := geom.Vector2{X: 1, Y: 1}

	for _, tc := range []struct {
		name           string
		size, scale    geom.Vector2
		zoom, rotation float64
		want           [4]geom.Vector2
	}{
		{"axis aligned", geom.Vector2{X: 10, Y: 20}, unit, 1, 0, [4]geom.Vector2{
			{X: 45, Y: 40}, {X: 55, Y: 40}, {X: 55, Y: 60}, {X: 45, Y: 60},
		}},
		// A quarter turn swaps the extents rigidly around the center.
		{"quarter turn", geom.Vector2{X: 10, Y: 20}, unit, 1, math.Pi / 2, [4]geom.Vector2{
			{X: 60, Y: 45}, {X: 60, Y: 55}, {X: 40, Y: 55}, {X: 40, Y: 45},
		}},
		// Zoom scales the extents from the snapped center.
		{"zoom", geom.Vector2{X: 10, Y: 10}, unit, 2, 0, [4]geom.Vector2{
			{X: 40, Y: 40}, {X: 60, Y: 40}, {X: 60, Y: 60}, {X: 40, Y: 60},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := rectCorners(center, tc.size, tc.scale, tc.zoom, tc.rotation)
			for i := range tc.want {
				if !got[i].AlmostEqual(tc.want[i], 1e-9) {
					t.Errorf("corner %d = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestEllipsePoints(t *testing.T) {
	center := geom.Vector2{X: 50, Y: 50}
	unit := geom.Vector2{X: 1, Y: 1}
	circle := geom.Vector2{X: 10, Y: 10}
	k := 4 * (math.Sqrt(2) - 1) / 3

	for _, tc := range []struct {
		name           string
		radii, scale   geom.Vector2
		zoom, rotation float64
		// want maps a point index to its expected position: anchors sit
		// at 3*i, the first control at 1.
		want map[int]geom.Vector2
	}{
		// Equal radii, uniform scale: the anchors sit on the axes at the
		// radius, the first controls at the kappa offsets.
		{"circle anchors and kappa control", circle, unit, 1, 0, map[int]geom.Vector2{
			0: {X: 60, Y: 50}, 3: {X: 50, Y: 60}, 6: {X: 40, Y: 50}, 9: {X: 50, Y: 40},
			1: {X: 60, Y: 50 + 10*k},
		}},
		// Unequal radii draw an ellipse: each axis anchor carries its own
		// radius - the geometry the shape declares.
		{"ellipse per-axis radii", geom.Vector2{X: 20, Y: 10}, unit, 1, 0, map[int]geom.Vector2{
			0: {X: 70, Y: 50}, 3: {X: 50, Y: 60},
		}},
		// The transform's scale composes with the radii per axis.
		{"scaled per axis", circle, geom.Vector2{X: 2, Y: 1}, 1, 0, map[int]geom.Vector2{
			0: {X: 70, Y: 50},
		}},
		// Mirrored scale is not a size: the extents stay positive.
		{"mirrored scale", geom.Vector2{X: 20, Y: 10}, geom.Vector2{X: -2, Y: 1}, 1, 0, map[int]geom.Vector2{
			0: {X: 90, Y: 50},
		}},
		// A quarter turn swaps the axes rigidly around the center.
		{"quarter turn", circle, unit, 1, math.Pi / 2, map[int]geom.Vector2{
			0: {X: 50, Y: 60}, 3: {X: 40, Y: 50},
		}},
		// Zoom scales the extents from the snapped center.
		{"zoom", circle, unit, 2, 0, map[int]geom.Vector2{
			0: {X: 70, Y: 50},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ellipsePoints(center, tc.radii, tc.scale, tc.zoom, tc.rotation)
			for i, want := range tc.want {
				if !got[i].AlmostEqual(want, 1e-9) {
					t.Errorf("point %d = %v, want %v", i, got[i], want)
				}
			}
		})
	}
}

func TestOffsetPoint(t *testing.T) {
	start := geom.Vector2{X: 10, Y: 10}
	unit := geom.Vector2{X: 1, Y: 1}

	for _, tc := range []struct {
		name           string
		offset, scale  geom.Vector2
		zoom, rotation float64
		want           geom.Vector2
	}{
		{"identity", geom.Vector2{X: 30, Y: 0}, unit, 1, 0, geom.Vector2{X: 40, Y: 10}},
		// A quarter turn points the offset down: screen Y grows.
		{"quarter turn", geom.Vector2{X: 30, Y: 0}, unit, 1, math.Pi / 2, geom.Vector2{X: 10, Y: 40}},
		{"scaled", geom.Vector2{X: 30, Y: 0}, geom.Vector2{X: 2, Y: 2}, 1, 0, geom.Vector2{X: 70, Y: 10}},
		// A line's From maps symmetrically: both endpoints route through
		// the same projection from one snapped position.
		{"negative offset", geom.Vector2{X: -30, Y: 0}, unit, 1, 0, geom.Vector2{X: -20, Y: 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := offsetPoint(start, tc.offset, tc.scale, tc.zoom, tc.rotation)
			if !got.AlmostEqual(tc.want, 1e-9) {
				t.Errorf("offset point = %v, want %v", got, tc.want)
			}
		})
	}
}
