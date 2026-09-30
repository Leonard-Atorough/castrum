package animation

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

func newAnimWorld(t *testing.T, anim Animation, clip AnimationClip) (*core.Entity, core.System, *core.Context) {
	t.Helper()
	w := core.NewWorld()
	store := NewClipStore()
	if err := store.Add(anim.Clip, clip); err != nil {
		t.Fatalf("add clip: %v", err)
	}
	entity, err := w.NewEntity(anim, core.Sprite{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	ctx := &core.Context{World: w, DeltaTime: 100 * time.Millisecond}
	return entity, NewAnimationSystem(store), ctx
}

func tick(t *testing.T, sys core.System, ctx *core.Context, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := sys.Update(ctx); err != nil {
			t.Fatalf("update: %v", err)
		}
	}
}

func readAnim(t *testing.T, e *core.Entity, ctx *core.Context) Animation {
	t.Helper()
	anim, ok := e.Component[Animation](ctx.World)
	if !ok {
		t.Fatal("animation component missing")
	}
	return anim
}

func readSprite(t *testing.T, e *core.Entity, ctx *core.Context) core.Sprite {
	t.Helper()
	sprite, ok := e.Component[core.Sprite](ctx.World)
	if !ok {
		t.Fatal("sprite component missing")
	}
	return sprite
}

func region(s core.Sprite) (core.AtlasSource, bool) {
	src, ok := s.Drawable.(core.AtlasSource)
	return src, ok
}

func TestSystemAdvancesAndStamps(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk"},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b"}, FPS: 10, Loop: LoopForever},
	)
	tick(t, sys, ctx, 1)

	anim := readAnim(t, entity, ctx)
	if anim.Current != 1 {
		t.Fatalf("current = %d, want 1", anim.Current)
	}
	src, ok := region(readSprite(t, entity, ctx))
	if !ok || src.Atlas != "atlas" || src.Region != "b" {
		t.Fatalf("drawable = %v, want atlas/b", readSprite(t, entity, ctx).Drawable)
	}

	// Second tick wraps the two-frame loop back to the start.
	tick(t, sys, ctx, 1)
	anim = readAnim(t, entity, ctx)
	if anim.Current != 0 {
		t.Fatalf("current after wrap = %d, want 0", anim.Current)
	}
}

// A frame slower than the tick rate must accumulate partial time
// across ticks or the animation never advances at all.
func TestSystemAccumulatesSubFrameTime(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk"},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b", "c"}, FPS: 10, Loop: LoopNone},
	)
	ctx.DeltaTime = 50 * time.Millisecond
	tick(t, sys, ctx, 1)

	anim := readAnim(t, entity, ctx)
	if anim.Current != 0 {
		t.Fatalf("current after one 50ms tick of a 10fps clip = %d, want 0", anim.Current)
	}
	if math.Abs(anim.Elapsed-0.05) > 1e-9 {
		t.Fatalf("elapsed = %v, want ~0.05 (accumulation lost)", anim.Elapsed)
	}

	tick(t, sys, ctx, 1)
	anim = readAnim(t, entity, ctx)
	if anim.Current != 1 {
		t.Fatalf("current after two ticks = %d, want 1", anim.Current)
	}
}

// Any number of frames may cross in one tick; the advance is closed
// form, not one step per tick.
func TestSystemAdvancesMultipleFramesPerTick(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk"},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b", "c"}, FPS: 10, Loop: LoopForever},
	)
	ctx.DeltaTime = 250 * time.Millisecond // two 100ms frames per tick
	tick(t, sys, ctx, 1)

	anim := readAnim(t, entity, ctx)
	if anim.Current != 2 {
		t.Fatalf("current = %d, want 2", anim.Current)
	}
	if anim.Elapsed >= 0.1 {
		t.Fatalf("elapsed = %v, want below one frame (remainder consumed)", anim.Elapsed)
	}
}

func TestLoopNoneHoldsLastFramePaused(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk"},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b"}, FPS: 10, Loop: LoopNone},
	)
	tick(t, sys, ctx, 1)
	if anim := readAnim(t, entity, ctx); anim.Current != 1 {
		t.Fatalf("current = %d, want 1", anim.Current)
	}

	tick(t, sys, ctx, 1)
	anim := readAnim(t, entity, ctx)
	if anim.Current != 1 || !anim.Paused {
		t.Fatalf("current = %d, paused = %v, want 1 and paused", anim.Current, anim.Paused)
	}
	src, ok := region(readSprite(t, entity, ctx))
	if !ok || src.Region != "b" {
		t.Fatalf("drawable region = %v, want b (last frame held)", src.Region)
	}

	// A paused animation stays put.
	tick(t, sys, ctx, 5)
	if anim := readAnim(t, entity, ctx); anim.Current != 1 {
		t.Fatalf("current after further ticks = %d, want 1", anim.Current)
	}
}

// The first tick stamps frame 0 before any frame has crossed: a
// spawned animation is visible immediately.
func TestFirstTickStampsFrameZero(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk"},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b"}, FPS: 10, Loop: LoopForever},
	)
	ctx.DeltaTime = 50 * time.Millisecond // below one frame duration
	tick(t, sys, ctx, 1)

	anim := readAnim(t, entity, ctx)
	if anim.Current != 0 {
		t.Fatalf("current = %d, want 0", anim.Current)
	}
	src, ok := region(readSprite(t, entity, ctx))
	if !ok || src.Atlas != "atlas" || src.Region != "a" {
		t.Fatalf("drawable = %v, want atlas/a on the first tick", readSprite(t, entity, ctx).Drawable)
	}
}

