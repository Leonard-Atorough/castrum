package main

// Command animate demonstrates the animation slice: a torch flickers
// through a grid-atlas clip at a fixed frame rate, driven entirely by
// engine wiring - the game only registers the atlas, the clip, and the
// entity.
//
// Assets resolve against the default filesystem: the game's working
// directory, so paths may point anywhere relative to it. Run from the
// repository root:
//
//	go run ./examples/animate
//
// wander demonstrates the embed.FS alternative for single-binary
// distribution, passed via castrum.WithFilesystem.

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	TorchLightFPS  = 10
	TorchLightLoop = animation.LoopForever

	AssetTorchLight     = "torch_light"
	ClipFlickeringTorch = "flickering_torch"

	Scale = 4
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(
		castrum.WithTitle("castrum - animate"),
	)
	if err != nil {
		return err
	}

	if err := g.AssetServer().RegisterGridAtlas(AssetTorchLight, "examples/animate/torch_light.png", 16, 28, "torch_light"); err != nil {
		return err
	}

	clipStore := g.Clips()

	if err := clipStore.Add(ClipFlickeringTorch, animation.AnimationClip{
		Source: AssetTorchLight,
		Frames: []string{
			"torch_light_0",
			"torch_light_1",
			"torch_light_2",
			"torch_light_3",
			"torch_light_4",
			"torch_light_5",
		},
		FPS:  TorchLightFPS,
		Loop: TorchLightLoop,
	}); err != nil {
		return err
	}

	_, err = g.World().NewEntity(
		core.Sprite{},
		core.Transform{
			Scale: geom.Vector2{X: Scale, Y: Scale},
		},
		animation.Animation{
			Clip: ClipFlickeringTorch,
		},
	)
	if err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrint(screen, "castrum animate: Enjoy the flame\nFPS: "+fmt.Sprintf("%.2f", ebiten.ActualFPS()))
		return nil
	})

	return runner.Run()
}
