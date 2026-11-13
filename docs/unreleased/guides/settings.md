# Settings

The `settings` package holds game-defined player preferences in typed groups. A group is the **desired** value; your game decides how to apply it and whether to save it.

The runnable [settings example](../../../examples/settings/main.go) restores, validates, applies, and saves preferences through an application-owned directory. Run it from `examples/settings` with the repository's development workspace and `go run .`. It writes `preferences.json` beneath the OS user configuration directory; `-directory` selects another location. Application in this command-line example means printing the values, not updating a live game.

## Define a group

Define your own preference type and, when needed, methods for validation and copying. This example uses a value receiver, so `Preferences.Validate` and `Preferences.Clone` can be adapted directly to a policy:

```go
type Preferences struct {
	TextSize int      `json:"text_size"`
	Hints    []string `json:"hints"`
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

## Save and restore engine preferences

`audio.Preferences` stores `master`, `music`, and `sfx` levels. `input.Preferences` stores JSON-native `bindings` and the map-wide `deadzone`. Persist these snapshots, not their `settings.Group` wrappers. They can also be fields of an application-owned document registered with `SetJSON`. There is no separate settings registry, automatic file I/O, or implicit application.

Choose an application-data location and open an `asset.Directory`; see [Save and reload a document](assets.md#save-and-reload-a-document) for directory ownership and write guarantees. `castrum.NewAssetCodecs` includes audio and input preferences, so no per-type JSON registration is needed. With that open directory named `directory`, construct the preference loader and saver:

```go
codecs := castrum.NewAssetCodecs()
loader, err := asset.NewLoader(directory, asset.LoaderOptions{
	Codecs: codecs,
	CachePolicy: asset.CacheBypass,
})
if err != nil {
	return err
}
saver, err := asset.NewSaver(directory, asset.SaverOptions{Codecs: codecs})
if err != nil {
	return err
}
```

The loader reads mutable preference documents independently of its decoded cache. Each loader and saver snapshots registrations at construction, so configure game-owned document codecs or replacements first. Each preset call returns independent codecs. The directory remains caller-owned and must be closed after both facades finish using it.

`castrum.New` selects this preset when `WithAssetCodecs` is absent or nil; an explicit codec configuration is used as supplied. The low-level `asset.NewCodecs` stays engine-independent. Pass the configured set through `castrum.WithAssetCodecs(codecs)` if the game's content loader should use it too. Audio sample decoders remain runner-installed at the effective sample rate.

Restore defaults only when the file is missing. Decode failures and invalid values are errors, not a reason to overwrite the player's file:

```go
prefs, err := audio.NewPreferences()
if err != nil {
	return err
}
restored, err := loader.Load[audio.Preferences]("audio.json")
if err == nil {
	if err := prefs.Set(restored); err != nil {
		return err
	}
} else if !errors.Is(err, fs.ErrNotExist) {
	return err
}
```

These codecs handle representation, not domain validation. Restore snapshots through the group's `Set` method before [applying mixer levels](audio.md#audio-preferences). Missing JSON fields keep their zero values; loading does not merge them with constructor defaults. Loading changes neither the mixer nor the player's files.

For input, create a group with `input.NewPreferences(bindings)`, then pass the complete restored snapshot through `prefs.Set(restored)` so its deadzone is retained too. Apply the validated value with `castrum.WithInputPreferences(value)` at startup or `ctx.Actions.SetPreferences(value)` at runtime. Neither loading nor editing the group applies it automatically. See [Store desired input preferences](input.md#store-desired-input-preferences) for validation, copying, and live-update behavior.

Save an owned snapshot when the player confirms a change:

```go
snapshot, err := prefs.Value()
if err != nil {
	return err
}
if err := saver.Save("audio.json", snapshot); err != nil {
	return err
}
```

Saving and application are separate operations. If applying a change must succeed before it is persisted, arrange that ordering in the game. The settings example follows the same workflow for game-defined preferences.

### Run the persistence example

From `examples/settings`, use the development workspace and run:

```sh
go run . -directory ./preferences-data
```

This explicitly chooses a native save directory. Without `-directory`, the command creates `castrum-examples/settings` beneath the OS user configuration directory returned by `os.UserConfigDir`.

The command reports the destination of `preferences.json` after saving. First-run defaults are used only when the document is missing. Malformed JSON, invalid preferences, and other read failures return errors without applying or replacing the document.

Application means printing the values in this command-line example. After printing restored values, it sets text size to 20 and adds the `jump` hint if absent, applies the result, and saves a copied snapshot. A second run restores the saved values without accumulating duplicate hints. Directory close errors are returned too.

## Ownership and persistence

Groups are not safe for concurrent use. Read and copy the needed value on the game-loop goroutine before handing it to a worker; return results to the game loop before changing the group.

The settings package does not apply preferences, retain a separate applied or saved value, serialize data, or write files. Keep any last-applied or last-saved value yourself if you need to display or restore it. A persistence implementation must arrange safe ordering of writes to the same destination.
