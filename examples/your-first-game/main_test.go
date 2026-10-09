package main

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/timer"
)

// newTestGame builds the game headless: no runner, no window - the
// fixed phase still runs the engine's collision and timer systems,
// so the gameplay wiring is exercised for real. The spawner system
// is left out: its waves draw from the global random source, and
// the tests place their own enemies instead.
func newTestGame(t *testing.T) (*castrum.Game, *core.Entity) {
	t.Helper()
	g, err := castrum.New(castrum.WithTitle("test"))
	if err != nil {
		t.Fatal(err)
	}
	// The runner normally publishes the logical size; the bullet
	// system culls by what the camera can see, so the headless
	// game needs it too.
	g.Context().LogicalWidth = 1280
	g.Context().LogicalHeight = 720
	world := g.World()

	// The hull the enemies chase, as the game builds it.
	hull, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 200, Y: 360}},
	)
	if err != nil {
		t.Fatal(err)
	}

	// A turret with the gun's cooldown timer, idle as at spawn.
	turret, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 200, Y: 360}, Scale: geom.Vector2{X: 1, Y: 1}},
		timer.NewTimer(FireCooldown, false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := turret.Update(world, func(t *timer.Timer) { t.Running = false }); err != nil {
		t.Fatal(err)
	}

	// The reload readout text, as the game builds it.
	readout, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 200, Y: 430}, Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{Drawable: core.TextSource{Font: AssetFont, Text: "", Size: 18}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := g.AddSystem(core.PhaseFixed, "bullets", bulletsSystem(g.MainCamera())); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFixed, "enemies", enemiesSystem(hull)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFixed, "reload.readout", reloadReadoutSystem(readout, turret)); err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	return g, turret
}

// advance steps the fixed phase: one call per ~17ms of game time,
// which is one fixed tick at the default rate most calls.
func advance(t *testing.T, g *castrum.Game, ticks int) {
	t.Helper()
	for i := 0; i < ticks; i++ {
		if err := g.Advance(17 * time.Millisecond); err != nil {
			t.Fatalf("advance: %v", err)
		}
	}
}

// A bullet that reaches an enemy destroys both halves of the pair
// in the same tick: the collision resolved through the layers, and
// each side's system reads its own contacts.
func TestBulletAndEnemyDestroyEachOther(t *testing.T) {
	g, _ := newTestGame(t)
	world := g.World()

	target, err := spawnEnemy(world, geom.Vector2{X: 500, Y: 360})
	if err != nil {
		t.Fatal(err)
	}

	// The bullet flies along +X: facing (0, -1) rotated by pi/2.
	shot, err := spawnBullet(world, geom.Vector2{X: 400, Y: 360}, math.Pi/2)
	if err != nil {
		t.Fatal(err)
	}

	advance(t, g, 120)

	if _, ok := shot.Component[core.Transform](world); ok {
		t.Error("bullet should be destroyed on hitting the enemy")
	}
	if _, ok := target.Component[core.Transform](world); ok {
		t.Error("enemy should be destroyed when the bullet hits it")
	}
}

// An enemy drives toward the tank and faces it as it goes.
func TestEnemyChasesHull(t *testing.T) {
	g, _ := newTestGame(t)
	world := g.World()

	// Due east of the hull, so the chase is a straight line west.
	target, err := spawnEnemy(world, geom.Vector2{X: 600, Y: 360})
	if err != nil {
		t.Fatal(err)
	}

	advance(t, g, 10)

	transform, _ := target.Component[core.Transform](world)
	if transform.Position.X >= 600 {
		t.Errorf("enemy x = %v, want it closing on the hull at x=200", transform.Position.X)
	}
	if math.Abs(transform.Rotation-(-math.Pi/2)) > 1e-9 {
		t.Errorf("enemy rotation = %v, want it facing west at -pi/2", transform.Rotation)
	}
}

// A bullet that misses everything flies past what the camera can
// see and is destroyed there - the live set never grows without
// bound.
func TestBulletDiesOutOfView(t *testing.T) {
	g, _ := newTestGame(t)
	world := g.World()

	// Facing -X: rotation -pi/2 sends (0, -1) to (-1, 0).
	shot, err := spawnBullet(world, geom.Vector2{X: 60, Y: 60}, -math.Pi/2)
	if err != nil {
		t.Fatal(err)
	}

	advance(t, g, 120)

	if _, ok := shot.Component[core.Transform](world); ok {
		t.Error("out-of-view bullet should be destroyed")
	}
}

// The reload readout hides while the gun is ready and shows the
// remaining time while the cooldown runs: the timer's Running field
// is the whole state machine.
func TestReloadReadoutFollowsCooldown(t *testing.T) {
	g, turret := newTestGame(t)
	world := g.World()

	readout := findReadout(t, world)

	advance(t, g, 3)
	// Idle: hidden.
	if sprite, _ := readout.Component[core.Sprite](world); !sprite.Hidden {
		t.Error("idle gun should hide the reload text")
	}

	// Fire: the cooldown restarts and the readout shows the count.
	if err := turret.Update(world, func(t *timer.Timer) { t.Restart() }); err != nil {
		t.Fatal(err)
	}
	advance(t, g, 3)
	sprite, _ := readout.Component[core.Sprite](world)
	if sprite.Hidden {
		t.Fatal("cooling gun should show the reload text")
	}
	text, ok := sprite.Drawable.(core.TextSource)
	if !ok || text.Text == "" {
		t.Errorf("reload text = %+v, want a countdown string", sprite.Drawable)
	}

	// Once the cooldown completes the readout hides again.
	advance(t, g, 120)
	if sprite, _ := readout.Component[core.Sprite](world); !sprite.Hidden {
		t.Error("ready gun should hide the reload text")
	}
}

// findReadout resolves the text sprite the test game spawned; the
// systems close over it, so the test reaches it by its drawable.
func findReadout(t *testing.T, world *core.World) *core.Entity {
	t.Helper()
	query := core.NewQuery(world).With(core.Sprite{})
	for e := range query.Execute() {
		sprite, _ := e.Component[core.Sprite]()
		if _, ok := sprite.Drawable.(core.TextSource); ok {
			return core.NewEntity(e.ID())
		}
	}
	t.Fatal("no readout text sprite found")
	return nil
}
