# Audio

This guide describes the in-development audio API and is not compatible with Castrum v0.3.0. Use the [released v0.3.0 audio guide](../../v0.3.0/guides/audio.md) for projects pinned to that version.

Audio is state on the world. An `audio.Source` describes how playback should behave; a separate `audio.Clip` or `audio.Playlist` selects what to play. The engine reconciles those components with the runner's players. You do not call a backend player directly. Reconciliation runs during frame updates, independently of simulation pause and time scale. It is inactive until a runner attaches an `audio.Controller`; the default runner does this during construction.

An entity playing audio has one `Source` and exactly one selector: `Clip` for a single asset or `Playlist` for a sequence. A `Source` without a selector does not play audio, and an entity with both selectors is rejected by the audio system.

The runnable audio examples are standalone projects pinned to v0.3.0. They show the released API, not the split API described here. Their assets are in the shared [examples/audio package](../../../examples/audio):

| Example | Run command | Focus |
| --- | --- | --- |
| [Playback](../../../examples/audio/playback) | `go -C examples/audio/playback run .` | Eager OneShots, streamed looping music, source pause and restart |
| [Mixing](../../../examples/audio/mixing) | `go -C examples/audio/mixing run .` | Master/group volume and global pause policies |
| [Balance](../../../examples/audio/balance) | `go -C examples/audio/balance run .` | Live stereo balance, updates while paused, and preservation on restart |

The parent directory contains the shared asset package, not a runnable command. Each example binary embeds the recordings and needs no external asset directory. Attribution is in [CREDITS.md](../../../CREDITS.md).

## Play a sound effect

For a fire-and-forget effect, pass a source and clip to `audio.OneShot`:

```go
if ctx.Input.KeyPressed(input.KeySpace) {
	shot := audio.NewSource()
	shot.Pause = audio.PauseContinues
	clip := audio.Clip{Audio: AssetMagic}
	if _, err := audio.OneShot(ctx.World, shot, clip); err != nil {
		return err
	}
}
```

`NewSource` supplies the required unity volume. `OneShot` spawns an engine-owned entity and destroys it after a non-looping play finishes. Every call owns a separate play, so overlapping effects layer instead of interrupting one another.

A one-shot must not use `LoopForever`; a play that never finishes could never be reclaimed. Use a regular `Source` on a game entity for looping audio or for a play you need to keep and control.

When a non-looping track finishes, the audio system emits an `audio.Finished` signal carrying the source's entity ID and asset ID. For a clip or an exhausted playlist, it also sets `Source.Completed`; an automatically advancing playlist clears it for the next track. The game reconciles audio after user frame systems, so read `Finished` in `PhasePostFrame` before the frame window closes. Poll `Source.Completed` from anywhere when the exact frame does not matter. See the [signals guide](signals.md) for reader rules.

## Play music or ambience

Create a `Source` and `Clip` on the same entity when the play should remain in your world. This example puts a looping track on the music bus and streams it from its file:

```go
music := audio.NewSource()
music.Group = audio.GroupMusic
music.Loop = audio.LoopForever
music.Load = audio.LoadStream
music.Pause = audio.PauseHolds
_, err := g.World().NewEntity(music, audio.Clip{Audio: AssetMusic})
if err != nil {
	return err
}
```

The entity owns both the playback settings and selected clip. Destroying it or removing its selector causes the runner to release its player on the next reconciliation. A finished non-looping play also releases its playback resources, but its entity remains. An entity can carry other components, making it useful for sounds attached to a character, vehicle, or zone.

## Play a playlist

Use `Playlist` when an entity should advance between assets. `NewPlaylist` copies the supplied track list and selects the first track when one is present. Each track must be a non-empty asset ID; an empty playlist selects nothing. Set `RepeatAll` to wrap automatically after the last track; the default `RepeatNone` stops at the last track:

```go
playlist := audio.NewPlaylist([]asset.ID{
	"audio/menu.ogg",
	"audio/level.ogg",
})
playlist.Repeat = audio.RepeatAll

music := audio.NewSource()
music.Group = audio.GroupMusic
music.Load = audio.LoadStream
entity, err := g.World().NewEntity(music, playlist)
if err != nil {
	return err
}
```

`Source.Loop` repeats the current asset; `Playlist.Repeat` controls whether the sequence wraps. Leave `Source.Loop` at `LoopNone` when the playlist should advance automatically. A `LoopForever` source never completes a track, so the playlist cannot advance on completion.

The audio system emits `audio.Finished` for each completed track. When another track is available, it advances the playlist and starts that selection on the next audio-system pass. At the final track with `RepeatNone`, the source becomes completed. `Playlist.Next` and `Previous` wrap when called directly, regardless of `Repeat`; `Seek` selects a position without wrapping. Write the updated component back to the entity:

