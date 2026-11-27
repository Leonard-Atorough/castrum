package audio

import (
	"errors"
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

// Finished is the signal the audio system emits on the update that observes
// a track finishing. [Player.Completed] reports when the player has no next
// track. A [LoopTrack] play does not finish.
//
// The game reconciles audio after frame systems when a controller is attached.
// Read Finished in the post-frame phase before the frame window closes, or poll
// [Player.Completed] when the exact frame does not matter.
type Finished struct {
	// Entity is the entity whose Source finished. A [OneShot]'s entity is
	// destroyed on the same update, so its ID identifies the play only.
	Entity core.Entity
	// Audio is the asset ID of the finished play.
	Audio asset.ID
}

// Playback is the resolved state that a [Controller] applies to a play.
// [NewAudioSystem] derives it from a [Player], a [Mixer], and optionally a [Listener].
type Playback struct {
	// Playing reports whether playback should advance. False holds its position.
	Playing bool
	// Restart requests a fresh player from the beginning. The audio system
	// sets it on the first sync, a track selection change, or
	// [Player.Restart].
	Restart bool
	// Volume is the final mix, including spatial attenuation, in [0, 1].
	// Zero is silence: runtime fade targets are safe here.
	Volume float64
	// Rate is the resolved playback multiplier; 1 is the normal rate.
	Rate float64
	// Loop is LoopTrack to repeat this track, or LoopNone to play it once.
	// Sequence repetition is managed by the audio system.
	Loop LoopMode
	// Load selects eager decode or per-play streaming.
	Load LoadMode
	// Balance is the resolved channel attenuation, independent of Volume.
	Balance float64
}

func fold(s Player, mixer *Mixer) Playback {
	loop := LoopNone
	if s.Settings.Loop == LoopTrack {
		loop = LoopTrack
	}
	volume := s.Settings.Volume * mixer.MasterVolume() * mixer.BusVolume(s.Settings.Bus)
	if mixer.MasterMuted() || mixer.BusMuted(s.Settings.Bus) {
		volume = 0
	}
	return Playback{
		Playing: !s.Paused && !mixer.busLocallyPaused(s.Settings.Bus) && (!mixer.AllPaused() || s.Settings.Pause == PauseContinues),
		Volume:  volume,
		Rate:    resolveMultiplier(s.Settings.PlaybackMultiplier),
		Loop:    loop,
		Load:    s.Settings.Load,
		Balance: s.Settings.Balance,
	}
}

func resolveMultiplier(multiplier float64) float64 {
	if multiplier == 0 {
		return 1
	}
	return multiplier
}

// Controller applies resolved [Playback] state to backend players. Runners
// implement it and attach it to the game's [Backend].
type Controller interface {
	// Sync applies want to the player for entityID and source. live reports
	// whether the play has not finished, not whether it is currently playing:
	// a paused play is live, while a finished [LoopNone] play is not.
	//
	// A finished play must not resume merely because want.Playing is true;
	// want.Restart requests an explicit restart. The audio system calls Sync
	// for each [Player] on every run.
	Sync(entityID core.EntityID, source asset.ID, want Playback) (live bool, err error)
	// Sweep releases players for entities not passed to Sync since the
	// previous sweep. The audio system calls it after each reconciliation
	// pass in which all players were synced, including spatial-error passes.
	Sweep() error
}

// NewAudioSystem returns a system that advances mixer fades and reconciles players.
// It validates each player's settings before applying them to the controller.
// If controller is nil, the system advances fades without reconciling players.
func NewAudioSystem(controller Controller) core.SystemFunc {
	s := &system{
		controller: controller,
	}
	return s.Update
}

type system struct {
	controller Controller

	players  *core.Query
	oneshots *core.Query
	hub      *signal.Hub
	camera   *render.CameraResolver
}

func (s *system) Update(ctx *core.Context) error {

	mixer, err := ctx.World.Resource[*Mixer]()
	if err != nil {
		return fmt.Errorf("audio system: resolve *Mixer: %w", err)
	}
	if s.controller == nil {
		return nil
	}
	if s.camera == nil {
		s.camera = render.NewCameraResolver(ctx.World)
	}
	spatial := spatialPass{ctx: ctx, camera: s.camera}
	var spatialErr error
	if s.players == nil {
		s.players = core.NewQuery(ctx.World).With(Player{})
	}
	if s.oneshots == nil {
		s.oneshots = core.NewQuery(ctx.World).With(Player{}, oneshot{})
	}
	if s.hub == nil {
		resolved, err := ctx.World.Resource[*signal.Hub]()
		if err != nil {
			return fmt.Errorf("audio system: resolve signal hub: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
		}
		s.hub = resolved
	}

	if err := mixer.update(ctx.DeltaTime); err != nil {
		return fmt.Errorf("audio system: advance mixer: %w", err)
	}

	var done []core.Entity
	for e := range s.players.Execute() {
		p, _ := e.Component[Player]()
		if err := p.Settings.Validate(); err != nil {
			return fmt.Errorf("audio: entity %d: %w", e.ID(), err)
		}
		if p.current < 0 || p.current >= p.source.Len() {
			return fmt.Errorf("audio: entity %d: track index %d out of range for %d tracks", e.ID(), p.current, p.source.Len())
		}
		audio := p.source.At(p.current)

		restart := !p.synced || p.syncedRestarts != p.restarts ||
			p.syncedSelection != p.selectionVersion ||
			p.syncedAudio != audio

		want := fold(p, mixer)
		if err := spatial.apply(e.Entity(), p.Settings, &want); err != nil {
			if spatialErr == nil {
				spatialErr = err
			}
			want.Playing = false
			want.Restart = false
			if _, err := s.controller.Sync(e.ID(), audio, want); err != nil {
				return fmt.Errorf("audio: entity %d: %w", e.ID(), err)
			}
			continue
		}
		want.Restart = restart
		live, err := s.controller.Sync(e.ID(), audio, want)
		if err != nil {
			return fmt.Errorf("audio: entity %d: %w", e.ID(), err)
		}

		if restart {
			p.completed = false
			p.synced = true
			p.syncedRestarts = p.restarts
			p.syncedSelection = p.selectionVersion
			p.syncedAudio = audio
		}
		changed := restart
		if !live && p.Settings.Loop != LoopTrack && !p.completed {
			p.completed = true
			changed = true
			s.hub.Emit(Finished{Entity: e.Entity(), Audio: audio})
			if p.current+1 < p.source.Len() || p.Settings.Loop == LoopAll {
				p.Next()
				p.completed = false
			}
		}
		if changed {
			e.SetComponent(p)
		}
	}

	for e := range s.oneshots.Execute() {
		p, _ := e.Component[Player]()
		if !p.completed {
			continue
		}
		done = append(done, e.Entity())
	}
	for _, entity := range done {
		if err := ctx.World.DestroyEntity(entity); err != nil {
			return fmt.Errorf("audio: destroy one-shot: %w", err)
		}
	}

	if err := s.controller.Sweep(); err != nil {
		return errors.Join(spatialErr, fmt.Errorf("audio: sweep: %w", err))
	}
	return spatialErr
}
