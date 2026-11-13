package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
)

type controller struct {
	finished bool
	starts   []asset.ID
	want     audio.Playback
}

func (c *controller) Sync(_ core.EntityID, source asset.ID, want audio.Playback) (bool, error) {
	c.want = want
	if want.Restart {
		c.starts = append(c.starts, source)
		c.finished = false
	}
	return !c.finished, nil
}

func (*controller) Sweep() error { return nil }

func TestCompletionAdvancesAndExhaustsOrRepeats(t *testing.T) {
	for _, repeat := range []audio.RepeatMode{audio.RepeatNone, audio.RepeatAll} {
		t.Run(fmt.Sprint(repeat), func(t *testing.T) {
			g, demo, err := newGame()
			if err != nil {
				t.Fatal(err)
			}
			if err := demo.entity.Update(g.World(), func(p *audio.Playlist) { p.Repeat = repeat }); err != nil {
				t.Fatal(err)
			}
			backend := &controller{}
			if err := g.World().MustResource[*audio.Backend]().Attach(backend); err != nil {
				t.Fatal(err)
			}
			if err := g.Startup(); err != nil {
				t.Fatal(err)
			}
			advance := func() {
				t.Helper()
				if err := g.Advance(time.Millisecond); err != nil {
					t.Fatal(err)
				}
			}
			advance()
			backend.finished = true
			advance()
			p, _ := demo.entity.Component[audio.Playlist](g.World())
			if p.Current() != audioassets.Music || demo.completions != 1 || demo.last != audioassets.Magic {
				t.Fatal("completion did not advance selection or reach the post-frame observer")
			}
			advance()
			if len(backend.starts) != 2 || backend.starts[1] != audioassets.Music {
				t.Fatal("next track did not start on the next reconciliation")
			}
			backend.finished = true
			advance()
			advance()
			source, _ := demo.entity.Component[audio.Source](g.World())
			if demo.completions != 2 || demo.last != audioassets.Music {
				t.Fatal("final completion was missed or delivered more than once")
			}
			if repeat == audio.RepeatNone {
				if !source.Completed || len(backend.starts) != 2 {
					t.Fatal("non-repeating playlist restarted after exhaustion")
				}
			} else if source.Completed || len(backend.starts) != 3 || backend.starts[2] != audioassets.Magic {
				t.Fatal("sequence repeat did not wrap to the first track")
			}
		})
	}
}

func TestManualSelectionPauseAndRepeatControls(t *testing.T) {
	g, demo, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	backend := &controller{}
	if err := g.World().MustResource[*audio.Backend]().Attach(backend); err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		key    input.Key
		track  asset.ID
		paused bool
		repeat audio.RepeatMode
	}{
		{input.KeyNone, audioassets.Magic, false, audio.RepeatNone},
		{input.KeyP, audioassets.Music, false, audio.RepeatNone},
		{input.KeyN, audioassets.Magic, false, audio.RepeatNone},
		{input.KeyR, audioassets.Magic, false, audio.RepeatAll},
		{input.KeySpace, audioassets.Magic, true, audio.RepeatAll},
		{input.KeyN, audioassets.Music, true, audio.RepeatAll},
		{input.KeySpace, audioassets.Music, false, audio.RepeatAll},
		{input.KeyR, audioassets.Music, false, audio.RepeatNone},
	} {
		g.Context().Input = &input.Snapshot{}
		g.Context().Input.Keys[step.key].Pressed = true
		if err := g.Advance(time.Millisecond); err != nil {
			t.Fatal(err)
		}
		p, _ := demo.entity.Component[audio.Playlist](g.World())
		source, _ := demo.entity.Component[audio.Source](g.World())
		if p.Current() != step.track || p.Repeat != step.repeat || source.Paused != step.paused ||
			backend.want.Playing == step.paused || backend.want.Loop != audio.LoopNone {
			t.Fatalf("control %v: playlist=%+v source=%+v playback=%+v", step.key, p, source, backend.want)
		}
		if demo.completions != 0 {
			t.Fatal("manual selection emitted a natural completion")
		}
	}
}
