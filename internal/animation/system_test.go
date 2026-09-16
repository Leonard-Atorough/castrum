package animation

import (
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
)

func setupTestWorld(t *testing.T) (*ecs.World, *ClipStore, *events.EventBus) {
	t.Helper()
	world := ecs.NewWorld()
	store := NewClipStore()
	bus := events.NewEventBus()
	world.SetResource(store)
	world.SetResource(bus)
	return world, store, bus
}

func buildTestClip(t *testing.T, store *ClipStore, id string, frames []string, fps float64, loop Loop) {
	t.Helper()
	a := testAtlas()
	b := store.NewBuilder(id, a)
	for _, f := range frames {
		b.AddFrame(f)
	}
	if _, err := b.SetFPS(fps).SetLoop(loop).Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
}

func spawnAnimatingEntity(t *testing.T, world *ecs.World, clipID string) ecs.EntityID {
	t.Helper()
	e, err := world.CreateWithComponents("test",
		components.NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 0, Y: 0}),
		components.Sprite{Visible: true, Opacity: 1},
		components.NewAnimation(clipID, true),
	)
	if err != nil {
		t.Fatalf("CreateWithComponents() error = %v", err)
	}
	return e.ID
}

// ---------------------------------------------------------------------------
// System.Init
// ---------------------------------------------------------------------------

func TestSystemInitResolvesResources(t *testing.T) {
	world, store, bus := setupTestWorld(t)
	sys := NewSystem()
	if err := sys.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if sys.store != store {
		t.Error("Init() did not resolve clip store from world resource")
	}
	if sys.eventBus != bus {
		t.Error("Init() did not resolve event bus from world resource")
	}
	if sys.query == nil {
		t.Error("Init() did not create query")
	}
}

func TestSystemInitFailsWithoutClipStore(t *testing.T) {
	world := ecs.NewWorld()
	world.SetResource(events.NewEventBus())
	sys := NewSystem()
	if err := sys.Init(world); err == nil {
		t.Fatal("Init() without ClipStore returned nil error")
	}
}

func TestSystemInitFailsWithoutEventBus(t *testing.T) {
	world := ecs.NewWorld()
	world.SetResource(NewClipStore())
	sys := NewSystem()
	if err := sys.Init(world); err == nil {
		t.Fatal("Init() without EventBus returned nil error")
	}
}

// ---------------------------------------------------------------------------
// System.Update — frame advancement
// ---------------------------------------------------------------------------

func TestUpdateAdvancesFrameTime(t *testing.T) {
	world, store, _ := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1"}, 10, LoopNone)
	entity := spawnAnimatingEntity(t, world, "walk")

	sys := NewSystem()
	sys.Init(world)

	if err := sys.Update(world, 0.05); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameTime < 0.04 || anim.FrameTime >= 0.1 {
		t.Errorf("FrameTime = %v, want in [0.04, 0.1)", anim.FrameTime)
	}
	if anim.FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 (threshold not met)", anim.FrameIndex)
	}
}

func TestUpdateAdvancesFrame(t *testing.T) {
	world, store, _ := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1"}, 10, LoopNone)
	entity := spawnAnimatingEntity(t, world, "walk")

	sys := NewSystem()
	sys.Init(world)

	// 0.15s at 10 FPS (0.1s/frame) crosses one frame boundary.
	if err := sys.Update(world, 0.15); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1", anim.FrameIndex)
	}
}

func TestUpdateMultiFrameAdvance(t *testing.T) {
	world, store, _ := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1", "frame_2"}, 10, LoopNone)
	entity := spawnAnimatingEntity(t, world, "walk")

	sys := NewSystem()
	sys.Init(world)

	// 0.35s at 10 FPS crosses three frame boundaries: index 0->1->2->3 (out of range).
	if err := sys.Update(world, 0.35); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	// LoopNone: should clamp to last frame and stop.
	if anim.FrameIndex != 2 {
		t.Errorf("FrameIndex = %d, want 2 (clamped to last)", anim.FrameIndex)
	}
	if anim.Playing {
		t.Error("Playing = true, want false (clip finished)")
	}
}

// ---------------------------------------------------------------------------
// System.Update — looping
// ---------------------------------------------------------------------------

func TestUpdateLoopForeverWrapsToStart(t *testing.T) {
	world, store, bus := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1"}, 10, LoopForever)
	entity := spawnAnimatingEntity(t, world, "walk")

	var loopedEvents []AnimationEvent
	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		if e.Type == AnimationEventLooped {
			loopedEvents = append(loopedEvents, e)
		}
	}, false)

	sys := NewSystem()
	sys.Init(world)

	// 0.25s at 10 FPS crosses two boundaries: 0->1->2 (wraps to 0).
	if err := sys.Update(world, 0.25); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 (wrapped)", anim.FrameIndex)
	}
	if !anim.Playing {
		t.Error("Playing = false, want true (looping clip continues)")
	}
	if len(loopedEvents) != 1 {
		t.Errorf("looped events = %d, want 1", len(loopedEvents))
	}
}

// ---------------------------------------------------------------------------
// System.Update — non-looping (LoopNone)
// ---------------------------------------------------------------------------

