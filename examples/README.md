# Examples

Choose an example by what you want to learn. Start with the starter and tutorial, use focused examples for individual features, then explore the integrated game.

Source on `main` follows the engine's development API. Release tags preserve source matching that release; standalone example modules are pinned to it when the release is tagged.

## Start here

| Project | What it teaches | What to try or observe | Before reading | Guide |
| --- | --- | --- | --- | --- |
| [Starter](../templates/starter) | Minimal project structure, input bindings, fixed-tick movement, and the runner | Move the blue square with WASD; use this as your project scaffold | Basic Go | [Starter instructions](../templates/starter/README.md) |
| [Your first game](your-first-game) | A complete tutorial game combining movement, aiming, combat, audio, timers, and collision | Drive and aim the tank, survive enemy waves, and watch the score and reload display | Starter structure and basic entities/systems | [Your first game](../docs/unreleased/getting-started/your-first-game.md) |

The tutorial is a complete game, not the shortest introduction to each API it uses. Use the focused examples below when you only need one feature.

## Focused examples

Each example has one primary teaching purpose. Supporting controls, shapes, text, or gameplay make that feature observable; they are not a shared example framework.

### Rendering and presentation

| Project | What it teaches | What to try or observe | Before reading | Guide |
| --- | --- | --- | --- | --- |
| [Wander](wander) | Fixed-tick movement rendered with transform interpolation, using an embedded sheet region | Watch the sprite move smoothly between randomly chosen targets | Transforms and fixed ticks | [Rendering](../docs/unreleased/guides/rendering.md) |
| [Animate](animate) | File-backed sheets, setup validation, explicit grid authoring, and animation completion | SPACE pauses, R restarts, 1 plays once, and 2 loops; `-author` saves metadata without opening a window | Sprites and asset loading | [Spritesheets](../docs/unreleased/guides/spritesheets.md#run-the-complete-example) |
| [Primitive](primitive) | Rectangle, ellipse, and line drawables with different styles | Compare the rows and columns for outlines, rotation, scale, and alpha | Sprites and transforms | [Rendering](../docs/unreleased/guides/rendering.md) |
| [Text](text) | Text as a drawable, including changing content, tint, and scale | Compare the static labels and changing text | Sprites and asset loading | [Rendering](../docs/unreleased/guides/rendering.md) |
| [Camera](camera) | Camera movement, rotation, zoom, and bounds | Use arrows to move, A/D to rotate, and W/S to zoom; observe the bounded view | Transforms and input bindings | [Camera](../docs/unreleased/guides/camera.md) |
| [Tween](tween) | Caller-advanced tweening on fixed versus frame clocks | Pause, change simulation speed, or restart; compare motion with tint progression | Fixed and frame phases | [Tweening](../docs/unreleased/guides/tweening.md) |

### Input and gameplay

| Project | What it teaches | What to try or observe | Before reading | Guide |
| --- | --- | --- | --- | --- |
| [Input](input) | Named keyboard/gamepad bindings alongside raw cursor and wheel input | Drive the tank, aim with the mouse, and zoom with Q/E or the wheel | Entities, transforms, and cameras | [Input](../docs/unreleased/guides/input.md) |
| [Rebinding](rebinding) | Restore controls, edit an owned draft, confirm/cancel, apply, and save | F2 edits ping; choose a key, ENTER confirms, ESC cancels; restart to restore | Input actions and typed preferences | [Persistent rebinding](../docs/unreleased/guides/input.md#run-the-persistent-rebinding-example) |
| [Collision](collision) | Circle/box detection, layer masks, triggers, and contact-entry reactions | Move through hazards and pickups; observe health and score changes | Queries and fixed systems | [Collision](../docs/unreleased/guides/collision.md) |
| [Raycast](raycast) | Segment queries against synchronized collider state, with layer filtering | Aim with the cursor; inspect the displayed hit or miss | Colliders and camera coordinates | [Collision queries](../docs/unreleased/guides/collision.md) |
| [Scenes](scenes) | Blueprint switching, additive scenes, lifecycle hooks, and entity ownership | SPACE changes location; S toggles the scanner in the field | Entities, queries, and resources | [Scenes](../docs/unreleased/guides/scenes.md) |
| [State](state) | Queued mode changes, enter/exit hooks, restart, and lifecycle signals | SPACE cycles modes; R restarts; compare Current/Pending and the hook/signal log | Frame systems and input actions | [State](../docs/unreleased/guides/state.md#run-the-focused-example) |
| [Time control](time-control) | Simulation pause and scale while frame-phase controls keep running | Pause or change speed; compare the orbiting bodies with the live status display | Fixed and frame phases | [Time in Castrum](../docs/unreleased/guides/time-control.md) |
| [Timer](timer) | One-shot/repeating timers, completion state, and timer groups | R restarts the one-shot, SPACE starts the held timer, and P pauses one group | Components and fixed ticks | [Timers](../docs/unreleased/guides/timer.md) |
| [Signals](signals) | Consuming engine signals, emitting game signals, and independent readers across phases | Watch the repeating charge/countdown/launch sequence and its debug messages | Timers and phase ordering | [Signals](../docs/unreleased/guides/signals.md) |
| [Scheduling](scheduling) | Explicit `Before`/`After` insertion within one user phase | Run the headless command; observe sample/apply/audit/report despite registration order | Basic systems; no window required | [Relative ordering](../docs/unreleased/reference/the-scheduler.md#run-the-relative-ordering-example) |
| [Settings](settings) | Restore, validate, apply, and save typed preferences | Compare printed values across runs; writes to the OS user config directory by default, or a supplied `-directory` | Basic Go; no running game required | [Settings](../docs/unreleased/guides/settings.md#run-the-persistence-example) |

Collision demonstrates detection and game-owned reactions, not physical separation or collision resolution. Settings is a command-line persistence example, not a preferences screen.

### Audio

These examples embed shared recordings. Audio output is required to hear the behavior; attribution is in [CREDITS.md](../CREDITS.md).

| Project | What it teaches | What to try or observe | Before reading | Guide |
| --- | --- | --- | --- | --- |
| [Playback](audio/playback) | Eager one-shots versus persistent streamed music | SPACE plays an effect; P pauses music and R restarts it | Audio sources and clips | [Audio](../docs/unreleased/guides/audio.md) |
| [Mixing](audio/mixing) | Master/group volume and global pause policies | Adjust volumes with arrows; ENTER pauses the mixer and SPACE plays an effect that continues through pause | Playback example | [Audio](../docs/unreleased/guides/audio.md) |
| [Balance](audio/balance) | Live per-source stereo balance and preservation across pause/restart | Adjust balance with LEFT/RIGHT or 1/2/3; SPACE pauses and R restarts | Playback example; stereo output | [Audio](../docs/unreleased/guides/audio.md) |
| [Spatial](audio/spatial) | A moving listener, a stationary spatial source, and a supplied attenuation curve | Move the listener marker with WASD/arrows and hear the source change with position | Transforms and audio sources/listeners | [Audio](../docs/unreleased/guides/audio.md) |

## Integrated games

| Project | What it teaches | What to try or observe | Before reading | Guides |
| --- | --- | --- | --- | --- |
| [Signal salvage](signal-salvage) | Scenes, signals, collision, timers, and system groups coordinating a complete round | SPACE starts; collect five cores before timeout, avoid the pulse, and use ENTER to return to briefing | Focused scenes, signals, timers, and collision examples | [Scenes](../docs/unreleased/guides/scenes.md), [Signals](../docs/unreleased/guides/signals.md) |

Use this game to see features cooperate, not as the first explanation of each API. The first-game tutorial above remains the guided route through building a game.

## Run development examples

From the repository root, create a local Go workspace:

```sh
go work init .
go work use -r ./examples
go work use ./templates/starter
```

If the workspace already exists, skip `init` and run the `use` commands to include new modules. The ignored workspace resolves Castrum to the checkout without changing each module's dependency requirement.

Run an example from its directory:

```sh
cd examples/audio/playback
go run .
```

Run its tests from the same directory with `go test ./...`. Most runnable directories are standalone modules using the `github.com/Leonard-Atorough/castrum-examples` namespace. Spatial currently belongs to the root Castrum module and runs with the same `go run .` command from its directory. The `audio/` parent contains shared assets, not a runnable command.

Root `go test ./...` does not traverse standalone modules. PR and release CI build, vet, and test those modules separately against the checkout; the root module includes the shared audio assets and spatial command. See [Contributing](../CONTRIBUTING.md#develop-examples-against-the-checkout) for validation commands and the release lifecycle.

## Run released examples

Check out the release tag you want to learn from and disable workspace resolution with `GOWORK=off` in your environment. Then run the example from its directory. The source and dependencies at that tag describe the same release.

Do not assume development source will run against the version currently recorded in its module file: new APIs may not exist in that published version yet. Use the workspace for development and a release tag for a released example.
