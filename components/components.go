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



type Animation struct {
	Frames     []string
	FrameIndex int
	FrameCount int
	FrameTime  float64 // time elapsed since the last frame change
	FrameSpeed float64 // seconds per frame
	Loop       bool
	Playing    bool // indicates whether the animation is currently playing
	AutoPlay   bool // indicates whether the animation should start playing automatically
}

func (a *Animation) Reset() {
	a.FrameIndex = 0
	a.FrameTime = 0
	a.Playing = a.AutoPlay
}

// Spin rotates an entity's Transform by AngularVelocity radians per second.
type Spin struct {
	AngularVelocity float64
}
