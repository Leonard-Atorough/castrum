package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/leonard-atorough/castrum/ecs"
	internalassets "github.com/leonard-atorough/castrum/internal/assets"
	"go.yaml.in/yaml/v3"
)

// LoadResult holds the result of an async load operation.
type LoadResult struct {
	Value any
	Err   error
}

type TextureData struct {
	Image  image.Image
	Width  int
	Height int
}

type AtlasMeta struct {
	Regions []AtlasRegionMeta `yaml:"regions"`
}

type AtlasRegionMeta struct {
	Name       string `yaml:"name"`
	X, Y, W, H int    `yaml:"x,y,w,h"`
}

// Blueprint represents a reusable template for creating entities with predefined components.
type Blueprint struct {
	Name       string          `yaml:"name"`
	Components []ComponentData `yaml:"components"`
	Version    string          `yaml:"version"`
}

// ComponentData represents the data required to instantiate a component for an entity.
type ComponentData struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

type Assets struct {
	loader *Loader
	saver  *Saver
}

func NewAssets(filesystem fs.FS) *Assets {
	service := newAssetService(filesystem)
	loader := newLoader(service)
	return &Assets{
		loader: loader,
		saver:  newSaver(service, loader.Invalidate),
	}
}

func (a *Assets) AssetLoader() *Loader {
	return a.loader
}

func (a *Assets) AssetSaver() *Saver {
	return a.saver
}

// CreateFromBlueprint constructs and creates an entity from blueprint data.
func CreateFromBlueprint(world *ecs.World, registry *ComponentRegistry, bp *Blueprint) (*ecs.Entity, error) {
	components := make([]ecs.Component, len(bp.Components))
	for i, comp := range bp.Components {
		instance, err := registry.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		components[i] = instance
	}

	return world.CreateWithComponents(bp.Name, components...)
}

func registerDefaultDecoders(s *internalassets.Service) {
	registerDefaultDecoder(s, FormatYAML, decodeBlueprintYAML)
	registerDefaultDecoder(s, FormatYML, decodeBlueprintYAML)
	registerDefaultDecoder(s, FormatJSON, decodeAtlasMetaJSON)
	registerDefaultDecoder(s, FormatPNG, decodeTexture)
	registerDefaultDecoder(s, FormatJPG, decodeTexture)
	registerDefaultDecoder(s, FormatJPEG, decodeTexture)
}

func registerDefaultEncoders(s *internalassets.Service) {
	registerDefaultEncoder(s, FormatYAML, encodeBlueprintYAML)
	registerDefaultEncoder(s, FormatYML, encodeBlueprintYAML)
	registerDefaultEncoder(s, FormatPNG, encodeTexturePNG)
	registerDefaultEncoder(s, FormatJPG, encodeTextureJPEG)
	registerDefaultEncoder(s, FormatJPEG, encodeTextureJPEG)
}

func newAssetService(filesystem fs.FS) *internalassets.Service {
	service := internalassets.NewService(filesystem)
	registerDefaultDecoders(service)
	registerDefaultEncoders(service)
	return service
}

func registerDefaultDecoder[T any](service *internalassets.Service, format Format, decoder Decoder[T]) {
	if err := service.RegisterDecoder(
		reflect.TypeFor[T](),
		string(format),
		func(ctx context.Context, reader io.Reader) (any, error) {
			return decoder(ctx, reader)
		},
		false,
	); err != nil {
		panic(fmt.Sprintf("failed to register default decoder for format %s: %v", format, err))
	}
}

func registerDefaultEncoder[T any](service *internalassets.Service, format Format, encoder Encoder[T]) {
	if err := service.RegisterEncoder(
		reflect.TypeFor[T](),
		string(format),
		func(ctx context.Context, writer io.Writer, value any) error {
			return encoder(ctx, writer, value.(T))
		},
		false,
	); err != nil {
		panic(fmt.Sprintf("failed to register default encoder for format %s: %v", format, err))
	}
}

func decodeBlueprintYAML(_ context.Context, reader io.Reader) (Blueprint, error) {
	var blueprint Blueprint
	if err := yaml.NewDecoder(reader).Decode(&blueprint); err != nil {
		return Blueprint{}, err
	}
	return blueprint, nil
}

func encodeBlueprintYAML(_ context.Context, writer io.Writer, blueprint Blueprint) error {
	return yaml.NewEncoder(writer).Encode(blueprint)
}

func decodeTexture(_ context.Context, reader io.Reader) (TextureData, error) {
	decoded, _, err := image.Decode(reader)
	if err != nil {
		return TextureData{}, err
	}
	bounds := decoded.Bounds()
	return TextureData{Image: decoded, Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

func encodeTexturePNG(_ context.Context, writer io.Writer, texture TextureData) error {
	if texture.Image == nil {
		return fmt.Errorf("texture image is nil")
	}
	return png.Encode(writer, texture.Image)
}

func encodeTextureJPEG(_ context.Context, writer io.Writer, texture TextureData) error {
	if texture.Image == nil {
		return fmt.Errorf("texture image is nil")
	}
	return jpeg.Encode(writer, texture.Image, &jpeg.Options{Quality: 90})
}

func decodeAtlasMetaJSON(_ context.Context, reader io.Reader) (AtlasMeta, error) {
	var meta AtlasMeta
	if err := json.NewDecoder(reader).Decode(&meta); err != nil {
		return AtlasMeta{}, err
	}
	return meta, nil
}

// AssetError represents an error that occurred during the loading or saving of an asset.
type AssetError struct {
	Message string
	Err     error
	Source  string
}

func (e *AssetError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v (source: %s)", e.Message, e.Err, e.Source)
	}
	return fmt.Sprintf("%s (source: %s)", e.Message, e.Source)
}

func (e *AssetError) Unwrap() error {
	return e.Err
}

func resolveFormat(assetPath string, explicit Format) Format {
	if explicit != "" {
		return explicit
	}
	switch strings.TrimPrefix(strings.ToLower(pathpkg.Ext(assetPath)), ".") {
	case "yml":
		return FormatYML
	case "jpeg":
		return FormatJPEG
	default:
		return Format(strings.TrimPrefix(strings.ToLower(pathpkg.Ext(assetPath)), "."))
	}
}

// normalizeAssetPath returns the canonical slash-separated path used by fs.FS.
func normalizeAssetPath(assetPath string) string {
	return pathpkg.Clean(assetPath)
}

// normalizeSavePath returns the canonical path used by the host filesystem.
func normalizeSavePath(assetPath string) string {
	return filepath.Clean(assetPath)
}

func assetIDForSavePath(assetPath string) ID {
	return ID(pathpkg.Clean(filepath.ToSlash(assetPath)))
}

func resolveLoadOptions(path string, options ...LoadOption) *LoadOptions {
	path = normalizeAssetPath(path)
	opts := &LoadOptions{CachePolicy: CachePolicyDefault}
	for _, option := range options {
		option.applyLoad(opts)
	}
	if opts.ID == "" {
		opts.ID = ID(pathpkg.Clean(path))
	}
	if opts.Format == "" {
		opts.Format = resolveFormat(path, "")
	}
	return opts
}
