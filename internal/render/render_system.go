package render

import "github.com/leonard-atorough/castrum/internal/core"

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
	return nil
	// NOTE: The render system and the renderer work together to handle rendering, with the renderer being what talks to ebiten.
	// The render system is responsible for managing the renderable, transform and animation components, doing things like handling animation updates, visibility checks, and sorting entities by layer. The renderer is responsible for actually drawing the entities to the screen using ebiten.
}

func (rs *RenderSystem) Shutdown(world *core.World) error {
	return nil
}
