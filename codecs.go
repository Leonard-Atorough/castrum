package castrum

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/input"
)

// NewAssetCodecs returns asset defaults with JSON codecs for [audio.Preferences]
// and [input.Preferences]. Preference validation and application remain explicit.
//
// Configure custom codecs before passing the result to [WithAssetCodecs],
// [asset.NewLoader], or [asset.NewSaver]. Each call returns independent codecs.
func NewAssetCodecs() *asset.Codecs {
	codecs := asset.NewCodecs()
	if err := codecs.SetJSON[audio.Preferences](); err != nil {
		panic(fmt.Sprintf("castrum: register audio preferences: %v", err))
	}
	if err := codecs.SetJSON[input.Preferences](); err != nil {
		panic(fmt.Sprintf("castrum: register input preferences: %v", err))
	}
	return codecs
}
