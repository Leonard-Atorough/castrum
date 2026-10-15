// Package audioassets embeds the recordings shared by the audio examples.
// Asset attribution is recorded in CREDITS.md at the repository root.
package audioassets

import "embed"

// Files contains the sound effect and music track.
//
//go:embed "Blade Recall Magic 02.wav" arpmedia-retro-arcade-game-music-577821.mp3
var Files embed.FS

// Magic is the eagerly loaded sound effect.
const Magic = "Blade Recall Magic 02.wav"

// Music is the streamed music track.
const Music = "arpmedia-retro-arcade-game-music-577821.mp3"
