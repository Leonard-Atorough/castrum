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
	entry, ok := core.NewQuery(g.World()).With(animation.AnimatedSprite{}).First()
	if !ok {
		t.Fatal("setup did not spawn an animated sprite")
	}
	torch := entry.Entity()
	read := func() animation.AnimatedSprite {
		t.Helper()
		value, ok := torch.Component[animation.AnimatedSprite](g.World())
		if !ok {
			t.Fatal("animated sprite lost its playback component")
		}
		return value
	}
	regionOf := func(value animation.AnimatedSprite) string {
		t.Helper()
		frame, ok, err := value.CurrentFrame()
		if err != nil || !ok {
			t.Fatalf("CurrentFrame = %v, %v, %v", frame, ok, err)
		}
		return string(frame.Region)
	}
	if got := regionOf(read()); got != "torch_light_0" {
		t.Fatalf("initial frame=%s, want the file-backed first frame", got)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
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
	before := regionOf(read())
	if before == "torch_light_0" {
		t.Fatal("looping animation did not advance")
	}
	press(input.KeySpace)
	tick(12)
	paused := read()
	if !paused.Paused || regionOf(paused) != before {
		t.Fatalf("pause changed the playhead: before=%s after=%s", before, regionOf(paused))
	}
	press(input.KeySpace)
	tick(12)
	resumed := read()
	if resumed.Paused || regionOf(resumed) == before {
		t.Fatalf("resume did not advance playback: %s", regionOf(resumed))
	}

	press(input.Key1)
	if got := read(); got.Loop != animation.LoopNone || got.Paused || regionOf(got) != "torch_light_0" {
		t.Fatalf("play-once control did not restart: frame=%s", regionOf(got))
	}
	for count := 0; count < 60 && !read().HasCompleted(); count++ {
		tick(1)
	}
	completed := read()
	if !completed.HasCompleted() || !completed.JustCompleted(g.Context().Tick) || !completed.Paused || regionOf(completed) != "torch_light_5" {
		t.Fatalf("one-shot completion=%s, want paused final frame and completion edge", regionOf(completed))
	}
	panel := g.World().MustResource[*debug.Panel]()
	if !slices.Contains(panel.Lines(), "finished - R restarts playback") {
		t.Fatal("completion did not reach the example readout")
	}
	tick(12)
	held := read()
	if regionOf(held) != "torch_light_5" || !held.Paused || !held.HasCompleted() {
		t.Fatalf("completed animation did not hold: frame=%s", regionOf(held))
	}
	press(input.KeyR)
	if got := read(); got.HasCompleted() || got.Paused || regionOf(got) != "torch_light_0" {
		t.Fatalf("restart did not clear completion: frame=%s", regionOf(got))
	}
	press(input.Key2)
	tick(60)
	looped := read()
	if looped.Loop != animation.LoopAll || looped.Paused || looped.HasCompleted() {
		t.Fatalf("loop mode did not resume: frame=%s", regionOf(looped))
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
