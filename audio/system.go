package audio

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
)

// Finished is the signal the audio system emits on the update that observes
// a [LoopNone] play finishing, the same update that sets [Source.Completed].
//
// The audio system runs in the frame schedule, so Finished is readable only
// by frame systems registered after it during the same frame phase; it does
// not survive into the next frame. React to it from a frame system, or poll
// [Source.Completed] from anywhere when the exact frame does not matter.
type Finished struct {
	// Entity is the entity whose Source finished. A [OneShot]'s entity is
	// destroyed on the same update, so its ID identifies the play only.
	Entity core.EntityID
	// Audio is the asset ID of the finished play.
	Audio asset.ID
}

// Playback is the resolved state that a [Controller] applies to a play.
// [NewAudioSystem] derives it from a [Source] and a [Mixer].
type Playback struct {
	// Playing reports whether playback should advance. False holds its position.
	Playing bool
	// Restart requests a fresh player from the beginning. The audio system
	// sets it on the first sync, a clip or playlist selection change, or
	// [Source.Restart].
	Restart bool
	// Volume is the final mix — master x group x play — in [0, 1].
	// Zero is silence: runtime fade targets are safe here.
	Volume float64
	// Rate is the resolved playback multiplier; 1 is the normal rate.
	Rate float64
	// Loop is the play's repeat behavior after the last sample.
	Loop LoopMode
	// Load selects eager decode or per-play streaming.
	Load LoadMode
	// Balance is the per-play channel attenuation from [Source.Balance],
	// independent of Volume.
	Balance float64
}

func fold(s Source, mixer *Mixer) Playback {
	return Playback{
		Playing: !s.Paused && (!mixer.Paused() || s.Pause == PauseContinues),
		Volume:  s.Volume * mixer.Master() * mixer.GroupVolume(s.Group),
		Rate:    resolveMultiplier(s.PlaybackMultiplier),
		Loop:    s.Loop,
		Load:    s.Load,
		Balance: s.Balance,
	}
}

func resolveMultiplier(multiplier float64) float64 {
	if multiplier == 0 {
		return 1
	}
	return multiplier
}

// Controller applies resolved [Playback] state to backend players. Runners
// implement it and pass it to [NewAudioSystem].
type Controller interface {
	// Sync applies want to the player for entityID and source. live reports
	// whether the play has not finished, not whether it is currently playing:
	// a paused play is live, while a finished [LoopNone] play is not.
	//
	// A finished play must not resume merely because want.Playing is true;
	// want.Restart requests an explicit restart. The audio system calls Sync
	// for each [Source] on every run.
	Sync(entityID core.EntityID, source asset.ID, want Playback) (live bool, err error)
	// Sweep releases players for entities not passed to Sync since the
	// previous sweep. The audio system calls it after each reconciliation pass.
	Sweep() error
}

// NewAudioSystem returns a [core.System] that reconciles [Source] components
// with controller and mixer, emits a [Finished] signal when a [LoopNone] play
// completes, and destroys completed [OneShot] entities. Without an audio
// system, sources do not play.
func NewAudioSystem(controller Controller, mixer *Mixer) core.System {
	var sourceQuery, oneshotQuery, playlistQuery, ambiguousSelectorQuery *core.Query
	var hub *signal.Hub
	return core.SystemFunc(func(ctx *core.Context) error {
		if sourceQuery == nil {
			sourceQuery = core.NewQuery(ctx.World).With(Source{}, Clip{}).Without(Playlist{})
		}
		if oneshotQuery == nil {
			oneshotQuery = core.NewQuery(ctx.World).With(Source{}, oneshot{})
		}
		if playlistQuery == nil {
			playlistQuery = core.NewQuery(ctx.World).With(Source{}, Playlist{}).Without(Clip{})
		}
		if ambiguousSelectorQuery == nil {
			ambiguousSelectorQuery = core.NewQuery(ctx.World).With(Source{}, Clip{}, Playlist{})
		}
		if hub == nil {
			resolved, err := ctx.World.Resource[*signal.Hub]()
			if err != nil {
				return fmt.Errorf("audio system: resolve signal hub: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
			}
			hub = resolved
		}
		for e := range ambiguousSelectorQuery.Execute() {
			return fmt.Errorf("audio: entity %d has both Clip and Playlist selectors", e.ID())
		}
		for e := range sourceQuery.Execute() {
			s, _ := e.Component[Source]()
			cl, _ := e.Component[Clip]()

			restart := s.syncedRestarts != s.restarts ||
				s.syncedPlaylist || s.syncedAudio != cl.Audio
			if restart {
				s.Completed = false
				s.syncedRestarts = s.restarts
				s.syncedAudio = cl.Audio
				s.syncedPlaylist = false
			}

			want := fold(s, mixer)
			want.Restart = restart
			live, err := controller.Sync(e.ID(), cl.Audio, want)
			if err != nil {
				return fmt.Errorf("audio: entity %d: %w", e.ID(), err)
			}

			changed := restart
			if !live && s.Loop == LoopNone && !s.Completed {
				s.Completed = true
				changed = true
				hub.Emit(Finished{Entity: e.ID(), Audio: cl.Audio})
			}
			if changed {
				e.SetComponent(s)
			}
		}

		for e := range playlistQuery.Execute() {
			s, _ := e.Component[Source]()
			pl, _ := e.Component[Playlist]()

			current := pl.Current()
			if current == "" {
				if s.syncedPlaylist {
					s.Completed = false
					s.syncedAudio = ""
					s.syncedPlaylist = false
					s.syncedPlaylistVersion = 0
					e.SetComponent(s)
				}
				continue
			}
			restart := s.syncedRestarts != s.restarts ||
				!s.syncedPlaylist || s.syncedPlaylistVersion != pl.selectionVersion ||
				s.syncedAudio != current
			if restart {
				s.Completed = false
				s.syncedRestarts = s.restarts
				s.syncedAudio = current
				s.syncedPlaylist = true
				s.syncedPlaylistVersion = pl.selectionVersion
			}

			want := fold(s, mixer)
			want.Restart = restart
			live, err := controller.Sync(e.ID(), current, want)
			if err != nil {
				return fmt.Errorf("audio: entity %d: %w", e.ID(), err)
			}

			changed := restart
			if !live && s.Loop == LoopNone && !s.Completed {
				s.Completed = true
				changed = true
				hub.Emit(Finished{Entity: e.ID(), Audio: current})
				if pl.advanceAfterCompletion() {
					e.SetComponent(pl)
					s.Completed = false
				}
			}
			if changed {
				e.SetComponent(s)
			}
		}

		var done []*core.Entity
		for e := range oneshotQuery.Execute() {
			s, _ := e.Component[Source]()
			if !s.Completed {
				continue
			}
			shot, _ := e.Component[oneshot]()
			done = append(done, shot.self)
		}
		for _, entity := range done {
			if err := ctx.World.DestroyEntity(entity); err != nil {
				return fmt.Errorf("audio: destroy one-shot: %w", err)
			}
		}

		if err := controller.Sweep(); err != nil {
			return fmt.Errorf("audio: sweep: %w", err)
		}
		return nil
	})
}
