// Command signal-salvage is a small, asset-free game built from Castrum's
// scenes, signals, collision, timers, system groups, and built-in font.
// Collect five cores before time runs out; avoid the moving pulse.
//
// Run from the repository root:
//
//	go run ./examples/signal-salvage
package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/timer"
)

const (
	screenWidth  = 960
	screenHeight = 540
	roundLength  = 20 * time.Second
	coreGoal     = 5
	playerSpeed  = 280.0
	roundGroup   = "round"
)

const (
	playerLayer = iota
	coreLayer
	droneLayer
)

var (
	backgroundColor = color.NRGBA{R: 8, G: 18, B: 36, A: 255}
	fieldColor      = color.NRGBA{R: 13, G: 31, B: 54, A: 255}
	borderColor     = color.NRGBA{R: 47, G: 86, B: 111, A: 255}
	playerColor     = color.NRGBA{R: 74, G: 226, B: 231, A: 255}
	coreColor       = color.NRGBA{R: 255, G: 217, B: 93, A: 255}
	droneColor      = color.NRGBA{R: 255, G: 91, B: 105, A: 255}
)

type pilot struct{}

type dataCore struct {
	collected bool
}

type pulseDrone struct {
	phase float64
}

type roundClock struct{}

type scoreLabel struct{}

type clockLabel struct{}

type coreCollected struct{}

type roundEnded struct {
	won bool
}

type runState struct {
	active           bool
	finished         bool
	won              bool
	score            int
	displayedSeconds int
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	game, _, err := newGame()
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(game, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	return game.Run(runner)
}

func newGame() (*castrum.Game, *runState, error) {
	bindings := input.Bindings{
		"move_x": {
			input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD},
			input.KeyPairInput{Negative: input.KeyArrowLeft, Positive: input.KeyArrowRight},
		},
		"move_y": {
			input.KeyPairInput{Negative: input.KeyW, Positive: input.KeyS},
			input.KeyPairInput{Negative: input.KeyArrowUp, Positive: input.KeyArrowDown},
		},
		"start":   {input.KeyInput{Key: input.KeySpace}},
		"restart": {input.KeyInput{Key: input.KeyEnter}},
	}
	game, err := castrum.New(
		castrum.WithTitle("Signal Salvage"),
		castrum.WithBindings(bindings),
		castrum.WithCollision(),
		castrum.WithScenes(),
		castrum.WithTimer(),
	)
	if err != nil {
		return nil, nil, err
	}
	if err := game.MainCamera().Update(game.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}
	}); err != nil {
		return nil, nil, err
	}

	loader := game.World().MustResource[*asset.Loader]()
	if err := builtin.Register(loader); err != nil {
		return nil, nil, err
	}

	director := game.World().MustResource[*scene.Director]()
	hub := game.World().MustResource[*signal.Hub]()
	panel := game.World().MustResource[*debug.Panel]()
	state := &runState{displayedSeconds: -1}

	arena, scoreQuery, clockQuery, playerQuery, droneQuery, err := makeArenaBlueprint(game.World())
	if err != nil {
		return nil, nil, err
	}
	briefing := makeBriefingBlueprint()

	if err := addRoundSystems(game, director, hub, panel, state, scoreQuery, clockQuery, playerQuery, droneQuery); err != nil {
		return nil, nil, err
	}
	if err := game.SetGroupEnabled(roundGroup, false); err != nil {
		return nil, nil, err
	}
	if err := game.AddSystem(core.PhaseFrame, "demo.scene-controls", core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Actions.JustPressed("start") {
			if _, ready := director.ByName("briefing"); ready {
				state.active = true
				state.finished = false
				state.won = false
				state.score = 0
				state.displayedSeconds = -1
				if err := game.SetGroupEnabled(roundGroup, true); err != nil {
					return err
				}
				_, err := director.Switch(arena)
				return err
			}
		}
		if ctx.Actions.JustPressed("restart") {
			if _, done := director.ByName("result"); done {
				_, err := director.Switch(briefing)
				return err
			}
		}
		return nil
	})); err != nil {
		return nil, nil, err
	}
	panel.Status("controls", "WASD / arrows: move   SPACE: start   ENTER: briefing")
	panel.Status("objective", "Recover five cores. Avoid the red pulse.")
	if _, err := director.Load(briefing); err != nil {
		return nil, nil, err
	}
	return game, state, nil
}

