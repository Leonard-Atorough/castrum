// Command scenes demonstrates scene switching, additive scene loads,
// lifecycle hooks, and queries over director-owned entities.
//
// SPACE switches between the hangar and the salvage field; S toggles a
// scanner overlay while the salvage field is live. Scene requests queue in
// the director and are applied by the scene system, so their effects appear
// on the frame after the press.
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

// spin marks an entity the "hangar spin" system rotates, in radians per second.
type spin struct{ speed float64 }

// drift marks an entity the "salvage drift" system moves, in pixels per second.
type drift struct{ speed float64 }

// stage couples a blueprint with the name and hooks the director receives
// whenever that scene is requested.
type stage struct {
	name  string
	scene scene.Scene
	hooks scene.Hooks
}

// hangar and field are the roots SPACE switches between; scanner is the
// additive overlay S toggles while the field is live.
var (
	hangar  = stage{name: "hangar", scene: hangarScene, hooks: locationHooks("hangar")}
	field   = stage{name: "field", scene: fieldScene, hooks: locationHooks("salvage field")}
	scanner = stage{name: "scanner", scene: scannerScene, hooks: scannerHooks()}
)

// demo holds the director and the instance ids the controls requested.
type demo struct {
	game     *castrum.Game
	director *scene.Director
	root     scene.InstanceID // the root SPACE controls, from queue to teardown
	scanner  scene.InstanceID // the scanner overlay, zero while absent
}

// atField reports whether the salvage field root is live. Controls read the
// director's live roots instead of the recorded ids, so presses while a
// switch is queued act on the scene that is on screen.
func (d *demo) atField() bool {
	for _, root := range d.director.Roots() {
		if name, ok := d.director.Name(root); ok && name == field.name {
			return true
		}
	}
	return false
}

// controls turns the two actions into director requests and nothing else;
// travel is checked first so both presses queue in one frame.
func (d *demo) controls(ctx *core.Context) error {
	if ctx.Actions.JustPressed("travel") {
		if err := d.travel(); err != nil {
			return err
		}
	}
	if ctx.Actions.JustPressed("scanner") {
		return d.toggleScanner()
	}
	return nil
}

func (d *demo) travel() error {
	if d.atField() {
		return d.switchStage(hangar)
	}
	return d.switchStage(field)
}

// switchStage queues s to replace every live root and records the reserved
// instance id.
func (d *demo) switchStage(s stage) error {
	id, err := d.director.Switch(s.scene, nil, scene.WithName(s.name), scene.WithHooks(s.hooks))
	if err != nil {
		return err
	}
	d.root = id
	return nil
}

// toggleScanner loads the scanner overlay alongside the field, or queues its
// removal when it is already live.
func (d *demo) toggleScanner() error {
	if !d.atField() {
		return nil
	}
	if d.director.Live(d.scanner) {
		return d.director.Remove(d.scanner)
	}
	id, err := d.director.Load(scanner.scene, nil, scene.WithName(scanner.name), scene.WithHooks(scanner.hooks))
	if err != nil {
		return err
	}
	d.scanner = id
	return nil
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
	g.World().MustResource[*debug.Panel]().Status("controls", "SPACE: travel   S: toggle scanner in field")

	// Queries and systems work on scene entities like any other entity, so
	// they are registered once and run regardless of which scene is live.
	if err := d.addSpinSystem(); err != nil {
		return nil, err
	}
	if err := d.addDriftSystem(); err != nil {
		return nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "scene controls", core.SystemFunc(d.controls)); err != nil {
		return nil, err
	}

	id, err := d.director.Load(hangar.scene, nil, scene.WithName(hangar.name), scene.WithHooks(hangar.hooks))
	if err != nil {
		return nil, err
	}
	d.root = id
	return d, nil
}

// hangarScene builds the landing bay: a backdrop with a ship spinning in place.
func hangarScene(any) (scene.Node, error) {
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
}

// fieldScene builds the salvage field: a backdrop with six pieces of drifting debris.
func fieldScene(any) (scene.Node, error) {
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
			sprite.Sprite{Drawable: sprite.RectShape{Size: geom.Vector2{X: 1280, Y: 460}}, Style: sprite.Style{Color: color.NRGBA{R: 20, G: 35, B: 60, A: 255}}},
		},
		Children: debris,
	}, nil
}

// scannerScene builds a circular scanner outline laid over the field.
func scannerScene(any) (scene.Node, error) {
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
}

// locationHooks labels the debug panel while the location is entering.
func locationHooks(name string) scene.Hooks {
	return scene.Hooks{Enter: func(ctx *core.Context, _ *scene.Handle) error {
		ctx.World.MustResource[*debug.Panel]().Status("location", "Location: "+name)
		return nil
	}}
}

// scannerHooks reports the overlay entering and clears the label when it exits.
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

// addSpinSystem registers the fixed system rotating entities with a spin.
func (d *demo) addSpinSystem() error {
	query := core.NewQuery(d.game.World()).With(core.Transform{}, spin{})
	return d.game.AddSystem(core.PhaseFixed, "hangar spin", core.SystemFunc(func(ctx *core.Context) error {
		for entity := range query.Execute() {
			motion, _ := entity.Component[spin]()
			entity.Update(func(t *core.Transform) {
				t.Rotation += motion.speed * ctx.DeltaTime.Seconds()
			})
		}
		return nil
	}))
}

// addDriftSystem registers the fixed system drifting entities with a drift
// and wrapping them back to the field's left edge.
func (d *demo) addDriftSystem() error {
	query := core.NewQuery(d.game.World()).With(core.Transform{}, drift{})
	return d.game.AddSystem(core.PhaseFixed, "salvage drift", core.SystemFunc(func(ctx *core.Context) error {
		for entity := range query.Execute() {
			motion, _ := entity.Component[drift]()
			entity.Update(func(t *core.Transform) {
				t.Position.X += motion.speed * ctx.DeltaTime.Seconds()
				if t.Position.X > 1300 {
					t.Position.X = -120
				}
			})
		}
		return nil
	}))
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

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
