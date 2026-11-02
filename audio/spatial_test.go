package audio

import (
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

func provideListener(t *testing.T, w *core.World, listener *Listener) {
	t.Helper()
	if err := w.Provide(func(*core.World) (*Listener, error) { return listener, nil }); err != nil {
		t.Fatal(err)
	}
}

func TestSpatialPlayback(t *testing.T) {
	for _, tc := range []struct {
		name        string
		position    geom.Vector2
		curve       geom.Ease
		bias        float64
		wantGain    float64
		wantBalance float64
	}{
		{name: "center", wantGain: 1},
		{name: "right", position: geom.Vector2{X: 50}, wantGain: 0.5, wantBalance: 0.5},
		{name: "left", position: geom.Vector2{X: -50}, wantGain: 0.5, wantBalance: -0.5},
		{name: "diagonal", position: geom.Vector2{X: 30, Y: 40}, wantGain: 0.5, wantBalance: 0.3},
		{name: "range", position: geom.Vector2{X: 100}, wantBalance: 1},
		{name: "outside range", position: geom.Vector2{X: -150}, wantBalance: -1},
		{name: "quadratic", position: geom.Vector2{X: 50}, curve: geom.EaseInQuad, wantGain: 0.75, wantBalance: 0.5},
		{name: "bias clamped", position: geom.Vector2{X: 50}, bias: 0.75, wantGain: 0.5, wantBalance: 1},
		{name: "negative bias clamped", position: geom.Vector2{X: -50}, bias: -0.75, wantGain: 0.5, wantBalance: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := Source{Volume: 0.8, Spatial: true, Balance: tc.bias, Attenuation: tc.curve}
			entity, ctx := newAudioWorld(t, source, "sound.wav")
			if err := entity.AddComponent(ctx.World, core.Transform{Position: tc.position}); err != nil {
				t.Fatal(err)
			}
			provideListener(t, ctx.World, &Listener{Enabled: true, Mode: FixedPosition, Range: 100})
			mixer := NewMixer()
			if err := mixer.SetMaster(0.5); err != nil {
				t.Fatal(err)
			}
			if err := mixer.SetGroupVolume(GroupSFX, 0.5); err != nil {
				t.Fatal(err)
			}
			fake := &fakeController{live: true}
			runAudio(t, NewAudioSystem(fake, mixer), ctx)
			got := fake.syncs[0].want
			if math.Abs(got.Volume-0.2*tc.wantGain) > 1e-12 || math.Abs(got.Balance-tc.wantBalance) > 1e-12 || !got.Playing {
				t.Fatalf("playback = %+v, want volume %v, balance %v, playing", got, 0.2*tc.wantGain, tc.wantBalance)
			}
			stored := readSource(t, entity, ctx)
			if stored.Volume != source.Volume || stored.Balance != source.Balance {
				t.Fatalf("spatialization modified source: %+v", stored)
			}
		})
	}
}

func TestSpatialLiveUpdatesAndPause(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Volume: 1, Balance: -0.25, Spatial: true}, "sound.wav")
	if err := entity.AddComponent(ctx.World, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
		t.Fatal(err)
	}
	listener := &Listener{Enabled: true, Mode: FixedPosition, Range: 100}
	provideListener(t, ctx.World, listener)
	fake := &fakeController{live: true}
	mixer := NewMixer()
	sys := NewAudioSystem(fake, mixer)
	runAudio(t, sys, ctx)

	mixer.PauseAll()
	if err := entity.Update(ctx.World, func(tr *core.Transform) { tr.Position.X = -50 }); err != nil {
		t.Fatal(err)
	}
	runAudio(t, sys, ctx)
	got := fake.syncs[len(fake.syncs)-1].want
	if got.Playing || got.Restart || got.Volume != 0.5 || got.Balance != -0.75 {
		t.Fatalf("paused movement = %+v", got)
	}
	mixer.ResumeAll()
	listener.Position.X = -25
	runAudio(t, sys, ctx)
	got = fake.syncs[len(fake.syncs)-1].want
	if !got.Playing || got.Restart || got.Volume != 0.75 || got.Balance != -0.5 {
		t.Fatalf("moved listener = %+v", got)
	}
	if err := entity.Update(ctx.World, func(s *Source) { s.Attenuation = geom.EaseInQuad }); err != nil {
		t.Fatal(err)
	}
	runAudio(t, sys, ctx)
	got = fake.syncs[len(fake.syncs)-1].want
	if got.Restart || got.Volume != 0.9375 || got.Balance != -0.5 {
		t.Fatalf("updated curve = %+v", got)
	}
	listener.Enabled = false
	runAudio(t, sys, ctx)
	got = fake.syncs[len(fake.syncs)-1].want
	if !got.Playing || got.Restart || got.Volume != 1 || got.Balance != -0.25 {
		t.Fatalf("disabled listener = %+v", got)
	}
	listener.Enabled = true
	if err := entity.Update(ctx.World, func(s *Source) { s.Spatial = false }); err != nil {
		t.Fatal(err)
	}
	if err := entity.RemoveComponent[core.Transform](ctx.World); err != nil {
		t.Fatal(err)
	}
	runAudio(t, sys, ctx)
	got = fake.syncs[len(fake.syncs)-1].want
	if got.Restart || got.Volume != 1 || got.Balance != -0.25 {
		t.Fatalf("disabled spatial source = %+v", got)
	}
	if err := entity.RemoveComponent[Source](ctx.World); err != nil {
		t.Fatal(err)
	}
	before := len(fake.syncs)
	runAudio(t, sys, ctx)
	if len(fake.syncs) != before || fake.sweeps != 7 {
		t.Fatalf("removed source: syncs = %d, sweeps = %d", len(fake.syncs), fake.sweeps)
	}
}

