# Settings

The `settings` package holds game-defined player preferences in typed groups. A group is the **desired** value; your game decides how to apply it and whether to save it.

The runnable [settings example](../../../examples/unreleased/settings/main.go) defines preferences, validates and copies them, then reads them before and after a change. Run it from `examples/unreleased/settings` with `go run .`.

## Define a group

Define your own preference type and, when needed, methods for validation and copying. This example uses a value receiver, so `Preferences.Validate` and `Preferences.Clone` can be adapted directly to a policy:

```go
type Preferences struct {
	TextSize int
	Hints    []string
}

func (p Preferences) Validate() error {
	if p.TextSize <= 0 {
		return fmt.Errorf("text size must be positive")
	}
	return nil
}

func (p Preferences) Clone() Preferences {
	p.Hints = slices.Clone(p.Hints)
	return p
}

prefs, err := settings.New(Preferences{TextSize: 16}, settings.Policy[Preferences]{
	Validator: settings.ValidatorFunc[Preferences](Preferences.Validate),
	Cloner:    settings.ClonerFunc[Preferences](Preferences.Clone),
})
if err != nil {
	return err
}
```

`New` copies and validates the initial value and returns an error if it is invalid. The policy fields are optional: `settings.Policy[Preferences]{}` accepts all values and uses ordinary Go copies. A typed nil validator or cloner is rejected; use a nil field when a policy is not needed.

Provide a cloner when a value contains mutable data, such as slices, maps, or pointers. It must copy all mutable data reachable from the value without changing its argument. Without one, ordinary copies can share mutable data with the caller. Validators must not mutate their argument. A policy method must not replace the group; `Set` and `Update` on the same group return an error while the policy runs. Other callback side effects are not rolled back.

If systems need the group, register it as a [world resource](ecs.md#resources):

```go
if err := g.World().Provide(func(*core.World) (*settings.Group[Preferences], error) {
	return prefs, nil
}); err != nil {
	return err
}
```

Resolve it with `g.World().Resource[*settings.Group[Preferences]]()` and handle the error. Groups of different preference types are different resource types; registering a second resource of the same type is an error. The group is not registered automatically.

## Read and change preferences

`Value` returns a copy according to the policy. To replace the whole value, call `Set`. To change part of it, use `Update`:

```go
if err := prefs.Update(func(p Preferences) Preferences {
	p.TextSize = 20
	p.Hints = append(p.Hints, "jump")
	return p
}); err != nil {
	return err
}
```

`Update` passes a copy to the callback, then validates and stores its result. With a suitable cloner for mutable data, an invalid replacement leaves the group unchanged. The callback must not replace the group or mutate it through `Set` or `Update`; those nested mutations return an error. Side effects of the callback are not rolled back.

Copies of a `Group` share its state. Copying a value returned by `Value`, or changing it in an `Update` callback, is safe only when the policy's cloner copies any mutable data the value contains.

## Apply preferences

Changing the group does not apply the value to the game. For example, updating the desired text size does not resize any text. A system can read the preferences and apply them when it runs:

```go
if err := g.AddSystem(core.PhaseFrame, "apply text size", core.SystemFunc(func(*core.Context) error {
	value, err := prefs.Value()
	if err != nil {
		return err
	}
	return applyTextSize(value.TextSize)
})); err != nil {
	return err
}
```

`applyTextSize` is your game's function; it is not part of the settings package. This system calls it every frame, including frames where the preferences are unchanged. If applying a setting is expensive or has side effects, the system can remember the last successfully applied value and skip identical values. Return application errors so the game or runner can handle them.

## Configure the initial window

The default runner accepts display preferences when it is constructed. `ebitrun.NewDisplayPreferences` creates a group with the runner's default window size, resizing, and vertical sync settings. With a game `g` already created, change the group and pass its value to the runner:

```go
display, err := ebitrun.NewDisplayPreferences()
if err != nil {
	return err
}
if err := display.Set(ebitrun.DisplayPreferences{
	Window: ebitrun.Size{Width: 1600, Height: 900},
	Resizable: true,
	VSync: true,
}); err != nil {
	return err
}
window, err := display.Value()
if err != nil {
	return err
}
runner, err := ebitrun.New(g, ebitrun.WithDisplayPreferences(window))
if err != nil {
	return err
}
```

The window dimensions must be positive. These are startup settings: changing the group later does not change an existing runner. Options passed after `WithDisplayPreferences` override its values.

## Ownership and persistence

Groups are not safe for concurrent use. Read and copy the needed value on the game-loop goroutine before handing it to a worker; return results to the game loop before changing the group.

The package does not apply preferences, retain a separate applied or saved value, serialize data, or write files. Keep any last-applied or last-saved value yourself if you need to display or restore it. A persistence implementation must arrange safe ordering of writes to the same destination.
