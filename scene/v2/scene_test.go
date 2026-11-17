package scene

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

type point struct{ X, Y float64 }

type invalidatable struct{}

func (invalidatable) Validate() error { return errors.New("bad component") }

type keyedProps struct{ Key string }

type statData struct {
	Level int `json:"level"`
}

func decodeStat(r io.Reader) (statData, error) {
	var s statData
	return s, json.NewDecoder(r).Decode(&s)
}

// statLoader returns a loader that decodes statData from the given files.
func statLoader(t *testing.T, files ...string) *asset.Loader {
	t.Helper()
	fsmap := fstest.MapFS{}
	for _, name := range files {
		fsmap[name] = &fstest.MapFile{Data: []byte(`{"level":3}`)}
	}
	codecs := asset.NewCodecs()
	if err := codecs.SetDecoder(asset.FormatJSON, decodeStat); err != nil {
		t.Fatalf("register stat decoder: %v", err)
	}
	loader, err := asset.NewLoader(fsmap, asset.LoaderOptions{Codecs: codecs})
	if err != nil {
		t.Fatalf("loader: %v", err)
	}
	return loader
}

func static(tree Node) Scene {
	return func(any) (Node, error) { return tree, nil }
}

type lifeProps struct{ Key string }

func lifeScene() Scene {
	return Of(func(p lifeProps) Node {
		return Node{Key: p.Key, Children: []Node{{Key: "child"}}}
	})
}

func countPointEntities(world *core.World) int {
	n := 0
	for range core.NewQuery(world).With(point{}).Execute() {
		n++
	}
	return n
}

func TestOf(t *testing.T) {
	s := Of(func(k keyedProps) Node { return Node{Key: k.Key} })

	tree, err := s(keyedProps{"root"})
	if err != nil || tree.Key != "root" {
		t.Fatalf("typed build: tree=%+v err=%v", tree, err)
	}

	_, err = s("not keyed")
	if err == nil || !strings.Contains(err.Error(), "string") || !strings.Contains(err.Error(), "keyedProps") {
		t.Fatalf("wrong props type: %v", err)
	}

	if _, err = s(nil); err == nil {
		t.Fatal("nil props accepted for value type")
	}
}

func TestWarm(t *testing.T) {
	loader := statLoader(t, "stat.json")
	w := Warm[statData]("stat.json")

	if w.id() != "stat.json" {
		t.Fatalf("id: %v", w.id())
	}
	if err := w.warm(loader); err != nil {
		t.Fatalf("warm: %v", err)
	}
	if err := Warm[statData]("missing.json").warm(loader); err == nil {
		t.Fatal("warm of missing asset succeeded")
	}
}

func leaf(key string) Node {
	return Node{Key: key, Components: []any{point{}}}
}

