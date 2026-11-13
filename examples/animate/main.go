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
	"github.com/Leonard-Atorough/castrum/render"
)

//go:embed torch_light.png torch_light.sheet.json
var files embed.FS

const (
	sheetPath = "torch_light.sheet.json"

	TorchLightFPS  = 10
	TorchLightLoop = animation.LoopForever

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
			"pause":   {input.KeyInput{Key: input.KeySpace}},
			"restart": {input.KeyInput{Key: input.KeyR}},
			"once":    {input.KeyInput{Key: input.Key1}},
			"loop":    {input.KeyInput{Key: input.Key2}},
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

	clipStore := g.World().MustResource[*animation.ClipStore]()

	clip := animation.Clip{
		Source: source,
		Frames: []string{
			"torch_light_0",
			"torch_light_1",
			"torch_light_2",
			"torch_light_3",
			"torch_light_4",
			"torch_light_5",
		},
		FPS: TorchLightFPS,
	}
	for _, name := range clip.Frames {
		if _, err := sheet.Region(name); err != nil {
			return nil, err
		}
	}
	if err := clipStore.Add(ClipFlickeringTorch, clip); err != nil {
		return nil, err
	}

	torch, err := g.World().NewEntity(
		render.Sprite{Drawable: render.SheetSource{Sheet: source, Region: clip.Frames[0]}},
		core.Transform{
			Scale: geom.Vector2{X: Scale, Y: Scale},
		},
		animation.Animation{
			Clip: ClipFlickeringTorch,
			Loop: TorchLightLoop,
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
		value, ok := torch.Component[animation.Animation](ctx.World)
		if !ok {
			return fmt.Errorf("animated torch has no playback component")
		}
		panel.Status("playback", fmt.Sprintf("frame %d   paused: %t   completed: %t", value.Current, value.Paused, value.HasCompleted()))
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
		return torch.Update(ctx.World, func(value *animation.Animation) {
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
				value.Loop = animation.LoopForever
				value.Restart()
			}
		})
	})
}
