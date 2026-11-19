// Command scenes demonstrates scene switching, additive scene loads,
// lifecycle hooks, and queries over director-owned entities - with every
// scene's content authored in JSON files instead of Go builders.
//
// SPACE switches between the hangar and the salvage field; S toggles a
// scanner overlay while the salvage field is live. Scene requests queue in
// the director and are applied by the scene system, so their effects appear
// on the frame after the press.
//
// The three scene files are data: nodes reference engine components by
// registry key (core.Transform, sprite.Primitive, sprite.Text) and the
// example's own gameplay state by a key the example registers
// (scenes.spin, scenes.drift). The hangar preloads the built-in font
// through its assets section, so the location is a self-contained unit:
// entering it loads what it draws.
//
// Run from this directory:
//
//	go run .
package main

import (
	"embed"
	"encoding/json"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/serialization"
)

// spin marks an entity the "hangar spin" system rotates, in radians per
// second. Authored under the "scenes.spin" registry key, so the JSON scene
// attaches it to the ship like any engine component.
type spin struct {
	Speed float64 `json:"speed"`
}

// Validate keeps the file contract sharp: a scene that ships a spinning
// part without a rotation rate fails at spawn, and an authored file
// holding a non-positive speed fails at decode.
func (s spin) Validate() error {
	if s.Speed <= 0 {
		return errSpinSpeed
	}
	return nil
}

var errSpinSpeed = &fileError{"spin speed must be positive"}

type fileError struct{ msg string }

func (e *fileError) Error() string { return e.msg }

// drift marks an entity the "salvage drift" system moves, in pixels per
// second, authored under "scenes.drift".
type drift struct {
	Speed float64 `json:"speed"`
}

func (d drift) Validate() error {
	if d.Speed <= 0 {
		return errDriftSpeed
	}
	return nil
}

var errDriftSpeed = &fileError{"drift speed must be positive"}

// sceneFiles holds the authored scenes: hangar, field, and scanner overlay.
//
//go:embed hangar.json field.json scanner.json
var sceneFiles embed.FS

// sceneFrom compiles one authored file into a scene. All failures name
// the file, the registry key, and the node path, the engine's boundary
// error promise.
func sceneFrom(reg *serialization.Registry, name string) (scene.Scene, error) {
	raw, err := sceneFiles.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var file serialization.File
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	return file.Scene(reg)
}

// stage couples an authored scene file with the name and hooks the
// director receives whenever that scene is requested.
type stage struct {
	name  string
	file  string
	hooks scene.Hooks
	run   scene.Scene // compiled once during setup
}

var (
	hangar  = stage{name: "hangar", file: "hangar.json", hooks: locationHooks("hangar")}
	field   = stage{name: "field", file: "field.json", hooks: locationHooks("salvage field")}
	scanner = stage{name: "scanner", file: "scanner.json", hooks: scannerHooks()}
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
	id, err := d.director.Switch(s.run, nil, scene.WithName(s.name), scene.WithHooks(s.hooks))
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
	id, err := d.director.Load(scanner.run, nil, scene.WithName(scanner.name), scene.WithHooks(scanner.hooks))
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

	// The example's gameplay state crosses the data boundary like any
	// engine component: registered under a stable key, then authored in
	// JSON. The registry is provided by castrum.New with the engine's
	// own components already registered.
	reg := g.World().MustResource[*serialization.Registry]()
	if err := reg.Component[spin]("scenes.spin"); err != nil {
		return nil, err
	}
	if err := reg.Component[drift]("scenes.drift"); err != nil {
		return nil, err
	}

	// Compile the authored scenes once; switching and reloading reuse the
	// compiled nodes.
	for _, s := range []*stage{&hangar, &field, &scanner} {
		compiled, err := sceneFrom(reg, s.file)
		if err != nil {
			return nil, err
		}
		s.run = compiled
	}

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

	id, err := d.director.Load(hangar.run, nil, scene.WithName(hangar.name), scene.WithHooks(hangar.hooks))
	if err != nil {
		return nil, err
	}
	d.root = id
	return d, nil
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
				t.Rotation += motion.Speed * ctx.DeltaTime.Seconds()
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
				t.Position.X += motion.Speed * ctx.DeltaTime.Seconds()
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
