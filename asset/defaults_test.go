package asset

import (
	"io"
	"strings"
	"testing"
)

func TestNewRegistersDefaultDecoders(t *testing.T) {
	pngBytes := encodeTestPNG(t, 4, 3)
	a := newTestLoader(t, newTestFS(map[string]string{
		"tex.png": string(pngBytes),
	}), LoaderOptions{})

	tex, err := a.Load[TextureData]("tex.png")
	if err != nil {
		t.Fatalf("Load TextureData: %v", err)
	}
	if tex.Width != 4 || tex.Height != 3 {
		t.Fatalf("dimensions = %dx%d, want 4x3", tex.Width, tex.Height)
	}

}

func TestDefaultDecodersAnswerToUserRegistration(t *testing.T) {
	codecs := NewCodecs()
	if err := codecs.SetDecoder(FormatPNG, Decoder[TextureData](func(io.Reader) (TextureData, error) {
		return TextureData{Width: 999, Height: 999}, nil
	})); err != nil {
		t.Fatalf("replace configured decoder: %v", err)
	}
	a := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})

	if err := a.RegisterDecoder(FormatPNG, decodeTexture); err == nil {
		t.Error("duplicate registration of a default should error")
	}

	tex, err := a.LoadReader[TextureData](strings.NewReader(""), FormatPNG)
	if err != nil {
		t.Fatalf("LoadReader after override: %v", err)
	}
	if tex.Width != 999 {
		t.Fatalf("decoder after override = %+v, want the replacement", tex)
	}
}
