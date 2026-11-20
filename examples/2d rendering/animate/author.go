package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/spritesheet"
)

func authorSheet(name string) (err error) {
	loader, err := asset.NewLoader(files, asset.LoaderOptions{})
	if err != nil {
		return err
	}
	texture, err := loader.Load[asset.TextureData]("torch_light.png")
	if err != nil {
		return err
	}
	sheet, err := spritesheet.Grid("torch_light.png", texture.Image.Bounds(), 16, 28, "torch_light")
	if err != nil {
		return err
	}
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
