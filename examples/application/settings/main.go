// Command settings restores, validates, applies, and saves game-owned
// preferences. Run with go run .; -directory selects a different save root.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/settings"
)

type preferences struct {
	TextSize int      `json:"text_size"`
	Hints    []string `json:"hints"`
}

func (p preferences) Validate() error {
	if p.TextSize <= 0 {
		return fmt.Errorf("text size must be positive")
	}
	return nil
}

func (p preferences) Clone() preferences {
	p.Hints = slices.Clone(p.Hints)
	return p
}

func apply(out io.Writer, prefs *settings.Group[preferences]) error {
	value, err := prefs.Value()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Text size: %d, hints: %v\n", value.TextSize, value.Hints)
	return err
}

func run(root string, out io.Writer) (err error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	directory, err := asset.NewDirectory(root)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, directory.Close())
	}()

	codecs := asset.NewCodecs()
	if err := codecs.SetJSON[preferences](); err != nil {
		return err
	}
	loader, err := asset.NewLoader(directory, asset.LoaderOptions{
		Codecs: codecs, CachePolicy: asset.CacheBypass,
	})
	if err != nil {
		return err
	}
	saver, err := asset.NewSaver(directory, asset.SaverOptions{Codecs: codecs})
	if err != nil {
		return err
	}
	value, err := loader.Load[preferences]("preferences.json")
	if errors.Is(err, fs.ErrNotExist) {
		value = preferences{TextSize: 16, Hints: []string{"move"}}
	} else if err != nil {
		return fmt.Errorf("restore preferences: %w", err)
	}
	prefs, err := settings.New(value, settings.Policy[preferences]{
		Validator: settings.ValidatorFunc[preferences](preferences.Validate),
		Cloner:    settings.ClonerFunc[preferences](preferences.Clone),
	})
	if err != nil {
		return err
	}
	if err := apply(out, prefs); err != nil {
		return err
	}
	if err := prefs.Update(func(p preferences) preferences {
		p.TextSize = 20
		if !slices.Contains(p.Hints, "jump") {
			p.Hints = append(p.Hints, "jump")
		}
		return p
	}); err != nil {
		return err
	}
	if err := apply(out, prefs); err != nil {
		return err
	}
	snapshot, err := prefs.Value()
	if err != nil {
		return err
	}
	if err := saver.Save("preferences.json", snapshot); err != nil {
		return fmt.Errorf("save preferences: %w", err)
	}
	_, err = fmt.Fprintf(out, "Saved preferences to %s\n", filepath.Join(root, "preferences.json"))
	return err
}

func main() {
	root := flag.String("directory", "", "save directory (default: castrum-examples/settings beneath the OS user config directory)")
	flag.Parse()
	if *root == "" {
		config, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		*root = filepath.Join(config, "castrum-examples", "settings")
	}
	if err := run(*root, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
