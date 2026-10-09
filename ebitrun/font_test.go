package ebitrun

import (
	"math"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/Leonard-Atorough/castrum/asset"
	"golang.org/x/image/font/gofont/goregular"
)

func newFontServer() *asset.Server {
	return asset.New(fstest.MapFS{"fonts/go.ttf": &fstest.MapFile{Data: goregular.TTF}})
}

func TestFontProviderFaceCaches(t *testing.T) {
	provider := newFontProvider(newFontServer())

	face, err := provider.Face("fonts/go.ttf", 16)
	if err != nil {
		t.Fatalf("face: %v", err)
	}
	again, err := provider.Face("fonts/go.ttf", 16)
	if err != nil {
		t.Fatal(err)
	}
	if face != again {
		t.Error("Face should return the same face for the same font and size")
	}

	other, err := provider.Face("fonts/go.ttf", 24)
	if err != nil {
		t.Fatal(err)
	}
	if other == face {
		t.Error("a different size should resolve a different face")
	}
}

func TestFontProviderMissingFontNamesIt(t *testing.T) {
	provider := newFontProvider(newFontServer())
	if _, err := provider.Face("fonts/ghost.ttf", 16); err == nil ||
		!strings.Contains(err.Error(), "fonts/ghost.ttf") {
		t.Errorf("missing font = %v, want an error naming the font", err)
	}
}

// The engine measures text with its own shaper and draws through
// ebitextext; this is the pin that the two agree. If the metrics
// ever diverge, culled bounds and drawn positions drift with them.
func TestMeasureAgreesWithDrawingMetrics(t *testing.T) {
	server := newFontServer()
	provider := newFontProvider(server)
	font, err := server.Load[asset.FontData]("fonts/go.ttf")
	if err != nil {
		t.Fatalf("load font: %v", err)
	}

	for _, sample := range []string{"hello", "score: 123", "castrum!!!"} {
		for _, size := range []float64{12, 16, 32} {
			face, err := provider.Face("fonts/go.ttf", size)
			if err != nil {
				t.Fatal(err)
			}
			drawWidth, drawHeight := text.Measure(sample, face, 0)
			width, height := font.Measure(size, sample)

			if math.Abs(width-drawWidth) > 0.5 {
				t.Errorf("Measure(%q, %v): width %v, drawing width %v", sample, size, width, drawWidth)
			}
			if math.Abs(height-drawHeight) > 0.5 {
				t.Errorf("Measure(%q, %v): height %v, drawing height %v", sample, size, height, drawHeight)
			}
		}
	}
}
