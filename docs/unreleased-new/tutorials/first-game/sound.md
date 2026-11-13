# Adding sound

You have a complete round from [victory or defeat](victory-or-defeat.md). If you'd like to add a little feedback, this optional chapter adds looping music and a sound for each shot.

## Load the audio

Create `sound.go` next to the other Go files:

```go
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
)

const (
	musicFile = "audio/arpmedia-retro-arcade-game-music-577821.mp3"
	shotFile  = "audio/GUNArtl_Rocket Launcher Fire_02.wav"
)

func addSound(g *castrum.Game) error {
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData](shotFile); err != nil {
		return fmt.Errorf("load shot sound: %w", err)
	}
	music := audio.NewSource()
	music.Volume = 0.5
	music.Group = audio.GroupMusic
	music.Loop = audio.LoopForever
	music.Load = audio.LoadStream
	music.Pause = audio.PauseHolds
	if _, err := g.World().NewEntity(music, audio.Clip{Audio: musicFile}); err != nil {
		return fmt.Errorf("create music: %w", err)
	}
	return nil
}

func playShot(world *core.World) error {
	source := audio.NewSource()
	source.Pause = audio.PauseContinues
	if _, err := audio.OneShot(world, source, audio.Clip{Audio: shotFile}); err != nil {
		return fmt.Errorf("play shot: %w", err)
	}
	return nil
}
```

In `main.go`, call `addSound` **after** the `ebitrun.New(g)` error check and before `return g.Run(runner)`:

```go
	if err := addSound(g); err != nil {
		return err
	}
```

The runner registers the audio decoders, so the shot preload must happen after the runner is created. Preloading makes a missing or invalid shot file fail at setup. The music uses streaming instead; its file is opened for playback, so a problem with it can surface later.

## Play the shot

In `shots.go`, find `spawnShot`. At the end of the function, immediately after the successful `world.NewEntity` call, replace its final `return nil` with:

```go
	return playShot(world)
```

Run `go run .` again. The music should loop while you play, and each accepted click should make a firing sound. `audio.OneShot` creates a temporary audio entity and removes it when playback finishes; the looping music is a separate, persistent entity.

If you don't hear anything, check that the `audio/` folder is in the project directory and that your system's audio output is available. The existing [audio guide](../../../unreleased/guides/audio.md) covers audio loading, volume, and playback in more depth.

## Where to go next

You made a game from a world of entities and components, systems that update them, and a runner that presents the result. Try changing a few values: make enemies faster, add another starting position and raise `targetScore`, or change the shot speed. If you add an enemy without changing the target score, the round can end before every enemy is destroyed.

When you're ready to explore further:

- The [finished tank example](../../../../examples/your-first-game/main.go) adds a reload timer, waves that get faster, and more on-screen feedback. Its source is a complete program, but it isn't identical to the smaller game you just built.
- Learn more about [placing images](../../guides/rendering/images.md), [using cameras](../../guides/rendering/cameras.md), and [animating spritesheets](../../guides/rendering/spritesheets.md).
- The existing [input guide](../../../unreleased/guides/input.md), [collision guide](../../../unreleased/guides/collision.md), and [ECS guide](../../../unreleased/guides/ecs.md) cover more ways to build game behavior.
- Return to the [manual](../../README.md) to find assets, audio, project setup, and other topics.
