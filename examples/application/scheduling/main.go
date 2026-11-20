// Command scheduling demonstrates explicit ordering within a user phase.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
)

func run(out io.Writer) error {
	g, err := castrum.New()
	if err != nil {
		return err
	}
	var sampled, applied int
	if err := g.AddSystem(core.PhaseFrame, "apply", core.SystemFunc(func(*core.Context) error {
		applied = sampled * 2
		_, err := fmt.Fprintln(out, "apply:", applied)
		return err
	})); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFrame, "report", core.SystemFunc(func(*core.Context) error {
		_, err := fmt.Fprintln(out, "report:", applied)
		return err
	})); err != nil {
		return err
	}
	// Relative insertion targets must already exist in the same user phase.
	if err := g.AddSystem(core.PhaseFrame, "sample", core.SystemFunc(func(*core.Context) error {
		sampled = 21
		_, err := fmt.Fprintln(out, "sample:", sampled)
		return err
	}), castrum.Before("apply")); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFrame, "audit", core.SystemFunc(func(*core.Context) error {
		_, err := fmt.Fprintln(out, "audit:", applied)
		return err
	}), castrum.After("apply")); err != nil {
		return err
	}
	if err := g.Startup(); err != nil {
		return err
	}
	return g.Advance(0)
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