```go
if err := entity.Update(g.World(), func(p *audio.Playlist) {
	p.Next()
}); err != nil {
	return err
}
```

Track edits use the same component-update pattern. `AddTrack`, `InsertTrack`, and `RemoveTrack` return errors for invalid input. `Clear` or removing the last track leaves a valid empty playlist: the next reconciliation releases the old playback without completing or replaying it. Adding a track later starts that selection from the beginning under the existing source and mixer pause rules. You can also remove the `Playlist` component or replace it with a `Clip`.

## Choose how audio loads

`Source.Load` selects how audio reaches the player. `audio.LoadEager` is the zero value; `audio.LoadStream` is the explicit streaming choice. `Clip` and each playlist entry select asset IDs independently of this loading policy.

### Eager loading

Eager loading decodes the complete file into cached `asset.AudioData`. Multiple players can share those decoded samples, so it is a good fit for short effects that play often:

```go
shot := audio.NewSource()
// LoadEager is the zero value, so this is optional.
shot.Load = audio.LoadEager
clip := audio.Clip{Audio: "audio/confirm.ogg"}
_, err := g.World().NewEntity(shot, clip)
if err != nil {
	return err
}
```

Preload required effects after creating the runner if you want missing files and decode errors to surface during setup:

```go
runner, err := ebitrun.New(g)
if err != nil {
	return err
}

if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData]("audio/confirm.ogg"); err != nil {
	return err
}
```

The runner registers the WAV, MP3, and OGG audio decoders when `ebitrun.New` is called. Loading `asset.AudioData` before that registration fails because the decoder is not available yet. The [assets guide](assets.md) explains typed asset loading in more detail.

### Streaming loading

Streaming opens and decodes a source for each play instead of placing the entire decoded track in the asset cache. Use it for long music or ambience that would be wasteful to hold as one PCM buffer:

```go
music := audio.NewSource()
music.Group = audio.GroupMusic
music.Load = audio.LoadStream
music.Loop = audio.LoopForever
clip := audio.Clip{Audio: "audio/music.ogg"}
_, err := g.World().NewEntity(music, clip)
if err != nil {
	return err
}
```

The `ebitrun` provider opens the file, chooses the WAV, MP3, or OGG decoder, and keeps the source open for that play. It closes the source when the play is released. Streaming playback is lazy, so a missing file, unsupported format, decode error, or non-seekable source appears when the runner creates the player.

Looping and position resets require a seek-capable file. Normal directory files and `embed.FS` files satisfy this requirement; verify custom filesystem implementations before using them for streaming. See [Publishing your game](publishing-your-game.md) for the embedded-filesystem pattern.

## Configure a source

`audio.Source` contains playback settings and has several useful defaults:

- `Volume` is the finite per-play level in `(0, 1]`. Use mixer levels for muting; zero and NaN are rejected.
- `Balance` attenuates the opposite stereo channel: `-1` mutes the right, `1` mutes the left, and zero preserves the original mix. Values must be finite and in `[-1, 1]`.
- `Spatial` opts the play into listener-relative volume and balance. It is off by default.
- `Attenuation` shapes spatial volume falloff; nil uses a linear curve.
- `Group` selects `GroupSFX` or `GroupMusic`. The zero value is `GroupSFX`.
- `Loop` selects `LoopNone` or `LoopForever`. The zero value is `LoopNone`.
- `Pause` selects `PauseHolds` or `PauseContinues`. The zero value is `PauseHolds`.
- `Load` selects eager or streaming playback. The zero value is `LoadEager`.
- `PlaybackMultiplier` scales the desired rate. Zero means normal rate; negative values are invalid.
- `Paused` holds this play's position without losing it.
- `Completed` is owned by the audio system and is true when a clip finishes or a playlist exhausts with `RepeatNone`.

`NewSource` is the easiest way to create a valid source:

```go
source := audio.NewSource()
source.Group = audio.GroupSFX
source.Volume = 0.75
```

`Clip.Audio` must be a non-empty asset ID. A `Playlist` may be empty, but each track ID must be non-empty and its repeat mode must be supported. The world validates each component when you spawn it and whenever you write it back with `SetComponent`, `Update`, or `AddComponent`. Invalid component state returns an error before it enters the world.

## Control playback

Playback control is ordinary component state. Query the source, change it, and write it back:

```go
for e := range sources.Execute() {
	source, _ := e.Component[audio.Source]()
	if introDone {
		source.Restart()
		e.SetComponent(source)
	}
}
```

`Paused` holds the current position. `Restart` rewinds to the beginning, clears `Completed`, and resumes the play. Changing `Clip.Audio` or selecting another playlist entry also starts that selection from the beginning. A finished non-looping play sets `Completed`; it does not restart by itself. Call `Restart` or change the selection when the play should begin again.

