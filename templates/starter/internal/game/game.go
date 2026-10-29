// Package game defines the starter game's world and systems.
package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
)

const playerSpeed = 240

// New creates the game world and registers the player's movement system.
func New() (*castrum.Game, error) {
	bindings := input.Bindings{
		"move_x": []input.Input{
			input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD},
		},
		"move_y": []input.Input{
			input.KeyPairInput{Negative: input.KeyW, Positive: input.KeyS},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("My Game"),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}

	player, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{
			Drawable: render.RectShape{Size: geom.Vector2{X: 48, Y: 48}},
			Color:    color.RGBA{R: 80, G: 180, B: 240, A: 255},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create player: %w", err)
	}

	if err := g.AddSystem(core.PhaseFixed, "player-movement", core.SystemFunc(func(ctx *core.Context) error {
		moveX := ctx.Actions.Axis("move_x")
		moveY := ctx.Actions.Axis("move_y")
		length := math.Hypot(moveX, moveY)
		if length > 1 {
			moveX /= length
			moveY /= length
		}
		if moveX == 0 && moveY == 0 {
			return nil
		}

		distance := playerSpeed * ctx.DeltaTime.Seconds()
		if err := player.Update(ctx.World, func(t *core.Transform) {
			t.Position.X += moveX * distance
			t.Position.Y += moveY * distance
		}); err != nil {
			return fmt.Errorf("move player: %w", err)
		}
		return nil
	})); err != nil {
		return nil, fmt.Errorf("register player movement: %w", err)
	}

	return g, nil
}
