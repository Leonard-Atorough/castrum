// Command scenes demonstrates scene-owned entities and lifecycle hooks.
//
// SPACE switches between two rooms. S loads or removes a scanner alongside
// the current room. Requests made by the controls are applied on the next
// frame; switching removes both the old room and any loaded scanner.
//
// The field's entry hook spawns moving obstacles through the director, so
// they are removed with the field. The movement system is registered once
// and queries them in the same world as every other entity.
//
// Run from this directory:
//
//	go run .
package main

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/sprite"
)

type roomProps struct {
	name       string
	background color.RGBA
}

var (
	hangar = roomProps{name: "hangar", background: color.RGBA{R: 30, G: 65, B: 90, A: 255}}
	field  = roomProps{name: "field", background: color.RGBA{R: 20, G: 35, B: 60, A: 255}}
)

// roomScene is a reusable blueprint: each call builds a node tree from props.
// Child nodes are separate entities; their positions are world coordinates.
var roomScene = scene.Of(func(p roomProps) scene.Node {
	return scene.Node{
		Key:    p.name,
		Assets: []scene.Preload{scene.Warm[asset.FontData](builtin.Font)},
		Components: []any{
			core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
			sprite.Primitive{
				Shape: sprite.ShapeRect,
				Rect:  geom.Vector2{X: 1280, Y: 460},
				Style: sprite.Style{Color: &p.background},
			},
		},
		Children: []scene.Node{{
			Key: "title",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 640, Y: 220}},
				sprite.Text{Asset: builtin.Font, Text: p.name, TextSize: 28},
			},
		}},
	}
})

var scannerScene = scene.Of(func(_ struct{}) scene.Node {
	return scene.Node{
		Key: "scanner",
		Components: []any{
			core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
			sprite.Primitive{
				Shape: sprite.ShapeCircle,
				Radii: geom.Vector2{X: 190, Y: 190},
				Style: sprite.Style{
					Layer: 2, Outline: true, StrokeWidth: 3,
					Color: &color.RGBA{R: 90, G: 240, B: 130, A: 255},
				},
			},
		},
	}
})

type drift struct {
	speed float64
}

type demo struct {
	game     *castrum.Game
	director *scene.Director
	location string
	room     scene.InstanceID
	scanner  scene.InstanceID
}

func newGame() (*demo, error) {
	g := castrum.New(
		castrum.WithTitle("Scenes"),
		castrum.WithScenes(),
		castrum.WithBindings(input.Bindings{
			"travel":  []input.Input{input.KeyInput{Key: input.KeySpace}.Binding()},
			"scanner": []input.Input{input.KeyInput{Key: input.KeyS}.Binding()},
		}),
	)
	if err := g.MainCamera().Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: 640, Y: 360}
	}); err != nil {
		return nil, err
	}

	d := &demo{game: g, director: g.World().MustResource[*scene.Director]()}
	g.World().MustResource[*debug.Panel]().Status("controls", "SPACE: switch rooms   S: toggle scanner")

	// Scene entities use ordinary components, so one fixed system can move
	// field obstacles regardless of how or when they were spawned.
	moving := core.NewQuery(g.World()).With(core.Transform{}, drift{})
	if err := g.AddSystem(core.PhaseFixed, "drift", core.SystemFunc(func(ctx *core.Context) error {
		for entity := range moving.Execute() {
			motion, _ := entity.Component[drift]()
			entity.Update(func(t *core.Transform) {
				t.Position.X += motion.speed * ctx.DeltaTime.Seconds()
				if t.Position.X > 1300 {
					t.Position.X = -120
				}
			})
		}
		return nil
	})); err != nil {
		return nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "scene controls", core.SystemFunc(d.controls)); err != nil {
		return nil, err
	}

	// Load queues the first room for the scene startup system.
	room, err := d.director.Load(roomScene, hangar,
		scene.WithName(hangar.name), scene.WithHooks(d.roomHooks(hangar.name)))
	if err != nil {
		return nil, err
	}
	d.room = room
	return d, nil
}

func (d *demo) controls(ctx *core.Context) error {
	// Give switching priority if both actions occur in the same frame.
	if ctx.Actions.JustPressed("travel") {
		next := field
		if d.location == field.name {
			next = hangar
		}
		id, err := d.director.Switch(roomScene, next,
			scene.WithName(next.name), scene.WithHooks(d.roomHooks(next.name)))
		if err != nil {
			return err
		}
		d.room = id
		return nil
	}
	if ctx.Actions.JustPressed("scanner") {
		if d.director.Live(d.scanner) {
			return d.director.Remove(d.scanner)
		}
		id, err := d.director.Load(scannerScene, struct{}{},
			scene.WithName("scanner"), scene.WithHooks(scannerHooks()))
		if err != nil {
			return err
		}
		d.scanner = id
	}
	return nil
}

func (d *demo) roomHooks(name string) scene.Hooks {
	return scene.Hooks{
		Enter: func(ctx *core.Context, h *scene.Handle) error {
			if name == field.name {
				for i := range 6 {
					// Spawn associates each obstacle with this scene instance.
					_, err := d.director.Spawn(ctx, h.ID(),
						core.Transform{Position: geom.Vector2{X: 260 + float64(i)*130, Y: 270 + float64(i%3)*120}},
						sprite.Primitive{
							Shape: sprite.ShapeRect,
							Rect:  geom.Vector2{X: 55, Y: 35},
							Style: sprite.Style{Color: &color.RGBA{R: 220, G: 145, B: 75, A: 255}, Layer: 1},
						},
						drift{speed: 25 + float64(i)*8},
					)
					if err != nil {
						return fmt.Errorf("spawn obstacle %d: %w", i+1, err)
					}
				}
			}
			d.location = name
			panel := ctx.World.MustResource[*debug.Panel]()
			panel.Status("location", "Location: "+name)
			panel.Message("Enter " + name)
			return nil
		},
		Exit: func(ctx *core.Context, _ *scene.Handle) error {
			// A replacement enters before the old room exits, so do not clear
			// the shared location status here.
			ctx.World.MustResource[*debug.Panel]().Message("Exit " + name)
			return nil
		},
	}
}

func scannerHooks() scene.Hooks {
	return scene.Hooks{
		Enter: func(ctx *core.Context, _ *scene.Handle) error {
			ctx.World.MustResource[*debug.Panel]().Status("scanner", "Scanner: active")
			return nil
		},
		Exit: func(ctx *core.Context, _ *scene.Handle) error {
			ctx.World.MustResource[*debug.Panel]().Status("scanner", "")
			return nil
		},
	}
}

func run() error {
	d, err := newGame()
	if err != nil {
		return err
	}
	runner := ebitrun.New(ebitrun.WithDebugOverlay())
	runner.Run(d.game)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
