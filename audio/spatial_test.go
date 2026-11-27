package audio

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

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
			settings := Settings{Volume: 0.8, Bus: "sfx", Spatial: true, Balance: tc.bias, Attenuation: tc.curve}
			h := newAudioHarness(t)
			entity := h.spawn(Track("sound.wav"), settings)
			if err := entity.AddComponent(h.world, core.Transform{Position: tc.position}); err != nil {
				t.Fatal(err)
			}
			*h.listener = Listener{Enabled: true, Mode: FixedPosition, Range: 100}
			if err := h.mixer.SetMasterVolume(0.5); err != nil {
				t.Fatal(err)
			}
			if err := h.mixer.SetBusVolume(BusName("sfx"), 0.5); err != nil {
				t.Fatal(err)
			}
			h.update()
			got := h.lastSync(entity.ID()).want
			if math.Abs(got.Volume-0.2*tc.wantGain) > 1e-12 || math.Abs(got.Balance-tc.wantBalance) > 1e-12 || !got.Playing {
				t.Fatalf("playback = %+v, want volume %v, balance %v, playing", got, 0.2*tc.wantGain, tc.wantBalance)
			}
			stored := h.player(entity)
			if stored.Settings.Volume != settings.Volume || stored.Settings.Balance != settings.Balance {
				t.Fatalf("spatialization modified source: %+v", stored)
			}
		})
	}
}

func TestSpatialLiveUpdatesAndPause(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Balance: -0.25, Spatial: true})
	if err := entity.AddComponent(h.world, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
		t.Fatal(err)
	}
	*h.listener = Listener{Enabled: true, Mode: FixedPosition, Range: 100}
	h.update()

	h.mixer.PauseAll()
	if err := entity.Update(h.world, func(tr *core.Transform) { tr.Position.X = -50 }); err != nil {
		t.Fatal(err)
	}
	h.update()
	got := h.lastSync(entity.ID()).want
	if got.Playing || got.Restart || got.Volume != 0.5 || got.Balance != -0.75 {
		t.Fatalf("paused movement = %+v", got)
	}
	h.mixer.ResumeAll()
	h.listener.Position.X = -25
	h.update()
	got = h.lastSync(entity.ID()).want
	if !got.Playing || got.Restart || got.Volume != 0.75 || got.Balance != -0.5 {
		t.Fatalf("moved listener = %+v", got)
	}
	if err := entity.Update(h.world, func(s *Player) { s.Settings.Attenuation = geom.EaseInQuad }); err != nil {
		t.Fatal(err)
	}
	h.update()
	got = h.lastSync(entity.ID()).want
	if got.Restart || got.Volume != 0.9375 || got.Balance != -0.5 {
		t.Fatalf("updated curve = %+v", got)
	}
	h.listener.Enabled = false
	h.update()
	got = h.lastSync(entity.ID()).want
	if !got.Playing || got.Restart || got.Volume != 1 || got.Balance != -0.25 {
		t.Fatalf("disabled listener = %+v", got)
	}
	h.listener.Enabled = true
	if err := entity.Update(h.world, func(s *Player) { s.Settings.Spatial = false }); err != nil {
		t.Fatal(err)
	}
	if err := entity.RemoveComponent[core.Transform](h.world); err != nil {
		t.Fatal(err)
	}
	h.update()
	got = h.lastSync(entity.ID()).want
	if got.Restart || got.Volume != 1 || got.Balance != -0.25 {
		t.Fatalf("disabled spatial source = %+v", got)
	}
	if err := entity.RemoveComponent[Player](h.world); err != nil {
		t.Fatal(err)
	}
	before := len(h.controller.syncs)
	h.update()
	if len(h.controller.syncs) != before || h.controller.sweeps != 7 {
		t.Fatalf("removed source: syncs = %d, sweeps = %d", len(h.controller.syncs), h.controller.sweeps)
	}
}

func provideWindow(t *testing.T, world *core.World, size int) {
	t.Helper()
	err := world.Provide(func(*core.World) (render.Window, error) {
		return render.NewStaticWindow(render.Viewport{LogicalWidth: size, LogicalHeight: size, WindowWidth: size, WindowHeight: size}), nil
	})
	if err != nil {
		t.Fatalf("provide window: %v", err)
	}
}

func TestDisabledListenerDoesNotRequireTransform(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true, Balance: 0.5})
	h.update()
	if got := h.lastSync(entity.ID()).want; got.Volume != 1 || got.Balance != 0.5 {
		t.Fatalf("disabled listener playback = %+v", got)
	}
}

