package ebitrun

import (
	"bytes"
	"errors"
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
	loader   *asset.Loader
	players  map[core.EntityID]*audioSession
	touched  map[core.EntityID]struct{}
}

// audioSession retains finished metadata after releasing playback resources
// so reconciliation cannot implicitly replay a completed source.
type audioSession struct {
	player  *ebitaudio.Player
	balance balanceStream
	// source is the open fs file behind a streaming play, if any;
	// eager plays share loader-cached bytes and have none.
	source   io.Closer
	loop     audio.LoopMode
	load     audio.LoadMode
	held     bool
	finished bool
}

func (s *audioSession) release() error {
	s.finished = true
	if s.player != nil {
		s.player.PauseAndStopReading()
		s.player = nil
	}
	s.balance = nil
	if s.source == nil {
		return nil
	}
	source := s.source
	s.source = nil
	return source.Close()
}

func newAudioProvider(audioCtx *ebitaudio.Context, loader *asset.Loader) *audioProvider {
	registerAudioCodecs(loader, audioCtx.SampleRate())
	return &audioProvider{
		audioCtx: audioCtx,
		loader:   loader,
		players:  make(map[core.EntityID]*audioSession),
		touched:  make(map[core.EntityID]struct{}),
	}
}

// streamDecoders is the format-to-decoder table: one table, two
// consumers — the eager codecs registered into the asset loader, and
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
func registerAudioCodecs(loader *asset.Loader, sampleRate int) {
	for format, decode := range streamDecoders {
		err := loader.RegisterDecoder(format, func(r io.Reader) (asset.AudioData, error) {
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
// recreates the playback from the beginning. Volume and balance apply to
// live sessions on every sync, including while held, without restarting.
// Finished sessions release playback resources but remain recorded until
// rebuilt or swept.
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
		if err := s.release(); err != nil {
			return false, fmt.Errorf("castrum/ebiten: audio: release source %q: %w", source, err)
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

	if err := s.balance.setBalance(want.Balance); err != nil {
		return false, fmt.Errorf("castrum/ebiten: audio: source %q: %w", source, err)
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
		if err := s.release(); err != nil {
			return false, fmt.Errorf("castrum/ebiten: audio: complete source %q: %w", source, err)
		}
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
		if err := s.release(); err != nil {
			return fmt.Errorf("castrum/ebiten: audio: sweep: release entity %d: %w", id, err)
		}
		delete(a.players, id)
	}
	clear(a.touched)
	return nil
}

func (a *audioProvider) createPlayer(source asset.ID, want audio.Playback) (_ *audioSession, err error) {
	if want.Rate != 1 {
		return nil, fmt.Errorf("castrum/ebiten: audio: playback rate %v is not supported by this runner", want.Rate)
	}

	session := &audioSession{
		loop: want.Loop,
		load: want.Load,
	}
	defer func() {
		if err != nil && session.source != nil {
			if closeErr := session.source.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("castrum/ebiten: audio: close source %q: %w", source, closeErr))
			}
		}
	}()

	var reader io.ReadSeeker
	var length int64
	switch want.Load {
	case audio.LoadEager:
		data, err := a.loader.Load[asset.AudioData](string(source))
		if err != nil {
			return nil, fmt.Errorf("castrum/ebiten: audio: load eager audio: %w", err)
		}
		reader = bytes.NewReader(data.PCM)
		length = int64(len(data.PCM))
	case audio.LoadStream:
		file, format, err := a.loader.Open(string(source))
		if err != nil {
			return nil, fmt.Errorf("castrum/ebiten: audio: open stream source: %w", err)
		}
		session.source = file
		decode, ok := streamDecoders[format]
		if !ok {
			return nil, fmt.Errorf("castrum/ebiten: audio: no stream decoder for format %q", format)
		}
		if _, ok := file.(io.Seeker); !ok {
			return nil, fmt.Errorf("castrum/ebiten: audio: stream source %q is not seekable", source)
		}
		decoded, err := decode(a.audioCtx.SampleRate(), file)
		if err != nil {
			return nil, fmt.Errorf("castrum/ebiten: audio: decode stream: %w", err)
		}
		reader = decoded
		length = decoded.Length()
	default:
		return nil, fmt.Errorf("castrum/ebiten: audio: invalid load mode %d for source %q", want.Load, source)
	}
	if want.Loop == audio.LoopForever {
		reader = ebitaudio.NewInfiniteLoop(reader, length)
	}
	session.balance, err = newBalanceReader(reader, want.Balance)
	if err != nil {
		return nil, fmt.Errorf("castrum/ebiten: audio: source %q: %w", source, err)
	}
	session.player, err = a.audioCtx.NewPlayer(session.balance)
	if err != nil {
		return nil, fmt.Errorf("castrum/ebiten: audio: create player for source %q: %w", source, err)
	}
	session.player.SetVolume(want.Volume)
	return session, nil
}
