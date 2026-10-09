package asset

import (
	"math"
	"testing"
	"testing/fstest"

	"golang.org/x/image/font/gofont/goregular"
)

func fontFS() fstest.MapFS {
	return fstest.MapFS{"fonts/goregular.ttf": &fstest.MapFile{Data: goregular.TTF}}
}

func TestFontDecode(t *testing.T) {
	server := New(fontFS())

	font, err := server.Load[FontData]("fonts/goregular.ttf")
	if err != nil {
		t.Fatalf("font decode: %v", err)
	}
	if len(font.Raw) == 0 {
		t.Error("decoded font carries no raw bytes for the runner")
	}

	// The same asset under the same identity is served from cache.
	again, err := server.Load[FontData]("fonts/goregular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	if &again == &font {
		t.Error("expected independent copies from the cache")
	}
}

func TestFontDecodeRejectsCorruptBytes(t *testing.T) {
	server := New(fstest.MapFS{"fonts/broken.ttf": &fstest.MapFile{Data: []byte("this is not a font")}})
	if _, err := server.Load[FontData]("fonts/broken.ttf"); err == nil {
		t.Error("corrupt font bytes should fail to decode")
	}
}

func TestFontMeasure(t *testing.T) {
	server := New(fontFS())
	font, err := server.Load[FontData]("fonts/goregular.ttf")
	if err != nil {
		t.Fatal(err)
	}

	width, height := font.Measure(24, "hello, castrum")
	if width <= 0 || height <= 0 {
		t.Fatalf("Measure(24, sample) = %v x %v, want positive", width, height)
	}

	if w, h := font.Measure(24, ""); w != 0 || h != height {
		t.Errorf("Measure of empty text = %v x %v, want zero width, full height", w, h)
	}

	doubleWidth, doubleHeight := font.Measure(48, "hello, castrum")
	if doubleWidth <= width || doubleHeight <= height {
		t.Errorf("doubling the size should grow the measure: %v x %v vs %v x %v",
			doubleWidth, doubleHeight, width, height)
	}

	againWidth, _ := font.Measure(24, "hello, castrum")
	if math.Abs(againWidth-width) > 1e-9 {
		t.Errorf("Measure is not stable: %v then %v", width, againWidth)
	}
}
