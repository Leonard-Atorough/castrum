// Command playback demonstrates eagerly cached OneShots and a persistent,
// streamed music source. Space plays an effect; P pauses the music and R
// restarts it. Each effect owns its playback and is reclaimed on completion.
//
// Run from this directory:
//
//	go run .
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
	g := castrum.New(
		castrum.WithTitle("castrum - audio playback"),
		castrum.WithFilesystem(audioassets.Files),
	)
	runner := ebitrun.New(ebitrun.WithDebugOverlay())
	// Register codecs through the runner before preloading. The first effect
	// then has no decode latency; overlapping plays share cached PCM.
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData](audioassets.Magic); err != nil {
		return err
	}
	music, err := audio.NewPlayer(audio.Track(audioassets.Music), audio.Settings{
		Volume: 1, Bus: audio.BusName("music"), Load: audio.LoadStream, Loop: audio.LoopTrack,
	})
	if err != nil {
		return err
	}
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
	runner.Run(g)
	return nil
}

func playbackSystem(music core.Entity, panel *debug.Panel) core.SystemFunc {
	return func(ctx *core.Context) error {
		current, _ := ctx.World.Resource[*input.Snapshot]()
		if current.KeyPressed(input.KeySpace) {
			if _, err := audio.OneShot(ctx.World, audio.Track(audioassets.Magic), audio.DefaultSettings()); err != nil {
				return err
			}
			panel.Message("OneShot playing")
		}
		if !current.KeyPressed(input.KeyP) && !current.KeyPressed(input.KeyR) {
			return nil
		}
		return music.Update(ctx.World, func(s *audio.Player) {
			if current.KeyPressed(input.KeyP) {
				s.Paused = !s.Paused
				panel.Message(fmt.Sprintf("music paused: %t", s.Paused))
			}
			if current.KeyPressed(input.KeyR) {
				s.Restart()
				panel.Message("music restarted")
			}
		})
	}
}
