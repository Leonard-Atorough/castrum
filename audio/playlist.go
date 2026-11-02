package audio

import (
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
)

type RepeatMode uint8

const (
	RepeatNone RepeatMode = iota
	RepeatAll
)

type Playlist struct {
	tracks           []asset.ID
	current          int
	selectionVersion uint64
	Repeat           RepeatMode
}

func (p Playlist) Validate() error {
	if len(p.tracks) == 0 {
		return fmt.Errorf("audio: playlist must have at least one track")
	}
	for i, track := range p.tracks {
		if track == "" {
			return fmt.Errorf("audio: playlist track %d must not be empty", i)
		}
	}
	if p.Repeat != RepeatNone && p.Repeat != RepeatAll {
		return fmt.Errorf("audio: invalid repeat mode")
	}
	return nil
}

// NewPlaylist creates a playlist from tracks.
func NewPlaylist(tracks []asset.ID) Playlist {
	return Playlist{
		tracks:  slices.Clone(tracks),
		current: 0,
	}
}

// Len returns the number of tracks in the playlist.
func (p Playlist) Len() int {
	return len(p.tracks)
}

// Current returns the current track, or an empty ID if there is no track.
func (p Playlist) Current() asset.ID {
	if len(p.tracks) == 0 || p.current < 0 || p.current >= len(p.tracks) {
		return ""
	}
	return p.tracks[p.current]
}

// TrackAt returns the track at position, or an empty ID if position is out of range.
func (p Playlist) TrackAt(position int) asset.ID {
	if position < 0 || position >= len(p.tracks) {
		return ""
	}
	return p.tracks[position]
}

// Next advances to the next track, wrapping to the first track at the end.
func (p *Playlist) Next() {
	if len(p.tracks) == 0 {
		return
	}
	p.current = (p.current + 1) % len(p.tracks)
	p.selectionVersion++
}

// Previous moves to the previous track, wrapping to the last track at the beginning.
func (p *Playlist) Previous() {
	if len(p.tracks) == 0 {
		return
	}
	p.current = (p.current - 1 + len(p.tracks)) % len(p.tracks)
	p.selectionVersion++
}

// Seek sets the current track to position if it is in range.
func (p *Playlist) Seek(position int) {
	if position < 0 || position >= len(p.tracks) {
		return
	}
	if p.current == position {
		return
	}
	p.current = position
	p.selectionVersion++
}

// Clear removes all tracks from the playlist.
func (p *Playlist) Clear() {
	p.tracks = nil
	p.current = 0
	p.selectionVersion++
}

// AddTrack appends track to the playlist.
//
// It returns an error if track is empty.
func (p *Playlist) AddTrack(track asset.ID) error {
	if track == "" {
		return fmt.Errorf("playlist: track cannot be empty")
	}
	if len(p.tracks) == 0 {
		p.current = 0
	}
	p.tracks = append(p.tracks, track)
	return nil
}

// RemoveTrack removes the track at position.
//
// It returns an error if position is out of range.
func (p *Playlist) RemoveTrack(position int) error {
	if position < 0 || position >= len(p.tracks) {
		return fmt.Errorf("playlist: position %d out of range", position)
	}
	wasCurrent := position == p.current
	p.tracks = append(p.tracks[:position], p.tracks[position+1:]...)
	if len(p.tracks) == 0 {
		p.current = 0
	} else if position < p.current {
		p.current--
	} else if p.current >= len(p.tracks) {
		p.current = len(p.tracks) - 1
	}
	if wasCurrent {
		p.selectionVersion++
	}
	return nil
}

// InsertTrack inserts track at position, or appends it if position exceeds the playlist length.
//
// It returns an error if position is negative.
func (p *Playlist) InsertTrack(position int, track asset.ID) error {
	if position < 0 {
		return fmt.Errorf("playlist: position cannot be negative")
	}
	if track == "" {
		return fmt.Errorf("playlist: track cannot be empty")
	}
	if position > len(p.tracks) {
		return p.AddTrack(track)
	}
	hadTracks := len(p.tracks) > 0
	p.tracks = append(p.tracks[:position], append([]asset.ID{track}, p.tracks[position:]...)...)
	if !hadTracks {
		p.current = 0
	} else if position <= p.current {
		p.current++
	}
	return nil
}

func (p *Playlist) advanceAfterCompletion() bool {
	if len(p.tracks) == 0 || p.current < 0 || p.current >= len(p.tracks) {
		return false
	}
	if p.current+1 < len(p.tracks) {
		p.current++
		p.selectionVersion++
		return true
	}
	if p.Repeat == RepeatAll {
		p.current = 0
		p.selectionVersion++
		return true
	}
	return false
}
