package render

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Camera defines the framing for a camera entity. Pair it with a
// [core.Transform]: the transform's position and rotation are the
// camera's, and its uniform Scale is the camera's zoom, where zero
// components mean 1. The renderer uses a primary camera and interpolates
// it like any other entity. A user-spawned primary takes precedence over
// the engine's default camera.
type Camera struct {
	// Primary marks this camera as eligible for rendering.
	Primary bool
	// Anchor is the viewport point the camera's position maps to: (0,0)
	// is the center and (-0.5,-0.5) is the top-left corner.
	Anchor geom.Vector2
	// Bounds is the world-space rectangle the camera keeps its view
	// inside. A zero or empty rectangle means unbounded.
	Bounds geom.Rect
}

// Validate checks that the camera's anchor lies within the viewport.
func (c Camera) Validate() error {
	if math.IsNaN(c.Anchor.X) || c.Anchor.X < -0.5 || c.Anchor.X > 0.5 {
		return fmt.Errorf("anchor X must be between -0.5 and 0.5, got %v", c.Anchor.X)
	}
	if math.IsNaN(c.Anchor.Y) || c.Anchor.Y < -0.5 || c.Anchor.Y > 0.5 {
		return fmt.Errorf("anchor Y must be between -0.5 and 0.5, got %v", c.Anchor.Y)
	}
	return nil
}

// SpawnCamera creates a new camera entity in the given world at the specified
// position with the given zoom; zero zoom means 1. Invalid framing or a
// negative zoom fail at setup. The camera renders only when cam.Primary is
// set.
func SpawnCamera(w *core.World, cam Camera, position geom.Vector2, zoom float64) (*core.Entity, error) {
	if err := cam.Validate(); err != nil {
		return nil, err
	}
	if zoom < 0 {
		return nil, fmt.Errorf("camera zoom must not be negative, got %v", zoom)
	}
	if zoom == 0 {
		zoom = 1
	}
	return w.NewEntity(
		cam,
		core.Transform{Position: position, Scale: geom.Vector2{X: zoom, Y: zoom}},
	)
}

// engineCamera is a marker component used to distinguish the engine's
// default camera from user-spawned cameras.
type engineCamera struct{}

// SpawnEngineCamera creates the engine's default primary camera at the world
// origin with zoom 1. It is used as a fallback when no user primary camera is available.
func SpawnEngineCamera(w *core.World) (*core.Entity, error) {
	return w.NewEntity(
		Camera{Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: geom.Vector2{X: 1, Y: 1}},
		engineCamera{},
	)
}

// CameraView is a camera's resolved framing for a rendered frame:
// interpolated position and rotation, zoom, and the anchor. The collector
// publishes it as the world's [*CameraView] resource; it holds the most
// recently drawn view, one frame stale for frame-phase readers.
type CameraView struct {
	// Position is the interpolated camera position in world space.
	Position geom.Vector2
	// Zoom is the scale used to project between world and screen coordinates.
	Zoom float64
	// Rotation is the camera's interpolated rotation in radians. Positive
	// values rotate the camera counter-clockwise in world space; a sprite
	// rotated the same amount appears unrotated.
	Rotation float64
	// Anchor is the viewport point the camera's position maps to: (0,0)
	// is the center and (-0.5,-0.5) is the top-left corner.
	Anchor geom.Vector2

	// aabb is the axis-aligned bounding box of the camera's view in world
	// space. It is a conservative culling rectangle that may be larger
	// than the actual view.
	aabb geom.Rect
}

// WorldToScreen projects a world-space point to pixel coordinates in a
// render target of the given size. The camera's position maps to its
// anchor point in the viewport. Each coordinate is rounded to the nearest
// whole pixel.
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

// anchorOffset returns the screen-space position of the camera's anchor
// point: the zero anchor sits at the viewport's center, and the anchor
// range [-0.5, 0.5] spans the viewport.
func anchorOffset(anchor geom.Vector2, width, height int) geom.Vector2 {
	return geom.Vector2{
		X: (anchor.X + 0.5) * float64(width),
		Y: (anchor.Y + 0.5) * float64(height),
	}
}

// cameraZoom interpolates a camera's [core.Transform] scale into a uniform
// zoom. Zero components mean 1, as elsewhere in the engine; a negative or
// non-uniform scale is invalid framing, because a camera view must not be
// anisotropic. Errors name the offending scale.
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
		if s.scale.X <= 0 || s.scale.Y <= 0 {
			return 0, fmt.Errorf("camera zoom (Transform.Scale) must be positive, %s scale %v", s.when, s.scale)
		}
		if s.scale.X != s.scale.Y {
			return 0, fmt.Errorf("camera zoom (Transform.Scale) must be uniform, %s scale %v", s.when, s.scale)
		}
	}
	return prev.Lerp(curr, alpha).X, nil
}

// resolveCamera interpolates a camera entity into its view for a frame,
// applies bounds clamping, and derives the culling rectangle. Width and
// height are the logical resolution.
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

// viewGeometry returns the world-space vector from the camera position
// to the center of its anchored view, and the view's half size, expanded
// to cover rotation (its circumscribed half extents).
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

// cullRect returns the conservative world rectangle the camera can see: the
// viewport expanded to cover rotation, centered on the anchored view.
func (v CameraView) cullRect(width, height int) geom.Rect {
	offset, half := v.viewGeometry(width, height)
	return geom.RectFromCenterSize(v.Position.Add(offset), half.Mul(2))
}

// clampToBounds keeps the anchored view inside bounds: per axis, the view
// center clamps into bounds shrunk by the view's half extents, pinning to
// the bounds center where the view cannot fit. Rotated views clamp
// conservatively through their circumscribed extents.
func (v *CameraView) clampToBounds(bounds geom.Rect, width, height int) {
	offset, half := v.viewGeometry(width, height)
	center := v.Position.Add(offset)
	pinned := bounds.Center()
	center.X = clampAxis(center.X, bounds.Min.X+half.X, bounds.Max.X-half.X, pinned.X)
	center.Y = clampAxis(center.Y, bounds.Min.Y+half.Y, bounds.Max.Y-half.Y, pinned.Y)
	v.Position = center.Sub(offset)
}

// clampAxis clamps v into [min, max], or returns pin when the range is empty.
func clampAxis(v, min, max, pin float64) float64 {
	if min > max {
		return pin
	}
	return math.Max(min, math.Min(max, v))
}
