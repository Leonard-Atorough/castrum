package castrum

import (
	"errors"
	"io"
	"math"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/gametime"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

func TestNewOptions(t *testing.T) {
	type wantOptions struct {
		title        string
		fixedTPS     int
		maxFrameTime time.Duration
		maxTicks     int
	}
	cases := []struct {
		name string
		opts []option
		want wantOptions
	}{
		{name: "defaults", want: wantOptions{title: "castrum", fixedTPS: 60,
			maxFrameTime: 250 * time.Millisecond, maxTicks: 5}},
		{name: "overrides",
			opts: []option{WithTitle("Demo"), WithFixedTPS(120),
				WithMaxFrameTime(time.Second), WithMaxTicksPerFrame(2)},
			want: wantOptions{title: "Demo", fixedTPS: 120,
				maxFrameTime: time.Second, maxTicks: 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := newTestGame(t, c.opts...).Options()
			if opts.Title != c.want.title {
				t.Errorf("Title = %q, want %q", opts.Title, c.want.title)
			}
			if opts.FixedTPS != c.want.fixedTPS {
				t.Errorf("FixedTPS = %d, want %d", opts.FixedTPS, c.want.fixedTPS)
			}
			if opts.MaxFrameTime != c.want.maxFrameTime {
				t.Errorf("MaxFrameTime = %v, want %v", opts.MaxFrameTime, c.want.maxFrameTime)
			}
			if opts.MaxTicksPerFrame != c.want.maxTicks {
				t.Errorf("MaxTicksPerFrame = %d, want %d", opts.MaxTicksPerFrame, c.want.maxTicks)
			}
			if dt := time.Second / time.Duration(c.want.fixedTPS); opts.FixedDT() != dt {
				t.Errorf("FixedDT = %v, want %v", opts.FixedDT(), dt)
			}
		})
	}
}

func TestNewMakesBuiltinAssetsAvailable(t *testing.T) {
	g := newTestGame(t)
	font, err := g.World().MustResource[*asset.Loader]().Load[asset.FontData](builtin.Font)
	if err != nil {
		t.Fatalf("load built-in font: %v", err)
	}
	if len(font.Raw) == 0 {
		t.Fatal("built-in font has no raw data")
	}
}

func TestNewComposesBuiltinsWithProvidedFilesystem(t *testing.T) {
	filesystem := fstest.MapFS{
		"sprites/player.txt": &fstest.MapFile{Data: []byte("player")},
	}
	g := newTestGame(t, WithFilesystem(filesystem))

	loader := g.World().MustResource[*asset.Loader]()
	requireBuiltinFont(t, loader)

	file, _, err := loader.Open("sprites/player.txt")
	if err != nil {
		t.Fatalf("open game asset: %v", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read game asset: %v", err)
	}
	if string(data) != "player" {
		t.Fatalf("game asset = %q, want %q", data, "player")
	}
}

func TestNewGamesSeeLaterAssetCodecs(t *testing.T) {
	type levelFile struct{ Name string }
	codecs := asset.NewCodecs()
	if err := codecs.SetJSON[levelFile](); err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{"level.json": {Data: []byte(`{"name":"forest"}`)}}
	first := newTestGame(t, WithFilesystem(files), WithAssetCodecs(codecs))
	if err := codecs.SetDecoder(asset.FormatJSON, func(io.Reader) (levelFile, error) {
		return levelFile{Name: "new configuration"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	second := newTestGame(t, WithFilesystem(files), WithAssetCodecs(codecs))
	for _, game := range []*Game{first, second} {
		loader := game.World().MustResource[*asset.Loader]()
		value, err := loader.Load[levelFile]("level.json")
		if err != nil || value.Name != "new configuration" {
			t.Fatalf("game codec = %+v, %v; want %q", value, err, "new configuration")
		}
		requireBuiltinFont(t, loader)
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
			g, err := newGame(c.opts...)
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
	g := newTestGame(t)
	camera := g.MainCamera()
	if !g.World().HasEntity(camera) {
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
	g := newTestGame(t)
	cases := []struct {
		name    string
		resolve func(*Game) error
	}{
		{"asset loader", resolveResource[*asset.Loader]},
		{"audio mixer", resolveResource[*audio.Mixer]},
		{"signal hub", resolveResource[*signal.Hub]},
		{"scheduler", resolveResource[*gametime.Scheduler]},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.resolve(g); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		})
	}
	listener, err := g.World().Resource[*audio.Listener]()
	if err != nil || listener == nil || listener.Enabled {
		t.Errorf("audio listener = %v, error %v, want disabled listener", listener, err)
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

func resolveResource[T any](g *Game) error {
	value, err := g.World().Resource[T]()
	if err != nil {
		return err
	}
	if reflect.ValueOf(&value).Elem().IsZero() {
		return errors.New("nil resource")
	}
	return nil
}

// requireBuiltinFont fails the test if the loader cannot load the built-in
// font.
func requireBuiltinFont(t *testing.T, loader *asset.Loader) {
	t.Helper()
	if _, err := loader.Load[asset.FontData](builtin.Font); err != nil {
		t.Fatalf("load built-in font: %v", err)
	}
}

// tickCounter returns a system that increments count on each run.
func tickCounter(count *int) core.System {
	return core.SystemFunc(func(*core.Context) error { *count++; return nil })
}

// frameCounter returns a system that increments count and records the delta
// time of each run.
func frameCounter(count *int, delta *time.Duration) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		*count++
		*delta = ctx.DeltaTime
		return nil
	})
}

func checkAlpha(t *testing.T, g *Game, want float64) {
	t.Helper()
	if math.Abs(g.Alpha()-want) > 1e-9 {
		t.Errorf("Alpha = %v, want %v", g.Alpha(), want)
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

func TestBeginRunOnce(t *testing.T) {
	g := newTestGame(t)
	if err := g.BeginRun(); err != nil {
		t.Fatalf("first BeginRun error = %v", err)
	}
	if err := g.BeginRun(); err == nil {
		t.Error("second BeginRun should return an error")
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
	"./gametime",
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
	addTestSystem(t, g, core.PhaseFrame, "frame counter", frameCounter(&frames, &frameDT))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))

	step := 100 * time.Millisecond
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("before pause: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	checkAlpha(t, g, 0.4)

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
	checkAlpha(t, g, 0.4)

	g.Resume()
	if g.Paused() {
		t.Fatal("Paused = true after Resume")
	}
	// The accumulator was frozen at 100ms; resuming continues from there,
	// so the half-due tick still needs the rest of its interval.
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 4 {
		t.Fatalf("after resume: ticks=%d frames=%d, want 0/4 (paused time is not recovered)", ticks, frames)
	}
	checkAlpha(t, g, 0.8)
	advanceTestGame(t, g, step)
	if ticks != 1 || frames != 5 {
		t.Fatalf("tick after resume: ticks=%d frames=%d, want 1/5", ticks, frames)
	}
}

func TestFrameSystemPauseAppliesWithinSameAdvance(t *testing.T) {
	// The frame phase runs before Advance reads the paused flag.
	g := newTestGame(t, WithFixedTPS(4))
	var ticks, frames int
	addTestSystem(t, g, core.PhaseFrame, "pauser", core.SystemFunc(func(*core.Context) error {
		frames++
		g.Pause()
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))

	advanceTestGame(t, g, 250*time.Millisecond)
	if frames != 1 || ticks != 0 {
		t.Fatalf("frame-system Pause: frames=%d ticks=%d, want 1/0 (same-call boundary)", frames, ticks)
	}
	advanceTestGame(t, g, 250*time.Millisecond)
	if frames != 2 || ticks != 0 {
		t.Fatalf("while paused: frames=%d ticks=%d, want 2/0", frames, ticks)
	}
}

func TestFrameSystemResumeAppliesWithinSameAdvance(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(4))
	g.Pause()
	var ticks, frames int
	addTestSystem(t, g, core.PhaseFrame, "resumer", core.SystemFunc(func(*core.Context) error {
		frames++
		g.Resume()
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))

	advanceTestGame(t, g, 250*time.Millisecond)
	if frames != 1 || ticks != 1 {
		t.Fatalf("frame-system Resume: frames=%d ticks=%d, want 1/1 (resume applies to the same call)", frames, ticks)
	}
}

func TestFrameSystemTimeScaleAppliesWithinSameAdvance(t *testing.T) {
	// FixedTPS 10 => fixedDT = 100ms; scale 2 doubles the contribution.
	g := newTestGame(t, WithFixedTPS(10))
	var ticks, frames int
	addTestSystem(t, g, core.PhaseFrame, "scaler", core.SystemFunc(func(*core.Context) error {
		frames++
		g.SetTimeScale(2)
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))

	advanceTestGame(t, g, 100*time.Millisecond)
	if frames != 1 || ticks != 2 {
		t.Fatalf("frame-system SetTimeScale(2): frames=%d ticks=%d, want 1/2 (scale applies to the same call's contribution)", frames, ticks)
	}
}

func TestFixedPhaseControlChangeDoesNotInterruptCurrentAdvance(t *testing.T) {
	// The paused check happens once before the tick loop, so a Pause in the
	// first fixed tick cannot stop the queued second.
	g := newTestGame(t, WithFixedTPS(4), WithMaxFrameTime(time.Second))
	var ticks int
	pausedInTick := false
	addTestSystem(t, g, core.PhaseFixed, "pauser", core.SystemFunc(func(*core.Context) error {
		ticks++
		if !pausedInTick {
			pausedInTick = true
			g.Pause()
		}
		return nil
	}))

	advanceTestGame(t, g, 500*time.Millisecond)
	if ticks != 2 {
		t.Fatalf("pause set during first fixed tick: ticks=%d, want 2 (catch-up not interrupted)", ticks)
	}
	advanceTestGame(t, g, 500*time.Millisecond)
	if ticks != 2 {
		t.Fatalf("later advance while paused: ticks=%d, want still 2", ticks)
	}
}

func TestTimeScaleScalesAccumulationOnly(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(10))
	var ticks, frames int
	var frameDT time.Duration
	addTestSystem(t, g, core.PhaseFrame, "frame counter", frameCounter(&frames, &frameDT))
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))

	g.SetTimeScale(0.5)
	step := 100 * time.Millisecond
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("half speed, first advance: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	checkAlpha(t, g, 0.5)
	if frameDT != step {
		t.Errorf("frame DeltaTime at half speed = %v, want %v (frame time is never scaled)", frameDT, step)
	}

	advanceTestGame(t, g, step)
	if ticks != 1 || frames != 2 {
		t.Fatalf("half speed, second advance: ticks=%d frames=%d, want 1/2", ticks, frames)
	}
	checkAlpha(t, g, 0)
}

func TestTimeScaleFastForwardRespectsSpiralGuard(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(100), WithMaxFrameTime(time.Second))
	var ticks int
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))

	g.SetTimeScale(10)
	advanceTestGame(t, g, time.Second)
	if ticks != 5 {
		t.Errorf("ticks = %d, want 5 (fast forward still capped at MaxTicksPerFrame)", ticks)
	}
	checkAlpha(t, g, 0)
}

func TestTimeScaleChangePreservesRemainder(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(4))
	var ticks int
	addTestSystem(t, g, core.PhaseFixed, "tick counter", tickCounter(&ticks))
	g.SetTimeScale(0.5)
	advanceTestGame(t, g, 250*time.Millisecond)
	if ticks != 0 {
		t.Fatalf("half speed: ticks = %d, want 0", ticks)
	}
	checkAlpha(t, g, 0.5)
	g.SetTimeScale(2)
	advanceTestGame(t, g, 125*time.Millisecond)
	if ticks != 1 {
		t.Fatalf("double speed: ticks = %d, want 1", ticks)
	}
	checkAlpha(t, g, 0.5)
	advanceTestGame(t, g, 125*time.Millisecond)
	if ticks != 2 {
		t.Fatalf("double speed next frame: ticks = %d, want 2", ticks)
	}
	checkAlpha(t, g, 0.5)
}

func TestSetTimeScaleRejectsNonPositive(t *testing.T) {
	g := newTestGame(t)
	if g.TimeScale() != 1 {
		t.Fatalf("default TimeScale = %v, want 1", g.TimeScale())
	}
	for _, bad := range []float64{0, -0.5, -2} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("SetTimeScale(%v) should panic", bad)
				}
			}()
			g.SetTimeScale(bad)
		}()
	}
	if g.TimeScale() != 1 {
		t.Errorf("TimeScale after rejected values = %v, want 1 (unchanged)", g.TimeScale())
	}

	g.SetTimeScale(2)
	if g.TimeScale() != 2 {
		t.Errorf("TimeScale = %v, want 2", g.TimeScale())
	}

	// Scale is remembered while paused and applies after Resume.
	g.Pause()
	g.SetTimeScale(0.25)
	g.Resume()
	if g.TimeScale() != 0.25 {
		t.Errorf("TimeScale after pause/resume = %v, want 0.25", g.TimeScale())
	}
}
