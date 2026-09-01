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
	game := castrum.NewGame(config)

	if err := game.Systems.Register("rotator", 0, &RotatorSystem{}, game.World); err != nil {
		log.Fatalf("failed to register rotator system: %v", err)
	}

	// The camera sits at the world origin and centers it on screen, so
	// spawning here puts the square in the middle of the window.
	_, err := game.World.CreateWithComponents("Square",
		components.Transform{
			Position: types.Vector2{},
			Scale:    types.Vector2{X: 100, Y: 100},
			Color:    color.RGBA{R: 220, G: 60, B: 60, A: 255},
		},
		components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: true},
		components.Spin{AngularVelocity: 1.5},
	)
	if err != nil {
		log.Fatalf("failed to spawn square: %v", err)
	}

	ebiten.SetWindowSize(config.Window.Width, config.Window.Height)
	ebiten.SetWindowTitle(config.Window.Title)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
