# Contributing to Castrum

Thanks for wanting to contribute. This document covers how changes are made, reviewed, and released, and what we expect from a contribution - including how AI fits in.

## Getting started

Castrum is a Go module with no generated code and no build steps beyond the Go toolchain:

```sh
git clone https://github.com/Leonard-Atorough/castrum.git
cd castrum
go build ./...
go vet ./...
go test ./...
```

On Linux, the Ebitengine dependencies need the usual windowing headers (`libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libgl1-mesa-dev`). CI runs `gofmt -l .` - a contribution is formatted when that command prints nothing.

## What a change needs

Every change keeps the gates green, and the gates are the floor, not the ceiling:

- `gofmt -l .` is empty, `go vet ./...` and `go test ./...` pass, and merged coverage stays at or above the threshold in the PR checks (85% at the time of writing, with per-package coverage published on every run).
- Public API has Go documentation that states the contract - what a caller can rely on and what fails how - rather than the implementation's internals. A reader who cannot act on a sentence does not need it.
- A behavior change comes with a test that fails without it. A performance claim comes with a number from the `benchmark/` module, before and after, compared with benchstat.
- Docs move with the code: a feature lands with its guide section and, where it helps, a runnable example. The getting-started tutorial's code blocks are verbatim-synced with `examples/your-first-game` - a change to one needs the other.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org); CI validates the PR title against the same types the release tooling derives versions from.

## Pull requests

Open a PR against `main`. The PR template's checklist is not ceremony - the boxes describe the review your PR will get. Keep the change as small as the fix; a PR that does one thing lands faster than one that does three. If a change is breaking, say so in the type line (`feat!`, `fix!`) - pre-1.0 allows breaking releases, and the changelog is generated from your commits, so the commit message is the changelog.

## AI policy

**AI-assisted coding is welcome here. Vibe coding is not.**

The difference is ownership:

- **AI-assisted** means a human directs the work, understands every line that lands, reviews the full diff, and answers for the result. The AI is a faster keyboard - the design decisions, the acceptance criteria, and the final judgment are a person's. This repository is itself developed this way, and its history is the model: human-owned design, AI-assisted implementation, human review of everything, and objective gates (tests, vet, coverage, benchmarks) that neither party can talk past.
- **Vibe coding** means delegating authorship and review to the model - pasting output that works without understanding why, and shipping it. Contributions like that are rejected regardless of whether the tests pass, because the tests cannot cover intent and a reviewer cannot review a diff nobody understands.

If you used AI on a contribution, say so in the PR description the way the project does in its README - what the AI did, and what you did. Transparency costs one sentence; its absence costs the review.

## Releases

Releases are cut by the Release workflow on `main`, which derives the next version from conventional commits, generates the changelog with git-cliff, tags, and publishes the tutorial asset bundle alongside. Pre-1.0 releases are marked as prereleases; the API is unstable until the version says otherwise, and every release should be readable as potentially breaking.

## Conduct

Be precise and be decent. Technical disagreement is how the engine gets better; anything directed at a person rather than their code is how contributions end.
