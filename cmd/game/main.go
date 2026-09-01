// Command game is the reference/example game for the castrum engine: it
// spawns a single spinning square to exercise the rendering and system
// pipeline end to end.
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/types"
)

func main() {
	config := castrum.DefaultConfig()
	config.Window.Width = 1920
	config.Window.Height = 1080
	config.Graphics.VirtualWidth = 1920
	config.Graphics.VirtualHeight = 1080
	config.Engine.EnableDebug = true

	game := castrum.NewGame(config)

	if err := game.Systems.Register("rotator", 0, &RotatorSystem{}, game.World); err != nil {
		log.Fatalf("failed to register rotator system: %v", err)
	}

	// Lets create a grid of squares around the center.
	gridSize := 20
	spacing := 20.0
	for i := -gridSize; i <= gridSize; i++ {
		for j := -gridSize; j <= gridSize; j++ {
			if i == 0 && j == 0 {
				continue // Skip the center square
			}
			_, err := game.World.CreateWithComponents(
				"Square",
				components.Transform{
					Position: types.Vector2{X: float64(i) * spacing, Y: float64(j) * spacing},
					Scale:    types.Vector2{X: 10, Y: 10},
					Color:    color.RGBA{R: 60, G: 220, B: 60, A: 255},
				},
				components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: true},
				components.Spin{AngularVelocity: 1.5 + 0.1*float64(i+j)},
			)
			if err != nil {
				log.Fatalf("failed to spawn grid square: %v", err)
			}
		}
	}

	ebiten.SetWindowSize(config.Window.Width, config.Window.Height)
	ebiten.SetWindowTitle(config.Window.Title)
	ebiten.SetFullscreen(config.Window.Fullscreen)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
