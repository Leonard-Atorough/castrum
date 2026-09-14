package atlas

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
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

type Store struct {
	mu      sync.RWMutex
	atlases map[atlasKey]*any
}

func NewStore() *Store {
	return &Store{
		atlases: make(map[atlasKey]*any),
	}
}

func (s *Store) get(atlasID, assetID string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.atlases[newAtlasKey(atlasID, assetID)]
	if !ok {
		return nil, false
	}
	return *entry, ok
}

func (s *Store) set(atlasID, assetID string, atlas any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.atlases[newAtlasKey(atlasID, assetID)] = &atlas
}

type Service struct {
	store     *Store
	loadGroup *singleflight.Group
}

func NewService(store *Store) *Service {
	return &Service{
		store:     store,
		loadGroup: &singleflight.Group{},
	}
}

func (s *Service) Get(atlasID, assetID string) (any, error) {
	atlas, err, _ := s.loadGroup.Do(atlasID+assetID, func() (any, error) {
		atlas, ok := s.store.get(atlasID, assetID)
		if !ok {
			return nil, nil
		}
		return atlas, nil
	})
	if err != nil || atlas == nil {
		return nil, fmt.Errorf("atlas not found for atlasID=%s, assetID=%s", atlasID, assetID)
	}
	return atlas, nil
}

func (s *Service) Set(atlasID, assetID string, atlas any) {
	s.store.set(atlasID, assetID, atlas)
}