func addRoundSystems(game *castrum.Game, director *scene.Director, hub *signal.Hub, panel *debug.Panel, state *runState, scoreQuery, clockQuery, playerQuery, droneQuery *core.Query) error {
	world := game.World()
	coreEvents := hub.NewReader[collision.Entered]()

	// Score signals are resolved before clock completions, so a core collected
	// on the final tick counts as a win rather than a timeout.
	systems := []struct {
		name string
		run  core.System
	}{
		{
			name: "round.player-motion",
			run: core.SystemFunc(func(ctx *core.Context) error {
				for entry := range playerQuery.Execute() {
					direction := geom.Vector2{
						X: ctx.Actions.Axis("move_x"),
						Y: ctx.Actions.Axis("move_y"),
					}
					if direction.Length() > 1 {
						direction = direction.Normalize()
					}
					entry.Update(func(t *core.Transform) {
						t.Position = t.Position.Add(direction.Mul(playerSpeed * ctx.DeltaTime.Seconds()))
						t.Position.X = math.Max(42, math.Min(screenWidth-42, t.Position.X))
						t.Position.Y = math.Max(104, math.Min(screenHeight-42, t.Position.Y))
					})
				}
				return nil
			}),
		},
		{
			name: "round.pulse-motion",
			run: core.SystemFunc(func(ctx *core.Context) error {
				for entry := range droneQuery.Execute() {
					drone, _ := entry.Component[pulseDrone]()
					drone.phase += ctx.DeltaTime.Seconds() * 1.5
					entry.SetComponent(drone)
					entry.Update(func(t *core.Transform) {
						t.Position = geom.Vector2{
							X: screenWidth/2 + math.Cos(drone.phase)*235,
							Y: 275 + math.Sin(drone.phase*1.7)*115,
						}
					})
				}
				return nil
			}),
		},
		{
			name: "round.collision-events",
			run: core.SystemFunc(func(ctx *core.Context) error {
				player, ok := playerQuery.First()
				if !ok {
					return nil
				}
				for event, ok := coreEvents.Next(); ok; event, ok = coreEvents.Next() {
					if !state.active || state.finished || event.Entity != player.ID() {
						continue
					}
					other := core.NewEntity(event.Other)
					if other.HasComponent[dataCore](world) {
						coreState, _ := other.Component[dataCore](world)
						if coreState.collected {
							continue
						}
						if err := other.Update(world, func(c *dataCore) { c.collected = true }); err != nil {
							return err
						}
						if err := other.Update(world, func(s *render.Sprite) { s.Hidden = true }); err != nil {
							return err
						}
						if err := other.Update(world, func(c *collision.Collider) { c.Active = false }); err != nil {
							return err
						}
						hub.Emit(coreCollected{})
						continue
					}
					if other.HasComponent[pulseDrone](world) {
						hub.Emit(roundEnded{})
					}
				}
				return nil
			}),
		},
		{
			name: "round.score",
			run: hub.Consumer(func(_ *core.Context, _ coreCollected) error {
				if !state.active || state.finished {
					return nil
				}
				state.score++
				panel.Message(fmt.Sprintf("CORE RECOVERED  %d / %d", state.score, coreGoal))
				if entry, ok := scoreQuery.First(); ok {
					entry.Update(func(sprite *render.Sprite) {
						sprite.Drawable = textDrawable(fmt.Sprintf("CORES  %d / %d", state.score, coreGoal), 24)
					})
				}
				if state.score == coreGoal {
					hub.Emit(roundEnded{won: true})
				}
				return nil
			}),
		},
		{
			name: "round.clock",
			run: hub.Consumer(func(_ *core.Context, event timer.Completed) error {
				if !state.active || state.finished ||
					!core.NewEntity(event.Entity).HasComponent[roundClock](world) {
					return nil
				}
				hub.Emit(roundEnded{})
				return nil
			}),
		},
		{
			name: "round.clock-display",
			run: core.SystemFunc(func(_ *core.Context) error {
				for entry := range clockQuery.Execute() {
					clock, _ := entry.Component[timer.Timer]()
					seconds := int(math.Ceil(max(0, clock.Duration-clock.Elapsed).Seconds()))
					if seconds == state.displayedSeconds {
						continue
					}
					state.displayedSeconds = seconds
					entry.Update(func(sprite *render.Sprite) {
						sprite.Drawable = textDrawable(fmt.Sprintf("TIME  %02d", seconds), 24)
					})
				}
				return nil
			}),
		},
		{
			name: "round.resolve",
			run: hub.Consumer(func(_ *core.Context, event roundEnded) error {
				if !state.active || state.finished {
					return nil
				}
				state.finished = true
				state.active = false
				state.won = event.won

				if err := game.SetGroupEnabled(roundGroup, false); err != nil {
					return err
				}

				message := "SIGNAL LOST"
				if event.won {
					message = "SIGNAL RECOVERED"
				}
				panel.Message(message)

				_, err := director.Switch(makeResultBlueprint(state.won, state.score))
				return err
			}),
		},
	}

	for _, entry := range systems {
		if err := game.AddSystemToGroup(core.PhaseFixed, roundGroup, entry.name, entry.run); err != nil {
			return err
		}
	}
	return nil
}

