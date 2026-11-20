package main

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

func TestNewGameBuildsSpectrum(t *testing.T) {
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	// 3 shape rows x 5 variant columns.
	const want = 15
	count := 0
	for e := range core.NewQuery(g.World()).With(sprite.Primitive{}, core.Transform{}).Execute() {
		count++
		p, _ := e.Component[sprite.Primitive]()
		transform, _ := e.Component[core.Transform]()
		if p.Style.Color == nil {
			t.Errorf("entity %d has no color; the row hue should have been assigned", e.ID())
		}
		if transform.Position == (geom.Vector2{}) {
			t.Errorf("entity %d sits at the origin; the grid should have placed it", e.ID())
		}
	}
	if count != want {
		t.Fatalf("primitive entities = %d, want %d", count, want)
	}
	camera, ok := g.MainCamera().Component[core.Transform](g.World())
	if !ok || camera.Position != (geom.Vector2{X: screenW / 2, Y: screenH / 2}) {
		t.Fatalf("camera position = %v (ok=%t), want the screen center", camera.Position, ok)
	}
}

func TestSpinAdvancesRotationOnFixedTicks(t *testing.T) {
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	rotationSum := func() float64 {
		total := 0.0
		for e := range core.NewQuery(g.World()).With(sprite.Primitive{}, core.Transform{}).Execute() {
			transform, _ := e.Component[core.Transform]()
			total += transform.Rotation
		}
		return total
	}
	before := rotationSum()
	g.Context().Input = nil
	const ticks = 10
	for range ticks {
		if err := g.Advance(g.Options().FixedDT()); err != nil {
			t.Fatal(err)
		}
	}
	// One spinner per row: 3 entities advance rotationSpeed per tick.
	want := before + 3*rotationSpeed*ticks
	if after := rotationSum(); after < want-1e-9 || after > want+1e-9 {
		t.Fatalf("rotation sum after %d ticks = %v, want %v", ticks, after, want)
	}
}