func TestDisabledListenerDoesNotRequireTransform(t *testing.T) {
	_, ctx := newAudioWorld(t, Source{Volume: 1, Spatial: true, Balance: 0.5}, "sound.wav")
	provideListener(t, ctx.World, &Listener{})
	fake := &fakeController{live: true}
	runAudio(t, NewAudioSystem(fake, NewMixer()), ctx)
	if got := fake.syncs[0].want; got.Volume != 1 || got.Balance != 0.5 {
		t.Fatalf("disabled listener playback = %+v", got)
	}
}

func TestSpatialCameraFollow(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Volume: 1, Spatial: true}, "sound.wav")
	ctx.LogicalWidth, ctx.LogicalHeight = 100, 100
	if err := entity.AddComponent(ctx.World, core.Transform{Position: geom.Vector2{X: 100, Y: 150}}); err != nil {
		t.Fatal(err)
	}
	if _, err := render.SpawnEngineCamera(ctx.World); err != nil {
		t.Fatal(err)
	}
	camera, err := ctx.World.NewEntity(
		render.Camera{Primary: true, Bounds: geom.RectFromMinMax(geom.Vector2{}, geom.Vector2{X: 200, Y: 200})},
		core.Transform{Position: geom.Vector2{X: 300, Y: 100}, Rotation: math.Pi / 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	provideListener(t, ctx.World, &Listener{Enabled: true, Range: 100})
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())
	runAudio(t, sys, ctx)
	// Bounds clamp the user camera to x=150; rotation makes (0, 50)
	// camera-right. The source is then moved to isolate that axis.
	if err := entity.Update(ctx.World, func(tr *core.Transform) { tr.Position.X = 150 }); err != nil {
		t.Fatal(err)
	}
	runAudio(t, sys, ctx)
	got := fake.syncs[len(fake.syncs)-1].want
	if math.Abs(got.Volume-0.5) > 1e-12 || math.Abs(got.Balance-0.5) > 1e-12 || got.Restart {
		t.Fatalf("camera-follow playback = %+v, want volume 0.5 and right balance 0.5", got)
	}
	if err := camera.Update(ctx.World, func(tr *core.Transform) { tr.Position.X = 100 }); err != nil {
		t.Fatal(err)
	}
	runAudio(t, sys, ctx)
	got = fake.syncs[len(fake.syncs)-1].want
	if math.Abs(got.Volume-(1-math.Sqrt(0.5))) > 1e-12 || math.Abs(got.Balance-0.5) > 1e-12 {
		t.Fatalf("moved camera playback = %+v", got)
	}
}

