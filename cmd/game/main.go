// Command game is the reference/example game for the castrum engine: it
// spawns a single spinning square to exercise the rendering and system
// pipeline end to end.
package main

import (
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	gamesystems "github.com/leonard-atorough/castrum/cmd/game/systems"
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

	game := castrum.NewGame(config, os.DirFS("."))

	// Set camera bounds to the grid extent
	// Grid: 60×60 with spacing 34 pixels = -2040 to +2040 in each direction
	// Each square is 32×32, so extends ±16 from center
	gridSizeH := 30
	gridSizeW := 30
	spacing := 34.0
	squareRadius := 16.0
	cameraOverflow := 100.0

	minX := -float64(gridSizeW)*spacing - squareRadius - cameraOverflow
	maxX := float64(gridSizeW)*spacing + squareRadius + cameraOverflow
	minY := -float64(gridSizeH)*spacing - squareRadius - cameraOverflow
	maxY := float64(gridSizeH)*spacing + squareRadius + cameraOverflow

	game.Camera.Bounds = types.NewRect(types.NewVector2(minX, minY), types.NewVector2(maxX, maxY))

	// Register input controller (runs first to read input and set velocity)
	if err := game.Systems.Register("player_controller", -1, gamesystems.NewPlayerController(game), game.World); err != nil {
		log.Fatalf("failed to register player controller: %v", err)
	}

	// Register movement system (applies velocity to position)
	if err := game.Systems.Register("movement", 0, gamesystems.NewMovementSystem(game.Camera), game.World); err != nil {
		log.Fatalf("failed to register movement system: %v", err)
	}

	// // Register rotator system
	// if err := game.Systems.Register("rotator", 0, &gamesystems.RotatorSystem{}, game.World); err != nil {
	// 	log.Fatalf("failed to register rotator system: %v", err)
	// }

	// Register the pulse system
	if err := game.Systems.Register("pulse", 0, &gamesystems.PulseSystem{}, game.World); err != nil {
		log.Fatalf("failed to register pulse system: %v", err)
	}

	// Register the camera system (runs after movement to update the camera position)
	if err := game.Systems.Register("camera", 1, &gamesystems.CameraSystem{Camera: game.Camera, Input: game.Input}, game.World); err != nil {
		log.Fatalf("failed to register camera system: %v", err)
	}

	// Spawn a controllable circle on Layer1 at the center
	_, err := game.World.CreateWithComponents(
		"player_circle",
		components.Transform{
			Position: types.Vector2{X: 0, Y: 0},
			Scale:    types.Vector2{X: 1, Y: 1},
		},
		components.Renderable{TexturePath: "cmd/game/example.png", Visible: true, Layer: components.Layer1},
		gamecomponents.Player{},
		gamecomponents.Velocity{Linear: types.Vector2{X: 0, Y: 0}},
	)
	if err != nil {
		log.Fatalf("failed to spawn player circle: %v", err)
	}

	// Lets create a grid of squares around the center.
	for i := -gridSizeW; i <= gridSizeW; i++ {
		for j := -gridSizeH; j <= gridSizeH; j++ {
			if i == 0 && j == 0 {
				continue // Skip the center square
			}
			_, err := game.World.CreateWithComponents(
				"Square",
				components.Transform{
					Position: types.Vector2{X: float64(i) * spacing, Y: float64(j) * spacing},
					Scale:    types.Vector2{X: 32, Y: 32},
					Color:    color.RGBA{R: 60, G: 220, B: 60, A: 255},
				},
				components.Renderable{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0},
				// components.Spin{AngularVelocity: 1.5 + 0.1*float64(i+j)},
				gamecomponents.Pulse{StartScale: types.Vector2{X: 32, Y: 32}, Amplitude: 0.5, Frequency: 2.0, TimeOffset: float64(i+j) * 0.1},
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
