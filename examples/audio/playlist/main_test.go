package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
)

type controller struct {
	finished bool
	starts   []asset.ID
}

func (c *controller) Sync(_ core.EntityID, source asset.ID, want audio.Playback) (bool, error) {
	if want.Restart {
		c.starts = append(c.starts, source)
		c.finished = false
	}
	return !c.finished, nil
}

func (*controller) Sweep() error { return nil }

func TestFinishedTrackReachesObserver(t *testing.T) {
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
	advance := func() {
		t.Helper()
		if err := g.Advance(time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	advance()
	backend.finished = true
	advance()

	player, ok := demo.entity.Component[audio.Player](g.World())
	if !ok || player.Current() != audioassets.Music || demo.completions != 1 || demo.last != audioassets.Magic {
		t.Fatal("completion did not advance selection or reach the post-frame observer")
	}
	advance()
	if len(backend.starts) != 2 || backend.starts[1] != audioassets.Music {
		t.Fatalf("started tracks = %v, want the second track to start", backend.starts)
	}
}
