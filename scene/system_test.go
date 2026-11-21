package scene

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

type marker struct{}

func markerScene() Scene {
	return Of(func(p lifeProps) Node {
		return Node{Key: p.Key, Components: []any{marker{}}, Children: []Node{
			{Key: "m", Components: []any{marker{}}},
		}}
	})
}

func countMarkers(world *core.World) int {
	n := 0
	for range core.NewQuery(world).With(marker{}).Execute() {
		n++
	}
	return n
}

func newTestSystem(t *testing.T) (*Director, core.System, *core.Context) {
	t.Helper()
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*asset.Loader, error) {
		return asset.NewLoader(fstest.MapFS{}, asset.LoaderOptions{})
	}); err != nil {
		t.Fatalf("provide loader: %v", err)
	}
	d := NewDirector()
	return d, NewSystem(d), &core.Context{World: world}
}

func flush(t *testing.T, sys core.System, ctx *core.Context) {
	t.Helper()
	if err := sys.Update(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSwitchReplacesScene(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	a, _ := d.Switch(lifeScene(), lifeProps{"a"})
	flush(t, sys, ctx)
	if !d.Live(a) || len(d.Entities(a)) != 2 || len(d.roots) != 1 || d.roots[0] != a {
		t.Fatalf("after first switch: live=%v ents=%v roots=%v", d.Live(a), d.Entities(a), d.roots)
	}
	oldEnt := d.Entities(a)[0]
	if !ctx.World.HasEntity(oldEnt) {
		t.Fatal("entered entity not in world")
	}

	b, _ := d.Switch(lifeScene(), lifeProps{"b"})
	flush(t, sys, ctx)
	if ctx.World.HasEntity(oldEnt) {
		t.Error("replaced scene entity survived switch")
	}
	if !d.Live(b) || d.Live(a) || len(d.roots) != 1 || d.roots[0] != b {
		t.Fatalf("after second switch: a=%v b=%v roots=%v", d.Live(a), d.Live(b), d.roots)
	}
}

func TestLoadIsAdditive(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	a, _ := d.Load(lifeScene(), lifeProps{"a"})
	b, _ := d.Load(lifeScene(), lifeProps{"b"})
	flush(t, sys, ctx)

	if !d.Live(a) || !d.Live(b) {
		t.Fatalf("instances not live: a=%v b=%v", d.Live(a), d.Live(b))
	}
	if len(d.roots) != 2 || d.roots[0] != a || d.roots[1] != b {
		t.Fatalf("roots=%v", d.roots)
	}
}

func TestUpdateRebuildsAndInherits(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	id, _ := d.Load(lifeScene(), lifeProps{"v1"}, WithName("home"))
	flush(t, sys, ctx)
	oldEnt := d.Entities(id)[0]

	if err := d.Update(id, lifeScene(), lifeProps{"v2"}); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)
	if name, _ := d.Name(id); name != "home" {
		t.Fatalf("rebuild did not inherit name: %q", name)
	}
	if ctx.World.HasEntity(oldEnt) || d.Entities(id)[0] == oldEnt {
		t.Fatal("rebuild did not respawn entities")
	}
	if intree := d.instances[id].tree.Key; intree != "v2" {
		t.Fatalf("rebuild did not replace tree: %q", intree)
	}

	if err := d.Update(id, lifeScene(), lifeProps{"v3"}, WithName("away")); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)
	if name, _ := d.Name(id); name != "away" {
		t.Fatalf("explicit name not applied: %q", name)
	}
}

func TestRemoveUnloadsInstance(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	id, _ := d.Load(lifeScene(), lifeProps{"a"})
	flush(t, sys, ctx)
	ent := d.Entities(id)[0]

	if err := d.Remove(id); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)
	if len(d.instances) != 0 || len(d.roots) != 0 {
		t.Fatalf("records not cleared: instances=%d roots=%v", len(d.instances), d.roots)
	}
	if ctx.World.HasEntity(ent) {
		t.Error("entity survived remove")
	}
}

func TestHooksFireAroundLifecycle(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	var events []string
	hooks := Hooks{
		Enter: func(*core.Context, *Handle) error {
			events = append(events, "enter")
			return nil
		},
		Exit: func(_ *core.Context, h *Handle) error {
			events = append(events, fmt.Sprintf("exit:%d", h.ID()))
			if !ctx.World.HasEntity(d.Entities(h.ID())[0]) {
				t.Error("exit hook ran after teardown")
			}
			return nil
		},
	}
	id, _ := d.Load(lifeScene(), lifeProps{"r"}, WithHooks(hooks))
	if err := sys.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if want := []string{"enter"}; !equalStrings(events, want) {
		t.Fatalf("after enter: %v", events)
	}

	if err := d.Remove(id); err != nil {
		t.Fatal(err)
	}
	if err := sys.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if want := []string{"enter", fmt.Sprintf("exit:%d", id)}; !equalStrings(events, want) {
		t.Fatalf("lifecycle: %v", events)
	}
}

