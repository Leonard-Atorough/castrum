package asset

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"testing/synctest"
)

// spriteMeta stands in for the kinds of values games decode. Tests use a
// real JSON decoder over fstest.MapFS so the public surface is exercised
// the way games exercise it.
type spriteMeta struct {
	Name string `json:"name"`
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
	if err := a.RegisterDecoder(FormatJSON, jsonDecoder[spriteMeta](), false); err != nil {
		t.Fatalf("RegisterDecoder: %v", err)
	}
}

func TestLoadDecodesRegisteredAsset(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}))
	mustRegisterJSONDecoder(t, a)

	meta, err := a.Load[spriteMeta]("sprites/player.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if meta.Name != "player" {
		t.Fatalf("decoded %+v, want name %q", meta, "player")
	}
}

func TestLoadNormalizesPath(t *testing.T) {
	// "./"-prefixed and dot-segmented names must resolve like the clean
	// name: the cleaned path reaches the fs open, not just the cache id.
	a := NewLoader(newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}))
	mustRegisterJSONDecoder(t, a)

	for _, name := range []string{"./sprites/player.json", "sprites/./player.json", "sprites/../sprites/player.json"} {
		if _, err := a.Load[spriteMeta](name); err != nil {
			t.Errorf("Load(%q): %v", name, err)
		}
	}
}

func TestLoadResolvesFormatCaseInsensitively(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"DATA.JSON": `{"name":"player"}`}))
	mustRegisterJSONDecoder(t, a)

	meta, err := a.Load[spriteMeta]("DATA.JSON")
	if err != nil {
		t.Fatalf("Load with an uppercase extension: %v", err)
	}
	if meta.Name != "player" {
		t.Fatalf("decoded %+v, want name %q", meta, "player")
	}
}

func TestLoadServesFromCache(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}))
	var decodes atomic.Int32
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(r io.Reader) (spriteMeta, error) {
		decodes.Add(1)
		return jsonDecoder[spriteMeta]()(r)
	}), false); err != nil {
		t.Fatalf("RegisterDecoder: %v", err)
	}

	first, err := a.Load[spriteMeta]("sprites/player.json")
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	second, err := a.Load[spriteMeta]("sprites/player.json")
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if decodes.Load() != 1 {
		t.Fatalf("decoded %d times, want 1 (second Load must be a cache hit)", decodes.Load())
	}
	if first != second {
		t.Fatalf("cache served different values: %+v vs %+v", first, second)
	}
}

func TestLoadWithIDOverridesCacheIdentity(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{
		"a.json": `{"name":"a"}`,
		"b.json": `{"name":"b"}`,
	}))
	mustRegisterJSONDecoder(t, a)

	// Same ID, different files: the collision rule - first load wins.
	if _, err := a.Load[spriteMeta]("a.json", WithID("shared")); err != nil {
		t.Fatalf("first Load: %v", err)
	}
	collided, err := a.Load[spriteMeta]("b.json", WithID("shared"))
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if collided.Name != "a" {
		t.Fatalf("collision returned %+v, want the first load's value", collided)
	}

	// Without WithID, different files have different identities.
	plain, err := a.Load[spriteMeta]("b.json")
	if err != nil {
		t.Fatalf("Load without ID override: %v", err)
	}
	if plain.Name != "b" {
		t.Fatalf("Load without ID override returned %+v, want %q", plain, "b")
	}
}

func TestLoadWithFormatOverridesExtensionAndCacheIdentity(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"data": "contents"}))
	var jsonDecodes, yamlDecodes int
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		jsonDecodes++
		return spriteMeta{Name: "json"}, nil
	}), false); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterDecoder(FormatYAML, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		yamlDecodes++
		return spriteMeta{Name: "yaml"}, nil
	}), false); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		format Format
		want   string
	}{{FormatJSON, "json"}, {FormatYAML, "yaml"}, {FormatJSON, "json"}} {
		got, err := a.Load[spriteMeta]("data", WithFormat(tc.format))
		if err != nil || got.Name != tc.want {
			t.Fatalf("Load with format %q = %+v, %v; want %q", tc.format, got, err, tc.want)
		}
	}
	if jsonDecodes != 1 || yamlDecodes != 1 {
		t.Errorf("decodes: JSON %d, YAML %d; want one each", jsonDecodes, yamlDecodes)
	}
}

