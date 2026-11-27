// Command textures demonstrates whole-texture sprites with styles and transforms.
//
// Each sprite uses the entire image at its pixel dimensions before scaling.
//
// The embedded texture is preloaded at setup, so a missing or malformed image
// fails at launch. See CREDITS.md for its source and license.
//
// The first row shows plain, tinted, faded, and mirrored sprites. The second
// shows spinning, stretched, rotated, mirrored, and tinted variants.
//
// Run from this directory:
//
//	go run .
package main

import (
	"embed"
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

//go:embed gopher.png
var files embed.FS

const (
	texturePath = "gopher.png"

	// rotationSpeed is radians per fixed tick.
	rotationSpeed = 0.05

	screenW, screenH = 1280, 720

	cellX0, cellDX = 170, 235
	rowY0, rowDY   = 230, 290

	// cellScale shrinks the 288x288 texture to a 216px cell.
	cellScale = 0.75
)

type cell struct {
	s     sprite.Sprite
	scale geom.Vector2
	spin  bool
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := newGame()
	if err != nil {
		return err
	}
	ebitrun.New(ebitrun.WithDebugOverlay()).Run(g)
	return nil
}

func newGame() (*castrum.Game, error) {
	g := castrum.New(
		castrum.WithTitle("castrum - textures"),
		castrum.WithFilesystem(files),
	)

	// Preload at setup: a missing or undecodable texture fails here, at
	// launch, rather than surfacing as a first-frame collect error.
	loader := g.World().MustResource[*asset.Loader]()
	texture, err := loader.Load[asset.TextureData](texturePath)
	if err != nil {
		return nil, err
	}

	// The engine camera frames the world from the screen's center, so
	// world coordinates are screen coordinates at zoom 1.
	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return nil, err
	}

	gold := &color.RGBA{R: 255, G: 200, B: 60, A: 255}
	faded := &color.RGBA{R: 255, G: 200, B: 60, A: 110}
	plain := sprite.Sprite{Asset: texturePath}
	tinted := sprite.Sprite{Asset: texturePath, Style: sprite.Style{Color: gold}}
	flipH := sprite.Sprite{Asset: texturePath, Style: sprite.Style{FlipH: true}}
	flipV := sprite.Sprite{Asset: texturePath, Style: sprite.Style{FlipV: true}}

	spectrum := []struct {
		cells []cell
	}{
		{
			cells: []cell{
				{s: plain},
				{s: tinted},
				// Alpha in the style color is the drawable's opacity.
				{s: sprite.Sprite{Asset: texturePath, Style: sprite.Style{Color: faded}}},
				{s: flipH},
				{s: flipV},
			},
		},
		{
			cells: []cell{
				{s: plain, spin: true},
				// A zero scale axis reads as unscaled; the stretch
				// column declares both axes.
				{s: plain, scale: geom.Vector2{X: 0.45, Y: 1.1}},
				{s: sprite.Sprite{Asset: texturePath}, scale: geom.Vector2{X: cellScale, Y: cellScale}},
				{s: sprite.Sprite{Asset: texturePath, Style: sprite.Style{FlipH: true, FlipV: true}}},
				{s: sprite.Sprite{Asset: texturePath, Style: sprite.Style{FlipH: true, Color: gold}}},
			},
		},
	}

	var spinners []core.Entity
	for r, row := range spectrum {
		for c, variant := range row.cells {
			// A zero scale reads as unscaled; only the stretched
			// variants declare one, so the rest keep cellScale.
			scale := variant.scale
			if scale == (geom.Vector2{}) {
				scale = geom.Vector2{X: cellScale, Y: cellScale}
			}
			// The rotated variant declares its rotation through the
			// transform, not the sprite.
			transform := core.Transform{
				Position: geom.Vector2{X: float64(cellX0 + c*cellDX), Y: float64(rowY0 + r*rowDY)},
				Scale:    scale,
			}
			if r == 1 && c == 2 {
				transform.Rotation = 0.6
			}
			entity, err := g.World().NewEntity(variant.s, transform)
			if err != nil {
				return nil, err
			}
			if variant.spin {
				spinners = append(spinners, entity)
			}
		}
	}

	// Spin: each fixed tick advances the spinner's rotation. The
	// engine's prev-transform capture and the renderer's interpolation
	// make the spin smooth at any display rate, with no per-frame work.
	if err := g.AddSystem(core.PhaseFixed, "spin", core.SystemFunc(func(ctx *core.Context) error {
		for _, entity := range spinners {
			if err := entity.Update(ctx.World, func(t *core.Transform) { t.Rotation += rotationSpeed }); err != nil {
				return err
			}
		}
		return nil
	})); err != nil {
		return nil, err
	}

	panel := g.World().MustResource[*debug.Panel]()
	panel.Message(fmt.Sprintf("%s is %dx%d px; size comes from the image, preloaded at setup", texturePath, texture.Width, texture.Height))
	panel.Message("row 1: plain, tinted, half-faded, flipH, flipV")
	panel.Message("row 2: spinning, stretched, rotated, flip both, tint + flip")

	return g, nil
}
