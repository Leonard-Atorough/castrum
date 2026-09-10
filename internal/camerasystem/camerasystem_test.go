package camerasystem

import (
	"math"
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

const epsilonRect = 1e-9

func vecAlmostEqual(a, b geom.Vector2) bool {
	const epsilon = 1e-9
	return math.Abs(a.X-b.X) < epsilon && math.Abs(a.Y-b.Y) < epsilon
}

func TestSystem_Update_ClampsCamera(t *testing.T) {
	sys := &System{}
	world := ecs.NewWorld()

	err := sys.Init(world)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	// Create a camera with bounds
	cam := components.NewCamera(800, 600)
	cam.Bounds = geom.Rect{
		Min: geom.Vector2{X: 0, Y: 0},
		Max: geom.Vector2{X: 1000, Y: 1000},
	}
	// Position it outside bounds
	cam.Position = geom.Vector2{X: -500, Y: 2000}
	cam.Primary = true

	// Create entity with camera
	eid, err := world.CreateWithComponents("camera", cam)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Update the system
	err = sys.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Retrieve camera and check it was clamped
	updated, err := world.GetComponent[components.Camera](eid.ID)
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}

	viewport := updated.ViewportBounds()
	if viewport.Min.X < updated.Bounds.Min.X-epsilonRect || viewport.Max.X > updated.Bounds.Max.X+epsilonRect ||
		viewport.Min.Y < updated.Bounds.Min.Y-epsilonRect || viewport.Max.Y > updated.Bounds.Max.Y+epsilonRect {
		t.Fatalf("viewport %v escapes bounds %v after system update", viewport, updated.Bounds)
	}
}

func TestSystem_Update_MultipleCamera(t *testing.T) {
	sys := &System{}
	world := ecs.NewWorld()

	err := sys.Init(world)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	// Create first camera with bounds
	cam1 := components.NewCamera(800, 600)
	cam1.Bounds = geom.Rect{
		Min: geom.Vector2{X: 0, Y: 0},
		Max: geom.Vector2{X: 500, Y: 500},
	}
	cam1.Position = geom.Vector2{X: -200, Y: -200}
	cam1.Primary = true

	eid1, err := world.CreateWithComponents("camera1", cam1)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Create second camera with different bounds
	cam2 := components.NewCamera(800, 600)
	cam2.Bounds = geom.Rect{
		Min: geom.Vector2{X: 1000, Y: 1000},
		Max: geom.Vector2{X: 2000, Y: 2000},
	}
	cam2.Position = geom.Vector2{X: 500, Y: 500}
	cam2.Primary = false

	eid2, err := world.CreateWithComponents("camera2", cam2)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	// Update system - should clamp both cameras
	err = sys.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Check first camera was clamped
	cam1Updated, _ := world.GetComponent[components.Camera](eid1.ID)
	vp1 := cam1Updated.ViewportBounds()
	if vp1.Min.X < cam1Updated.Bounds.Min.X-epsilonRect {
		t.Fatalf("camera1 viewport min escapes bounds after update")
	}

	// Check second camera was clamped
	cam2Updated, _ := world.GetComponent[components.Camera](eid2.ID)
	vp2 := cam2Updated.ViewportBounds()
	if vp2.Min.X < cam2Updated.Bounds.Min.X-epsilonRect {
		t.Fatalf("camera2 viewport min escapes bounds after update")
	}
}

func TestSystem_Update_UnboundedCameraNotAffected(t *testing.T) {
	sys := &System{}
	world := ecs.NewWorld()

	err := sys.Init(world)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	// Create unbounded camera (no explicit bounds = infinite)
	cam := components.NewCamera(800, 600)
	cam.Position = geom.Vector2{X: 1e6, Y: 1e6}
	cam.Primary = true

	eid, err := world.CreateWithComponents("camera", cam)
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	originalPos := cam.Position

	// Update system
	err = sys.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Check position was not modified
	updated, _ := world.GetComponent[components.Camera](eid.ID)

	if updated.Position != originalPos {
		t.Fatalf("unbounded camera position changed from %v to %v", originalPos, updated.Position)
	}
}
