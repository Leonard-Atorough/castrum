// Command text demonstrates text rendering: a static label, a
// per-tick counter whose string updates like a score display, and
// tinted and scaled text. Text is a drawable like any other - it
// lives on a Sprite with a Transform, so it culls, sorts, moves,
// and fades exactly like a shape or a texture.
//
// The built-in font is preloaded through the asset loader; each sprite names
// it, the collector measures it, and the runner draws it.
//
// Run from this directory:
//
//	go run .
package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

const (
	screenW, screenH = 1280, 720

	goFont = builtin.Font

	titleY = 200
	scoreY = 360
	notesY = 520
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g := castrum.New(
		castrum.WithTitle("castrum - text"),
	)

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	if _, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: titleY}, Scale: geom.Vector2{X: 1, Y: 1}},
		sprite.Text{Asset: goFont, Text: "hello, castrum", TextSize: 48},
	); err != nil {
		return err
	}

	score, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: scoreY}, Scale: geom.Vector2{X: 1, Y: 1}},
		sprite.Text{Asset: goFont, Text: "tick 0", TextSize: 32},
	)
	if err != nil {
		return err
	}

	if _, err := g.World().NewEntity(
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: notesY},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		sprite.Text{
			Asset:    goFont,
			Text:     "text culls, sorts, and fades like a sprite",
			TextSize: 24,
			Style:    sprite.Style{Color: &color.RGBA{R: 120, G: 200, B: 255, A: 255}},
		},
	); err != nil {
		return err
	}

	if _, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: screenH - 24}, Scale: geom.Vector2{X: 1, Y: 1}},
		sprite.Text{Asset: goFont, Text: "the counter is one sprite whose string is rewritten each tick", TextSize: 20},
	); err != nil {
		return err
	}

	elapsed := time.Duration(0)
	if err := g.AddSystem(core.PhaseFixed, "score", core.SystemFunc(func(ctx *core.Context) error {
		elapsed += ctx.DeltaTime
		return score.Update(ctx.World, func(s *sprite.Text) {
			s.Text = fmt.Sprintf("score: %d", int(elapsed.Seconds()))
		})
	})); err != nil {
		return err
	}

	ebitrun.New().Run(g)
	return nil
}
