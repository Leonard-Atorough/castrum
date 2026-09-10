package animation

import (
	"fmt"
	"sync"

	"github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
)

type AnimationEventType int

const (
	EventClipFinished AnimationEventType = iota
	EventClipLooped
)

// AnimationEvent is emitted by the animation system when clips loop or finish.
type AnimationEvent struct {
	EntityID ecs.EntityID
	ClipID   string
	Type     AnimationEventType
}

// AnimationClip represents a playable animation sequence.
// It references a TextureAtlas and a sequence of frame region names within that atlas,
// along with timing and looping configuration.
// Clips are created programmatically via AnimationManager, not loaded from disk.
type AnimationClip struct {
	Atlas      *atlas.TextureAtlas
	Frames     []string // Region names in the atlas (in order)
	FrameSpeed float64  // Time (in seconds) each frame is displayed
	Loop       bool     // Whether the animation repeats
}

type clipStorer interface {
	store(id string, clip *AnimationClip)
}

// AnimationClipBuilder is the builder interface for creating animation clips.
// Use method chaining to configure, then call Build() to register with the manager.
type AnimationClipBuilder struct {
	id         string
	atlas      *atlas.TextureAtlas
	frames     []string // Region names in the atlas
	frameSpeed float64
	loop       bool
	store      clipStorer
}

func NewAnimationClipBuilder(id string, atlas *atlas.TextureAtlas, store clipStorer) *AnimationClipBuilder {
	if store == nil {
		panic("animation clip builder requires a non-nil store")
	}
	return &AnimationClipBuilder{
		id:     id,
		atlas:  atlas,
		frames: make([]string, 0, 8),
		store:  store,
	}
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

	c.store.store(c.id, clip)
	return clip, nil
}

// AnimationClipStore orchestrates animation clip creation and storage.
// It is lightweight: it creates clips programmatically, validates them,
// and hands them off to the Animation system for playback orchestration.
type AnimationClipStore struct {
	mu    sync.RWMutex
	clips map[string]*AnimationClip
}

// NewAnimationClipStore creates a new AnimationClipStore.
func NewAnimationClipStore() *AnimationClipStore {
	return &AnimationClipStore{
		clips: make(map[string]*AnimationClip),
	}
}

// NewClip creates a new clip builder with the given ID and atlas.
// Use method chaining to configure, then call Build() to register.
func (m *AnimationClipStore) NewClip(id string, atlas *atlas.TextureAtlas) *AnimationClipBuilder {
	return &AnimationClipBuilder{
		id:     id,
		atlas:  atlas,
		frames: make([]string, 0, 8),
		store:  m,
	}
}

// Get retrieves a registered clip by ID, or nil if not found.
func (m *AnimationClipStore) Get(id string) *AnimationClip {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clips[id]
}

// store registers a clip (called by AnimationClipBuilder.Build).
func (m *AnimationClipStore) store(id string, clip *AnimationClip) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clips[id] = clip
}

// System processes animation playback for entities with Animation components.
// It delegates to the Manager for clip resolution and orchestrates frame advancement
// and event emission. Does not handle clip creation or configuration.
type System struct {
	query   *ecs.Query
	manager *AnimationClipStore
}

// NewSystem creates a new animation system with the given manager.
func NewSystem(manager *AnimationClipStore) *System {
	return &System{
		manager: manager,
	}
}

func (as *System) Init(world *ecs.World) error {
	as.query = world.NewQuery().WithRequiredComponents(
		components.Animation{},
		components.Sprite{},
	)

	return nil
}

// Update processes all Animation components, advancing frame time and emitting events.
func (as *System) Update(world *ecs.World, delta float64) error {
	bus, ok := ecs.GetResource[*events.EventBus](world)
	if !ok {
		return nil // EventBus not registered, skip event emission
	}

	for entry := range as.query.Execute() {
		anim, _ := entry.Get[components.Animation]()
		entityID := entry.EntityID

		if !anim.Playing {
			continue
		}

		// Look up the clip from the manager
		clip := as.manager.Get(anim.ClipPath)
		if clip == nil {
			// Skip if clip not found; log in production
			continue
		}

		// Advance frame time by delta * playback speed
		anim.FrameTime += delta * anim.PlaybackSpeed

		// Advance frames
		if anim.FrameTime >= clip.FrameSpeed {
			anim.FrameTime -= clip.FrameSpeed
			anim.FrameIndex++

			// Handle loop or stop
			if anim.FrameIndex >= len(clip.Frames) {
				if clip.Loop {
					anim.FrameIndex = 0
					bus.Emit(AnimationEvent{
						EntityID: entityID,
						ClipID:   anim.ClipPath,
						Type:     EventClipLooped,
					}, "AnimationSystem")
				} else {
					anim.FrameIndex = len(clip.Frames) - 1
					anim.Playing = false
					bus.Emit(AnimationEvent{
						EntityID: entityID,
						ClipID:   anim.ClipPath,
						Type:     EventClipFinished,
					}, "AnimationSystem")
				}
			}
		}

		_ = world.SetComponent(entityID, anim)
	}
	return nil
}

func (as *System) Shutdown(world *ecs.World) error {
	return nil
}
