package animation

import (
	"reflect"
	"testing"
	"time"
)

func TestAnimateScenarios(t *testing.T) {
	const dt = 100 * time.Millisecond
	mustAnim := func(t *testing.T, state string) Animator {
		t.Helper()
		a, err := NewAnimator("hero", state)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	tests := []struct {
		name   string
		anim   func(t *testing.T) Animator
		ticks  int
		mutate func(tick int, a *Animator)
		want   []tickTrace
	}{
		{
			name:  "loop wraps and emits Completed per pass",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "idle") },
			ticks: 3,
			want: []tickTrace{
				{State: "idle", Frames: []string{"body:b:1.00"}, Events: []string{"Started:idle"}},
				{State: "idle", Frames: []string{"body:a:1.00"}, Events: []string{"Completed:idle:1"}},
				{State: "idle", Frames: []string{"body:b:1.00"}},
			},
		},
		{
			name:  "loop once holds the final frame after one Completed",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "die") },
			ticks: 4,
			want: []tickTrace{
				{State: "die", Frames: []string{"body:e:1.00"}, Events: []string{"Started:die"}},
				{State: "die", Frames: []string{"body:e:1.00"}, Events: []string{"Completed:die:1"}},
				{State: "die", Frames: []string{"body:e:1.00"}},
				{State: "die", Frames: []string{"body:e:1.00"}},
			},
		},
		{
			name:  "Next transition wins and crossfades over the target's EnterBlend",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "attack") },
			ticks: 4,
			want: []tickTrace{
				{State: "idle", Frames: []string{"body:c:1.00", "body:a:0.00"}, Events: []string{"Started:attack", "Started:idle", "Completed:attack:1"}},
				{State: "idle", Frames: []string{"body:c:0.60", "body:b:0.40"}},
				{State: "idle", Frames: []string{"body:c:0.20", "body:a:0.80"}, Events: []string{"Completed:idle:1"}},
				{State: "idle", Frames: []string{"body:b:1.00"}},
			},
		},
		{
			name:  "ping pong reflects at both ends",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "bounce") },
			ticks: 6,
			want: []tickTrace{
				{State: "bounce", Frames: []string{"body:q:1.00"}, Events: []string{"Started:bounce"}},
				{State: "bounce", Frames: []string{"body:r:1.00"}},
				{State: "bounce", Frames: []string{"body:q:1.00"}, Events: []string{"Completed:bounce:1"}},
				{State: "bounce", Frames: []string{"body:p:1.00"}},
				{State: "bounce", Frames: []string{"body:q:1.00"}, Events: []string{"Completed:bounce:1"}},
				{State: "bounce", Frames: []string{"body:r:1.00"}},
			},
		},
		{
			name:  "desired state write starts the new state",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "idle") },
			ticks: 3,
			mutate: func(tick int, a *Animator) {
				if tick == 2 {
					a.State = "die"
				}
			},
			want: []tickTrace{
				{State: "idle", Frames: []string{"body:b:1.00"}, Events: []string{"Started:idle"}},
				{State: "die", Frames: []string{"body:e:1.00"}, Events: []string{"Started:die"}},
				{State: "die", Frames: []string{"body:e:1.00"}, Events: []string{"Completed:die:1"}},
			},
		},
		{
			name:  "speed multiplier completes a one-shot in one tick",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "die") },
			ticks: 2,
			mutate: func(tick int, a *Animator) {
				if tick == 1 {
					a.Speed = 2
				}
			},
			want: []tickTrace{
				{State: "die", Frames: []string{"body:e:1.00"}, Events: []string{"Started:die", "Completed:die:1"}},
				{State: "die", Frames: []string{"body:e:1.00"}},
			},
		},
		{
			name:  "paused holds the frame until resumed",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "idle") },
			ticks: 3,
			mutate: func(tick int, a *Animator) {
				if tick == 2 {
					a.Paused = true
				}
				if tick == 3 {
					a.Paused = false
				}
			},
			want: []tickTrace{
				{State: "idle", Frames: []string{"body:b:1.00"}, Events: []string{"Started:idle"}},
				{State: "idle", Frames: []string{"body:b:1.00"}},
				{State: "idle", Frames: []string{"body:a:1.00"}, Events: []string{"Completed:idle:1"}},
			},
		},
		{
			name:  "paused from the first tick still starts exactly once",
			anim:  func(t *testing.T) Animator { return mustAnim(t, "idle") },
			ticks: 2,
			mutate: func(tick int, a *Animator) {
				a.Paused = true
			},
			want: []tickTrace{
				{State: "idle", Frames: []string{"body:a:1.00"}, Events: []string{"Started:idle"}},
				{State: "idle", Frames: []string{"body:a:1.00"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runScenario(t, Animate, testSets(t), tt.anim(t), dt, tt.ticks, tt.mutate)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("trace mismatch\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}
