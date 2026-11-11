package asset

import "fmt"

// registerDefaults treats duplicate built-in registrations as an engine
// authoring error and panics. User codecs go through
// [Loader.RegisterDecoder] and return errors instead.
func (l *Loader) registerDefaults() {
	for _, format := range []Format{FormatPNG, FormatJPG, FormatJPEG} {
		mustRegister(l.RegisterDecoder(format, decodeTexture, false))
	}
	mustRegister(l.RegisterDecoder(FormatJSON, decodeAtlasMeta, false))
	for _, format := range []Format{FormatTTF, FormatOTF} {
		mustRegister(l.RegisterDecoder(format, decodeFont, false))
	}
}

func mustRegister(err error) {
	if err != nil {
		panic(fmt.Sprintf("asset: default decoder registration failed: %v", err))
	}
}
