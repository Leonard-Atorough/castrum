package asset

import (
	"encoding/json"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"
)

type spriteMeta struct {
	Name string `json:"name"`
}

func newTestLoader(t testing.TB, filesystem fs.FS, opts LoaderOptions) *Loader {
	t.Helper()
	loader, err := NewLoader(filesystem, opts)
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

func jsonDecoder[T any]() Decoder[T] {
	return func(reader io.Reader) (T, error) {
		var value T
		err := json.NewDecoder(reader).Decode(&value)
		return value, err
	}
}

func newTestFS(files map[string]string) fs.FS {
	mapFS := make(fstest.MapFS, len(files))
	for name, data := range files {
		mapFS[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return mapFS
}

func mustRegisterJSONDecoder(t *testing.T, a *Loader) {
	t.Helper()
	if err := a.codecs.SetDecoder(FormatJSON, jsonDecoder[spriteMeta]()); err != nil {
		t.Fatalf("SetDecoder: %v", err)
	}
}
