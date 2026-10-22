// Command balance demonstrates live per-source stereo balance. Balance can
// change while paused and survives a restart; it attenuates a channel rather
// than moving its content into the other channel.
//
// Run from the repository root:
//
//	go run ./examples/audio/balance
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
	g, err := castrum.New(
		castrum.WithTitle("castrum - audio balance"),
		castrum.WithFilesystem(audioassets.Files),
	)
	if err != nil {
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
	panel.Status("controls", "LEFT/RIGHT: adjust balance   1: left   2: center   3: right")
	panel.Status("note", "SPACE: pause/resume   R: restart (keeps balance, resumes playback); center preserves the mix")
	if err := g.AddSystem(core.PhaseFrame, "audio.balance", balanceSystem(entity, panel)); err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	return g.Run(runner)
}

func balanceSystem(music *core.Entity, panel *debug.Panel) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		return music.Update(ctx.World, func(s *audio.Source) {
			step := balanceSpeed * ctx.DeltaTime.Seconds()
			if ctx.Input.KeyHeld(input.KeyArrowLeft) {
				s.Balance -= step
			}
			if ctx.Input.KeyHeld(input.KeyArrowRight) {
				s.Balance += step
			}
			// Source validation rejects out-of-range values rather than
			// clamping them, so the interactive control supplies that bound.
			s.Balance = math.Max(-1, math.Min(1, s.Balance))
			if ctx.Input.KeyPressed(input.Key1) {
				s.Balance = -1
			}
			if ctx.Input.KeyPressed(input.Key2) {
				s.Balance = 0
			}
			if ctx.Input.KeyPressed(input.Key3) {
				s.Balance = 1
			}

			left := 1 - math.Max(s.Balance, 0)
			right := 1 + math.Min(s.Balance, 0)
			panel.Status("balance", fmt.Sprintf("balance %+.2f   left %3.0f%%   right %3.0f%%   paused: %t",
				s.Balance, left*100, right*100, s.Paused))
			if ctx.Input.KeyPressed(input.KeySpace) {
				s.Paused = !s.Paused
				panel.Message(fmt.Sprintf("paused: %t", s.Paused))
			}
			if ctx.Input.KeyPressed(input.KeyR) {
				s.Restart()
				panel.Message(fmt.Sprintf("restarted, balance %+.2f kept", s.Balance))
			}
		})
	})
}
