package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenW, screenH = 1280, 720

	fieldSize   = 50
	fieldGap    = 150
	circleRadii = 32

	moveSpeed = 300
	turnSpeed = 3
	zoomSpeed = 1.5
	zoomMin   = 0.25
	zoomMax   = 8
)

func main() {
	if err := run(); err != nil {
		fmt.Println("input: ", err)
	}
}

func run() error {
	bindings := input.Bindings{
		"move_forward": []input.Input{
			input.KeyInput{Key: input.KeyW},
			input.KeyInput{Key: input.KeyArrowUp},
			input.PadAxisInput{Axis: input.PadLeftStickY, Direction: -1},
		},
		"move_backward": []input.Input{
			input.KeyInput{Key: input.KeyS},
			input.KeyInput{Key: input.KeyArrowDown},
			input.PadAxisInput{Axis: input.PadLeftStickY, Direction: 1},
		},
		"turn_left": []input.Input{
			input.KeyInput{Key: input.KeyA},
			input.KeyInput{Key: input.KeyArrowLeft},
			input.PadAxisInput{Axis: input.PadRightStickX, Direction: -1},
		},
		"turn_right": []input.Input{
			input.KeyInput{Key: input.KeyD},
			input.KeyInput{Key: input.KeyArrowRight},
			input.PadAxisInput{Axis: input.PadRightStickX, Direction: 1},
		},
		"zoom": []input.Input{
			input.KeyPairInput{Positive: input.KeyQ, Negative: input.KeyE},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("castrum - input"),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return err
	}

	camera, err := g.World().NewEntity(
		core.Camera{Zoom: 1, Primary: true},
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		core.PrevTransform{Position: geom.Vector2{X: screenW / 2, Y: screenH / 2}},
	)
	if err != nil {
		return err
	}
	// spawn character entity
	character, err := g.World().NewEntity(
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		core.Sprite{
			Drawable: core.RectShape{Size: geom.Vector2{X: 32, Y: 64}},
			Color:    color.RGBA{G: 255},
		},
		core.PrevTransform{Position: geom.Vector2{X: screenW / 2, Y: screenH / 2}},
	)
	if err != nil {
		return err
	}

	// A sparse field to roam: the gap must exceed the circles' width
	// or the field reads as a solid mass, and the viewport culls
	// whatever is off screen.
	for i := range fieldSize {
		for j := range fieldSize {
			pos := geom.Vector2{X: float64(i * fieldGap), Y: float64(j * fieldGap)}
			if _, err := g.World().NewEntity(
				core.Transform{
					Position: pos,
					Scale:    geom.Vector2{X: 1, Y: 1},
				},
				core.Sprite{
					Drawable: core.CircleShape{Radii: geom.Vector2{X: circleRadii, Y: circleRadii}},
					Color:    color.RGBA{R: 255},
				},
				core.PrevTransform{Position: pos},
			); err != nil {
				return err
			}
		}
	}

	// system for camera movement and zoom
	if err := g.AddSystem(core.PhaseFixed, "input.character_move", characterMovementSystem(character)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "input.camera", cameraSystem(camera, character)); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrint(screen, "castrum input - use WASD or arrow keys to move, QE to zoom\nFPS: "+fmt.Sprintf("%.2f", ebiten.ActualFPS()))
		return nil
	})

	return runner.Run()
}

func cameraSystem(camera *core.Entity, character *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		cam, ok := camera.Component[core.Camera](ctx.World)
		if !ok {
			return fmt.Errorf("camera entity missing Camera component")
		}
		if !cam.Primary {
			return fmt.Errorf("camera entity is not primary")
		}
		camTransform, ok := camera.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("camera entity missing Transform component")
		}

		char, ok := character.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("character entity missing Transform component")
		}
		// Make the camera follow the character
		camTransform.Position = char.Position

		if ctx.Actions.Held("zoom") {
			cam.Zoom += ctx.Actions.Axis("zoom") * zoomSpeed * ctx.DeltaTime.Seconds()
			cam.Zoom = math.Max(zoomMin, math.Min(zoomMax, cam.Zoom))
		}

		// Both mutated components are value copies: write them back
		// or the follow and the zoom are lost.
		if err := camera.SetComponent(ctx.World, camTransform); err != nil {
			return err
		}
		return camera.SetComponent(ctx.World, cam)
	})
}

func characterMovementSystem(character *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		transform, ok := character.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("character entity missing Transform component")
		}
		// Forward is screen-up (negative Y in this world), rotated
		// by the character's rotation.
		facing := geom.Vector2{X: 0, Y: -1}.Rotate(transform.Rotation)

		if ctx.Actions.Held("move_forward") {
			transform.Position.X += facing.X * moveSpeed * ctx.DeltaTime.Seconds()
			transform.Position.Y += facing.Y * moveSpeed * ctx.DeltaTime.Seconds()
		}
		if ctx.Actions.Held("move_backward") {
			transform.Position.X -= facing.X * moveSpeed * ctx.DeltaTime.Seconds()
			transform.Position.Y -= facing.Y * moveSpeed * ctx.DeltaTime.Seconds()
		}
		if ctx.Actions.Held("turn_left") {
			transform.Rotation -= turnSpeed * ctx.DeltaTime.Seconds()
		}
		if ctx.Actions.Held("turn_right") {
			transform.Rotation += turnSpeed * ctx.DeltaTime.Seconds()
		}

		character.SetComponent(ctx.World, transform)
		return nil
	})
}
