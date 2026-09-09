package animation

import (
	"fmt"
	"sync"

	"github.com/leonard-atorough/castrum/internal/assets"
)

// AnimationClip represents a playable animation sequence.
// It references a TextureAtlas and a sequence of frame region names within that atlas,
// along with timing and looping configuration.
// Clips are created programmatically via AnimationManager, not loaded from disk.
type AnimationClip struct {
	Atlas      *assets.TextureAtlas
	Frames     []string // Region names in the atlas (in order)
	FrameSpeed float64  // Time (in seconds) each frame is displayed
	Loop       bool     // Whether the animation repeats
}

// AnimationClipBuilder is the builder interface for creating animation clips.
// Use method chaining to configure, then call Build() to register with the manager.
type AnimationClipBuilder struct {
	id         string
	atlas      *assets.TextureAtlas
	frames     []string // Region names in the atlas
	frameSpeed float64
	loop       bool
	manager    *Manager
}

// AddFrame appends a region name to this clip.
func (c *AnimationClipBuilder) AddFrame(regionName string) *AnimationClipBuilder {
	c.frames = append(c.frames, regionName)
	return c
}

// AddFrames appends multiple region names at once.
func (c *AnimationClipBuilder) AddFrames(regionNames ...string) *AnimationClipBuilder {
	c.frames = append(c.frames, regionNames...)
	return c
}

// SetFrameSpeed sets the time (in seconds) each frame is displayed.
func (c *AnimationClipBuilder) SetFrameSpeed(speed float64) *AnimationClipBuilder {
	c.frameSpeed = speed
	return c
}

// SetLoop sets whether the animation repeats.
func (c *AnimationClipBuilder) SetLoop(loop bool) *AnimationClipBuilder {
	c.loop = loop
	return c
}

// Build validates and registers this clip with the manager.
func (c *AnimationClipBuilder) Build() (*AnimationClip, error) {
	if c.atlas == nil {
		return nil, fmt.Errorf("animation clip %q: atlas is required", c.id)
	}
	if len(c.frames) == 0 {
		return nil, fmt.Errorf("animation clip %q: at least one frame is required", c.id)
	}
	if c.frameSpeed <= 0 {
		return nil, fmt.Errorf("animation clip %q: frame speed must be positive", c.id)
	}

	// Validate frame names exist in atlas
	for _, regionName := range c.frames {
		if _, ok := c.atlas.Regions[regionName]; !ok {
			return nil, fmt.Errorf("animation clip %q: region %q not found in atlas", c.id, regionName)
		}
	}

	clip := &AnimationClip{
		Atlas:      c.atlas,
		Frames:     c.frames,
		FrameSpeed: c.frameSpeed,
		Loop:       c.loop,
	}

	c.manager.store(c.id, clip)
	return clip, nil
}

// Manager orchestrates animation clip creation and storage.
// It is lightweight: it creates clips programmatically, validates them,
// and hands them off to the Animation system for playback orchestration.
type Manager struct {
	mu    sync.RWMutex
	clips map[string]*AnimationClip
}

// New creates a new Manager.
func NewManager() *Manager {
	return &Manager{
		clips: make(map[string]*AnimationClip),
	}
}

// NewClip creates a new clip builder with the given ID and atlas.
// Use method chaining to configure, then call Build() to register.
func (m *Manager) NewClip(id string, atlas *assets.TextureAtlas) *AnimationClipBuilder {
	return &AnimationClipBuilder{
		id:      id,
		atlas:   atlas,
		frames:  make([]string, 0, 8),
		manager: m,
	}
}

// Get retrieves a registered clip by ID, or nil if not found.
func (m *Manager) Get(id string) *AnimationClip {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clips[id]
}

// store registers a clip (called by AnimationClipBuilder.Build).
func (m *Manager) store(id string, clip *AnimationClip) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clips[id] = clip
}
