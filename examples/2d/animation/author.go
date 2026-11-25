package main

import (
	"errors"
	"os"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/spritesheet"
)

// authorSheets regenerates the .json metadata for every strip PNG into
// dir. Each strip is a horizontal row of 96x96 frames.
func authorSheets(dir string) (err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	loader, err := asset.NewLoader(files, asset.LoaderOptions{})
	if err != nil {
		return err
	}
	for _, strip := range strips {
		texture, loadErr := loader.Load[asset.TextureData](strip.sheet[:len(strip.sheet)-len(".json")] + ".png")
		if loadErr != nil {
			return loadErr
		}
		png := asset.ID(strip.sheet[:len(strip.sheet)-len(".json")] + ".png")
		sheet, gridErr := spritesheet.Grid(png, texture.Image.Bounds(), 96, 96, strip.prefix)
		if gridErr != nil {
			return gridErr
		}
		directory, dirErr := asset.NewDirectory(dir)
		if dirErr != nil {
			return dirErr
		}
		saver, saverErr := asset.NewSaver(directory, asset.SaverOptions{})
		if saverErr != nil {
			directory.Close()
			return saverErr
		}
		saveErr := sheet.Save(saver, strip.sheet)
		err = errors.Join(err, saveErr, directory.Close())
	}
	return err
}
