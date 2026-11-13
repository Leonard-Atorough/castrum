// Command wander demonstrates the engine renderer: a sheet-region sprite
// wanders to random screen points at fixed ticks, the engine renders
// it interpolated between ticks, and the debug overlay draws its
// rates above the engine-rendered world.
//
// Assets load from an embedded filesystem, passed via
// castrum.WithFilesystem for a single-binary game.
//
// Run from this directory:
//
//	go run .
package main

import (
	"embed"
	"math/rand"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/spritesheet"
)

//go:embed Dungeon_Character_2.png
var files embed.FS

const (
	// The sheet is 112x32 pixels of 16x16 tiles: 7 across, 2 down, so
	// the grid defines char_0 through char_13. Any region works.
	region = "char_0"
	speed  = 120 // world units per second

	// The camera sits at the center of the 1280x720 logical resolution
	// at zoom 1, so world coordinates are screen coordinates and every
	// random point lands on screen.
	screenW, screenH = 1280, 720
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(castrum.WithTitle("castrum — wander"), castrum.WithFilesystem(files))
	if err != nil {
		return err
	}

	// Slice the sheet at setup, before the window opens: a bad path
	// or uneven tile division fails the launch, not the first frame.
	// The engine provides the loader at New.
	texture, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData]("Dungeon_Character_2.png")
	if err != nil {
		return err
	}
	sheet, err := spritesheet.Grid("Dungeon_Character_2.png", texture.Image.Bounds(), 16, 16, "char")
	if err != nil {
		return err
	}

	// The engine camera frames the world from the screen's center.
	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// The sprite's zero-value style is shown and opaque; the Drawable
	// carries the picture.
	sprite, err := g.World().NewEntity(
		render.Sprite{Drawable: render.SheetSource{Sheet: asset.MemorySheet(sheet), Region: region}},
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: 4, Y: 4}, // 16px tiles read better at 64px
		},
	)
	if err != nil {
		return err
	}

	// Wander: each fixed tick moves the sprite toward its target at a
	// constant speed; reaching it picks a new random one. The engine's
	// prev-transform capture and the renderer's interpolation make the
	// motion smooth without any per-frame work here.
	if err := g.AddSystem(core.PhaseFixed, "wander", wanderSystem(sprite, randomPoint)); err != nil {
		return err
	}

	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}

func wanderSystem(sprite *core.Entity, nextTarget func() geom.Vector2) core.System {
	target := nextTarget()
	return core.SystemFunc(func(ctx *core.Context) error {
		return sprite.Update(ctx.World, func(t *core.Transform) {
			delta := target.Sub(t.Position)
			distance := delta.Length()
			step := speed * ctx.DeltaTime.Seconds()
			if distance <= step {
				t.Position = target
				target = nextTarget()
			} else {
				t.Position = t.Position.Add(delta.Mul(step / distance))
			}
		})
	})
}

// randomPoint picks a world point; with the camera framing the screen,
// every point is visible.
func randomPoint() geom.Vector2 {
	return geom.Vector2{
		X: rand.Float64() * screenW,
		Y: rand.Float64() * screenH,
	}
}
