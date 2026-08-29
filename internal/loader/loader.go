package loader

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Loader[T any] interface {
	Load(reader io.Reader) (T, error)
	LoadFromPath(path string) (T, error)
	LoadFromString(data string) (T, error)
	LoadFromBytes(data []byte) (T, error)
}

var loaders = map[string]func() any{
	"yaml": func() any { return YAMLLoader[any]{} },
}

func For[T any](format string) Loader[T] {
	if loaderFunc, ok := loaders[format]; ok {
		loader := loaderFunc()
		if typedLoader, ok := loader.(Loader[T]); ok {
			return typedLoader
		}
	}
	// If we don't have a loader for the format, or if the loader can't be cast to Loader[T], return a default loader that returns an error.
	return &defaultLoader[T]{format: format}
}

type defaultLoader[T any] struct {
	format string
}

func (dl *defaultLoader[T]) Load(reader io.Reader) (T, error) {
	return zero[T](), fmt.Errorf("no loader for format %s", dl.format)
}

func (dl *defaultLoader[T]) LoadFromPath(path string) (T, error) {
	return zero[T](), fmt.Errorf("no loader for format %s", dl.format)
}

func (dl *defaultLoader[T]) LoadFromString(data string) (T, error) {
	return zero[T](), fmt.Errorf("no loader for format %s", dl.format)
}

func (dl *defaultLoader[T]) LoadFromBytes(data []byte) (T, error) {
	return zero[T](), fmt.Errorf("no loader for format %s", dl.format)
}

type YAMLLoader[T any] struct{}

func (yl YAMLLoader[T]) Load(reader io.Reader) (T, error) {
	data := zero[T]()
	decoder := yaml.NewDecoder(reader)
	err := decoder.Decode(&data)
	return data, err
}

func (yl YAMLLoader[T]) LoadFromPath(path string) (T, error) {

	file, err := os.Open(path)
	if err != nil {
		return zero[T](), err
	}
	defer file.Close()

	return yl.Load(file)
}

func (yl YAMLLoader[T]) LoadFromString(data string) (T, error) {
	return yl.Load(strings.NewReader(data))
}

func (yl YAMLLoader[T]) LoadFromBytes(data []byte) (T, error) {
	return yl.Load(bytes.NewReader(data))
}

func zero[T any]() T {
	var zeroValue T
	return zeroValue
}
