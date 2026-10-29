package castrum

import (
	"errors"
	"math"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

func TestNewDefaults(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	opts := g.Options()
	if opts.Title != "castrum" {
		t.Errorf("Title = %q, want %q", opts.Title, "castrum")
	}
	if opts.FixedTPS != 60 {
		t.Errorf("FixedTPS = %d, want 60", opts.FixedTPS)
	}
	if opts.MaxFrameTime != 250*time.Millisecond {
		t.Errorf("MaxFrameTime = %v, want 250ms", opts.MaxFrameTime)
	}
	if opts.MaxTicksPerFrame != 5 {
		t.Errorf("MaxTicksPerFrame = %d, want 5", opts.MaxTicksPerFrame)
	}
	if opts.FixedDT() != time.Second/60 {
		t.Errorf("FixedDT = %v, want %v", opts.FixedDT(), time.Second/60)
	}
}

func TestOptionOverrides(t *testing.T) {
	g, err := New(
		WithTitle("Demo"),
		WithFixedTPS(120),
		WithMaxFrameTime(time.Second),
		WithMaxTicksPerFrame(2),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	opts := g.Options()

	if opts.Title != "Demo" {
		t.Errorf("Title = %q, want %q", opts.Title, "Demo")
	}
	if opts.FixedTPS != 120 {
		t.Errorf("FixedTPS = %d, want 120", opts.FixedTPS)
	}
	if opts.MaxFrameTime != time.Second {
		t.Errorf("MaxFrameTime = %v, want 1s", opts.MaxFrameTime)
	}
	if opts.MaxTicksPerFrame != 2 {
		t.Errorf("MaxTicksPerFrame = %d, want 2", opts.MaxTicksPerFrame)
	}
	if opts.FixedDT() != time.Second/120 {
		t.Errorf("FixedDT = %v, want %v", opts.FixedDT(), time.Second/120)
	}
}

func TestInvalidOptionsError(t *testing.T) {
	cases := []struct {
		name string
		opts []option
	}{
		{"zero tps", []option{WithFixedTPS(0)}},
		{"negative tps", []option{WithFixedTPS(-1)}},
		{"zero max frame time", []option{WithMaxFrameTime(0)}},
		{"zero max ticks", []option{WithMaxTicksPerFrame(0)}},
		{"unrepresentable tick interval", []option{WithFixedTPS(2_000_000_000)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g, err := New(c.opts...)
			if err == nil {
				t.Errorf("%s: expected an error", c.name)
			}
			if g != nil {
				t.Errorf("%s: New must not return a half-built game", c.name)
			}
		})
	}
}

func TestMainCamera(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	camera := g.MainCamera()
	if camera == nil {
		t.Fatal("MainCamera should return the engine camera")
	}
	cam, ok := camera.Component[render.Camera](g.World())
	if !ok || !cam.Primary {
		t.Fatalf("engine camera = %+v, ok %v, want primary", cam, ok)
	}
	transform, ok := camera.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("engine camera has no transform")
	}
	if transform.Position != (geom.Vector2{}) {
		t.Fatalf("engine camera position = %v, want the origin", transform.Position)
	}
	if transform.Scale.X != 1 || transform.Scale.Y != 1 {
		t.Fatalf("engine camera scale = %v, want zoom 1", transform.Scale)
	}
}

func TestNewProvidesResources(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if loader, err := g.World().Resource[*asset.Loader](); err != nil || loader == nil {
		t.Errorf("asset loader = %v, error %v", loader, err)
	}
	if store, err := g.World().Resource[*animation.ClipStore](); err != nil || store == nil {
		t.Errorf("clip store = %v, error %v", store, err)
	}
	if mixer, err := g.World().Resource[*audio.Mixer](); err != nil || mixer == nil {
		t.Errorf("audio mixer = %v, error %v", mixer, err)
	}
	if hub, err := g.World().Resource[*signal.Hub](); err != nil || hub == nil {
		t.Errorf("signal hub = %v, error %v", hub, err)
	}
	panel, err := g.World().Resource[*debug.Panel]()
	if err != nil || panel == nil {
		t.Fatalf("debug panel = %v, error %v", panel, err)
	}
	if !panel.Enabled(debug.FPS) || !panel.Enabled(debug.TPS) {
		t.Error("the engine's panel should start with FPS and TPS enabled")
	}
	if g.World() == nil || g.Context().World != g.World() {
		t.Error("Context.World should be the game world")
	}
}

func TestQuit(t *testing.T) {
	g := newTestGame(t)
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
	g := newTestGame(t)
	boom := errors.New("runner failed")
	if err := g.Run(&fakeRunner{err: boom}); !errors.Is(err, boom) {
		t.Fatalf("Run error = %v, want %v", err, boom)
	}
	if err := g.Run(&fakeRunner{}); err == nil {
		t.Error("second Run should return an error")
	}
}

// enginePackages are the game-facing packages that must stay free of
// backend imports: the engine, its subsystems, and its internals. The
// runner and the benchmark module are the deliberate exceptions - they
// are the backend and its measurement.
var enginePackages = []string{
	".",
	"./core",
	"./signal",
	"./debug",
	"./timer",
	"./animation",
	"./collision",
	"./audio",
	"./input",
	"./geom",
	"./render",
	"./asset",
	"./tween",
	"./internal/ecs",
	"./internal/runtime",
	"./internal/spatial",
}

func TestEnginePackagesHaveNoBackendImports(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	for _, pkg := range enginePackages {
		t.Run(pkg, func(t *testing.T) {
			out, err := exec.Command("go", "list", "-deps", pkg).Output()
			if err != nil {
				t.Fatalf("go list -deps: %v", err)
			}
			for _, dep := range strings.Split(string(out), "\n") {
				if strings.Contains(dep, "hajimehoshi") || strings.Contains(dep, "ebitengine") {
					t.Errorf("%s depends on backend package %q", pkg, dep)
				}
			}
		})
	}
}

// TestExamplesImportNoBackendDirectly flags examples that import the
// backend themselves instead of going through the runner or the debug
// panel. It is a migration list: a flagged example still runs, but its
// diagnostics should move to the debug panel.
func TestExamplesImportNoBackendDirectly(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	out, err := exec.Command("go", "list", "-f",
		`{{.ImportPath}}: {{join .Imports " "}}`, "./examples/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "hajimehoshi") && !strings.Contains(line, "ebitengine") {
			continue
		}
		t.Errorf("%s imports the backend directly; route diagnostics through the debug panel or the runner",
			strings.SplitN(line, ":", 2)[0])
	}
}

