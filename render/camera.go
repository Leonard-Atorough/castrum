package render

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Camera defines the framing of a camera entity.
//
// Its [core.Transform] supplies the position, rotation, and uniform zoom;
// zero scale components mean 1. The [Collector] selects a primary camera,
// preferring user-spawned cameras over the engine camera.
type Camera struct {
	// Primary marks this camera as eligible for rendering.
	Primary bool
	// Anchor is the viewport point the camera's position maps to: (0,0)
	// is the center and (-0.5,-0.5) is the top-left corner.
	Anchor geom.Vector2
	// Bounds limits the camera view to a world-space rectangle. An empty
	// rectangle means the view is unbounded.
	Bounds geom.Rect
}

// Validate reports an error if the camera's anchor lies outside the viewport.
func (c Camera) Validate() error {
	if math.IsNaN(c.Anchor.X) || c.Anchor.X < -0.5 || c.Anchor.X > 0.5 {
		return fmt.Errorf("anchor X must be between -0.5 and 0.5, got %v", c.Anchor.X)
	}
	if math.IsNaN(c.Anchor.Y) || c.Anchor.Y < -0.5 || c.Anchor.Y > 0.5 {
		return fmt.Errorf("anchor Y must be between -0.5 and 0.5, got %v", c.Anchor.Y)
	}
	return nil
}

// MovePosition returns the position reached by moving transform by a
// world-space displacement, without modifying transform.
//
// It constrains both the starting position and the destination to [Camera.Bounds],
// accounting for the logical viewport width and height in pixels, zoom,
// rotation, and anchor. A zero displacement only constrains the position.
// Empty bounds leave movement unbounded. If the view cannot fit on an axis,
// its center is pinned to the bounds center on that axis.
//
// MovePosition returns an error for a nonpositive viewport size, invalid anchor
// or zoom, non-finite position, displacement, or rotation, or a non-finite result.
func (c Camera) MovePosition(transform core.Transform, displacement geom.Vector2, width, height int) (geom.Vector2, error) {
	if width <= 0 || height <= 0 {
		return geom.Vector2{}, fmt.Errorf("camera movement: logical viewport must be positive, got %dx%d", width, height)
	}
	if err := c.Validate(); err != nil {
		return geom.Vector2{}, err
	}
	for _, value := range []struct {
		name  string
		value float64
	}{
		{"position X", transform.Position.X},
		{"position Y", transform.Position.Y},
		{"displacement X", displacement.X},
		{"displacement Y", displacement.Y},
		{"rotation", transform.Rotation},
	} {
		if math.IsNaN(value.value) || math.IsInf(value.value, 0) {
			return geom.Vector2{}, fmt.Errorf("camera movement: %s must be finite, got %v", value.name, value.value)
		}
	}
	zoom, err := cameraZoom(transform.Scale, transform.Scale, 1)
	if err != nil {
		return geom.Vector2{}, err
	}
	view := CameraView{
		Position: transform.Position,
		Zoom:     zoom,
		Rotation: transform.Rotation,
		Anchor:   c.Anchor,
	}
	if !c.Bounds.IsEmpty() {
		view.clampToBounds(c.Bounds, width, height)
	}
	view.Position = view.Position.Add(displacement)
	if !c.Bounds.IsEmpty() {
		view.clampToBounds(c.Bounds, width, height)
	}
	if math.IsNaN(view.Position.X) || math.IsInf(view.Position.X, 0) ||
		math.IsNaN(view.Position.Y) || math.IsInf(view.Position.Y, 0) {
		return geom.Vector2{}, fmt.Errorf("camera movement: resulting position must be finite, got %v", view.Position)
	}
	return view.Position, nil
}

// SpawnCamera creates a camera entity in w at position with the given zoom.
// A zoom of zero means 1. It returns an error for an invalid anchor or a
// negative or non-finite zoom.
func SpawnCamera(w *core.World, cam Camera, position geom.Vector2, zoom float64) (*core.Entity, error) {
	if err := cam.Validate(); err != nil {
		return nil, err
	}
	if zoom < 0 || math.IsNaN(zoom) || math.IsInf(zoom, 0) {
		return nil, fmt.Errorf("camera zoom must be finite and not negative, got %v", zoom)
	}
	if zoom == 0 {
		zoom = 1
	}
	return w.NewEntity(
		cam,
		core.Transform{Position: position, Scale: geom.Vector2{X: zoom, Y: zoom}},
	)
}

// engineCamera lets the collector prefer user cameras over the default camera.
type engineCamera struct{}

// SpawnEngineCamera creates the engine's default primary camera at the world
// origin with zoom 1. [Collector] uses it when no user-spawned primary camera
// is available.
func SpawnEngineCamera(w *core.World) (*core.Entity, error) {
	return w.NewEntity(
		Camera{Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: geom.Vector2{X: 1, Y: 1}},
		engineCamera{},
	)
}

// CameraView is the resolved framing of a camera for one collected frame.
//
// A [Collector] publishes it as the world's *CameraView resource after
// successful collection. Systems see the most recently collected view.
type CameraView struct {
	// Position is the camera position in world space after interpolation and bounds.
	Position geom.Vector2
	// Zoom is the scale used to project between world and screen coordinates.
	Zoom float64
	// Rotation is the interpolated camera rotation in radians. Positive values
	// rotate the view counterclockwise; sprites with the same rotation appear
	// unrotated.
	Rotation float64
	// Anchor is the viewport point the camera's position maps to: (0,0)
	// is the center and (-0.5,-0.5) is the top-left corner.
	Anchor geom.Vector2

	// aabb is the axis-aligned bounding box of the camera's view in world
	// space. It is a conservative culling rectangle that may be larger
	// than the actual view.
	aabb geom.Rect
}

