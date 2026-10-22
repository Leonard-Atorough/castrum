// Command mixing demonstrates master/group volume and global pause policies.
// Music holds during a global pause; OneShot effects continue through it.
//
// Run from the repository root:
//
//	go run ./examples/audio/mixing
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
	g, err := castrum.New(
		castrum.WithTitle("castrum - audio mixing"),
		castrum.WithFilesystem(audioassets.Files),
	)
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData](audioassets.Magic); err != nil {
		return err
	}
	music := audio.NewSource(audioassets.Music)
	music.Group = audio.GroupMusic
	music.Load = audio.LoadStream
	music.Loop = audio.LoopForever
	music.Pause = audio.PauseHolds
	if _, err := g.World().NewEntity(music); err != nil {
		return err
	}
	mixer := g.World().MustResource[*audio.Mixer]()
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "UP/DOWN: master volume   LEFT/RIGHT: music bus   SPACE: play effect")
	if err := g.AddSystem(core.PhaseFrame, "audio.mixing", mixingSystem(mixer, panel)); err != nil {
		return err
	}
	return g.Run(runner)
}

func mixingSystem(mixer *audio.Mixer, panel *debug.Panel) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		panel.Status("mixer", fmt.Sprintf("master %3.0f%%   music %3.0f%%   sfx %3.0f%%   global pause: %t",
			mixer.Master()*100, mixer.GroupVolume(audio.GroupMusic)*100,
			mixer.GroupVolume(audio.GroupSFX)*100, mixer.Paused()))

		step := volumeSpeed * ctx.DeltaTime.Seconds()
		if ctx.Input.KeyHeld(input.KeyArrowUp) {
			if err := mixer.SetMaster(mixer.Master() + step); err != nil {
				return err
			}
		}
		if ctx.Input.KeyHeld(input.KeyArrowDown) {
			if err := mixer.SetMaster(mixer.Master() - step); err != nil {
				return err
			}
		}
		if ctx.Input.KeyHeld(input.KeyArrowRight) {
			if err := mixer.SetGroupVolume(audio.GroupMusic, mixer.GroupVolume(audio.GroupMusic)+step); err != nil {
				return err
			}
		}
		if ctx.Input.KeyHeld(input.KeyArrowLeft) {
			if err := mixer.SetGroupVolume(audio.GroupMusic, mixer.GroupVolume(audio.GroupMusic)-step); err != nil {
				return err
			}
		}
		if ctx.Input.KeyPressed(input.KeyEnter) {
			mixer.TogglePause()
			panel.Message("global pause toggled - music holds, effects play through")
		}
		if ctx.Input.KeyPressed(input.KeySpace) {
			shot := audio.NewSource(audioassets.Magic)
			shot.Pause = audio.PauseContinues
			_, err := audio.OneShot(ctx.World, shot)
			return err
		}
		return nil
	})
}
