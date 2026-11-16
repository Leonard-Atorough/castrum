// Command animate loads an embedded JSON spritesheet and demonstrates
// playback controls and completion. Run with go run .; -author writes
// regenerated metadata without opening a window.
package main

import (
	"embed"
	"flag"
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
)

//go:embed torch_light.png torch_light.sheet.json
var files embed.FS

const (
	sheetPath = "torch_light.sheet.json"

	TorchLightFPS  = 10
	TorchLightLoop = animation.LoopAll

	ClipFlickeringTorch = "flickering_torch"

	Scale = 4
)

func main() {
	output := flag.String("author", "", "write spritesheet metadata to this file without opening a window")
	flag.Parse()
	if *output != "" {
		if err := authorSheet(*output); err != nil {
			panic(err)
		}
		fmt.Printf("Saved spritesheet to %s\n", *output)
		return
	}
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := newGame()
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	return g.Run(runner)
}

func newGame() (*castrum.Game, error) {
	g, err := castrum.New(
		castrum.WithTitle("castrum - animate"),
		castrum.WithFilesystem(files),
		castrum.WithBindings(input.Bindings{
			"pause":   {input.KeyInput{Key: input.KeySpace}.Binding()},
			"restart": {input.KeyInput{Key: input.KeyR}.Binding()},
			"once":    {input.KeyInput{Key: input.Key1}.Binding()},
			"loop":    {input.KeyInput{Key: input.Key2}.Binding()},
		}),
	)
	if err != nil {
		return nil, err
	}

	loader := g.World().MustResource[*asset.Loader]()
	source := asset.FileSheet(sheetPath)
	sheet, err := source.Resolve(loader)
	if err != nil {
		return nil, err
	}
	texture, err := loader.Load[asset.TextureData](string(sheet.Texture))
	if err != nil {
		return nil, err
	}
	if err := sheet.ValidateBounds(texture.Image.Bounds()); err != nil {
		return nil, err
	}

	frames := []string{
		"torch_light_0",
		"torch_light_1",
		"torch_light_2",
		"torch_light_3",
		"torch_light_4",
		"torch_light_5",
	}
	for _, name := range frames {
		if _, err := sheet.Region(name); err != nil {
			return nil, err
		}
	}
	clip, err := animation.NewSpriteClip(source, frames, TorchLightFPS)
	if err != nil {
		return nil, err
	}

	torchAnimated, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		return nil, err
	}
	torchAnimated.Loop = TorchLightLoop

	torch, err := g.World().NewEntity(
		torchAnimated,
		core.Transform{
			Scale: geom.Vector2{X: Scale, Y: Scale},
		},
	)
	if err != nil {
		return nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "animation.controls", playbackSystem(torch)); err != nil {
		return nil, err
	}
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "SPACE: pause/resume   R: restart   1: play once   2: loop")
	if err := g.AddSystem(core.PhaseFixed, "animation.readout", core.SystemFunc(func(ctx *core.Context) error {
		value, ok := torch.Component[animation.AnimatedSprite](ctx.World)
		if !ok {
			return fmt.Errorf("animated torch has no playback component")
		}
		frame, _, err := value.CurrentFrame()
		if err != nil {
			return err
		}
		panel.Status("playback", fmt.Sprintf("frame %s   paused: %t   completed: %t", frame.Region, value.Paused, value.HasCompleted()))
		if value.JustCompleted(ctx.Tick) {
			if value.HasCompleted() {
				panel.Message("finished - R restarts playback")
			} else {
				panel.Message("loop completed")
			}
		}
		return nil
	})); err != nil {
		return nil, err
	}
	return g, nil
}

func playbackSystem(torch *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		pause := ctx.Actions.JustPressed("pause")
		restart := ctx.Actions.JustPressed("restart")
		once := ctx.Actions.JustPressed("once")
		loop := ctx.Actions.JustPressed("loop")
		if !pause && !restart && !once && !loop {
			return nil
		}
		return torch.Update(ctx.World, func(value *animation.AnimatedSprite) {
			if pause {
				value.Paused = !value.Paused
			}
			if restart {
				value.Restart()
			}
			if once {
				value.Loop = animation.LoopNone
				value.Restart()
			}
			if loop {
				value.Loop = animation.LoopAll
				value.Restart()
			}
		})
	})
}