// WorldToScreen projects a world-space point to pixel coordinates for a
// viewport of the given size. The camera's position maps to its anchor, and
// each coordinate is rounded to the nearest pixel.
func (v CameraView) WorldToScreen(world geom.Vector2, screenWidth, screenHeight int) geom.Vector2 {
	d := world.Sub(v.Position).Rotate(-v.Rotation)
	p := d.Mul(v.Zoom).Add(anchorOffset(v.Anchor, screenWidth, screenHeight))
	return geom.Vector2{X: math.Round(p.X), Y: math.Round(p.Y)}
}

// ScreenToWorld converts a screen-space point to world coordinates without
// rounding, preserving fractional positions.
func (v CameraView) ScreenToWorld(screen geom.Vector2, screenWidth, screenHeight int) geom.Vector2 {
	d := screen.Sub(anchorOffset(v.Anchor, screenWidth, screenHeight)).Div(v.Zoom).Rotate(v.Rotation)
	return d.Add(v.Position)
}

func anchorOffset(anchor geom.Vector2, width, height int) geom.Vector2 {
	return geom.Vector2{
		X: (anchor.X + 0.5) * float64(width),
		Y: (anchor.Y + 0.5) * float64(height),
	}
}

func cameraZoom(prev, curr geom.Vector2, alpha float64) (float64, error) {
	if prev.X == 0 {
		prev.X = 1
	}
	if prev.Y == 0 {
		prev.Y = 1
	}
	if curr.X == 0 {
		curr.X = 1
	}
	if curr.Y == 0 {
		curr.Y = 1
	}
	for _, s := range []struct {
		when  string
		scale geom.Vector2
	}{
		{"previous", prev},
		{"current", curr},
	} {

		if math.IsInf(s.scale.X, 0) || math.IsInf(s.scale.Y, 0) ||
			math.IsNaN(s.scale.X) || math.IsNaN(s.scale.Y) {
			return 0, fmt.Errorf("camera zoom (Transform.Scale) must be finite, %s scale %v", s.when, s.scale)
		}
		if s.scale.X <= 0 || s.scale.Y <= 0 {
			return 0, fmt.Errorf("camera zoom (Transform.Scale) must be positive, %s scale %v", s.when, s.scale)
		}
		if s.scale.X != s.scale.Y {
			return 0, fmt.Errorf("camera zoom (Transform.Scale) must be uniform, %s scale %v", s.when, s.scale)
		}
	}
	return prev.Lerp(curr, alpha).X, nil
}

func resolveCamera(cam Camera, prev core.PrevTransform, curr core.Transform, alpha float64, width, height int) (CameraView, error) {
	zoom, err := cameraZoom(prev.Scale, curr.Scale, alpha)
	if err != nil {
		return CameraView{}, err
	}
	view := CameraView{
		Position: prev.Position.Lerp(curr.Position, alpha),
		Zoom:     zoom,
		Rotation: prev.Rotation + (curr.Rotation-prev.Rotation)*alpha,
		Anchor:   cam.Anchor,
	}
	if !cam.Bounds.IsEmpty() {
		view.clampToBounds(cam.Bounds, width, height)
	}
	view.aabb = view.cullRect(width, height)
	return view, nil
}

// The circumscribed extents keep culling and bounds checks conservative for
// rotated views.
func (v CameraView) viewGeometry(width, height int) (offset, half geom.Vector2) {
	viewW := float64(width) / v.Zoom
	viewH := float64(height) / v.Zoom
	// The position maps to the anchor point; the view center sits
	// anchor-sized units away from it, on the opposite side.
	offset = geom.Vector2{
		X: -v.Anchor.X * viewW,
		Y: -v.Anchor.Y * viewH,
	}.Rotate(v.Rotation)
	half = geom.Vector2{X: viewW / 2, Y: viewH / 2}
	if v.Rotation != 0 {
		s, c := math.Abs(math.Sin(v.Rotation)), math.Abs(math.Cos(v.Rotation))
		half = geom.Vector2{X: c*half.X + s*half.Y, Y: s*half.X + c*half.Y}
	}
	return offset, half
}

func (v CameraView) cullRect(width, height int) geom.Rect {
	offset, half := v.viewGeometry(width, height)
	return geom.RectFromCenterSize(v.Position.Add(offset), half.Mul(2))
}

func (v *CameraView) clampToBounds(bounds geom.Rect, width, height int) {
	offset, half := v.viewGeometry(width, height)
	center := v.Position.Add(offset)
	pinned := bounds.Center()
	center.X = clampAxis(center.X, bounds.Min.X+half.X, bounds.Max.X-half.X, pinned.X)
	center.Y = clampAxis(center.Y, bounds.Min.Y+half.Y, bounds.Max.Y-half.Y, pinned.Y)
	v.Position = center.Sub(offset)
}

func clampAxis(v, min, max, pin float64) float64 {
	if min > max {
		return pin
	}
	return math.Max(min, math.Min(max, v))
}
