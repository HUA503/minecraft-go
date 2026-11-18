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

	// Test 90-degree Right Turn: Look vector should be (+1, 0, 0), Right vector should be (0, -1)
	cam.Yaw = 90.0
	cam.Pitch = 0
	fx, fy, fz := cam.GetLookVector()
	if math.Abs(fx-1.0) > 1e-5 || math.Abs(fy) > 1e-5 || math.Abs(fz) > 1e-5 {
		t.Fatalf("at yaw=90, forward vector should be (1, 0, 0), got (%f, %f, %f)", fx, fy, fz)
	}
	rx, rz := cam.GetRightVector()
	if math.Abs(rx) > 1e-5 || math.Abs(rz-(-1.0)) > 1e-5 {
		t.Fatalf("at yaw=90, right vector should be (0, -1), got (%f, %f)", rx, rz)
	}
	// Point directly ahead at (10, 0, 0) should project at screen center (320, 240)
	pTurn, ok := cam.ProjectToScreen(10, 0, 0, 640, 480)
	if !ok || math.Abs(float64(pTurn.X)-320) > 1.0 || math.Abs(float64(pTurn.Y)-240) > 1.0 {
		t.Fatalf("target ahead at yaw=90 should project to screen center (320, 240), got (%f, %f)", pTurn.X, pTurn.Y)
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

func TestClipTriangleNear(t *testing.T) {
	zNear := 0.05

	// Case 1: All vertices behind near plane
	v0 := ViewVertex{X: 0, Y: 0, Z: 0.01}
	v1 := ViewVertex{X: 1, Y: 0, Z: -0.5}
	v2 := ViewVertex{X: 0, Y: 1, Z: 0.02}
	tris, n := ClipTriangleNear(v0, v1, v2, zNear)
	if n != 0 {
		t.Fatalf("expected 0 triangles for triangle fully behind near plane, got %d", n)
	}

	// Case 2: All vertices in front of near plane
	v0 = ViewVertex{X: 0, Y: 0, Z: 1.0}
	v1 = ViewVertex{X: 1, Y: 0, Z: 1.5}
	v2 = ViewVertex{X: 0, Y: 1, Z: 1.2}
	tris, n = ClipTriangleNear(v0, v1, v2, zNear)
	if n != 1 {
		t.Fatalf("expected 1 triangle for fully visible triangle, got %d", n)
	}
	if tris[0].V0.Z != 1.0 || tris[0].V1.Z != 1.5 || tris[0].V2.Z != 1.2 {
		t.Fatalf("expected unchanged vertices")
	}

	// Case 3: 1 vertex in front (v0), 2 behind (v1, v2)
	v0 = ViewVertex{X: 0, Y: 0, Z: 1.0}
	v1 = ViewVertex{X: 1, Y: 0, Z: -1.0}
	v2 = ViewVertex{X: 0, Y: 1, Z: -1.0}
	tris, n = ClipTriangleNear(v0, v1, v2, zNear)
	if n != 1 {
		t.Fatalf("expected 1 clipped triangle, got %d", n)
	}
	if math.Abs(tris[0].V1.Z-zNear) > 1e-5 || math.Abs(tris[0].V2.Z-zNear) > 1e-5 {
		t.Fatalf("clipped intersection points must have Z == zNear, got %f and %f", tris[0].V1.Z, tris[0].V2.Z)
	}

	// Case 4: 2 vertices in front (v1, v2), 1 behind (v0)
	v0 = ViewVertex{X: 0, Y: 0, Z: -1.0}
	v1 = ViewVertex{X: 1, Y: 0, Z: 1.0}
	v2 = ViewVertex{X: 0, Y: 1, Z: 1.0}
	tris, n = ClipTriangleNear(v0, v1, v2, zNear)
	if n != 2 {
		t.Fatalf("expected 2 clipped triangles (clipped quad), got %d", n)
	}
	for i := 0; i < 2; i++ {
		tri := tris[i]
		if tri.V0.Z < zNear-1e-5 || tri.V1.Z < zNear-1e-5 || tri.V2.Z < zNear-1e-5 {
			t.Fatalf("tri %d has vertex behind near plane: %f, %f, %f", i, tri.V0.Z, tri.V1.Z, tri.V2.Z)
		}
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
