package asset

import "fmt"

// registerDefaults treats duplicate built-in registrations as an engine
// authoring error and panics. User codecs go through
// [Server.RegisterDecoder] and return errors instead.
func (s *Server) registerDefaults() {
	for _, format := range []Format{FormatPNG, FormatJPG, FormatJPEG} {
		mustRegister(s.RegisterDecoder(format, decodeTexture, false))
	}
	mustRegister(s.RegisterDecoder(FormatJSON, decodeAtlasMeta, false))
	for _, format := range []Format{FormatTTF, FormatOTF} {
		mustRegister(s.RegisterDecoder(format, decodeFont, false))
	}
}

func mustRegister(err error) {
	if err != nil {
		panic(fmt.Sprintf("asset: default decoder registration failed: %v", err))
	}
}