func TestPauseStopsFixedOnlyFrameContinues(t *testing.T) {
	// FixedTPS 4 => fixedDT = 250ms.
	g := newTestGame(t, WithFixedTPS(4))
	var ticks, frames int
	var frameDT time.Duration
	addTestSystem(t, g, core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		frameDT = ctx.DeltaTime
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error {
		ticks++
		return nil
	}))

	step := 100 * time.Millisecond
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("before pause: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.4) > 1e-9 {
		t.Fatalf("alpha before pause = %v, want 0.4", alpha)
	}

	g.Pause()
	if !g.Paused() {
		t.Fatal("Paused = false after Pause")
	}
	advanceTestGame(t, g, 10*time.Second)
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 3 {
		t.Fatalf("while paused: ticks=%d frames=%d, want 0/3 - frame phase must continue", ticks, frames)
	}
	if frameDT != step {
		t.Errorf("frame DeltaTime while paused = %v, want %v (unscaled wall-clock time)", frameDT, step)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.4) > 1e-9 {
		t.Errorf("alpha while paused = %v, want 0.4 (frozen)", alpha)
	}

	g.Resume()
	if g.Paused() {
		t.Fatal("Paused = true after Resume")
	}
	// The accumulator was frozen at 100ms; resuming continues from there,
	// so the tick that was half-due before the pause still needs the rest
	// of its interval.
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 4 {
		t.Fatalf("after resume: ticks=%d frames=%d, want 0/4 (paused time is not recovered)", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.8) > 1e-9 {
		t.Errorf("alpha after resume = %v, want 0.8 (frozen 0.4 plus one frame)", alpha)
	}
	advanceTestGame(t, g, step)
	if ticks != 1 || frames != 5 {
		t.Fatalf("tick after resume: ticks=%d frames=%d, want 1/5", ticks, frames)
	}
}

