package castrum

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
)

func TestGameAudioPreferenceCodecDefaults(t *testing.T) {
	files := fstest.MapFS{"audio.json": {Data: []byte(`{"master":0.5,"music":0.25,"sfx":0}`)}}
	for _, codecs := range []*asset.Codecs{nil, NewAssetCodecs()} {
		game, err := New(WithFilesystem(files), WithAssetCodecs(codecs))
		if err != nil {
			t.Fatal(err)
		}
		levels, err := game.World().MustResource[*asset.Loader]().Load[audio.Preferences]("audio.json")
		if err != nil || levels != (audio.Preferences{Master: 0.5, Music: 0.25}) {
			t.Fatalf("audio preferences=%+v, error=%v", levels, err)
		}
	}
}

func TestAudioPreferencesPersistence(t *testing.T) {
	want := audio.Preferences{Master: 0.5, Music: 0.25}
	loaded := roundTripPreferences(t, want, `{"master":0.5,"music":0.25,"sfx":0}`)
	prefs, err := audio.NewPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if err := prefs.Set(loaded); err != nil {
		t.Fatal(err)
	}
}

func TestPreferenceCodecOverridesAreScoped(t *testing.T) {
	files := fstest.MapFS{"audio.json": {Data: []byte(`{"master":0.5,"music":0.25,"sfx":0}`)}}
	codecs := NewAssetCodecs()
	first, err := New(WithFilesystem(files), WithAssetCodecs(codecs))
	if err != nil {
		t.Fatal(err)
	}
	override := audio.Preferences{Master: 0.75, Music: 0.5, SFX: 0.25}
	if err := codecs.SetDecoder(asset.FormatJSON, func(io.Reader) (audio.Preferences, error) {
		return override, nil
	}); err != nil {
		t.Fatal(err)
	}
	second, err := New(WithFilesystem(files), WithAssetCodecs(codecs))
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := New(WithFilesystem(files))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		game *Game
		want audio.Preferences
	}{
		{first, audio.Preferences{Master: 0.5, Music: 0.25}},
		{second, override},
		{fresh, audio.Preferences{Master: 0.5, Music: 0.25}},
	} {
		got, err := tt.game.World().MustResource[*asset.Loader]().Load[audio.Preferences]("audio.json")
		if err != nil || got != tt.want {
			t.Fatalf("scoped preferences=%+v, error=%v; want %+v", got, err, tt.want)
		}
	}
	bare, err := New(WithFilesystem(files), WithAssetCodecs(&asset.Codecs{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bare.World().MustResource[*asset.Loader]().Load[audio.Preferences]("audio.json"); err == nil {
		t.Fatal("explicit empty codecs were replaced by engine defaults")
	}
	standalone, err := asset.NewLoader(files, asset.LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := standalone.Load[audio.Preferences]("audio.json"); err == nil {
		t.Fatal("engine codecs escaped into asset defaults")
	}
}

func roundTripPreferences[T any](t *testing.T, want T, document string) T {
	t.Helper()
	root := t.TempDir()
	directory, err := asset.NewDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	}()
	codecs := NewAssetCodecs()
	saver, err := asset.NewSaver(directory, asset.SaverOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	if err := saver.Save("preferences.json", want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	var gotJSON, wantJSON any
	if err := json.Unmarshal(data, &gotJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(document), &wantJSON); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Fatalf("saved JSON=%s, want %s", data, document)
	}
	loader, err := asset.NewLoader(directory, asset.LoaderOptions{Codecs: codecs, CachePolicy: asset.CacheBypass})
	if err != nil {
		t.Fatal(err)
	}
	got, err := loader.Load[T]("preferences.json")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restored preferences=%+v, want %+v", got, want)
	}
	return got
}
