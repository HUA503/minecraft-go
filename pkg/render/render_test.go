package render

import (
	"math"
	"testing"
)

func TestMeshFaceCullingIsolatedCube(t *testing.T) {
	world := map[[3]int32]int32{
		{0, 0, 0}: 1, // Stone
	}

	getBlock := func(x, y, z int32) int32 {
		return world[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)

	if len(mesh.Vertices) != 24 {
		t.Fatalf("expected 24 vertices for isolated cube, got %d", len(mesh.Vertices))
	}
	if len(mesh.Indices) != 36 {
		t.Fatalf("expected 36 indices for isolated cube, got %d", len(mesh.Indices))
	}
}

func TestMeshFaceCullingAdjacentCubes(t *testing.T) {
	world := map[[3]int32]int32{
		{0, 0, 0}: 1,
		{1, 0, 0}: 1,
	}

	getBlock := func(x, y, z int32) int32 {
		return world[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)

	if len(mesh.Vertices) != 40 {
		t.Fatalf("expected 40 vertices for two adjacent cubes with culled face, got %d", len(mesh.Vertices))
	}
	if len(mesh.Indices) != 60 {
		t.Fatalf("expected 60 indices, got %d", len(mesh.Indices))
	}
}

func TestCameraLookVectorAndProjection(t *testing.T) {
	cam := NewCamera(0, 0, 0)
	cam.Yaw = 0
	cam.Pitch = 0

	dx, dy, dz := cam.GetLookVector()
	if math.Abs(dx) > 1e-5 || math.Abs(dy) > 1e-5 || math.Abs(dz-1.0) > 1e-5 {
		t.Fatalf("expected look vector (0, 0, 1), got (%f, %f, %f)", dx, dy, dz)
	}

	// Point at (0, 0, 10) directly ahead should project to center of 640x480 screen
	p, ok := cam.ProjectToScreen(0, 0, 10, 640, 480)
	if !ok {
		t.Fatalf("point ahead should be visible")
	}
	if math.Abs(float64(p.X)-320) > 1.0 || math.Abs(float64(p.Y)-240) > 1.0 {
		t.Fatalf("expected projection at center (320, 240), got (%f, %f)", p.X, p.Y)
	}

	// Test Yaw: turning right (+10 deg) makes object ahead move to the LEFT on screen
	cam.Yaw = 10.0
	pRight, ok := cam.ProjectToScreen(0, 0, 10, 640, 480)
	if !ok || pRight.X >= 320 {
		t.Fatalf("turning right should make center object move LEFT (X < 320), got X=%f", pRight.X)
	}

	// Test Pitch: looking up (+10 deg) makes object ahead move DOWN on screen
	cam.Yaw = 0
	cam.Pitch = 10.0
	pUp, ok := cam.ProjectToScreen(0, 0, 10, 640, 480)
	if !ok || pUp.Y <= 240 {
		t.Fatalf("looking up should make center object move DOWN on screen (Y > 240), got Y=%f", pUp.Y)
	}
}

func TestHUDDrawing(t *testing.T) {
	fb := NewFramebuffer(640, 480)
	fb.Clear(0xFF000000)

	fb.DrawHUD(20.0, 20, 0)

	centerIdx := 240*640 + 320
	if fb.Color[centerIdx] != 0xFFFFFFFF {
		t.Fatalf("center pixel should be white crosshair, got 0x%X", fb.Color[centerIdx])
	}
}

func BenchmarkRasterizerTriangle(b *testing.B) {
	fb := NewFramebuffer(640, 480)
	p0 := ScreenPoint{X: 100, Y: 100, Depth: 5}
	p1 := ScreenPoint{X: 200, Y: 100, Depth: 5}
	p2 := ScreenPoint{X: 150, Y: 200, Depth: 5}

	v0 := NewScreenVertex(p0, 0, 0, 1, 0, 1.0)
	v1 := NewScreenVertex(p1, 1, 0, 1, 0, 1.0)
	v2 := NewScreenVertex(p2, 0.5, 1, 1, 0, 1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fb.DrawVoxelTrianglePerspective(v0, v1, v2)
	}
}
