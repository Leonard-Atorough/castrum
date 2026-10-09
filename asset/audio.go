package asset

// AudioData holds decoded PCM and its sample rate and length.
type AudioData struct {
	// PCM is 16-bit signed little-endian interleaved stereo audio at SampleRate.
	PCM []byte
	// SampleRate is the decoded audio sample rate.
	SampleRate int
	// Length is the decoded PCM length in bytes.
	Length int64
}
