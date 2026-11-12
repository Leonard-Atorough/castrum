// Package asset loads and saves typed data using format-specific codecs.
// Loader caches decoded values from an fs.FS; Saver encodes documents through
// a SaveStore or caller-owned writer. Codecs supplies reusable registration
// configuration without sharing caches or runtime state. Paths are relative
// and slash-separated. Media values stay backend-free; runners own conversion
// to backend textures, fonts, and audio.
package asset
