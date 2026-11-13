# Victory or defeat

The score tells us how many enemies the player has destroyed. Let's make it matter: destroying all three wins the round, while an enemy touching the hull loses it.

We'll introduce two ideas. A **state** describes which mode the game is in; a **scene** describes entities to create together. Here the modes are playing, victory, and defeat, and the result scene displays a message over the stopped game.

## Create the game modes

Create `ending.go` with these imports and declarations:

```go
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/state"
)

type gameMode int

const (
	playing gameMode = iota
	victory
	defeat
)
```

In `main.go`, add `github.com/Leonard-Atorough/castrum/state` to the imports. Replace the `castrum.New` call with:

```go
	modes := state.New[gameMode]().WithInitial(playing)
	g, err := castrum.New(
		castrum.WithTitle("Tank game"),
		castrum.WithBindings(bindings),
		castrum.WithState(modes),
		castrum.WithScenes(),
	)
```

Keep the existing error check. `WithState` lets the game process this machine, and `WithScenes` provides the scene director. The machine starts in `playing`.

## Choose what stops

Changing state alone doesn't stop systems. We'll put the gameplay systems in a group so we can stop them together when the round ends. The score display stays outside the group.

In `main.go`, add `castrum.InGroup("gameplay")` as the last argument to each of these six `g.AddSystem` calls: `tank.move`, `tank.aim`, `camera.follow`, `enemy.move`, `shot.fire`, and `shot.move`. For example, the movement registration becomes:

```go
	if err := g.AddSystem(core.PhaseFixed, "tank.move", movePlayer(player), castrum.InGroup("gameplay")); err != nil {
		return fmt.Errorf("register movement: %w", err)
	}
```

Keep the other registrations' names, systems, and error checks unchanged. Don't group `hud.score`: it was registered in `hud.go`, and we want the last score to remain visible.

## Request an outcome

Append this function to `ending.go`:

```go
func addEndings(g *castrum.Game, modes *state.Machine[gameMode], player *tank, progress *round) error {
	director := g.World().MustResource[*scene.Director]()
	modes.OnEnter(victory, resultScene(g, director, "victory", "YOU WIN"))
	modes.OnEnter(defeat, resultScene(g, director, "defeat", "GAME OVER"))

	if err := g.AddSystem(core.PhaseFixed, "round.result", core.SystemFunc(func(ctx *core.Context) error {
		if !modes.Is(playing) {
			return nil
		}
		if _, pending := modes.Pending(); pending {
			return nil
		}

		var next gameMode
		if progress.score >= targetScore {
			next = victory
		} else {
			contacts, ok := player.hull.Component[collision.Contacts](ctx.World)
			if !ok || len(contacts.Current) == 0 {
				return nil
			}
			next = defeat
		}
		if err := modes.Request(next); err != nil {
			return fmt.Errorf("request result: %w", err)
		}
		if err := g.SetGroupEnabled("gameplay", false); err != nil {
			return fmt.Errorf("stop gameplay: %w", err)
		}
		return nil
	})); err != nil {
		return fmt.Errorf("register round result: %w", err)
	}
	return nil
}
```

The system checks the score first, then checks the hull's contacts. If both conditions occur on the same tick, victory takes precedence. Once it requests an outcome, it stops the gameplay group immediately; that also prevents more shots or movement while the transition is pending.

A request is not an immediate state change. The state machine processes it on a later frame, then runs the matching enter hook. The pending check prevents the result system from requesting a conflicting outcome in the meantime.

## Show the result

Append this function to `ending.go`:

```go
func resultScene(g *castrum.Game, director *scene.Director, name, text string) core.SystemFunc {
	return func(ctx *core.Context) error {
		camera, ok := g.MainCamera().Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("show result: camera has no transform")
		}
		result := &scene.Blueprint{
			Name: name,
			EntityList: []scene.EntitySpec{{
				Name: "message",
				Components: []any{
					core.Transform{Position: camera.Position},
					render.Sprite{
						Drawable: render.TextSource{Font: builtin.Font, Text: text, Size: 48},
						Layer:    10,
					},
				},
			}},
		}
		if _, err := director.Load(result); err != nil {
			return fmt.Errorf("load %s scene: %w", name, err)
		}
		return nil
	}
}
```

The blueprint is a recipe for a scene containing one text entity. The enter hook loads the appropriate recipe when the state changes. We place its message at the stopped camera's position and draw it above the game.

This scene is loaded *alongside* the existing world; it doesn't take ownership of the tank or enemies we created earlier. Stopping the gameplay group is what keeps those entities still. The existing [state guide](../../../unreleased/guides/state.md) and [scene guide](../../../unreleased/guides/scenes.md) explain transitions and ownership in more depth.

## Play the finished round

In `main.go`, call `addEndings` after `addHUD` and before creating the runner:

```go
	if err := addEndings(g, modes, player, progress); err != nil {
		return err
	}
```

Run `go run .`. Shoot all three enemies to see **YOU WIN**. Run it again and let an enemy touch the hull to see **GAME OVER**. The tank and enemies stop when the round ends; close and rerun the game to start another round.

You now have a complete game loop, from input and movement to a visible outcome. If you'd like to add some feedback, continue with [adding sound](sound.md); otherwise, return to the [manual](../../README.md) to explore more ways to build on it.
