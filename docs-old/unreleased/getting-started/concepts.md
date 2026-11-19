# Key concepts

:warning: **Outdated draft.** This page predates the sprite component
redesign (tagged-union `Source`/`Kind` data replacing the `Drawable`
interface) and its examples no longer compile against the engine when
this note was written. Its original scope was the mental model: game, world, entities, components, systems, queries, resources, and the moving-square walkthrough. The previous
revision is in git history; everything below was replaced by this note
so no reader mistakes stale API for current API.

## API mapping

The sprite component was redesigned as serializable data. Future docs use:

| Before (this document) | After |
| --- | --- |
| `sprite.Sprite{Drawable: sprite.TextureSource{Texture: id}}` | `sprite.Sprite{Source: sprite.Source{Kind: sprite.KindTexture, Asset: id}}` |
| `sprite.Sprite{Drawable: sprite.SheetSource{Sheet: ref, Region: r}}` | `sprite.Sprite{Source: sprite.Source{Kind: sprite.KindSheet, Asset: sheetFile, Region: r}}` |
| `sprite.Sprite{Drawable: sprite.TextSource{Font: f, Text: t, Size: s}}` | `sprite.Sprite{Source: sprite.Source{Kind: sprite.KindText, Asset: f, Text: t, TextSize: s}}` |
| `sprite.Sprite{Drawable: sprite.RectShape{Size: v}}` | `sprite.Sprite{Source: sprite.Source{Kind: sprite.KindRect, Rect: v}}` |
| `sprite.Sprite{Drawable: sprite.CircleShape{Radii: v}}` | `sprite.Sprite{Source: sprite.Source{Kind: sprite.KindCircle, Radii: v}}` |
| `sprite.Sprite{Drawable: sprite.LineShape{From: a, To: b}}` | `sprite.Sprite{Source: sprite.Source{Kind: sprite.KindLine, From: a, To: b}}` |
| `Style.Color color.Color` (nil-tinted default views) | `Style.Color *color.RGBA` (nil leaves engine default tints) |
| `animated.CurrentFrame()` returning `sprite.SheetSource` | `animated.CurrentFrame()` returning the retained `asset.SheetRef` plus the region name |

## Notes for whoever rewrites this page

- Preserve the original scope and teaching order; only the sprite API
  moved. Everything else (world, systems, queries, resources,
  cameras, timers, collisions, tweens, sheets) still matches the
  engine.
- The redesigned component is plain data with JSON tags; `Kind`
  identities (`none`, `texture`, `sheet`, `text`, `rect`, `circle`,
  `line`) are wire contracts and examples should print them exactly.
- Validation is unchanged: components still implement `Validate` and
  the world still checks them at every write boundary. The doc can
  teach `Source.Validate` rules where it taught `Sprite.Validate`
  drawable cases.
- `examples/` and `templates/starter` are migrated and compile; mine
  them for current snippets instead of restoring this text.
