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

type RenderLayer int

const (
	Layer0 RenderLayer = iota
	Layer1
	Layer2
	Layer3
	Layer4
	Layer5
	Layer6
	Layer7
	Layer8
	Layer9
	Layer10
	// Debug layer for rendering debug information
	LayerDebug
)

type Renderable struct {
	TexturePath string
	Primitive   PrimitiveKind
	Layer       RenderLayer
	Visible     bool
	Data        any // holds additional data for the primitive, e.g., *Polygon for PrimitiveKindPolygon
}

type PrimitiveKind int

const (
	PrimitiveKindRectangle PrimitiveKind = iota
	PrimitiveKindCircle
	PrimitiveKindLine
	PrimitiveKindPolygon
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
