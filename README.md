# castrum

Castrum is a game engine designed to provide a flexible and efficient framework for developing games. It focuses on a clear separation between configuration, simulation, and rendering, allowing developers to fine-tune their game's behavior and performance.

You declare what your game contains - an entity with a position and a picture, an entity with a looping music file - and the engine renders it, advances it, and plays it every frame. There is no draw loop to write and no play call to make.

Castrum uses [Ebitengine](https://ebitengine.org) as its underlying rendering and window management library, leveraging its capabilities to handle graphics, input, and other low-level tasks efficiently. To facilitate this, castrum wraps Ebitengine functionality within its own abstractions, providing a more structured and game-focused interface for developers.

See the [feature overview](docs/getting-started/features.md) for what Castrum supports today, and the [roadmap](roadmap/README.md) for proposed feature sets and longer-term directions. Release targets are planning proposals rather than dated commitments; the guides and API reference describe what the current version actually supports. See the issues tab in the repository for live issues and features being planned/refined

## Documentation

- [Documentation](docs/getting-started/intro.md) - choose the introduction, concepts, first-game tutorial, or the full guides from there.
- [Examples](examples) - run complete programs covering rendering, shapes, input, animation, and audio.
- [API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) - browse exported types and methods.

## Status

Castrum is pre-1.0 and should be treated as unstable. Migration notes will be provided between versions to assist in seamless upgrades and the goal will be to minimise breaking changes as much as possible.

## The name

A _castrum_ (Latin) was a Roman fortified camp: a garrison built to a standard layout from local materials, quickly, wherever the legions needed to hold ground. The name fits an engine that aims to be the standard-built base your game stands on - small, planned, and hard to knock over.

## How this is made

Castrum is developed using an AI-assisted workflow: a person owns the design, the direction, and every line that lands; AI helps implement, test, and document it; the person reviews the whole diff, and nothing merges without passing gates nobody can talk past - vet, tests, coverage, and benchmarks. The contribution policy in [CONTRIBUTING.md](CONTRIBUTING.md) draws the same line for others: AI assistance is welcome, delegation is not.

We declare this because it is true, and because we would rather read a one-line "built with AI assistance, reviewed by me" than guess. If castrum is part of something you make, transparency in your own style is encouraged - say what the machine did and what you did. The tooling has changed how software gets written and honesty and integrity is how we as a community build trust in each other, and in the projects we create.

## Contributing

See the [CONTRIBUTING.md](CONTRIBUTING.md) file for guidelines on how to contribute to Castrum, including reporting issues, suggesting features, and submitting pull requests.

We welcome contributions from the community, whether it's fixing bugs, adding new features, improving documentation, or providing examples. Please follow the contribution guidelines to ensure a smooth and productive collaboration.

If you are new to Castrum or open-source contributions in general, consider starting with issues labeled as "good first issue" or "help wanted" to get familiar with the codebase and contribution process.

Thank you for considering contributing to Castrum! Your efforts help make the engine better for everyone in the community.
