package ebitrun

import (
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
		core.TextureSprite{Texture: "missing.png"},
		core.Sprite{},
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
		core.TextureSprite{Texture: "tex.png"},
		core.Sprite{},
		core.Transform{Position: geom.Vector2{X: 0, Y: 0}, Scale: spriteScale},
		core.PrevTransform{Position: geom.Vector2{X: 0, Y: 0}},
	); err != nil {
		t.Fatalf("spawn texture sprite: %v", err)
	}
	if _, err := g.World().NewEntity(
		core.AtlasSprite{Atlas: "sprites", Region: "player"},
		core.Sprite{},
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
		core.TextureSprite{Texture: "missing.png"},
		core.Sprite{},
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