func TestSameBatchRequestsExecuteOnce(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	id, _ := d.Load(lifeScene(), lifeProps{"v1"})
	if err := d.Update(id, lifeScene(), lifeProps{"v2"}, WithName("renamed")); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)

	if name, _ := d.Name(id); name != "renamed" {
		t.Fatalf("folded name not applied: %q", name)
	}
	if ents := d.Entities(id); len(ents) != 2 {
		t.Fatalf("folded batch spawned twice: %d entities", len(ents))
	}
	if len(d.pending) != 0 {
		t.Fatalf("queue not drained: %d", len(d.pending))
	}
}

func TestUnknownIdsFailAtExecution(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	if err := d.Update(99, lifeScene(), lifeProps{"x"}); err != nil {
		t.Fatal(err)
	}
	err := sys.Update(ctx)
	if err == nil || !strings.Contains(err.Error(), "99") {
		t.Fatalf("update of unknown id: %v", err)
	}
	if len(d.pending) != 0 {
		t.Fatal("queue not drained after errors")
	}

	if err := d.Remove(99); err != nil {
		t.Fatal(err)
	}
	if err := sys.Update(ctx); err == nil || !strings.Contains(err.Error(), "99") {
		t.Fatalf("remove of unknown id: %v", err)
	}
}

func TestSwitchStagesNewSceneBeforeTeardown(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	a, _ := d.Switch(lifeScene(), lifeProps{"a"})
	flush(t, sys, ctx)
	oldEnt := d.Entities(a)[0]

	newSawOld, oldSawOwnEntities := false, false
	d.instances[a].hooks = Hooks{
		Exit: func(*core.Context, *Handle) error {
			oldSawOwnEntities = ctx.World.HasEntity(oldEnt)
			return nil
		},
	}
	newHooks := Hooks{
		Enter: func(*core.Context, *Handle) error {
			newSawOld = d.Live(a) && ctx.World.HasEntity(oldEnt)
			return nil
		},
	}
	b, _ := d.Switch(lifeScene(), lifeProps{"b"}, WithHooks(newHooks))
	flush(t, sys, ctx)

	if !newSawOld {
		t.Error("outgoing scene was torn down before the new Enter hook")
	}
	if !oldSawOwnEntities {
		t.Error("outgoing scene's entities were destroyed before its Exit hook")
	}
	if d.Live(a) || ctx.World.HasEntity(oldEnt) {
		t.Error("outgoing scene survived committed switch")
	}
	if !d.Live(b) {
		t.Error("new scene not live after switch")
	}
}

func TestEnterHookFailureRollsBackSwitch(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	a, _ := d.Switch(lifeScene(), lifeProps{"a"})
	flush(t, sys, ctx)

	refuse := errors.New("refused")
	failing := Hooks{
		Enter: func(*core.Context, *Handle) error { return refuse },
	}
	b, _ := d.Switch(markerScene(), lifeProps{"b"}, WithHooks(failing))
	err := sys.Update(ctx)
	if err == nil || !errors.Is(err, refuse) {
		t.Fatalf("failed Enter hook: %v", err)
	}
	if !d.Live(a) || len(d.roots) != 1 || d.roots[0] != a {
		t.Fatalf("live scene disturbed by failed switch: live=%v roots=%v", d.Live(a), d.roots)
	}
	if d.Live(b) {
		t.Error("scene committed despite failed Enter hook")
	}
	if countMarkers(ctx.World) != 0 {
		t.Fatalf("staged entities leaked on rollback: %d", countMarkers(ctx.World))
	}
}

func TestUpdateRollsBackOnFailedStage(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	id, _ := d.Load(lifeScene(), lifeProps{"v1"}, WithName("home"))
	flush(t, sys, ctx)
	oldEnt := d.Entities(id)[0]
	exitFired := false
	d.instances[id].hooks = Hooks{
		Exit: func(*core.Context, *Handle) error {
			exitFired = true
			return nil
		},
	}

	refuse := errors.New("refused")
	failing := Hooks{
		Enter: func(*core.Context, *Handle) error { return refuse },
	}
	if err := d.Update(id, markerScene(), lifeProps{"v2"}, WithHooks(failing)); err != nil {
		t.Fatal(err)
	}
	if err := sys.Update(ctx); err == nil || !errors.Is(err, refuse) {
		t.Fatalf("failed rebuild: %v", err)
	}
	if !d.Live(id) || d.Entities(id)[0] != oldEnt {
		t.Fatal("failed rebuild mutated live instance's entities")
	}
	if name, _ := d.Name(id); name != "home" {
		t.Fatalf("failed rebuild replaced name: %q", name)
	}
	if exitFired {
		t.Error("Exit hook fired despite rolled-back rebuild")
	}
	if countMarkers(ctx.World) != 0 {
		t.Fatalf("staged entities leaked on rollback: %d", countMarkers(ctx.World))
	}
}

