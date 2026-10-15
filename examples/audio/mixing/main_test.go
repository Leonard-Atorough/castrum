package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
)

func TestMixingVolumeControls(t *testing.T) {
	for _, tt := range []struct {
		name   string
		key    input.Key
		master float64
		music  float64
	}{
		{"master down", input.KeyArrowDown, 0.5, 1},
		{"master upper clamp", input.KeyArrowUp, 1, 1},
		{"music down", input.KeyArrowLeft, 1, 0.5},
		{"music upper clamp", input.KeyArrowRight, 1, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mixer := audio.NewMixer()
			snapshot := &input.Snapshot{}
			snapshot.Keys[tt.key].Held = true
			ctx := &core.Context{World: core.NewWorld(), Input: snapshot, DeltaTime: time.Second}
			if err := mixingSystem(mixer).Update(ctx); err != nil {
				t.Fatal(err)
			}
			if mixer.Master() != tt.master || mixer.GroupVolume(audio.GroupMusic) != tt.music {
				t.Fatalf("master=%v music=%v; want %v, %v",
					mixer.Master(), mixer.GroupVolume(audio.GroupMusic), tt.master, tt.music)
			}
			if mixer.GroupVolume(audio.GroupSFX) != 1 {
				t.Fatal("music volume control changed the SFX bus")
			}
		})
	}
}

func TestMixingEffectsDuringGlobalPause(t *testing.T) {
	world := core.NewWorld()
	mixer := audio.NewMixer()
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeyEnter].Pressed = true
	snapshot.Keys[input.KeySpace].Pressed = true
	ctx := &core.Context{World: world, Input: snapshot}
	system := mixingSystem(mixer)
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !mixer.Paused() {
		t.Fatal("Enter did not pause mixer")
	}
	count := 0
	for entity := range core.NewQuery(world).With(audio.Source{}).Execute() {
		source, _ := entity.Component[audio.Source]()
		if source.Audio != audioassets.Magic || source.Group != audio.GroupSFX || source.Pause != audio.PauseContinues {
			t.Fatalf("effect = %+v, want SFX that continues through global pause", source)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("effect count = %d, want 1", count)
	}
	snapshot.Keys[input.KeySpace].Pressed = false
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mixer.Paused() {
		t.Fatal("second Enter did not resume mixer")
	}
}