// A missing clip reference fails the tick naming the clip and the
// entity.
func TestMissingClipFailsFast(t *testing.T) {
	w := core.NewWorld()
	if _, err := w.NewEntity(Animation{Clip: "nope"}, core.Sprite{}); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	ctx := &core.Context{World: w, DeltaTime: 100 * time.Millisecond}
	err := NewAnimationSystem(NewClipStore()).Update(ctx)
	if err == nil {
		t.Fatal("missing clip advanced without error")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error %q does not name the clip", err)
	}
	if !strings.Contains(err.Error(), "entity") {
		t.Fatalf("error %q does not name the entity", err)
	}
}

// PlaybackMultiplier 0 is the normal rate: the zero value plays.
func TestPlaybackMultiplierZeroIsNormalRate(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk"},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b"}, FPS: 10, Loop: LoopForever},
	)
	tick(t, sys, ctx, 1)
	if anim := readAnim(t, entity, ctx); anim.Current != 1 {
		t.Fatalf("current = %d, want 1 (zero multiplier must advance)", anim.Current)
	}
}

func TestPlaybackMultiplierScalesAdvance(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk", PlaybackMultiplier: 2},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b", "c", "d"}, FPS: 10, Loop: LoopNone},
	)
	tick(t, sys, ctx, 1)
	if anim := readAnim(t, entity, ctx); anim.Current != 2 {
		t.Fatalf("current = %d, want 2 (double rate, one tick)", anim.Current)
	}
}

func TestPausedAnimationSkips(t *testing.T) {
	entity, sys, ctx := newAnimWorld(t,
		Animation{Clip: "walk", Paused: true},
		AnimationClip{Source: "atlas", Frames: []string{"a", "b"}, FPS: 10, Loop: LoopForever},
	)
	tick(t, sys, ctx, 3)
	anim := readAnim(t, entity, ctx)
	if anim.Current != 0 || anim.Elapsed != 0 {
		t.Fatalf("current = %d, elapsed = %v, want untouched", anim.Current, anim.Elapsed)
	}
	if _, ok := region(readSprite(t, entity, ctx)); ok {
		t.Fatal("paused animation stamped the drawable")
	}
}

func TestAnimationValidate(t *testing.T) {
	cases := []struct {
		name string
		anim Animation
		want bool
	}{
		{"valid", Animation{Clip: "walk"}, true},
		{"empty clip", Animation{}, false},
		{"negative current", Animation{Clip: "walk", Current: -1}, false},
		{"negative elapsed", Animation{Clip: "walk", Elapsed: -1}, false},
		{"negative multiplier", Animation{Clip: "walk", PlaybackMultiplier: -1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.anim.Validate()
			if tc.want && err != nil {
				t.Fatalf("want valid, got %v", err)
			}
			if !tc.want && err == nil {
				t.Fatal("want invalid, got nil")
			}
		})
	}
}

func TestAnimationClipValidate(t *testing.T) {
	cases := []struct {
		name string
		clip AnimationClip
		want bool
	}{
		{"valid", AnimationClip{Source: "atlas", Frames: []string{"a"}, FPS: 10}, true},
		{"empty source", AnimationClip{Frames: []string{"a"}, FPS: 10}, false},
		{"no frames", AnimationClip{Source: "atlas", FPS: 10}, false},
		{"zero fps", AnimationClip{Source: "atlas", Frames: []string{"a"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.clip.Validate()
			if tc.want && err != nil {
				t.Fatalf("want valid, got %v", err)
			}
			if !tc.want && err == nil {
				t.Fatal("want invalid, got nil")
			}
		})
	}
}

func TestPlaybackMethods(t *testing.T) {
	a := Animation{Clip: "walk", Current: 1, Elapsed: 0.04}

	a.Restart()
	if a.Current != 0 || a.Elapsed != 0 || a.Paused {
		t.Fatalf("Restart: %+v", a)
	}

	a.Stop()
	if a.Current != 0 || a.Elapsed != 0 || !a.Paused {
		t.Fatalf("Stop: %+v", a)
	}

	a.Resume()
	if a.Paused {
		t.Fatal("Resume: still paused")
	}
	a.Pause()
	if !a.Paused {
		t.Fatal("Pause: not paused")
	}
}

func TestClipStoreAddCopiesFrames(t *testing.T) {
	store := NewClipStore()
	frames := []string{"a", "b"}
	if err := store.Add("walk", AnimationClip{Source: "atlas", Frames: frames, FPS: 10}); err != nil {
		t.Fatalf("add: %v", err)
	}

	// The store owns its frames: mutating the source slice must not
	// reach the stored clip.
	frames[0] = "mutated"
	clip, err := store.Get("walk")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if clip.Frames[0] != "a" {
		t.Fatalf("frames = %v, want the copy untouched", clip.Frames)
	}
}

func TestClipStoreAddErrors(t *testing.T) {
	store := NewClipStore()
	if err := store.Add("", AnimationClip{Source: "atlas", Frames: []string{"a"}, FPS: 10}); err == nil {
		t.Fatal("empty name accepted")
	}
	if err := store.Add("walk", AnimationClip{Source: "atlas", Frames: []string{"a"}, FPS: 10}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := store.Add("walk", AnimationClip{Source: "atlas", Frames: []string{"a"}, FPS: 10}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := store.Add("bad", AnimationClip{Source: "atlas", Frames: []string{"a"}}); err == nil {
		t.Fatal("invalid clip accepted")
	}
}

func TestClipStoreGetMissing(t *testing.T) {
	store := NewClipStore()
	if _, err := store.Get("nope"); err == nil {
		t.Fatal("missing clip returned without error")
	}
}