func TestSpatialCameraFollow(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true})
	provideWindow(t, h.world, 100)
	if err := entity.AddComponent(h.world, core.Transform{Position: geom.Vector2{X: 100, Y: 150}}); err != nil {
		t.Fatal(err)
	}
	if _, err := render.SpawnEngineCamera(h.world); err != nil {
		t.Fatal(err)
	}
	camera, err := h.world.NewEntity(
		render.Camera{Primary: true, Bounds: geom.RectFromMinMax(geom.Vector2{}, geom.Vector2{X: 200, Y: 200})},
		core.Transform{Position: geom.Vector2{X: 300, Y: 100}, Rotation: math.Pi / 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	*h.listener = Listener{Enabled: true, Range: 100}
	h.update()
	// Bounds clamp the user camera to x=150; rotation makes (0, 50)
	// camera-right. The source is then moved to isolate that axis.
	if err := entity.Update(h.world, func(tr *core.Transform) { tr.Position.X = 150 }); err != nil {
		t.Fatal(err)
	}
	h.update()
	got := h.lastSync(entity.ID()).want
	if math.Abs(got.Volume-0.5) > 1e-12 || math.Abs(got.Balance-0.5) > 1e-12 || got.Restart {
		t.Fatalf("camera-follow playback = %+v, want volume 0.5 and right balance 0.5", got)
	}
	if err := camera.Update(h.world, func(tr *core.Transform) { tr.Position.X = 100 }); err != nil {
		t.Fatal(err)
	}
	h.update()
	got = h.lastSync(entity.ID()).want
	if math.Abs(got.Volume-(1-math.Sqrt(0.5))) > 1e-12 || math.Abs(got.Balance-0.5) > 1e-12 {
		t.Fatalf("moved camera playback = %+v", got)
	}
}

func TestSpatialErrorKeepsHealthyPlaysSynced(t *testing.T) {
	h := newAudioHarness(t)
	*h.listener = Listener{Enabled: true, Mode: FixedPosition, Range: 100}
	broken := h.spawn(Track("sfx/broken.wav"), Settings{Volume: 1, Spatial: true})
	healthy := h.spawn(Track("sfx/healthy.wav"), Settings{Volume: 1, Spatial: true})
	removed := h.spawn(Track("sfx/removed.wav"), DefaultSettings())
	if err := healthy.AddComponent(h.world, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
		t.Fatal(err)
	}

	err := h.step()
	if err == nil || !strings.Contains(err.Error(), "requires a Transform") {
		t.Fatalf("Update() = %v, want Transform error", err)
	}
	call := h.lastSync(healthy.ID())
	if call.source != "sfx/healthy.wav" {
		t.Fatalf("source = %q, want sfx/healthy.wav", call.source)
	}
	if got := call.want; got.Volume != 0.5 || got.Balance != 0.5 {
		t.Fatalf("healthy playback = %+v, want spatialized mix", got)
	}
	if got := h.lastSync(broken.ID()).want; got.Playing || got.Restart {
		t.Fatalf("broken playback = %+v, want a hold without restart", got)
	}
	if h.controller.sweeps != 1 {
		t.Fatalf("sweeps = %d, want one after a fully synced pass", h.controller.sweeps)
	}

	if err := h.world.DestroyEntity(removed); err != nil {
		t.Fatal(err)
	}
	sweepErr := errors.New("sweep failed")
	h.controller.sweepErr = sweepErr
	err = h.step()
	if err == nil || !strings.Contains(err.Error(), "requires a Transform") || !errors.Is(err, sweepErr) {
		t.Fatalf("Update() = %v, want spatial and sweep errors", err)
	}
	if h.controller.sweeps != 2 || len(h.controller.syncs)-h.syncStart != 2 {
		t.Fatal("removed player was synced or error pass did not sweep")
	}
	h.controller.sweepErr = nil
	if err := broken.AddComponent(h.world, core.Transform{}); err != nil {
		t.Fatal(err)
	}
	h.update()
	if got := h.lastSync(healthy.ID()).want; got.Restart {
		t.Fatalf("healthy playback restarted after recovery: %+v", got)
	}
	if got := h.lastSync(broken.ID()).want; !got.Playing || !got.Restart {
		t.Fatalf("broken playback = %+v, want a first sync after recovery", got)
	}
	if h.controller.sweeps != 3 {
		t.Fatalf("sweeps = %d, want three after recovery", h.controller.sweeps)
	}
}

func TestSpatialErrorHoldsUntilRecovery(t *testing.T) {
	h := newAudioHarness(t)
	*h.listener = Listener{Enabled: true, Mode: FixedPosition, Range: 100}
	entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true})
	if err := entity.AddComponent(h.world, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
		t.Fatal(err)
	}
	h.update()

	h.listener.Range = 0
	if err := h.step(); err == nil {
		t.Fatal("Update() = nil, want range error")
	}
	h.listener.Range = 100
	h.update()
	if got := h.lastSync(entity.ID()).want; !got.Playing || got.Restart {
		t.Fatalf("recovered playback = %+v, want a resume without restart", got)
	}

	h.listener.Range = 0
	if err := entity.Update(h.world, func(p *Player) { p.Restart() }); err != nil {
		t.Fatal(err)
	}
	if err := h.step(); err == nil {
		t.Fatal("Update() = nil, want range error")
	}
	h.listener.Range = 100
	h.update()
	if got := h.lastSync(entity.ID()).want; !got.Playing || !got.Restart {
		t.Fatalf("recovered playback = %+v, want the pending restart", got)
	}
}

