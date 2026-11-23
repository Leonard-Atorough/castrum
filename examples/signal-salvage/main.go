// Command signal-salvage is a small, asset-free game built from Castrum's
// scenes, signals, collision, timers, system groups, and built-in font.
// Collect five cores before time runs out; avoid the moving pulse.
//
// Run from this directory:
//
//	go run .
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
	"github.com/Leonard-Atorough/castrum/gametime"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/signal"
	spr "github.com/Leonard-Atorough/castrum/sprite"
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
	backgroundColor = &color.RGBA{R: 8, G: 18, B: 36, A: 255}
	fieldColor      = &color.RGBA{R: 13, G: 31, B: 54, A: 255}
	borderColor     = &color.RGBA{R: 47, G: 86, B: 111, A: 255}
	playerColor     = &color.RGBA{R: 74, G: 226, B: 231, A: 255}
	coreColor       = &color.RGBA{R: 255, G: 217, B: 93, A: 255}
	droneColor      = &color.RGBA{R: 255, G: 91, B: 105, A: 255}
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

type resultProps struct {
	won   bool
	score int
}

type runState struct {
	active           bool
	finished         bool
	won              bool
	score            int
	displayedSeconds int

	// scene ids the controls hand out; gates such as "the briefing is on
	// screen" read the director instead of hunting for scene names
	briefing scene.InstanceID
	result   scene.InstanceID
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
	ebitrun.New(ebitrun.WithDebugOverlay()).Run(game)
	return nil
}

