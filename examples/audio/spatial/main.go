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
	"github.com/Leonard-Atorough/castrum/render"
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

	game, err := castrum.New(
		castrum.WithTitle("Spatial Audio"),
		castrum.WithFilesystem(audioassets.Files),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return err
	}

	playerStart := geom.Vector2{X: -200}
	player, err := game.World().NewEntity(
		core.Transform{Position: playerStart},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 20, Y: 20}},
			Color:    color.RGBA{G: 200, B: 255, A: 255},
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

	source := audio.NewSource()
	source.Spatial = true
	source.Attenuation = geom.EaseInOutCubic
	source.Loop = audio.LoopForever
	_, err = game.World().NewEntity(
		source,
		audio.Clip{Audio: audioassets.Music},
		core.Transform{},
		render.Sprite{
			Drawable: render.RectShape{Size: geom.Vector2{X: 80, Y: 80}},
			Color:    color.RGBA{R: 220, G: 100, A: 255},
		},
	)
	if err != nil {
		return err
	}

	if err := game.AddSystem(core.PhaseFixed, "player.move", movementSystem(listener, player)); err != nil {
		return err
	}

	runner, err := ebitrun.New(game)
	if err != nil {
		return err
	}

	return game.Run(runner)
}

func movementSystem(l *audio.Listener, p *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		dir := geom.Vector2{
			X: ctx.Actions.Axis("move_x"),
			Y: ctx.Actions.Axis("move_y"),
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
	})
}
