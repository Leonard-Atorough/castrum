package main

import (
	"log"

	"example.com/mygame/internal/game"
	"github.com/Leonard-Atorough/castrum/ebitrun"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	g, err := game.New()
	if err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	return g.Run(runner)
}