func TestLoadDeduplicatesConcurrentLoads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := NewLoader(newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}))
		decodes := atomic.Int32{}
		release := make(chan struct{})
		if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(r io.Reader) (spriteMeta, error) {
			decodes.Add(1)
			<-release // hold the first decode so peers pile up on the singleflight
			return jsonDecoder[spriteMeta]()(r)
		}), false); err != nil {
			t.Fatalf("RegisterDecoder: %v", err)
		}

		const peers = 8
		results := make([]spriteMeta, peers)
		errs := make([]error, peers)
		var done sync.WaitGroup
		done.Add(peers)
		for i := range peers {
			go func(i int) {
				results[i], errs[i] = a.Load[spriteMeta]("sprites/player.json")
				done.Done()
			}(i)
		}
		synctest.Wait()
		if got := decodes.Load(); got != 1 {
			t.Errorf("in-flight decodes = %d, want 1", got)
		}
		close(release)
		done.Wait()

		for i := range peers {
			if errs[i] != nil {
				t.Fatalf("peer %d: %v", i, errs[i])
			}
			if results[i].Name != "player" {
				t.Fatalf("peer %d decoded %+v", i, results[i])
			}
		}
		if decodes.Load() != 1 {
			t.Fatalf("%d peers triggered %d decodes, want 1", peers, decodes.Load())
		}
	})
}

func TestLoadPointerTypeCachesNil(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}))
	var decodes int
	if err := a.RegisterDecoder(FormatJSON, Decoder[*spriteMeta](func(io.Reader) (*spriteMeta, error) {
		decodes++
		return nil, nil // a pointer decoder may legitimately produce nil
	}), false); err != nil {
		t.Fatalf("RegisterDecoder: %v", err)
	}

	first, err := a.Load[*spriteMeta]("sprites/player.json")
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	if first != nil {
		t.Fatalf("first Load = %+v, want nil", first)
	}
	second, err := a.Load[*spriteMeta]("sprites/player.json")
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if second != nil {
		t.Fatalf("cached nil served as %+v, want nil", second)
	}
	if decodes != 1 {
		t.Errorf("decodes = %d, want 1", decodes)
	}
}

func TestLoadRetriesAfterDecodeFailure(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"data.json": `{}`}))
	wantErr := errors.New("temporary decode failure")
	decodes := 0
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		decodes++
		if decodes == 1 {
			return spriteMeta{}, wantErr
		}
		return spriteMeta{Name: "recovered"}, nil
	}), false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Load[spriteMeta]("data.json"); !errors.Is(err, wantErr) {
		t.Fatalf("first Load = %v, want wrapped %v", err, wantErr)
	}
	for i := 0; i < 2; i++ {
		got, err := a.Load[spriteMeta]("data.json")
		if err != nil || got.Name != "recovered" {
			t.Fatalf("Load after failure = %+v, %v; want recovered", got, err)
		}
	}
	if decodes != 2 {
		t.Errorf("decodes = %d, want one failed and one successful decode", decodes)
	}
}

func TestLoadMissingFileWrapsFSError(t *testing.T) {
	a := NewLoader(newTestFS(nil))
	mustRegisterJSONDecoder(t, a)

	_, err := a.Load[spriteMeta]("missing.json")
	if err == nil {
		t.Fatal("Load of a missing file should error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %v should unwrap to fs.ErrNotExist", err)
	}
	if !strings.Contains(err.Error(), "missing.json") {
		t.Errorf("error %v should name the asset", err)
	}
}

func TestLoadWithoutDecoderErrors(t *testing.T) {
	a := NewLoader(newTestFS(map[string]string{"sprites/player.xml": "<sprite/>"}))
	mustRegisterJSONDecoder(t, a)

	_, err := a.Load[spriteMeta]("sprites/player.xml")
	if err == nil {
		t.Fatal("Load with no registered decoder should error")
	}
	if !strings.Contains(err.Error(), "xml") {
		t.Errorf("error %v should name the format", err)
	}
}

