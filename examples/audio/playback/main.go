// Command playback demonstrates eagerly cached OneShots and a persistent,
// streamed music source. Space plays an effect; P pauses the music and R
// restarts it. Each effect owns its playback and is reclaimed on completion.
//
// Run from the repository root:
//
//	go run ./examples/audio/playback
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(
		castrum.WithTitle("castrum - audio playback"),
		castrum.WithFilesystem(audioassets.Files),
	)
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	// Register codecs through the runner before preloading. The first effect
	// then has no decode latency; overlapping plays share cached PCM.
	if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData](audioassets.Magic); err != nil {
		return err
	}
	music := audio.NewSource(audioassets.Music)
	music.Load = audio.LoadStream
	music.Loop = audio.LoopForever
	entity, err := g.World().NewEntity(music)
	if err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFrame, "audio.playback", playbackSystem(entity)); err != nil {
		return err
	}
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		source, ok := entity.Component[audio.Source](ctx.World)
		if !ok {
			return fmt.Errorf("audio playback: music source is missing")
		}
		ebitenutil.DebugPrint(screen, fmt.Sprintf(
			"Audio playback\n\nSPACE: play an eager OneShot (press again to overlap)\n"+
				"P: pause/resume music   R: restart music\n\n"+
				"Music: streamed, looping, source paused: %t\n"+
				"OneShots are reclaimed on completion; the music entity persists.",
			source.Paused))
		return nil
	})
	return g.Run(runner)
}

func playbackSystem(music *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Input.KeyPressed(input.KeySpace) {
			if _, err := audio.OneShot(ctx.World, audio.NewSource(audioassets.Magic)); err != nil {
				return err
			}
		}
		if !ctx.Input.KeyPressed(input.KeyP) && !ctx.Input.KeyPressed(input.KeyR) {
			return nil
		}
		return music.Update(ctx.World, func(s *audio.Source) {
			if ctx.Input.KeyPressed(input.KeyP) {
				s.Paused = !s.Paused
			}
			if ctx.Input.KeyPressed(input.KeyR) {
				s.Restart()
			}
		})
	})
}
