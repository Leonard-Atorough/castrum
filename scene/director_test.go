package scene

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func describeRequest(r request) string {
	kind := map[requestKind]string{
		requestSwitch: "switch",
		requestLoad:   "load",
		requestUpdate: "update",
		requestUnload: "unload",
	}[r.kind]
	return fmt.Sprintf("%s id=%d key=%q name=%q(%v) hooksSet=%v",
		kind, r.id, r.tree.Key, r.name, r.nameSet, r.hooksSet)
}

func TestQueueRejectsBadInput(t *testing.T) {
	d := NewDirector()
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{"nil scene switch", func() error { _, err := d.Switch(nil, nil); return err }, `scene: switch "": nil scene`},
		{"nil scene load", func() error { _, err := d.Load(nil, nil); return err }, `scene: load ""`},
		{"nil scene update", func() error { return d.Update(7, nil, nil) }, `scene: update ""`},
		{"invalid tree rejected", func() error {
			_, err := d.Switch(lifeScene(), nil) // props type mismatch
			return err
		}, "want scene.lifeProps"},
		{"zero id update", func() error { return d.Update(0, lifeScene(), lifeProps{"a"}) }, "invalid instance id 0"},
		{"zero id remove", func() error { return d.Remove(0) }, "invalid instance id 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want %q, got %v", tt.want, err)
			}
		})
	}
	if len(d.pending) != 0 {
		t.Fatalf("rejected requests queued: %d", len(d.pending))
	}
}

func TestCoalesceRules(t *testing.T) {
	tests := []struct {
		name  string
		calls func(d *Director)
		want  []string
	}{
		{
			name: "later switch replaces earlier switch",
			calls: func(d *Director) {
				_, _ = d.Switch(lifeScene(), lifeProps{"a"})
				_, _ = d.Switch(lifeScene(), lifeProps{"b"}, WithName("two"))
			},
			want: []string{`switch id=2 key="b" name="two"(true) hooksSet=false`},
		},
		{
			name: "update folds into pending load",
			calls: func(d *Director) {
				id, _ := d.Load(lifeScene(), lifeProps{"l1"})
				if err := d.Update(id, lifeScene(), lifeProps{"l2"}, WithName("ln")); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{`load id=1 key="l2" name="ln"(true) hooksSet=false`},
		},
		{
			name: "update folds into pending switch",
			calls: func(d *Director) {
				id, _ := d.Switch(lifeScene(), lifeProps{"s1"})
				if err := d.Update(id, lifeScene(), lifeProps{"s2"}, WithHooks(Hooks{})); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{`switch id=1 key="s2" name=""(false) hooksSet=true`},
		},
		{
			name: "remove cancels a pending load",
			calls: func(d *Director) {
				id, _ := d.Load(lifeScene(), lifeProps{"l1"})
				if err := d.Remove(id); err != nil {
					t.Fatal(err)
				}
			},
			want: nil,
		},
		{
			name: "update replaces update and keeps merged flags",
			calls: func(d *Director) {
				if err := d.Update(5, lifeScene(), lifeProps{"u1"}, WithName("n1")); err != nil {
					t.Fatal(err)
				}
				if err := d.Update(5, lifeScene(), lifeProps{"u2"}); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{`update id=5 key="u2" name="n1"(true) hooksSet=false`},
		},
		{
			name: "remove replaces update, then collapses",
			calls: func(d *Director) {
				if err := d.Update(5, lifeScene(), lifeProps{"u1"}, WithName("n1")); err != nil {
					t.Fatal(err)
				}
				if err := d.Remove(5); err != nil {
					t.Fatal(err)
				}
				if err := d.Remove(5); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{`unload id=5 key="" name="n1"(true) hooksSet=false`},
		},
		{
			name: "update revokes a pending remove",
			calls: func(d *Director) {
				if err := d.Remove(5); err != nil {
					t.Fatal(err)
				}
				if err := d.Update(5, lifeScene(), lifeProps{"back"}); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{`update id=5 key="back" name=""(false) hooksSet=false`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDirector()
			tt.calls(d)
			var got []string
			for _, r := range d.pending {
				got = append(got, describeRequest(r))
			}
			if strings.Join(got, "; ") != strings.Join(tt.want, "; ") {
				t.Fatalf("pending =\n  %s\nwant\n  %s", strings.Join(got, ";\n  "), strings.Join(tt.want, ";\n  "))
			}
		})
	}
}

func TestQueuedTreeIsSnapshotted(t *testing.T) {
	d := NewDirector()
	shared := []Node{{Key: "kid"}}
	s := Of(func(p lifeProps) Node {
		return Node{Key: p.Key, Children: shared}
	})
	if _, err := d.Switch(s, lifeProps{"root"}); err != nil {
		t.Fatal(err)
	}
	shared = append(shared, Node{Key: "sneaky"})

	if kids := d.pending[0].tree.Children; len(kids) != 1 || kids[0].Key != "kid" {
		t.Fatalf("queued tree mutated after queue: %+v", kids)
	}
}

func TestDirectorAccessors(t *testing.T) {
	d, sys, ctx := newTestSystem(t)

	if d.Live(1) {
		t.Error("unknown id reads live")
	}
	if _, ok := d.Name(1); ok {
		t.Error("unknown id reports name")
	}
	if d.Entities(1) != nil {
		t.Error("unknown id reports entities")
	}

	id, _ := d.Load(lifeScene(), lifeProps{"a"}, WithName("home"))
	if d.Live(id) {
		t.Error("pending instance reads live")
	}
	flush(t, sys, ctx)
	if !d.Live(id) {
		t.Fatal("live instance not reported live")
	}
	if name, ok := d.Name(id); !ok || name != "home" {
		t.Fatalf("Name: %q %v", name, ok)
	}
	ents := d.Entities(id)
	if len(ents) != 2 {
		t.Fatalf("Entities: %d", len(ents))
	}
	ents[0] = core.Entity{}
	if d.Entities(id)[0] == (core.Entity{}) {
		t.Error("Entities shares backing array")
	}

	second, _ := d.Load(lifeScene(), lifeProps{"b"})
	flush(t, sys, ctx)
	if roots := d.Roots(); len(roots) != 2 || roots[0] != id || roots[1] != second {
		t.Fatalf("Roots = %v, want [%d %d]", roots, id, second)
	}
	if err := d.Remove(second); err != nil {
		t.Fatal(err)
	}
	flush(t, sys, ctx)
	if roots := d.Roots(); len(roots) != 1 || roots[0] != id {
		t.Fatalf("Roots after remove = %v, want [%d]", roots, id)
	}
}
