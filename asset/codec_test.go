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
	if err := codecs.SetJSON[spriteMeta](); err != nil {
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

func TestSetJSONReplacesBothDirections(t *testing.T) {
	for _, existing := range []string{"decoder", "encoder", "both"} {
		t.Run(existing, func(t *testing.T) {
			codecs := &Codecs{}
			if existing != "encoder" {
				if err := codecs.SetDecoder(FormatJSON, func(io.Reader) (spriteMeta, error) {
					return spriteMeta{Name: "custom"}, nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			if existing != "decoder" {
				if err := codecs.SetEncoder(FormatJSON, func(w io.Writer, _ spriteMeta) error {
					_, err := io.WriteString(w, "custom")
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			if err := codecs.SetJSON[spriteMeta](); err != nil {
				t.Fatal(err)
			}
			loader := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
			saver, err := NewSaver(nil, SaverOptions{Codecs: codecs})
			if err != nil {
				t.Fatal(err)
			}
			got, err := loader.LoadReader[spriteMeta](strings.NewReader(`{"name":"standard"}`), FormatJSON)
			if err != nil || got.Name != "standard" {
				t.Fatalf("configured JSON decode = %+v, %v", got, err)
			}
			var output bytes.Buffer
			if err := saver.SaveWriter(&output, got, FormatJSON); err != nil {
				t.Fatal(err)
			}
			if output.String() != "{\"name\":\"standard\"}\n" {
				t.Fatalf("configured JSON encode = %q", output.String())
			}
		})
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
		(*Codecs)(nil).SetJSON[spriteMeta](),
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
	loader := newTestLoader(t, nil, LoaderOptions{})
	if _, err := loader.LoadReader[AtlasMeta](emptyStream(nil), FormatJSON); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream decode = %v, want io.EOF", err)
	}
	saver, err := NewSaver(unavailableStore(nil), SaverOptions{})
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

type customJSON string

func (v customJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal("custom:" + string(v))
}

func (v *customJSON) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*v = customJSON(strings.TrimPrefix(value, "custom:"))
	return nil
}

func TestJSONDocumentContract(t *testing.T) {
	codecs := NewCodecs()
	if err := codecs.SetJSON[customJSON](); err != nil {
		t.Fatal(err)
	}
	if err := codecs.SetJSON[spriteMeta](); err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	saver, err := NewSaver(nil, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := saver.SaveWriter(&output, customJSON("value"), FormatJSON); err != nil {
		t.Fatal(err)
	}
	if output.String() != "\"custom:value\"\n" {
		t.Fatalf("custom marshaler output = %q", output.String())
	}
	got, err := loader.LoadReader[customJSON](&output, FormatJSON)
	if err != nil || got != "value" {
		t.Fatalf("custom unmarshaler = %q, %v", got, err)
	}
	for _, input := range []string{`{"name":"ok"} {}`, `{"name":"ok"} garbage`, `{"name":5}`, `{"name":`, ``} {
		if _, err := loader.LoadReader[spriteMeta](strings.NewReader(input), FormatJSON); err == nil {
			t.Errorf("accepted invalid JSON document %q", input)
		}
	}
	gotMeta, err := loader.LoadReader[spriteMeta](strings.NewReader("{\"unknown\":true} \n\t"), FormatJSON)
	if err != nil || gotMeta.Name != "" {
		t.Fatalf("zero defaults/unknown fields/trailing whitespace = %+v, %v", gotMeta, err)
	}
	want := errors.New("source failed")
	_, err = loader.LoadReader[spriteMeta](io.MultiReader(strings.NewReader(`{}`), failedReader{want}), FormatJSON)
	if !errors.Is(err, want) {
		t.Fatalf("trailing read error = %v, want %v", err, want)
	}
}

type failedReader struct{ err error }

func (r failedReader) Read([]byte) (int, error) { return 0, r.err }
