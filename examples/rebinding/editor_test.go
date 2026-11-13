package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/input"
)

func openDirectory(t *testing.T, root string) *asset.Directory {
	t.Helper()
	directory, err := asset.NewDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	})
	return directory
}

func TestDraftCancelConfirmAndRestore(t *testing.T) {
	root := t.TempDir()
	directory := openDirectory(t, root)
	g, e, err := newGame(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	press := func(key input.Key) {
		t.Helper()
		snapshot := &input.Snapshot{}
		snapshot.Keys[key] = input.State{Pressed: true, Held: true}
		g.Context().Input = snapshot
		if err := g.Advance(time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	press(input.KeyF2)
	press(input.KeyEnter)
	if !e.editing || !strings.Contains(e.notice, "Choose") {
		t.Fatal("confirmation without a candidate was not reported")
	}
	press(input.KeyJ)
	if g.Context().Actions.Bindings()["ping"][0].Key.Key != input.KeySpace {
		t.Fatal("draft changed live controls")
	}
	press(input.KeyEscape)
	if _, err := os.Stat(filepath.Join(root, controlsFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("cancel wrote a file: %v", err)
	}
	press(input.KeySpace)
	if e.activations != 1 {
		t.Fatal("cancel changed active controls")
	}
	press(input.KeyF2)
	press(input.KeyJ)
	press(input.KeyEnter)
	press(input.KeySpace)
	if e.activations != 1 {
		t.Fatal("old binding still active after confirmation")
	}
	press(input.KeyJ)
	if e.activations != 2 {
		t.Fatal("confirmed binding was not applied")
	}
	before, err := os.ReadFile(filepath.Join(root, controlsFile))
	if err != nil {
		t.Fatal(err)
	}
	press(input.KeyF2)
	press(input.KeyK)
	press(input.KeyEscape)
	after, err := os.ReadFile(filepath.Join(root, controlsFile))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("cancel replaced the saved document")
	}
	restored, _, err := newGame(directory)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Context().Actions.Bindings()["ping"][0].Key.Key != input.KeyJ {
		t.Fatal("confirmed controls were not restored on startup")
	}
}

func TestInvalidDocumentsAreNotReplaced(t *testing.T) {
	for _, document := range []string{`{`, `{}`, `{"bindings":{},"deadzone":2}`} {
		t.Run(document, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, controlsFile)
			if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := newGame(openDirectory(t, root)); err == nil {
				t.Fatal("invalid document accepted")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != document {
				t.Fatal("invalid document was replaced")
			}
		})
	}
}

func TestRebindingPreservesRestoredDeadzoneAndOtherActions(t *testing.T) {
	directory := openDirectory(t, t.TempDir())
	saver, err := asset.NewSaver(directory, asset.SaverOptions{Codecs: castrum.NewAssetCodecs()})
	if err != nil {
		t.Fatal(err)
	}
	original := input.Preferences{
		Bindings: input.Bindings{
			"ping":  {input.KeyInput{Key: input.KeySpace}.Binding()},
			"other": {input.KeyInput{Key: input.KeyA}.Binding()},
		},
		Deadzone: 0.4,
	}
	if err := saver.Save(controlsFile, original); err != nil {
		t.Fatal(err)
	}
	g, _, err := newGame(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	for _, key := range []input.Key{input.KeyF2, input.KeyJ, input.KeyEnter} {
		g.Context().Input = &input.Snapshot{}
		g.Context().Input.Keys[key] = input.State{Pressed: true, Held: true}
		if err := g.Advance(time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	restored, _, err := newGame(directory)
	if err != nil {
		t.Fatal(err)
	}
	got := restored.Context().Actions.Preferences()
	if got.Deadzone != 0.4 || got.Bindings["other"][0].Key.Key != input.KeyA || got.Bindings["ping"][0].Key.Key != input.KeyJ {
		t.Fatalf("rebind lost unrelated preferences: %+v", got)
	}
}

type failingStore struct{}

func (failingStore) Write(string, func(io.Writer) error) error {
	return errors.New("disk unavailable")
}

func TestSaveFailureIsReturnedAfterExplicitApplication(t *testing.T) {
	g, e, err := newGame(openDirectory(t, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	e.saver, err = asset.NewSaver(failingStore{}, asset.SaverOptions{Codecs: castrum.NewAssetCodecs()})
	if err != nil {
		t.Fatal(err)
	}
	value, err := e.prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	value.Bindings["ping"] = []input.Input{input.KeyInput{Key: input.KeyJ}.Binding()}
	e.draft, e.candidate, e.editing = value, input.KeyJ, true
	ctx := g.Context()
	ctx.Input = &input.Snapshot{}
	ctx.Input.Keys[input.KeyEnter].Pressed = true
	if err := e.update(ctx); err == nil || !strings.Contains(err.Error(), "live controls changed but saving failed") {
		t.Fatalf("save failure=%v", err)
	}
	if ctx.Actions.Bindings()["ping"][0].Key.Key != input.KeyJ {
		t.Fatal("application did not precede persistence")
	}
}
