package animation

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
)

func newTestClip(t *testing.T, frames ...string) SpriteClip {
	t.Helper()
	clip, err := NewSpriteClip(animSheet, frames, 10)
	if err != nil {
		t.Fatal(err)
	}
	return clip
}

func newAnimatedSystemWorld(t *testing.T, animated AnimatedSprite, dt time.Duration) (core.Entity, core.System, *core.Context, *signal.Reader[Completed]) {
	t.Helper()
	w := core.NewWorld()
	if err := w.Provide(func(*core.World) (*signal.Hub, error) {
		return signal.NewHub(), nil
	}); err != nil {
		t.Fatal(err)
	}
	entity, err := w.NewEntity(animated)
	if err != nil {
		t.Fatal(err)
	}
	return entity, NewSpriteAnimationSystem(), &core.Context{World: w, DeltaTime: dt},
		w.MustResource[*signal.Hub]().NewReader[Completed]()
}

func readAnimated(t *testing.T, e core.Entity, w *core.World) AnimatedSprite {
	t.Helper()
	anim, ok := e.Component[AnimatedSprite](w)
	if !ok {
		t.Fatal("animated sprite component missing")
	}
	return anim
}

func animatedRegion(t *testing.T, e core.Entity, w *core.World) string {
	t.Helper()
	_, region, ok, err := readAnimated(t, e, w).CurrentFrame()
	if err != nil || !ok {
		t.Fatalf("CurrentFrame() = %v, %v, %v", region, ok, err)
	}
	return region
}

func drainCompleted(r *signal.Reader[Completed]) []Completed {
	var got []Completed
	for event, ok := r.Next(); ok; event, ok = r.Next() {
		got = append(got, event)
	}
	return got
}

func tickAt(t *testing.T, sys core.System, ctx *core.Context, tick uint64) {
	t.Helper()
	ctx.Tick = tick
	if err := sys.Update(ctx); err != nil {
		t.Fatalf("update at tick %d: %v", tick, err)
	}
}

func TestSpriteAnimationSystemAdvancement(t *testing.T) {
	tests := []struct {
		name  string
		speed float64
		dt    time.Duration
		want  []string
	}{
		{"default speed", 0, 100 * time.Millisecond, []string{"b"}},
		{"double speed", 2, 100 * time.Millisecond, []string{"c"}},
		{"sub-frame accumulation", 0, 50 * time.Millisecond, []string{"a", "b", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animated, err := NewAnimatedSprite(newTestClip(t, "a", "b", "c", "d"))
			if err != nil {
				t.Fatal(err)
			}
			animated.PlaybackMultiplier = tt.speed
			entity, sys, ctx, _ := newAnimatedSystemWorld(t, animated, tt.dt)
			for i, want := range tt.want {
				tickAt(t, sys, ctx, uint64(i+1))
				if got := animatedRegion(t, entity, ctx.World); got != want {
					t.Fatalf("tick %d: region = %q, want %q", i+1, got, want)
				}
			}
		})
	}
}

func TestSpriteAnimationSystemOneShot(t *testing.T) {
	animated, err := NewAnimatedSprite(newTestClip(t, "a", "b", "c"))
	if err != nil {
		t.Fatal(err)
	}
	entity, sys, ctx, reader := newAnimatedSystemWorld(t, animated, 100*time.Millisecond)
	for tick, want := range []string{"b", "c"} {
		tickAt(t, sys, ctx, uint64(tick+1))
		anim := readAnimated(t, entity, ctx.World)
		if got := animatedRegion(t, entity, ctx.World); got != want || anim.Paused || anim.HasCompleted() {
			t.Fatalf("tick %d: region %q, state %+v; want %q, playing", tick+1, got, anim, want)
		}
	}
	if got := drainCompleted(reader); len(got) != 0 {
		t.Fatalf("signals before completion = %v", got)
	}

	tickAt(t, sys, ctx, 3)
	anim := readAnimated(t, entity, ctx.World)
	if got := animatedRegion(t, entity, ctx.World); got != "c" || !anim.Paused || !anim.HasCompleted() || !anim.JustCompleted(3) {
		t.Fatalf("completion: region %q, state %+v", got, anim)
	}
	if got := drainCompleted(reader); len(got) != 1 || got[0] != (Completed{Entity: entity, Tick: 3, Loops: 1}) {
		t.Fatalf("completion signals = %v", got)
	}
	tickAt(t, sys, ctx, 4)
	anim = readAnimated(t, entity, ctx.World)
	if got := animatedRegion(t, entity, ctx.World); got != "c" || !anim.Paused || !anim.HasCompleted() || anim.JustCompleted(4) {
		t.Fatalf("held completion: region %q, state %+v", got, anim)
	}
	if got := drainCompleted(reader); len(got) != 0 {
		t.Fatalf("signals while held = %v", got)
	}

	if err := entity.Update(ctx.World, func(a *AnimatedSprite) { a.Paused = false }); err != nil {
		t.Fatal(err)
	}
	tickAt(t, sys, ctx, 5)
	if anim := readAnimated(t, entity, ctx.World); !anim.Paused || !anim.JustCompleted(5) {
		t.Fatalf("after unpausing: %+v, want completion again", anim)
	}
	if got := drainCompleted(reader); len(got) != 1 || got[0] != (Completed{Entity: entity, Tick: 5, Loops: 1}) {
		t.Fatalf("signals after unpausing = %v", got)
	}
}

