package game

type Game struct {
	accumulator float64
	fixedDelta  float64
}

func (g *Game) Update() error {
	g.accumulator += g.fixedDelta

	for g.accumulator >= g.fixedDelta {
		g.accumulator -= g.fixedDelta
	}

	return nil
}

func (*Game) Draw() error {
	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
