// Package asset loads typed data using format-specific decoders.
// Loader caches decoded values from an fs.FS. Codecs supplies reusable registration
// configuration without sharing caches or runtime state. Paths are relative
// and slash-separated. Media values stay backend-free; runners own conversion
// to backend textures, fonts, and audio.
package asset
