package castrum

import (
	"errors"
	"math"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestAdvanceAccumulatesFixedTicks(t *testing.T) {
	// FixedTPS 4 => fixedDT = 250ms.
	g, _ := New(WithFixedTPS(4))
	var ticks, frames int
	var frameDT, fixedDT time.Duration
	g.AddSystem(core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		frameDT = ctx.DeltaTime
		return nil
	}))
	g.AddSystem(core.PhaseFixed, "fixed counter", core.SystemFunc(func(ctx *core.Context) error {
		ticks++
		fixedDT = ctx.DeltaTime
		return nil
	}))

	step := 100 * time.Millisecond
	g.Advance(step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("after first advance: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.4) > 1e-9 {
		t.Errorf("alpha after first advance = %v, want 0.4", alpha)
	}

	g.Advance(step)
	if ticks != 0 || frames != 2 {
		t.Fatalf("after second advance: ticks=%d frames=%d, want 0/2", ticks, frames)
	}

	g.Advance(step)
	if ticks != 1 || frames != 3 {
		t.Fatalf("after third advance: ticks=%d frames=%d, want 1/3", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.2) > 1e-9 {
		t.Errorf("alpha after third advance = %v, want 0.2", alpha)
	}
	if frameDT != step {
		t.Errorf("frame DeltaTime = %v, want %v", frameDT, step)
	}
	if fixedDT != 250*time.Millisecond {
		t.Errorf("fixed DeltaTime = %v, want 250ms", fixedDT)
	}
}

func TestMaxFrameTimeClampsElapsed(t *testing.T) {
	// FixedTPS 1 => fixedDT = 1s; MaxFrameTime 50ms clamps a 10s stall.
	g, _ := New(WithFixedTPS(1), WithMaxFrameTime(50*time.Millisecond))
	var ticks int
	g.AddSystem(core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	g.Advance(10 * time.Second)
	if ticks != 0 {
		t.Errorf("ticks = %d, want 0 (elapsed clamped below one tick)", ticks)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.05) > 1e-9 {
		t.Errorf("alpha = %v, want 0.05", alpha)
	}
}

func TestMaxTicksPerFrameDropsBacklog(t *testing.T) {
	// FixedTPS 100 => fixedDT = 10ms; 1s due, capped at 5 ticks, backlog dropped.
	g, _ := New(WithFixedTPS(100), WithMaxFrameTime(time.Second))
	var ticks int
	g.AddSystem(core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	g.Advance(time.Second)
	if ticks != 5 {
		t.Errorf("ticks = %d, want 5 (capped at MaxTicksPerFrame)", ticks)
	}
	if alpha := g.Alpha(); alpha != 0 {
		t.Errorf("alpha = %v, want 0 (backlog dropped)", alpha)
	}
}

func TestSystemErrorPropagates(t *testing.T) {
	g, _ := New()
	boom := errors.New("boom")
	g.AddSystem(core.PhaseFixed, "boom", core.SystemFunc(func(ctx *core.Context) error { return boom }))

	err := g.Advance(100 * time.Millisecond)
	if !errors.Is(err, boom) {
		t.Fatalf("Advance error = %v, want wrapped %v", err, boom)
	}
	if !strings.Contains(err.Error(), "fixed") {
		t.Errorf("error %q should name the schedule", err)
	}
}

func TestStartupRunsOnce(t *testing.T) {
	g, _ := New()
	var runs int
	g.AddSystem(core.PhaseStartup, "startup counter", core.SystemFunc(func(ctx *core.Context) error { runs++; return nil }))

	if err := g.Startup(); err != nil {
		t.Fatalf("first Startup: %v", err)
	}
	if runs != 1 {
		t.Fatalf("startup systems ran %d times, want 1", runs)
	}
	if err := g.Startup(); err == nil {
		t.Error("second Startup should return an error")
	}
	if runs != 1 {
		t.Errorf("second Startup ran systems again: %d runs, want 1", runs)
	}
}

func TestContextWiredToWorld(t *testing.T) {
	g, _ := New()
	if g.World() == nil {
		t.Fatal("World = nil, want an initialized world")
	}
	if g.Context().World != g.World() {
		t.Error("Context().World differs from Game.World()")
	}
}

type startupProbe struct{}

func TestStartupResolvesEagerBeforeSystems(t *testing.T) {
	g, _ := New()
	resolved := false
	if err := g.World().ProvideEager(func(w *core.World) (*startupProbe, error) {
		resolved = true
		return &startupProbe{}, nil
	}); err != nil {
		t.Fatalf("ProvideEager: %v", err)
	}
	g.AddSystem(core.PhaseStartup, "probe", core.SystemFunc(func(ctx *core.Context) error {
		if !resolved {
			t.Error("eager resource not resolved before startup systems ran")
		}
		if _, err := ctx.World.Resource[*startupProbe](); err != nil {
			t.Errorf("startup system fetching eager resource: %v", err)
		}
		return nil
	}))

	if err := g.Startup(); err != nil {
		t.Fatalf("Startup: %v", err)
	}
}

func TestQuit(t *testing.T) {
	g, _ := New()
	if g.Quitting() {
		t.Fatal("new game should not be quitting")
	}
	g.Quit()
	if !g.Quitting() {
		t.Error("Quitting = false after Quit")
	}
}

type fakeRunner struct {
	err error
}

func (f *fakeRunner) Run() error { return f.err }

func TestRunOnce(t *testing.T) {
	g, _ := New()
	boom := errors.New("runner failed")
	if err := g.Run(&fakeRunner{err: boom}); !errors.Is(err, boom) {
		t.Fatalf("Run error = %v, want %v", err, boom)
	}
	if err := g.Run(&fakeRunner{}); err == nil {
		t.Error("second Run should return an error")
	}
}

// TestCoreHasNoBackendImports enforces runner-agnosticism: package castrum
// must never depend on a backend, directly or transitively.
func TestCoreHasNoBackendImports(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, dep := range strings.Split(string(out), "\n") {
		if strings.Contains(dep, "hajimehoshi") || strings.Contains(dep, "ebitengine") {
			t.Errorf("core depends on backend package %q", dep)
		}
	}
}

func TestEngineRegistersPrevTransformCapture(t *testing.T) {
	g, _ := New()
	entity, err := g.World().NewEntity(core.Transform{Position: geom.Vector2{X: 7, Y: 7}, Scale: geom.Vector2{X: 1, Y: 1}})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	// One full tick advances the fixed phase: the engine's registered
	// prev-capture system materializes the snapshot.
	if err := g.Advance(time.Second / 60); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	prev, ok := entity.Component[core.PrevTransform](g.World())
	if !ok {
		t.Fatal("engine prev-capture did not run during the fixed phase")
	}
	if prev.Position.X != 7 {
		t.Fatalf("prev = %+v, want the spawn position", prev)
	}
}
