// Command animated demonstrates a looping sprite animation using the
// castrum engine. It slices a 96x28 torch sprite sheet into six 16x28
// frames, defines a looping animation clip at 10 FPS, and spawns a
// single animated torch entity centered on screen.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
)

const (
	screenWidth  = 800
	screenHeight = 600

	torchAssetPath = "examples/animated/torch_light.png"
	torchAtlasID   = "torch_atlas"
	torchClipID    = "torch_flicker"

	frameW = 16
	frameH = 28
	scale  = 4 // magnify for visibility
)

func main() {
	config := castrum.DefaultConfig()
	config.Window.Title = "Castrum - Animated Torch"
	config.Window.Width = screenWidth
	config.Window.Height = screenHeight
	config.Graphics.VirtualWidth = screenWidth
	config.Graphics.VirtualHeight = screenHeight

	game, err := castrum.NewGame(config, os.DirFS("."))
	if err != nil {
		log.Fatal(err)
	}

	world := game.World()

	// 1. Slice the torch texture into 6 frames of 16x28.
	builder, err := game.NewAtlas(torchAtlasID, torchAssetPath, 96, 28)
	if err != nil {
		log.Fatal(err)
	}
	if _, errs := builder.GridSlice(frameW, frameH, "torch"); len(errs) > 0 {
		log.Fatal(errs[0])
	}
	torchAtlas, err := builder.Build()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Define a looping animation clip at 10 FPS using the new Game.NewClip
	// facade. No manual world.GetResource dance — one method.
	frameCount := 96 / frameW
	clipBuilder := game.NewClip(torchClipID, torchAtlas)
	for i := range frameCount {
		clipBuilder.AddFrame(fmt.Sprintf("torch_%d", i))
	}
	if _, err := clipBuilder.SetFPS(10).SetLoop(animation.LoopForever).Build(); err != nil {
		log.Fatal(err)
	}

	// 3. Spawn an animated torch entity at the world origin (screen center).
	_, err = world.CreateWithComponents("Torch",
		components.NewTransform(
			geom.Vector2{X: 0, Y: 0},
			0,
			geom.Vector2{X: scale, Y: scale},
		),
		components.Sprite{
			AtlasID:    torchAtlasID,
			RegionName: "torch_0",
			Visible:    true,
		},
		components.NewAnimation(torchClipID, true),
	)
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(config.Window.Title)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
