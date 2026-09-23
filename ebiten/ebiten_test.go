package ebiten

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
)

func TestRunnerDefaults(t *testing.T) {
	r, _ := New(mustGame())
	if r.opts.Window != (Size{1280, 720}) {
		t.Errorf("Window = %v, want 1280x720", r.opts.Window)
	}
	if r.opts.Logical != (Size{1280, 720}) {
		t.Errorf("Logical = %v, want 1280x720", r.opts.Logical)
	}
	if !r.opts.VSync {
		t.Error("VSync = false, want true")
	}
	if r.opts.Resizable {
		t.Error("Resizable = true, want false")
	}
}

func TestRunnerOptions(t *testing.T) {
	r, _ := New(mustGame(),
		WithWindowSize(1920, 1080),
		WithLogicalSize(640, 360),
		WithResizable(),
		WithoutVSync(),
	)
	if r.opts.Window != (Size{1920, 1080}) {
		t.Errorf("Window = %v, want 1920x1080", r.opts.Window)
	}
	if r.opts.Logical != (Size{640, 360}) {
		t.Errorf("Logical = %v, want 640x360", r.opts.Logical)
	}
	if !r.opts.Resizable {
		t.Error("Resizable = false, want true")
	}
	if r.opts.VSync {
		t.Error("VSync = true, want false")
	}
}

func mustGame() *castrum.Game {
	g, err := castrum.New()
	if err != nil {
		panic(err)
	}
	return g
}

func TestInvalidRunnerOptionsError(t *testing.T) {
	cases := []struct {
		name string
		opts []option
	}{
		{"zero window width", []option{WithWindowSize(0, 600)}},
		{"zero window height", []option{WithWindowSize(800, 0)}},
		{"zero logical size", []option{WithLogicalSize(0, 0)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := New(mustGame(), c.opts...)
			if err == nil {
				t.Errorf("%s: expected an error", c.name)
			}
			if r != nil {
				t.Errorf("%s: New must not return a half-built runner", c.name)
			}
		})
	}
}

func TestUpdateAdvancesGame(t *testing.T) {
	g := mustGame()
	var frames int
	g.AddSystem(core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		return nil
	}))
	r, _ := New(g)

	if err := r.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if frames != 1 {
		t.Errorf("frames = %d, want 1", frames)
	}

	g.Quit()
	if err := r.Update(); !errors.Is(err, ebiten.Termination) {
		t.Errorf("Update after Quit = %v, want ebiten.Termination", err)
	}
}

func TestDrawSetsAlphaAndDefersErrors(t *testing.T) {
	g, _ := castrum.New(castrum.WithFixedTPS(4)) // fixedDT = 250ms
	g.Advance(100 * time.Millisecond)            // alpha = 0.4

	var gotAlpha = -1.0
	boom := errors.New("draw boom")
	r, _ := New(g)
	r.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		gotAlpha = ctx.Alpha
		return nil
	})
	r.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		return boom
	})

	r.Draw(nil) // screen unused by the test draw funcs
	if math.Abs(gotAlpha-0.4) > 1e-9 {
		t.Errorf("draw alpha = %v, want 0.4", gotAlpha)
	}

	// Draw cannot return errors; the stored one surfaces from Update.
	err := r.Update()
	if !errors.Is(err, boom) {
		t.Errorf("Update after draw error = %v, want wrapped %v", err, boom)
	}
}
