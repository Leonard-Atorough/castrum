package core

import "fmt"

// Camera is the framing half of a camera entity: pair it with a
// [Transform], which owns the position. An entity with a Camera and
// Primary set is the camera the renderer uses; cameras move by moving
// their Transform, and interpolated rendering covers them the same as
// any other entity.
//
// The render target's dimensions are not part of the Camera: the runner
// owns the logical resolution. Keeping a camera inside level bounds is
// a game-system concern, not component data.
type Camera struct {
	// Zoom is a multiplier on the world-to-screen scale; 1.0 means none.
	Zoom float64
	// Primary marks this camera as the one the renderer uses.
	Primary bool
}

func (c Camera) Validate() error {
	if c.Zoom <= 0 {
		return fmt.Errorf("zoom must be positive")
	}
	return nil
}
