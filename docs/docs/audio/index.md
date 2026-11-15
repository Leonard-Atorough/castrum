# Audio

Castrum lets you add music and sound effects to your game. You can stream audio, play one-shot effects, mix sounds, and position them in a 2D world.

## Before you start

The Ebitengine runner supports WAV, MP3, and OGG Vorbis files; other runners may support different formats. It uses Ebitengine's decoders, so their input constraints apply. For example, WAV files must use 8-bit or 16-bit little-endian PCM with one or two channels. For details, see the Ebitengine documentation for [WAV](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/wav#DecodeWithSampleRate), [MP3](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/mp3#DecodeWithSampleRate), and [Ogg Vorbis](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio/vorbis#DecodeWithSampleRate). Those links show the latest published documentation; check the Ebitengine version used by your project for its exact constraints.

Audio paths are relative to the game's configured filesystem, which defaults to the current working directory. Eager audio can be preloaded after creating the runner; otherwise, loading starts when playback begins. Missing files and decoding errors can therefore surface during playback. Streaming opens a file for each play and requires a seekable source.

The runner resamples audio to its audio context's rate, which defaults to 44,100 Hz when the runner creates the context. For spatial audio, enable a listener with a positive, finite range and give each spatial player entity a `core.Transform`; `audio.OneShotAt` supplies the transform for a positioned one-shot. A camera-following listener also needs a primary camera and a positive logical viewport.

## Learn audio basics

Start with [Getting started with audio](./getting-started-with-audio.md) to add music and a sound effect to a game.

For API details, run `go doc github.com/Leonard-Atorough/castrum/audio` from your project.
