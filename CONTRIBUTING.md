# Contributing to Castrum

Thank you for your interest in contributing to the Castrum 2D game engine! This guide will help you understand how to contribute effectively.

## Reporting Issues

Use the appropriate issue template when reporting bugs, requesting features, or asking questions:

- **[BUG REPORT](.github/ISSUE_TEMPLATE/bug_report.yaml)**: Report crashes, unexpected behavior, or broken functionality.
  - Include reproduction steps, environment info, and error logs.
  - Label: `bug`

- **[FEATURE REQUEST](.github/ISSUE_TEMPLATE/feature_request.yaml)**: Suggest new capabilities for the engine.
  - Clarify the problem it solves and target phase (0.1.0 or post-0.1.0).
  - Label: `enhancement`

- **[DOCUMENTATION](.github/ISSUE_TEMPLATE/documentation.yaml)**: Report missing, unclear, or outdated docs.
  - Specify where and what needs improvement.
  - Label: `documentation`

- **[SECURITY VULNERABILITY](.github/ISSUE_TEMPLATE/security_vulnerability.yaml)**: Report security issues privately.
  - **Do NOT use for public disclosure.** Email maintainers directly instead.
  - Label: `security`

- **[QUESTION](.github/ISSUE_TEMPLATE/question.yaml)**: Ask how to do something or understand a concept.
  - Describe your goal and what you've already tried.
  - Label: `question`

## Version Targets

Castrum uses [semantic versioning](https://semver.org/).

### 0.1.0: Core Engine Ready (Current Target)

Target release: a fully functional 2D game engine with stable core systems.

**In Scope:**

- ECS foundation
- Fixed-timestep game loop
- Scene management
- Rendering (sprites, primitives, single camera)
- Input handling
- Animation & timers
- Basic collision
- Asset loading
- Configuration management

**Out of Scope:**

- Visual editor
- Persistence (save/load)
- Audio system
- Advanced physics
- Networking

**Exit Criteria:**

- [ ] All core systems (Phases 2–3) complete
- [ ] Playable prototype game
- [ ] > 80% test coverage on core
- [ ] API stable; public only via `pkg/castrum`
- [ ] Developer documentation complete

### 0.2.0+

Post-release features (editor, persistence, audio, advanced physics) based on community needs.

## Getting Started

1. Clone the repository:

   ```bash
   git clone https://github.com/leonard-atorough/castrum.git
   cd castrum
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Build and test:

   ```bash
   make build
   make test
   ```

4. Review the [ROADMAP.md](ROADMAP.md) to understand current phases and priorities.

## Code Guidelines

- **Language:** Go 1.27+
- **Style:** Follow [Effective Go](https://golang.org/doc/effective_go) and `gofmt`.
- **Tests:** Write tests for new features. Aim for >80% coverage on `internal/core`.
- **Comments:** Document exported functions and non-obvious logic.
- **Dependencies:** Keep external dependencies minimal. Ebiten is the only game-engine dependency.

## Commit Message Format

Castrum uses **Conventional Commits** to enable automatic versioning and changelog generation.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type

Required. One of:

- **feat:** New feature
- **fix:** Bug fix
- **perf:** Performance improvement
- **docs:** Documentation change
- **refactor:** Code refactor (no behavior change)
- **test:** Test changes
- **chore:** Build, CI/CD, or tooling changes
- **style:** Code formatting (no logic change)

### Scope

Optional. Affected area. Examples: `ecs`, `rendering`, `input`, `scene`, `camera`, `collision`, `animation`.

### Subject

Required. Concise description (imperative mood, lowercase, no period).

- ✅ "add entity hierarchy support"
- ✅ "fix scene unload memory leak"
- ❌ "Added entity hierarchy support"
- ❌ "Fixed scene unload memory leak."

### Body

Optional. Explain what and why (not how). Wrap at 72 characters. Use imperative mood.

### Footer

Optional. Reference issues: `Closes #123`, `Fixes #456`.

### Breaking Changes

If introducing breaking API changes, add footer:

```
BREAKING CHANGE: description of what changed and migration path
```

This triggers major version bump.

### Examples

```
feat(ecs): add parent-child entity relationships

Enable hierarchical entity structures for scene graphs.
Implement AddChild/RemoveChild/GetChildren operations.

Closes #42
```

```
fix(rendering): resolve layer sorting bug in sprite batch

Layer 5+ sprites were drawing in wrong order due to
missing stable sort in batch comparator.

Fixes #89
```

```
perf(query): optimize component type matching with hash map

Reduce query time from O(n*m) to O(n) by pre-indexing
component types instead of linear scan each query.
```

## Automatic Versioning

Commits merged to `main` trigger [semantic-release](https://semantic-release.gitbook.io/):

- `feat:` commits bump **minor** version (0.1.0 → 0.2.0)
- `fix:` and `perf:` commits bump **patch** version (0.1.0 → 0.1.1)
- `BREAKING CHANGE:` footer bumps **major** version (0.1.0 → 1.0.0)

Version is updated in `VERSION` file and tagged as GitHub release with auto-generated changelog.

## Development Workflow

1. Create a feature branch from `main`:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit:

   ```bash
   git commit -m "feat: clear description of change"
   ```

3. Run tests and linting:

   ```bash
   make test
   make lint
   ```

4. Push and open a pull request against `main`.

## Pull Request Process

- **Conventional Commits:** PR title must follow [Conventional Commits](#commit-message-format) format. Automated checks validate this.
- **PR Checks:** Must pass:
  - Conventional commit validation
  - Code formatting (`gofmt`)
  - Linting (`go vet`)
  - All tests with >75% coverage
- **Description:** Link issue, explain changes clearly.
- **Approval:** One maintainer approval before merge.
- **Auto-Release:** On merge to `main`, semantic-release automatically versions and publishes.

## Questions?

- Check the [README.md](README.md) and [ROADMAP.md](ROADMAP.md).
- Ask in a [QUESTION](.github/ISSUE_TEMPLATE/question.yaml) issue.
- Reach out to maintainers in discussions.

---

Thank you for contributing to Castrum! 🎮
