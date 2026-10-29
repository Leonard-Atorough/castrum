package scene

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

type position struct{ X float64 }

// failing is a component whose Validate always fails.
var _ core.Validatable = failing{}

type failing struct{}

func (failing) Validate() error { return errors.New("failing component") }

type pointerValidated struct{ value int }

func (p *pointerValidated) Validate() error {
	_ = p.value
	return nil
}

type failAfterBlueprintValidation struct {
	calls *int
}

func (f failAfterBlueprintValidation) Validate() error {
	*f.calls++
	if *f.calls > 1 {
		return errors.New("failing component")
	}
	return nil
}

func testBlueprint(name string) *Blueprint {
	return &Blueprint{
		Name: name,
		EntityList: []EntitySpec{
			{Name: "root", Components: []any{position{}}},
		},
	}
}

func subBlueprint(name string) Blueprint {
	return Blueprint{
		Name: name,
		EntityList: []EntitySpec{
			{Name: "only", Components: []any{position{}}},
		},
	}
}

// loadTestInstance queues a load for a named blueprint and returns its
// reserved ID.
func loadTestInstance(t *testing.T, d *Director, name string) InstanceID {
	t.Helper()
	id, err := d.Load(testBlueprint(name))
	if err != nil {
		t.Fatalf("Load %s: %v", name, err)
	}
	return id
}

// liveTestInstance loads an instance and marks it live directly. Liveness
// belongs to the scene systems, which are not implemented yet; request-side
// contracts still need live instances to test against.
func liveTestInstance(t *testing.T, d *Director, name string) InstanceID {
	t.Helper()
	id := loadTestInstance(t, d, name)
	d.instances[id].live = true
	return id
}

