package animation

import (
	"fmt"

	"github.com/leonard-atorough/castrum/internal/atlas"
)

// Loop controls how a clip repeats after reaching the last frame.
type Loop int

const (
	// LoopNone plays the clip once and holds on the last frame.
	LoopNone Loop = iota
	// LoopForever repeats the clip from the first frame after the last.
	LoopForever
)

// AnimationClip is an immutable animation definition: a sequence of atlas
// region names played back at a fixed frame rate. Clips are registered
// with the [ClipStore] and referenced by ID from [components.Animation].
type AnimationClip struct {
	AtlasID     atlas.ID
	Frames      []string
	FPS         float64
	Loop        Loop
	FrameEvents map[int][]string // frame index → event names; nil if none
}

// ClipBuilder constructs an [AnimationClip] using a fluent API. Obtained
// from [ClipStore.NewBuilder]; there is no standalone constructor.
type ClipBuilder struct {
	id     string
	atlas  *atlas.Atlas // for frame validation only; not stored on the clip
	frames []string
	fps    float64
	loop   Loop
	events map[int][]string
	store  *ClipStore
}

// Build validates the clip definition and registers it with the
// [ClipStore]. The returned clip is a copy; mutating it does not
// affect the store entry.
func (b *ClipBuilder) Build() (*AnimationClip, error) {
	if b.atlas == nil {
		return nil, fmt.Errorf("atlas is required for clip %q", b.id)
	}
	if len(b.frames) == 0 {
		return nil, fmt.Errorf("no frames added to clip %q", b.id)
	}
	if b.fps <= 0 {
		return nil, fmt.Errorf("FPS must be greater than 0 for clip %q", b.id)
	}
	for _, frame := range b.frames {
		if _, ok := b.atlas.Region(frame); !ok {
			return nil, fmt.Errorf("frame %q not found in atlas for clip %q", frame, b.id)
		}
	}

	frames := make([]string, len(b.frames))
	copy(frames, b.frames)

	var events map[int][]string
	if len(b.events) > 0 {
		events = make(map[int][]string, len(b.events))
		for k, v := range b.events {
			events[k] = append([]string(nil), v...)
		}
	}

	clip := &AnimationClip{
		AtlasID:     b.atlas.ID(),
		Frames:      frames,
		FPS:         b.fps,
		Loop:        b.loop,
		FrameEvents: events,
	}
	b.store.AddClip(b.id, clip)
	return clip, nil
}

// AddFrame appends a single atlas region name to the clip's frame sequence.
func (b *ClipBuilder) AddFrame(region string) *ClipBuilder {
	b.frames = append(b.frames, region)
	return b
}

// AddFrames appends multiple atlas region names to the clip's frame sequence.
func (b *ClipBuilder) AddFrames(regions ...string) *ClipBuilder {
	for _, region := range regions {
		b.AddFrame(region)
	}
	return b
}

// SetFPS sets the playback rate in frames per second.
func (b *ClipBuilder) SetFPS(fps float64) *ClipBuilder {
	b.fps = fps
	return b
}

// SetLoop sets whether the clip repeats after the last frame.
func (b *ClipBuilder) SetLoop(loop Loop) *ClipBuilder {
	b.loop = loop
	return b
}

// OnFrame tags a frame index with an event name. When playback reaches
// that frame, the system emits an [AnimationEvent] with the name.
func (b *ClipBuilder) OnFrame(frameIndex int, eventName string) *ClipBuilder {
	if b.events == nil {
		b.events = make(map[int][]string)
	}
	b.events[frameIndex] = append(b.events[frameIndex], eventName)
	return b
}
