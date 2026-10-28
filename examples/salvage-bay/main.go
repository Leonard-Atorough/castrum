// Command salvage-bay demonstrates scene transitions and entity queries.
// Space switches between the hangar and salvage field; S toggles the scanner.
//
// Run from the repository root:
//
//	go run ./examples/salvage-bay
package main

import (
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/scene"
)

type spin struct{ speed float64 }
type drift struct{ speed float64 }

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := newGame()
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	return g.Run(runner)
}

func newGame() (*castrum.Game, error) {
	g, err := castrum.New(
		castrum.WithTitle("Salvage Bay"),
		castrum.WithScenes(),
		castrum.WithBindings(input.Bindings{
			"travel":  []input.Input{input.KeyInput{Key: input.KeySpace}},
			"scanner": []input.Input{input.KeyInput{Key: input.KeyS}},
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

	director := g.World().MustResource[*scene.Director]()
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "SPACE: travel   S: toggle scanner in field")

	hangar := &scene.Blueprint{
		Name: "hangar",
		Hooks: scene.Hooks{Enter: func(*core.Context, *scene.Instance) error {
			panel.Status("location", "Location: hangar")
			return nil
		}},
		EntityList: []scene.EntitySpec{
			shape("dock", 640, 400, 460, 240, 0, color.NRGBA{R: 30, G: 65, B: 90, A: 255}),
			{
				Name: "ship",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
					render.Sprite{Drawable: render.RectShape{Size: geom.Vector2{X: 120, Y: 60}}, Color: color.NRGBA{R: 90, G: 210, B: 240, A: 255}, Layer: 1},
					spin{speed: 0.6},
				},
			},
		},
	}
	field := &scene.Blueprint{
		Name: "field",
		Hooks: scene.Hooks{Enter: func(*core.Context, *scene.Instance) error {
			panel.Status("location", "Location: salvage field")
			return nil
		}},
		EntityList: []scene.EntitySpec{
			shape("background", 640, 400, 1050, 460, 0, color.NRGBA{R: 20, G: 35, B: 60, A: 255}),
		},
	}
	for i := range 6 {
		debris := shape("debris "+string(rune('A'+i)), float64(260+i*130), float64(270+(i%3)*120), 55, 35, 1,
			color.NRGBA{R: 220, G: 145, B: 75, A: 255})
		debris.Components = append(debris.Components, drift{speed: float64(25 + i*8)})
		field.EntityList = append(field.EntityList, debris)
	}
	scanner := &scene.Blueprint{
		Name: "scanner",
		Hooks: scene.Hooks{
			Enter: func(*core.Context, *scene.Instance) error {
				panel.Status("scanner", "Scanner: active")
				return nil
			},
			Exit: func(*core.Context, *scene.Instance) error {
				panel.Status("scanner", "")
				return nil
			},
		},
		EntityList: []scene.EntitySpec{{
			Name: "scanner ring",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 640, Y: 400}},
				render.Sprite{
					Drawable: render.CircleShape{Radii: geom.Vector2{X: 190, Y: 190}}, Layer: 2,
					Outline: true, StrokeWidth: 3, Color: color.NRGBA{R: 90, G: 240, B: 130, A: 255},
				},
			},
		}},
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
			next := hangar
			if _, ok := director.ByName("hangar"); ok {
				next = field
			}
			_, err := director.Switch(next)
			return err
		}
		if ctx.Actions.JustPressed("scanner") {
			if _, ok := director.ByName("field"); !ok {
				return nil
			}
			if active, ok := director.ByName("scanner"); ok {
				return director.Remove(active.ID)
			}
			_, err := director.Load(scanner)
			return err
		}
		return nil
	})); err != nil {
		return nil, err
	}
	if _, err := director.Load(hangar); err != nil {
		return nil, err
	}
	return g, nil
}

func shape(name string, x, y, width, height float64, layer uint8, tint color.NRGBA) scene.EntitySpec {
	return scene.EntitySpec{
		Name: name,
		Components: []any{
			core.Transform{Position: geom.Vector2{X: x, Y: y}},
			render.Sprite{Drawable: render.RectShape{Size: geom.Vector2{X: width, Y: height}}, Color: tint, Layer: layer},
		},
	}
}