Changing `Balance` does not restart playback and applies independently of mixer volume, including while paused. It works with eager, streamed, and looping audio, and survives restarts. Balance does not move sound between channels: at either endpoint, content in the muted channel is lost. Already-buffered audio retains its previous balance, so updates are not instantaneous. Gain changes are not smoothed and abrupt changes can click.

In the [balance example](../../../examples/audio/balance), hold Left/Right to adjust balance or press 1/2/3 for left/center/right. Space pauses the source; change balance while paused, then resume with Space or restart with R. The displayed left/right gains are independent of source and mixer volume. Use headphones or stereo speakers to hear the difference.

The backend currently used by `ebitrun` supports only a resolved playback rate of `1`. The audio component accepts positive multipliers and the engine resolves zero to `1`, but every runner sync rejects other rates, including updates to existing, paused, or finished sessions. Rejection does not rebuild or change the session.

## Place sounds in the world

`castrum.New` provides a disabled `*audio.Listener` resource. Enable it and set a positive, finite range in world units. The default `FollowCamera` mode uses the primary camera; `FixedPosition` instead uses `Listener.Position` and world axes.

```go
listener := g.World().MustResource[*audio.Listener]()
listener.Range = 600
listener.Enabled = true

source := audio.NewSource()
source.Spatial = true
source.Attenuation = geom.EaseInQuad
source.Loop = audio.LoopForever
_, err := g.World().NewEntity(
	source,
	audio.Clip{Audio: "audio/waterfall.ogg"},
	core.Transform{Position: geom.Vector2{X: 200, Y: 40}},
)
if err != nil {
	return err
}
```

An enabled spatial source needs a `core.Transform`; its position is the sound's world position. Either a clip or a playlist can be spatial. A source without `Spatial`, or any source while the listener is disabled, plays without spatial adjustment. For a fire-and-forget spatial play, use `audio.OneShotAt`, which enables `Spatial` and attaches the transform for you:

```go
shot, err := audio.OneShotAt(ctx.World, audio.NewSource(), audio.Clip{Audio: "audio/impact.wav"}, impactPosition)
if err != nil {
	return err
}
```

Distance is measured from the listener to the source. The system clamps `distance / Range` to `[0, 1]`, passes it to `Source.Attenuation`, and subtracts the result from 1 to obtain the volume gain. A nil curve is linear; `geom.EaseInQuad` and the other easing functions offer alternatives. A custom curve must be pure, map 0 to 0 and 1 to 1, and return finite values in `[0, 1]`. Each enabled spatial application validates the curve's endpoints and sampled result; invalid results are reported during reconciliation.

The horizontal displacement divided by range produces a balance adjustment clamped to `[-1, 1]`. Camera-follow mode uses camera-local horizontal direction, including camera rotation; fixed-position mode uses world X. The system adds this adjustment to the source's authored balance and clamps the result. This is stereo channel attenuation, not true positional panning: sound in the attenuated channel is lost.

The camera listener uses the same primary-camera selection and bounds as rendering, but uses the current camera transform rather than the interpolated drawn view. Zoom does not alter range. Audio runs in the frame phase before fixed ticks; movement from those ticks reaches audio on the next frame. Other frame-system changes take effect according to their order relative to the audio system. Moving a listener or source, changing its curve, or toggling `Spatial` changes the resolved playback without restarting it, including while paused. The system does not write derived volume or balance to the source or change the transform.

An enabled listener requires a valid range and, in camera mode, a primary camera and positive logical viewport. Manual `NewAudioSystem` setups must provide a `*audio.Listener` resource when using spatial sources. Invalid listener state, missing transforms, and invalid curves are returned as audio errors. At or beyond range the play is silent, not paused, and may still finish or advance a playlist. Existing source and mixer pause, completion, restart, and one-shot cleanup rules still apply. Like balance changes, spatial updates may be buffered and are not smoothed.

## Mix and pause globally

The mixer is a world resource provided by `castrum.New`. Each play's effective volume is:

```text
master volume x group volume x source volume x spatial gain
```

Spatial gain is 1 for a non-spatial source or a disabled listener.

Adjust the master or one group without rewriting every source:

```go
mixer := g.World().MustResource[*audio.Mixer]()
if err := mixer.SetMaster(mixer.Master() - step); err != nil {
	return err
}
if err := mixer.SetGroupVolume(
	audio.GroupMusic,
	mixer.GroupVolume(audio.GroupMusic)-step,
); err != nil {
	return err
}
```

Mixer setters clamp their values to `[0, 1]`, including infinities at the corresponding endpoints. NaN returns an error without changing the current value. An unmodified group starts at unity volume. Muting a bus does not pause its plays; silent playback can still finish.

### Audio preferences

