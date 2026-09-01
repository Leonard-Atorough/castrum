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

	// Register input controller (runs first to read input and set velocity)
	if err := game.Systems.Register("player_controller", -1, NewPlayerController(game), game.World); err != nil {
		log.Fatalf("failed to register player controller: %v", err)
	}

	// Register movement system (applies velocity to position)
	if err := game.Systems.Register("movement", 0, &MovementSystem{}, game.World); err != nil {
		log.Fatalf("failed to register movement system: %v", err)
	}

	// // Register rotator system
	// if err := game.Systems.Register("rotator", 0, &RotatorSystem{}, game.World); err != nil {
	// 	log.Fatalf("failed to register rotator system: %v", err)
	// }

	// Register the pulse system
	if err := game.Systems.Register("pulse", 0, &PulseSystem{}, game.World); err != nil {
		log.Fatalf("failed to register pulse system: %v", err)
	}

	// Register the camera system (runs after movement to update the camera position)
	if err := game.Systems.Register("camera", 1, &CameraSystem{Camera: game.Camera, Input: game.Input}, game.World); err != nil {
		log.Fatalf("failed to register camera system: %v", err)
	}

	// Spawn a controllable circle on Layer1 at the center
	_, err := game.World.CreateWithComponents(
		"player_circle",
		components.Transform{
			Position: types.Vector2{X: 0, Y: 0},
			Scale:    types.Vector2{X: 25, Y: 25},
			Color:    color.RGBA{R: 255, G: 100, B: 100, A: 255},
		},
		components.Renderable{Primitive: components.PrimitiveKindCircle, Visible: true, Layer: components.Layer1},
		Player{},
		Velocity{Linear: types.Vector2{X: 0, Y: 0}},
	)
	if err != nil {
		log.Fatalf("failed to spawn player circle: %v", err)
	}

	// Lets create a grid of squares around the center.
	gridSizeH := 60
	gridSizeW := 60
	spacing := 20.0
	for i := -gridSizeW; i <= gridSizeW; i++ {
		for j := -gridSizeH; j <= gridSizeH; j++ {
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
				components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0},
				// components.Spin{AngularVelocity: 1.5 + 0.1*float64(i+j)},
				Pulse{StartScale: types.Vector2{X: 10, Y: 10}, Amplitude: 0.5, Frequency: 2.0, TimeOffset: float64(i+j) * 0.1},
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
