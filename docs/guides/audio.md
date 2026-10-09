# Audio

Audio is state on the world. An `audio.Source` identifies an asset and describes how it should play; the engine reconciles that state with the runner's players. You do not call a backend player directly. Write the source state, and the runner catches up on the next tick.

There are two common shapes:

- Use `audio.OneShot` for a short effect that the engine should reclaim after it finishes.
- Attach `audio.Source` to a game entity for music, ambience, or any play whose lifetime you control.

The complete runnable example is [examples/audio](../../examples/audio). It plays an eager sound effect, streams looping music, adjusts mixer volumes, and demonstrates global pause behavior.

## Play a sound effect

For a fire-and-forget effect, create a source with `audio.NewSource` and pass it to `audio.OneShot`:

```go
if ctx.Input.KeyPressed(input.KeySpace) {
	shot := audio.NewSource(AssetMagic)
	shot.Pause = audio.PauseContinues
	if _, err := audio.OneShot(ctx.World, shot); err != nil {
		return err
	}
}
```

`NewSource` supplies the required unity volume. `OneShot` spawns an engine-owned entity and destroys it after a non-looping play finishes. Every call owns a separate play, so overlapping effects layer instead of interrupting one another.

A one-shot must not use `LoopForever`; a play that never finishes could never be reclaimed. Use a regular `Source` on a game entity for looping audio or for a play you need to keep and control.

## Play music or ambience

Create a `Source` directly when the entity should remain in your world. This example puts a looping track on the music bus and streams it from its file:

```go
_, err := g.World().NewEntity(audio.Source{
	Audio:  AssetMusic,
	Volume: 1,
	Group:  audio.GroupMusic,
	Loop:   audio.LoopForever,
	Load:   audio.LoadStream,
	Pause:  audio.PauseHolds,
})
if err != nil {
	return err
}
```

The entity owns the playback state. Destroying it, removing its `Source`, or stopping the source causes the runner to release its player. An entity can also carry other components, making it useful for sounds attached to a character, vehicle, or zone.

## Choose how audio loads

`Source.Load` selects how audio reaches the player. `audio.LoadEager` is the zero value; `audio.LoadStream` is the explicit streaming choice.

### Eager loading

Eager loading decodes the complete file into cached `asset.AudioData`. Multiple players can share those decoded samples, so it is a good fit for short effects that play often:

```go
shot := audio.NewSource("audio/confirm.ogg")
// LoadEager is the zero value, so this is optional.
shot.Load = audio.LoadEager
```

Preload required effects after creating the runner if you want missing files and decode errors to surface during setup:

```go
runner, err := ebitrun.New(g)
if err != nil {
	return err
}

if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData]("audio/confirm.ogg"); err != nil {
	return err
}
```

The runner registers the WAV, MP3, and OGG audio decoders when `ebitrun.New` is called. Loading `asset.AudioData` before that registration fails because the decoder is not available yet. The [assets guide](assets.md) explains typed asset loading in more detail.

### Streaming loading

Streaming opens and decodes a source for each play instead of placing the entire decoded track in the asset cache. Use it for long music or ambience that would be wasteful to hold as one PCM buffer:

```go
music := audio.NewSource("audio/music.ogg")
music.Group = audio.GroupMusic
music.Load = audio.LoadStream
music.Loop = audio.LoopForever
```

The `ebitrun` provider opens the file, chooses the WAV, MP3, or OGG decoder, and keeps the source open for that play. It closes the source when the play is released. Streaming playback is lazy, so a missing file, unsupported format, decode error, or non-seekable source appears when the runner creates the player.

Looping and position resets require a seek-capable file. Normal directory files and `embed.FS` files satisfy this requirement; verify custom filesystem implementations before using them for streaming. See [Publishing your game](publishing-your-game.md) for the embedded-filesystem pattern.

## Configure a source

`audio.Source` has one required field and several useful defaults:

- `Audio` is the asset ID. It must not be empty.
- `Volume` is the per-play level in `(0, 1]`. Use mixer levels for muting; zero is rejected at spawn.
- `Group` selects `GroupSFX` or `GroupMusic`. The zero value is `GroupSFX`.
- `Loop` selects `LoopNone` or `LoopForever`. The zero value is `LoopNone`.
- `Pause` selects `PauseHolds` or `PauseContinues`. The zero value is `PauseHolds`.
- `Load` selects eager or streaming playback. The zero value is `LoadEager`.
- `PlaybackMultiplier` scales the desired rate. Zero means normal rate; negative values are invalid.
- `Paused` holds this play's position without losing it.

`NewSource` is the easiest way to create a valid source:

```go
source := audio.NewSource("audio/step.wav")
source.Group = audio.GroupSFX
source.Volume = 0.75
```

The world validates a source when you spawn it and whenever you write it back with `SetComponent`, `Update`, or `AddComponent`. Invalid source state returns an error before it enters the world.

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

`Paused` holds the current position. `Restart` rewinds to the beginning, clears `Completed`, and resumes the play. Changing `Audio` also starts the new asset from the beginning. A finished non-looping play sets `Completed`; it does not restart by itself. Call `Restart` or change `Audio` when the play should begin again.

The backend currently used by `ebitrun` supports only a resolved playback rate of `1`. The audio component accepts positive multipliers and the engine resolves zero to `1`, but setting another multiplier with this runner returns an error when the player is created.

## Mix and pause globally

The mixer is a world resource provided by `castrum.New`. Each play's effective volume is:

```text
master volume x group volume x source volume
```

Adjust the master or one group without rewriting every source:

```go
mixer := g.World().MustResource[*audio.Mixer]()
mixer.SetMaster(mixer.Master() - step)
mixer.SetGroupVolume(
	audio.GroupMusic,
	mixer.GroupVolume(audio.GroupMusic)-step,
)
```

Mixer setters clamp their values to `[0, 1]`. An unmodified group starts at unity volume.

Use `PauseAll`, `ResumeAll`, or `TogglePause` for a global pause menu. Each source decides how that pause affects it:

- `PauseHolds` freezes the play at its current position.
- `PauseContinues` lets the play continue while the mixer is paused.

The source's own `Paused` field and the mixer's pause state compose. This lets a pause menu hold music while allowing a UI confirmation effect to play through.

## Errors and lifecycle

Return errors from setup and systems. Common failures include an empty asset ID, an invalid source field, a missing or malformed audio file, an unsupported format, and a stream that cannot seek.

Create the runner before explicit audio preloads, and register required long-lived sources before `g.Run`. Audio errors from the runner are returned through the game loop and include the source involved. A source that is merely paused remains live; a finished `LoopNone` play becomes completed and stays that way until restarted or retargeted.

## Best practices

- Use `audio.OneShot` for short effects and let the engine reclaim them.
- Keep music and ambience on persistent entities so their lifetime is explicit.
- Prefer eager loading for short, frequently reused effects.
- Prefer streaming for long tracks, provided the filesystem is seekable.
- Put effects and music on separate groups so they can be mixed independently.
- Use `PauseHolds` for music and `PauseContinues` for sounds that should play through a pause menu.
- Use `audio.NewSource` so required defaults, especially `Volume`, are present.
- Handle errors from entity creation, preload calls, and system updates instead of relying on the first audible use to reveal a problem.

## Where to go next

- [examples/audio](../../examples/audio) - eager effects, streaming music, mixer controls, and global pause.
- [Assets](assets.md) - audio formats, typed preloads, caching, and streaming files.
- [Publishing your game](publishing-your-game.md) - embedding audio in a single binary.
- [Conventions](conventions.md) - zero values, validation boundaries, and state-driven control.
