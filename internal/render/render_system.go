package render

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

type RenderLayer int

const (
	Layer1 RenderLayer = iota
	Layer2
	Layer3
	Layer4
	Layer5
	Layer6
	Layer7
	Layer8
	Layer9
	Layer10
)

type RenderSystem struct {
	renderer *Renderer
}

func NewRenderSystem(renderer *Renderer) *RenderSystem {
	return &RenderSystem{
		renderer: renderer,
	}
}

func (rs *RenderSystem) Init(world *core.World) error {
	return nil
}

func (rs *RenderSystem) Update(world *core.World, delta float64) error {
	entities := world.Query(core.Types(components.Renderable{}, components.Transform{}, components.Animation{})...)

	for _, entityID := range entities {
		anim, err := world.GetComponent(entityID, core.Types(components.Animation{})[0])
		if err != nil {
			continue
		}
		renderable, err := world.GetComponent(entityID, core.Types(components.Renderable{})[0])
		if err != nil {
			continue
		}

		animComp := anim.(components.Animation)
		renderComp := renderable.(components.Renderable)
		if !animComp.Playing {
			continue
		}

		animComp.FrameTime += delta

		if animComp.FrameTime >= animComp.FrameSpeed {
			animComp.FrameTime = 0
			animComp.FrameIndex++
			if animComp.FrameIndex >= len(animComp.Frames) {
				if animComp.Loop {
					animComp.FrameIndex = 0
				} else {
					animComp.FrameIndex = len(animComp.Frames) - 1
					animComp.Playing = false
				}
			}
		}

		renderComp.TexturePath = animComp.Frames[animComp.FrameIndex]
	}
	return nil
}

func (rs *RenderSystem) Shutdown(world *core.World) error {
	return nil
}
