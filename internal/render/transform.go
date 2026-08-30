package render

import (
	"image/color"

	"github.com/leonard-atorough/castrum/types"
)

type Transform struct {
	Position types.Vector2
	Rotation float64
	Scale    types.Vector2
	Color    color.Color
}

func (t *Transform) Translate(offset types.Vector2) {
	t.Position.X += offset.X
	t.Position.Y += offset.Y
}

func (t *Transform) Rotate(angle float64) {
	t.Rotation += angle
}

func (t *Transform) ScaleBy(factor types.Vector2) {
	t.Scale.X *= factor.X
	t.Scale.Y *= factor.Y
}

func (t *Transform) SetPosition(position types.Vector2) {
	t.Position = position
}
