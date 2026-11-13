package castrum

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
)

// NewAssetCodecs returns asset defaults with JSON codecs for [audio.Preferences].
// Preference validation and application remain explicit.
//
// Configure custom codecs before passing the result to [WithAssetCodecs],
// [asset.NewLoader], or [asset.NewSaver]. Each call returns independent codecs.
func NewAssetCodecs() *asset.Codecs {
	codecs := asset.NewCodecs()
	if err := codecs.SetJSON[audio.Preferences](); err != nil {
		panic(fmt.Sprintf("castrum: register audio preferences: %v", err))
	}
	return codecs
}
