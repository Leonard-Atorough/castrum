// Command audio demonstrates audio playback end to end: SPACE plays
// a sound effect (eager, decoded once and shared), a music track
// loops from the file (streamed, never fully in memory), the arrow
// keys sweep the master and music volumes, and ENTER toggles the
// global pause - the music holds while the effect plays through,
// because each play states its own pause behavior.
//
// Assets are embedded into the binary - the self-contained pattern
// this example doubles as a demo of - passed via castrum.WithFilesystem.
// Run from the repository root (or anywhere):
//
//	go run ./examples/audio
package main

import (
	"embed"
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed "Blade Recall Magic 02.wav" arpmedia-retro-arcade-game-music-577821.mp3
var files embed.FS

// AssetMagic is the sound the spacebar plays. The zero load mode is
// eager: it decodes once into the asset cache and every press shares
// those bytes.
const AssetMagic = "Blade Recall Magic 02.wav"

// AssetMusic is the background track. Unlike the sound effect it
// streams: one open reader plays straight from the file, so the
// decoded minutes of music never sit in memory. Streaming is the
// explicit opt-in - the zero load mode is eager.
const AssetMusic = "arpmedia-retro-arcade-game-music-577821.mp3"

// volumeSpeed is how fast the arrow keys sweep the volume levels.
const volumeSpeed = 0.5

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(
		castrum.WithTitle("castrum - audio"),
		castrum.WithFilesystem(files),
	)
	if err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	// Decode the sound effect now, not on the first press: a missing
	// or unreadable file fails before the window opens, and the first
	// press has no decode latency. The music needs no preload - it
	// never fully decodes.
	if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData](AssetMagic); err != nil {
		return err
	}

	// The music entity: a looping, streaming Source on the music bus.
	// The engine reconciles it into a player on the first frame; no
	// runner API call plays it.
	if _, err := g.World().NewEntity(audio.Source{
		Audio:  AssetMusic,
		Volume: 1,
		Group:  audio.GroupMusic,
		Loop:   audio.LoopForever,
		Load:   audio.LoadStream,
		Pause:  audio.PauseHolds,
	}); err != nil {
		return err
	}

	mixer := g.World().MustResource[*audio.Mixer]()
	if err := g.AddSystem(core.PhaseFrame, "audio.spacebar", spacebarSystem()); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFrame, "audio.volume", volumeSystem(mixer)); err != nil {
		return err
	}

	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		state := "playing"
		if mixer.Paused() {
			state = "PAUSED (music holds, the sound effect plays through)"
		}
		ebitenutil.DebugPrint(screen,
			"castrum audio - SPACE plays the sound, UP/DOWN master volume, LEFT/RIGHT music volume, ENTER pause/play\n"+
				fmt.Sprintf("master %d%%   music %d%%   sfx %d%%\n%s\nFPS: %.2f",
					int(mixer.Master()*100), int(mixer.GroupVolume(audio.GroupMusic)*100),
					int(mixer.GroupVolume(audio.GroupSFX)*100), state, ebiten.ActualFPS()))
		return nil
	})

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}

// spacebarSystem plays the sound on each press of the spacebar: raw
// snapshot input - one key needs no bindings - and a fire-and-forget
// one-shot the engine reclaims when the play finishes. Overlapping
// presses layer, each press is its own entity.
func spacebarSystem() core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Input.KeyPressed(input.KeySpace) {
			return nil
		}
		// NewSource carries the unity default a literal would have
		// to spell out; the field writes cover the rest.
		shot := audio.NewSource(AssetMagic)
		shot.Pause = audio.PauseContinues
		_, err := audio.OneShot(ctx.World, shot)
		return err
	})
}

// volumeSystem drives the two buses with the arrow keys. Every
// play's audible level is the product master x group x play, so the
// effect bus can stay loud while the music ducks, and the setters
// clamp - no range checks here.
func volumeSystem(mixer *audio.Mixer) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		step := volumeSpeed * ctx.DeltaTime.Seconds()

		if ctx.Input.KeyHeld(input.KeyArrowUp) {
			mixer.SetMaster(mixer.Master() + step)
		}
		if ctx.Input.KeyHeld(input.KeyArrowDown) {
			mixer.SetMaster(mixer.Master() - step)
		}
		if ctx.Input.KeyHeld(input.KeyArrowRight) {
			mixer.SetGroupVolume(audio.GroupMusic, mixer.GroupVolume(audio.GroupMusic)+step)
		}
		if ctx.Input.KeyHeld(input.KeyArrowLeft) {
			mixer.SetGroupVolume(audio.GroupMusic, mixer.GroupVolume(audio.GroupMusic)-step)
		}
		if ctx.Input.KeyPressed(input.KeyEnter) {
			mixer.TogglePause()
		}
		return nil
	})
}

// The assets' attribution lives in CREDITS.md at the repository root.