func newGame() (*castrum.Game, *runState, error) {
	bindings := input.Bindings{
		"move_x": {
			input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}.Binding(),
			input.KeyPairInput{Negative: input.KeyArrowLeft, Positive: input.KeyArrowRight}.Binding(),
		},
		"move_y": {
			input.KeyPairInput{Negative: input.KeyW, Positive: input.KeyS}.Binding(),
			input.KeyPairInput{Negative: input.KeyArrowUp, Positive: input.KeyArrowDown}.Binding(),
		},
		"start":   {input.KeyInput{Key: input.KeySpace}.Binding()},
		"restart": {input.KeyInput{Key: input.KeyEnter}.Binding()},
	}
	game := castrum.New(
		castrum.WithTitle("Signal Salvage"),
		castrum.WithBindings(bindings),
		castrum.WithScenes(),
	)
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

	briefing := makeBriefingScene()
	arena, err := makeArenaScene()
	if err != nil {
		return nil, nil, err
	}
	result := makeResultScene()

	world := game.World()
	if err := addRoundSystems(game, director, hub, panel, state, result,
		core.NewQuery(world).With(scoreLabel{}, spr.Text{}),
		core.NewQuery(world).With(clockLabel{}, gametime.Timer{}, spr.Text{}),
		core.NewQuery(world).With(pilot{}, core.Transform{}),
		core.NewQuery(world).With(core.Transform{}, pulseDrone{}),
	); err != nil {
		return nil, nil, err
	}
	if err := game.SetGroupEnabled(roundGroup, false); err != nil {
		return nil, nil, err
	}
	if err := game.AddSystem(core.PhaseFrame, "demo.scene-controls", func(ctx *core.Context) error {
		actions, _ := ctx.World.Resource[input.Actions]()
		if actions.JustPressed("start") {
			if !director.Live(state.briefing) {
				return nil
			}
			state.active = true
			state.finished = false
			state.won = false
			state.score = 0
			state.displayedSeconds = -1
			if err := game.SetGroupEnabled(roundGroup, true); err != nil {
				return err
			}
			if _, err := director.Switch(arena, nil, scene.WithName("arena")); err != nil {
				return err
			}
		}
		if actions.JustPressed("restart") {
			if !director.Live(state.result) {
				return nil
			}
			id, err := director.Switch(briefing, nil, scene.WithName("briefing"))
			if err != nil {
				return err
			}
			state.briefing = id
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	panel.Status("controls", "WASD / arrows: move   SPACE: start   ENTER: briefing")
	panel.Status("objective", "Recover five cores. Avoid the red pulse.")
	id, err := director.Load(briefing, nil, scene.WithName("briefing"))
	if err != nil {
		return nil, nil, err
	}
	state.briefing = id
	return game, state, nil
}

func addRoundSystems(game *castrum.Game, director *scene.Director, hub *signal.Hub, panel *debug.Panel, state *runState, result scene.Scene, scoreQuery, clockQuery, playerQuery, droneQuery *core.Query) error {
	world := game.World()
	coreEvents := hub.NewReader[collision.Entered]()

	// Score signals are resolved before clock completions, so a core collected
	// on the final tick counts as a win rather than a timeout.
	systems := []struct {
		name string
		run  core.SystemFunc
	}{
		{
			name: "round.player-motion",
			run: func(ctx *core.Context) error {
				actions, _ := ctx.World.Resource[input.Actions]()
				for entry := range playerQuery.Execute() {
					direction := geom.Vector2{
						X: actions.Axis("move_x"),
						Y: actions.Axis("move_y"),
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
			},
		},
		{
			name: "round.pulse-motion",
			run: func(ctx *core.Context) error {
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
			},
		},
		{
			name: "round.collision-events",
			run: func(ctx *core.Context) error {
				player, ok := playerQuery.First()
				if !ok {
					return nil
				}
				return coreEvents.Consume(ctx, func(_ *core.Context, event collision.Entered) error {
					if !state.active || state.finished || event.Entity != player.Entity() {
						return nil
					}
					other := event.Other
					if other.HasComponent[dataCore](world) {
						coreState, _ := other.Component[dataCore](world)
						if coreState.collected {
							return nil
						}
						if err := other.Update(world, func(c *dataCore) { c.collected = true }); err != nil {
							return err
						}
						if err := other.Update(world, func(c *collision.Collider) { c.Active = false }); err != nil {
							return err
						}
						if err := other.Update(world, func(p *spr.Primitive) { p.Style.Hidden = true }); err != nil {
							return err
						}
						hub.Emit(coreCollected{})
						return nil
					}
					if other.HasComponent[pulseDrone](world) {
						hub.Emit(roundEnded{})
					}
					return nil
				})
			},
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
					entry.Update(func(x *spr.Text) {
						x.Text = fmt.Sprintf("CORES  %d / %d", state.score, coreGoal)
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
			run: hub.Consumer(func(_ *core.Context, event gametime.Completed) error {
				if !state.active || state.finished ||
					!event.Entity.HasComponent[roundClock](world) {
					return nil
				}
				hub.Emit(roundEnded{})
				return nil
			}),
		},
		{
			name: "round.clock-display",
			run: func(_ *core.Context) error {
				for entry := range clockQuery.Execute() {
					clock, _ := entry.Component[gametime.Timer]()
					seconds := int(math.Ceil(max(0, clock.Duration-clock.Elapsed).Seconds()))
					if seconds == state.displayedSeconds {
						continue
					}
					state.displayedSeconds = seconds
					entry.Update(func(x *spr.Text) {
						x.Text = fmt.Sprintf("TIME  %02d", seconds)
					})
				}
				return nil
			},
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

				id, err := director.Switch(result, resultProps{won: event.won, score: state.score}, scene.WithName("result"))
				if err != nil {
					return err
				}
				state.result = id
				return nil
			}),
		},
	}

	for _, entry := range systems {
		if err := game.AddSystem(core.PhaseFixed, entry.name, entry.run, castrum.InGroup(roundGroup)); err != nil {
			return err
		}
	}
	return nil
}

func makeBriefingScene() scene.Scene {
	return scene.Scene(func(any) (scene.Node, error) {
		return scene.Node{
			Key: "briefing",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}},
				spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: screenWidth, Y: screenHeight}, Style: spr.Style{Color: backgroundColor}},
			},
			Children: []scene.Node{
				textNode("title", geom.Vector2{X: screenWidth / 2, Y: 184}, "SIGNAL SALVAGE", 48, playerColor),
				textNode("objective", geom.Vector2{X: screenWidth / 2, Y: 252}, "RECOVER 5 CORES  /  AVOID THE PULSE", 20, coreColor),
				textNode("controls", geom.Vector2{X: screenWidth / 2, Y: 316}, "WASD OR ARROWS TO MOVE", 18, &color.RGBA{R: 255, G: 255, B: 255, A: 255}),
				textNode("start", geom.Vector2{X: screenWidth / 2, Y: 408}, "PRESS SPACE TO DEPLOY", 22, playerColor),
			},
		}, nil
	})
}