func makeBriefingBlueprint() *scene.Blueprint {
	return &scene.Blueprint{
		Name: "briefing",
		EntityList: []scene.EntitySpec{
			rectSpec("briefing-background", geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}, screenWidth, screenHeight, backgroundColor, 0),
			textSpec("title", geom.Vector2{X: screenWidth / 2, Y: 184}, "SIGNAL SALVAGE", 48, playerColor),
			textSpec("objective", geom.Vector2{X: screenWidth / 2, Y: 252}, "RECOVER 5 CORES  /  AVOID THE PULSE", 20, coreColor),
			textSpec("controls", geom.Vector2{X: screenWidth / 2, Y: 316}, "WASD OR ARROWS TO MOVE", 18, color.White),
			textSpec("start", geom.Vector2{X: screenWidth / 2, Y: 408}, "PRESS SPACE TO DEPLOY", 22, playerColor),
		},
	}
}

func makeArenaBlueprint(world *core.World) (*scene.Blueprint, *core.Query, *core.Query, *core.Query, *core.Query, error) {
	playerCollider, err := triggerCollider(16, playerLayer, coreLayer, droneLayer)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	coreCollider, err := triggerCollider(14, coreLayer, playerLayer)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	droneCollider, err := triggerCollider(26, droneLayer, playerLayer)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	specs := []scene.EntitySpec{
		rectSpec("field", geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}, screenWidth, screenHeight, backgroundColor, 0),
		rectSpec("arena", geom.Vector2{X: screenWidth / 2, Y: 286}, 900, 428, fieldColor, 1),
		{
			Name: "arena-border",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: 286}},
				render.Sprite{
					Drawable:    render.RectShape{Size: geom.Vector2{X: 900, Y: 428}},
					Outline:     true,
					StrokeWidth: 2,
					Color:       borderColor,
					Layer:       2,
				},
			},
		},
		{
			Name: "score",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 180, Y: 43}},
				render.Sprite{Drawable: textDrawable("CORES  0 / 5", 24), Color: coreColor, Layer: 5},
				scoreLabel{},
			},
		},
		{
			Name: "round-clock",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: 780, Y: 43}},
				render.Sprite{Drawable: textDrawable("TIME  20", 24), Color: color.White, Layer: 5},
				clockLabel{},
				roundClock{},
				timer.NewTimer(roundLength, false),
			},
		},
		textSpec("instructions", geom.Vector2{X: screenWidth / 2, Y: 519}, "COLLECT GOLD  /  DODGE RED", 16, borderColor),
		{
			Name: "pilot",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: 424}},
				render.Sprite{
					Drawable: render.CircleShape{Radii: geom.Vector2{X: 16, Y: 16}},
					Color:    playerColor,
					Layer:    4,
				},
				playerCollider,
				pilot{},
			},
		},
		{
			Name: "pulse-drone",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: screenWidth/2 + 235, Y: 275}},
				render.Sprite{
					Drawable:    render.CircleShape{Radii: geom.Vector2{X: 26, Y: 26}},
					Color:       droneColor,
					Outline:     true,
					StrokeWidth: 4,
					Layer:       4,
				},
				droneCollider,
				pulseDrone{},
			},
		},
	}
	for i, position := range []geom.Vector2{
		{X: 164, Y: 156},
		{X: 304, Y: 372},
		{X: 654, Y: 150},
		{X: 798, Y: 370},
		{X: 480, Y: 266},
	} {
		specs = append(specs, scene.EntitySpec{
			Name: fmt.Sprintf("core-%d", i+1),
			Components: []any{
				core.Transform{Position: position},
				render.Sprite{
					Drawable:    render.CircleShape{Radii: geom.Vector2{X: 12, Y: 12}},
					Color:       coreColor,
					Outline:     true,
					StrokeWidth: 3,
					Layer:       3,
				},
				coreCollider,
				dataCore{},
			},
		})
	}

	blueprint := &scene.Blueprint{Name: "arena", EntityList: specs}
	if err := blueprint.Validate(); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return blueprint,
		core.NewQuery(world).With(scoreLabel{}, render.Sprite{}),
		core.NewQuery(world).With(clockLabel{}, timer.Timer{}, render.Sprite{}),
		core.NewQuery(world).With(pilot{}, core.Transform{}),
		core.NewQuery(world).With(core.Transform{}, pulseDrone{}),
		nil
}

