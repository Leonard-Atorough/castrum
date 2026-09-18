package audio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

type playerKey struct {
	entityID ecs.EntityID
}

type playerState struct {
	player *audio.Player
	track  *AudioTrack
	// data     *assets.AudioData
	position geom.Vector2
	entityID ecs.EntityID
}

// AudioService owns the registry of audio tracks and the per-entity players
// that play them. It resolves track PCM data into players on demand and
// applies a volume mix of master, group, track, and per-entity volumes.
// Concurrent access is guarded by an RWMutex.
type AudioService struct {
	ctx     *audio.Context
	loader  *assets.Loader
	tracks  map[ID]*AudioTrack
	players map[playerKey]*playerState
	config  castrum.AudioConfig
	mu      sync.RWMutex
	context context.Context
}

// NewAudioService creates an AudioService backed by the given Ebiten audio
// context, asset loader, and volume configuration. The context is used to
// create players at playback time.
func NewAudioService(ctx *audio.Context, loader *assets.Loader, config castrum.AudioConfig, context context.Context) *AudioService {
	return &AudioService{
		ctx:     ctx,
		loader:  loader,
		tracks:  make(map[ID]*AudioTrack),
		players: make(map[playerKey]*playerState),
		config:  config,
		context: context,
	}
}

// AddTrack loads audio data from assetPath via the service's loader, resamples
// it to the audio context's sample rate, and registers the resulting track
// under id. It returns an error if the asset cannot be loaded.
func (s *AudioService) AddTrack(id ID, assetPath string, group Group, loop LoopMode, volume float64) error {
	data, err := s.loader.Load[assets.AudioData](s.context, assetPath)
	if err != nil {
		return err
	}

	pcm := s.resampleToContext(&data)

	s.tracks[id] = NewAudioTrack(id, pcm, group, loop, volume)
	return nil
}

// GetTrack returns the track registered under id. The bool reports whether
// a track was found.
func (s *AudioService) GetTrack(id ID) (*AudioTrack, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	track, ok := s.tracks[id]
	return track, ok
}

// RemoveTrack unregisters the track under id. It does not stop players
// currently playing that track.
func (s *AudioService) RemoveTrack(id ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tracks, id)
}

// HasTrack reports whether a track is registered under id.
func (s *AudioService) HasTrack(id ID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tracks[id]
	return ok
}

// HasPlayer reports whether a player state exists for entityID.
func (s *AudioService) HasPlayer(entityID ecs.EntityID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.players[playerKey{entityID: entityID}]
	return ok
}

// IsPlaying reports whether the player for entityID is actively playing.
// It returns false if no player exists for entityID.
func (s *AudioService) IsPlaying(entityID ecs.EntityID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.players[playerKey{entityID: entityID}]
	if !ok || state.player == nil {
		return false
	}
	return state.player.IsPlaying()
}

// IsLooping reports whether the player for entityID is set to loop indefinitely.
// It returns false if no player exists for entityID.
func (s *AudioService) IsLooping(entityID ecs.EntityID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.players[playerKey{entityID: entityID}]
	if !ok || state.player == nil {
		return false
	}
	return state.track.loop == LoopForever
}

// SyncPlayer synchronizes the player for entityID with the desired playing
// state and per-entity volume. The player is created lazily on the first
// request to play. The effective volume is the product of master, group,
// track, and the given per-entity volume. It returns an error if trackID is
// not registered.
func (as *AudioService) SyncPlayer(entityID ecs.EntityID, trackID ID, playing bool, volume float64) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	track, ok := as.tracks[trackID]
	if !ok {
		return fmt.Errorf("track not found: %v", trackID)
	}

	key := playerKey{entityID: entityID}
	state, ok := as.players[key]
	if !ok {
		state = &playerState{entityID: entityID, track: track}
		as.players[key] = state
	}

	// Lazy-create player
	if playing && state.player == nil {
		src := io.ReadSeeker(bytes.NewReader(track.pcm))
		if track.loop == LoopForever {
			src = audio.NewInfiniteLoop(src, int64(len(track.pcm)))
		}
		player, err := as.ctx.NewPlayer(src)
		if err != nil {
			return fmt.Errorf("failed to create audio player: %w", err)
		}
		state.player = player
	}

	// Compute effective volume: master → group → track → component
	effectiveVol := as.config.MasterVolume
	switch track.group {
	case GroupMusic:
		effectiveVol *= as.config.MusicVolume
	case GroupSFX:
		effectiveVol *= as.config.SFXVolume
	}
	effectiveVol *= volume * track.volume

	// Apply volume and sync play state
	if state.player != nil {
		state.player.SetVolume(effectiveVol)
		if playing && !state.player.IsPlaying() {
			state.player.Play()
		} else if !playing && state.player.IsPlaying() {
			state.player.Pause()
		}
	}

	return nil
}

// StopPlayer pauses the player for entityID and stops it reading its source.
// The player state is retained. If entityID has no registered player,
// StopPlayer is a no-op.
func (s *AudioService) StopPlayer(entityID ecs.EntityID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pauseAndStopPlayer(playerKey{entityID: entityID})
	return nil
}

// RemovePlayer stops and removes the player state for entityID, discarding it.
// If entityID has no registered state, RemovePlayer is a no-op.
func (s *AudioService) RemovePlayer(entityID ecs.EntityID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := playerKey{entityID: entityID}
	s.pauseAndStopPlayer(key)
	delete(s.players, key)
	return nil
}

func (s *AudioService) resampleToContext(data *assets.AudioData) []byte {
	if data.SampleRate == s.ctx.SampleRate() {
		return data.PCM // already good
	}
	src := bytes.NewReader(data.PCM)
	resampled := audio.ResampleReader(src, data.Length, data.SampleRate, s.ctx.SampleRate())
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, resampled)
	return buf.Bytes()
}

// pauseAndStopPlayer pauses the player for key and stops it reading its
// source. If no state or no player exists for key, it is a no-op.
func (s *AudioService) pauseAndStopPlayer(key playerKey) {
	if state, ok := s.players[key]; ok && state.player != nil {
		state.player.PauseAndStopReading()
	}
}
