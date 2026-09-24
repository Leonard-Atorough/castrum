package asset

import "fmt"

// registerDefaults wires the engine's built-in decoders. A failure here
// can only be an engine authoring bug — a duplicate in this static
// table — so it panics: no caller is positioned to recover from it
// (Must-style, the same rule as template.Must). User codecs go through
// [Asset.RegisterDecoder] and get errors instead.
func (a *Asset) registerDefaults() {
	for _, format := range []Format{FormatPNG, FormatJPG, FormatJPEG} {
		mustRegister(a.RegisterDecoder(format, decodeTexture, false))
	}
	mustRegister(a.RegisterDecoder(FormatJSON, decodeAtlasMeta, false))
}

func mustRegister(err error) {
	if err != nil {
		panic(fmt.Sprintf("asset: default decoder registration failed: %v", err))
	}
}