func TestLoadReaderDecodesWithoutCaching(t *testing.T) {
	a := NewLoader(nil)
	var decodes atomic.Int32
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(r io.Reader) (spriteMeta, error) {
		decodes.Add(1)
		return jsonDecoder[spriteMeta]()(r)
	}), false); err != nil {
		t.Fatalf("RegisterDecoder: %v", err)
	}

	for i := range 2 {
		meta, err := a.LoadReader[spriteMeta](strings.NewReader(`{"name":"player"}`), FormatJSON)
		if err != nil {
			t.Fatalf("LoadReader %d: %v", i, err)
		}
		if meta.Name != "player" {
			t.Fatalf("LoadReader %d decoded %+v", i, meta)
		}
	}
	if decodes.Load() != 2 {
		t.Fatalf("LoadReader decoded %d times, want 2 (no caching)", decodes.Load())
	}
}

func TestLoadReaderRequiresFormat(t *testing.T) {
	a := NewLoader(nil)
	mustRegisterJSONDecoder(t, a)

	if _, err := a.LoadReader[spriteMeta](strings.NewReader(`{}`), "xml"); err == nil {
		t.Error("LoadReader with an unregistered format should error")
	}
}

func TestRegisterDecoderValidation(t *testing.T) {
	a := NewLoader(nil)

	if err := a.RegisterDecoder(FormatJSON, jsonDecoder[spriteMeta](), false); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	if err := a.RegisterDecoder(FormatJSON, jsonDecoder[spriteMeta](), false); err == nil {
		t.Error("duplicate registration without override should error")
	}
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](nil), false); err == nil {
		t.Error("nil decoder should error")
	}
	if err := a.RegisterDecoder("", jsonDecoder[spriteMeta](), false); err == nil {
		t.Error("empty format should error")
	}
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		return spriteMeta{Name: "replaced"}, nil
	}), true); err != nil {
		t.Fatalf("override registration: %v", err)
	}

	meta, err := a.LoadReader[spriteMeta](strings.NewReader(`{}`), FormatJSON)
	if err != nil {
		t.Fatalf("LoadReader after override: %v", err)
	}
	if meta.Name != "replaced" {
		t.Fatalf("decoder after override = %+v, want the replacement", meta)
	}
}

func TestNewDefaultsToFilesystem(t *testing.T) {
	a := NewLoader(nil)
	if a == nil {
		t.Fatal("NewLoader(nil) returned nil")
	}
	if a.fs == nil {
		t.Fatal("NewLoader(nil) left the filesystem nil")
	}
}

func TestCacheCountsHits(t *testing.T) {
	c := newCache()
	key := newLoadKey("test", reflect.TypeFor[spriteMeta](), FormatJSON)
	c.put(key, spriteMeta{Name: "x"})

	for i := range 3 {
		if _, ok := c.get(key); !ok {
			t.Fatalf("get %d missed after put", i)
		}
	}
	if hits := c.entries[key].hits.Load(); hits != 3 {
		t.Fatalf("hits = %d after 3 gets, want 3", hits)
	}
	if _, ok := c.get(newLoadKey("other", reflect.TypeFor[spriteMeta](), FormatJSON)); ok {
		t.Error("get with an unknown key should miss")
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := newCache()
	typ := reflect.TypeFor[spriteMeta]()

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := newLoadKey("shared", typ, FormatJSON)
			c.put(key, spriteMeta{Name: "v"})
			c.get(key)
		}(i)
	}
	wg.Wait()

	key := newLoadKey("shared", typ, FormatJSON)
	if _, ok := c.get(key); !ok {
		t.Fatal("cache entry lost after concurrent access")
	}
}

func TestOpenReturnsFileAndFormat(t *testing.T) {
	mapFS := fstest.MapFS{"sfx/tone.wav": &fstest.MapFile{Data: []byte("RIFF-data")}}
	a := NewLoader(mapFS)

	file, format, err := a.Open("sfx/tone.wav")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer file.Close()
	if format != FormatWAV {
		t.Fatalf("format = %q, want wav", format)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(file, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "RIFF" {
		t.Fatalf("read %q, want RIFF", buf)
	}
}

func TestOpenErrors(t *testing.T) {
	a := NewLoader(fstest.MapFS{})
	if _, _, err := a.Open(""); err == nil {
		t.Error("empty path: want error")
	}
	if _, _, err := a.Open("missing.wav"); err == nil {
		t.Error("missing file: want error")
	}
}
