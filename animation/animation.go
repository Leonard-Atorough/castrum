// Package animation provides structures and systems for handling sprite animations using texture atlases.
// It includes builders for creating animation clips and a store for managing them.
// It also defines events related to animation playback.
package animation

import (
	"fmt"
	"sync"

	"github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
)

// AnimationEventType defines the type of events emitted by the animation system.
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

// AnimationClip represents a sequence of frames from a texture atlas that can be played back as an animation.
type AnimationClip struct {
	Atlas      *atlas.TextureAtlas // Reference to the texture atlas containing the frames
	Frames     []string            // Region names in the atlas (in order)
	FrameSpeed float64             // Time (in seconds) each frame is displayed
	Loop       bool                // Whether the animation repeats
}

type clipStorer interface {
	store(id string, clip *AnimationClip)
}

// AnimationClipBuilder provides a fluent interface for constructing AnimationClip instances.
type AnimationClipBuilder struct {
	id         string
	atlas      *atlas.TextureAtlas
	frames     []string // Region names in the atlas
	frameSpeed float64
	loop       bool
	store      clipStorer
}

// NewAnimationClipBuilder initializes a new builder for an animation clip.
// The clip will be registered with the provided store upon calling Build().
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

// AnimationClipStore manages the storage and retrieval of animation clips.
// Provides thread-safe access to animation clips and facilitates their creation via builders.
type AnimationClipStore struct {
	mu    sync.RWMutex
	clips map[string]*AnimationClip
}

// NewAnimationClipStore initializes and returns a new AnimationClipStore instance.
func NewAnimationClipStore() *AnimationClipStore {
	return &AnimationClipStore{
		clips: make(map[string]*AnimationClip),
	}
}

// NewBuilder creates a new animation clip builder with the given ID and atlas.
// Use method chaining to configure the builder, then call Build() to register the clip.
func (m *AnimationClipStore) NewBuilder(id string, atlas *atlas.TextureAtlas) *AnimationClipBuilder {
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

// AnimationSystem processes animation playback for entities with Animation components.
// It delegates to the Manager for clip resolution and orchestrates frame advancement
// and event emission. Does not handle clip creation or configuration.
type AnimationSystem struct {
	query   *ecs.Query
	manager *AnimationClipStore
}

// NewSystem creates a new animation system with the given manager.
func NewSystem(manager *AnimationClipStore) *AnimationSystem {
	return &AnimationSystem{
		manager: manager,
	}
}

func (as *AnimationSystem) Init(world *ecs.World) error {
	as.query = world.NewQuery().WithRequiredComponents(
		components.Animation{},
		components.Sprite{},
	)

	return nil
}

// Update processes all Animation components, advancing frame time and emitting events.
func (as *AnimationSystem) Update(world *ecs.World, delta float64) error {
	bus, ok := ecs.GetResource[*events.EventBus](world)
	if !ok {
		return nil
	}

	for entry := range as.query.Execute() {
		anim, _ := entry.Get[components.Animation]()
		entityID := entry.EntityID

		if !anim.Playing {
			continue
		}

		clip := as.manager.Get(anim.ClipPath)
		if clip == nil {
			continue
		}

		anim.FrameTime += delta * anim.PlaybackSpeed

		if anim.FrameTime >= clip.FrameSpeed {
			anim.FrameTime -= clip.FrameSpeed
			anim.FrameIndex++

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

func (as *AnimationSystem) Shutdown(world *ecs.World) error {
	return nil
}
