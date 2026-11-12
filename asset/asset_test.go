package asset

import (
	"encoding/json"
	"errors"
	"fmt"
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
	if err := a.RegisterDecoder(FormatJSON, jsonDecoder[spriteMeta]()); err != nil {
		t.Fatalf("RegisterDecoder: %v", err)
	}
}

func TestLoadDecodesRegisteredAsset(t *testing.T) {
	a := newTestLoader(t, newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}), LoaderOptions{})
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
	a := newTestLoader(t, newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}), LoaderOptions{})
	decodes := 0
	if err := a.RegisterDecoder(FormatJSON, func(r io.Reader) (spriteMeta, error) {
		decodes++
		return jsonDecoder[spriteMeta]()(r)
	}); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"./sprites/player.json", "sprites/./player.json", "sprites/../sprites/player.json"} {
		if _, err := a.Load[spriteMeta](name); err != nil {
			t.Errorf("Load(%q): %v", name, err)
		}
	}
	if decodes != 1 {
		t.Fatalf("equivalent paths decoded %d times, want 1", decodes)
	}
}

func TestLoadResolvesFormatCaseInsensitively(t *testing.T) {
	a := newTestLoader(t, newTestFS(map[string]string{"DATA.JSON": `{"name":"player"}`}), LoaderOptions{})
	mustRegisterJSONDecoder(t, a)

	meta, err := a.Load[spriteMeta]("DATA.JSON")
	if err != nil {
		t.Fatalf("Load with an uppercase extension: %v", err)
	}
	if meta.Name != "player" {
		t.Fatalf("decoded %+v, want name %q", meta, "player")
	}
}

func TestLoadCachePolicies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		policy  CachePolicy
		want    [5]string
		decodes [5]int
	}{
		{"reuse", CacheReuse, [5]string{"original", "original", "fresh", "original", "original"}, [5]int{1, 1, 2, 2, 2}},
		{"bypass", CacheBypass, [5]string{"original", "cached", "fresh", "cached", "last"}, [5]int{1, 2, 3, 3, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := fstest.MapFS{}
			loader := newTestLoader(t, files, LoaderOptions{CachePolicy: tc.policy})
			decodes := 0
			if err := loader.RegisterDecoder(FormatJSON, func(r io.Reader) (spriteMeta, error) {
				decodes++
				return jsonDecoder[spriteMeta]()(r)
			}); err != nil {
				t.Fatal(err)
			}
			steps := []struct {
				source string
				opts   []loadOption
			}{
				{"original", nil},
				{"cached", []loadOption{WithCachePolicy(CacheReuse)}},
				{"fresh", []loadOption{WithCachePolicy(CacheBypass)}},
				{"later", []loadOption{WithCachePolicy(CacheReuse)}},
				{"last", nil},
			}
			for i, step := range steps {
				files["profile.json"] = &fstest.MapFile{Data: []byte(`{"name":"` + step.source + `"}`)}
				got, err := loader.Load[spriteMeta]("profile.json", step.opts...)
				if err != nil || got.Name != tc.want[i] || decodes != tc.decodes[i] {
					t.Fatalf("source %q = %+v, %v; decodes=%d; want %q and %d decodes",
						step.source, got, err, decodes, tc.want[i], tc.decodes[i])
				}
			}
		})
	}
}

func TestLoadBypassDoesNotJoinOrReplaceCachedWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files := fstest.MapFS{"profile.json": {Data: []byte(`{"name":"old"}`)}}
		loader := newTestLoader(t, files, LoaderOptions{})
		oldGate, freshGate := make(chan struct{}), make(chan struct{})
		var decodes atomic.Int32
		if err := loader.RegisterDecoder(FormatJSON, func(r io.Reader) (*spriteMeta, error) {
			value, err := jsonDecoder[*spriteMeta]()(r)
			if decodes.Add(1) == 1 {
				<-oldGate
			} else {
				<-freshGate
			}
			return value, err
		}); err != nil {
			t.Fatal(err)
		}
		var cached *spriteMeta
		var cachedErr error
		oldDone := make(chan struct{})
		go func() {
			cached, cachedErr = loader.Load[*spriteMeta]("profile.json")
			close(oldDone)
		}()
		synctest.Wait()
		files["profile.json"] = &fstest.MapFile{Data: []byte(`{"name":"new"}`)}
		var fresh [2]*spriteMeta
		var errs [2]error
		var done sync.WaitGroup
		for i := range fresh {
			done.Go(func() {
				fresh[i], errs[i] = loader.Load[*spriteMeta]("profile.json", WithCachePolicy(CacheBypass))
			})
		}
		synctest.Wait()
		if got := decodes.Load(); got != 3 {
			close(freshGate)
			close(oldGate)
			done.Wait()
			<-oldDone
			t.Fatalf("independent decodes = %d, want 3", got)
		}
		close(freshGate)
		done.Wait()
		select {
		case <-oldDone:
			t.Error("cached read finished before its decoder was released")
		default:
		}
		close(oldGate)
		<-oldDone
		if cachedErr != nil || cached == nil || cached.Name != "old" {
			t.Fatalf("old cached read = %+v, %v", cached, cachedErr)
		}
		for i := range fresh {
			if errs[i] != nil || fresh[i] == nil || fresh[i].Name != "new" {
				t.Fatalf("bypass %d = %+v, %v", i, fresh[i], errs[i])
			}
		}
		if fresh[0] == fresh[1] {
			t.Fatal("bypass reads shared one decoded object")
		}
		got, err := loader.Load[*spriteMeta]("profile.json")
		if err != nil || got != cached || decodes.Load() != 3 {
			t.Fatalf("cached read after bypass = %+v, %v; decodes = %d", got, err, decodes.Load())
		}
	})
}

