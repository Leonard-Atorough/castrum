package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
)

func TestBalanceControls(t *testing.T) {
	for _, tt := range []struct {
		name      string
		key       input.Key
		held      bool
		initial   float64
		paused    bool
		want      float64
		wantPause bool
	}{
		{"sweep left", input.KeyArrowLeft, true, 0, false, -0.5, false},
		{"sweep right", input.KeyArrowRight, true, 0, false, 0.5, false},
		{"clamp left", input.KeyArrowLeft, true, -0.9, false, -1, false},
		{"clamp right", input.KeyArrowRight, true, 0.9, false, 1, false},
		{"left preset", input.Key1, false, 0, false, -1, false},
		{"center preset", input.Key2, false, 1, false, 0, false},
		{"right preset", input.Key3, false, 0, false, 1, false},
		{"pause", input.KeySpace, false, 0.5, false, 0.5, true},
		{"resume", input.KeySpace, false, -0.5, true, -0.5, false},
		{"adjust while paused", input.KeyArrowRight, true, 0, true, 0.5, true},
		{"restart preserves balance", input.KeyR, false, -0.5, true, -0.5, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			world := core.NewWorld()
			source := audio.NewSource(audioassets.Music)
			source.Balance = tt.initial
			source.Paused = tt.paused
			source.Completed = true
			entity, err := world.NewEntity(source)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := &input.Snapshot{}
			snapshot.Keys[tt.key].Held = tt.held
			snapshot.Keys[tt.key].Pressed = !tt.held
			ctx := &core.Context{World: world, Input: snapshot, DeltaTime: time.Second}
			if err := balanceSystem(entity, debug.NewPanel()).Update(ctx); err != nil {
				t.Fatal(err)
			}
			got, ok := entity.Component[audio.Source](world)
			if !ok || got.Balance != tt.want || got.Paused != tt.wantPause {
				t.Fatalf("source = %+v, want balance=%v paused=%t", got, tt.want, tt.wantPause)
			}
			if tt.key == input.KeyR && got.Completed {
				t.Fatal("restart did not clear completion")
			}
			if err := got.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
