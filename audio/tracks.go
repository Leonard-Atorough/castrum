package audio

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
)

// RepeatMode controls whether a [Source]'s track sequence restarts after its
// last track finishes.
type RepeatMode uint8

const (
	// RepeatNone stops the sequence after the last track finishes.
	RepeatNone RepeatMode = iota
	// RepeatAll restarts the sequence from the first track after the last
	// track finishes.
	RepeatAll
)

// Len returns the number of tracks in the source's sequence.
func (s Source) Len() int {
	if s.Track == "" {
		return 0
	}
	return 1 + len(s.More)
}

// Current returns the selected track, or an empty ID if the source has no
// track at the selected position.
func (s Source) Current() asset.ID {
	return s.TrackAt(s.index)
}

// TrackAt returns the track at position, or an empty ID if position is out of
// range. Position 0 is [Source.Track].
func (s Source) TrackAt(position int) asset.ID {
	if position < 0 || position >= s.Len() {
		return ""
	}
	if position == 0 {
		return s.Track
	}
	return s.More[position-1]
}

// SetTrack replaces the sequence with a single track and selects it.
//
// It returns an error if track is empty.
func (s *Source) SetTrack(track asset.ID) error {
	if track == "" {
		return fmt.Errorf("audio: track cannot be empty")
	}
	s.Track = track
	s.More = nil
	s.index = 0
	s.selectionVersion++
	return nil
}

// Next advances to the next track, wrapping to the first track at the end.
func (s *Source) Next() {
	if s.Len() == 0 {
		return
	}
	s.index = (s.index + 1) % s.Len()
	s.selectionVersion++
}

// Previous moves to the previous track, wrapping to the last track at the
// beginning.
func (s *Source) Previous() {
	if s.Len() == 0 {
		return
	}
	s.index = (s.index - 1 + s.Len()) % s.Len()
	s.selectionVersion++
}

// Seek selects the track at position if it is in range.
func (s *Source) Seek(position int) {
	if position < 0 || position >= s.Len() {
		return
	}
	if s.index == position {
		return
	}
	s.index = position
	s.selectionVersion++
}

// Clear removes all tracks from the source.
func (s *Source) Clear() {
	s.Track = ""
	s.More = nil
	s.index = 0
	s.selectionVersion++
}

// AddTrack appends track to the source's sequence.
//
// It returns an error if track is empty.
func (s *Source) AddTrack(track asset.ID) error {
	if track == "" {
		return fmt.Errorf("audio: track cannot be empty")
	}
	if s.Track == "" && len(s.More) == 0 {
		s.Track = track
		s.index = 0
		return nil
	}
	s.More = append(s.More, track)
	return nil
}

// RemoveTrack removes the track at position.
//
// It returns an error if position is out of range.
func (s *Source) RemoveTrack(position int) error {
	if position < 0 || position >= s.Len() {
		return fmt.Errorf("audio: position %d out of range", position)
	}
	wasCurrent := position == s.index
	if position == 0 {
		if len(s.More) > 0 {
			s.Track = s.More[0]
			s.More = s.More[1:]
		} else {
			s.Track = ""
		}
	} else {
		s.More = append(s.More[:position-1], s.More[position:]...)
	}
	switch {
	case s.Len() == 0:
		s.index = 0
	case position < s.index:
		s.index--
	case s.index >= s.Len():
		s.index = s.Len() - 1
	}
	if wasCurrent {
		s.selectionVersion++
	}
	return nil
}

// InsertTrack inserts track at position, or appends it if position exceeds the
// sequence length.
//
// It returns an error if position is negative or track is empty.
func (s *Source) InsertTrack(position int, track asset.ID) error {
	if position < 0 {
		return fmt.Errorf("audio: position cannot be negative")
	}
	if track == "" {
		return fmt.Errorf("audio: track cannot be empty")
	}
	hadTracks := s.Track != "" || len(s.More) > 0
	if !hadTracks {
		s.Track = track
		s.index = 0
		return nil
	}
	if position > s.Len() {
		return s.AddTrack(track)
	}
	if position == 0 {
		s.More = append([]asset.ID{s.Track}, s.More...)
		s.Track = track
	} else {
		s.More = append(s.More[:position-1], append([]asset.ID{track}, s.More[position-1:]...)...)
	}
	if position <= s.index {
		s.index++
	}
	return nil
}

func (s *Source) advanceAfterCompletion() bool {
	if s.Len() == 0 || s.index < 0 || s.index >= s.Len() {
		return false
	}
	if s.index+1 < s.Len() {
		s.index++
		s.selectionVersion++
		return true
	}
	if s.Repeat == RepeatAll {
		s.index = 0
		s.selectionVersion++
		return true
	}
	return false
}
