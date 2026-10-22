// Command animate demonstrates frame animation: a torch flickers
// through a grid-atlas clip at a fixed frame rate, driven entirely by
// engine wiring - the game only registers the atlas, the clip, and
// the entity; the renderer stays animation-blind.
//
// Assets are embedded into the binary via //go:embed, passed to the
// engine with castrum.WithFilesystem. Run from the repository root
// (or anywhere):
//
//	go run ./examples/animate
package main

import (
	"embed"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

//go:embed torch_light.png
var files embed.FS

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
		castrum.WithFilesystem(files),
		castrum.WithAnimation(),
	)
	if err != nil {
		return err
	}

	if err := g.World().MustResource[*asset.Loader]().RegisterGridAtlas(AssetTorchLight, "torch_light.png", 16, 28, "torch_light"); err != nil {
		return err
	}

	clipStore := g.World().MustResource[*animation.ClipStore]()

	if err := clipStore.Add(ClipFlickeringTorch, animation.Clip{
		Source: AssetTorchLight,
		Frames: []string{
			"torch_light_0",
			"torch_light_1",
			"torch_light_2",
			"torch_light_3",
			"torch_light_4",
			"torch_light_5",
		},
		FPS: TorchLightFPS,
	}); err != nil {
		return err
	}

	_, err = g.World().NewEntity(
		render.Sprite{},
		core.Transform{
			Scale: geom.Vector2{X: Scale, Y: Scale},
		},
		animation.Animation{
			Clip: ClipFlickeringTorch,
			Loop: TorchLightLoop,
		},
	)
	if err != nil {
		return err
	}

	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}

	return g.Run(runner)
}
