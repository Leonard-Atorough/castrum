package asset

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCodecSnapshotsAndLocalRegistration(t *testing.T) {
	codecs := NewCodecs()
	if err := setTestJSON[spriteMeta](codecs); err != nil {
		t.Fatal(err)
	}
	oldLoader := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	oldSaver, err := NewSaver(nil, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	if err := codecs.SetDecoder(FormatJSON, func(io.Reader) (spriteMeta, error) {
		return spriteMeta{Name: "new decoder"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := codecs.SetEncoder(FormatJSON, func(w io.Writer, _ spriteMeta) error {
		_, err := io.WriteString(w, "new encoder")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	newLoader := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	newSaver, err := NewSaver(nil, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		loader *Loader
		saver  *Saver
		name   string
		output string
	}{
		{oldLoader, oldSaver, "original", "{\"name\":\"original\"}\n"},
		{newLoader, newSaver, "new decoder", "new encoder"},
	} {
		got, err := tc.loader.LoadReader[spriteMeta](strings.NewReader(`{"name":"original"}`), FormatJSON)
		if err != nil || got.Name != tc.name {
			t.Fatalf("snapshot decode = %+v, %v; want %q", got, err, tc.name)
		}
		var output bytes.Buffer
		if err := tc.saver.SaveWriter(&output, spriteMeta{Name: "original"}, FormatJSON); err != nil {
			t.Fatal(err)
		}
		if output.String() != tc.output {
			t.Fatalf("snapshot encode = %q, want %q", output.String(), tc.output)
		}
	}

	local := Format("local")
	if err := oldLoader.RegisterDecoder(local, jsonDecoder[spriteMeta]()); err != nil {
		t.Fatal(err)
	}
	if err := oldSaver.RegisterEncoder(local, func(w io.Writer, v spriteMeta) error {
		return json.NewEncoder(w).Encode(v)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := newLoader.LoadReader[spriteMeta](strings.NewReader(`{}`), local); err == nil {
		t.Fatal("local decoder escaped into a sibling loader")
	}
	var output bytes.Buffer
	if err := newSaver.SaveWriter(&output, spriteMeta{}, local); err == nil {
		t.Fatal("local encoder escaped into a sibling saver")
	}
	if _, err := codecs.decoder[spriteMeta](local); err == nil {
		t.Fatal("local decoder escaped into configuration")
	}
	if _, err := codecs.encoder[spriteMeta](local); err == nil {
		t.Fatal("local encoder escaped into configuration")
	}
}

func TestCodecConfigurationErrors(t *testing.T) {
	codecs := &Codecs{}
	encoder := Encoder[spriteMeta](func(io.Writer, spriteMeta) error { return nil })
	decoder := jsonDecoder[spriteMeta]()
	for _, err := range []error{
		codecs.SetEncoder(FormatJSON, Encoder[spriteMeta](nil)),
		codecs.SetDecoder(FormatJSON, Decoder[spriteMeta](nil)),
		codecs.SetEncoder("", encoder),
		codecs.SetDecoder("", decoder),
		(*Codecs)(nil).SetEncoder(FormatJSON, encoder),
		(*Codecs)(nil).SetDecoder(FormatJSON, decoder),
	} {
		if err == nil {
			t.Fatal("invalid codec registration succeeded")
		}
	}
	if err := codecs.SetEncoder(FormatJSON, encoder); err != nil {
		t.Fatal(err)
	}
	if err := codecs.SetEncoder(FormatJSON, encoder); err != nil {
		t.Fatalf("setting an existing encoder: %v", err)
	}
}

func TestFacadesRejectInvalidDependencies(t *testing.T) {
	if _, err := NewLoader((*memoryStore)(nil), LoaderOptions{}); err == nil {
		t.Fatal("nil pointer filesystem accepted")
	}
	if _, err := NewSaver((*memoryStore)(nil), SaverOptions{}); err == nil {
		t.Fatal("typed nil store accepted")
	}
	loader := newTestLoader(t, nil, LoaderOptions{})
	for _, reader := range []io.Reader{nil, (*bytes.Reader)(nil)} {
		if _, err := loader.LoadReader[AtlasMeta](reader, FormatJSON); err == nil {
			t.Fatal("nil reader accepted")
		}
	}
}

func TestLoaderAcceptsNilMapFilesystem(t *testing.T) {
	loader, err := NewLoader(fstest.MapFS(nil), LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loader.Load[AtlasMeta]("missing.json"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("empty filesystem load = %v, want fs.ErrNotExist", err)
	}
}

type emptyStream []byte

func (emptyStream) Read([]byte) (int, error)    { return 0, io.EOF }
func (emptyStream) Write(p []byte) (int, error) { return len(p), nil }

type unavailableStore map[string]struct{}

func (unavailableStore) Write(string, func(io.Writer) error) error { return fs.ErrPermission }

func TestFacadesAcceptNilMapAndSliceImplementations(t *testing.T) {
	codecs := NewCodecs()
	if err := setTestJSON[AtlasMeta](codecs); err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	if _, err := loader.LoadReader[AtlasMeta](emptyStream(nil), FormatJSON); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream decode = %v, want io.EOF", err)
	}
	saver, err := NewSaver(unavailableStore(nil), SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	if err := saver.SaveWriter(emptyStream(nil), AtlasMeta{}, FormatJSON); err != nil {
		t.Fatalf("nil slice writer: %v", err)
	}
	if err := saver.Save("atlas.json", AtlasMeta{}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("nil map store = %v, want fs.ErrPermission", err)
	}
}

func setTestJSON[T any](codecs *Codecs) error {
	if err := codecs.SetDecoder(FormatJSON, jsonDecoder[T]()); err != nil {
		return err
	}
	return codecs.SetEncoder(FormatJSON, func(w io.Writer, value T) error {
		return json.NewEncoder(w).Encode(value)
	})
}
