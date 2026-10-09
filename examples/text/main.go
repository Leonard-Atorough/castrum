// Command text demonstrates text rendering: a static label, a
// per-tick counter whose string updates like a score display, and
// tinted and scaled text. Text is a drawable like any other - it
// lives on a Sprite with a Transform, so it culls, sorts, moves,
// and fades exactly like a shape or a texture.
//
// The font is a regular ttf loaded through the asset server from an
// embedded filesystem; nothing registers at draw time - the sprite
// names the font, the collector measures it, the runner draws it.
//
// Run from the repository root:
//
//	go run ./examples/text
package main

import (
	"embed"
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
)

//go:embed fonts/GoRegular.ttf
var fontFiles embed.FS

const (
	screenW, screenH = 1280, 720

	goFont = "fonts/GoRegular.ttf"

	// rowY places the three text rows; every row is a separate text
	// sprite, centered on its own position.
	titleY = 200
	scoreY = 360
	notesY = 520
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(
		castrum.WithTitle("castrum - text"),
		castrum.WithFilesystem(fontFiles),
	)
	if err != nil {
		return err
	}

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// A static label: white text - the nil-color default - centered
	// on the position like every other drawable.
	if _, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: titleY}, Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{Drawable: core.TextSource{Font: goFont, Text: "hello, castrum", Size: 48}},
	); err != nil {
		return err
	}

	// A score display: the same sprite, its string rewritten each
	// tick - text is plain component state, so updating it is an
	// update closure, nothing else.
	score, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: scoreY}, Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{Drawable: core.TextSource{Font: goFont, Text: "score: 0", Size: 32}},
	)
	if err != nil {
		return err
	}

	// Tinted, scaled text: the sprite color dyes the glyphs and the
	// transform scale stretches them, both exactly as for a shape.
	if _, err := g.World().NewEntity(
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: notesY},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		core.Sprite{
			Drawable: core.TextSource{Font: goFont, Text: "text culls, sorts, and fades like any drawable", Size: 24},
			Color:    color.NRGBA{R: 120, G: 200, B: 255, A: 255},
		},
	); err != nil {
		return err
	}

	// One fixed system rewrites the score's string every second; the
	// engine re-measures and re-draws it without any other wiring.
	elapsed := time.Duration(0)
	if err := g.AddSystem(core.PhaseFixed, "score", core.SystemFunc(func(ctx *core.Context) error {
		elapsed += ctx.DeltaTime
		return score.Update(ctx.World, func(s *core.Sprite) {
			s.Drawable = core.TextSource{Font: goFont, Text: fmt.Sprintf("score: %d", int(elapsed.Seconds())), Size: 32}
		})
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrintAt(screen,
			"the counter is one sprite whose string is rewritten each tick",
			60, screenH-40)
		return nil
	})

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}
