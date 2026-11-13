# Rebind an action

Let players change a control by editing a copy of the active bindings and applying the replacement. This example assumes the game was created with bindings and that `actions` is its non-nil `ctx.Actions`.

```go
func bindFireToKey(actions *input.ActionMap, key input.Key) error {
	if key <= input.KeyNone || key >= input.NumKeys {
		return fmt.Errorf("cannot bind fire to key %v", key)
	}
	bindings := actions.Bindings()
	bindings["fire"] = []input.Input{
		input.KeyInput{Key: key}.Binding(),
	}
	if err := actions.SetBindings(bindings); err != nil {
		return fmt.Errorf("rebind fire: %w", err)
	}
	return nil
}
```

The function uses `fmt` and the `input` package. Call it from your settings handler after the player confirms a key. It rejects `KeyNone` and out-of-range keys, which cannot activate the action. This replaces **all** bindings for `"fire"` with that key; preserve or add controller bindings in the list if you want to keep them. Other actions remain in the copied map.

`Bindings` returns a copy. Editing it alone doesn't affect the game, and `SetBindings` validates the binding structure. An invalid replacement returns an error and leaves the current controls unchanged, but `SetBindings` does not reject every nonfunctional key value; check captured input as above. A successful change takes effect on the next input update, not immediately in the current snapshot.

Keep controls for confirming or cancelling rebinding available independently of the action being edited. The [rebinding example](../../../../examples/rebinding) reserves raw keys for its editor and demonstrates confirm, cancel, and persistence.

## Keep the choice between runs

Changing the live action map doesn't save it. For durable settings, maintain an input preference snapshot, validate and apply it, then save it using your project's storage. Restore the snapshot before creating the game or apply it to the action map later. The existing [input preferences workflow](../../../unreleased/guides/input.md#store-desired-input-preferences) and [settings guide](../../../unreleased/guides/settings.md#save-and-restore-engine-preferences) cover that process and its error handling.
