package castrum

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/signal"
)

func TestSetGroupEnabledAcrossPhasesAndRegistrations(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var fixed, frame, late, other, ungrouped int
	if err := g.AddSystemToGroup(core.PhaseFixed, "mission", "first fixed", core.SystemFunc(func(*core.Context) error {
		fixed++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFixed, "other", "other fixed", core.SystemFunc(func(*core.Context) error {
		other++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFixed, "ungrouped fixed", core.SystemFunc(func(*core.Context) error {
		ungrouped++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("new group updates = %d, want 1", fixed)
	}
	if err := g.SetGroupEnabled("mission", false); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFrame, "mission", "late frame", core.SystemFunc(func(*core.Context) error {
		frame++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFixed, "mission", "late fixed", core.SystemFunc(func(*core.Context) error {
		late++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance inactive: %v", err)
	}
	if fixed != 1 || frame != 0 || late != 0 {
		t.Fatalf("inactive updates: fixed=%d frame=%d late=%d", fixed, frame, late)
	}
	if other != 2 || ungrouped != 2 {
		t.Fatalf("unrelated updates: other=%d ungrouped=%d, want 2 each", other, ungrouped)
	}
	if err := g.SetGroupEnabled("mission", true); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if fixed != 2 || frame != 1 || late != 1 {
		t.Fatalf("active updates: fixed=%d frame=%d late=%d, want 2/1/1", fixed, frame, late)
	}
}

func TestSetGroupEnabledRejectsUnknownGroups(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "missing"} {
		if err := g.SetGroupEnabled(name, false); err == nil {
			t.Errorf("SetGroupEnabled(%q) accepted an invalid group", name)
		}
	}
	var updates int
	if err := g.AddSystemToGroup(core.PhaseFixed, "missing", "later", core.SystemFunc(func(*core.Context) error {
		updates++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if updates != 1 {
		t.Fatalf("rejected group change affected later registration: updates=%d", updates)
	}
}

func TestSetGroupEnabledAffectsLaterSystemsInCurrentPhase(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	var updates int
	if err := g.AddSystem(core.PhaseFrame, "disable", core.SystemFunc(func(*core.Context) error {
		return g.SetGroupEnabled("mission", false)
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFrame, "mission", "later", core.SystemFunc(func(*core.Context) error {
		updates++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if updates != 0 {
		t.Fatalf("disabled system ran in current phase: updates=%d", updates)
	}
}

func newTestGame(t *testing.T, opts ...option) *Game {
	t.Helper()
	g, err := New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g
}

func addTestSystem(t *testing.T, g *Game, phase core.Phase, name string, system core.System) {
	t.Helper()
	if err := g.AddSystem(phase, name, system); err != nil {
		t.Fatalf("AddSystem(%s, %q): %v", phase, name, err)
	}
}

func advanceTestGame(t *testing.T, g *Game, elapsed time.Duration) {
	t.Helper()
	if err := g.Advance(elapsed); err != nil {
		t.Fatalf("Advance(%v): %v", elapsed, err)
	}
}

func TestAdvanceAccumulatesFixedTicks(t *testing.T) {
	// FixedTPS 4 => fixedDT = 250ms.
	g := newTestGame(t, WithFixedTPS(4))
	var ticks, frames int
	var frameDT, fixedDT time.Duration
	addTestSystem(t, g, core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		frameDT = ctx.DeltaTime
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "fixed counter", core.SystemFunc(func(ctx *core.Context) error {
		ticks++
		fixedDT = ctx.DeltaTime
		return nil
	}))

	step := 100 * time.Millisecond
	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("after first advance: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.4) > 1e-9 {
		t.Errorf("alpha after first advance = %v, want 0.4", alpha)
	}

	advanceTestGame(t, g, step)
	if ticks != 0 || frames != 2 {
		t.Fatalf("after second advance: ticks=%d frames=%d, want 0/2", ticks, frames)
	}

	advanceTestGame(t, g, step)
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
	g := newTestGame(t, WithFixedTPS(1), WithMaxFrameTime(50*time.Millisecond))
	var ticks int
	var frameDT time.Duration
	addTestSystem(t, g, core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))
	addTestSystem(t, g, core.PhaseFrame, "frame clock", core.SystemFunc(func(ctx *core.Context) error {
		frameDT = ctx.DeltaTime
		return nil
	}))

	advanceTestGame(t, g, 10*time.Second)
	if ticks != 0 {
		t.Errorf("ticks = %d, want 0 (elapsed clamped below one tick)", ticks)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.05) > 1e-9 {
		t.Errorf("alpha = %v, want 0.05", alpha)
	}
	if frameDT != 50*time.Millisecond {
		t.Errorf("frame DeltaTime = %v, want 50ms", frameDT)
	}
}

func TestMaxTicksPerFrameDropsBacklog(t *testing.T) {
	// FixedTPS 100 => fixedDT = 10ms; 1s due, capped at 5 ticks, backlog dropped.
	g := newTestGame(t, WithFixedTPS(100), WithMaxFrameTime(time.Second))
	var ticks int
	addTestSystem(t, g, core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	advanceTestGame(t, g, time.Second)
	if ticks != 5 {
		t.Errorf("ticks = %d, want 5 (capped at MaxTicksPerFrame)", ticks)
	}
	if alpha := g.Alpha(); alpha != 0 {
		t.Errorf("alpha = %v, want 0 (backlog dropped)", alpha)
	}
	advanceTestGame(t, g, 0)
	if ticks != 5 {
		t.Errorf("ticks after zero elapsed = %d, want 5", ticks)
	}
}

func TestSystemErrorPropagates(t *testing.T) {
	g := newTestGame(t)
	boom := errors.New("boom")
	addTestSystem(t, g, core.PhaseFixed, "boom", core.SystemFunc(func(ctx *core.Context) error { return boom }))

	err := g.Advance(100 * time.Millisecond)
	if !errors.Is(err, boom) {
		t.Fatalf("Advance error = %v, want wrapped %v", err, boom)
	}
	if !strings.Contains(err.Error(), "fixed") {
		t.Errorf("error %q should name the schedule", err)
	}
}

func TestStartupRunsOnce(t *testing.T) {
	g := newTestGame(t)
	var runs int
	addTestSystem(t, g, core.PhaseStartup, "startup counter", core.SystemFunc(func(ctx *core.Context) error { runs++; return nil }))

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

type startupProbe struct{}

func TestStartupResolvesEagerBeforeSystems(t *testing.T) {
	g := newTestGame(t)
	resolved := false
	if err := g.World().ProvideEager(func(w *core.World) (*startupProbe, error) {
		resolved = true
		return &startupProbe{}, nil
	}); err != nil {
		t.Fatalf("ProvideEager: %v", err)
	}
	addTestSystem(t, g, core.PhaseStartup, "probe", core.SystemFunc(func(ctx *core.Context) error {
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

func TestEngineRegistersPrevTransformCapture(t *testing.T) {
	g := newTestGame(t)
	entity, err := g.World().NewEntity(core.Transform{Position: geom.Vector2{X: 7, Y: 7}, Scale: geom.Vector2{X: 1, Y: 1}})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	prev, ok := entity.Component[core.PrevTransform](g.World())
	if !ok || prev.Position != (geom.Vector2{X: 7, Y: 7}) {
		t.Fatalf("spawn prev = %v, ok %v, want the spawn position (7, 7)", prev, ok)
	}

	addTestSystem(t, g, core.PhaseFixed, "move", core.SystemFunc(func(ctx *core.Context) error {
		return entity.Update(ctx.World, func(transform *core.Transform) {
			transform.Position = geom.Vector2{X: 9, Y: 9}
		})
	}))
	if err := g.Advance(time.Second / 60); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	prev, ok = entity.Component[core.PrevTransform](g.World())
	if !ok || prev.Position != (geom.Vector2{X: 7, Y: 7}) {
		t.Fatalf("after tick prev = %v, ok %v, want tick-start position (7, 7)", prev, ok)
	}
	transform, ok := entity.Component[core.Transform](g.World())
	if !ok || transform.Position != (geom.Vector2{X: 9, Y: 9}) {
		t.Fatalf("after tick transform = %v, ok %v, want moved position (9, 9)", transform, ok)
	}
	advanceTestGame(t, g, time.Second/60)
	prev, ok = entity.Component[core.PrevTransform](g.World())
	if !ok || prev.Position != (geom.Vector2{X: 9, Y: 9}) {
		t.Fatalf("after second tick prev = %v, ok %v, want prior movement (9, 9)", prev, ok)
	}
}

func TestSignalLifecycleInGameLoop(t *testing.T) {
	// FixedTPS 4 => fixedDT = 250ms; tick 1 lands inside frame 3.
	g := newTestGame(t, WithFixedTPS(4))
	hub := g.World().MustResource[*signal.Hub]()

	type hit struct{}
	tickReader := hub.NewReader[hit]()
	frameReader := hub.NewReader[hit]()
	var tickAt, frameAt []uint64

	addTestSystem(t, g, core.PhaseFixed, "emitter", core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Tick == 1 {
			hub.Emit(hit{})
		}
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "tick consumer", core.SystemFunc(func(ctx *core.Context) error {
		for _, ok := tickReader.Next(); ok; _, ok = tickReader.Next() {
			tickAt = append(tickAt, ctx.Tick)
		}
		return nil
	}))
	addTestSystem(t, g, core.PhaseFrame, "frame consumer", core.SystemFunc(func(ctx *core.Context) error {
		for _, ok := frameReader.Next(); ok; _, ok = frameReader.Next() {
			frameAt = append(frameAt, ctx.Frame)
		}
		return nil
	}))

	step := 100 * time.Millisecond
	for range 6 {
		if err := g.Advance(step); err != nil {
			t.Fatalf("Advance: %v", err)
		}
	}

	if !reflect.DeepEqual(tickAt, []uint64{1}) {
		t.Fatalf("tick deliveries = %v, want exactly [1]", tickAt)
	}
	if !reflect.DeepEqual(frameAt, []uint64{4}) {
		t.Fatalf("frame deliveries = %v, want exactly [4] (the frame after the tick)", frameAt)
	}
}
