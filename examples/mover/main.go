package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/input"
)

const (
	screenWidth  = 800
	screenHeight = 600
	playerSize   = 32
	moveSpeed    = 200 // pixels per second
)

// Input actions defined by the engine's default key bindings (arrow keys).
const (
	actionMoveUp    = input.Action("move_up")
	actionMoveDown  = input.Action("move_down")
	actionMoveLeft  = input.Action("move_left")
	actionMoveRight = input.Action("move_right")
)

func main() {
	config := castrum.DefaultConfig()
	config.Window.Title = "Castrum - Mover"
	config.Window.Width = screenWidth
	config.Window.Height = screenHeight
	config.Graphics.VirtualWidth = screenWidth
	config.Graphics.VirtualHeight = screenHeight

	game, err := castrum.NewGame(config, nil)
	if err != nil {
		log.Fatal(err)
	}

	world := game.World()

	// Spawn a green square at the world origin. The default camera sits at the
	// origin too, so the square appears centered on screen. For primitive
	// sprites, Sprite.Size is the base dimension in pixels and Transform.Scale
	// is a multiplier on top of it ({1,1} = no scaling).
	player, err := world.CreateWithComponents("Player",
		components.NewTransform(
			geom.Vector2{X: 0, Y: 0},
			0,
			geom.Vector2{X: 1, Y: 1},
			geom.Vector2{X: 0, Y: 0},
		),
		components.Sprite{
			Primitive: components.PrimitiveKindRectangle,
			Size:      geom.Vector2{X: playerSize, Y: playerSize},
			Visible:   true,
			Color:     color.RGBA{R: 60, G: 220, B: 60, A: 255},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := world.RegisterSystem(
		"player_controller",
		ecs.SystemPriorityPrePhysics,
		&moveSystem{player: player.ID, input: game.Input()},
	); err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(config.Window.Title)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// moveSystem reads directional input each fixed tick and translates the player
// entity. It implements ecs.System.
type moveSystem struct {
	player ecs.EntityID
	input  input.Reader
}

func (s *moveSystem) Init(_ *ecs.World) error     { return nil }
func (s *moveSystem) Shutdown(_ *ecs.World) error { return nil }

func (s *moveSystem) Update(world *ecs.World, dt float64) error {
	t, err := world.GetComponent[components.Transform](s.player)
	if err != nil {
		return err
	}

	var dx, dy float64
	if s.input.ActionHeld(actionMoveLeft) {
		dx -= 1
	}
	if s.input.ActionHeld(actionMoveRight) {
		dx += 1
	}
	if s.input.ActionHeld(actionMoveUp) {
		dy -= 1
	}
	if s.input.ActionHeld(actionMoveDown) {
		dy += 1
	}

	t.Position.X += dx * moveSpeed * dt
	t.Position.Y += dy * moveSpeed * dt

	// Keep the square fully visible. The camera is at the world origin, so the
	// visible range is [-screen/2, +screen/2] on each axis.
	half := float64(playerSize) / 2
	t.Position.X = clamp(t.Position.X, -float64(screenWidth)/2+half, float64(screenWidth)/2-half)
	t.Position.Y = clamp(t.Position.Y, -float64(screenHeight)/2+half, float64(screenHeight)/2-half)

	return world.SetComponent(s.player, t)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
