package atlas

import (
	"fmt"
	"sync"
)

// Store is the concurrent in-memory atlas registry. It maps atlas IDs
// to [Atlas] instances.
type Store struct {
	mu      sync.RWMutex
	atlases map[string]*Atlas
}

func NewStore() *Store {
	return &Store{
		atlases: make(map[string]*Atlas),
	}
}

func (s *Store) get(atlasID string) (*Atlas, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.atlases[atlasID]
	if !ok {
		return nil, false
	}
	return entry, ok
}

func (s *Store) set(atlasID string, atlas *Atlas) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.atlases[atlasID] = atlas
}

// Service provides read/write access to the atlas [Store]. It is the
// production backing store for [Builder] and the texture provider.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Get(atlasID string) (*Atlas, error) {
	atlas, ok := s.store.get(atlasID)
	if !ok {
		return nil, fmt.Errorf("atlas not found for atlasID=%s", atlasID)
	}
	return atlas, nil
}

func (s *Service) Set(atlasID string, atlas *Atlas) error {
	if atlas == nil {
		return fmt.Errorf("cannot set nil atlas for atlasID=%s", atlasID)
	}
	s.store.set(atlasID, atlas)
	return nil
}

func (s *Service) Has(atlasID string) bool {
	_, ok := s.store.get(atlasID)
	return ok
}

func (s *Service) Delete(atlasID string) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	delete(s.store.atlases, atlasID)
}

func (s *Service) DeleteByAssetID(assetID string) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	for id, atlas := range s.store.atlases {
		if atlas.AssetID() == assetID {
			delete(s.store.atlases, id)
		}
	}
}

func (s *Service) Clear() {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.atlases = make(map[string]*Atlas)
}
