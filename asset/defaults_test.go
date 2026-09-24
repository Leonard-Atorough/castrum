package asset

import (
	"io"
	"strings"
	"testing"
)

func TestNewRegistersDefaultDecoders(t *testing.T) {
	// Defaults wired end to end through the public surface: real png
	// decode, real json decode, cache and format inference included.
	pngBytes := encodeTestPNG(t, 4, 3)
	a := New(newTestFS(map[string]string{
		"tex.png":    string(pngBytes),
		"atlas.json": `{"regions":[{"name":"player","x":0,"y":0,"w":4,"h":3}]}`,
	}))

	tex, err := a.Load[TextureData]("tex.png")
	if err != nil {
		t.Fatalf("Load TextureData: %v", err)
	}
	if tex.Width != 4 || tex.Height != 3 {
		t.Fatalf("dimensions = %dx%d, want 4x3", tex.Width, tex.Height)
	}

	meta, err := a.Load[AtlasMeta]("atlas.json")
	if err != nil {
		t.Fatalf("Load AtlasMeta: %v", err)
	}
	if len(meta.Regions) != 1 || meta.Regions[0].Name != "player" {
		t.Fatalf("meta = %+v, want one region named player", meta)
	}
}

func TestDefaultDecodersAnswerToUserRegistration(t *testing.T) {
	a := New(nil)

	// A user duplicating a default gets an error — the Must-panic is
	// reserved for the engine's own static table.
	if err := a.RegisterDecoder(FormatPNG, decodeTexture, false); err == nil {
		t.Error("duplicate registration of a default should error")
	}

	// Override replaces the default, observable through decoding.
	if err := a.RegisterDecoder(FormatPNG, Decoder[TextureData](func(io.Reader) (TextureData, error) {
		return TextureData{Width: 999, Height: 999}, nil
	}), true); err != nil {
		t.Fatalf("override registration: %v", err)
	}
	tex, err := a.LoadReader[TextureData](strings.NewReader(""), FormatPNG)
	if err != nil {
		t.Fatalf("LoadReader after override: %v", err)
	}
	if tex.Width != 999 {
		t.Fatalf("decoder after override = %+v, want the replacement", tex)
	}
}
