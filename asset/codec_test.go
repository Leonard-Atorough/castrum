package asset

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDecoderSnapshotsAndLocalRegistration(t *testing.T) {
	codecs := NewCodecs()
	if err := codecs.SetDecoder(FormatJSON, jsonDecoder[spriteMeta]()); err != nil {
		t.Fatal(err)
	}
	old := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	if err := codecs.SetDecoder(FormatJSON, func(io.Reader) (spriteMeta, error) {
		return spriteMeta{Name: "new decoder"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	current := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	for _, tc := range []struct {
		loader *Loader
		name   string
	}{{old, "original"}, {current, "new decoder"}} {
		got, err := tc.loader.LoadReader[spriteMeta](strings.NewReader(`{"name":"original"}`), FormatJSON)
		if err != nil || got.Name != tc.name {
			t.Fatalf("snapshot decode = %+v, %v; want %q", got, err, tc.name)
		}
	}
	local := Format("local")
	if err := old.RegisterDecoder(local, jsonDecoder[spriteMeta]()); err != nil {
		t.Fatal(err)
	}
	if _, err := current.LoadReader[spriteMeta](strings.NewReader(`{}`), local); err == nil {
		t.Fatal("local decoder escaped into a sibling loader")
	}
	if _, err := codecs.decoder[spriteMeta](local); err == nil {
		t.Fatal("local decoder escaped into configuration")
	}
}

func TestDecoderConfigurationErrors(t *testing.T) {
	codecs := &Codecs{}
	for _, err := range []error{
		codecs.SetDecoder(FormatJSON, Decoder[spriteMeta](nil)),
		codecs.SetDecoder("", jsonDecoder[spriteMeta]()),
		(*Codecs)(nil).SetDecoder(FormatJSON, jsonDecoder[spriteMeta]()),
	} {
		if err == nil {
			t.Fatal("invalid decoder configuration succeeded")
		}
	}
}

func TestLoaderRejectsNilPointers(t *testing.T) {
	if _, err := NewLoader((*fstest.MapFS)(nil), LoaderOptions{}); err == nil {
		t.Fatal("nil pointer filesystem accepted")
	}
	loader := newTestLoader(t, nil, LoaderOptions{})
	for _, reader := range []io.Reader{nil, (*bytes.Reader)(nil)} {
		if _, err := loader.LoadReader[AtlasMeta](reader, FormatJSON); err == nil {
			t.Fatal("nil reader accepted")
		}
	}
}

func TestLoaderAcceptsNilMapFilesystem(t *testing.T) {
	loader := newTestLoader(t, fstest.MapFS(nil), LoaderOptions{})
	if _, err := loader.Load[AtlasMeta]("missing.json"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("empty filesystem load = %v, want fs.ErrNotExist", err)
	}
}

func TestLoaderPreservesNilInterface(t *testing.T) {
	loader := newTestLoader(t, newTestFS(map[string]string{"value.nil": ""}), LoaderOptions{})
	decodes := 0
	if err := loader.RegisterDecoder(Format("nil"), func(io.Reader) (any, error) {
		decodes++
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if value, err := loader.Load[any]("value.nil"); err != nil || value != nil {
			t.Fatalf("nil interface load = %v, %v", value, err)
		}
	}
	if value, err := loader.LoadReader[any](strings.NewReader(""), Format("nil")); err != nil || value != nil {
		t.Fatalf("nil interface stream decode = %v, %v", value, err)
	}
	if decodes != 2 {
		t.Fatalf("decodes = %d, want one cached and one stream decode", decodes)
	}
}
