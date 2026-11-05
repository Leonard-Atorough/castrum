// Command settings demonstrates reading and changing game-owned preferences.
// Run from this directory with go run .
package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/Leonard-Atorough/castrum/settings"
)

type preferences struct {
	TextSize int
	Hints    []string
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

func run(out io.Writer) error {
	prefs, err := settings.New(preferences{TextSize: 16, Hints: []string{"move"}}, settings.Policy[preferences]{
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
		p.Hints = append(p.Hints, "jump")
		return p
	}); err != nil {
		return err
	}
	return apply(out, prefs)
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