Use audio preferences when you want to keep the player's desired levels separately from the mixer's runtime state. `audio.NewPreferences` creates a group with master, music, and sound-effects levels at unity. It does not change the mixer: your game decides when and how to apply the values. If you only need runtime volume control, you can use the mixer directly without creating a preferences group.

At startup, create the group, optionally set values loaded by your game, and apply its current value to the mixer:

```go
prefs, err := audio.NewPreferences()
if err != nil {
	return err
}
value, err := prefs.Value()
if err != nil {
	return err
}
mixer := g.World().MustResource[*audio.Mixer]()
if err := mixer.SetMaster(value.Master); err != nil {
	return err
}
if err := mixer.SetGroupVolume(audio.GroupMusic, value.Music); err != nil {
	return err
}
if err := mixer.SetGroupVolume(audio.GroupSFX, value.SFX); err != nil {
	return err
}
```

Keep `prefs` wherever your settings menu needs it, or register it as a [world resource](ecs.md#resources) for other systems to resolve. The mixer is provided by `castrum.New`; the preferences group is not registered automatically.

When a music slider changes, update the desired value and explicitly apply that level:

```go
if err := prefs.Update(func(p audio.Preferences) audio.Preferences {
	p.Music = sliderValue
	return p
}); err != nil {
	return err
}
value, err := prefs.Value()
if err != nil {
	return err
}
if err := mixer.SetGroupVolume(audio.GroupMusic, value.Music); err != nil {
	return err
}
```

Preference levels must be finite and in `[0, 1]`; an invalid update leaves the group and mixer unchanged. A successful update changes only the preferences; if a later mixer call fails, the preferences remain changed. The mixer change reaches playback on the next audio reconciliation and may be buffered. Applying only the changed level avoids overwriting unrelated runtime mixer changes, but the game must still coordinate temporary effects that write that same level. Preferences are not saved automatically; see the [settings guide](settings.md) for validation, copying, and applying custom preferences.

### Pause all playback

Use `PauseAll`, `ResumeAll`, or `TogglePause` for a global pause menu. Each source decides how that pause affects it:

- `PauseHolds` freezes the play at its current position.
- `PauseContinues` lets the play continue while the mixer is paused.

The source's own `Paused` field always holds that play, including a `PauseContinues` source. Resuming the mixer does not clear individual source pauses.

`Game.Pause` stops fixed simulation, not audio playback or reconciliation. Use the mixer separately when a game pause should hold audio. Simulation time scale does not change playback speed; see [Time control](time-control.md) for the simulation controls.

## Errors and lifecycle

Return errors from setup and systems. Common failures include an empty clip or playlist track ID, an empty playlist, an invalid source field, a missing or malformed audio file, an unsupported format, and a stream that cannot seek. Attaching both a `Clip` and a `Playlist` to one `Source` is rejected by the audio system when it runs.

Create the runner before explicit audio preloads, and register required long-lived sources before `g.Run`. Audio errors from the runner are returned through the game loop and identify the entity and failure stage; they may also include the selected asset. A source that is merely paused remains live; a finished clip or a playlist at its final non-repeating track becomes completed until restarted or retargeted.

The default runner releases a completed play's player, balance reader, and streaming file once it observes playback has stopped, not merely when decoding reaches EOF. It retains a finished-session record so later reconciliation does not replay the source. Restarting or changing the clip or playlist selection creates a fresh player and reopens streamed audio. Paused and looping plays retain their playback resources.

Completion does not destroy an ordinary source entity. `OneShot` entities are destroyed during the reconciliation that observes completion; their playback resources are released then, and their finished-session records are swept on the following reconciliation.

## Best practices

- Use `audio.OneShot` for short effects and let the engine reclaim them.
- Keep music and ambience on persistent entities so their lifetime is explicit.
- Prefer eager loading for short, frequently reused effects.
- Prefer streaming for long tracks, provided the filesystem is seekable.
- Put effects and music on separate groups so they can be mixed independently.
- Use `PauseHolds` for music and `PauseContinues` for sounds that should play through a pause menu.
- Use `audio.NewSource` so the required default `Volume` is present, and attach exactly one media selector.
- Handle errors from entity creation, preload calls, and system updates instead of relying on the first audible use to reveal a problem.

## Where to go next

- [Playback example](../../../examples/audio/playback) - eager effects, streamed music, source pause and restart.
- [Mixing example](../../../examples/audio/mixing) - mixer volume and global pause policies.
- [Balance example](../../../examples/audio/balance) - live per-source stereo balance.
- [Assets](assets.md) - audio formats, typed preloads, caching, and streaming files.
- [Publishing your game](publishing-your-game.md) - embedding audio in a single binary.
- [Conventions](conventions.md) - zero values, validation boundaries, and state-driven control.
