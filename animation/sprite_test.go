package animation

import (
	"math"
	"reflect"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/sprite"
)

var animSheet = asset.FileSheet("atlas.json")

func TestNewSpriteClipRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name   string
		source asset.SheetRef
		frames []string
		fps    float64
	}{
		{name: "source", frames: []string{"a"}, fps: 10},
		{name: "no frames", source: animSheet, fps: 10},
		{name: "empty frame", source: animSheet, frames: []string{"a", ""}, fps: 10},
		{name: "zero rate", source: animSheet, frames: []string{"a"}},
		{name: "negative rate", source: animSheet, frames: []string{"a"}, fps: -1},
		{name: "infinite rate", source: animSheet, frames: []string{"a"}, fps: math.Inf(1)},
		{name: "nan rate", source: animSheet, frames: []string{"a"}, fps: math.NaN()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewSpriteClip(tt.source, tt.frames, tt.fps); err == nil {
				t.Fatal("NewSpriteClip accepted invalid metadata")
			}
		})
	}
}

func TestSpriteClipIsSharedWithoutExposingFrames(t *testing.T) {
	frames := []string{"a", "b"}
	clip, err := NewSpriteClip(animSheet, frames, 10)
	if err != nil {
		t.Fatal(err)
	}
	frames[0] = "changed"

	first, err := NewAnimatedSprite(clip)
	if err != nil {
		t.Fatal(err)
	}
	var second AnimatedSprite
	if err := second.SetClip(clip); err != nil {
		t.Fatal(err)
	}
	frames[0] = "changed again"

	for _, animated := range []AnimatedSprite{first, second} {
		_, region, ok, err := animated.CurrentFrame()
		if err != nil || !ok || region != "a" {
			t.Fatalf("CurrentFrame() = %v, %v, %v; want original first frame", region, ok, err)
		}
	}
}

func TestAnimatedSpriteZeroValue(t *testing.T) {
	var animated AnimatedSprite
	if err := animated.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	if _, _, ok, err := animated.CurrentFrame(); err != nil || ok {
		t.Fatalf("CurrentFrame() = %t, %v; want no frame", ok, err)
	}
	if animated.HasCompleted() || animated.JustCompleted(0) {
		t.Fatal("clipless sprite reported completion")
	}
	animated.Paused = true
	animated.Restart()
	if !animated.Paused {
		t.Fatal("Restart changed a clipless sprite")
	}
	if _, err := NewAnimatedSprite(SpriteClip{}); err == nil {
		t.Fatal("NewAnimatedSprite accepted an invalid clip")
	}
}

func TestAnimatedSpriteRejectsInvalidSettings(t *testing.T) {
	tests := []struct {
		name     string
		animated AnimatedSprite
	}{
		{name: "loop", animated: AnimatedSprite{Loop: LoopMode(-1)}},
		{name: "speed", animated: AnimatedSprite{PlaybackMultiplier: math.NaN()}},
		{name: "style", animated: AnimatedSprite{Style: sprite.Style{Layer: 32}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.animated.Validate(); err == nil {
				t.Fatal("Validate accepted invalid settings")
			}
			if _, _, _, err := tt.animated.CurrentFrame(); err == nil {
				t.Fatal("CurrentFrame accepted invalid settings")
			}
		})
	}
}

func TestAnimatedSpriteSetClipAndRestart(t *testing.T) {
	clip, err := NewSpriteClip(animSheet, []string{"a", "b"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	animated, err := NewAnimatedSprite(clip)
	if err != nil {
		t.Fatal(err)
	}
	animated.Loop = LoopAll
	animated.PlaybackMultiplier = 2
	animated.Style.Layer = 7
	animated.frame = 1
	animated.elapsed = 0.05
	animated.completedOn = 4
	animated.completed = true
	animated.Paused = true

	before := animated
	if err := animated.SetClip(SpriteClip{}); err == nil {
		t.Fatal("SetClip accepted an invalid clip")
	}
	if !reflect.DeepEqual(animated, before) {
		t.Fatal("invalid SetClip changed the sprite")
	}

	if err := animated.SetClip(clip); err != nil {
		t.Fatal(err)
	}
	if animated.frame != 0 || animated.elapsed != 0 || animated.completedOn != 0 || animated.completed || animated.Paused {
		t.Fatalf("SetClip did not reset playback: %+v", animated)
	}
	if animated.Loop != LoopAll || animated.PlaybackMultiplier != 2 || animated.Style.Layer != 7 {
		t.Fatalf("SetClip changed playback settings or style: %+v", animated)
	}
	if ref, region, ok, err := animated.CurrentFrame(); err != nil || !ok || region != "a" || ref != animSheet {
		t.Fatalf("CurrentFrame() = %v, %v, %v; want first frame", region, ok, err)
	}

	animated.frame = 1
	animated.elapsed = 0.05
	animated.completedOn = 5
	animated.completed = true
	animated.Paused = true
	animated.Restart()
	if animated.frame != 0 || animated.elapsed != 0 || animated.completedOn != 0 || animated.completed || animated.Paused {
		t.Fatalf("Restart did not reset playback: %+v", animated)
	}

	otherSheet := asset.FileSheet("other.json")
	other, err := NewSpriteClip(otherSheet, []string{"c"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if err := animated.SetClip(other); err != nil {
		t.Fatal(err)
	}
	if ref, region, ok, err := animated.CurrentFrame(); err != nil || !ok || region != "c" || ref != otherSheet {
		t.Fatalf("CurrentFrame() = %v, %v, %v; want replacement frame", region, ok, err)
	}
}