func TestLoaderFetchedLazily(t *testing.T) {
	world := core.NewWorld()
	d := NewDirector()
	sys := NewSystem(d)
	ctx := &core.Context{World: world}

	id, _ := d.Switch(lifeScene(), lifeProps{"plain"})
	if err := sys.Update(ctx); err != nil {
		t.Fatalf("asset-free scene required loader resource: %v", err)
	}
	if !d.Live(id) {
		t.Fatal("asset-free scene not live")
	}

	warmScene := Of(func(p lifeProps) Node {
		return Node{Key: p.Key, Assets: []Preload{Warm[statData]("a.json")}}
	})
	hot, err := d.Load(warmScene, lifeProps{"hot"})
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.Update(ctx); err == nil || !strings.Contains(err.Error(), "scene: system") {
		t.Fatalf("wanted loader error, got %v", err)
	}
	if d.Live(hot) {
		t.Error("failed load committed")
	}
}

func TestSpawnFromHooks(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	var enterEnt core.Entity
	loaded := Hooks{
		Enter: func(c *core.Context, h *Handle) error {
			ent, err := d.Spawn(c, h.ID(), marker{})
			if err != nil {
				return err
			}
			enterEnt = ent
			return nil
		},
	}
	id, _ := d.Load(lifeScene(), lifeProps{"m"}, WithHooks(loaded))
	flush(t, sys, ctx)
	if !ctx.World.HasEntity(enterEnt) {
		t.Fatal("spawn from Enter hook did not survive entry")
	}
	if ents := d.Entities(id); len(ents) != 3 {
		t.Fatalf("Enter-hook spawn not owned: %d entities", len(ents))
	}

	var updateEnt, exitEnt core.Entity
	updateHooks := Hooks{
		Enter: func(c *core.Context, h *Handle) error {
			ent, err := d.Spawn(c, h.ID(), marker{})
			if err != nil {
				return err
			}
			updateEnt = ent
			return nil
		},
	}
	d.instances[id].hooks = Hooks{
		Exit: func(c *core.Context, h *Handle) error {
			ent, err := d.Spawn(c, h.ID(), marker{})
			if err != nil {
				return err
			}
			exitEnt = ent
			return nil
		},
	}
	if err := d.Update(id, lifeScene(), lifeProps{"v2"}, WithHooks(updateHooks)); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)

	if !ctx.World.HasEntity(updateEnt) {
		t.Error("Enter-hook spawn destroyed with the old generation")
	}
	if ctx.World.HasEntity(exitEnt) {
		t.Error("Exit-hook spawn leaked past the commit")
	}
	if ents := d.Entities(id); len(ents) != 3 {
		t.Fatalf("ownership after rebuild: %d entities", len(ents))
	}
	if n := countMarkers(ctx.World); n != 1 {
		t.Fatalf("world holds %d owned markers, want 1", n)
	}
}

func TestExecuteFlushesInputEdges(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	am, err := input.New(input.Bindings{
		"confirm": []input.Input{input.KeyInput{Key: input.KeySpace}.Binding()},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	press := input.Snapshot{}
	press.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	am.Update(&press, 1.0/60)
	if !am.JustPressed("confirm") {
		t.Fatal("setup: no press edge")
	}
	if err := ctx.World.Provide(func(*core.World) (*input.ActionMap, error) {
		return am, nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := d.Switch(lifeScene(), lifeProps{"menu"}); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)
	if am.JustPressed("confirm") {
		t.Fatal("press edge survived the batch that applied the scene")
	}

	// an empty batch leaves edges alone
	release := input.Snapshot{}
	release.Keys[input.KeySpace] = input.State{Released: true}
	am.Update(&release, 1.0/60)
	am.Update(&press, 1.0/60)
	if !am.JustPressed("confirm") {
		t.Fatal("setup: no second press edge")
	}
	flush(t, sys, ctx)
	if !am.JustPressed("confirm") {
		t.Fatal("empty batch flushed edges")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
