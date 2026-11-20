package main

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/settings"
)

const controlsFile = "controls.json"

type editor struct {
	prefs       *settings.Group[input.Preferences]
	saver       *asset.Saver
	draft       input.Preferences
	candidate   input.Key
	editing     bool
	activations int
	notice      string
}

func restore(directory *asset.Directory) (*editor, error) {
	codecs := castrum.NewAssetCodecs()
	loader, err := asset.NewLoader(directory, asset.LoaderOptions{Codecs: codecs, CachePolicy: asset.CacheBypass})
	if err != nil {
		return nil, err
	}
	saver, err := asset.NewSaver(directory, asset.SaverOptions{Codecs: codecs})
	if err != nil {
		return nil, err
	}
	value, err := loader.Load[input.Preferences](controlsFile)
	if errors.Is(err, fs.ErrNotExist) {
		value = input.Preferences{
			Bindings: input.Bindings{"ping": {input.KeyInput{Key: input.KeySpace}.Binding()}},
			Deadzone: input.DefaultDeadzone,
		}
	} else if err != nil {
		return nil, fmt.Errorf("restore controls: %w", err)
	}
	prefs, err := input.NewPreferences(value.Bindings)
	if err != nil {
		return nil, err
	}
	if err := prefs.Set(value); err != nil {
		return nil, err
	}
	return &editor{prefs: prefs, saver: saver, notice: "F2 opens an owned draft; nothing is saved until confirmation."}, nil
}

func (e *editor) update(ctx *core.Context) error {
	if !e.editing {
		if ctx.Input.KeyPressed(input.KeyF2) {
			value, err := e.prefs.Value()
			if err != nil {
				return err
			}
			e.draft, e.candidate, e.editing = value, input.KeyNone, true
			e.notice = "Press a candidate key. The active binding is unchanged."
		} else if ctx.Actions.JustPressed("ping") {
			e.activations++
		}
		return nil
	}
	if ctx.Input.KeyPressed(input.KeyEscape) {
		e.editing = false
		e.notice = "Cancelled: active controls and saved file are unchanged."
		return nil
	}
	if ctx.Input.KeyPressed(input.KeyEnter) {
		if e.candidate == input.KeyNone {
			e.notice = "Choose a candidate key before confirming."
			return nil
		}
		if err := e.prefs.Set(e.draft); err != nil {
			return err
		}
		value, err := e.prefs.Value()
		if err != nil {
			return err
		}
		if err := ctx.Actions.SetPreferences(value); err != nil {
			return fmt.Errorf("apply controls: %w", err)
		}
		if err := e.saver.Save(controlsFile, value); err != nil {
			return fmt.Errorf("live controls changed but saving failed: %w", err)
		}
		e.editing = false
		e.notice = "Applied and saved. Restart the example to restore these controls."
		return nil
	}
	for key := input.Key(1); key < input.NumKeys; key++ {
		if key != input.KeyF2 && ctx.Input.KeyPressed(key) {
			e.candidate = key
			e.draft.Bindings["ping"] = []input.Input{input.KeyInput{Key: key}.Binding()}
			e.notice = "ENTER applies and saves; ESC discards this draft."
			break
		}
	}
	return nil
}
