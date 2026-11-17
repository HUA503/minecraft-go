package render

import (
	"testing"
)

func TestMeshFaceCullingIsolatedCube(t *testing.T) {
	// Single solid block at (0, 0, 0) surrounded by air
	world := map[[3]int32]int32{
		{0, 0, 0}: 1, // Stone
	}

	getBlock := func(x, y, z int32) int32 {
		return world[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)

	// An isolated cube has 6 exposed faces -> 6 * 4 = 24 vertices, 6 * 6 = 36 indices
	if len(mesh.Vertices) != 24 {
		t.Fatalf("expected 24 vertices for isolated cube, got %d", len(mesh.Vertices))
	}
	if len(mesh.Indices) != 36 {
		t.Fatalf("expected 36 indices for isolated cube, got %d", len(mesh.Indices))
	}
}

func TestMeshFaceCullingAdjacentCubes(t *testing.T) {
	// Two adjacent blocks at (0, 0, 0) and (1, 0, 0)
	world := map[[3]int32]int32{
		{0, 0, 0}: 1,
		{1, 0, 0}: 1,
	}

	getBlock := func(x, y, z int32) int32 {
		return world[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)

	// Two cubes sharing a face have (6 + 6 - 2) = 10 exposed faces -> 10 * 4 = 40 vertices, 60 indices
	if len(mesh.Vertices) != 40 {
		t.Fatalf("expected 40 vertices for two adjacent cubes with culled face, got %d", len(mesh.Vertices))
	}
	if len(mesh.Indices) != 60 {
		t.Fatalf("expected 60 indices, got %d", len(mesh.Indices))
	}
}

func TestRasterizerTriangleAndDepth(t *testing.T) {
	fb := NewFramebuffer(64, 64)
	fb.Clear(0xFF000000) // Clear black

	// Draw red triangle at depth 0.5
	t1_p0 := ScreenPoint{X: 10, Y: 10, Depth: 0.5}
	t1_p1 := ScreenPoint{X: 30, Y: 10, Depth: 0.5}
	t1_p2 := ScreenPoint{X: 20, Y: 30, Depth: 0.5}
	fb.DrawTriangle(t1_p0, t1_p1, t1_p2, 0xFF0000FF) // Red

	// Pixel at (20, 15) should be red
	idx := 15*64 + 20
	if fb.Color[idx] != 0xFF0000FF {
		t.Fatalf("expected red pixel at (20, 15), got 0x%X", fb.Color[idx])
	}
	if fb.Depth[idx] != 0.5 {
		t.Fatalf("expected depth 0.5, got %f", fb.Depth[idx])
	}

	// Draw green triangle BEHIND at depth 0.8 over the same area
	t2_p0 := ScreenPoint{X: 10, Y: 10, Depth: 0.8}
	t2_p1 := ScreenPoint{X: 30, Y: 10, Depth: 0.8}
	t2_p2 := ScreenPoint{X: 20, Y: 30, Depth: 0.8}
	fb.DrawTriangle(t2_p0, t2_p1, t2_p2, 0xFF00FF00) // Green

	// Depth test should discard green triangle: pixel should remain red!
	if fb.Color[idx] != 0xFF0000FF {
		t.Fatalf("depth test failed: behind triangle overwrote front pixel (got 0x%X)", fb.Color[idx])
	}
}