func TestBlueprintValidateAccepts(t *testing.T) {
	bp := &Blueprint{
		Name:   "level",
		Assets: []asset.ID{"sprites/tank.png"},

		EntityList: []EntitySpec{
			{Name: "tank", Components: []any{position{}}},
			{Name: "floor", Children: []EntitySpec{
				{Name: "tile", Components: []any{position{}}},
			}},
			{Name: "spawner", SubScenes: []Blueprint{subBlueprint("enemies")}},
		},
	}
	if err := bp.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestBlueprintValidateRejects(t *testing.T) {
	cases := []struct {
		name     string
		mut      func(*Blueprint)
		contains string
	}{
		{"missing name", func(bp *Blueprint) { bp.Name = "" }, ""},
		{"empty entity list", func(bp *Blueprint) { bp.EntityList = nil }, ""},
		{"empty asset ID", func(bp *Blueprint) {
			bp.Assets = []asset.ID{"ok", ""}
		}, ""},
		{"unnamed entity", func(bp *Blueprint) {
			bp.EntityList[0].Children = []EntitySpec{{Components: []any{position{}}}}
		}, `"level"`},
		{"duplicate sibling names", func(bp *Blueprint) {
			bp.EntityList = append(bp.EntityList, EntitySpec{Name: "root"})
		}, `"root"`},
		{"nil component", func(bp *Blueprint) {
			bp.EntityList[0].Components = []any{position{}, nil}
		}, ""},
		{"failing Validatable", func(bp *Blueprint) {
			bp.EntityList[0].Components = []any{failing{}}
		}, ""},
		{"invalid subscene", func(bp *Blueprint) {
			bad := subBlueprint("enemies")
			bad.EntityList[0].Components = []any{nil}
			bp.EntityList[0].SubScenes = []Blueprint{bad}
		}, `"enemies"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp := testBlueprint("level")
			tc.mut(bp)
			err := bp.Validate()
			if err == nil {
				t.Fatalf("Validate accepted %s", tc.name)
			}
			if tc.contains != "" && !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error %q missing context %s", err, tc.contains)
			}
		})
	}
}

func TestBlueprintValidateAllowsSameNameAcrossScopes(t *testing.T) {
	bp := testBlueprint("level")
	bp.EntityList = append(bp.EntityList,
		EntitySpec{Name: "left", Children: []EntitySpec{{Name: "inner"}}},
		EntitySpec{Name: "right", Children: []EntitySpec{{Name: "inner"}}},
	)
	if err := bp.Validate(); err != nil {
		t.Fatalf("Validate rejected sibling-scoped names: %v", err)
	}
}

func TestBlueprintValidateAllowsDeepNesting(t *testing.T) {
	// Value nesting is finite by construction; repeated composition must
	// validate rather than recurse without bound.
	bp := testBlueprint("level")
	sub := testBlueprint("room")
	sub.EntityList[0].SubScenes = []Blueprint{subBlueprint("cellar")}
	bp.EntityList[0].SubScenes = []Blueprint{*sub}
	if err := bp.Validate(); err != nil {
		t.Fatalf("Validate rejected deep nesting: %v", err)
	}
}

func TestBlueprintRejectsPointerComponentsBeforeQueueing(t *testing.T) {
	var nilPointer *pointerValidated
	for _, component := range []any{&position{}, &pointerValidated{}, nilPointer} {
		for _, placement := range []string{"root", "child", "subscene"} {
			t.Run(fmt.Sprintf("%T/%s", component, placement), func(t *testing.T) {
				bp := testBlueprint("level")
				switch placement {
				case "root":
					bp.EntityList[0].Components = []any{component}
				case "child":
					bp.EntityList[0].Children = []EntitySpec{{Name: "child", Components: []any{component}}}
				case "subscene":
					sub := subBlueprint("nested")
					sub.EntityList[0].Components = []any{component}
					bp.EntityList[0].SubScenes = []Blueprint{sub}
				}
				d := NewDirector()
				id := liveTestInstance(t, d, "existing")
				before, next := len(d.pending), d.next
				for name, call := range map[string]func() error{
					"Validate": bp.Validate,
					"Load":     func() error { _, err := d.Load(bp); return err },
					"Switch":   func() error { _, err := d.Switch(bp); return err },
					"Update":   func() error { return d.Update(id, bp) },
				} {
					if err := call(); err == nil || !strings.Contains(err.Error(), "pointer") {
						t.Errorf("%s error = %v, want pointer error", name, err)
					}
				}
				if len(d.pending) != before || d.next != next {
					t.Fatal("rejected request mutated Director state")
				}
			})
		}
	}
}

func TestRejectedRequestsReserveNothing(t *testing.T) {
	cases := []struct {
		name string
		call func(*Director) error
	}{
		{"Switch nil", func(d *Director) error {
			_, err := d.Switch(nil)
			return err
		}},
		{"Switch invalid", func(d *Director) error {
			bp := testBlueprint("level")
			bp.EntityList[0].Components = []any{nil}
			_, err := d.Switch(bp)
			return err
		}},
		{"Load nil", func(d *Director) error {
			_, err := d.Load(nil)
			return err
		}},
		{"Load invalid", func(d *Director) error {
			bp := testBlueprint("level")
			bp.EntityList[0].Components = []any{nil}
			_, err := d.Load(bp)
			return err
		}},
		{"Update nil", func(d *Director) error {
			return d.Update(1, nil)
		}},
		{"Update invalid", func(d *Director) error {
			bp := testBlueprint("elite")
			bp.EntityList[0].Components = []any{nil}
			return d.Update(1, bp)
		}},
		{"Update unknown", func(d *Director) error {
			return d.Update(9, testBlueprint("elite"))
		}},
		{"Update pending", func(d *Director) error {
			return d.Update(1, testBlueprint("elite"))
		}},
		{"Remove unknown", func(d *Director) error {
			return d.Remove(9)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDirector()
			// Instance 1 is a pending target for the Update cases.
			loadTestInstance(t, d, "arena")
			before, next := len(d.pending), d.next
			if err := tc.call(d); err == nil {
				t.Fatalf("%s: request accepted", tc.name)
			}
			if len(d.pending) != before || d.next != next {
				t.Fatalf("%s: rejected request mutated Director state", tc.name)
			}
		})
	}
}

func TestSwitchAndLoadReserveAndQueue(t *testing.T) {
	cases := []struct {
		name string
		call func(*Director, *Blueprint) (InstanceID, error)
		kind requestKind
	}{
		{"Switch", (*Director).Switch, requestSwitch},
		{"Load", (*Director).Load, requestLoad},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDirector()
			bp := testBlueprint("level")
			id, err := tc.call(d, bp)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if len(d.pending) != 1 {
				t.Fatalf("pending requests: got %d, want 1", len(d.pending))
			}
			req := d.pending[0]
			if req.kind != tc.kind || req.id != id || req.blueprint == bp || req.blueprint.Name != bp.Name {
				t.Fatalf("queued request: got %+v", req)
			}
			if rec, ok := d.instances[id]; !ok || rec.live {
				t.Fatalf("reserved instance: got %+v, ok=%v", rec, ok)
			}
			view, ok := d.Get(id)
			if !ok || view.ID != id || view.Name != "level" || view.Blueprint.Name != bp.Name {
				t.Fatalf("Get view: got %+v, ok=%v", view, ok)
			}
		})
	}
}

func TestLoadSnapshotsBlueprintStructure(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := testBlueprint("level")
	bp.Assets = []asset.ID{"original"}
	bp.EntityList[0].Components = []any{position{X: 1}}
	bp.EntityList[0].Children = []EntitySpec{{
		Name:       "child",
		Components: []any{position{X: 2}},
	}}
	bp.EntityList[0].SubScenes = []Blueprint{subBlueprint("nested")}

	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bp.Name = "changed"
	bp.Assets[0] = "changed"
	bp.EntityList[0].Name = "changed"
	bp.EntityList[0].Components[0] = position{X: 10}
	bp.EntityList[0].Children[0].Name = "changed child"
	bp.EntityList[0].Children[0].Components[0] = position{X: 20}
	bp.EntityList[0].SubScenes[0].Name = "changed nested"

	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	inst, ok := d.Get(id)
	if !ok || inst.Name != "level" || inst.Blueprint.Name != "level" {
		t.Fatalf("instance after load: %+v, found = %t", inst, ok)
	}
	if inst.Blueprint.Assets[0] != "original" {
		t.Fatalf("blueprint slices were not snapshotted: %+v", inst.Blueprint)
	}
	root := inst.Blueprint.EntityList[0]
	if root.Name != "root" || root.Components[0] != (position{X: 1}) ||
		root.Children[0].Name != "child" || root.Children[0].Components[0] != (position{X: 2}) ||
		root.SubScenes[0].Name != "nested" {
		t.Fatalf("entity tree was not snapshotted: %+v", root)
	}
	if got := ownedCount(ctx.World, id); got != 2 {
		t.Fatalf("root entities: got %d, want root and child", got)
	}
	nested, ok := d.ByName("nested")
	if !ok || ownedCount(ctx.World, nested.ID) != 1 {
		t.Fatalf("nested scene did not spawn from snapshot: instance=%+v, found=%t", nested, ok)
	}
}

func TestInstanceBlueprintSnapshotIsDetached(t *testing.T) {
	d := NewDirector()
	bp := testBlueprint("level")
	bp.Assets = []asset.ID{"original"}
	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	view, ok := d.Get(id)
	if !ok {
		t.Fatal("Get did not find pending instance")
	}
	view.Blueprint.Name = "mutated view"
	view.Blueprint.Assets[0] = "mutated asset"
	view.Blueprint.EntityList[0].Name = "mutated entity"

	again, ok := d.Get(id)
	if !ok || again.Blueprint.Name != "level" ||
		again.Blueprint.Assets[0] != "original" ||
		again.Blueprint.EntityList[0].Name != "root" {
		t.Fatalf("mutating returned snapshot changed Director state: %+v, found = %t", again, ok)
	}
}

func TestHookAndNestedInstanceBlueprintViewsAreDetached(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	child := subBlueprint("nested")
	child.Assets = []asset.ID{"child asset"}
	bp := nestedBlueprint("root", child)
	bp.Assets = []asset.ID{"root asset"}
	bp.Hooks.Enter = func(_ *core.Context, self *Instance) error {
		self.Blueprint.Assets[0] = "changed"
		self.Blueprint.EntityList[0].SubScenes[0].Name = "changed"
		return nil
	}
	bp.Hooks.Exit = func(_ *core.Context, self *Instance) error {
		self.Blueprint.Assets[0] = "changed"
		return nil
	}
	id := enterTestInstance(t, d, ctx, bp)
	view, ok := d.ByName("nested")
	if !ok {
		t.Fatal("nested scene missing after mutating parent hook view")
	}
	view.Blueprint.Assets[0] = "changed"
	view.Blueprint.EntityList[0].Name = "changed"
	again, ok := d.Get(view.ID)
	if !ok || again.Blueprint.Assets[0] != "child asset" || again.Blueprint.EntityList[0].Name != "only" {
		t.Fatalf("nested instance changed through view: %+v, found=%t", again, ok)
	}
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain remove: %v", err)
	}
}

func TestLoadKeepsExistingInstances(t *testing.T) {
	d := NewDirector()
	loadTestInstance(t, d, "menu")
	loadTestInstance(t, d, "hud")
	if len(d.instances) != 2 || len(d.pending) != 2 {
		t.Fatalf("instances=%d pending=%d, want 2 and 2", len(d.instances), len(d.pending))
	}
}

func TestGetUnknownID(t *testing.T) {
	d := NewDirector()
	if _, ok := d.Get(42); ok {
		t.Fatal("Get reported an unknown instance")
	}
}

func TestByNameFindsPending(t *testing.T) {
	d := NewDirector()
	id := loadTestInstance(t, d, "menu")
	view, ok := d.ByName("menu")
	if !ok || view.ID != id {
		t.Fatalf("ByName: got %+v, ok=%v", view, ok)
	}
}

func TestByNameUnknown(t *testing.T) {
	d := NewDirector()
	if _, ok := d.ByName("nothing"); ok {
		t.Fatal("ByName reported an unknown instance")
	}
}

func TestByNameLowestIDWins(t *testing.T) {
	d := NewDirector()
	first := loadTestInstance(t, d, "arena")
	loadTestInstance(t, d, "arena")
	view, ok := d.ByName("arena")
	if !ok || view.ID != first {
		t.Fatalf("ByName: got %+v, ok=%v; want lowest ID %d", view, ok, first)
	}
}

func TestUpdateQueuesForLiveInstance(t *testing.T) {
	d := NewDirector()
	id := liveTestInstance(t, d, "arena")
	bp := testBlueprint("elite")
	if err := d.Update(id, bp); err != nil {
		t.Fatalf("Update: %v", err)
	}
	req := d.pending[1]
	if req.kind != requestUpdate || req.id != id || req.blueprint == bp || req.blueprint.Name != bp.Name {
		t.Fatalf("queued request: got %+v", req)
	}
	// Identity survives: the record keeps its original name even though the
	// replacement blueprint carries a different one.
	if rec := d.instances[id]; rec.name != "arena" {
		t.Fatalf("record name: got %q, want %q", rec.name, "arena")
	}
}

func TestRemoveQueuesTarget(t *testing.T) {
	cases := []struct {
		name string
		live bool
	}{
		{"pending target", false},
		{"live target", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDirector()
			var id InstanceID
			if tc.live {
				id = liveTestInstance(t, d, "menu")
			} else {
				id = loadTestInstance(t, d, "menu")
			}
			if err := d.Remove(id); err != nil {
				t.Fatalf("Remove: %v", err)
			}
			req := d.pending[1]
			if req.kind != requestUnload || req.id != id {
				t.Fatalf("queued request: got %+v", req)
			}
			if d.next != 2 {
				t.Fatalf("Remove reserved an ID: next=%d", d.next)
			}
		})
	}
}

func TestRequestsQueueInCallOrder(t *testing.T) {
	d := NewDirector()
	if _, err := d.Switch(testBlueprint("level")); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	loadTestInstance(t, d, "hud")
	arenaID := liveTestInstance(t, d, "arena")
	if err := d.Update(arenaID, testBlueprint("elite")); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := d.Remove(arenaID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	want := []requestKind{requestSwitch, requestLoad, requestLoad, requestUpdate, requestUnload}
	if len(d.pending) != len(want) {
		t.Fatalf("pending requests: got %d, want %d", len(d.pending), len(want))
	}
	for i, kind := range want {
		if d.pending[i].kind != kind {
			t.Fatalf("request %d: got %v, want %v", i, d.pending[i].kind, kind)
		}
	}
}

func TestInstanceIDsAreSequential(t *testing.T) {
	d := NewDirector()
	want := []InstanceID{1, 2, 3}
	for i, wantID := range want {
		if id := loadTestInstance(t, d, "arena"); id != wantID {
			t.Fatalf("instance %d: got ID %d, want %d", i, id, wantID)
		}
	}
}

// --- actor fixtures ---

func ownedCount(world *core.World, id InstanceID) int {
	n := 0
	for e := range core.NewQuery(world).With(OwnershipRecord{}).Execute() {
		if rec, ok := e.Component[OwnershipRecord](); ok && rec.Instance == id {
			n++
		}
	}
	return n
}

func drainOnce(t *testing.T, d *Director, ctx *core.Context) error {
	t.Helper()
	return NewFrameSystem(d).Update(ctx)
}

// enterTestInstance queues a load and drains it, returning the live ID.
func enterTestInstance(t *testing.T, d *Director, ctx *core.Context, bp *Blueprint) InstanceID {
	t.Helper()
	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load %s: %v", bp.Name, err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	return id
}

func nestedBlueprint(name string, subs ...Blueprint) *Blueprint {
	return &Blueprint{
		Name: name,
		EntityList: []EntitySpec{
			{Name: "root", Components: []any{position{}}, SubScenes: subs},
		},
	}
}

// --- actor: drain and enter ---

func TestDrainNoRequestsIsNoOp(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
}

func TestDrainLoadsInstance(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := testBlueprint("level")

	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if n := ownedCount(ctx.World, id); n != 1 {
		t.Fatalf("owned entities: got %d, want 1", n)
	}
	if rec := d.instances[id]; !rec.live {
		t.Fatal("instance not live after drain")
	}
}

func TestDrainSpawnsChildrenUnderSameOwner(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := testBlueprint("level")
	bp.EntityList[0].Children = []EntitySpec{{Name: "child", Components: []any{position{}}}}
	id := enterTestInstance(t, d, ctx, bp)
	if n := ownedCount(ctx.World, id); n != 2 {
		t.Fatalf("owned entities: got %d, want 2", n)
	}
}

func TestDrainEntersSubscenesAsChildInstances(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := nestedBlueprint("level", subBlueprint("tank"))
	id := enterTestInstance(t, d, ctx, bp)
	rec := d.instances[id]
	if len(rec.children) != 1 {
		t.Fatalf("children: got %d, want 1", len(rec.children))
	}
	child := d.instances[rec.children[0]]
	if !child.live || child.parent != id || child.name != "tank" {
		t.Fatalf("child record: got %+v", child)
	}
	if n := ownedCount(ctx.World, child.id); n != 1 {
		t.Fatalf("child owned entities: got %d, want 1", n)
	}
}

func TestDrainEntersSubscenesOnNestedEntitySpecs(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	var entered, exited []string
	makeSub := func(name string) Blueprint {
		bp := subBlueprint(name)
		bp.Hooks.Enter = func(_ *core.Context, self *Instance) error {
			entered = append(entered, self.Name)
			return nil
		}
		bp.Hooks.Exit = func(_ *core.Context, self *Instance) error {
			exited = append(exited, self.Name)
			return nil
		}
		return bp
	}
	bp := &Blueprint{
		Name: "level",
		Hooks: Hooks{Enter: func(ctx *core.Context, self *Instance) error {
			entered = append(entered, self.Name)
			if got := ownedCount(ctx.World, self.ID); got != 4 {
				t.Errorf("parent entities at Enter: got %d, want 4", got)
			}
			return nil
		}},
		EntityList: []EntitySpec{
			{
				Name:      "root",
				SubScenes: []Blueprint{makeSub("root scene")},
				Children: []EntitySpec{{
					Name:      "child",
					SubScenes: []Blueprint{makeSub("child scene")},
					Children:  []EntitySpec{{Name: "grandchild", SubScenes: []Blueprint{makeSub("grandchild scene")}}},
				}},
			},
			{Name: "other root", SubScenes: []Blueprint{makeSub("other scene")}},
		},
	}
	id := enterTestInstance(t, d, ctx, bp)
	if got, want := fmt.Sprint(entered), "[level root scene child scene grandchild scene other scene]"; got != want {
		t.Fatalf("enter order = %s, want %s", got, want)
	}
	children := append([]InstanceID(nil), d.instances[id].children...)
	if len(children) != 4 {
		t.Fatalf("scene children = %v, want 4", children)
	}
	for _, childID := range children {
		child := d.instances[childID]
		if child == nil || !child.live || child.parent != id || ownedCount(ctx.World, childID) != 1 {
			t.Fatalf("subscene %d not live with its own entity: %+v", childID, child)
		}
	}
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain remove: %v", err)
	}
	if got, want := fmt.Sprint(exited), "[other scene grandchild scene child scene root scene]"; got != want {
		t.Fatalf("exit order = %s, want %s", got, want)
	}
	for _, instanceID := range append(children, id) {
		if _, ok := d.Get(instanceID); ok || ownedCount(ctx.World, instanceID) != 0 {
			t.Errorf("instance %d survived removal", instanceID)
		}
	}
}

func TestDrainRollsBackFailureInNestedEntitySubscene(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	var exited []string
	good := subBlueprint("good")
	good.Hooks.Exit = func(_ *core.Context, self *Instance) error {
		exited = append(exited, self.Name)
		return nil
	}
	bad := subBlueprint("bad")
	bad.Hooks.Enter = func(*core.Context, *Instance) error { return errors.New("entry failed") }
	bp := nestedBlueprint("level", good)
	bp.EntityList[0].Children = []EntitySpec{{
		Name:     "child",
		Children: []EntitySpec{{Name: "grandchild", SubScenes: []Blueprint{bad}}},
	}}
	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := drainOnce(t, d, ctx); err == nil || !strings.Contains(err.Error(), "entry failed") {
		t.Fatalf("drain error = %v, want entry failure", err)
	}
	if got := fmt.Sprint(exited); got != "[good]" {
		t.Fatalf("rollback exits = %s, want [good]", got)
	}
	if len(d.instances) != 0 || ownedCount(ctx.World, id) != 0 {
		t.Fatalf("scene records or parent entities survived rollback: records=%d", len(d.instances))
	}
	for range core.NewQuery(ctx.World).With(OwnershipRecord{}).Execute() {
		t.Fatal("scene-owned entity survived rollback")
	}
}

func TestDrainEnterHookOrderParentBeforeChild(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	var order []string
	parent := nestedBlueprint("level", subBlueprint("tank"))
	parent.Hooks = Hooks{Enter: func(ctx *core.Context, self *Instance) error {
		order = append(order, self.Name)
		if n := ownedCount(ctx.World, self.ID); n != 1 {
			t.Errorf("parent enter: owned entities = %d, want 1 (children not spawned yet)", n)
		}
		return nil
	}}
	child := subBlueprint("tank")
	child.Hooks = Hooks{Enter: func(ctx *core.Context, self *Instance) error {
		order = append(order, self.Name)
		if n := ownedCount(ctx.World, self.ID); n != 1 {
			t.Errorf("child enter: owned entities = %d, want 1", n)
		}
		return nil
	}}
	parent.EntityList[0].SubScenes[0] = child
	enterTestInstance(t, d, ctx, parent)
	if fmt.Sprint(order) != "[level tank]" {
		t.Fatalf("hook order: got %v", order)
	}
}

func TestDrainRollsBackOnEnterHookFailure(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := testBlueprint("level")
	bp.Hooks = Hooks{Enter: func(ctx *core.Context, self *Instance) error {
		return errors.New("no entry")
	}}
	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := drainOnce(t, d, ctx); err == nil {
		t.Fatal("drain accepted a failing enter hook")
	}
	if n := ownedCount(ctx.World, id); n != 0 {
		t.Fatalf("owned entities after rollback: got %d, want 0", n)
	}
	if _, ok := d.Get(id); ok {
		t.Fatal("record survived rollback")
	}
}

func TestDrainRollsBackSubsceneFailureExitingEnteredChildren(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bad := subBlueprint("boss")
	bad.Hooks.Enter = func(*core.Context, *Instance) error { return errors.New("entry failed") }
	bp := nestedBlueprint("level", subBlueprint("tank"), bad)
	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := drainOnce(t, d, ctx); err == nil {
		t.Fatal("drain accepted a failing subscene")
	}
	if n := ownedCount(ctx.World, id); n != 0 {
		t.Fatalf("parent entities after rollback: got %d, want 0", n)
	}
	if _, ok := d.Get(id); ok {
		t.Fatal("parent record survived rollback")
	}
	// The tank subscene fully entered before its sibling failed; rollback
	// must exit it: no records and no tagged entities anywhere.
	if len(d.instances) != 0 {
		t.Fatalf("records after rollback: %d remain", len(d.instances))
	}
	remaining := 0
	for e := range core.NewQuery(ctx.World).With(OwnershipRecord{}).Execute() {
		_ = e
		remaining++
	}
	if remaining != 0 {
		t.Fatalf("tagged entities after rollback: %d remain", remaining)
	}
}

func TestDrainReportsSubsceneRollbackErrors(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	entryErr := errors.New("entry failed")
	firstExitErr := errors.New("first exit failed")
	secondExitErr := errors.New("second exit failed")
	var exits []string
	first := subBlueprint("first")
	first.Hooks.Exit = func(_ *core.Context, self *Instance) error {
		exits = append(exits, self.Name)
		return firstExitErr
	}
	second := subBlueprint("second")
	second.Hooks.Exit = func(_ *core.Context, self *Instance) error {
		exits = append(exits, self.Name)
		return secondExitErr
	}
	bad := subBlueprint("bad")
	bad.Hooks.Enter = func(*core.Context, *Instance) error { return entryErr }
	bp := nestedBlueprint("root", first, second, bad)

	id, err := d.Load(bp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	err = drainOnce(t, d, ctx)
	for _, want := range []error{entryErr, firstExitErr, secondExitErr} {
		if !errors.Is(err, want) {
			t.Errorf("drain error = %v, missing %v", err, want)
		}
	}
	if got := fmt.Sprint(exits); got != "[second first]" {
		t.Errorf("rollback exits = %s, want [second first]", got)
	}
	if len(d.instances) != 0 || ownedCount(ctx.World, id) != 0 {
		t.Errorf("rollback left records or parent entities: records=%d", len(d.instances))
	}
	for range core.NewQuery(ctx.World).With(OwnershipRecord{}).Execute() {
		t.Fatal("scene-owned entity survived rollback")
	}
}

func TestDrainSpawnFailurePreservesOtherInstances(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}

	active := testBlueprint("active")

	activeID := enterTestInstance(t, d, ctx, active)

	failed := testBlueprint("failed")

	failed.EntityList[0].Components = []any{failAfterBlueprintValidation{calls: new(int)}}
	failedID, err := d.Load(failed)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := drainOnce(t, d, ctx); err == nil {
		t.Fatal("drain accepted an invalid component")
	}

	if n := ownedCount(ctx.World, activeID); n != 1 {
		t.Fatalf("active scene entities after failed spawn: got %d, want 1", n)
	}
	if _, ok := d.Get(failedID); ok {
		t.Fatal("failed instance record survived rollback")
	}
}

func TestDrainConsumesFailedRequestAndPreservesTail(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bad := testBlueprint("bad")
	bad.Hooks.Enter = func(*core.Context, *Instance) error { return errors.New("entry failed") }
	badID, err := d.Load(bad)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	goodID := loadTestInstance(t, d, "good")
	if err := drainOnce(t, d, ctx); err == nil {
		t.Fatal("drain accepted a failing request")
	}
	if len(d.pending) != 1 {
		t.Fatalf("pending requests: got %d, want 1 (unattempted tail only)", len(d.pending))
	}
	if got := d.pending[0]; got.kind != requestLoad || got.id != goodID {
		t.Fatalf("pending request = %+v, want the unattempted load for instance %d", got, goodID)
	}
	if _, ok := d.Get(badID); ok {
		t.Fatalf("failed instance %d remains reserved", badID)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain preserved tail: %v", err)
	}
	if inst, ok := d.Get(goodID); !ok || inst.Name != "good" {
		t.Fatalf("unattempted request was not applied: instance = %+v, found = %t", inst, ok)
	}
}

func TestDrainFlushesEdgesWhenApplied(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	am, err := input.New(input.Bindings{
		input.Action("jump"): []input.Input{input.KeyInput{Key: input.KeyA}},
	}, 0)
	if err != nil {
		t.Fatalf("input.New: %v", err)
	}
	snap := &input.Snapshot{}
	snap.Keys[input.KeyA] = input.State{Pressed: true, Held: true}
	am.Update(snap, 0)
	ctx.Actions = am
	if !am.JustPressed(input.Action("jump")) {
		t.Fatal("precondition: edge not latched")
	}
	enterTestInstance(t, d, ctx, testBlueprint("level"))
	if am.JustPressed(input.Action("jump")) {
		t.Fatal("edge survived a scene transition")
	}
}

func TestApplyUnloadPendingTargetDropsRecordAndRequests(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	id, err := d.Load(testBlueprint("menu"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	req := request{kind: requestUnload, id: id}
	if err := apply(d, ctx, req); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := d.Get(id); ok {
		t.Fatal("pending record survived unload")
	}
	if len(d.pending) != 0 {
		t.Fatalf("queued requests: got %d, want 0", len(d.pending))
	}
}

func TestApplyUpdateRejectsPendingTarget(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	id, err := d.Load(testBlueprint("menu"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	req := request{kind: requestUpdate, id: id, blueprint: testBlueprint("elite")}
	if err := apply(d, ctx, req); err == nil {
		t.Fatal("apply updated a pending instance")
	}
}

// --- actor: exit and rebuild (red until exit/rebuild are implemented) ---

func TestUnloadDestroysOwnedEntities(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := testBlueprint("level")

	id := enterTestInstance(t, d, ctx, bp)
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if n := ownedCount(ctx.World, id); n != 0 {
		t.Fatalf("owned entities after unload: got %d, want 0", n)
	}
	if _, ok := d.Get(id); ok {
		t.Fatal("record survived unload")
	}
}

func TestUnloadRunsExitHookWithView(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	var seen *Instance
	bp := testBlueprint("arena")
	bp.Hooks = Hooks{Exit: func(ctx *core.Context, self *Instance) error {
		seen = &Instance{ID: self.ID, Name: self.Name}
		return nil
	}}
	id := enterTestInstance(t, d, ctx, bp)
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if seen == nil {
		t.Fatal("exit hook never ran")
	}
	if seen.ID != id || seen.Name != "arena" {
		t.Fatalf("exit view: got %+v", seen)
	}
}

func TestUnloadCommitsAfterExitHookErrors(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	childErr := errors.New("child exit failed")
	parentErr := errors.New("parent exit failed")
	var order []string
	child := subBlueprint("child")

	child.Hooks.Exit = func(*core.Context, *Instance) error {
		order = append(order, "child")
		return childErr
	}
	parent := nestedBlueprint("parent", child)

	parent.Hooks.Exit = func(*core.Context, *Instance) error {
		order = append(order, "parent")
		return parentErr
	}
	id := enterTestInstance(t, d, ctx, parent)
	childID := d.instances[id].children[0]
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	err := drainOnce(t, d, ctx)
	if err == nil {
		t.Fatal("drain ignored exit hook errors")
	}
	if !errors.Is(err, childErr) || !errors.Is(err, parentErr) {
		t.Fatalf("drain error = %v, want both exit hook errors", err)
	}
	if !strings.Contains(err.Error(), "teardown committed") {
		t.Fatalf("drain error does not report committed teardown: %v", err)
	}
	if fmt.Sprint(order) != "[child parent]" {
		t.Fatalf("exit order: got %v, want [child parent]", order)
	}
	for _, instanceID := range []InstanceID{id, childID} {
		if _, ok := d.Get(instanceID); ok {
			t.Errorf("instance %d survived teardown", instanceID)
		}
		if n := ownedCount(ctx.World, instanceID); n != 0 {
			t.Errorf("instance %d entities after teardown: got %d, want 0", instanceID, n)
		}
	}
	if len(d.pending) != 0 {
		t.Fatalf("failed remove request was retried: pending=%v", d.pending)
	}
}

func TestUnloadKeepsSiblingInstances(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	first := enterTestInstance(t, d, ctx, testBlueprint("menu"))
	second := enterTestInstance(t, d, ctx, testBlueprint("hud"))
	if err := d.Remove(first); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if n := ownedCount(ctx.World, second); n != 1 {
		t.Fatalf("sibling entities: got %d, want 1", n)
	}
	if rec := d.instances[second]; !rec.live {
		t.Fatal("sibling no longer live")
	}
}

func TestUnloadRunsChildExitsBeforeParent(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	var order []string
	child := subBlueprint("tank")
	child.Hooks = Hooks{Exit: func(ctx *core.Context, self *Instance) error {
		order = append(order, self.Name)
		return nil
	}}
	parent := nestedBlueprint("level", child)
	parent.Hooks = Hooks{Exit: func(ctx *core.Context, self *Instance) error {
		order = append(order, self.Name)
		return nil
	}}
	id := enterTestInstance(t, d, ctx, parent)
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if fmt.Sprint(order) != "[tank level]" {
		t.Fatalf("exit order: got %v", order)
	}
}

func TestParentExitHookSeesChildEntities(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	child := subBlueprint("tank")
	parent := nestedBlueprint("level", child)
	parent.Hooks = Hooks{Exit: func(ctx *core.Context, self *Instance) error {
		childIDs := d.instances[self.ID].children
		for _, cid := range childIDs {
			if n := ownedCount(ctx.World, cid); n != 1 {
				t.Errorf("child %d entities at parent exit: got %d, want 1", cid, n)
			}
		}
		return nil
	}}
	id := enterTestInstance(t, d, ctx, parent)
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
}

func TestSwitchTearsDownRootsThenEnters(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	old := enterTestInstance(t, d, ctx, testBlueprint("menu"))
	if _, err := d.Switch(testBlueprint("level")); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if n := ownedCount(ctx.World, old); n != 0 {
		t.Fatalf("old root entities: got %d, want 0", n)
	}
	if view, ok := d.ByName("level"); !ok || !d.instances[view.ID].live {
		t.Fatal("new root not live after switch")
	}
}

func TestSwitchEntersReplacementAfterExitHookError(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	exitErr := errors.New("exit failed")
	oldBlueprint := testBlueprint("menu")
	oldBlueprint.Hooks.Exit = func(*core.Context, *Instance) error {
		return exitErr
	}
	oldID := enterTestInstance(t, d, ctx, oldBlueprint)
	newBlueprint := testBlueprint("level")
	newID, err := d.Switch(newBlueprint)
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}
	err = drainOnce(t, d, ctx)
	if err == nil || !errors.Is(err, exitErr) {
		t.Fatalf("drain error = %v, want exit hook error", err)
	}
	if !strings.Contains(err.Error(), "teardown committed") {
		t.Fatalf("drain error does not report committed teardown: %v", err)
	}
	if _, ok := d.Get(oldID); ok || ownedCount(ctx.World, oldID) != 0 {
		t.Fatal("old root survived switch teardown")
	}
	if inst, ok := d.Get(newID); !ok || !d.instances[newID].live || inst.Name != "level" {
		t.Fatalf("replacement instance = %+v, found = %t; want live level", inst, ok)
	}
}

func TestUpdateRebuildsPreservingIdentity(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	bp := &Blueprint{
		Name:       "arena",
		EntityList: []EntitySpec{{Name: "grunt", Components: []any{position{X: 1}}}},
	}
	id := enterTestInstance(t, d, ctx, bp)
	elite := &Blueprint{
		Name:       "elite",
		EntityList: []EntitySpec{{Name: "elite", Components: []any{position{X: 2}}}},
	}
	if err := d.Update(id, elite); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	rec := d.instances[id]
	if !rec.live || rec.name != "arena" || rec.blueprint.Name != "elite" {
		t.Fatalf("record after rebuild: %+v", rec)
	}
	var x float64
	for e := range core.NewQuery(ctx.World).With(OwnershipRecord{}, position{}).Execute() {
		if p, ok := e.Component[position](); ok {
			x = p.X
		}
	}
	if x != 2 {
		t.Fatalf("entity after rebuild: got position %v, want 2", x)
	}
}

func TestUpdateReplacesSubsceneTree(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	oldChild := subBlueprint("old child")

	old := nestedBlueprint("root", oldChild)

	id := enterTestInstance(t, d, ctx, old)
	oldChildID := d.instances[id].children[0]

	newChild := subBlueprint("new child")

	replacement := nestedBlueprint("replacement", newChild)

	if err := d.Update(id, replacement); err != nil {
		t.Fatalf("Update with subscene: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain update: %v", err)
	}
	if _, ok := d.Get(oldChildID); ok || ownedCount(ctx.World, oldChildID) != 0 {
		t.Fatal("old subscene survived rebuild")
	}
	children := d.instances[id].children
	if len(children) != 1 || children[0] == oldChildID {
		t.Fatalf("children after rebuild = %v, want only replacement subscene", children)
	}
	newChildID := children[0]

	if err := d.Update(id, testBlueprint("no subscenes")); err != nil {
		t.Fatalf("Update without subscenes: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain second update: %v", err)
	}
	if len(d.instances[id].children) != 0 {
		t.Fatalf("children after second rebuild = %v, want none", d.instances[id].children)
	}
	if _, ok := d.Get(newChildID); ok || ownedCount(ctx.World, newChildID) != 0 {
		t.Fatal("replacement subscene survived second rebuild")
	}
	if err := d.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain remove: %v", err)
	}
	if _, ok := d.Get(id); ok || ownedCount(ctx.World, id) != 0 {
		t.Fatal("root survived removal")
	}
}

func TestFailedSubsceneUpdateUnlinksFromParent(t *testing.T) {
	cases := []struct {
		name        string
		replacement func() *Blueprint
	}{
		{"spawn failure", func() *Blueprint {
			bp := testBlueprint("bad")
			bp.EntityList[0].Components = []any{failAfterBlueprintValidation{calls: new(int)}}
			return bp
		}},
		{"enter hook failure", func() *Blueprint {
			bp := testBlueprint("bad")
			bp.Hooks.Enter = func(*core.Context, *Instance) error {
				return errors.New("enter failed")
			}
			return bp
		}},
		{"nested enter failure", func() *Blueprint {
			child := subBlueprint("bad nested")
			child.Hooks.Enter = func(*core.Context, *Instance) error { return errors.New("enter failed") }
			return nestedBlueprint("bad", child)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDirector()

			ctx := &core.Context{World: core.NewWorld()}
			failingChild := subBlueprint("target")

			sibling := subBlueprint("sibling")

			parentID := enterTestInstance(t, d, ctx, nestedBlueprint("root", failingChild, sibling))
			childID := d.instances[parentID].children[0]
			siblingID := d.instances[parentID].children[1]

			if err := d.Update(childID, tc.replacement()); err != nil {
				t.Fatalf("Update: %v", err)
			}
			if err := drainOnce(t, d, ctx); err == nil {
				t.Fatal("drain accepted failing replacement")
			}
			if _, ok := d.Get(childID); ok || ownedCount(ctx.World, childID) != 0 {
				t.Fatal("failed subscene survived")
			}
			children := d.instances[parentID].children
			if len(children) != 1 || children[0] != siblingID {
				t.Fatalf("parent children = %v, want sibling %d", children, siblingID)
			}
			if err := d.Remove(parentID); err != nil {
				t.Fatalf("Remove parent: %v", err)
			}
			if err := drainOnce(t, d, ctx); err != nil {
				t.Fatalf("drain remove: %v", err)
			}
			if _, ok := d.Get(siblingID); ok || ownedCount(ctx.World, siblingID) != 0 {
				t.Fatal("sibling survived parent removal")
			}
		})
	}
}

func TestUpdateSnapshotsBlueprintStructure(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	id := enterTestInstance(t, d, ctx, testBlueprint("arena"))
	replacement := testBlueprint("elite")
	replacement.EntityList[0].Components = []any{position{X: 1}}
	if err := d.Update(id, replacement); err != nil {
		t.Fatalf("Update: %v", err)
	}
	replacement.Name = "changed"
	replacement.EntityList[0].Name = "changed entity"
	replacement.EntityList[0].Components[0] = position{X: 9}

	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	inst, ok := d.Get(id)
	if !ok || inst.Blueprint.Name != "elite" ||
		inst.Blueprint.EntityList[0].Name != "root" ||
		inst.Blueprint.EntityList[0].Components[0] != (position{X: 1}) {
		t.Fatalf("updated blueprint was not snapshotted: %+v, found = %t", inst, ok)
	}
}

func TestUpdateRunsExitThenEnter(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	var order []string
	bp := testBlueprint("arena")
	bp.Hooks = Hooks{
		Enter: func(ctx *core.Context, self *Instance) error {
			order = append(order, "enter:"+self.Name)
			return nil
		},
		Exit: func(ctx *core.Context, self *Instance) error {
			order = append(order, "exit:"+self.Name)
			return nil
		},
	}
	id := enterTestInstance(t, d, ctx, bp)
	elite := testBlueprint("elite")
	elite.Hooks.Enter = func(ctx *core.Context, self *Instance) error {
		order = append(order, "enter:"+self.Name)
		return nil
	}
	if err := d.Update(id, elite); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := drainOnce(t, d, ctx); err != nil {
		t.Fatalf("drain: %v", err)
	}
	want := "[enter:arena exit:arena enter:arena]"
	if fmt.Sprint(order) != want {
		t.Fatalf("hook order: got %v, want %v", order, want)
	}
}

func TestUpdateRebuildsAfterExitHookError(t *testing.T) {
	d := NewDirector()
	ctx := &core.Context{World: core.NewWorld()}
	exitErr := errors.New("exit failed")
	old := testBlueprint("arena")
	old.Hooks.Exit = func(*core.Context, *Instance) error {
		return exitErr
	}
	id := enterTestInstance(t, d, ctx, old)
	replacement := testBlueprint("elite")
	entered := false
	replacement.Hooks.Enter = func(*core.Context, *Instance) error {
		entered = true
		return nil
	}
	if err := d.Update(id, replacement); err != nil {
		t.Fatalf("Update: %v", err)
	}
	err := drainOnce(t, d, ctx)
	if err == nil || !errors.Is(err, exitErr) {
		t.Fatalf("drain error = %v, want exit hook error", err)
	}
	if !strings.Contains(err.Error(), "teardown committed") {
		t.Fatalf("drain error does not report committed teardown: %v", err)
	}
	if !entered {
		t.Fatal("replacement enter hook did not run")
	}
	inst, ok := d.Get(id)
	if !ok || !d.instances[id].live || inst.Blueprint.Name != replacement.Name {
		t.Fatalf("updated instance = %+v, found = %t; want live replacement blueprint", inst, ok)
	}
	if got := ownedCount(ctx.World, id); got != 1 {
		t.Fatalf("replacement entities: got %d, want 1", got)
	}
}
