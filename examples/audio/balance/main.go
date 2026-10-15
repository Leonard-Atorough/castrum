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
	"github.com/Leonard-Atorough/castrum/ebitrun"
	audioassets "github.com/Leonard-Atorough/castrum/examples/audio"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	if err := g.AddSystem(core.PhaseFrame, "audio.balance", balanceSystem(entity)); err != nil {
		return err
	}
	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		source, ok := entity.Component[audio.Source](ctx.World)
		if !ok {
			return fmt.Errorf("audio balance: music source is missing")
		}
		left := 1 - math.Max(source.Balance, 0)
		right := 1 + math.Min(source.Balance, 0)
		ebitenutil.DebugPrint(screen, fmt.Sprintf(
			"Audio balance - use headphones or stereo speakers\n\n"+
				"LEFT/RIGHT: adjust balance   1: left   2: center   3: right\n"+
				"SPACE: pause/resume   R: restart (keeps balance, resumes playback)\n\n"+
				"Balance: %+.2f   Left gain: %3.0f%%   Right gain: %3.0f%%   Paused: %t\n\n"+
				"Center preserves the original stereo mix. Endpoints mute one channel.\n"+
				"Try changing balance while paused, then resume or restart.\n"+
				"Buffered audio can delay changes; abrupt changes may click.",
			source.Balance, left*100, right*100, source.Paused))
		return nil
	})
	return g.Run(runner)
}

func balanceSystem(music *core.Entity) core.System {
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
			if ctx.Input.KeyPressed(input.KeySpace) {
				s.Paused = !s.Paused
			}
			if ctx.Input.KeyPressed(input.KeyR) {
				s.Restart()
			}
		})
	})
}
