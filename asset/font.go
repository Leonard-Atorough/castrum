package asset

import (
	"bytes"
	"io"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// FontData is a decoded font asset. It provides text measurements through
// [FontData.Measure] and retains the original bytes for renderers.
type FontData struct {
	// Raw contains the original font-file bytes for creating renderer-specific faces.
	Raw []byte

	face   *font.Face
	shaper shaping.HarfbuzzShaper
}

func decodeFont(reader io.Reader) (FontData, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return FontData{}, err
	}
	face, err := font.ParseTTF(bytes.NewReader(raw))
	if err != nil {
		return FontData{}, err
	}
	return FontData{Raw: raw, face: face}, nil
}

// Measure returns the shaped width and horizontal line height of text at size,
// in pixels. The width includes shaping effects such as kerning. Empty text has
// zero width and the same height as non-empty text.
//
// Measure is not safe for concurrent use on the same FontData value.
func (f FontData) Measure(size float64, text string) (width, height float64) {
	if text == "" {
		return 0, f.lineHeight(size)
	}
	runes := []rune(text)
	out := f.shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      f.face,
		Size:      fixed.Int26_6(size * (1 << 6)),
	})
	return float64(out.Advance) / 64, f.lineHeight(size)
}

func (f FontData) lineHeight(size float64) float64 {
	extents, ok := f.face.FontHExtents()
	if !ok {
		return size
	}
	scale := size / float64(f.face.Upem())
	return (float64(extents.Ascender) - float64(extents.Descender)) * scale
}
