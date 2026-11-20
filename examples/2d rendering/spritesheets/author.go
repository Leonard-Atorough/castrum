package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/spritesheet"
)

// authorSheet regenerates the sheet document for gopher_sheet.png: a regular
// grid of cellSize x cellSize cells named gopher_N in row-major order, plus
// the hand-added irregular region spanning two cells. It writes only metadata;
// the texture itself is never saved.
func authorSheet(name string) (err error) {
	loader, err := asset.NewLoader(files, asset.LoaderOptions{})
	if err != nil {
		return err
	}
	texture, err := loader.Load[asset.TextureData](texturePath)
	if err != nil {
		return err
	}
	sheet, err := spritesheet.Grid(texturePath, texture.Image.Bounds(), cellSize, cellSize, "gopher")
	if err != nil {
		return err
	}
	// Irregular regions sit alongside grid ones: whatever the grid did
	// not produce is authored by hand.
	sheet.Regions[wideRegion] = asset.SheetRegion{X: cellSize, Y: 0, W: 2 * cellSize, H: cellSize}

	root := filepath.Dir(name)
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
	saver, err := asset.NewSaver(directory, asset.SaverOptions{})
	if err != nil {
		return err
	}
	return sheet.Save(saver, filepath.Base(name))
}
