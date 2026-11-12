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

type memoryStore struct {
	files fstest.MapFS
	calls int
	err   error
}

func (s *memoryStore) Open(name string) (fs.File, error) { return s.files.Open(name) }

func (s *memoryStore) Write(name string, encode func(io.Writer) error) error {
	s.calls++
	var output bytes.Buffer
	if err := encode(&output); err != nil {
		return err
	}
	if s.err != nil {
		return s.err
	}
	if s.files == nil {
		s.files = make(fstest.MapFS)
	}
	s.files[name] = &fstest.MapFile{Data: bytes.Clone(output.Bytes())}
	return nil
}

func TestSaveRoundTripAndFormatDispatch(t *testing.T) {
	codecs := NewCodecs()
	if err := setTestJSON[spriteMeta](codecs); err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{}
	saver, err := NewSaver(store, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	want := spriteMeta{Name: "player"}
	if err := saver.Save("./profiles/player.data", want, WithFormat(FormatJSON)); err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, store, LoaderOptions{Codecs: codecs})
	got, err := loader.Load[spriteMeta]("profiles/player.data", WithFormat(FormatJSON))
	if err != nil || got != want {
		t.Fatalf("round trip = %+v, %v; want %+v", got, err, want)
	}
	if err := saver.Save("PROFILE.JSON", want); err != nil {
		t.Fatalf("format inference: %v", err)
	}
	before := store.calls
	if err := saver.Save("wrong.json", "unregistered type"); err == nil {
		t.Fatal("format alone selected an encoder for the wrong type")
	}
	for _, name := range []string{"", ".", "../outside.json", "/outside.json", `profiles\player.json`} {
		if err := saver.Save(name, want); err == nil {
			t.Errorf("Save(%q) accepted invalid name", name)
		}
	}
	if err := saver.Save("player.json", want, nil); err == nil {
		t.Fatal("nil save option accepted")
	}
	if store.calls != before {
		t.Fatal("invalid saves invoked storage")
	}
}

func TestSaveFailuresPreserveCauses(t *testing.T) {
	want := errors.New("write failed")
	store := &memoryStore{files: fstest.MapFS{"profile.json": {Data: []byte("old")}}}
	saver, err := NewSaver(store, SaverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := saver.RegisterEncoder(FormatJSON, func(w io.Writer, _ spriteMeta) error {
		if _, err := io.WriteString(w, "partial"); err != nil {
			return err
		}
		return want
	}); err != nil {
		t.Fatal(err)
	}
	err = saver.Save("profile.json", spriteMeta{})
	var assetErr *AssetError
	if !errors.Is(err, want) || !errors.As(err, &assetErr) || assetErr.ID != "profile.json" {
		t.Fatalf("encoding error lost context: %v", err)
	}
	if string(store.files["profile.json"].Data) != "old" {
		t.Fatal("failed encoding published incomplete output")
	}

	good, err := NewSaver(store, SaverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := good.RegisterEncoder(FormatJSON, func(w io.Writer, v string) error {
		_, err := io.WriteString(w, v)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	store.err = want
	if err := good.Save("profile.json", "new"); !errors.Is(err, want) {
		t.Fatalf("storage error lost cause: %v", err)
	}
	if string(store.files["profile.json"].Data) != "old" {
		t.Fatal("failed store published output")
	}
	if err := good.RegisterEncoder(FormatJSON, func(io.Writer, string) error { return nil }); err == nil {
		t.Fatal("local encoder registration replaced an existing encoder")
	}
}

type trackedStream struct {
	bytes.Buffer
	closed bool
	err    error
}

func (s *trackedStream) Close() error { s.closed = true; return nil }
func (s *trackedStream) Write(p []byte) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.Buffer.Write(p)
}

func TestStreamOwnershipAndWriterErrors(t *testing.T) {
	codecs := NewCodecs()
	if err := setTestJSON[spriteMeta](codecs); err != nil {
		t.Fatal(err)
	}
	saver, err := NewSaver(nil, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	stream := &trackedStream{}
	if err := saver.SaveWriter(stream, spriteMeta{Name: "player"}, FormatJSON); err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, nil, LoaderOptions{Codecs: codecs})
	got, err := loader.LoadReader[spriteMeta](stream, FormatJSON)
	if err != nil || got.Name != "player" {
		t.Fatalf("stream round trip = %+v, %v", got, err)
	}
	if stream.closed {
		t.Fatal("facades closed a caller-owned stream")
	}
	if err := saver.Save("player.json", spriteMeta{}); err == nil || !strings.Contains(err.Error(), "store") {
		t.Fatalf("stream-only path save = %v", err)
	}
	for _, writer := range []io.Writer{nil, (*bytes.Buffer)(nil)} {
		if err := saver.SaveWriter(writer, spriteMeta{}, FormatJSON); err == nil {
			t.Fatal("nil writer accepted")
		}
	}
	want := errors.New("writer failed")
	stream.err = want
	err = saver.SaveWriter(stream, spriteMeta{}, FormatJSON)
	var assetErr *AssetError
	if !errors.Is(err, want) || !errors.As(err, &assetErr) || assetErr.Op != "Encode" || assetErr.ID != "" {
		t.Fatalf("writer failure lost pathless encoding context: %v", err)
	}
}

func TestCodecsPreserveNilValues(t *testing.T) {
	codecs := &Codecs{}
	decodes := 0
	if err := codecs.SetDecoder(Format("nil"), func(io.Reader) (any, error) {
		decodes++
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := codecs.SetEncoder(Format("nil"), func(_ io.Writer, value any) error {
		calls++
		if value != nil {
			t.Fatalf("nil interface became %T", value)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := setTestJSON[*spriteMeta](codecs); err != nil {
		t.Fatal(err)
	}
	loader := newTestLoader(t, newTestFS(map[string]string{"value.nil": "", "pointer.json": "null"}), LoaderOptions{Codecs: codecs})
	saver, err := NewSaver(nil, SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	if value, err := loader.LoadReader[any](strings.NewReader(""), Format("nil")); err != nil || value != nil {
		t.Fatalf("nil interface stream decode = %v, %v", value, err)
	}
	for range 2 {
		value, err := loader.Load[any]("value.nil")
		if err != nil || value != nil {
			t.Fatalf("nil interface load = %v, %v", value, err)
		}
		if err := saver.SaveWriter(io.Discard, value, Format("nil")); err != nil {
			t.Fatal(err)
		}
		pointer, err := loader.Load[*spriteMeta]("pointer.json")
		if err != nil || pointer != nil {
			t.Fatalf("typed nil pointer load = %v, %v", pointer, err)
		}
		var output bytes.Buffer
		if err := saver.SaveWriter(&output, pointer, FormatJSON); err != nil {
			t.Fatal(err)
		}
		if output.String() != "null\n" {
			t.Fatalf("typed nil pointer encoded as %q", output.String())
		}
	}
	if decodes != 2 || calls != 2 {
		t.Fatalf("decodes=%d encodes=%d, want one stream decode, one cached decode, and two encodes", decodes, calls)
	}
}
