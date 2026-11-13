// Command rebinding restores controls and edits them through a confirm/cancel draft.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

func newGame(directory *asset.Directory) (*castrum.Game, *editor, error) {
	e, err := restore(directory)
	if err != nil {
		return nil, nil, err
	}
	value, err := e.prefs.Value()
	if err != nil {
		return nil, nil, err
	}
	g, err := castrum.New(castrum.WithTitle("castrum - persistent rebinding"), castrum.WithInputPreferences(value))
	if err != nil {
		return nil, nil, err
	}
	if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
		return nil, nil, err
	}
	if err := g.MainCamera().Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: 560, Y: 200}
	}); err != nil {
		return nil, nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "edit controls", core.SystemFunc(e.update)); err != nil {
		return nil, nil, err
	}
	labels := make([]*core.Entity, 4)
	for line := range labels {
		label, err := g.World().NewEntity(
			core.Transform{Position: geom.Vector2{X: 560, Y: float64(80 + line*70)}},
			render.Sprite{Drawable: render.TextSource{Font: builtin.Font, Size: 20}},
		)
		if err != nil {
			return nil, nil, err
		}
		labels[line] = label
	}
	if err := g.AddSystem(core.PhaseFrame, "show controls", core.SystemFunc(func(ctx *core.Context) error {
		var keys []string
		for _, bound := range ctx.Actions.Bindings()["ping"] {
			if bound.Key != nil {
				keys = append(keys, bound.Key.Key.String())
			}
		}
		lines := []string{
			"F2: edit ping   ENTER: confirm + save   ESC: cancel",
			fmt.Sprintf("Active keyboard: %s | Ping count: %d", strings.Join(keys, ", "), e.activations),
			fmt.Sprintf("Editing: %t | Candidate: %s", e.editing, e.candidate),
			e.notice,
		}
		for i, label := range labels {
			if err := label.Update(ctx.World, func(sprite *render.Sprite) {
				sprite.Drawable = render.TextSource{Font: builtin.Font, Text: lines[i], Size: 20}
			}); err != nil {
				return err
			}
		}
		return nil
	})); err != nil {
		return nil, nil, err
	}
	return g, e, nil
}

func run(root string) (err error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	directory, err := asset.NewDirectory(root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	g, _, err := newGame(directory)
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithWindowSize(1120, 400), ebitrun.WithLogicalSize(1120, 400))
	if err != nil {
		return err
	}
	return g.Run(runner)
}

func main() {
	root := flag.String("directory", "", "save directory (default: castrum-examples/rebinding beneath the OS user config directory)")
	flag.Parse()
	if *root == "" {
		config, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		*root = filepath.Join(config, "castrum-examples", "rebinding")
	}
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
