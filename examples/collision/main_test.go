package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/geom"
)

func collisionGame(t *testing.T, stats playerStats) (*castrum.Game, *core.Entity) {
	t.Helper()
	g, err := castrum.New(castrum.WithFixedTPS(100))
	if err != nil {
		t.Fatal(err)
	}
	collider, err := collision.NewCollider(collision.Circle{Radius: playerRadius})
	if err != nil {
		t.Fatal(err)
	}
	player, err := g.World().NewEntity(core.Transform{}, collider, stats)
	if err != nil {
		t.Fatal(err)
	}
	return g, player
}

func advanceCollision(t *testing.T, g *castrum.Game) {
	t.Helper()
	if err := g.Advance(10 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestHazardsDamageOnlyOnEntry(t *testing.T) {
	for _, initial := range []playerStats{
		newPlayerStats(maxHealth, moveSpeed),
		{maxHealth: maxHealth, currentHealth: 3, maxSpeed: moveSpeed, currentSpeed: minMoveSpeed + 2},
	} {
		g, player := collisionGame(t, initial)
		for _, layers := range []uint32{hazardLayers, collision.Layers(playerLayer)} {
			if _, err := spawnRect(g.World(), geom.Vector2{}, geom.Vector2{X: -10, Y: -10},
				geom.Vector2{X: 10, Y: 10}, barColor, layers); err != nil {
				t.Fatal(err)
			}
		}
		trigger, err := collision.NewCollider(collision.Circle{Radius: 10})
		if err != nil {
			t.Fatal(err)
		}
		trigger.Layers = hazardLayers
		trigger.Trigger = true
		if _, err := g.World().NewEntity(core.Transform{}, trigger); err != nil {
			t.Fatal(err)
		}
		if err := g.AddSystem(core.PhaseFixed, "hazard", hazardSystem(player, debug.NewPanel())); err != nil {
			t.Fatal(err)
		}
		if err := g.Startup(); err != nil {
			t.Fatal(err)
		}
		want := initial
		for tick, position := range []geom.Vector2{{}, {}, {X: 1000}, {}} {
			if err := player.Update(g.World(), func(transform *core.Transform) {
				transform.Position = position
			}); err != nil {
				t.Fatal(err)
			}
			advanceCollision(t, g)
			if tick == 0 || tick == 3 {
				want.currentHealth = max(want.currentHealth-hitHealthLoss, 0)
				want.currentSpeed = max(want.currentSpeed-hitSpeedLoss, minMoveSpeed)
			}
			got, _ := player.Component[playerStats](g.World())
			if got != want {
				t.Fatalf("tick %d: stats=%+v, want %+v; only solid hazard entries should damage", tick+1, got, want)
			}
		}
	}
}

func TestPickupsHealAndScoreOnlyOnEntry(t *testing.T) {
	initial := newPlayerStats(maxHealth, moveSpeed)
	initial.currentHealth = maxHealth - 1
	g, player := collisionGame(t, initial)
	collider, err := collision.NewCollider(collision.Circle{Radius: pickupRadius})
	if err != nil {
		t.Fatal(err)
	}
	collider.Layers = collision.Layers(pickupLayer)
	collider.Mask = collision.Mask(playerLayer)
	collider.Trigger = true
	pickup, err := g.World().NewEntity(core.Transform{}, collider)
	if err != nil {
		t.Fatal(err)
	}
	score := 0
	if err := g.AddSystem(core.PhaseFixed, "pickups",
		pickupSystem(player, []*core.Entity{pickup}, &score, debug.NewPanel())); err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	for tick, step := range []struct {
		position geom.Vector2
		score    int
	}{
		{geom.Vector2{}, 1},
		{geom.Vector2{}, 1},
		{geom.Vector2{X: 1000}, 1},
		{geom.Vector2{}, 2},
	} {
		if err := pickup.SetComponent(g.World(), core.Transform{Position: step.position}); err != nil {
			t.Fatal(err)
		}
		advanceCollision(t, g)
		got, _ := player.Component[playerStats](g.World())
		if score != step.score || got.currentHealth != maxHealth || got.currentSpeed != initial.currentSpeed {
			t.Fatalf("tick %d: score=%d stats=%+v, want score=%d and health=%d", tick+1, score, got, step.score, maxHealth)
		}
		if tick == 0 || tick == 3 {
			transform, _ := pickup.Component[core.Transform](g.World())
			if transform.Position == step.position {
				t.Fatal("collected pickup did not respawn")
			}
		}
	}
}
