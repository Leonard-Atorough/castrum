package ebitrun

import (
	"bytes"
	"fmt"
	"io"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// audioStream is a decoded, seekable audio stream at the audio
// context's sample rate: what the streaming path feeds players from.
type audioStream interface {
	io.ReadSeeker
	// Length returns the decoded length in bytes.
	Length() int64
}

// audioProvider is the runner's [audio.Controller]: it owns the
// ebiten players and applies the reconciler's resolved
// [audio.Playback] state to them, one session per synced entity.
type audioProvider struct {
	audioCtx *ebitaudio.Context
	server   *asset.Server
	players  map[core.EntityID]*audioSession
	touched  map[core.EntityID]struct{}
}

// audioSession is one entity's player plus what the provider must
// release on completion: a Player never closes its source, so the
// streaming path parks its open file here.
type audioSession struct {
	player *ebitaudio.Player
	// source is the open fs file behind a streaming play, if any;
	// eager plays share server-cached bytes and have none.
	source   io.Closer
	loop     audio.LoopMode
	load     audio.LoadMode
	held     bool
	finished bool
}

func newAudioProvider(audioCtx *ebitaudio.Context, server *asset.Server) *audioProvider {
	registerAudioCodecs(server, audioCtx.SampleRate())
	return &audioProvider{
		audioCtx: audioCtx,
		server:   server,
		players:  make(map[core.EntityID]*audioSession),
		touched:  make(map[core.EntityID]struct{}),
	}
}

// streamDecoders is the format-to-decoder table: one table, two
// consumers — the eager codecs registered into the asset server, and
// the streaming path's per-player decode.
var streamDecoders = map[asset.Format]func(int, io.Reader) (audioStream, error){
	asset.FormatMP3: func(rate int, r io.Reader) (audioStream, error) {
		return mp3.DecodeWithSampleRate(rate, r)
	},
	asset.FormatWAV: func(rate int, r io.Reader) (audioStream, error) {
		return wav.DecodeWithSampleRate(rate, r)
	},
	asset.FormatOGG: func(rate int, r io.Reader) (audioStream, error) {
		return vorbis.DecodeWithSampleRate(rate, r)
	},
}

// registerAudioCodecs registers [asset.AudioData] decoders for the
// built-in audio formats, each bound to the audio context's rate:
// decode always resamples to that rate at decode time, and a source
// already at it never resamples. DecodeWithoutResampling is never
// used - a source at a different rate would play pitched.
//
// A failure here is an engine authoring bug (a duplicate in this
// static table), so it panics: no caller is positioned to recover.
// The override flag makes a re-registration idempotent; a second
// Runner for one game is rejected by the resource provide, not here.
func registerAudioCodecs(server *asset.Server, sampleRate int) {
	for format, decode := range streamDecoders {
		err := server.RegisterDecoder(format, func(r io.Reader) (asset.AudioData, error) {
			stream, err := decode(sampleRate, r)
			if err != nil {
				return asset.AudioData{}, err
			}
			return drain(stream, sampleRate)
		}, true)
		if err != nil {
			panic(fmt.Sprintf("castrum/ebiten: audio: register codec %s: %v", format, err))
		}
	}
}

// drain reads a decoded stream into the cached AudioData value. The
// sample rate is the bound context rate by DecodeWithSampleRate's
// contract; the length is the stream's own report.
func drain(s audioStream, sampleRate int) (asset.AudioData, error) {
	pcm, err := io.ReadAll(s)
	if err != nil {
		return asset.AudioData{}, fmt.Errorf("read decoded audio: %w", err)
	}
	return asset.AudioData{PCM: pcm, SampleRate: sampleRate, Length: s.Length()}, nil
}

// Sync makes the playback of entityID match want, per the
// [audio.Controller] contract. This runner bakes the loop and load
// modes into the player at creation, so changing either in want
// recreates the playback from the beginning. Volume applies on
// every sync, including while held.
//
// Note: variable-speed playback is not supported; a rate other
// than 1 is an error. Errors name the audio source.
func (a *audioProvider) Sync(entityID core.EntityID, source asset.ID, want audio.Playback) (live bool, err error) {
	s, exists := a.players[entityID]

	if !exists && !want.Playing && !want.Restart {
		return false, nil
	}
	a.touched[entityID] = struct{}{}

	if exists && (want.Restart || want.Loop != s.loop || want.Load != s.load) {
		s.player.PauseAndStopReading()
		if s.source != nil {
			if err := s.source.Close(); err != nil {
				return false, fmt.Errorf("castrum/ebiten: audio: close source: %w", err)
			}
		}
		delete(a.players, entityID)
		exists = false
	}

	if !exists {
		s, err = a.createPlayer(source, want)
		if err != nil {
			return false, err
		}
		a.players[entityID] = s
		if want.Playing {
			s.player.Play()
		} else {
			s.held = true
		}
		return true, nil
	}

	if s.finished {
		return false, nil
	}

	if !want.Playing {
		if !s.held {
			s.player.Pause()
			s.held = true
		}
		s.player.SetVolume(want.Volume)
		return true, nil
	}

	if s.held {
		s.player.Play()
		s.held = false
	}

	if !s.player.IsPlaying() {
		s.finished = true
		return false, nil
	}

	s.player.SetVolume(want.Volume)
	return true, nil
}

// Sweep releases the sessions of entities not synced since the
// previous sweep, per the [audio.Controller] contract: it stops the
// player and closes a streaming play's open file.
//
// Note: Sweep stops at the first error; sessions not yet examined
// are released by the next sweep.
func (a *audioProvider) Sweep() error {
	for id, s := range a.players {
		if _, ok := a.touched[id]; ok {
			continue
		}
		s.player.PauseAndStopReading()
		if s.source != nil {
			if err := s.source.Close(); err != nil {
				return fmt.Errorf("castrum/ebiten: audio: sweep: close source: %w", err)
			}
		}
		delete(a.players, id)
	}
	clear(a.touched)
	return nil
}

func (a *audioProvider) createPlayer(source asset.ID, want audio.Playback) (*audioSession, error) {
	if want.Rate != 1 {
		return nil, fmt.Errorf("castrum/ebiten: audio: playback rate %v is not supported by this runner", want.Rate)
	}

	session := &audioSession{
		loop: want.Loop,
		load: want.Load,
	}

	switch want.Load {
	case audio.LoadEager:
		data, err := a.server.Load[asset.AudioData](string(source))
		if err != nil {
			return nil, fmt.Errorf("castrum/ebiten: audio: load eager audio: %w", err)
		}
		var player *ebitaudio.Player
		if want.Loop == audio.LoopForever {
			player, err = a.audioCtx.NewPlayer(
				ebitaudio.NewInfiniteLoop(bytes.NewReader(data.PCM), int64(len(data.PCM))),
			)
			if err != nil {
				return nil, fmt.Errorf("castrum/ebiten: audio: create looping player: %w", err)
			}
		} else {
			player = a.audioCtx.NewPlayerFromBytes(data.PCM)
		}
		player.SetVolume(want.Volume)
		session.player = player
	case audio.LoadStream:
		file, format, err := a.server.Open(string(source))
		if err != nil {
			return nil, fmt.Errorf("castrum/ebiten: audio: open stream source: %w", err)
		}
		decode, ok := streamDecoders[format]
		if !ok {
			file.Close()
			return nil, fmt.Errorf("castrum/ebiten: audio: no stream decoder for format %q", format)
		}
		if _, ok := file.(io.Seeker); !ok {
			file.Close()
			return nil, fmt.Errorf("castrum/ebiten: audio: stream source %q is not seekable", source)
		}
		decoded, err := decode(a.audioCtx.SampleRate(), file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("castrum/ebiten: audio: decode stream: %w", err)
		}
		reader := io.Reader(decoded)
		if want.Loop == audio.LoopForever {
			reader = ebitaudio.NewInfiniteLoop(decoded, decoded.Length())
		}
		player, err := a.audioCtx.NewPlayer(reader)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("castrum/ebiten: audio: create stream player: %w", err)
		}
		player.SetVolume(want.Volume)
		session.player = player
		session.source = file
	}
	return session, nil
}
