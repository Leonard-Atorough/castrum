# Core principles

The reference tier exists for one reader: the developer who wants to know why castrum is shaped this way before trusting it with a game, or before extending the engine itself. Everything here is optional knowledge - the [getting-started tier](../getting-started/concepts.md) and the [guides](../guides/conventions.md) cover usage.

Three pages. This one records the rules that generated the API. [Engine design](engine-design.md) shows how the packages fit. [The runner separation](runner-separation.md) covers the one boundary that is a promise.

## The game declares, the engine decides

The game writes state; the engine makes it true. An entity with a `Sprite` renders. An entity with an audio `Source` plays. To steer either, the game writes fields - pause it, retarget it, recolor it - and the engine reconciles the world to match on its own schedule.

This split keeps scheduling where it belongs. Which tick, which frame, drawn interpolated by how much - these are engine decisions, made with the whole frame in view. A game issuing draw calls and play calls would be making them blind.

So the declare-and-reconcile shape is architecture. A sprite appears without drawing code and music starts without a play call because the renderer and the audio system are watching the world, and the convenience of that is a side effect.

## The zero value is a contract

Every component sits in one of two states: valid in its zero value, or rejected loudly at the moment it enters the world. A zero `Transform.Scale` reads as unscaled. An `audio.Source` missing its volume fails at spawn, with a message that states the range.

The two-state rule is what makes struct-literal spawning safe. Write the fields you care about, leave the rest at zero, and trust the result. A field with a sensible zero says so in its doc comment; a field without one gets a constructor, like `audio.NewSource`, or a spawn error that tells you the range.

The reasoning is simple: silent misbehavior is the most expensive failure an engine can hand a game. A component that produces wrong pixels fails far from its cause, in a shipped game. A component that fails at spawn fails at the author's desk, with the error naming the rule that was broken. The [conventions guide](../guides/conventions.md) carries this as day-to-day rules; this page is the why.

## Errors are values; state is single-threaded

Engine APIs return errors. The one panic in a well-formed castrum program lives in `main`, because a game that failed to construct has nothing left to recover into. Everywhere else, errors travel up as values, pick up context on the way, and surface from `g.Run`.

Everything runs on the loop's thread - systems, draws, setup. One thread is what lets the engine protect its own state cheaply, and it keeps game code reading like a plain program instead of a locking exercise.

The two rules share a motivation: castrum games should fail debuggably. A returned error naming the failing system beats a panic three frames later. One thread beats a race report.

## Earned API surface

Every public API earns its place. A feature arrives when a second real consumer exists. An abstraction arrives when a second implementation does. That is why castrum ships one runner, waits for a real alternative part before building a plugin mechanism, and grows features on demand rather than on a roadmap's guess.

For the reader, the consequence is a small API on purpose. When something you need seems missing, the gap is usually deliberate, and the design notes are the paper trail showing how each surface earned its place. The path to closing it is an issue describing the use case - the second consumer that earns the feature.

<!-- MAINTENANCE NOTE: this page is the public distillation of the
     internal design ledgers (.internal/design and .internal/decisions).
     Keep every claim rewriteable from a ledger entry. The
     single-threaded contract has its own documentation effort
     tracked in issue #30 - align wording when it lands. -->
