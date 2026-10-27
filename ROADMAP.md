# Roadmap

Castrum aims to be a dependable, Go-native 2D game framework on Ebitengine: a
clear way to compose game state and behavior, a useful gameplay toolkit, and
a practical path from a new project to a playable, distributable game.

## How planning works

GitHub Issues are the working backlog for proposals, scope, acceptance
criteria, and implementation. Version labels group candidates; they are not
dates or release promises. Work becomes a commitment only after it is
deliberately selected and scoped. The guides and API reference, not this
roadmap, describe the behavior supported by the current release.

- [v0.3.0 candidates](https://github.com/Leonard-Atorough/castrum/issues?q=is%3Aissue%20is%3Aopen%20label%3Av0.3.0)
- [v1.0.0 candidates](https://github.com/Leonard-Atorough/castrum/issues?q=is%3Aissue%20is%3Aopen%20label%3Av1.0.0)
- [Deferred ideas](https://github.com/Leonard-Atorough/castrum/issues?q=is%3Aissue%20is%3Aopen%20label%3Adeferred)
- [All open issues](https://github.com/Leonard-Atorough/castrum/issues)

The remaining v0.3.0 proposals from the former version plan are now tracked
as [a scene/state lifecycle](https://github.com/Leonard-Atorough/castrum/issues/118),
[a project starter or template](https://github.com/Leonard-Atorough/castrum/issues/119),
and [verified platform build and packaging guidance](https://github.com/Leonard-Atorough/castrum/issues/120).

## Planning principles

1. Prioritize a coherent route from a new Go project to a complete 2D game
   over isolated feature count.
2. Keep Go and Ebitengine strengths visible through small, typed, composable
   APIs.
3. Define ownership, timing, lifecycle, and error behavior before adding
   convenience layers.
4. Give user-facing features documentation, tests, and runnable examples.
5. Require user evidence and a design before taking on large systems such as
   editors, full physics, or networking.
