// Command spatial demonstrates positional audio with a movable camera.
package main

import (
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/sprite"
)

const (
	moveSpeed = 200
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"move_x": {
			input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}.Binding(),
			input.KeyPairInput{Negative: input.KeyArrowLeft, Positive: input.KeyArrowRight}.Binding(),
		},
		"move_y": {
			input.KeyPairInput{Negative: input.KeyW, Positive: input.KeyS}.Binding(),
			input.KeyPairInput{Negative: input.KeyArrowUp, Positive: input.KeyArrowDown}.Binding(),
		},
	}

	game := castrum.New(
		castrum.WithTitle("Spatial Audio"),
		castrum.WithFilesystem(audioassets.Files),
		castrum.WithBindings(bindings),
	)

	playerStart := geom.Vector2{X: -200}
	player, err := game.World().NewEntity(
		core.Transform{Position: playerStart},
		sprite.Primitive{
			Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: 20, Y: 20},
			Style: sprite.Style{Color: &color.RGBA{G: 200, B: 255, A: 255}},
		},
	)
	if err != nil {
		return err
	}

	listener := game.World().MustResource[*audio.Listener]()
	listener.Range = 800
	listener.Enabled = true
	listener.Mode = audio.FixedPosition
	listener.Position = playerStart

	source, err := audio.NewPlayer(audio.Track(audioassets.Music), audio.Settings{
		Volume: 1, Spatial: true, Attenuation: geom.EaseInOutCubic, Loop: audio.LoopTrack,
	})
	if err != nil {
		return err
	}
	_, err = game.World().NewEntity(
		source,
		core.Transform{},
		sprite.Primitive{
			Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 80, Y: 80},
			Style: sprite.Style{Color: &color.RGBA{R: 220, G: 100, A: 255}},
		},
	)
	if err != nil {
		return err
	}

	if err := game.AddSystem(core.PhaseFixed, "player.move", movementSystem(listener, player)); err != nil {
		return err
	}

	runner := ebitrun.New()

	runner.Run(game)
	return nil
}

func movementSystem(l *audio.Listener, p core.Entity) core.SystemFunc {
	return func(ctx *core.Context) error {
		actions := ctx.World.MustResource[input.Actions]()
		dir := geom.Vector2{
			X: actions.Axis("move_x"),
			Y: actions.Axis("move_y"),
		}

		if !dir.IsZero() {
			dir = dir.Normalize()
		}

		var position geom.Vector2
		if err := p.Update(ctx.World, func(t *core.Transform) {
			t.Position = t.Position.Add(dir.Mul(moveSpeed * ctx.DeltaTime.Seconds()))
			position = t.Position
		}); err != nil {
			return err
		}
		l.Position = position
		return nil
	}
}
