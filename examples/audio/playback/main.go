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
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
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
	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	// Register codecs through the runner before preloading. The first effect
	// then has no decode latency; overlapping plays share cached PCM.
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData](audioassets.Magic); err != nil {
		return err
	}
	music := audio.NewSource(audioassets.Music)
	music.Load = audio.LoadStream
	music.Loop = audio.LoopForever
	entity, err := g.World().NewEntity(music)
	if err != nil {
		return err
	}
	panel := g.World().MustResource[*debug.Panel]()
	panel.Message("SPACE: play an eager OneShot   P: pause/resume music   R: restart music")
	panel.Message("OneShots are reclaimed on completion; the music entity persists")
	if err := g.AddSystem(core.PhaseFrame, "audio.playback", playbackSystem(entity, panel)); err != nil {
		return err
	}
	return g.Run(runner)
}

func playbackSystem(music *core.Entity, panel *debug.Panel) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Input.KeyPressed(input.KeySpace) {
			if _, err := audio.OneShot(ctx.World, audio.NewSource(audioassets.Magic)); err != nil {
				return err
			}
			panel.Message("OneShot playing")
		}
		if !ctx.Input.KeyPressed(input.KeyP) && !ctx.Input.KeyPressed(input.KeyR) {
			return nil
		}
		return music.Update(ctx.World, func(s *audio.Source) {
			if ctx.Input.KeyPressed(input.KeyP) {
				s.Paused = !s.Paused
				panel.Message(fmt.Sprintf("music paused: %t", s.Paused))
			}
			if ctx.Input.KeyPressed(input.KeyR) {
				s.Restart()
				panel.Message("music restarted")
			}
		})
	})
}
