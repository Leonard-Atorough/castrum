package main

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/timer"
)

// newTestGame builds the game headless: no runner, no window - the
// fixed phase still runs the engine's collision and timer systems,
// so the gameplay wiring is exercised for real. The spawner timer
// is paused so its waves never fire: the tests place their own
// enemies instead.
func newTestGame(t *testing.T) (*castrum.Game, *core.Entity, *core.Entity) {
	t.Helper()
	g, err := castrum.New(castrum.WithTitle("test"), castrum.WithTimer(), castrum.WithCollision())
	if err != nil {
		t.Fatal(err)
	}
	// The runner normally publishes the logical size; the bullet
	// system culls by what the camera can see, so the headless
	// game needs it too.
	g.Context().LogicalWidth = 1280
	g.Context().LogicalHeight = 720
	world := g.World()

	// The hull the enemies chase, carrying its hit circle as the
	// game builds it.
	playerCollider, err := collision.NewCollider(collision.Circle{Radius: TankRadius})
	if err != nil {
		t.Fatal(err)
	}
	playerCollider.Layers = collision.Layers(playerLayer)
	playerCollider.Mask = collision.Mask(enemyLayer)
	hull, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 200, Y: 360}},
		playerCollider,
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

	// The spawner, paused so no wave fires in a test.
	spawner, err := world.NewEntity(
		timer.NewTimer(SpawnEvery, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := spawner.Update(world, func(t *timer.Timer) { t.Running = false }); err != nil {
		t.Fatal(err)
	}

	// The run's state and the presentation sprites, as the game
	// builds them.
	stats, err := world.NewEntity(game{})
	if err != nil {
		t.Fatal(err)
	}
	readout, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 200, Y: 430}, Scale: geom.Vector2{X: 1, Y: 1}},
		render.Sprite{Drawable: render.TextSource{Font: AssetFont, Text: "", Size: 18}},
	)
	if err != nil {
		t.Fatal(err)
	}
	banner, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 200, Y: 360}},
		render.Sprite{Hidden: true, Layer: 10},
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, sys := range []struct {
		name string
		s    core.System
	}{
		{"bullets", bulletsSystem(g.MainCamera(), stats)},
		{"enemy.spawn", spawnSystem(spawner, hull, g.MainCamera(), stats)},
		{"enemies", enemiesSystem(hull, stats)},
		{"game.danger", dangerSystem(hull, stats)},
		{"game.result", resultSystem(banner, hull, stats)},
		{"reload.readout", reloadReadoutSystem(readout, turret, stats)},
	} {
		if err := g.AddSystem(core.PhaseFixed, sys.name, sys.s); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	return g, turret, stats
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
// in the same tick and scores the kill: the collision resolved
// through the layers, and each side's system reads its own state.
func TestBulletAndEnemyDestroyEachOther(t *testing.T) {
	g, _, stats := newTestGame(t)
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
	run, _ := stats.Component[game](world)
	if run.Score != 1 {
		t.Errorf("score = %d, want 1", run.Score)
	}
	if run.Won || run.Lost {
		t.Errorf("won %v lost %v, want the run to continue", run.Won, run.Lost)
	}
}

// An enemy drives toward the tank and faces it as it goes.
func TestEnemyChasesHull(t *testing.T) {
	g, _, _ := newTestGame(t)
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
	g, _, _ := newTestGame(t)
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

// The score reaching WinScore wins the run: the next kill flips
// Won, the gameplay freezes, and the banner reads YOU WIN.
func TestWinScoreEndsRun(t *testing.T) {
	g, _, stats := newTestGame(t)
	world := g.World()

	if err := stats.Update(world, func(s *game) { s.Score = WinScore - 1 }); err != nil {
		t.Fatal(err)
	}
	if _, err := spawnEnemy(world, geom.Vector2{X: 500, Y: 360}); err != nil {
		t.Fatal(err)
	}
	shot, err := spawnBullet(world, geom.Vector2{X: 400, Y: 360}, math.Pi/2)
	if err != nil {
		t.Fatal(err)
	}

	advance(t, g, 120)

	if _, ok := shot.Component[core.Transform](world); ok {
		t.Error("bullet should be destroyed on hitting the enemy")
	}
	run, _ := stats.Component[game](world)
	if !run.Won {
		t.Error("the winning kill should set Won")
	}
	if run.Lost {
		t.Error("a win should not also read as lost")
	}
	banner := findBanner(t, world, "YOU WIN")
	if banner == nil {
		t.Error("the banner should read YOU WIN")
	}
}

// An enemy touching the tank ends the run: the loss flag goes up,
// the enemies freeze, and the banner reads GAME OVER.
func TestTankTouchedEndsRun(t *testing.T) {
	g, _, stats := newTestGame(t)
	world := g.World()

	// Overlapping the hull: the touch is immediate.
	toucher, err := spawnEnemy(world, geom.Vector2{X: 210, Y: 360})
	if err != nil {
		t.Fatal(err)
	}

	advance(t, g, 3)

	run, _ := stats.Component[game](world)
	if !run.Lost {
		t.Error("a touching enemy should set Lost")
	}
	if run.Won {
		t.Error("a loss should not also read as won")
	}

	// The touch must not read as a kill: the enemy survives it and
	// the score stays zero.
	if run.Score != 0 {
		t.Errorf("score = %d, want 0 - a touch is not a kill", run.Score)
	}
	if _, ok := toucher.Component[core.Transform](world); !ok {
		t.Fatal("the touching enemy should survive - only bullets kill")
	}

	// The run is over, so the world freezes: the enemy sits still.
	before, ok := toucher.Component[core.Transform](world)
	if !ok {
		t.Fatal("the touching enemy should still exist after the freeze")
	}
	advance(t, g, 20)
	after, ok := toucher.Component[core.Transform](world)
	if !ok {
		t.Fatal("the touching enemy should still exist after the freeze")
	}
	if before.Position != after.Position {
		t.Errorf("enemy moved after the run ended: %v -> %v", before.Position, after.Position)
	}
	if findBanner(t, world, "GAME OVER") == nil {
		t.Error("the banner should read GAME OVER")
	}
}

// The waves accelerate with the score: the spawn interval shrinks
// by SpawnShave per kill, down to MinSpawnEvery.
func TestSpawnPacingFollowsScore(t *testing.T) {
	g, _, stats := newTestGame(t)
	world := g.World()

	spawner := findSpawner(t, world)
	if err := stats.Update(world, func(s *game) { s.Score = 50 }); err != nil {
		t.Fatal(err)
	}
	advance(t, g, 1)
	sp, _ := spawner.Component[timer.Timer](world)
	if want := SpawnEvery - 50*SpawnShave; sp.Duration != want {
		t.Errorf("interval = %v, want %v", sp.Duration, want)
	}

	// Far past the threshold, the interval clamps to the floor.
	if err := stats.Update(world, func(s *game) { s.Score = 500 }); err != nil {
		t.Fatal(err)
	}
	advance(t, g, 1)
	sp, _ = spawner.Component[timer.Timer](world)
	if sp.Duration != MinSpawnEvery {
		t.Errorf("interval = %v, want the floor %v", sp.Duration, MinSpawnEvery)
	}
}

// The reload readout hides while the gun is ready and shows the
// remaining time while the cooldown runs: the timer's Running field
// is the whole state machine.
func TestReloadReadoutFollowsCooldown(t *testing.T) {
	g, turret, _ := newTestGame(t)
	world := g.World()

	readout := findReadout(t, world)

	advance(t, g, 3)
	// Idle: hidden.
	if sprite, _ := readout.Component[render.Sprite](world); !sprite.Hidden {
		t.Error("idle gun should hide the reload text")
	}

	// Fire: the cooldown restarts and the readout shows the count.
	if err := turret.Update(world, func(t *timer.Timer) { t.Restart() }); err != nil {
		t.Fatal(err)
	}
	advance(t, g, 3)
	sprite, _ := readout.Component[render.Sprite](world)
	if sprite.Hidden {
		t.Fatal("cooling gun should show the reload text")
	}
	text, ok := sprite.Drawable.(render.TextSource)
	if !ok || text.Text == "" {
		t.Errorf("reload text = %+v, want a countdown string", sprite.Drawable)
	}

	// Once the cooldown completes the readout hides again.
	advance(t, g, 120)
	if sprite, _ := readout.Component[render.Sprite](world); !sprite.Hidden {
		t.Error("ready gun should hide the reload text")
	}
}

// findReadout resolves the countdown sprite: the text sprite that
// is not the banner.
func findReadout(t *testing.T, world *core.World) *core.Entity {
	t.Helper()
	query := core.NewQuery(world).With(render.Sprite{}, core.Transform{})
	for e := range query.Execute() {
		sprite, _ := e.Component[render.Sprite]()
		if _, ok := sprite.Drawable.(render.TextSource); ok {
			return core.NewEntity(e.ID())
		}
	}
	t.Fatal("no countdown sprite found")
	return nil
}

// findBanner resolves the end banner and checks its text.
func findBanner(t *testing.T, world *core.World, want string) *core.Entity {
	t.Helper()
	query := core.NewQuery(world).With(render.Sprite{})
	for e := range query.Execute() {
		sprite, _ := e.Component[render.Sprite]()
		if text, ok := sprite.Drawable.(render.TextSource); ok && text.Text == want {
			return core.NewEntity(e.ID())
		}
	}
	return nil
}

// findSpawner resolves the paused spawner entity by its timer.
func findSpawner(t *testing.T, world *core.World) *core.Entity {
	t.Helper()
	query := core.NewQuery(world).With(timer.Timer{})
	for e := range query.Execute() {
		timerData, _ := e.Component[timer.Timer]()
		if timerData.Repeating {
			return core.NewEntity(e.ID())
		}
	}
	t.Fatal("no repeating spawner found")
	return nil
}
