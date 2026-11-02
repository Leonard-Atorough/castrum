package audio

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

// ListenerMode selects how a listener is positioned.
type ListenerMode int

const (
	// FollowCamera uses the resolved primary camera's position and orientation.
	FollowCamera ListenerMode = iota
	// FixedPosition uses the listener's position and world axes.
	FixedPosition
)

// Listener controls spatial playback for sources in the world.
// The zero value disables spatialization.
type Listener struct {
	// Enabled enables spatialization for sources that opt in.
	Enabled bool
	// Mode selects camera-following or a fixed position.
	Mode ListenerMode
	// Position is the world-space listener position in [FixedPosition] mode.
	Position geom.Vector2
	// Range is the positive finite distance in world units at which sound is silent.
	Range float64
}

type spatialPass struct {
	ctx      *core.Context
	camera   *render.CameraResolver
	resolved bool
	listener *Listener
	position geom.Vector2
	rotation float64
}

func (p *spatialPass) resolve() error {
	listener, err := p.ctx.World.Resource[*Listener]()
	if err != nil {
		return fmt.Errorf("audio: resolve listener: %w; provide *audio.Listener or construct the game with castrum.New", err)
	}
	if listener == nil {
		return fmt.Errorf("audio: listener must not be nil")
	}
	p.listener = listener
	if !listener.Enabled {
		p.resolved = true
		return nil
	}
	if !finite(listener.Range) || listener.Range <= 0 {
		return fmt.Errorf("audio: listener range must be positive and finite, got %v", listener.Range)
	}
	switch listener.Mode {
	case FixedPosition:
		p.position = listener.Position
	case FollowCamera:
		if p.ctx.LogicalWidth <= 0 || p.ctx.LogicalHeight <= 0 {
			return fmt.Errorf("audio: camera listener requires a positive logical viewport, got %dx%d", p.ctx.LogicalWidth, p.ctx.LogicalHeight)
		}
		view, ok, err := p.camera.Resolve(1, p.ctx.LogicalWidth, p.ctx.LogicalHeight)
		if err != nil {
			return fmt.Errorf("audio: resolve listener camera: %w", err)
		}
		if !ok {
			return fmt.Errorf("audio: camera listener requires a primary camera")
		}
		p.position, p.rotation = view.Position, view.Rotation
	default:
		return fmt.Errorf("audio: invalid listener mode %d", listener.Mode)
	}
	if !finite(p.position.X) || !finite(p.position.Y) || !finite(p.rotation) {
		return fmt.Errorf("audio: listener position and rotation must be finite")
	}
	p.resolved = true
	return nil
}

func (p *spatialPass) apply(id core.EntityID, s Source, want *Playback) error {
	if !s.Spatial {
		return nil
	}
	if !p.resolved {
		if err := p.resolve(); err != nil {
			return err
		}
	}
	if !p.listener.Enabled {
		return nil
	}
	transform, ok := core.NewEntity(id).Component[core.Transform](p.ctx.World)
	if !ok {
		return fmt.Errorf("audio: entity %d: spatial source requires a Transform", id)
	}
	if !finite(transform.Position.X) || !finite(transform.Position.Y) {
		return fmt.Errorf("audio: entity %d: spatial source position must be finite", id)
	}
	delta := transform.Position.Sub(p.position)
	local := delta.Rotate(-p.rotation)
	if !finite(delta.X) || !finite(delta.Y) || !finite(local.X) || !finite(local.Y) {
		return fmt.Errorf("audio: entity %d: spatial displacement must be finite", id)
	}
	t := math.Min(math.Hypot(delta.X, delta.Y)/p.listener.Range, 1)
	curve := s.Attenuation
	if curve == nil {
		curve = geom.EaseLinear
	}
	start, end, attenuation := curve(0), curve(1), curve(t)
	if start != 0 || end != 1 || !finite(attenuation) || attenuation < 0 || attenuation > 1 {
		return fmt.Errorf("audio: entity %d: attenuation curve must map 0 to 0 and 1 to 1 and return finite values in [0, 1]", id)
	}
	want.Volume *= 1 - attenuation
	want.Balance = math.Max(-1, math.Min(1, s.Balance+local.X/p.listener.Range))
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
