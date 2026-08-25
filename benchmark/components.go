package benchmark

import "github.com/leonard-atorough/castrum/ecs"

// Position component represents a 2D position
type Position struct {
	X, Y float64
}

func (p Position) Name() string         { return "Position" }
func (p Position) Clone() ecs.Component { return Position{X: p.X, Y: p.Y} }

// Velocity component represents movement speed
type Velocity struct {
	X, Y float64
}

func (v Velocity) Name() string         { return "Velocity" }
func (v Velocity) Clone() ecs.Component { return Velocity{X: v.X, Y: v.Y} }

// Health component represents entity health
type Health struct {
	Value int
}

func (h Health) Name() string         { return "Health" }
func (h Health) Clone() ecs.Component { return Health{Value: h.Value} }

// Sprite component represents renderable sprite
type Sprite struct {
	TextureID string
	Width, Height int
}

func (s Sprite) Name() string         { return "Sprite" }
func (s Sprite) Clone() ecs.Component { return Sprite{TextureID: s.TextureID, Width: s.Width, Height: s.Height} }
