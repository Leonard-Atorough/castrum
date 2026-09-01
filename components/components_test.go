package components

import "testing"

func TestAnimation_Reset(t *testing.T) {
	t.Run("AutoPlay true resumes playing after reset", func(t *testing.T) {
		a := &Animation{FrameIndex: 3, FrameTime: 1.5, Playing: false, AutoPlay: true}
		a.Reset()

		if a.FrameIndex != 0 || a.FrameTime != 0 {
			t.Fatalf("expected frame state reset, got FrameIndex=%d FrameTime=%v", a.FrameIndex, a.FrameTime)
		}
		if !a.Playing {
			t.Fatal("expected Playing to follow AutoPlay=true")
		}
	})

	t.Run("AutoPlay false leaves the animation paused after reset", func(t *testing.T) {
		a := &Animation{FrameIndex: 3, FrameTime: 1.5, Playing: true, AutoPlay: false}
		a.Reset()

		if a.Playing {
			t.Fatal("expected Playing to follow AutoPlay=false")
		}
	})
}