func makeResultBlueprint(won bool, score int) *scene.Blueprint {
	title, tint := "SIGNAL LOST", droneColor
	if won {
		title, tint = "SIGNAL RECOVERED", playerColor
	}
	return &scene.Blueprint{
		Name: "result",
		EntityList: []scene.EntitySpec{
			rectSpec("result-background", geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}, screenWidth, screenHeight, backgroundColor, 0),
			textSpec("result-title", geom.Vector2{X: screenWidth / 2, Y: 208}, title, 42, tint),
			textSpec("result-score", geom.Vector2{X: screenWidth / 2, Y: 278}, fmt.Sprintf("CORES RECOVERED  %d / %d", score, coreGoal), 20, coreColor),
			textSpec("result-prompt", geom.Vector2{X: screenWidth / 2, Y: 378}, "PRESS ENTER FOR BRIEFING", 20, color.White),
		},
	}
}

func triggerCollider(radius float64, layer int, masks ...int) (collision.Collider, error) {
	collider, err := collision.NewCollider(collision.Circle{Radius: radius})
	if err != nil {
		return collision.Collider{}, err
	}
	collider.Layers = collision.Layers(layer)
	collider.Mask = collision.Mask(masks...)
	collider.Trigger = true
	return collider, nil
}

func rectSpec(name string, position geom.Vector2, width, height float64, tint color.Color, layer uint8) scene.EntitySpec {
	return scene.EntitySpec{
		Name: name,
		Components: []any{
			core.Transform{Position: position},
			render.Sprite{
				Drawable: render.RectShape{Size: geom.Vector2{X: width, Y: height}},
				Color:    tint,
				Layer:    layer,
			},
		},
	}
}

func textSpec(name string, position geom.Vector2, text string, size float64, tint color.Color) scene.EntitySpec {
	return scene.EntitySpec{
		Name: name,
		Components: []any{
			core.Transform{Position: position},
			render.Sprite{Drawable: textDrawable(text, size), Color: tint, Layer: 5},
		},
	}
}

func textDrawable(text string, size float64) render.TextSource {
	return render.TextSource{Font: builtin.Font, Text: text, Size: size}
}
