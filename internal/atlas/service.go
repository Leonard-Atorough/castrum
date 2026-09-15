package atlas

import (
	"fmt"
	"sync"
)

type atlasKey struct {
	atlasID string
	assetID string
}

func newAtlasKey(atlasID, assetID string) atlasKey {
	return atlasKey{
		atlasID: atlasID,
		assetID: assetID,
	}
}

// Store is the concurrent in-memory atlas registry. It maps
// (atlasID, assetID) pairs to [Atlas] instances.
type Store struct {
	mu      sync.RWMutex
	atlases map[atlasKey]*Atlas
}

func NewStore() *Store {
	return &Store{
		atlases: make(map[atlasKey]*Atlas),
	}
}

func (s *Store) get(atlasID, assetID string) (*Atlas, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.atlases[newAtlasKey(atlasID, assetID)]
	if !ok {
		return nil, false
	}
	return entry, ok
}

func (s *Store) set(atlasID, assetID string, atlas *Atlas) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.atlases[newAtlasKey(atlasID, assetID)] = atlas
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

func (s *Service) Get(atlasID, assetID string) (*Atlas, error) {
	atlas, ok := s.store.get(atlasID, assetID)
	if !ok {
		return nil, fmt.Errorf("atlas not found for atlasID=%s, assetID=%s", atlasID, assetID)
	}
	return atlas, nil
}

func (s *Service) Set(atlasID, assetID string, atlas *Atlas) error {
	if atlas == nil {
		return fmt.Errorf("cannot set nil atlas for atlasID=%s, assetID=%s", atlasID, assetID)
	}
	s.store.set(atlasID, assetID, atlas)
	return nil
}

func (s *Service) Has(atlasID, assetID string) bool {
	_, ok := s.store.get(atlasID, assetID)
	return ok
}

func (s *Service) Delete(atlasID, assetID string) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	delete(s.store.atlases, newAtlasKey(atlasID, assetID))
}

func (s *Service) DeleteByAssetID(assetID string) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	for key := range s.store.atlases {
		if key.assetID == assetID {
			delete(s.store.atlases, key)
		}
	}
}

func (s *Service) Clear() {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.atlases = make(map[atlasKey]*Atlas)
}
