// Command state demonstrates queued mode changes, lifecycle hooks, and signals.
package main

import (
	"fmt"
	"strings"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/state"
)

type mode string

const (
	idle    mode = "idle"
	playing mode = "playing"
	done    mode = "done"
)

type demonstration struct {
	machine *state.Machine[mode]
	events  []string
}

func (d *demonstration) record(event string) {
	d.events = append(d.events, event)
	if len(d.events) > 6 {
		d.events = d.events[1:]
	}
}

func (d *demonstration) controls(ctx *core.Context) error {
	if ctx.Actions.JustPressed("restart") {
		return d.machine.Restart()
	}
	if ctx.Actions.JustPressed("next") {
		switch d.machine.Current() {
		case idle:
			return d.machine.Request(playing)
		case playing:
			return d.machine.Request(done)
		case done:
			return d.machine.Request(idle)
		}
	}
	return nil
}

func (d *demonstration) readout() string {
	pending := "none"
	if next, ok := d.machine.Pending(); ok {
		pending = string(next)
	}
	return fmt.Sprintf("SPACE: next mode   R: restart current mode\n\nCurrent: %s   Pending: %s\n\n%s",
		d.machine.Current(), pending, strings.Join(d.events, "\n"))
}

func newGame() (*castrum.Game, *demonstration, error) {
	d := &demonstration{machine: state.New[mode]().WithInitial(idle)}
	for _, value := range []mode{idle, playing, done} {
		d.machine.OnEnter(value, func(*core.Context) error {
			d.record("enter " + string(value))
			return nil
		})
		d.machine.OnExit(value, func(*core.Context) error {
			d.record("exit " + string(value))
			return nil
		})
	}
	g, err := castrum.New(
		castrum.WithTitle("castrum - queued state changes"),
		castrum.WithState(d.machine),
		castrum.WithBindings(input.Bindings{
			"next":    {input.KeyInput{Key: input.KeySpace}.Binding()},
			"restart": {input.KeyInput{Key: input.KeyR}.Binding()},
		}),
	)
	if err != nil {
		return nil, nil, err
	}
	if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
		return nil, nil, err
	}
	if err := g.MainCamera().Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: 480, Y: 270}
	}); err != nil {
		return nil, nil, err
	}
	labels := make([]*core.Entity, 10)
	for line := range labels {
		label, err := g.World().NewEntity(
			core.Transform{Position: geom.Vector2{X: 480, Y: float64(36 + line*46)}},
			render.Sprite{Drawable: render.TextSource{Font: builtin.Font, Size: 22}},
		)
		if err != nil {
			return nil, nil, err
		}
		labels[line] = label
	}
	if err := g.AddSystem(core.PhaseFrame, "request mode", core.SystemFunc(d.controls)); err != nil {
		return nil, nil, err
	}
	hub := g.World().MustResource[*signal.Hub]()
	if err := g.AddSystem(core.PhaseFrame, "observe initial mode", hub.Consumer(
		func(_ *core.Context, event state.Initialized[mode]) error {
			d.record("initialized " + string(event.State))
			return nil
		},
	)); err != nil {
		return nil, nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "observe mode changes", hub.Consumer(
		func(_ *core.Context, event state.Changed[mode]) error {
			d.record(fmt.Sprintf("observed %s -> %s", event.From, event.To))
			return nil
		},
	)); err != nil {
		return nil, nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "show mode", core.SystemFunc(func(ctx *core.Context) error {
		lines := strings.Split(d.readout(), "\n")
		for i, label := range labels {
			text := ""
			if i < len(lines) {
				text = lines[i]
			}
			if err := label.Update(ctx.World, func(sprite *render.Sprite) {
				sprite.Drawable = render.TextSource{Font: builtin.Font, Text: text, Size: 22}
			}); err != nil {
				return err
			}
		}
		return nil
	})); err != nil {
		return nil, nil, err
	}
	return g, d, nil
}

func run() error {
	g, _, err := newGame()
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithWindowSize(960, 540), ebitrun.WithLogicalSize(960, 540))
	if err != nil {
		return err
	}
	return g.Run(runner)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