func TestValidateNode(t *testing.T) {
	tests := []struct {
		name string
		tree Node
		want string
	}{
		{"valid nested tree", Node{Key: "root", Children: []Node{
			{Key: "left", Children: []Node{leaf("leaf")}},
			{Key: "right", Children: []Node{leaf("leaf")}},
		}}, ""},
		{"same key on different levels", Node{Key: "root", Children: []Node{
			{Key: "x", Children: []Node{leaf("x")}},
		}}, ""},
		{"empty root key", Node{}, `node at ""`},
		{"empty child key", Node{Key: "root", Children: []Node{{}}}, `node at "root/"`},
		{"slash in key", Node{Key: "a/b"}, `must not contain '/'`},
		{"duplicate sibling keys", Node{Key: "root", Children: []Node{leaf("x"), leaf("x")}}, `duplicate sibling key "x"`},
		{"nil component", Node{Key: "n", Components: []any{nil}}, "component at index 0 is nil"},
		{"pointer component", Node{Key: "n", Components: []any{&point{}}}, "is a pointer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validate(static(tt.tree), nil)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestValidatePropagatesBuilderError(t *testing.T) {
	boom := errors.New("builder exploded")
	s := func(any) (Node, error) { return Node{}, boom }
	if _, err := validate(s, nil); !errors.Is(err, boom) {
		t.Fatalf("builder error not propagated: %v", err)
	}
}

func TestWalkAssets(t *testing.T) {
	tree := Node{Key: "root", Assets: []Preload{
		Warm[statData]("a.json"),
		Warm[statData]("b.json"),
	}, Children: []Node{
		{Key: "kid", Assets: []Preload{Warm[statData]("c.json")}},
	}}
	stop := errors.New("stop")
	var order []asset.ID
	err := walkAssets(tree, func(p Preload) error {
		order = append(order, p.id())
		if p.id() == "b.json" {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) {
		t.Fatalf("visit error not propagated: %v", err)
	}
	if len(order) != 2 || order[0] != "a.json" || order[1] != "b.json" {
		t.Fatalf("walk did not short-circuit: %v", order)
	}
}

func TestPreload(t *testing.T) {
	loader := statLoader(t, "stat.json")
	tree := Node{Key: "root", Assets: []Preload{Warm[statData]("stat.json")}, Children: []Node{
		{Key: "kid", Assets: []Preload{Warm[statData]("missing.json")}},
	}}
	err := preload(loader, tree)
	if err == nil || !errors.Is(err, ErrAssetLoadFailed) {
		t.Fatalf("want ErrAssetLoadFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing.json") {
		t.Fatalf("error omits asset id: %v", err)
	}

	if err := preload(loader, Node{Key: "ok", Assets: []Preload{Warm[statData]("stat.json")}}); err != nil {
		t.Fatalf("valid assets failed to warm: %v", err)
	}
}

func TestEnter(t *testing.T) {
	world := core.NewWorld()
	loader := statLoader(t, "stat.json")
	tree := Node{Key: "root", Components: []any{point{}}, Assets: []Preload{Warm[statData]("stat.json")}, Children: []Node{
		{Key: "kid", Components: []any{point{}}},
	}}

	inst := &instance{id: 1, name: "scene"}
	if err := enter(world, loader, inst, tree); err != nil {
		t.Fatal(err)
	}
	if !inst.live || inst.tree.Key != "root" || len(inst.entities) != 2 {
		t.Fatalf("enter state: %+v", inst)
	}
	for i, ent := range inst.entities {
		if !world.HasEntity(ent) {
			t.Fatalf("entity %d not in world", i)
		}
	}

	badAssets := Node{Key: "bad", Assets: []Preload{Warm[statData]("missing.json")}}
	untouched := &instance{id: 2, name: "untouched"}
	if err := enter(world, loader, untouched, badAssets); err == nil {
		t.Fatal("preload failure not reported")
	} else if untouched.live || untouched.entities != nil || untouched.tree.Key != "" {
		t.Fatalf("failed entry mutated instance: %+v", untouched)
	}
	if !world.HasEntity(inst.entities[0]) {
		t.Fatal("failed entry of another instance disturbed a live one")
	}

	broken := Node{Key: "root", Components: []any{point{}}, Children: []Node{
		{Key: "ok", Components: []any{point{}}},
		{Key: "bad", Components: []any{invalidatable{}}},
	}}
	fresh := &instance{id: 3}
	err := enter(world, nil, fresh, broken)
	if err == nil || !strings.Contains(err.Error(), `"root/bad"`) {
		t.Fatalf("spawn failure: %v", err)
	} else if strings.Contains(err.Error(), "destroy entity") {
		t.Fatalf("cleanup during failed entry also failed: %v", err)
	}
	if fresh.live || fresh.entities != nil {
		t.Fatalf("failed entry mutated instance: %+v", fresh)
	}
	if countPointEntities(world) != 2 {
		t.Fatalf("partial spawn leaked entities: %d left", countPointEntities(world)-2)
	}
}

func TestEnterPanicsOnLiveInstance(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("enter on live instance did not panic")
		}
	}()
	live := &instance{id: 1, live: true}
	_ = enter(core.NewWorld(), nil, live, Node{Key: "root"})
}

func TestExit(t *testing.T) {
	world := core.NewWorld()
	inst := &instance{id: 1}
	if err := enter(world, nil, inst, Node{Key: "root", Children: []Node{{Key: "kid"}}}); err != nil {
		t.Fatal(err)
	}
	entities := inst.entities

	if err := exit(world, inst); err != nil {
		t.Fatal(err)
	}
	if inst.live || inst.entities != nil || inst.tree.Key != "" {
		t.Fatalf("exit did not reset instance: %+v", inst)
	}
	for i, ent := range entities {
		if world.HasEntity(ent) {
			t.Fatalf("entity %d survived exit", i)
		}
	}
}

func TestDestroy(t *testing.T) {
	world := core.NewWorld()
	ent, err := world.NewEntity()
	if err != nil {
		t.Fatal(err)
	}
	if err := destroy(world, nil); err != nil {
		t.Fatalf("destroy of no entities: %v", err)
	}
	if err := world.DestroyEntity(ent); err != nil {
		t.Fatal(err)
	}
	err = destroy(world, []core.Entity{ent})
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("entity %d", ent.ID())) {
		t.Fatalf("destroy of dead entity: %v", err)
	}
}

func TestJoin(t *testing.T) {
	for _, tt := range []struct{ path, key, want string }{
		{"", "k", "k"},
		{"p", "k", "p/k"},
	} {
		if got := join(tt.path, tt.key); got != tt.want {
			t.Errorf("join(%q, %q) = %q, want %q", tt.path, tt.key, got, tt.want)
		}
	}
}