func makeArenaScene() (scene.Scene, error) {
	playerCollider, err := triggerCollider(16, playerLayer, coreLayer, droneLayer)
	coreCollider, err := triggerCollider(14, coreLayer, playerLayer)
	if err != nil {
		return nil, err
	}
	droneCollider, err := triggerCollider(26, droneLayer, playerLayer)
	if err != nil {
		return nil, err
	}
	return scene.Scene(func(any) (scene.Node, error) {
		children := []scene.Node{
			{
				Key: "arena",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: 286}},
					spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 900, Y: 428}, Style: spr.Style{Color: fieldColor, Layer: 1}},
				},
			},
			{
				Key: "arena-border",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: 286}},
					spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: 900, Y: 428},
						Style: spr.Style{Outline: true, StrokeWidth: 2, Color: borderColor, Layer: 2}},
				},
			},
			{
				Key: "score",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: 180, Y: 43}},
					withTextStyle(textDrawable("CORES  0 / 5", 24), spr.Style{Color: coreColor, Layer: 5}),
					scoreLabel{},
				},
			},
			{
				Key: "round-clock",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: 780, Y: 43}},
					withTextStyle(textDrawable("TIME  20", 24), spr.Style{Color: &color.RGBA{R: 255, G: 255, B: 255, A: 255}, Layer: 5}),
					clockLabel{},
					roundClock{},
					gametime.NewTimer(roundLength, false),
				},
			},
			textNode("instructions", geom.Vector2{X: screenWidth / 2, Y: 519}, "COLLECT GOLD  /  DODGE RED", 16, borderColor),
			{
				Key: "pilot",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: 424}},
					spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 16, Y: 16},
						Style: spr.Style{Color: playerColor, Layer: 4}},
					playerCollider,
					pilot{},
				},
			},
			{
				Key: "pulse-drone",
				Components: []any{
					core.Transform{Position: geom.Vector2{X: screenWidth/2 + 235, Y: 275}},
					spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 26, Y: 26},
						Style: spr.Style{Color: droneColor, Outline: true, StrokeWidth: 4, Layer: 4}},
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
			children = append(children, scene.Node{
				Key: fmt.Sprintf("core-%d", i+1),
				Components: []any{
					core.Transform{Position: position},
					spr.Primitive{Shape: spr.ShapeCircle, Radii: geom.Vector2{X: 12, Y: 12},
						Style: spr.Style{Color: coreColor, Outline: true, StrokeWidth: 3, Layer: 3}},
					coreCollider,
					dataCore{},
				},
			})
		}
		return scene.Node{
			Key: "field",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}},
				spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: screenWidth, Y: screenHeight}, Style: spr.Style{Color: backgroundColor}},
			},
			Children: children,
		}, nil
	}), nil
}

func makeResultScene() scene.Scene {
	return scene.Of(func(p resultProps) scene.Node {
		title, tint := "SIGNAL LOST", droneColor
		if p.won {
			title, tint = "SIGNAL RECOVERED", playerColor
		}
		return scene.Node{
			Key: "result",
			Components: []any{
				core.Transform{Position: geom.Vector2{X: screenWidth / 2, Y: screenHeight / 2}},
				spr.Primitive{Shape: spr.ShapeRect, Rect: geom.Vector2{X: screenWidth, Y: screenHeight}, Style: spr.Style{Color: backgroundColor}},
			},
			Children: []scene.Node{
				textNode("result-title", geom.Vector2{X: screenWidth / 2, Y: 208}, title, 42, tint),
				textNode("result-score", geom.Vector2{X: screenWidth / 2, Y: 278}, fmt.Sprintf("CORES RECOVERED  %d / %d", p.score, coreGoal), 20, coreColor),
				textNode("result-prompt", geom.Vector2{X: screenWidth / 2, Y: 378}, "PRESS ENTER FOR BRIEFING", 20, &color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			},
		}
	})
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

func textNode(key string, position geom.Vector2, text string, size float64, tint *color.RGBA) scene.Node {
	return scene.Node{
		Key: key,
		Components: []any{
			core.Transform{Position: position},
			withTextStyle(textDrawable(text, size), spr.Style{Color: tint, Layer: 5}),
		},
	}
}

func textDrawable(text string, size float64) spr.Text {
	return spr.Text{Asset: builtin.Font, Text: text, TextSize: size}
}

// withTextStyle copies a text component with its style applied, for
// scene node component lists that pair a text with per-surface styling.
func withTextStyle(x spr.Text, style spr.Style) spr.Text {
	x.Style = style
	return x
}
