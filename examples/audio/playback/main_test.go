package main

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
)

func TestPlaybackOneShotsAndSourceControls(t *testing.T) {
	world := core.NewWorld()
	music := audio.NewSource(audioassets.Music)
	music.Load = audio.LoadStream
	music.Loop = audio.LoopForever
	entity, err := world.NewEntity(music)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeySpace].Pressed = true
	ctx := &core.Context{World: world, Input: snapshot}
	system := playbackSystem(entity, debug.NewPanel())
	for range 2 {
		if err := system.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	snapshot.Keys[input.KeySpace].Pressed = false
	snapshot.Keys[input.KeySpace].Held = true
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	count := 0
	for e := range core.NewQuery(world).With(audio.Source{}).Execute() {
		source, _ := e.Component[audio.Source]()
		if source.Audio == audioassets.Magic {
			if source.Load != audio.LoadEager || source.Loop != audio.LoopNone {
				t.Fatalf("effect = %+v, want eager non-looping source", source)
			}
			count++
		}
	}
	if count != 2 {
		t.Fatalf("effects = %d, want two independent presses and none from hold", count)
	}
	snapshot.Keys[input.KeyP].Pressed = true
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	source, _ := entity.Component[audio.Source](world)
	if !source.Paused {
		t.Fatal("P did not pause music")
	}
	snapshot.Keys[input.KeyP].Pressed = false
	snapshot.Keys[input.KeyR].Pressed = true
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	source, _ = entity.Component[audio.Source](world)
	if source.Paused || source.Completed || source.Load != audio.LoadStream || source.Loop != audio.LoopForever {
		t.Fatalf("restarted source = %+v", source)
	}
}
