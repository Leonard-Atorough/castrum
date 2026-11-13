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
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

type demonstration struct {
	entity      *core.Entity
	completions int
	last        asset.ID
}

func (d *demonstration) controls(ctx *core.Context) error {
	if ctx.Input.KeyPressed(input.KeyN) || ctx.Input.KeyPressed(input.KeyP) || ctx.Input.KeyPressed(input.KeyR) {
		if err := d.entity.Update(ctx.World, func(p *audio.Playlist) {
			if ctx.Input.KeyPressed(input.KeyN) {
				p.Next()
			} else if ctx.Input.KeyPressed(input.KeyP) {
				p.Previous()
			}
			if ctx.Input.KeyPressed(input.KeyR) {
				if p.Repeat == audio.RepeatNone {
					p.Repeat = audio.RepeatAll
				} else {
					p.Repeat = audio.RepeatNone
				}
			}
		}); err != nil {
			return err
		}
	}
	if ctx.Input.KeyPressed(input.KeySpace) {
		return d.entity.Update(ctx.World, func(s *audio.Source) { s.Paused = !s.Paused })
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
	source := audio.NewSource()
	source.Load = audio.LoadStream
	source.Group = audio.GroupMusic
	entity, err := g.World().NewEntity(source, audio.NewPlaylist([]asset.ID{audioassets.Magic, audioassets.Music}))
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
			if event.Entity == entity.ID() {
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
	labels := make([]*core.Entity, 4)
	for line := range labels {
		label, err := g.World().NewEntity(
			core.Transform{Position: geom.Vector2{X: 560, Y: float64(80 + line*70)}},
			render.Sprite{Drawable: render.TextSource{Font: builtin.Font, Size: 20}},
		)
		if err != nil {
			return nil, nil, err
		}
		labels[line] = label
	}
	if err := g.AddSystem(core.PhasePostFrame, "show playlist", core.SystemFunc(func(ctx *core.Context) error {
		p, ok := entity.Component[audio.Playlist](ctx.World)
		if !ok {
			return fmt.Errorf("playlist component is missing")
		}
		s, ok := entity.Component[audio.Source](ctx.World)
		if !ok {
			return fmt.Errorf("source component is missing")
		}
		repeat := "off"
		if p.Repeat == audio.RepeatAll {
			repeat = "all"
		}
		lines := []string{
			"N: next   P: previous   R: repeat sequence   SPACE: pause",
			fmt.Sprintf("Selected: %s", p.Current()),
			fmt.Sprintf("Repeat: %s | Paused: %t | Exhausted: %t", repeat, s.Paused, s.Completed),
			fmt.Sprintf("Completed tracks: %d | Last: %s", d.completions, d.last),
		}
		for i, label := range labels {
			if err := label.Update(ctx.World, func(sprite *render.Sprite) {
				sprite.Drawable = render.TextSource{Font: builtin.Font, Text: lines[i], Size: 20}
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