func TestSpatialErrorKeepsHealthyPlaysSynced(t *testing.T) {
	w := newWorldWithHub(t)
	provideListener(t, w, &Listener{Enabled: true, Mode: FixedPosition, Range: 100})
	if _, err := w.NewEntity(Source{Volume: 1, Spatial: true}, Clip{Audio: "sfx/broken.wav"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.NewEntity(
		Source{Volume: 1, Spatial: true}, Clip{Audio: "sfx/healthy.wav"},
		core.Transform{Position: geom.Vector2{X: 50}},
	); err != nil {
		t.Fatal(err)
	}
	fake := &fakeController{live: true}
	err := NewAudioSystem(fake, NewMixer()).Update(&core.Context{World: w})
	if err == nil || !strings.Contains(err.Error(), "requires a Transform") {
		t.Fatalf("Update() = %v, want Transform error", err)
	}
	if len(fake.syncs) != 1 || fake.syncs[0].source != "sfx/healthy.wav" {
		t.Fatalf("syncs = %+v, want the healthy play only", fake.syncs)
	}
	if got := fake.syncs[0].want; got.Volume != 0.5 || got.Balance != 0.5 {
		t.Fatalf("healthy playback = %+v, want spatialized mix", got)
	}
	if fake.sweeps != 1 {
		t.Fatalf("sweeps = %d, want the pass to complete", fake.sweeps)
	}
}

func TestAttenuationEndpointsValidatedOnce(t *testing.T) {
	endpoints := 0
	curve := func(t float64) float64 {
		if t == 0 || t == 1 {
			endpoints++
		}
		return t
	}
	entity, ctx := newAudioWorld(t, Source{Volume: 1, Spatial: true, Attenuation: curve}, "sound.wav")
	if err := entity.AddComponent(ctx.World, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
		t.Fatal(err)
	}
	provideListener(t, ctx.World, &Listener{Enabled: true, Mode: FixedPosition, Range: 100})
	sys := NewAudioSystem(&fakeController{live: true}, NewMixer())
	runAudio(t, sys, ctx)
	runAudio(t, sys, ctx)
	if endpoints != 2 {
		t.Fatalf("endpoint calls = %d, want 2 (one validation) across two updates", endpoints)
	}
}

func TestSpatialErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		listener   *Listener
		transform  *core.Transform
		curve      geom.Ease
		provide    bool
		noViewport bool
		want       string
	}{
		{name: "missing listener", want: "resolve listener"},
		{name: "nil listener", provide: true, want: "listener must not be nil"},
		{name: "zero range", listener: &Listener{Enabled: true, Mode: FixedPosition}, want: "range"},
		{name: "infinite range", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: math.Inf(1)}, want: "range"},
		{name: "nan range", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: math.NaN()}, want: "range"},
		{name: "invalid mode", listener: &Listener{Enabled: true, Mode: ListenerMode(9), Range: 100}, want: "mode"},
		{name: "invalid listener position", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100, Position: geom.Vector2{X: math.NaN()}}, want: "listener position"},
		{name: "missing viewport", listener: &Listener{Enabled: true, Range: 100}, noViewport: true, want: "viewport"},
		{name: "missing camera", listener: &Listener{Enabled: true, Range: 100}, want: "primary camera"},
		{name: "missing transform", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, want: "Transform"},
		{name: "invalid source position", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, transform: &core.Transform{Position: geom.Vector2{Y: math.Inf(1)}}, want: "position"},
		{name: "invalid curve endpoint", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, transform: &core.Transform{}, curve: func(float64) float64 { return 0 }, want: "attenuation curve"},
		{name: "invalid curve result", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, transform: &core.Transform{Position: geom.Vector2{X: 50}}, curve: func(t float64) float64 {
			if t == 0.5 {
				return math.NaN()
			}
			return t
		}, want: "attenuation curve"},
		{name: "curve above unity", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, transform: &core.Transform{Position: geom.Vector2{X: 50}}, curve: func(t float64) float64 {
			if t == 0.5 {
				return 2
			}
			return t
		}, want: "attenuation curve"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entity, ctx := newAudioWorld(t, Source{Volume: 1, Spatial: true, Attenuation: tc.curve}, "sound.wav")
			if !tc.noViewport {
				ctx.LogicalWidth, ctx.LogicalHeight = 100, 100
			}
			if tc.listener != nil || tc.provide {
				provideListener(t, ctx.World, tc.listener)
			}
			if tc.transform != nil {
				if err := entity.AddComponent(ctx.World, *tc.transform); err != nil {
					t.Fatal(err)
				}
			}
			err := NewAudioSystem(&fakeController{live: true}, NewMixer()).Update(ctx)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Update() = %v, want %q error", err, tc.want)
			}
		})
	}
}
