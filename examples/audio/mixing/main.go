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
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData](audioassets.Magic); err != nil {
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
	if err := g.AddSystem(core.PhaseFrame, "audio.mixing", mixingSystem(mixer)); err != nil {
		return err
	}
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrint(screen, fmt.Sprintf(
			"Audio mixing\n\nUP/DOWN: master volume   LEFT/RIGHT: music bus\n"+
				"SPACE: play effect   ENTER: global pause/resume\n\n"+
				"Master: %3.0f%%   Music: %3.0f%%   SFX: %3.0f%%   Global pause: %t\n"+
				"Audible level = master x group x source volume.\n"+
				"On global pause, music holds; effects play through.",
			mixer.Master()*100, mixer.GroupVolume(audio.GroupMusic)*100,
			mixer.GroupVolume(audio.GroupSFX)*100, mixer.Paused()))
		return nil
	})
	return g.Run(runner)
}

func mixingSystem(mixer *audio.Mixer) core.System {
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
		if ctx.Input.KeyPressed(input.KeySpace) {
			shot := audio.NewSource(audioassets.Magic)
			shot.Pause = audio.PauseContinues
			_, err := audio.OneShot(ctx.World, shot)
			return err
		}
		return nil
	})
}