func TestSpriteAnimationSystemLooping(t *testing.T) {
	animated, err := NewAnimatedSprite(newTestClip(t, "a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	animated.Loop = LoopAll
	entity, sys, ctx, reader := newAnimatedSystemWorld(t, animated, 100*time.Millisecond)
	for i, tt := range []struct {
		dt     time.Duration
		region string
		loops  uint64
	}{
		{100 * time.Millisecond, "b", 0},
		{100 * time.Millisecond, "a", 1},
		{500 * time.Millisecond, "b", 2},
		{500 * time.Millisecond, "a", 3},
	} {
		tick := uint64(i + 1)
		ctx.DeltaTime = tt.dt
		tickAt(t, sys, ctx, tick)
		anim := readAnimated(t, entity, ctx.World)
		if got := animatedRegion(t, entity, ctx.World); got != tt.region || anim.HasCompleted() || anim.JustCompleted(tick) != (tt.loops > 0) {
			t.Fatalf("tick %d: region %q, state %+v", tick, got, anim)
		}
		got := drainCompleted(reader)
		if tt.loops == 0 {
			if len(got) != 0 {
				t.Fatalf("tick %d: unexpected signals %v", tick, got)
			}
		} else if len(got) != 1 || got[0] != (Completed{Entity: entity, Tick: tick, Loops: tt.loops}) {
			t.Fatalf("tick %d: signals = %v, want %d loops", tick, got, tt.loops)
		}
	}
	anim := readAnimated(t, entity, ctx.World)
	anim.Loop = LoopNone
	if anim.HasCompleted() {
		t.Fatal("switching to LoopNone treated a wrap as one-shot completion")
	}
}

func TestSpriteAnimationSystemSkipsWithoutStoppingOtherSprites(t *testing.T) {
	clipless, sys, ctx, reader := newAnimatedSystemWorld(t, AnimatedSprite{}, 100*time.Millisecond)
	animated, err := NewAnimatedSprite(newTestClip(t, "a", "b", "c"))
	if err != nil {
		t.Fatal(err)
	}
	animated.Paused = true
	paused, err := ctx.World.NewEntity(animated)
	if err != nil {
		t.Fatal(err)
	}
	animated.Paused = false
	active, err := ctx.World.NewEntity(animated)
	if err != nil {
		t.Fatal(err)
	}
	tickAt(t, sys, ctx, 1)
	if _, _, ok, err := readAnimated(t, clipless, ctx.World).CurrentFrame(); err != nil || ok {
		t.Fatalf("clipless CurrentFrame() = %v, %v", ok, err)
	}
	if got := animatedRegion(t, paused, ctx.World); got != "a" {
		t.Fatalf("paused region = %q, want a", got)
	}
	if got := animatedRegion(t, active, ctx.World); got != "b" {
		t.Fatalf("active region = %q, want b", got)
	}
	if err := paused.Update(ctx.World, func(a *AnimatedSprite) { a.Paused = false }); err != nil {
		t.Fatal(err)
	}
	tickAt(t, sys, ctx, 2)
	if got := animatedRegion(t, paused, ctx.World); got != "b" {
		t.Fatalf("resumed region = %q, want b", got)
	}
	if got := drainCompleted(reader); len(got) != 0 {
		t.Fatalf("unexpected signals = %v", got)
	}
}

func TestSpriteAnimationSystemRejectsInvalidEntryWrites(t *testing.T) {
	clip, err := NewAnimatedSprite(newTestClip(t, "a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	paused := clip
	paused.Paused = true
	for _, tt := range []struct {
		name     string
		animated AnimatedSprite
	}{
		{"active", clip},
		{"paused", paused},
		{"clipless", AnimatedSprite{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			entity, sys, ctx, reader := newAnimatedSystemWorld(t, tt.animated, 100*time.Millisecond)
			for entry := range core.NewQuery(ctx.World).With(AnimatedSprite{}).Execute() {
				anim, _ := entry.Component[AnimatedSprite]()
				anim.PlaybackMultiplier = -1
				entry.SetComponent(anim)
			}
			before := readAnimated(t, entity, ctx.World)
			if err := sys.Update(ctx); err == nil || !strings.Contains(err.Error(), "playback multiplier") {
				t.Fatalf("Update() = %v, want invalid multiplier error", err)
			}
			if got := readAnimated(t, entity, ctx.World); !reflect.DeepEqual(got, before) {
				t.Fatalf("failed update changed sprite: got %+v, want %+v", got, before)
			}
			if got := drainCompleted(reader); len(got) != 0 {
				t.Fatalf("failed update emitted signals: %v", got)
			}
		})
	}
}

func TestSpriteAnimationSystemRejectsUnrepresentableAdvance(t *testing.T) {
	animated, err := NewAnimatedSprite(newTestClip(t, "a", "b"))
	if err != nil {
		t.Fatal(err)
	}
	animated.PlaybackMultiplier = math.MaxFloat64
	entity, sys, ctx, reader := newAnimatedSystemWorld(t, animated, 100*time.Millisecond)
	if err := sys.Update(ctx); err == nil || !strings.Contains(err.Error(), "not representable") {
		t.Fatalf("Update() = %v, want a not-representable advance error", err)
	}
	if got := readAnimated(t, entity, ctx.World); !reflect.DeepEqual(got, animated) {
		t.Fatalf("failed update changed sprite: got %+v, want %+v", got, animated)
	}
	if got := drainCompleted(reader); len(got) != 0 {
		t.Fatalf("failed update emitted signals: %v", got)
	}
}
