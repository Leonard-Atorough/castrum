package animation

import "github.com/Leonard-Atorough/castrum/core"

// Started is emitted by the animation system when an [Animator] begins playing
// a state, whether from a desired-state change or a state's Next transition.
type Started struct {
	// Entity is the entity whose state started.
	Entity core.Entity
	// Tick is the fixed tick of the start.
	Tick uint64
	// State is the name of the state that started.
	State string
}

// Completed is emitted by the animation system when a state reaches the end of
// a pass. The system emits one signal per animator per tick, even if a looping
// state wraps multiple times in one tick.
type Completed struct {
	// Entity is the entity whose state completed a pass.
	Entity core.Entity
	// Tick is the fixed tick of the completion.
	Tick uint64
	// State is the name of the state that completed a pass.
	State string
	// Loops is the number of passes ended in this tick; 1 for a one-shot state.
	Loops uint64
}
