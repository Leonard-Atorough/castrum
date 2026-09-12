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
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

func main() {
	config := castrum.DefaultConfig()
	config.Window.Width = 1920
	config.Window.Height = 1080
	config.Graphics.VirtualWidth = 1920
	config.Graphics.VirtualHeight = 1080
	config.Engine.EnableDebug = true

	game, err := castrum.NewGame(config, os.DirFS("."))
	if err != nil {
		log.Fatalf("failed to create game: %v", err)
	}

	world := game.World()

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

	// Get the camera entity and update its bounds
	cam := game.CameraEntity().ID
	camComp, err := world.GetComponent[components.Camera](cam)
	if err != nil {
		log.Fatalf("failed to get camera component: %v", err)
	}

	camComp.Bounds = geom.Rect{Min: geom.Vector2{X: minX, Y: minY}, Max: geom.Vector2{X: maxX, Y: maxY}}
	world.SetComponent(cam, camComp)

	// Register input controller (runs first to read input and set velocity)
	if err := world.RegisterSystem("player_controller", -1, gamesystems.NewPlayerController(game.Input())); err != nil {
		log.Fatalf("failed to register player controller: %v", err)
	}

	// Register movement system (applies velocity to position)
	if err := world.RegisterSystem("movement", ecs.SystemPriorityPrePhysics, gamesystems.NewMovementSystem()); err != nil {
		log.Fatalf("failed to register movement system: %v", err)
	}

	// Register the pulse system
	if err := world.RegisterSystem("pulse", ecs.SystemPriorityPrePhysics, &gamesystems.PulseSystem{}); err != nil {
		log.Fatalf("failed to register pulse system: %v", err)
	}

	// Register the camera system (runs after movement to update the camera position)
	// CameraSystem now queries the camera from the world, no need to pass it
	if err := world.RegisterSystem("camera", ecs.SystemPriorityPostPhysics, &gamesystems.CameraSystem{Input: game.Input()}); err != nil {
		log.Fatalf("failed to register camera system: %v", err)
	}

	// Register the collision system (runs after movement to handle collision response)
	if err := world.RegisterSystem("collision", ecs.SystemPriorityPostPhysics, gamesystems.NewCollisionSystem()); err != nil {
		log.Fatalf("failed to register collision system: %v", err)
	}

	// Spawn a controllable circle on Layer1 at the center
	_, createErr := world.CreateWithComponents(
		"player",
		components.Transform{
			Position: geom.Vector2{X: 0, Y: 0},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		components.Sprite{TexturePath: "example.png", Visible: true, Layer: 1},
		gamecomponents.Player{},
		gamecomponents.Velocity{Linear: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.Rect{Min: geom.Vector2{X: -16, Y: -16}, Max: geom.Vector2{X: 16, Y: 16}}, true, false, 0, 1),
	)
	if createErr != nil {
		log.Fatalf("failed to spawn player circle: %v", createErr)
	}

	// Lets create a grid of squares around the center.
	for i := -gridSizeW; i <= gridSizeW; i++ {
		for j := -gridSizeH; j <= gridSizeH; j++ {
			if i == 0 && j == 0 {
				continue // Skip the center square
			}
			_, err := world.CreateWithComponents(
				"Square",
				components.Transform{
					Position: geom.Vector2{X: float64(i) * spacing, Y: float64(j) * spacing},
					Scale:    geom.Vector2{X: 32, Y: 32},
					Color:    color.RGBA{R: 60, G: 220, B: 60, A: 255},
				},
				components.Sprite{Primitive: components.PrimitiveKindRectangle, Visible: true, Layer: 0},
				gamecomponents.Pulse{StartScale: geom.Vector2{X: 32, Y: 32}, Amplitude: 0.5, Frequency: 1, TimeOffset: float64(i+j) * 0.1},
			)
			if err != nil {
				log.Fatalf("failed to spawn grid square: %v", err)
			}
		}
	}

	// Spawn some circles with colliders for collision testing
	circlePositions := []geom.Vector2{
		{X: 200, Y: 200},
		{X: -200, Y: 200},
		{X: 200, Y: -200},
		{X: -200, Y: -200},
		{X: 400, Y: 0},
		{X: -400, Y: 0},
	}

	for i, pos := range circlePositions {
		_, err := world.CreateWithComponents(
			"circle_obstacle",
			components.Transform{
				Position: pos,
				Scale:    geom.Vector2{X: 30, Y: 30},
				Color:    color.RGBA{R: 255, G: 100, B: 100, A: 255},
			},
			components.Sprite{Primitive: components.PrimitiveKindCircle, Visible: true, Layer: 1},
			components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, true, false, 1, 0),
		)
		if err != nil {
			log.Fatalf("failed to spawn circle %d: %v", i, err)
		}
	}

	ebiten.SetWindowSize(config.Window.Width, config.Window.Height)
	ebiten.SetWindowTitle(config.Window.Title)
	ebiten.SetFullscreen(config.Window.Fullscreen)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
