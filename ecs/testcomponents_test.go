package ecs

type TestPosition struct {
	X, Y float64
}

func (p TestPosition) Name() string     { return "TestPosition" }
func (p TestPosition) Clone() Component { return TestPosition{X: p.X, Y: p.Y} }

type TestVelocity struct {
	X, Y float64
}

func (v TestVelocity) Name() string     { return "TestVelocity" }
func (v TestVelocity) Clone() Component { return TestVelocity{X: v.X, Y: v.Y} }

type TestHealth struct {
	Value int
}

func (h TestHealth) Name() string     { return "TestHealth" }
func (h TestHealth) Clone() Component { return TestHealth{Value: h.Value} }

type TestSprite struct {
	TextureID     string
	Width, Height int
}

func (s TestSprite) Name() string { return "TestSprite" }
func (s TestSprite) Clone() Component {
	return TestSprite{TextureID: s.TextureID, Width: s.Width, Height: s.Height}
}