type readTrackingFS struct {
	fs.FS
	opens, closes atomic.Int32
	closeErr      error
	closeGate     <-chan struct{}
}

func (s *readTrackingFS) Open(name string) (fs.File, error) {
	s.opens.Add(1)
	file, err := s.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return &readTrackingFile{File: file, owner: s}, nil
}

type readTrackingFile struct {
	fs.File
	owner *readTrackingFS
}

func (f *readTrackingFile) Close() error {
	if f.owner.closeGate != nil {
		<-f.owner.closeGate
	}
	f.owner.closes.Add(1)
	return errors.Join(f.File.Close(), f.owner.closeErr)
}

func TestLoadRejectsInvalidOptionsBeforeIO(t *testing.T) {
	files := &readTrackingFS{FS: fstest.MapFS{}}
	for _, policy := range []CachePolicy{-1, 2} {
		if _, err := NewLoader(files, LoaderOptions{CachePolicy: policy}); err == nil {
			t.Fatalf("constructor accepted cache policy %d", policy)
		}
	}
	loader := newTestLoader(t, files, LoaderOptions{})
	for _, opt := range []loadOption{nil, (*formatOption)(nil), WithCachePolicy(-1), WithCachePolicy(2)} {
		_, err := loader.Load[AtlasMeta]("profile.json", opt)
		var assetErr *AssetError
		if !errors.As(err, &assetErr) || assetErr.Op != "Load" || assetErr.ID != "profile.json" {
			t.Fatalf("invalid load option lost error context: %v", err)
		}
	}
	if files.opens.Load() != 0 {
		t.Fatal("invalid options invoked storage")
	}
}

func TestLoadFailuresDoNotPublishValues(t *testing.T) {
	for _, policy := range []CachePolicy{CacheReuse, CacheBypass} {
		for _, failure := range []struct {
			name          string
			decode, close bool
		}{{"decode", true, false}, {"close", false, true}, {"decode-and-close", true, true}} {
			t.Run(fmt.Sprintf("%d/%s", policy, failure.name), func(t *testing.T) {
				closeErr, decodeErr := errors.New("close failed"), errors.New("decode failed")
				files := &readTrackingFS{
					FS: newTestFS(map[string]string{"profile.json": `{"name":"value"}`}),
				}
				if failure.close {
					files.closeErr = closeErr
				}
				failDecode := failure.decode
				loader := newTestLoader(t, files, LoaderOptions{CachePolicy: policy})
				if err := loader.RegisterDecoder(FormatJSON, func(r io.Reader) (spriteMeta, error) {
					if failDecode {
						return spriteMeta{Name: "partial"}, decodeErr
					}
					return jsonDecoder[spriteMeta]()(r)
				}); err != nil {
					t.Fatal(err)
				}
				got, err := loader.Load[spriteMeta]("profile.json")
				if got != (spriteMeta{}) || errors.Is(err, closeErr) != failure.close || errors.Is(err, decodeErr) != failure.decode {
					t.Fatalf("failed read = %+v, %v; want zero value and underlying causes", got, err)
				}
				var assetErr *AssetError
				if !errors.As(err, &assetErr) || assetErr.ID != "profile.json" {
					t.Fatalf("close failure lost asset context: %v", err)
				}
				failDecode = false
				files.closeErr = nil
				got, err = loader.Load[spriteMeta]("profile.json")
				if err != nil || got.Name != "value" || files.opens.Load() != 2 || files.closes.Load() != 2 {
					t.Fatalf("retry after failed read = %+v, %v; opens=%d closes=%d",
						got, err, files.opens.Load(), files.closes.Load())
				}
			})
		}
	}
}

