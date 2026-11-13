package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
)

func TestAuthoredSheetMatchesShippedDocument(t *testing.T) {
	root := filepath.Join(t.TempDir(), "authored")
	name := filepath.Join(root, "torch.data")
	if err := authorSheet(name); err != nil {
		t.Fatal(err)
	}
	directory, err := asset.NewDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := directory.Close(); err != nil {
			t.Error(err)
		}
	}()
	loader, err := asset.NewLoader(directory, asset.LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	authored, err := asset.FileSheet("torch.data").Resolve(loader)
	if err != nil {
		t.Fatal(err)
	}
	content, err := asset.NewLoader(files, asset.LoaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	shipped, err := asset.FileSheet(sheetPath).Resolve(content)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(authored, shipped) {
		t.Fatalf("authored sheet=%+v, want shipped %+v", authored, shipped)
	}
	texture, err := content.Load[asset.TextureData](string(authored.Texture))
	if err != nil {
		t.Fatal(err)
	}
	if err := authored.ValidateBounds(texture.Image.Bounds()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "torch.data" {
		t.Fatalf("authoring should save only metadata: %v", entries)
	}
}

func TestAuthorSheetRejectsUnusableDestination(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, []byte("keep this file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := authorSheet(filepath.Join(parent, "torch.json")); err == nil {
		t.Fatal("authoring reported success for a file used as the parent directory")
	}
	data, err := os.ReadFile(parent)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep this file" {
		t.Fatalf("authoring replaced the existing file: %q", data)
	}
}

func TestAnimationControlsAndCompletion(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := core.NewQuery(g.World()).With(animation.Animation{}, render.Sprite{}).First()
	if !ok {
		t.Fatal("setup did not spawn an animated sprite")
	}
	torch := core.NewEntity(entry.ID())
	sprite, _ := torch.Component[render.Sprite](g.World())
	if sprite.Drawable != (render.SheetSource{Sheet: asset.FileSheet(sheetPath), Region: "torch_light_0"}) {
		t.Fatalf("initial drawable=%+v, want file-backed first frame", sprite.Drawable)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	read := func() animation.Animation {
		t.Helper()
		value, ok := torch.Component[animation.Animation](g.World())
		if !ok {
			t.Fatal("animated sprite lost its playback component")
		}
		return value
	}
	tick := func(count int) {
		t.Helper()
		g.Context().Input = nil
		for range count {
			if err := g.Advance(g.Options().FixedDT()); err != nil {
				t.Fatal(err)
			}
		}
	}
	press := func(key input.Key) {
		t.Helper()
		snapshot := &input.Snapshot{}
		snapshot.Keys[key].Pressed = true
		g.Context().Input = snapshot
		if err := g.Advance(0); err != nil {
			t.Fatal(err)
		}
	}
	tick(12)
	before := read()
	if before.Current == 0 {
		t.Fatal("looping animation did not advance")
	}
	press(input.KeySpace)
	tick(12)
	paused := read()
	if !paused.Paused || paused.Current != before.Current || paused.Elapsed != before.Elapsed {
		t.Fatalf("pause changed the playhead: before=%+v after=%+v", before, paused)
	}
	press(input.KeySpace)
	tick(12)
	if got := read(); got.Paused || got.Current == paused.Current {
		t.Fatalf("resume did not advance playback: %+v", got)
	}

	press(input.Key1)
	if got := read(); got.Loop != animation.LoopNone || got.Current != 0 || got.Paused {
		t.Fatalf("play-once control did not restart: %+v", got)
	}
	for count := 0; count < 60 && !read().HasCompleted(); count++ {
		tick(1)
	}
	completed := read()
	if !completed.HasCompleted() || !completed.JustCompleted(g.Context().Tick) || !completed.Paused || completed.Current != 5 {
		t.Fatalf("one-shot completion=%+v, want paused final frame and completion edge", completed)
	}
	panel := g.World().MustResource[*debug.Panel]()
	if !slices.Contains(panel.Lines(), "finished - R restarts playback") {
		t.Fatal("completion did not reach the example readout")
	}
	tick(12)
	if got := read(); got.Current != completed.Current || got.CompletedOn != completed.CompletedOn {
		t.Fatalf("completed animation did not hold: %+v", got)
	}
	sprite, _ = torch.Component[render.Sprite](g.World())
	if sprite.Drawable != (render.SheetSource{Sheet: asset.FileSheet(sheetPath), Region: "torch_light_5"}) {
		t.Fatalf("completed drawable=%+v, want file-backed final frame", sprite.Drawable)
	}
	press(input.KeyR)
	if got := read(); got.HasCompleted() || got.Paused || got.Current != 0 || got.Elapsed != 0 {
		t.Fatalf("restart did not clear completion: %+v", got)
	}
	press(input.Key2)
	tick(60)
	if got := read(); got.Loop != animation.LoopForever || got.Paused || got.HasCompleted() || got.CompletedOn == 0 {
		t.Fatalf("loop mode did not wrap: %+v", got)
	}
	if !slices.Contains(panel.Lines(), "loop completed") {
		t.Fatal("loop wrap did not reach the example readout")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("normal animation startup/playback wrote files: %v", entries)
	}
}
