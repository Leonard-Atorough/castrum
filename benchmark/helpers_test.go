// Package benchmark holds castrum's engine baselines: the ECS,
// scheduler, asset, render, and collision hot paths. It is a
// separate module, so the engine's vet, test, and coverage runs
// never compile it; run it with:
//
//	go test -bench=. -benchmem -run='^$' ./...
//
// Benchmarks follow the community go-ecs-benchmark and Bevy
// bevy_ecs suites in operation and reporting - ns/op, B/op, and
// allocs/op with sizes 100 / 1000 / 10000 - so numbers line up
// against other libraries.
package benchmark

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	"golang.org/x/image/font/gofont/goregular"

	"testing/fstest"
)

// benchTag is a marker component for query benchmarks.
type benchTag struct{}

// Sinks defeat dead-code elimination across benchmarks.
var (
	sinkI   int
	sinkF   float64
	sinkEnt core.EntityID
)

// benchPNG encodes a w x h PNG in memory.
func benchPNG(b *testing.B, w, h int) []byte {
	b.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		b.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// benchWAV builds a small 16-bit stereo WAV at 44100 Hz.
func benchWAV() []byte {
	const sampleRate = 44100
	pcm := make([]byte, 256*4)
	for i := range pcm {
		pcm[i] = byte(i)
	}
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+len(pcm)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*4))
	binary.Write(&buf, binary.LittleEndian, uint16(4))
	binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(len(pcm)))
	buf.Write(pcm)
	return buf.Bytes()
}

// benchFS builds an in-memory filesystem with the benchmark assets.
func benchFS(b *testing.B) fstest.MapFS {
	b.Helper()
	return fstest.MapFS{
		"tex.png":   {Data: benchPNG(b, 256, 256)},
		"font.ttf":  {Data: goregular.TTF},
		"tone.wav":  {Data: benchWAV()},
		"tiles.png": {Data: benchPNG(b, 512, 512)},
	}
}

// drawCtx builds the context Collect runs under.
func drawCtx(world *core.World, alpha float64) *core.Context {
	return &core.Context{
		World:         world,
		Tick:          1,
		DeltaTime:     time.Second / 60,
		Alpha:         alpha,
		LogicalWidth:  1280,
		LogicalHeight: 720,
	}
}

// spriteWorld builds a world whose n sprites fill the logical
// viewport, with an asset server and a primary camera in place.
// shape selects a world of circle shapes; textures resolve a
// preloaded texture. spread expands the field beyond the viewport
// so 1 - viewportFraction of the sprites sit off-screen.
func spriteWorld(b *testing.B, n int, shape bool, viewportFraction float64) (*core.World, *render.Collector) {
	b.Helper()
	world := core.NewWorld()
	server := asset.New(benchFS(b))
	if err := world.Provide(func(*core.World) (*asset.Server, error) {
		return server, nil
	}); err != nil {
		b.Fatal(err)
	}
	if _, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{X: 640, Y: 360}},
		render.Camera{Primary: true, Zoom: 1},
	); err != nil {
		b.Fatal(err)
	}
	for i := range n {
		x := float64(i%64) * 20
		y := float64(i/64) * 20
		if viewportFraction < 1 {
			// Spread the population over a field larger than the
			// viewport; only the requested fraction lands inside it.
			x = (float64(i%160) / 160) * (1280 / viewportFraction)
			y = (float64(i/160) / 160) * (720 / viewportFraction)
		}
		sprite := render.Sprite{Layer: uint8(i % 3), SortOrder: int8(i % 5)}
		if shape {
			sprite.Drawable = render.CircleShape{Radii: geom.Vector2{X: 16, Y: 16}}
		} else {
			sprite.Drawable = render.TextureSource{Texture: "tex.png"}
		}
		if _, err := world.NewEntity(
			core.Transform{Position: geom.Vector2{X: x, Y: y}},
			sprite,
		); err != nil {
			b.Fatal(err)
		}
	}
	if !shape {
		if _, err := server.Load[asset.TextureData]("tex.png"); err != nil {
			b.Fatalf("preload tex.png: %v", err)
		}
	}
	return world, render.NewCollector(world)
}
