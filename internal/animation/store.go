package animation

import (
	"sync"

	"github.com/leonard-atorough/castrum/internal/atlas"
)

// ClipStore is the concurrent in-memory registry for animation clips.
// It is registered as a world resource by [castrum.NewGame] and shared
// between the animation [System] and the renderer.
type ClipStore struct {
	clips map[string]*AnimationClip
	mu    sync.RWMutex
}

// NewClipStore creates an empty clip store.
func NewClipStore() *ClipStore {
	return &ClipStore{
		clips: make(map[string]*AnimationClip),
	}
}

// NewBuilder returns a [ClipBuilder] wired to this store.
func (s *ClipStore) NewBuilder(id string, atlas *atlas.Atlas) *ClipBuilder {
	return &ClipBuilder{
		id:     id,
		atlas:  atlas,
		events: make(map[int][]string),
		store:  s,
	}
}

// Get returns the clip registered under id. The bool is false if no
// clip is registered.
func (s *ClipStore) Get(id string) (*AnimationClip, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clip, ok := s.clips[id]
	return clip, ok
}

// AddClip registers a clip under the given id.
func (s *ClipStore) AddClip(id string, clip *AnimationClip) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clips[id] = clip
}