func TestLoadInFlightIdentityUsesExactType(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		loader := newTestLoader(t, newTestFS(map[string]string{"value.json": `{}`}), LoaderOptions{})
		gate := make(chan struct{})
		var first, second func() error
		var types [2]reflect.Type
		{
			type document struct{ Name string }
			types[0] = reflect.TypeFor[document]()
			if err := loader.RegisterDecoder(FormatJSON, func(io.Reader) (document, error) {
				<-gate
				return document{Name: "first"}, nil
			}); err != nil {
				t.Fatal(err)
			}
			first = func() error {
				got, err := loader.Load[document]("value.json")
				if err != nil || got.Name != "first" {
					return fmt.Errorf("first type = %+v, %v", got, err)
				}
				return nil
			}
		}
		{
			type document struct{ Name string }
			types[1] = reflect.TypeFor[document]()
			if err := loader.RegisterDecoder(FormatJSON, func(io.Reader) (document, error) {
				return document{Name: "second"}, nil
			}); err != nil {
				t.Fatal(err)
			}
			second = func() error {
				got, err := loader.Load[document]("value.json")
				if err != nil || got.Name != "second" {
					return fmt.Errorf("second type = %+v, %v", got, err)
				}
				return nil
			}
		}
		if types[0] == types[1] || types[0].String() != types[1].String() {
			t.Fatal("test requires distinct types with the same printed name")
		}
		firstDone, secondDone := make(chan error, 1), make(chan error, 1)
		go func() { firstDone <- first() }()
		synctest.Wait()
		go func() { secondDone <- second() }()
		synctest.Wait()
		select {
		case err := <-secondDone:
			if err != nil {
				t.Error(err)
			}
		default:
			t.Error("different types joined the same in-flight load")
		}
		close(gate)
		if err := <-firstDone; err != nil {
			t.Error(err)
		}
		if _, err := loader.Load[AtlasMeta]("value.json"); err != nil {
			t.Fatalf("same-path third representation: %v", err)
		}
	})
}

func TestLoadWithFormatOverridesExtensionAndCacheIdentity(t *testing.T) {
	a := newTestLoader(t, newTestFS(map[string]string{"data": "contents"}), LoaderOptions{})
	var jsonDecodes, yamlDecodes int
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		jsonDecodes++
		return spriteMeta{Name: "json"}, nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterDecoder(FormatYAML, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		yamlDecodes++
		return spriteMeta{Name: "yaml"}, nil
	})); err != nil {
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
	for _, failure := range []string{"none", "error", "panic"} {
		t.Run(failure, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				files := &readTrackingFS{
					FS:        newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}),
					closeGate: release,
				}
				a := newTestLoader(t, files, LoaderOptions{})
				decodes := atomic.Int32{}
				completed := atomic.Int32{}
				wantErr := errors.New("decode failed")
				fail := failure != "none"
				if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(r io.Reader) (spriteMeta, error) {
					decodes.Add(1)
					if fail {
						if failure == "panic" {
							panic(wantErr)
						}
						return spriteMeta{Name: "partial"}, wantErr
					}
					return jsonDecoder[spriteMeta]()(r)
				})); err != nil {
					t.Fatalf("RegisterDecoder: %v", err)
				}

				const peers = 8
				results := make([]spriteMeta, peers)
				errs := make([]error, peers)
				panics := make([]any, peers)
				var done sync.WaitGroup
				done.Add(peers)
				for i := range peers {
					go func(i int) {
						defer done.Done()
						defer func() {
							panics[i] = recover()
							completed.Add(1)
						}()
						results[i], errs[i] = a.Load[spriteMeta]("sprites/player.json")
					}(i)
				}
				synctest.Wait()
				if got := decodes.Load(); got != 1 {
					t.Errorf("in-flight decodes = %d, want 1", got)
				}
				if got := completed.Load(); got != 0 {
					t.Errorf("%d loads completed before the file closed", got)
				}
				close(release)
				done.Wait()

				for i := range peers {
					if failure == "panic" {
						if panics[i] != wantErr {
							t.Fatalf("peer %d panic = %v, want %v", i, panics[i], wantErr)
						}
					} else if panics[i] != nil || (fail && !errors.Is(errs[i], wantErr)) || (!fail && errs[i] != nil) {
						t.Fatalf("peer %d error = %v, panic = %v", i, errs[i], panics[i])
					}
					want := spriteMeta{Name: "player"}
					if fail {
						want = spriteMeta{}
					}
					if results[i] != want {
						t.Fatalf("peer %d decoded %+v, want %+v", i, results[i], want)
					}
				}
				if decodes.Load() != 1 {
					t.Fatalf("%d peers triggered %d decodes, want 1", peers, decodes.Load())
				}
				if files.opens.Load() != 1 || files.closes.Load() != 1 {
					t.Fatalf("shared read opens=%d closes=%d, want 1 each", files.opens.Load(), files.closes.Load())
				}
				fail = false
				for range 2 {
					got, err := a.Load[spriteMeta]("sprites/player.json")
					if err != nil || got.Name != "player" {
						t.Fatalf("load after shared work = %+v, %v", got, err)
					}
				}
				wantDecodes := int32(1)
				if failure != "none" {
					wantDecodes = 2
				}
				if decodes.Load() != wantDecodes || files.opens.Load() != wantDecodes || files.closes.Load() != wantDecodes {
					t.Fatalf("completed work: decodes=%d opens=%d closes=%d; want %d each",
						decodes.Load(), files.opens.Load(), files.closes.Load(), wantDecodes)
				}
			})
		})
	}
}