func TestTimeScaleScalesAccumulationOnly(t *testing.T) {
	// FixedTPS 10 => fixedDT = 100ms.
	g := newTestGame(t, WithFixedTPS(10))
	var ticks, frames int
	var frameDT time.Duration
	addTestSystem(t, g, core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		frameDT = ctx.DeltaTime
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	if err := g.SetTimeScale(0.5); err != nil {
		t.Fatalf("SetTimeScale(0.5): %v", err)
	}
	step := 100 * time.Millisecond
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("half speed, first advance: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.5) > 1e-9 {
		t.Errorf("alpha at half speed = %v, want 0.5", alpha)
	}
	if frameDT != step {
		t.Errorf("frame DeltaTime at half speed = %v, want %v (frame time is never scaled)", frameDT, step)
	}

	advanceTestGame(t, g, step)
	if ticks != 1 || frames != 2 {
		t.Fatalf("half speed, second advance: ticks=%d frames=%d, want 1/2", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha) > 1e-9 {
		t.Errorf("alpha after tick = %v, want 0", alpha)
	}
}

func TestTimeScaleFastForwardRespectsSpiralGuard(t *testing.T) {
	// A one-second frame at scale 10 is capped at MaxTicksPerFrame ticks.
	g := newTestGame(t, WithFixedTPS(100), WithMaxFrameTime(time.Second))
	var ticks int
	addTestSystem(t, g, core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	if err := g.SetTimeScale(10); err != nil {
		t.Fatalf("SetTimeScale(10): %v", err)
	}
	advanceTestGame(t, g, time.Second)
	if ticks != 5 {
		t.Errorf("ticks = %d, want 5 (fast forward still capped at MaxTicksPerFrame)", ticks)
	}
	if alpha := g.Alpha(); alpha != 0 {
		t.Errorf("alpha = %v, want 0 (backlog dropped)", alpha)
	}
}

func TestTimeScaleChangePreservesRemainder(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(4))
	var ticks int
	addTestSystem(t, g, core.PhaseFixed, "tick counter", core.SystemFunc(func(*core.Context) error {
		ticks++
		return nil
	}))
	if err := g.SetTimeScale(0.5); err != nil {
		t.Fatal(err)
	}
	advanceTestGame(t, g, 250*time.Millisecond)
	if ticks != 0 || math.Abs(g.Alpha()-0.5) > 1e-9 {
		t.Fatalf("half speed: ticks=%d alpha=%v, want 0 and 0.5", ticks, g.Alpha())
	}
	if err := g.SetTimeScale(2); err != nil {
		t.Fatal(err)
	}
	advanceTestGame(t, g, 125*time.Millisecond)
	if ticks != 1 || math.Abs(g.Alpha()-0.5) > 1e-9 {
		t.Fatalf("double speed: ticks=%d alpha=%v, want 1 and 0.5", ticks, g.Alpha())
	}
	advanceTestGame(t, g, 125*time.Millisecond)
	if ticks != 2 || math.Abs(g.Alpha()-0.5) > 1e-9 {
		t.Fatalf("double speed next frame: ticks=%d alpha=%v, want 2 and 0.5", ticks, g.Alpha())
	}
}

func TestSetTimeScaleRejectsNonPositive(t *testing.T) {
	g := newTestGame(t)
	if g.TimeScale() != 1 {
		t.Fatalf("default TimeScale = %v, want 1", g.TimeScale())
	}
	for _, bad := range []float64{0, -0.5, -2} {
		if err := g.SetTimeScale(bad); err == nil {
			t.Errorf("SetTimeScale(%v) should return an error", bad)
		}
	}
	if g.TimeScale() != 1 {
		t.Errorf("TimeScale after rejected values = %v, want 1 (unchanged)", g.TimeScale())
	}

	if err := g.SetTimeScale(2); err != nil {
		t.Fatal(err)
	}
	if g.TimeScale() != 2 {
		t.Errorf("TimeScale = %v, want 2", g.TimeScale())
	}

	// Scale is remembered while paused and applies after Resume.
	g.Pause()
	if err := g.SetTimeScale(0.25); err != nil {
		t.Fatal(err)
	}
	g.Resume()
	if g.TimeScale() != 0.25 {
		t.Errorf("TimeScale after pause/resume = %v, want 0.25", g.TimeScale())
	}
}
