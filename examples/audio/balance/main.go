// Command balance demonstrates live per-source stereo balance. Balance can
// change while paused and survives a restart; it attenuates a channel rather
// than moving its content into the other channel.
//
// Run from this directory:
//
//	go run .
package main

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
)

const balanceSpeed = 0.5

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g := castrum.New(
		castrum.WithTitle("castrum - audio balance"),
		castrum.WithFilesystem(audioassets.Files),
	)

	p, err := audio.NewPlayer(audio.Track(audioassets.Music), audio.Settings{Load: audio.LoadStream, Loop: audio.LoopTrack, Volume: 1})
	if err != nil {
		return err
	}
	entity, err := g.World().NewEntity(p)
	if err != nil {
		return err
	}
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "LEFT/RIGHT: adjust balance   1: left   2: center   3: right")
	panel.Status("note", "SPACE: pause/resume   R: restart (keeps balance, resumes playback); center preserves the mix")
	if err := g.AddSystem(core.PhaseFrame, "audio.balance", balanceSystem(entity, panel)); err != nil {
		return err
	}
	runner := ebitrun.New(ebitrun.WithDebugOverlay())
	runner.Run(g)
	return nil
}

func balanceSystem(music core.Entity, panel *debug.Panel) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		current, _ := ctx.World.Resource[*input.Snapshot]()
		return music.Update(ctx.World, func(s *audio.Player) {
			step := balanceSpeed * ctx.DeltaTime.Seconds()
			if current.KeyHeld(input.KeyArrowLeft) {
				s.Settings.Balance -= step
			}
			if current.KeyHeld(input.KeyArrowRight) {
				s.Settings.Balance += step
			}
			// Source validation rejects out-of-range values rather than
			// clamping them, so the interactive control supplies that bound.
			s.Settings.Balance = math.Max(-1, math.Min(1, s.Settings.Balance))
			if current.KeyPressed(input.Key1) {
				s.Settings.Balance = -1
			}
			if current.KeyPressed(input.Key2) {
				s.Settings.Balance = 0
			}
			if current.KeyPressed(input.Key3) {
				s.Settings.Balance = 1
			}

			left := math.Sqrt(1 - math.Max(s.Settings.Balance, 0))
			right := math.Sqrt(1 - math.Max(-s.Settings.Balance, 0))
			panel.Status("balance", fmt.Sprintf("balance %+.2f   left %3.0f%%   right %3.0f%%   paused: %t",
				s.Settings.Balance, left*100, right*100, s.Paused))
			if current.KeyPressed(input.KeySpace) {
				s.Paused = !s.Paused
				panel.Message(fmt.Sprintf("paused: %t", s.Paused))
			}
			if current.KeyPressed(input.KeyR) {
				s.Restart()
				panel.Message(fmt.Sprintf("restarted, balance %+.2f kept", s.Settings.Balance))
			}
		})
	})
}
