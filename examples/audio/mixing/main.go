// Command mixing demonstrates master/group volume and global pause policies.
// Music holds during a global pause; OneShot effects continue through it.
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

const volumeSpeed = 0.5

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g := castrum.New(
		castrum.WithTitle("castrum - audio mixing"),
		castrum.WithFilesystem(audioassets.Files),
	)
	runner := ebitrun.New(ebitrun.WithDebugOverlay())
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData](audioassets.Magic); err != nil {
		return err
	}
	p, err := audio.NewPlayer(
		audio.Track(audioassets.Music),
		audio.Settings{Load: audio.LoadStream, Loop: audio.LoopTrack, Bus: audio.BusName("music"), Pause: audio.PauseHolds, Volume: 1},
	)
	if err != nil {
		return err
	}
	if _, err := g.World().NewEntity(p); err != nil {
		return err
	}
	mixer := g.World().MustResource[*audio.Mixer]()
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "UP/DOWN: master volume   LEFT/RIGHT: music bus   ENTER: pause   SPACE: play effect")
	if err := g.AddSystem(core.PhaseFrame, "audio.mixing", mixingSystem(mixer, panel)); err != nil {
		return err
	}
	runner.Run(g)
	return nil
}

func mixingSystem(mixer *audio.Mixer, panel *debug.Panel) core.SystemFunc {
	return func(ctx *core.Context) error {
		current := ctx.World.MustResource[*input.Snapshot]()
		panel.Status("mixer", fmt.Sprintf("master %3.0f%%   music %3.0f%%   sfx %3.0f%%   global pause: %t",
			mixer.MasterVolume()*100, mixer.BusVolume("music")*100,
			mixer.BusVolume("sfx")*100, mixer.AllPaused()))

		step := volumeSpeed * ctx.DeltaTime.Seconds()
		if current.KeyHeld(input.KeyArrowUp) {
			if err := mixer.SetMasterVolume(mixer.MasterVolume() + step); err != nil {
				return err
			}
		}
		if current.KeyHeld(input.KeyArrowDown) {
			if err := mixer.SetMasterVolume(mixer.MasterVolume() - step); err != nil {
				return err
			}
		}
		if current.KeyHeld(input.KeyArrowRight) {
			if err := mixer.SetBusVolume(audio.BusName("music"), mixer.BusVolume(audio.BusName("music"))+step); err != nil {
				return err
			}
		}
		if current.KeyHeld(input.KeyArrowLeft) {
			if err := mixer.SetBusVolume(audio.BusName("music"), mixer.BusVolume(audio.BusName("music"))-step); err != nil {
				return err
			}
		}
		if current.KeyPressed(input.KeyEnter) {
			mixer.ToggleAllPaused()
			panel.Message("global pause toggled - music holds, effects play through")
		}
		if current.KeyPressed(input.KeySpace) {
			_, err := audio.OneShot(ctx.World, audio.Track(audioassets.Magic), audio.Settings{Pause: audio.PauseContinues, Volume: 1})
			return err
		}
		return nil
	}
}
