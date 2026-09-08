---
name: ui-layout-system
description: >
  Foundational architecture for castrum's UI manager: anchor-based box model,
  virtual resolution scaling, retained widget tree (NOT ECS-driven UI). Consult
  when building, extending, or refactoring any UI/widget code.
---

# UI Manager Architecture

## Core Decisions (settled — do not revisit without explicit discussion)

1. **Layout model: Box model + normalized anchors.** Every element is a rect
   derived from anchor point(s) on the parent plus pixel offsets. Stretching
   is emergent (anchorMin==anchorMax for point anchor; spread anchors stretch),
   not special-cased. No flexbox negotiation, no Cassowary constraint solver.
2. **Placement lives in the parent, not the child.** Layout strategies
   (VerticalStack, Grid, manual) are strategies attached to _containers_ that
   rewrite child anchor/offset inputs. Widgets never position themselves.
3. **Behavior via `Widget` interface** (retained tree), not component/system
   registration. Leaf widgets do NOT implement layout methods. See `Widget`
   contract below.
4. **UI is NOT in the ECS world.** Deliberate. UI is hundreds of heterogeneous
   nodes with deep hierarchy and polymorphic behavior — wrong fit for
   archetype storage. If ECS interop is needed, add a bridge component
   (`WidgetRef` entity → UI element id), not a migration.

## Coordinate Spaces

- All layout math runs in a **virtual design resolution** (define once,
  e.g. 1920×1080). Anchors, offsets, min-sizes, font sizes live there.
- One uniform scale factor maps virtual → screen. NEVER scale X and Y
  independently.
- Default mode: `ScaleExpand` (uniform scale, root rect grows beyond design
  size in one axis — anchors absorb the difference). `ScaleFit` available
  for fixed-framing modes.
- Hit-testing: unwind screen point → virtual point before delivering input
  to widgets: `vx = (sx - offsetX) / scale`.

## Widget Interface Contract

```go
type Widget interface {
    Bounds() Rect                  // float rect, derived — not mutable truth
    Children() []Widget
    Draw(dst *ebiten.Image, parentTransform Transform)
    // Input handlers receive LOCAL points; return true to consume (stop bubbling)
    OnMouseDown(localPt image.Point, button int) bool
}
```

### Contract rules:

Widgets store layout inputs (anchor, offsets, minSize), never resolved bounds as mutable state. Bounds() is a pure derivation from the parent chain.
Draw(dst, parentTransform) receives the composed parent transform; widgets compose their own local transform on top (right-to-left: Translate(offset) → Scale(scale)).
Input handlers return true = consumed (stops bubbling), false = propagate.
Layout math uses float rects; convert to image.Rectangle only at the draw / hit-test boundary. Never resolve positions with ints mid-layout.

## Layout Strategies

- **Attached to containers via a Layouter interface, NOT per-widget:**

```go
type Layouter interface {
Arrange(self Widget, content Rect) // content = parent rect minus padding
// Measure(content) Vec2 — add ONLY when a measured-content widget
// (labels, tooltips) actually exists. Not before.
}
```

- **Strategies emit anchors + offsets for children, never absolute rects. A parent resize therefore reflows everything for free on the next pass.**
- **Known strategies: Manual (nil / raw anchors), VerticalStack, HorizontalStack, Grid. Add others when a feature demands them.**

## Layout Resolution Timing

- Dirty-flag driven, NOT per-frame. Mutating layout inputs marks dirty.
- Dirty propagates UP (child change → ancestors' content bounds may change), resolution resolves DOWN (parent → children) in one pass before render.
  New instances start dirty.

## Deferred Work (note, don't build)

- **Text sharpness:** pass effective device scale into text renderer so glyphs rasterize at device resolution (avoids soft fractional-scale text).
- **Pixel snapping:** SnapToPixel() helper for hairline borders if/when fractional scaling causes artifacts.
- **Measure phase:** only when real text-measuring widgets exist.

## Anti-Patterns (do not do these)

- **Independent X/Y screen-ratio scaling (distortion).**
- **Widgets storing absolute bounds as mutable truth (dirty-flag drift).**
- **Layout responsibility on leaf widgets.**
- **Moving UI into the ECS world for architectural purity without a concrete blocking feature.**
- **Per-frame unconditional relayout.**
