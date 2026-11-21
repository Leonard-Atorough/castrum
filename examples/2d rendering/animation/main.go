// Command animation demonstrates playback of a spritesheet clip.
//
// SPACE pauses or resumes playback, R restarts, 1 plays once, 2 loops, and 4
// swaps to a reversed clip. The built-in fixed-phase animation system advances
// the [animation.AnimatedSprite]; the readout reports completion and loop events.
//
// The clip uses the torch_light_N regions of torch_light.json, authored with
// [spritesheet.Grid]. See the spritesheets example for authoring. Run with
// -author <file> to regenerate the metadata without opening a window.
//
// Run from this directory:
//
//	go run .
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

//go:embed torch_light.png torch_light.json
var files embed.FS

const (
	texturePath = "torch_light.png"
	sheetPath   = "torch_light.json"

	torchFPS  = 10
	torchLoop = animation.LoopAll

	// scale enlarges the 16x28 cells to 64x112 so the frames read clearly.
	scale = 4

	screenW, screenH = 1280, 720
)

var frames = []string{
	"torch_light_0",
	"torch_light_1",
	"torch_light_2",
	"torch_light_3",
	"torch_light_4",
	"torch_light_5",
}

func reversedFrames() []string {
	out := make([]string, len(frames))
	for i, name := range frames {
		out[len(frames)-1-i] = name
	}
	return out
}

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
	ebitrun.New(ebitrun.WithDebugOverlay()).Run(g)
	return nil
}

func newGame() (*castrum.Game, error) {
	g, err := castrum.New(
		castrum.WithTitle("castrum - animation"),
		castrum.WithFilesystem(files),
		castrum.WithBindings(input.Bindings{
			"pause":   {input.KeyInput{Key: input.KeySpace}.Binding()},
			"restart": {input.KeyInput{Key: input.KeyR}.Binding()},
			"once":    {input.KeyInput{Key: input.Key1}.Binding()},
			"loop":    {input.KeyInput{Key: input.Key2}.Binding()},
			"swap":    {input.KeyInput{Key: input.Key4}.Binding()},
		}),
	)

	// Resolve the sheet and validate every clip frame at setup, so a bad
	// document or region name fails the launch, not the first frame.
	loader := g.World().MustResource[*asset.Loader]()
	source := asset.FileSheet(sheetPath)
	sheet, err := source.Resolve(loader)
	if err != nil {
		return nil, err
	}
	texture, err := loader.Load[asset.TextureData](texturePath)
	if err != nil {
		return nil, err
	}
	if err := sheet.ValidateBounds(texture.Image.Bounds()); err != nil {
		return nil, err
	}
	for _, name := range frames {
		if _, err := sheet.Region(name); err != nil {
			return nil, err
		}
	}

	clip, err := animation.NewSpriteClip(source, frames, torchFPS)
	if err != nil {
		return nil, err
	}
	reversed, err := animation.NewSpriteClip(source, reversedFrames(), torchFPS)
	if err != nil {
		return nil, err
	}

	torch, err := animation.NewAnimatedSprite(clip)
	if err != nil {
		return nil, err
	}
	torch.Loop = torchLoop

	// The engine camera frames the world from the screen's center, so
	// world coordinates are screen coordinates at zoom 1.
	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return nil, err
	}

	entity, err := g.World().NewEntity(
		torch,
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: scale, Y: scale},
		},
	)
	if err != nil {
		return nil, err
	}

	if err := g.AddSystem(core.PhaseFrame, "animation.controls", playbackSystem(entity, reversed)); err != nil {
		return nil, err
	}
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "SPACE: pause/resume   R: restart   1: play once   2: loop   4: swap clip")
	if err := g.AddSystem(core.PhaseFixed, "animation.readout", core.SystemFunc(func(ctx *core.Context) error {
		value, ok := entity.Component[animation.AnimatedSprite](ctx.World)
		if !ok {
			return fmt.Errorf("animated torch has no playback component")
		}
		_, region, _, err := value.CurrentFrame()
		if err != nil {
			return err
		}
		panel.Status("playback", fmt.Sprintf("frame %s   paused: %t   completed: %t", region, value.Paused, value.HasCompleted()))
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

func playbackSystem(entity core.Entity, reversed animation.SpriteClip) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		pause := ctx.Actions.JustPressed("pause")
		restart := ctx.Actions.JustPressed("restart")
		once := ctx.Actions.JustPressed("once")
		loop := ctx.Actions.JustPressed("loop")
		swap := ctx.Actions.JustPressed("swap")
		if !pause && !restart && !once && !loop && !swap {
			return nil
		}
		return entity.Update(ctx.World, func(value *animation.AnimatedSprite) {
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
			if swap {
				// SetClip replaces the clip and resets completion; the
				// reversed clip was validated in newGame.
				if err := value.SetClip(reversed); err != nil {
					panic(err)
				}
			}
		})
	})
}
