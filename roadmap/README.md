# Castrum roadmap

Castrum aims to be a dependable, Go-native 2D game framework on Ebitengine: a
clear way to compose game state and behavior, a useful built-in gameplay
toolkit, and a development workflow that helps games reach players. The
roadmap is a product direction, not a promise to reproduce every feature of a
large general-purpose engine.

## How to use this roadmap

Release targets are feature sets, not dates. An item belongs to a target when
its user-visible behavior and acceptance criteria are clear enough to build
and verify. The targets are proposals until work is explicitly selected.

- [v0.1.0: Complete the first public beta](v0.1.0.md) closes the known gaps in
  the initial feature set and establishes release gates.
- [v0.2.0: Control and presentation](v0.2.0.md) adds game-level time control
  and rounds out camera and audio behavior.
- [v0.3.0: Compose complete games](v0.3.0.md) proposes a small set of
  higher-level tools for building and shipping a complete 2D game.
- [Longer-term capability map](future.md) records valuable directions that
  need design, user evidence, or scope decisions before becoming release
  commitments.

When this roadmap moves into work tracking, keep these feature-set documents
as the source for intent and acceptance criteria; track implementation tasks
and ownership in the work tracker.

## Planning principles

1. **Ship a coherent 2D framework.** Prioritize the common path from a new Go
   project to a playable, distributable game over isolated feature count.
2. **Keep Go and Ebitengine strengths visible.** Prefer small, typed,
   composable APIs and make Ebitengine's rendering and platform capabilities
   accessible instead of recreating them without a clear benefit.
3. **Keep the core dependable.** Define timing, ownership, errors, lifecycle,
   and backend boundaries before layering convenience APIs on top.
4. **Make features discoverable.** User-facing features need docs and a
   runnable example; release-level changes need tests and appropriate quality
   gates.
5. **Use evidence for expensive bets.** Editors, full physics, networking,
   and similar large systems need demonstrated demand and a design before
   entering a version target.

## Baseline

The project is pre-1.0 and its public API may change. The current engine
already has a fixed-step loop and scheduler phases, an archetype ECS, typed
resources and queries, transforms and camera basics, sprites and primitives,
animation playback, action-based and raw input, synchronous typed asset
loading with atlas support, and a wired default audio provider for eager and
streaming playback. See the current guides and API for exact behavior;
version plans below call out gaps rather than treating every idea in the
previous engine as implemented or required.

The known v0.1.0 gaps are recorded in that target rather than marked as
implemented here. The older Castrum-old roadmap is historical input only; its
feature checklist and architectural choices are not commitments for this
engine.
