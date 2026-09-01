package types

import "math"

type Polygon struct {
	Points []Vector2
}

func (p *Polygon) AddPoint(point Vector2) {
	p.Points = append(p.Points, point)
}

func (p *Polygon) RemovePoint(index int) {
	if index < 0 || index >= len(p.Points) {
		return
	}
	p.Points = append(p.Points[:index], p.Points[index+1:]...)
}

func (p *Polygon) GetPoint(index int) Vector2 {
	if index < 0 || index >= len(p.Points) {
		return Vector2{}
	}
	return p.Points[index]
}

func (p *Polygon) GetPoints() []Vector2 {
	return p.Points
}

func (p *Polygon) NumPoints() int {
	return len(p.Points)
}

func (p *Polygon) Clear() {
	p.Points = []Vector2{}
}

func (p *Polygon) GetEdges() [][2]Vector2 {
	edges := make([][2]Vector2, 0, len(p.Points))
	for i := 0; i < len(p.Points); i++ {
		next := (i + 1) % len(p.Points)
		edges = append(edges, [2]Vector2{p.Points[i], p.Points[next]})
	}
	return edges
}

// Perimeter calculates the perimeter of the polygon by summing the lengths of its edges.
func (p *Polygon) Perimeter() float64 {
	edges := p.GetEdges()
	perimeter := 0.0
	for _, edge := range edges {
		dx := edge[1].X - edge[0].X
		dy := edge[1].Y - edge[0].Y
		perimeter += math.Sqrt(dx*dx + dy*dy)
	}
	return perimeter
}

// Area calculates the area of the polygon using the shoelace formula.
// It returns 0 if the polygon has fewer than 3 points.
func (p *Polygon) Area() float64 {
	area := 0.0
	n := len(p.Points)
	if n < 3 {
		return 0.0
	}
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += p.Points[i].X * p.Points[j].Y
		area -= p.Points[j].X * p.Points[i].Y
	}
	return math.Abs(area) / 2.0
}

//Contains determines if a given point is inside the polygon using the ray-casting algorithm.
// It returns true if the point is inside the polygon, and false otherwise.
func (p *Polygon) Contains(point Vector2) bool {
	n := len(p.Points)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := range n {
		pi := p.Points[i]
		pj := p.Points[j]
		if ((pi.Y > point.Y) != (pj.Y > point.Y)) &&
			(point.X < (pj.X-pi.X)*(point.Y-pi.Y)/(pj.Y-pi.Y)+pi.X) {
			inside = !inside
		}
		j = i
	}
	return inside
}

// BoundingBox calculates the axis-aligned bounding box of the polygon.
func (p *Polygon) BoundingBox() (min, max Vector2) {
	if len(p.Points) == 0 {
		return Vector2{}, Vector2{}
	}
	min = p.Points[0]
	max = p.Points[0]
	for _, point := range p.Points[1:] {
		if point.X < min.X {
			min.X = point.X
		}
		if point.Y < min.Y {
			min.Y = point.Y
		}
		if point.X > max.X {
			max.X = point.X
		}
		if point.Y > max.Y {
			max.Y = point.Y
		}
	}
	return min, max
}

// Centroid calculates the centroid (geometric center) of the polygon.
// It returns a zero vector if the polygon has no points.
func (p *Polygon) Centroid() Vector2 {
	n := len(p.Points)
	if n == 0 {
		return Vector2{}
	}
	centroid := Vector2{}
	for _, point := range p.Points {
		centroid.X += point.X
		centroid.Y += point.Y
	}
	centroid.X /= float64(n)
	centroid.Y /= float64(n)
	return centroid
}

func (p *Polygon) Translate(offset Vector2) {
	for i := range p.Points {
		p.Points[i].X += offset.X
		p.Points[i].Y += offset.Y
	}
}
