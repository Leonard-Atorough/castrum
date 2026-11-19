// Command playlist demonstrates selection, sequence repeat, and track completion.
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/signal"
	spr "github.com/Leonard-Atorough/castrum/sprite"
)

type demonstration struct {
	entity      core.Entity
	completions int
	last        asset.ID
}

func (d *demonstration) controls(ctx *core.Context) error {
	if ctx.Input.KeyPressed(input.KeyN) || ctx.Input.KeyPressed(input.KeyP) || ctx.Input.KeyPressed(input.KeyR) {
		if err := d.entity.Update(ctx.World, func(p *audio.Player) {
			if ctx.Input.KeyPressed(input.KeyN) {
				p.Next()
			} else if ctx.Input.KeyPressed(input.KeyP) {
				p.Prev()
			}
			if ctx.Input.KeyPressed(input.KeyR) {
				if p.Settings.Loop == audio.LoopNone {
					p.Settings.Loop = audio.LoopAll
				} else {
					p.Settings.Loop = audio.LoopNone
				}
			}
		}); err != nil {
			return err
		}
	}
	if ctx.Input.KeyPressed(input.KeySpace) {
		return d.entity.Update(ctx.World, func(p *audio.Player) { p.Paused = !p.Paused })
	}
	return nil
}

func newGame() (*castrum.Game, *demonstration, error) {
	g, err := castrum.New(
		castrum.WithTitle("castrum - playlist controls"),
		castrum.WithFilesystem(audioassets.Files),
	)
	if err != nil {
		return nil, nil, err
	}
	if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
		return nil, nil, err
	}
	player, err := audio.NewPlayer(audio.Tracks(audioassets.Magic, audioassets.Music), audio.Settings{
		Volume: 1, Load: audio.LoadStream, Group: audio.GroupMusic,
	})
	if err != nil {
		return nil, nil, err
	}
	entity, err := g.World().NewEntity(player)
	if err != nil {
		return nil, nil, err
	}
	d := &demonstration{entity: entity}
	if err := g.AddSystem(core.PhaseFrame, "playlist controls", core.SystemFunc(d.controls)); err != nil {
		return nil, nil, err
	}
	hub := g.World().MustResource[*signal.Hub]()
	if err := g.AddSystem(core.PhasePostFrame, "observe finished tracks", hub.Consumer(
		func(_ *core.Context, event audio.Finished) error {
			if event.Entity == entity {
				d.completions++
				d.last = event.Audio
			}
			return nil
		},
	)); err != nil {
		return nil, nil, err
	}
	if err := g.MainCamera().Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: 560, Y: 200}
	}); err != nil {
		return nil, nil, err
	}
	labels := make([]core.Entity, 4)
	for line := range labels {
		label, err := g.World().NewEntity(
			core.Transform{Position: geom.Vector2{X: 560, Y: float64(80 + line*70)}},
			spr.Sprite{Source: spr.Source{Kind: spr.KindText, Asset: builtin.Font, TextSize: 20}},
		)
		if err != nil {
			return nil, nil, err
		}
		labels[line] = label
	}
	if err := g.AddSystem(core.PhasePostFrame, "show playlist", core.SystemFunc(func(ctx *core.Context) error {
		p, ok := entity.Component[audio.Player](ctx.World)
		if !ok {
			return fmt.Errorf("player component is missing")
		}
		repeat := "off"
		if p.Settings.Loop == audio.LoopAll {
			repeat = "all"
		}
		lines := []string{
			"N: next   P: previous   R: repeat sequence   SPACE: pause",
			fmt.Sprintf("Selected: %s", p.Current()),
			fmt.Sprintf("Repeat: %s | Paused: %t | Exhausted: %t", repeat, p.Paused, p.Completed()),
			fmt.Sprintf("Completed tracks: %d | Last: %s", d.completions, d.last),
		}
		for i, label := range labels {
			if err := label.Update(ctx.World, func(sprite *spr.Sprite) {
				sprite.Source = spr.Source{Kind: spr.KindText, Asset: builtin.Font, Text: lines[i], TextSize: 20}
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
	runner, err := ebitrun.New(g, ebitrun.WithWindowSize(1120, 400), ebitrun.WithLogicalSize(1120, 400))
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
