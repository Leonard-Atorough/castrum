# Getting started with audio

In this tutorial, we'll guide you through adding a looping music track and a sound effect that plays when you press the spacebar. By the end, you'll have a basic understanding of how to work with audio in Castrum.

## Before you begin

You'll need a Go module that uses the Castrum version matching this documentation, a compatible Go toolchain, and a working audio output. Use a new or empty project so the `main.go` below doesn't conflict with an existing game.

Add two audio files to the project:

- `audio/music.mp3`: a music track.
- `audio/effect.wav`: a sound effect for later in the tutorial.

You can use your own files. If you choose different paths or formats, update the asset paths in the code to match. By default, Castrum resolves these paths relative to the directory you run the game from.

Before writing any audio code, let's set up the basics of a Castrum game. In your project directory, create a `main.go` file:

```go
package main

import (
	"log"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/ebitrun"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	game, err := castrum.New(castrum.WithTitle("Audio tutorial"))
	if err != nil {
		return err
	}

	runner, err := ebitrun.New(game)
	if err != nil {
		return err
	}

    // We'll add the audio player after this point.

	return game.Run(runner)
}
```

## Add an audio player

To play audio, all you need is an audio player component on an entity. A player has two main parts:

- `audio.Source`: a sequence of asset IDs, called tracks, that the player can play.
- `audio.Settings`: playback settings such as volume, looping, and loading mode.

Let's add an audio player to our game. Add `"github.com/Leonard-Atorough/castrum/audio"` to the imports in `main.go`. Then add the following code inside `run`, after creating the runner and before `return game.Run(runner)`:

```go
music, err := audio.NewPlayer(
	audio.Track("audio/music.mp3"),
	audio.Settings{
		Volume: 1,
		Bus:  audio.BusName("music"),
		Load:   audio.LoadStream,
		Loop:   audio.LoopTrack,
	},
)
if err != nil {
	return err
}

if _, err := game.World().NewEntity(music); err != nil {
	return err
}
```

The player streams the music and loops the track. The entity keeps the player in the world while the game runs.

That's it! Run the game from your project directory:

```sh
go run .
```

You should hear your music track looping away until you close the game window.

## Control the audio player

Chances are you don't want the music to play indefinitely until you close the game. Let's add controls to pause and resume it with the P key.

First, keep a handle to the music entity: replace the `NewEntity` block in `run` with the following code.

```go
musicEntity, err := game.World().NewEntity(music)
if err != nil {
	return err
}
```

Add `"github.com/Leonard-Atorough/castrum/core"` and `"github.com/Leonard-Atorough/castrum/input"` to the imports. Then add this function outside `run`:

```go
func audioControls(musicEntity core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		current, _ := ctx.World.Resource[*input.Snapshot]()
		if current.KeyPressed(input.KeyP) {
			return musicEntity.Update(ctx.World, func(player *audio.Player) {
				player.Paused = !player.Paused
			})
		}
		return nil
	})
}
```

Register the system inside `run`, after creating the music entity and before `return game.Run(runner)`:

```go
if err := game.AddSystem(core.PhaseFrame, "audio controls", audioControls(musicEntity)); err != nil {
	return err
}
```

The frame system checks for P and updates the player on the music entity. Add both the system and its registration before running the game, so `musicEntity` is used.

Run the game again and press the P key to pause and resume the music.

## Play a sound effect

What's a game without a sound effect? Let's play one when you press Space. Upate `audioControls` to this version:

```go
func audioControls(musicEntity core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		current, _ := ctx.World.Resource[*input.Snapshot]()
		if current.KeyPressed(input.KeySpace) {
			if _, err := audio.OneShot(ctx.World, audio.Track("audio/effect.wav"), audio.DefaultSettings()); err != nil {
				return err
			}
		}
		if current.KeyPressed(input.KeyP) {
			return musicEntity.Update(ctx.World, func(player *audio.Player) {
				player.Paused = !player.Paused
			})
		}
		return nil
	})
}
```

Each call to `audio.OneShot` creates a temporary player that the audio system removes when playback finishes. Run the game and give Space a few taps: the effects can overlap, while P still pauses and resumes the music.

## Next steps

That's your first bit of audio in Castrum! You've played and controlled music and added a sound effect. To explore the API further, run `go doc github.com/Leonard-Atorough/castrum/audio` from your project.
