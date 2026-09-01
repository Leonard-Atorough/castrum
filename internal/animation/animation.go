package animation

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

type AnimationManager struct {
}

func NewAnimationManager() *AnimationManager {
	return &AnimationManager{}
}

func (am *AnimationManager) Update(world *core.World, delta float64) error {
	entities := world.Query(core.Types(components.Renderable{}, components.Transform{}, components.Animation{})...)

	for _, entityID := range entities {
		animComp, err := core.GetComponent[components.Animation](world, entityID)
		if err != nil {
			continue
		}
		renderComp, err := core.GetComponent[components.Renderable](world, entityID)
		if err != nil {
			continue
		}

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
		_ = core.SetComponent(world, entityID, renderComp)
	}
	return nil
}
