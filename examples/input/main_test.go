package main

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
)

func TestWheelZoomRunsOncePerFrame(t *testing.T) {
	for _, tt := range []struct {
		name    string
		elapsed time.Duration
		paused  bool
		ticks   int
	}{
		{"no fixed ticks", 0, false, 0},
		{"one fixed tick", 10 * time.Millisecond, false, 1},
		{"catch-up ticks", 30 * time.Millisecond, false, 3},
		{"paused simulation", 30 * time.Millisecond, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, err := castrum.New(castrum.WithFixedTPS(100))
			if err != nil {
				t.Fatal(err)
			}

			camera := g.MainCamera()
			if err := camera.Update(g.World(), func(c *core.Transform) {
				c.Scale = geom.Vector2{X: 1, Y: 1}
			}); err != nil {
				t.Fatal(err)
			}
			if err := g.AddSystem(core.PhaseFrame, "zoom", zoomSystem(camera, debug.NewPanel())); err != nil {
				t.Fatal(err)
			}
			ticks := 0
			if err := g.AddSystem(core.PhaseFixed, "counter", core.SystemFunc(func(ctx *core.Context) error {
				ticks++
				return nil
			})); err != nil {
				t.Fatal(err)
			}
			if err := g.Startup(); err != nil {
				t.Fatal(err)
			}
			if tt.paused {
				g.Pause()
			}
			snapshot := &input.Snapshot{}
			snapshot.Mouse.Wheel = geom.Vector2{X: 0.25, Y: 2.5}
			g.Context().Input = snapshot
			if err := g.Advance(tt.elapsed); err != nil {
				t.Fatal(err)
			}
			if ticks != tt.ticks {
				t.Fatalf("fixed ticks = %d, want %d", ticks, tt.ticks)
			}
			want := geom.Vector2{X: 1.25, Y: 1.25}
			if got, ok := camera.Component[core.Transform](g.World()); !ok || got.Scale != want {
				t.Fatalf("zoom scale = %v, want %v after one frame", got.Scale, want)
			}
			g.Resume()
			snapshot.Mouse.Wheel = geom.Vector2{}
			if err := g.Advance(30 * time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if got, ok := camera.Component[core.Transform](g.World()); !ok || got.Scale != want {
				t.Fatalf("zero-scroll update reapplied movement: scale=%v", got.Scale)
			}
		})
	}
}

func TestTankMovementUsesBindingsAndFacing(t *testing.T) {
	for _, tt := range []struct {
		name     string
		keys     []input.Key
		pad      bool
		position geom.Vector2
		rotation float64
	}{
		{"keyboard forward and left", []input.Key{input.KeyW, input.KeyA}, false, geom.Vector2{X: 75}, math.Pi/2 - 0.75},
		{"arrows backward and right", []input.Key{input.KeyArrowDown, input.KeyArrowRight}, false, geom.Vector2{X: -75}, math.Pi/2 + 0.75},
		{"opposing inputs", []input.Key{input.KeyW, input.KeyS, input.KeyA, input.KeyD}, false, geom.Vector2{}, math.Pi / 2},
		{"gamepad forward and right", nil, true, geom.Vector2{X: 75}, math.Pi/2 + 0.75},
	} {
		t.Run(tt.name, func(t *testing.T) {
			world := core.NewWorld()
			body, err := world.NewEntity(core.Transform{Rotation: math.Pi / 2})
			if err != nil {
				t.Fatal(err)
			}
			actions, err := input.New(tankBindings(), 0)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := &input.Snapshot{}
			for _, key := range tt.keys {
				snapshot.Keys[key].Held = true
			}
			if tt.pad {
				snapshot.Pads[0].Connected = true
				snapshot.Pads[0].Axes[input.PadLeftStickY] = -1
				snapshot.Pads[0].Axes[input.PadRightStickX] = 1
			}
			actions.Update(snapshot, 0.25)
			ctx := &core.Context{World: world, Actions: actions, Input: snapshot, DeltaTime: 250 * time.Millisecond}
			if err := tankBodySystem(body).Update(ctx); err != nil {
				t.Fatal(err)
			}
			got, _ := body.Component[core.Transform](world)
			if !got.Position.AlmostEqual(tt.position, 1e-9) || math.Abs(got.Rotation-tt.rotation) > 1e-9 {
				t.Fatalf("transform=%+v, want position=%v rotation=%v", got, tt.position, tt.rotation)
			}
		})
	}
}

func TestTurretUsesPublishedViewAndCameraFollowsHull(t *testing.T) {
	world := core.NewWorld()
	hullPosition := geom.Vector2{X: 110, Y: 50}
	body, err := world.NewEntity(core.Transform{Position: hullPosition})
	if err != nil {
		t.Fatal(err)
	}
	turret, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: -5}, Rotation: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	camera, err := render.SpawnEngineCamera(world)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: world, Input: &input.Snapshot{}, LogicalWidth: 200, LogicalHeight: 100}
	system := turretSystem(turret, body)
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	got, _ := turret.Component[core.Transform](world)
	if got.Position != (geom.Vector2{X: -5}) || got.Rotation != 0.5 {
		t.Fatalf("aiming before a published view changed the turret: %+v", got)
	}
	view := &render.CameraView{
		Position: geom.Vector2{X: 100, Y: 50}, Zoom: 2, Rotation: math.Pi / 2,
	}
	if err := world.Provide(func(*core.World) (*render.CameraView, error) { return view, nil }); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		cursor   geom.Vector2
		rotation float64
	}{
		{geom.Vector2{X: 100, Y: 10}, math.Pi / 2},
		{geom.Vector2{X: 140, Y: 50}, -math.Pi + math.Atan(0.5)},
	} {
		ctx.Input.Mouse.Cursor = tt.cursor
		if err := system.Update(ctx); err != nil {
			t.Fatal(err)
		}
		got, _ = turret.Component[core.Transform](world)
		if got.Position != hullPosition || math.Abs(got.Rotation-tt.rotation) > 1e-9 {
			t.Fatalf("cursor %v: turret=%+v, want mounted at %v with rotation=%v", tt.cursor, got, hullPosition, tt.rotation)
		}
	}
	if err := cameraSystem(camera, body).Update(ctx); err != nil {
		t.Fatal(err)
	}
	transform, _ := camera.Component[core.Transform](world)
	if transform.Position != hullPosition {
		t.Fatalf("camera position=%v, want hull position=%v", transform.Position, hullPosition)
	}
}
