package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
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
	texture, err := content.Load[asset.TextureData](texturePath)
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

// TestControlsDrivePlaybackComponent validates the example's own control
// system: each bound key maps to the intended mutation of the AnimatedSprite
// component. How the engine advances or completes clips is engine behavior,
// covered by the animation package's own tests.
func TestControlsDrivePlaybackComponent(t *testing.T) {
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
		_, region, ok, err := value.CurrentFrame()
		if err != nil || !ok {
			t.Fatalf("CurrentFrame = %v, %v, %v", region, ok, err)
		}
		return region
	}
	if got := regionOf(read()); got != frames[0] {
		t.Fatalf("initial frame=%s, want the clip's first frame %s", got, frames[0])
	}
	if got := read().Loop; got != animation.LoopAll {
		t.Fatalf("initial loop=%v, want LoopAll", got)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
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

	press(input.KeySpace)
	if got := read(); !got.Paused {
		t.Fatal("SPACE did not pause playback")
	}
	press(input.KeySpace)
	if got := read(); got.Paused {
		t.Fatal("second SPACE did not resume playback")
	}
	press(input.Key1)
	if got := read(); got.Loop != animation.LoopNone || regionOf(got) != frames[0] {
		t.Fatalf("play-once: loop=%v frame=%s, want LoopNone restarted at %s", got.Loop, regionOf(got), frames[0])
	}
	press(input.Key2)
	if got := read(); got.Loop != animation.LoopAll || regionOf(got) != frames[0] {
		t.Fatalf("loop: loop=%v frame=%s, want LoopAll restarted at %s", got.Loop, regionOf(got), frames[0])
	}
	press(input.KeyR)
	if got := read(); got.Paused || regionOf(got) != frames[0] {
		t.Fatalf("restart: frame=%s paused=%t, want %s unpaused", regionOf(got), got.Paused, frames[0])
	}
	// Swap selects the reversed clip, whose first frame is the original last.
	press(input.Key4)
	if got := regionOf(read()); got != frames[len(frames)-1] {
		t.Fatalf("swap: frame=%s, want the reversed clip's first frame %s", got, frames[len(frames)-1])
	}

	// The readout system is registered and produces a playback status line.
	g.Context().Input = nil
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	panel := g.World().MustResource[*debug.Panel]()
	if !slices.ContainsFunc(panel.StatusLines(), func(line string) bool {
		return strings.HasPrefix(line, "frame ")
	}) {
		t.Fatalf("readout system produced no playback status: %v", panel.StatusLines())
	}
}
