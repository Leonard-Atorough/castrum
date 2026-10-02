package asset

// AudioData is a decoded audio asset: the full PCM buffer plus the
// metadata its decoder probed. It is the cached Load value type behind
// the runner-registered audio codecs; the sample rate and length are
// decoder outputs, not author inputs.
type AudioData struct {
	// PCM is the decoded audio buffer: 16-bit signed little-endian interleaved stereo at SampleRate.
	PCM []byte
	// SampleRate is the decoded audio sample rate.
	SampleRate int
	// Length is the decoded length in bytes, as the backend's loop helpers expect.
	Length int64
}
