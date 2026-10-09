package benchmark

import (
	"fmt"
	"io"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// benchServer builds a server over the benchmark assets.
func benchServer(b *testing.B) *asset.Server {
	b.Helper()
	return asset.New(benchFS(b))
}

// BenchmarkLoadWarmTexture reports the cache-hit load the collector
// pays per texture sprite per frame.
func BenchmarkLoadWarmTexture(b *testing.B) {
	server := benchServer(b)
	if _, err := server.Load[asset.TextureData]("tex.png"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()

	for b.Loop() {
		if _, err := server.Load[asset.TextureData]("tex.png"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadWarmFont reports the cache-hit load for a font.
func BenchmarkLoadWarmFont(b *testing.B) {
	server := benchServer(b)
	if _, err := server.Load[asset.FontData]("font.ttf"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()

	for b.Loop() {
		if _, err := server.Load[asset.FontData]("font.ttf"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadColdPNG reports a texture decode. A unique cache
// identity per iteration defeats the cache, so each load decodes
// the 256x256 PNG from bytes; the stdlib codec dominates this cost.
func BenchmarkLoadColdPNG(b *testing.B) {
	server := benchServer(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := asset.ID(fmt.Sprintf("tex-%d", i))
		if _, err := server.Load[asset.TextureData]("tex.png", asset.WithID(id)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadColdWAV reports an audio decode through the runner's
// wav codec, the eager path's per-track cost. The ebiten codec
// dominates this cost.
func BenchmarkLoadColdWAV(b *testing.B) {
	server := benchServer(b)
	err := server.RegisterDecoder(asset.FormatWAV, func(r io.Reader) (asset.AudioData, error) {
		stream, err := wav.DecodeWithSampleRate(44100, r)
		if err != nil {
			return asset.AudioData{}, err
		}
		pcm, err := io.ReadAll(stream)
		if err != nil {
			return asset.AudioData{}, err
		}
		return asset.AudioData{PCM: pcm, SampleRate: 44100, Length: stream.Length()}, nil
	}, true)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := asset.ID(fmt.Sprintf("tone-%d", i))
		if _, err := server.Load[asset.AudioData]("tone.wav", asset.WithID(id)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFontParse reports one cold font parse - the Go regular
// face's setup cost - through the engine's font decoder.
func BenchmarkFontParse(b *testing.B) {
	server := benchServer(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := asset.ID(fmt.Sprintf("font-%d", i))
		if _, err := server.Load[asset.FontData]("font.ttf", asset.WithID(id)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRegisterGridAtlas reports registering a 32x32-tile grid
// over a 512x512 texture: the setup path's atlas cost.
func BenchmarkRegisterGridAtlas(b *testing.B) {
	server := benchServer(b)
	if _, err := server.Load[asset.TextureData]("tiles.png"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		id := asset.AtlasID(fmt.Sprintf("tiles-%d", i))
		if err := server.RegisterGridAtlas(id, "tiles.png", 16, 16, "tile"); err != nil {
			b.Fatal(err)
		}
	}
}
