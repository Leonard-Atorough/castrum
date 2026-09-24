package asset

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func encodeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buf.Bytes()
}

func TestDecodeTextureReportsDimensions(t *testing.T) {
	// Dimensions deliberately asymmetric to catch swapped width/height.
	tex, err := decodeTexture(bytes.NewReader(encodeTestPNG(t, 4, 3)))
	if err != nil {
		t.Fatalf("decodeTexture: %v", err)
	}
	if tex.Image == nil {
		t.Fatal("decoded image is nil")
	}
	if tex.Width != 4 || tex.Height != 3 {
		t.Fatalf("dimensions = %dx%d, want 4x3", tex.Width, tex.Height)
	}
}

func TestDecodeTextureRejectsGarbage(t *testing.T) {
	if _, err := decodeTexture(bytes.NewReader([]byte("not an image"))); err == nil {
		t.Fatal("decodeTexture on garbage input should error")
	}
}