func TestUpdateLoopNoneStopsAtLastFrame(t *testing.T) {
	world, store, bus := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1"}, 10, LoopNone)
	entity := spawnAnimatingEntity(t, world, "walk")

	var finishedEvents []AnimationEvent
	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		if e.Type == AnimationEventFinished {
			finishedEvents = append(finishedEvents, e)
		}
	}, false)

	sys := NewSystem()
	sys.Init(world)

	// 0.25s at 10 FPS: 0->1->2 (out of range, LoopNone clamps to 1).
	if err := sys.Update(world, 0.25); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1 (last frame)", anim.FrameIndex)
	}
	if anim.Playing {
		t.Error("Playing = true, want false (clip finished)")
	}
	if len(finishedEvents) != 1 {
		t.Errorf("finished events = %d, want 1", len(finishedEvents))
	}
	if finishedEvents[0].EntityID != entity {
		t.Errorf("finished event EntityID = %d, want %d", finishedEvents[0].EntityID, entity)
	}
}

// ---------------------------------------------------------------------------
// System.Update — frame events
// ---------------------------------------------------------------------------

func TestUpdateEmitsFrameEvent(t *testing.T) {
	world, store, bus := setupTestWorld(t)
	a := testAtlas()
	b := store.NewBuilder("walk", a)
	b.AddFrame("frame_0").AddFrame("frame_1").SetFPS(10).OnFrame(1, "hit")
	if _, err := b.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	entity := spawnAnimatingEntity(t, world, "walk")

	var frameEvents []AnimationEvent
	bus.On(func(_ events.EventMeta, e AnimationEvent) {
		if e.Type == AnimationEventFrame {
			frameEvents = append(frameEvents, e)
		}
	}, false)

	sys := NewSystem()
	sys.Init(world)

	// 0.15s at 10 FPS: 0->1, frame 1 is tagged "hit".
	if err := sys.Update(world, 0.15); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if len(frameEvents) != 1 {
		t.Fatalf("frame events = %d, want 1", len(frameEvents))
	}
	if frameEvents[0].EventName != "hit" {
		t.Errorf("EventName = %q, want %q", frameEvents[0].EventName, "hit")
	}
	if frameEvents[0].FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1", frameEvents[0].FrameIndex)
	}
	if frameEvents[0].EntityID != entity {
		t.Errorf("EntityID = %d, want %d", frameEvents[0].EntityID, entity)
	}
}

// ---------------------------------------------------------------------------
// System.Update — edge cases
// ---------------------------------------------------------------------------

func TestUpdateIgnoresNonPlayingAnimation(t *testing.T) {
	world, store, _ := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1"}, 10, LoopNone)
	entity, _ := world.CreateWithComponents("test",
		components.NewTransform(geom.Vector2{}, 0, geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 0, Y: 0}),
		components.Sprite{Visible: true, Opacity: 1},
		components.Animation{ClipID: "walk", Playing: false, PlaybackSpeed: 1.0},
	)

	sys := NewSystem()
	sys.Init(world)

	if err := sys.Update(world, 1.0); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity.ID)
	if anim.FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 (not playing)", anim.FrameIndex)
	}
}

func TestUpdateSkipsMissingClip(t *testing.T) {
	world, _, _ := setupTestWorld(t)
	entity := spawnAnimatingEntity(t, world, "nonexistent")

	sys := NewSystem()
	sys.Init(world)

	if err := sys.Update(world, 0.5); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity)
	if anim.FrameIndex != 0 {
		t.Errorf("FrameIndex = %d, want 0 (clip not found)", anim.FrameIndex)
	}
}

func TestUpdateRespectsPlaybackSpeed(t *testing.T) {
	world, store, _ := setupTestWorld(t)
	buildTestClip(t, store, "walk", []string{"frame_0", "frame_1"}, 10, LoopNone)
	entity, _ := world.CreateWithComponents("test",
		components.NewTransform(geom.Vector2{}, 0, geom.Vector2{X: 1, Y: 1}, geom.Vector2{X: 0, Y: 0}),
		components.Sprite{Visible: true, Opacity: 1},
		components.Animation{ClipID: "walk", Playing: true, PlaybackSpeed: 2.0},
	)

	sys := NewSystem()
	sys.Init(world)

	// 0.1s * speed 2.0 = 0.2s effective, at 10 FPS = 2 frame advances.
	if err := sys.Update(world, 0.1); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	anim, _ := world.GetComponent[components.Animation](entity.ID)
	// 0->1->2 (out of range, LoopNone clamps to 1).
	if anim.FrameIndex != 1 {
		t.Errorf("FrameIndex = %d, want 1 (double speed)", anim.FrameIndex)
	}
	if anim.Playing {
		t.Error("Playing = true, want false (clip finished at 2x speed)")
	}
}

func TestUpdateUninitializedReturnsError(t *testing.T) {
	world, _, _ := setupTestWorld(t)
	sys := &System{} // not initialized via Init
	err := sys.Update(world, 0.1)
	if err == nil {
		t.Fatal("Update() on uninitialized system returned nil error")
	}
}

func TestSystemShutdown(t *testing.T) {
	world, _, _ := setupTestWorld(t)
	sys := NewSystem()
	sys.Init(world)
	if err := sys.Shutdown(world); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}
