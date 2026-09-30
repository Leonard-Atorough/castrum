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

	moveSpeed   = 300
	turnSpeed   = 3
	barrelLen   = 56
	barrelWidth = 12
	zoomSpeed   = 1.5
	zoomMin     = 0.25
	zoomMax     = 8
)

func main() {
	if err := run(); err != nil {
		panic(err)
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

	// The engine camera frames the world; a follow system moves it.
	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// The tank: a hull the bindings drive, and a turret barrel that
	// aims at the mouse. A zero scale reads as unscaled. The barrel is
	// a line shape anchored at the position - zero From pivots it on
	// the hull's center.
	body, err := g.World().NewEntity(
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
		},
		core.Sprite{
			Drawable: core.RectShape{Size: geom.Vector2{X: 64, Y: 96}},
			Color:    color.RGBA{G: 200},
		},
	)
	if err != nil {
		return err
	}
	turret, err := g.World().NewEntity(
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
		},
		core.Sprite{
			Drawable:    core.LineShape{To: geom.Vector2{X: 0, Y: -barrelLen}},
			Outline:     true,
			StrokeWidth: barrelWidth,
			Color:       color.RGBA{R: 120, A: 255},
		},
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
				},
				core.Sprite{
					Drawable: core.CircleShape{Radii: geom.Vector2{X: circleRadii, Y: circleRadii}},
					Color:    color.RGBA{R: 255},
				},
			); err != nil {
				return err
			}
		}
	}

	// system for camera movement and zoom
	if err := g.AddSystem(core.PhaseFixed, "input.tank_body", tankBodySystem(body)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "input.turret", turretSystem(turret, body, camera)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "input.camera", cameraSystem(camera, body)); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrint(screen, "castrum input - WASD or arrows or stick to drive, mouse aims the turret, QE zooms\nFPS: "+fmt.Sprintf("%.2f", ebiten.ActualFPS()))
		return nil
	})

	return runner.Run()
}

func cameraSystem(camera *core.Entity, body *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, _ := body.Component[core.Transform](ctx.World)

		// The camera follows the hull.
		if err := camera.Update(ctx.World, func(t *core.Transform) {
			t.Position = hull.Position
		}); err != nil {
			return err
		}

		if ctx.Actions.Held("zoom") {
			if err := camera.Update(ctx.World, func(c *core.Camera) {
				c.Zoom += ctx.Actions.Axis("zoom") * zoomSpeed * ctx.DeltaTime.Seconds()
				c.Zoom = math.Max(zoomMin, math.Min(zoomMax, c.Zoom))
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// tankBodySystem drives the hull like a tank: the bindings move it
// along its facing and turn it in place, independent of where the
// turret aims.
func tankBodySystem(body *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		return body.Update(ctx.World, func(t *core.Transform) {
			// Forward is screen-up (negative Y in this world), rotated
			// by the hull's rotation.
			facing := geom.Vector2{X: 0, Y: -1}.Rotate(t.Rotation)

			if ctx.Actions.Held("move_forward") {
				t.Position = t.Position.Add(facing.Mul(moveSpeed * ctx.DeltaTime.Seconds()))
			}
			if ctx.Actions.Held("move_backward") {
				t.Position = t.Position.Sub(facing.Mul(moveSpeed * ctx.DeltaTime.Seconds()))
			}
			if ctx.Actions.Held("turn_left") {
				t.Rotation -= turnSpeed * ctx.DeltaTime.Seconds()
			}
			if ctx.Actions.Held("turn_right") {
				t.Rotation += turnSpeed * ctx.DeltaTime.Seconds()
			}
		})
	})
}

// turretSystem mounts the barrel on the hull and aims it at the
// mouse. The cursor is raw input - read straight from the snapshot,
// not through bindings - and projects to world space through the same
// camera view the renderer frames with. Rotation 0 faces screen-up,
// so the aim angle is atan2(x, -y) of the direction to the cursor.
func turretSystem(turret *core.Entity, body *core.Entity, camera *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, _ := body.Component[core.Transform](ctx.World)
		cam, _ := camera.Component[core.Camera](ctx.World)
		camT, _ := camera.Component[core.Transform](ctx.World)

		world := core.CameraView{Position: camT.Position, Zoom: cam.Zoom}.
			ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight)
		aim := world.Sub(hull.Position)

		return turret.Update(ctx.World, func(t *core.Transform) {
			t.Position = hull.Position
			t.Rotation = math.Atan2(aim.X, -aim.Y)
		})
	})
}
