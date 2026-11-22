# Game, Runner and World

Every castrum program consists of two parts. The `Game` and the `Runner`. By default castrum uses Ebitenengine as its runner. The separation is intentionally designed to avoid a hard dependency on Ebiten and to allow for the integration of other programs or custome runners.

A basic castrum Program, as seen in the setup, looks like this:

```go
import (
	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/ebitrun"
)

func main() {
	game := castrum.New(castrum.WithTitle("Hello World"))
	ebitrun.New().Run(game)
}
```

Nice an easy. This program creates a new Castrum game with the configured title and a new `ebitrun` runner which takes the game as an entry point to start the game loop.

## Game

Game is primary struct you'll interact with when building in Castrum. It provides a `World` which holds your game's entities and components. At a glance Game gives you access to methods to:

- Access the `World` to manage entities and components.
- Add Systems to the `Schedule`; these systems are the core logic that drive the game's behavior.
- Control game speed and execution through access to the `Simulation` clock.
- Access the game `Context`, which carries per cycle information and other runtime data.

## World

World is the `ECS` (Entity-Component-System) container within Castrum. It manages all entities and their associated components, providing the foundation for the game's logic and state management. World lets us create, modify, and query entities and their components efficiently, enabling the implementation of complex game behaviors. World also gives us access to global resources and tools necessary for using various engine features effectively.

We'll go into Castrum's ECS architecture in more detail in the next section, explaining how to define components, create entities, and implement systems to drive the game's behavior.
