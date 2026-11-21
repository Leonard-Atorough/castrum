// Command spritesheets demonstrates sheet-backed sprites: a sprite.Sprite
// names a sheet document by asset ID plus a region within it, and the render
// collector resolves the document through the asset loader and draws that
// rectangle of the texture. An empty Region draws the whole texture instead
// (see the textures example).
//
// The shipped gopher_sheet.json was authored in Go: spritesheet.Grid slices
// the 1152x288 texture into four 288x288 cells named gopher_0 through
// gopher_3, and one hand-added irregular region, gopher_wide, spans two cells.
// Run with -author <file> to regenerate the metadata without opening a window.
//
// Setup resolves the sheet document, checks every region against the decoded
// texture, and confirms the regions the sprites use - a bad document fails the
// launch instead of the first collected frame.
//
// Run from this directory:
//
//	go run .
package main

import (
	"embed"
	"flag"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

//go:embed gopher_sheet.png gopher_sheet.json
var files embed.FS

const (
	texturePath = "gopher_sheet.png"
	sheetPath   = "gopher_sheet.json"

	// The sheet is one row of four cellSize x cellSize cells.
	cellSize   = 288
	gridFrames = 4

	// wideRegion spans two cells; it is authored by hand, not by the grid.
	wideRegion = "gopher_wide"

	screenW, screenH = 1280, 720
)

var gridRegions = []string{"gopher_0", "gopher_1", "gopher_2", "gopher_3"}

func main() {
	output := flag.String("author", "", "write spritesheet metadata to this file without opening a window")
	flag.Parse()
	if *output != "" {
		if err := authorSheet(*output); err != nil {
			panic(err)
		}
		fmt.Printf("Saved spritesheet to %s\n", *output)
		return
	}
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
		castrum.WithTitle("castrum - spritesheets"),
		castrum.WithFilesystem(files),
	)

	// Setup-time validation: resolve the document, check every region
	// against the decoded texture, and confirm the regions named below.
	// Without this, a bad sheet surfaces as a first-frame collect error.
	loader := g.World().MustResource[*asset.Loader]()
	sheet, err := asset.FileSheet(sheetPath).Resolve(loader)
	if err != nil {
		return nil, err
	}
	texture, err := loader.Load[asset.TextureData](string(sheet.Texture))
	if err != nil {
		return nil, err
	}
	if err := sheet.ValidateBounds(texture.Image.Bounds()); err != nil {
		return nil, err
	}
	regions := append(slices.Clone(gridRegions), wideRegion)
	for _, name := range regions {
		if _, err := sheet.Region(name); err != nil {
			return nil, err
		}
	}

	// The engine camera frames the world from the screen's center, so
	// world coordinates are screen coordinates at zoom 1.
	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return nil, err
	}

	// Top row: the four grid regions at native size, side by side.
	// 4*288 = 1152 px centered in 1280, so the first cell's center is
	// at 64 + 144 = 208.
	for i, name := range gridRegions {
		if _, err := g.World().NewEntity(
			sprite.Sprite{Asset: sheetPath, Region: name},
			core.Transform{Position: geom.Vector2{X: float64(208 + i*cellSize), Y: 300}},
		); err != nil {
			return nil, err
		}
	}

	// Below: the irregular two-cell region at half scale.
	if _, err := g.World().NewEntity(
		sprite.Sprite{Asset: sheetPath, Region: wideRegion},
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: 560},
			Scale:    geom.Vector2{X: 0.5, Y: 0.5},
		},
	); err != nil {
		return nil, err
	}

	panel := g.World().MustResource[*debug.Panel]()
	panel.Message(fmt.Sprintf("%s: %d grid cells of %dpx + %q spanning two cells", sheetPath, gridFrames, cellSize, wideRegion))
	panel.Message("regions resolve through the loader at collect time; metadata was validated at setup")
	panel.Message("run with -author <file> to regenerate the sheet document without a window")

	return g, nil
}