func TestLoadPointerTypeCachesNil(t *testing.T) {
	a := newTestLoader(t, newTestFS(map[string]string{"sprites/player.json": `{"name":"player"}`}), LoaderOptions{})
	var decodes int
	if err := a.RegisterDecoder(FormatJSON, Decoder[*spriteMeta](func(io.Reader) (*spriteMeta, error) {
		decodes++
		return nil, nil // a pointer decoder may legitimately produce nil
	})); err != nil {
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

func TestLoadMissingFileWrapsFSError(t *testing.T) {
	a := newTestLoader(t, newTestFS(nil), LoaderOptions{})
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
	a := newTestLoader(t, newTestFS(map[string]string{"sprites/player.xml": "<sprite/>"}), LoaderOptions{})
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
	a := newTestLoader(t, nil, LoaderOptions{})
	var decodes atomic.Int32
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(r io.Reader) (spriteMeta, error) {
		decodes.Add(1)
		return jsonDecoder[spriteMeta]()(r)
	})); err != nil {
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
	a := newTestLoader(t, nil, LoaderOptions{})
	mustRegisterJSONDecoder(t, a)

	if _, err := a.LoadReader[spriteMeta](strings.NewReader(`{}`), "xml"); err == nil {
		t.Error("LoadReader with an unregistered format should error")
	}
}

func TestRegisterDecoderValidation(t *testing.T) {
	a := newTestLoader(t, nil, LoaderOptions{})

	if err := a.RegisterDecoder(FormatJSON, jsonDecoder[spriteMeta]()); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	if err := a.RegisterDecoder(FormatJSON, jsonDecoder[spriteMeta]()); err == nil {
		t.Error("duplicate registration without override should error")
	}
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](nil)); err == nil {
		t.Error("nil decoder should error")
	}
	if err := a.RegisterDecoder("", jsonDecoder[spriteMeta]()); err == nil {
		t.Error("empty format should error")
	}
	if err := a.RegisterDecoder(FormatJSON, Decoder[spriteMeta](func(io.Reader) (spriteMeta, error) {
		return spriteMeta{Name: "replaced"}, nil
	})); err == nil {
		t.Fatal("duplicate registration should not replace the decoder")
	}

	meta, err := a.LoadReader[spriteMeta](strings.NewReader(`{"name":"original"}`), FormatJSON)
	if err != nil {
		t.Fatalf("LoadReader after rejected replacement: %v", err)
	}
	if meta.Name != "original" {
		t.Fatalf("decoder after rejected replacement = %+v, want original", meta)
	}
}

func TestNewDefaultsToFilesystem(t *testing.T) {
	a := newTestLoader(t, nil, LoaderOptions{})
	if a == nil {
		t.Fatal("NewLoader(nil) returned nil")
	}
	if a.fs == nil {
		t.Fatal("NewLoader(nil) left the filesystem nil")
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
	a := newTestLoader(t, mapFS, LoaderOptions{})

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
	a := newTestLoader(t, fstest.MapFS{}, LoaderOptions{})
	if _, _, err := a.Open(""); err == nil {
		t.Error("empty path: want error")
	}
	if _, _, err := a.Open("missing.wav"); err == nil {
		t.Error("missing file: want error")
	}
}
