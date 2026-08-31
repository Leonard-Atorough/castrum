package components

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

type Renderable struct {
	TexturePath string
	Primitive   PrimitiveKind
	Layer       int
	Visible     bool
}

type PrimitiveKind int

const (
	PrimitiveKindRectangle PrimitiveKind = iota
	PrimitiveKindCircle
	PrimitiveKindLine
)
