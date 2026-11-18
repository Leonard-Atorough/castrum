package serialization

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

type trader struct {
	Name     string  `json:"name"`
	Patience float64 `json:"patience"`
}

func (t trader) Validate() error {
	if t.Name == "" {
		return errTraderName
	}
	return nil
}

var errTraderName = &validateError{"trader: name must not be empty"}

type validateError struct{ msg string }

func (e *validateError) Error() string { return e.msg }

type station struct {
	Credits int64 `json:"credits"`
}

func newTestRegistry(t *testing.T) (*Registry, *asset.Codecs) {
	t.Helper()
	world := core.NewWorld()
	codecs := asset.NewCodecs()
	if err := world.Provide(func(*core.World) (*asset.Codecs, error) {
		return codecs, nil
	}); err != nil {
		t.Fatalf("provide codecs: %v", err)
	}
	return NewRegistry(world), codecs
}

func TestComponentRegistration(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}
	if err := reg.Component[trader]("game.Trader"); err == nil {
		t.Fatal("duplicate component key accepted")
	}
	if err := reg.Component[station](""); err == nil {
		t.Fatal("empty component key accepted")
	}
	// A second type under a second key is fine.
	if err := reg.Component[station]("game.Station"); err != nil {
		t.Fatalf("Component second type: %v", err)
	}
}

func TestResourceRegistration(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Resource[station]("game.Station"); err != nil {
		t.Fatalf("Resource: %v", err)
	}
	if err := reg.Resource[station]("game.Station"); err == nil {
		t.Fatal("duplicate resource key accepted")
	}
	if err := reg.Resource[station](""); err == nil {
		t.Fatal("empty resource key accepted")
	}
}

func TestComponentRoundTrip(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}

	value, err := reg.DecodeComponent("game.Trader", []byte(`{"name":"Vex","patience":0.8}`))
	if err != nil {
		t.Fatalf("DecodeComponent: %v", err)
	}
	got, ok := value.(trader)
	if !ok {
		t.Fatalf("decoded value type %T, want trader", value)
	}
	if got.Name != "Vex" || got.Patience != 0.8 {
		t.Fatalf("decoded = %+v, want name Vex patience 0.8", got)
	}

	var buf bytes.Buffer
	if err := reg.EncodeComponent("game.Trader", got, &buf); err != nil {
		t.Fatalf("EncodeComponent: %v", err)
	}
	if !strings.Contains(buf.String(), `"name":"Vex"`) {
		t.Fatalf("encoded = %q, want the trader fields", buf.String())
	}
}

func TestDecodeComponentErrors(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}

	// Unknown keys name the key.
	if _, err := reg.DecodeComponent("game.Unknown", []byte(`{}`)); err == nil ||
		!strings.Contains(err.Error(), "game.Unknown") {
		t.Fatalf("unknown key error = %v, want it to name game.Unknown", err)
	}

	// Invalid JSON wraps the codec error with the key.
	if _, err := reg.DecodeComponent("game.Trader", []byte(`{`)); err == nil ||
		!strings.Contains(err.Error(), "game.Trader") {
		t.Fatalf("decode error = %v, want it to name game.Trader", err)
	}

	// Validatable components are rejected at the boundary.
	if _, err := reg.DecodeComponent("game.Trader", []byte(`{"name":"","patience":1}`)); err == nil ||
		!strings.Contains(err.Error(), "name must not be empty") {
		t.Fatalf("validation error = %v, want the trader constraint", err)
	}
}

func TestEncodeComponentTypeMismatch(t *testing.T) {
	reg, _ := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}
	var buf bytes.Buffer
	if err := reg.EncodeComponent("game.Trader", station{Credits: 1}, &buf); err == nil {
		t.Fatal("mismatched value type accepted")
	}
}

func TestCodecsResolvedLazily(t *testing.T) {
	world := core.NewWorld()
	reg := NewRegistry(world)

	// Before the codecs resource exists, registration fails with the
	// resolution error.
	if err := reg.Component[trader]("game.Trader"); err == nil ||
		!strings.Contains(err.Error(), "resolve asset codecs") {
		t.Fatalf("pre-provide error = %v, want codec resolution failure", err)
	}

	codecs := asset.NewCodecs()
	if err := world.Provide(func(*core.World) (*asset.Codecs, error) {
		return codecs, nil
	}); err != nil {
		t.Fatalf("provide codecs: %v", err)
	}

	// After provision, registration works - lazily resolved.
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component after provide: %v", err)
	}
	// Installing a custom codec on the game's codecs after registration
	// is picked up by the registry's captured instance.
	if _, err := reg.DecodeComponent("game.Trader", []byte(`{"name":"Vex"}`)); err != nil {
		t.Fatalf("DecodeComponent: %v", err)
	}
}

func TestComponentInstallsJSONOnGameCodecs(t *testing.T) {
	reg, codecs := newTestRegistry(t)
	if err := reg.Component[trader]("game.Trader"); err != nil {
		t.Fatalf("Component: %v", err)
	}
	var buf bytes.Buffer
	if err := codecs.Encode[trader](&buf, trader{Name: "Vex"}, asset.FormatJSON); err != nil {
		t.Fatalf("encode through game codecs: %v", err)
	}
}