func TestAttenuationEndpointsRevalidated(t *testing.T) {
	endpoint := 1.0
	curve := func(t float64) float64 {
		if t == 1 {
			return endpoint
		}
		return t
	}
	h := newAudioHarness(t)
	entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true, Attenuation: curve})
	if err := entity.AddComponent(h.world, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
		t.Fatal(err)
	}
	*h.listener = Listener{Enabled: true, Mode: FixedPosition, Range: 100}
	h.update()
	endpoint = 0
	if err := h.step(); err == nil || !strings.Contains(err.Error(), "attenuation curve") {
		t.Fatalf("Update() = %v, want invalid endpoint error", err)
	}
	if got := h.lastSync(entity.ID()).want; got.Playing || got.Restart {
		t.Fatalf("invalid curve playback = %+v, want a hold without restart", got)
	}
	if h.controller.sweeps != 2 {
		t.Fatalf("sweeps = %d, want one per fully synced pass", h.controller.sweeps)
	}
}

func TestAttenuationClosuresValidateCapturedEndpoints(t *testing.T) {
	factory := func(start, end float64) geom.Ease {
		return func(t float64) float64 {
			switch t {
			case 0:
				return start
			case 1:
				return end
			default:
				return t
			}
		}
	}
	for _, tc := range []struct {
		name       string
		start, end float64
		valid      bool
	}{
		{name: "valid", end: 1, valid: true},
		{name: "wrong start", start: 0.25, end: 1},
		{name: "wrong end", end: 0.75},
		{name: "nan endpoint", end: math.NaN()},
		{name: "infinite endpoint", end: math.Inf(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			curve := factory(tc.start, tc.end)
			h := newAudioHarness(t)
			entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true, Attenuation: curve})
			if err := entity.AddComponent(h.world, core.Transform{Position: geom.Vector2{X: 50}}); err != nil {
				t.Fatal(err)
			}
			*h.listener = Listener{Enabled: true, Mode: FixedPosition, Range: 100}
			err := h.step()
			if tc.valid {
				if err != nil || len(h.controller.syncs) != 1 || h.controller.syncs[0].want.Volume != 0.5 {
					t.Fatalf("valid curve: syncs=%+v err=%v", h.controller.syncs, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "attenuation curve") || len(h.controller.syncs) != 1 ||
				h.controller.syncs[0].want.Playing || h.controller.syncs[0].want.Restart {
				t.Fatalf("invalid curve: syncs=%+v err=%v", h.controller.syncs, err)
			}
		})
	}
}

func TestSpatialMissingListener(t *testing.T) {
	h := newAudioHarnessWithWorld(t, core.NewWorld())
	h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true})
	if err := h.step(); err == nil || !strings.Contains(err.Error(), "resolve listener") {
		t.Fatalf("Update() = %v, want missing listener error", err)
	}
}

func TestSpatialNilListener(t *testing.T) {
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*Listener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	h := newAudioHarnessWithWorld(t, world)
	h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true})
	if err := h.step(); err == nil || !strings.Contains(err.Error(), "listener must not be nil") {
		t.Fatalf("Update() = %v, want nil listener error", err)
	}
}

func TestSpatialErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		listener   *Listener
		transform  *core.Transform
		curve      geom.Ease
		noViewport bool
		want       string
	}{
		{name: "zero range", listener: &Listener{Enabled: true, Mode: FixedPosition}, want: "range"},
		{name: "infinite range", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: math.Inf(1)}, want: "range"},
		{name: "nan range", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: math.NaN()}, want: "range"},
		{name: "invalid mode", listener: &Listener{Enabled: true, Mode: ListenerMode(9), Range: 100}, want: "mode"},
		{name: "invalid listener position", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100, Position: geom.Vector2{X: math.NaN()}}, want: "listener position"},
		{name: "missing viewport", listener: &Listener{Enabled: true, Range: 100}, noViewport: true, want: "viewport"},
		{name: "missing camera", listener: &Listener{Enabled: true, Range: 100}, want: "primary camera"},
		{name: "missing transform", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, want: "Transform"},
		{name: "invalid source position", listener: &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, transform: &core.Transform{Position: geom.Vector2{Y: math.Inf(1)}}, want: "position"},
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
			h := newAudioHarness(t)
			*h.listener = *tc.listener
			entity := h.spawn(Track("sound.wav"), Settings{Volume: 1, Spatial: true, Attenuation: tc.curve})
			if !tc.noViewport {
				provideWindow(t, h.world, 100)
			}
			if tc.transform != nil {
				if err := entity.AddComponent(h.world, *tc.transform); err != nil {
					t.Fatal(err)
				}
			}
			err := h.step()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Update() = %v, want %q error", err, tc.want)
			}
		})
	}
}
