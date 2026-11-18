// Command scenes demonstrates scene transitions and entity queries.
// Space switches between the hangar and salvage field; S toggles the scanner.
//
// Run from this directory:
//
//	go run .
package main

import (
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/sprite"
)

type spin struct{ speed float64 }
type drift struct{ speed float64 }

// demo tracks the ids the scene controls hand out. Travel direction and
// scanner gating read the director's live roots instead of mirroring them,
// so queued-but-unapplied switches cannot drift from what is on screen.
type demo struct {
	game     *castrum.Game
	director *scene.Director
	root     scene.InstanceID // the latest root queue, live once applied
	scanner  scene.InstanceID // the scanner overlay, zero while absent
}

// atField reports whether the salvage field is currently live.
func (d *demo) atField() bool {
	for _, root := range d.director.Roots() {
		if name, ok := d.director.Name(root); ok && name == "field" {
			return true
		}
	}
	return false
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	d, err := newGame()
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(d.game, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	return d.game.Run(runner)
}

func newGame() (*demo, error) {
	g, err := castrum.New(
		castrum.WithTitle("Scenes"),
		castrum.WithScenes(),
		castrum.WithBindings(input.Bindings{
			"travel":  []input.Input{input.KeyInput{Key: input.KeySpace}.Binding()},
			"scanner": []input.Input{input.KeyInput{Key: input.KeyS}.Binding()},
		}),
	)
	if err != nil {
		return nil, err
	}
	if err := g.MainCamera().Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: 640, Y: 360}
	}); err != nil {
		return nil, err
	}
	d := &demo{game: g, director: g.World().MustResource[*scene.Director]()}
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "SPACE: travel   S: toggle scanner in field")

	hangar := scene.Scene(func(any) (scene.Node, error) {
		return scene.Node{
			Key: "hangar",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
				sprite.Sprite{Drawable: sprite.RectShape{Size: geom.Vector2{X: 460, Y: 240}}, Style: sprite.Style{Color: color.NRGBA{R: 30, G: 65, B: 90, A: 255}}},
			},
			Children: []scene.Node{{
				Key: "ship",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
					sprite.Sprite{Drawable: sprite.RectShape{Size: geom.Vector2{X: 120, Y: 60}}, Style: sprite.Style{Color: color.NRGBA{R: 90, G: 210, B: 240, A: 255}, Layer: 1}},
					spin{speed: 0.6},
				},
			}},
		}, nil
	})
	hangarHooks := scene.Hooks{
		Enter: func(*core.Context, *scene.Handle) error {
			panel.Status("location", "Location: hangar")
			return nil
		},
	}
	field := scene.Scene(func(any) (scene.Node, error) {
		debris := make([]scene.Node, 0, 6)
		for i := range 6 {
			debris = append(debris, scene.Node{
				Key: string(rune('A' + i)),
				Components: []any{
					core.Transform{Position: geom.Vector2{X: float64(260 + i*130), Y: float64(270 + (i%3)*120)}},
					sprite.Sprite{Drawable: sprite.RectShape{Size: geom.Vector2{X: 55, Y: 35}}, Style: sprite.Style{Color: color.NRGBA{R: 220, G: 145, B: 75, A: 255}, Layer: 1}},
					drift{speed: float64(25 + i*8)},
				},
			})
		}
		return scene.Node{
			Key: "field",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
				sprite.Sprite{Drawable: sprite.RectShape{Size: geom.Vector2{X: 1050, Y: 460}}, Style: sprite.Style{Color: color.NRGBA{R: 20, G: 35, B: 60, A: 255}}},
			},
			Children: debris,
		}, nil
	})
	fieldHooks := scene.Hooks{
		Enter: func(*core.Context, *scene.Handle) error {
			panel.Status("location", "Location: salvage field")
			return nil
		},
	}
	scanner := scene.Scene(func(any) (scene.Node, error) {
		return scene.Node{
			Key: "scanner",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
				sprite.Sprite{
					Drawable: sprite.CircleShape{Radii: geom.Vector2{X: 190, Y: 190}},
					Style:    sprite.Style{Layer: 2, Outline: true, StrokeWidth: 3, Color: color.NRGBA{R: 90, G: 240, B: 130, A: 255}},
				},
			},
		}, nil
	})
	scannerHooks := scene.Hooks{
		Enter: func(*core.Context, *scene.Handle) error {
			panel.Status("scanner", "Scanner: active")
			return nil
		},
		Exit: func(*core.Context, *scene.Handle) error {
			panel.Status("scanner", "")
			return nil
		},
	}

	spinQuery := core.NewQuery(g.World()).With(core.Transform{}, spin{})
	if err := g.AddSystem(core.PhaseFixed, "hangar spin", core.SystemFunc(func(ctx *core.Context) error {
		for entity := range spinQuery.Execute() {
			motion, _ := entity.Component[spin]()
			entity.Update(func(t *core.Transform) { t.Rotation += motion.speed * ctx.DeltaTime.Seconds() })
		}
		return nil
	})); err != nil {
		return nil, err
	}
	driftQuery := core.NewQuery(g.World()).With(core.Transform{}, drift{})
	if err := g.AddSystem(core.PhaseFixed, "salvage drift", core.SystemFunc(func(ctx *core.Context) error {
		for entity := range driftQuery.Execute() {
			motion, _ := entity.Component[drift]()
			entity.Update(func(t *core.Transform) {
				t.Position.X += motion.speed * ctx.DeltaTime.Seconds()
				if t.Position.X > 1160 {
					t.Scale = geom.Vector2{X: 0, Y: 0}
					t.Position.X = 120
					t.Scale = geom.Vector2{X: 1, Y: 1}
				}
			})
		}
		return nil
	})); err != nil {
		return nil, err
	}

	if err := g.AddSystem(core.PhaseFrame, "scene controls", core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Actions.JustPressed("travel") {
			next, name, hooks := field, "field", fieldHooks
			if d.atField() {
				next, name, hooks = hangar, "hangar", hangarHooks
			}
			id, err := d.director.Switch(next, nil, scene.WithName(name), scene.WithHooks(hooks))
			if err != nil {
				return err
			}
			d.root = id
			return nil
		}
		if ctx.Actions.JustPressed("scanner") {
			if !d.atField() {
				return nil
			}
			if d.director.Live(d.scanner) {
				return d.director.Remove(d.scanner)
			}
			id, err := d.director.Load(scanner, nil, scene.WithName("scanner"), scene.WithHooks(scannerHooks))
			if err != nil {
				return err
			}
			d.scanner = id
		}
		return nil
	})); err != nil {
		return nil, err
	}
	id, err := d.director.Load(hangar, nil, scene.WithName("hangar"), scene.WithHooks(hangarHooks))
	if err != nil {
		return nil, err
	}
	d.root = id
	return d, nil
}
