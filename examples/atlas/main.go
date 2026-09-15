// Command atlas demonstrates texture atlas slicing. It slices a 96x28
// torch sprite sheet into six 16x28 frames and renders them in a 3x2 grid
// with random rotations to show that each region is independently sourced.
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
)

const (
	screenWidth  = 800
	screenHeight = 600

	torchAssetPath = "examples/atlas/torch_light.png"
	torchAtlasID   = "torch_atlas"

	frameW = 16
	frameH = 28
	scale  = 4 // magnify each frame for visibility
)

const (
	gridCols = 3
	gridRows = 2
	gap      = 32 // pixels between grid cells
	cellW    = frameW*scale + gap
	cellH    = frameH*scale + gap
)

func main() {
	config := castrum.DefaultConfig()
	config.Window.Title = "Castrum - Atlas Slicing"
	config.Window.Width = screenWidth
	config.Window.Height = screenHeight
	config.Graphics.VirtualWidth = screenWidth
	config.Graphics.VirtualHeight = screenHeight

	game, err := castrum.NewGame(config, os.DirFS("."))
	if err != nil {
		log.Fatal(err)
	}

	world := game.World()

	// Slice the torch texture into 6 frames of 16x28 using a grid slice.
	// GridSlice auto-names regions as "torch_0", "torch_1", ... left-to-right.
	builder, err := game.NewAtlas(torchAtlasID, torchAssetPath, 96, 28)
	if err != nil {
		log.Fatal(err)
	}
	if _, errs := builder.GridSlice(frameW, frameH, "torch"); len(errs) > 0 {
		log.Fatal(errs[0])
	}
	if _, err := builder.Build(); err != nil {
		log.Fatal(err)
	}

	// Spawn a 3x2 grid of sprites, one per frame, each with a random rotation.
	frameCount := gridCols * gridRows
	// Grid is centered on the world origin (where the default camera sits).
	startX := -float64(gridCols*cellW)/2 + float64(cellW)/2
	startY := -float64(gridRows*cellH)/2 + float64(cellH)/2

	for i := range frameCount {
		col := i % gridCols
		row := i / gridCols
		x := startX + float64(col*cellW)
		y := startY + float64(row*cellH)
		rotation := rand.Float64() * 2 * 3.141592653589793

		_, err := world.CreateWithComponents(
			fmt.Sprintf("Torch_%d", i),
			components.NewTransform(
				geom.Vector2{X: x, Y: y},
				rotation,
				geom.Vector2{X: scale, Y: scale},
			),
			components.Sprite{
				AtlasID:    torchAtlasID,
				RegionName: fmt.Sprintf("torch_%d", i),
				Visible:    true,
			},
		)
		if err != nil {
			log.Fatalf("failed to spawn torch %d: %v", i, err)
		}
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(config.Window.Title)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
