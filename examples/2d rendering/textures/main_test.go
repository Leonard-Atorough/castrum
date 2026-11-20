package main

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

func TestNewGamePreloadsTextureAndSpawnsGrid(t *testing.T) {
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	// The setup preload resolves the embedded texture and its bounds.
	texture, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData](texturePath)
	if err != nil {
		t.Fatal(err)
	}
	if texture.Width != 288 || texture.Height != 288 {
		t.Fatalf("texture = %dx%d, want 288x288", texture.Width, texture.Height)
	}

	// 2 rows x 5 columns of whole-texture sprites: asset set, region empty.
	const want = 10
	count, flipH, flipV := 0, 0, 0
	for e := range core.NewQuery(g.World()).With(sprite.Sprite{}, core.Transform{}).Execute() {
		count++
		s, _ := e.Component[sprite.Sprite]()
		if s.Asset != texturePath || s.Region != "" {
			t.Errorf("entity %d sprite = %+v, want asset %q with no region", e.ID(), s, texturePath)
		}
		if s.Style.FlipH {
			flipH++
		}
		if s.Style.FlipV {
			flipV++
		}
	}
	if count != want {
		t.Fatalf("sprite entities = %d, want %d", count, want)
	}
	// flipH: single-flip, flip-both, tint+flip cells; flipV: single-flip, flip-both.
	if flipH != 3 || flipV != 2 {
		t.Fatalf("flips: flipH=%d flipV=%d, want 3 and 2", flipH, flipV)
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
		for e := range core.NewQuery(g.World()).With(sprite.Sprite{}, core.Transform{}).Execute() {
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
	// Exactly one cell spins; the rotated cell's 0.6 rad is constant.
	want := before + rotationSpeed*ticks
	if after := rotationSum(); after < want-1e-9 || after > want+1e-9 {
		t.Fatalf("rotation sum after %d ticks = %v, want %v", ticks, after, want)
	}
}
