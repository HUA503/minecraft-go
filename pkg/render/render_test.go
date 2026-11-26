package render

import (
	"math"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestCommon26CubesHaveOfficialFaceTextures(t *testing.T) {
	for _, block := range inventory.ExtendedBlockDefinitions() {
		if block.RuntimeID < 14100 || block.RuntimeID >= 14120 {
			continue
		}
		for _, face := range []struct{ name, texture string }{
			{"top", block.TextureTop}, {"bottom", block.TextureBottom}, {"side", block.TextureSide},
		} {
			if _, ok := GetOfficialTexture("block", face.texture); !ok {
				t.Errorf("%s %s texture %q is missing from bundled vanilla assets", block.Name, face.name, face.texture)
			}
		}
	}
}

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

func TestSlabSelectionOutlineUsesPartialShape(t *testing.T) {
	cam := NewCamera(0.5, 0.5, 0)
	full := NewFramebuffer(160, 120)
	slab := NewFramebuffer(160, 120)
	full.DrawBlockOutline(cam, 0, 0, 4)
	slab.DrawBlockStateOutline(cam, 0, 0, 4, inventory.ItemStoneSlab)
	lineBounds := func(fb *Framebuffer) (int, int, bool) {
		minY, maxY, found := fb.Height, -1, false
		for y := 0; y < fb.Height; y++ {
			for x := 0; x < fb.Width; x++ {
				if fb.Color[y*fb.Width+x] == 0xFF000000 {
					minY = min(minY, y)
					maxY = max(maxY, y)
					found = true
				}
			}
		}
		return minY, maxY, found
	}
	fullMin, fullMax, fullFound := lineBounds(full)
	slabMin, slabMax, slabFound := lineBounds(slab)
	if !fullFound || !slabFound {
		t.Fatalf("expected projected outline pixels (full=%t, slab=%t)", fullFound, slabFound)
	}
	if slabMax-slabMin >= fullMax-fullMin {
		t.Fatalf("slab outline height [%d,%d] should be shorter than cube [%d,%d]", slabMin, slabMax, fullMin, fullMax)
	}
}

func TestSlabMeshUsesTheSamePartialShapesAsCollision(t *testing.T) {
	for _, texture := range []string{"smooth_stone", "smooth_stone_slab_side", "oak_planks"} {
		if _, ok := GetOfficialTexture("block", texture); !ok {
			t.Fatalf("slab block texture %q is missing from bundled Java assets", texture)
		}
	}
	for _, tc := range []struct {
		id      int32
		wantMin float32
		wantMax float32
	}{
		{inventory.ItemStoneSlab, 0, .5},
		{inventory.BlockStoneSlabTop, .5, 1},
		{inventory.ItemOakSlab, 0, .5},
		{inventory.BlockOakSlabTop, .5, 1},
	} {
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return tc.id
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 24 {
			t.Fatalf("slab state %d generated %d vertices, want 24", tc.id, len(mesh.Vertices))
		}
		minY, maxY := float32(math.Inf(1)), float32(math.Inf(-1))
		for _, vertex := range mesh.Vertices {
			minY = min(minY, vertex.Y)
			maxY = max(maxY, vertex.Y)
			if vertex.BlockID != inventory.BlockItemForState(tc.id) {
				t.Errorf("slab state %d mesh texture ID = %d, want canonical item/block %d", tc.id, vertex.BlockID, inventory.BlockItemForState(tc.id))
			}
		}
		if minY != tc.wantMin || maxY != tc.wantMax {
			t.Errorf("slab state %d mesh Y bounds = [%g,%g], want [%g,%g]", tc.id, minY, maxY, tc.wantMin, tc.wantMax)
		}
	}
	for _, block := range inventory.ExtendedBlockDefinitions() {
		if block.RuntimeID < inventory.FirstWoodSlabItem || block.RuntimeID > inventory.LastWoodSlabItem {
			continue
		}
		if Textures[block.RuntimeID] == nil {
			t.Errorf("wood slab %s has no loaded texture", block.Name)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return block.RuntimeID
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 24 {
			t.Errorf("wood slab %s generated %d vertices, want 24", block.Name, len(mesh.Vertices))
			continue
		}
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID != block.RuntimeID {
				t.Errorf("wood slab %s mesh uses texture ID %d, want %d", block.Name, vertex.BlockID, block.RuntimeID)
				break
			}
		}
	}
	for _, block := range inventory.ExtendedBlockDefinitions() {
		if block.RuntimeID < inventory.FirstColorSlabItem || block.RuntimeID > inventory.LastColorSlabItem {
			continue
		}
		if Textures[block.RuntimeID] == nil {
			t.Errorf("colored slab %s has no loaded texture", block.Name)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return block.RuntimeID
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 24 {
			t.Errorf("colored slab %s generated %d vertices, want 24", block.Name, len(mesh.Vertices))
			continue
		}
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID != block.RuntimeID {
				t.Errorf("colored slab %s mesh uses texture ID %d, want %d", block.Name, vertex.BlockID, block.RuntimeID)
				break
			}
		}
	}
	for _, block := range inventory.ExtendedBlockDefinitions() {
		if block.RuntimeID < inventory.FirstMasonrySlabItem || block.RuntimeID > inventory.LastMasonrySlabItem {
			continue
		}
		if Textures[block.RuntimeID] == nil {
			t.Errorf("masonry slab %s has no loaded texture", block.Name)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return block.RuntimeID
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 24 {
			t.Errorf("masonry slab %s generated %d vertices, want 24", block.Name, len(mesh.Vertices))
			continue
		}
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID != block.RuntimeID {
				t.Errorf("masonry slab %s mesh uses texture ID %d, want %d", block.Name, vertex.BlockID, block.RuntimeID)
				break
			}
		}
	}
	if !IsOpaqueCube(inventory.BlockStoneSlabDouble) {
		t.Error("double slab should occlude neighbors as a full cube")
	}
}

func TestJava263ColoredStairMeshesFollowOrientedVoxelShape(t *testing.T) {
	for _, stair := range inventory.ModernStairItemDefinitions() {
		if Textures[stair.RuntimeID] == nil {
			t.Errorf("stair %s has no block texture", stair.Name)
			continue
		}
		for facing := inventory.StairFacing(0); facing < 4; facing++ {
			for _, upsideDown := range []bool{false, true} {
				state, ok := inventory.StairBlockState(stair.RuntimeID, facing, upsideDown)
				if !ok {
					t.Fatalf("failed to construct stair state for %s", stair.Name)
				}
				mesh := BuildSectionMesh(func(x, y, z int32) int32 {
					if x == 0 && y == 0 && z == 0 {
						return state
					}
					return 0
				}, 0, 0, 0)
				if len(mesh.Vertices) != 88 {
					t.Errorf("stair %s state (%d,%t) generated %d vertices, want 88 exposed half-cell faces", stair.Name, facing, upsideDown, len(mesh.Vertices))
					continue
				}
				var minX, minY, minZ = float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))
				var maxX, maxY, maxZ = float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))
				for _, vertex := range mesh.Vertices {
					minX, minY, minZ = min(minX, vertex.X), min(minY, vertex.Y), min(minZ, vertex.Z)
					maxX, maxY, maxZ = max(maxX, vertex.X), max(maxY, vertex.Y), max(maxZ, vertex.Z)
					if vertex.BlockID != stair.RuntimeID {
						t.Errorf("stair %s mesh texture ID = %d, want canonical item %d", stair.Name, vertex.BlockID, stair.RuntimeID)
						break
					}
				}
				if minX != 0 || minY != 0 || minZ != 0 || maxX != 1 || maxY != 1 || maxZ != 1 {
					t.Errorf("stair %s state (%d,%t) mesh bounds = [%g,%g]x[%g,%g]x[%g,%g]", stair.Name, facing, upsideDown, minX, maxX, minY, maxY, minZ, maxZ)
				}
			}
		}
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

func TestAmbientOcclusionSmoothLighting(t *testing.T) {
	// A floor block at (0, 0, 0) and an elevated adjacent wall block at (1, 1, 0)
	world := map[[3]int32]int32{
		{0, 0, 0}: 1, // Floor
		{1, 1, 0}: 1, // Wall next to floor
	}

	getBlock := func(x, y, z int32) int32 {
		return world[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)

	// Check top face of floor block (x=0, y=0, z=0, face=2)
	// Vertices near x=1 should have reduced lighting due to AO wall shadow
	foundShadedVertex := false
	for _, v := range mesh.Vertices {
		if v.BlockID == 1 && v.FaceIndex == 2 && v.Y == 1.0 {
			if v.LightingMult < FaceLighting[2] {
				foundShadedVertex = true
				break
			}
		}
	}

	if !foundShadedVertex {
		t.Fatalf("expected ambient occlusion to shade vertex near adjacent wall block")
	}
}

func TestAmbientOcclusionReadsDiagonalBlockAcrossSectionCorner(t *testing.T) {
	blocks := map[[3]int32]int32{
		{15, 15, 15}: 1, // Target cube in section (0, 0, 0).
	}
	getBlock := func(x, y, z int32) int32 {
		return blocks[[3]int32{x, y, z}]
	}
	cornerVertexLight := func() float32 {
		mesh := BuildSectionMesh(getBlock, 0, 0, 0)
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID == 1 && vertex.FaceIndex == 0 && vertex.X == 16 && vertex.Y == 16 && vertex.Z == 16 {
				return vertex.LightingMult
			}
		}
		t.Fatal("could not find target cube's +X corner vertex")
		return 0
	}

	withoutDiagonal := cornerVertexLight()
	blocks[[3]int32{16, 16, 16}] = 1 // Occupies the diagonal neighbor section.
	withDiagonal := cornerVertexLight()

	if withoutDiagonal != FaceLighting[0] {
		t.Fatalf("unoccluded corner light = %v, want %v", withoutDiagonal, FaceLighting[0])
	}
	want := FaceLighting[0] * 0.8
	if withDiagonal != want {
		t.Fatalf("diagonal block AO light = %v, want %v", withDiagonal, want)
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

func TestCameraSphereInFrustum(t *testing.T) {
	cam := NewCamera(0, 0, 0)
	const width, height = 640, 480
	tanHalfY := math.Tan(cam.Fov * math.Pi / 360)
	tanHalfX := tanHalfY * width / height

	tests := []struct {
		name            string
		x, y, z, radius float64
		want            bool
	}{
		{name: "center ahead", x: 0, y: 0, z: 10, radius: 1, want: true},
		{name: "fully behind", x: 0, y: 0, z: -10, radius: 1, want: false},
		{name: "intersects left edge", x: -10*tanHalfX - 0.5, y: 0, z: 10, radius: 1, want: true},
		{name: "outside left edge", x: -10*tanHalfX - 2, y: 0, z: 10, radius: 1, want: false},
		{name: "intersects top edge", x: 0, y: 10*tanHalfY + 0.5, z: 10, radius: 1, want: true},
		{name: "outside top edge", x: 0, y: 10*tanHalfY + 2, z: 10, radius: 1, want: false},
		{name: "intersects near plane", x: 0, y: 0, z: 0.5, radius: 1, want: true},
		{name: "outside near plane", x: 0, y: 0, z: -2, radius: 1, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cam.SphereInFrustum(tt.x, tt.y, tt.z, tt.radius, width, height, 0.1); got != tt.want {
				t.Fatalf("SphereInFrustum(%v, %v, %v, r=%v) = %v, want %v", tt.x, tt.y, tt.z, tt.radius, got, tt.want)
			}
		})
	}
}

func TestCameraSphereInFrustumConservativeOnInvalidInputs(t *testing.T) {
	cam := NewCamera(0, 0, 0)
	if !cam.SphereInFrustum(0, 0, 10, 1, 0, 480, 0.1) {
		t.Fatal("invalid viewport must conservatively retain the sphere")
	}
	cam.Fov = math.NaN()
	if !cam.SphereInFrustum(0, 0, 10, 1, 640, 480, 0.1) {
		t.Fatal("invalid camera FOV must conservatively retain the sphere")
	}
}

func TestCameraSphereInFrustumTracksYawAndPitch(t *testing.T) {
	cam := NewCamera(0, 0, 0)
	cam.Yaw = 90
	if !cam.SphereInFrustum(10, 0, 0, 1, 640, 480, 0.1) {
		t.Fatal("sphere along a 90-degree yaw direction should be visible")
	}
	if cam.SphereInFrustum(0, 0, 10, 1, 640, 480, 0.1) {
		t.Fatal("sphere behind a 90-degree yaw direction should be culled")
	}

	cam.Yaw = 0
	cam.Pitch = 30
	if !cam.SphereInFrustum(0, 5, 8.660254037844386, 1, 640, 480, 0.1) {
		t.Fatal("sphere along a 30-degree pitch direction should be visible")
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

func TestZBufferSubMillimeterPrecision(t *testing.T) {
	fb := NewFramebuffer(100, 100)
	fb.Clear(0xFF000000)

	// Far triangle: depth 3.0 meters (Stone, blockID 1)
	p0 := ScreenPoint{X: 10, Y: 10, Depth: 3.0}
	p1 := ScreenPoint{X: 90, Y: 10, Depth: 3.0}
	p2 := ScreenPoint{X: 50, Y: 90, Depth: 3.0}
	v0 := NewScreenVertex(p0, 0, 0, 1, 0, 1.0)
	v1 := NewScreenVertex(p1, 1, 0, 1, 0, 1.0)
	v2 := NewScreenVertex(p2, 0.5, 1, 1, 0, 1.0)
	fb.DrawVoxelTrianglePerspective(v0, v1, v2)

	centerIdx := 50*100 + 50
	stoneColor := fb.Color[centerIdx]
	if fb.Depth[centerIdx] < 2.99 || fb.Depth[centerIdx] > 3.01 {
		t.Fatalf("expected depth ~3.0, got %f", fb.Depth[centerIdx])
	}

	// Near triangle: depth 2.8 meters (Grass, blockID 2)
	np0 := ScreenPoint{X: 10, Y: 10, Depth: 2.8}
	np1 := ScreenPoint{X: 90, Y: 10, Depth: 2.8}
	np2 := ScreenPoint{X: 50, Y: 90, Depth: 2.8}
	nv0 := NewScreenVertex(np0, 0, 0, 2, 2, 1.0)
	nv1 := NewScreenVertex(np1, 1, 0, 2, 2, 1.0)
	nv2 := NewScreenVertex(np2, 0.5, 1, 2, 2, 1.0)
	fb.DrawVoxelTrianglePerspective(nv0, nv1, nv2)

	grassColor := fb.Color[centerIdx]
	if grassColor == stoneColor {
		t.Fatalf("near triangle (grass) must overwrite far triangle (stone)")
	}
	if fb.Depth[centerIdx] < 2.79 || fb.Depth[centerIdx] > 2.81 {
		t.Fatalf("expected depth ~2.8, got %f", fb.Depth[centerIdx])
	}

	// Subsequent far triangle: depth 5.0 meters
	fp0 := ScreenPoint{X: 10, Y: 10, Depth: 5.0}
	fp1 := ScreenPoint{X: 90, Y: 10, Depth: 5.0}
	fp2 := ScreenPoint{X: 50, Y: 90, Depth: 5.0}
	fv0 := NewScreenVertex(fp0, 0, 0, 1, 0, 1.0)
	fv1 := NewScreenVertex(fp1, 1, 0, 1, 0, 1.0)
	fv2 := NewScreenVertex(fp2, 0.5, 1, 1, 0, 1.0)
	fb.DrawVoxelTrianglePerspective(fv0, fv1, fv2)

	if fb.Color[centerIdx] != grassColor {
		t.Fatalf("subsequent far triangle must NOT overwrite closer grass triangle")
	}
}

func TestCelestialBodies3D(t *testing.T) {
	fb := NewFramebuffer(400, 300)
	cam := NewCamera(0, 64, 0)

	// Noon daylight (tick 6000): sun should be visible high in the sky
	fb.ClearSkyWithTime(6000)
	cam.Pitch = 85.0 // Look up towards sun at zenith
	fb.DrawCelestialBodies(cam, 6000)

	// Verify sun pixels were drawn on upper screen
	hasSunPixel := false
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.Color[y*fb.Width+x]
			// Sun core is 0xFFFFFFCC
			if c == 0xFFFFFFCC {
				hasSunPixel = true
				break
			}
		}
		if hasSunPixel {
			break
		}
	}
	if !hasSunPixel {
		t.Fatalf("expected sun to be rendered in 3D sky at noon when looking up")
	}

	// Midnight (tick 18000): moon should be visible high in the sky
	fb.ClearSkyWithTime(18000)
	fb.DrawCelestialBodies(cam, 18000)

	hasMoonPixel := false
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.Color[y*fb.Width+x]
			if c == 0xFFF0F4FF {
				hasMoonPixel = true
				break
			}
		}
		if hasMoonPixel {
			break
		}
	}
	if !hasMoonPixel {
		t.Fatalf("expected moon to be rendered in 3D sky at midnight when looking up")
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

func TestPauseMenuAndFont(t *testing.T) {
	fb := NewFramebuffer(800, 600)
	fb.Clear(0xFF102030)

	// Test 5x7 Font text rendering
	w := fb.DrawText5x7(10, 10, "HELLO MINECRAFT", 0xFFFFFFFF, 1)
	if w <= 0 {
		t.Fatalf("expected positive text width, got %d", w)
	}

	// Test Pause Menu rendering and button hit boxes
	backRect, quitRect := fb.DrawPauseMenu(400, 300)
	if backRect[2] <= 0 || backRect[3] <= 0 {
		t.Fatalf("invalid back button rect: %v", backRect)
	}
	if quitRect[2] <= 0 || quitRect[3] <= 0 {
		t.Fatalf("invalid quit button rect: %v", quitRect)
	}

	// Verify back button is above quit button
	if backRect[1] >= quitRect[1] {
		t.Fatalf("expected back button Y (%d) < quit button Y (%d)", backRect[1], quitRect[1])
	}
}

func TestDebugScreenOverlay(t *testing.T) {
	fb := NewFramebuffer(800, 600)
	fb.Clear(0xFF000000)

	info := DebugInfo{
		FPS:           128,
		FrameTimeMs:   7.81,
		X:             8.5,
		Y:             65.0,
		Z:             8.5,
		BlockX:        8,
		BlockY:        65,
		BlockZ:        8,
		ChunkX:        0,
		ChunkZ:        0,
		Facing:        "south (towards +Z)",
		Yaw:           0,
		Pitch:         0,
		TimeOfDay:     6000,
		TargetedBlock: "minecraft:grass_block",
		HasTarget:     true,
		TargetPos:     [3]int32{8, 64, 8},
	}

	fb.DrawDebugScreen(info)

	// Verify that text was drawn to top-left area
	drawnPixels := 0
	for y := 4; y < 80; y++ {
		for x := 4; x < 200; x++ {
			if fb.Color[y*fb.Width+x] == 0xFFFFFFFF {
				drawnPixels++
			}
		}
	}

	if drawnPixels == 0 {
		t.Fatalf("expected white text pixels in F3 debug overlay")
	}
}

func TestChatAndCommandOverlay(t *testing.T) {
	fb := NewFramebuffer(800, 600)
	fb.Clear(0xFF000000)

	msgs := []ChatMessage{
		{Text: "<Player> Hello Minecraft!", Timestamp: time.Now()},
		{Text: "[GameMode] Set game mode to Creative", Timestamp: time.Now()},
	}

	fb.DrawChatOverlay(msgs, true, "/gamemode survival")

	// Verify that chat text was rendered in the bottom area
	drawnPixels := 0
	for y := fb.Height - 60; y < fb.Height-10; y++ {
		for x := 4; x < 300; x++ {
			if fb.Color[y*fb.Width+x] == 0xFFFFFFFF {
				drawnPixels++
			}
		}
	}

	if drawnPixels == 0 {
		t.Fatalf("expected white text pixels in chat overlay")
	}
}

func TestHurtVignette(t *testing.T) {
	fb := NewFramebuffer(640, 480)
	fb.Clear(0xFF000000)

	fb.DrawHurtVignette(1.0)

	// Check top-left corner border pixel (should have red component > 0)
	pixel := fb.Color[0]
	r := (pixel >> 16) & 0xFF
	if r == 0 {
		t.Fatalf("expected red vignette on screen border, got %X", pixel)
	}
}

func TestFoliageCrossedQuadsAndOpaqueExposure(t *testing.T) {
	// Floor grass block at (0, 0, 0) and tall grass foliage directly above it at (0, 1, 0)
	worldMap := map[[3]int32]int32{
		{0, 0, 0}: 2,  // Grass Block (Opaque)
		{0, 1, 0}: 31, // Tall Grass (Foliage)
	}

	getBlock := func(x, y, z int32) int32 {
		return worldMap[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)

	// Grass block has 6 exposed faces (including +Y top face under tall grass!) -> 24 vertices
	// Tall grass has 2 crossed diagonal double-sided quads -> 8 vertices
	// Total vertices = 32
	if len(mesh.Vertices) != 32 {
		t.Fatalf("expected 32 vertices (24 for grass block with exposed top + 8 for foliage), got %d", len(mesh.Vertices))
	}

	// Verify foliage vertices have FaceIndex == 6
	foliageVertCount := 0
	for _, v := range mesh.Vertices {
		if v.BlockID == 31 {
			if v.FaceIndex != 6 {
				t.Errorf("expected FaceIndex == 6 for foliage, got %d", v.FaceIndex)
			}
			foliageVertCount++
		}
	}
	if foliageVertCount != 8 {
		t.Fatalf("expected 8 foliage vertices, got %d", foliageVertCount)
	}

	// Verify Grass Block top face (FaceIndex == 2, Y == 1.0) was NOT culled by tall grass
	topFaceFound := false
	for _, v := range mesh.Vertices {
		if v.BlockID == 2 && v.FaceIndex == 2 {
			topFaceFound = true
			break
		}
	}
	if !topFaceFound {
		t.Fatalf("grass block top face (+Y) was incorrectly culled by tall grass above it")
	}
}

func TestWaterInternalFaceCullingAndDepthStability(t *testing.T) {
	// 1. Two adjacent water blocks (ID 8 and ID 9)
	world := map[[3]int32]int32{
		{0, 0, 0}: 8,
		{1, 0, 0}: 9,
	}
	getBlock := func(x, y, z int32) int32 {
		return world[[3]int32{x, y, z}]
	}

	mesh := BuildSectionMesh(getBlock, 0, 0, 0)
	// Two cubes with 1 shared culled face = 10 faces = 40 vertices
	if len(mesh.Vertices) != 40 {
		t.Fatalf("expected 40 vertices for two adjacent water blocks with internal face culled, got %d", len(mesh.Vertices))
	}

	// 2. Translucent water depth buffer test
	fb := NewFramebuffer(64, 64)
	fb.ClearSky()

	// Step 1: Draw opaque stone floor at depth 15.0
	// Pixel (32, 32): depth 15.0
	v0 := NewScreenVertex(ScreenPoint{X: 10, Y: 10, Depth: 15.0}, 0, 0, 1, 2, 1.0)
	v1 := NewScreenVertex(ScreenPoint{X: 50, Y: 10, Depth: 15.0}, 1, 0, 1, 2, 1.0)
	v2 := NewScreenVertex(ScreenPoint{X: 10, Y: 50, Depth: 15.0}, 0, 1, 1, 2, 1.0)
	fb.DrawVoxelTrianglePerspective(v0, v1, v2)

	idx := 20*fb.Width + 20
	if math.Abs(float64(fb.Depth[idx]-15.0)) > 1e-3 {
		t.Fatalf("expected stone depth 15.0, got %f", fb.Depth[idx])
	}
	stoneCol := fb.Color[idx]

	// Step 2: Blend the nearer water surface over the opaque scene without
	// replacing the opaque depth buffer.
	wv0 := NewScreenVertex(ScreenPoint{X: 10, Y: 10, Depth: 12.0}, 0, 0, 8, 2, 1.0)
	wv1 := NewScreenVertex(ScreenPoint{X: 50, Y: 10, Depth: 12.0}, 1, 0, 8, 2, 1.0)
	wv2 := NewScreenVertex(ScreenPoint{X: 10, Y: 50, Depth: 12.0}, 0, 1, 8, 2, 1.0)
	fb.DrawTranslucentVoxelTrianglePerspective(wv0, wv1, wv2)

	if math.Abs(float64(fb.Depth[idx]-15.0)) > 1e-3 {
		t.Fatalf("transparent water changed opaque depth from 15.0 to %f", fb.Depth[idx])
	}
	waterBlendedCol := fb.Color[idx]
	if waterBlendedCol == stoneCol {
		t.Fatalf("water did not blend with stone background")
	}

	// A second transparent layer in front must still blend, while one behind
	// the opaque floor must fail the depth test.
	nearV0 := NewScreenVertex(ScreenPoint{X: 10, Y: 10, Depth: 11.0}, 0, 0, 8, 2, 1.0)
	nearV1 := NewScreenVertex(ScreenPoint{X: 50, Y: 10, Depth: 11.0}, 1, 0, 8, 2, 1.0)
	nearV2 := NewScreenVertex(ScreenPoint{X: 10, Y: 50, Depth: 11.0}, 0, 1, 8, 2, 1.0)
	fb.DrawTranslucentVoxelTrianglePerspective(nearV0, nearV1, nearV2)
	if fb.Color[idx] == waterBlendedCol {
		t.Fatal("near transparent layer was not blended over the far layer")
	}
	if math.Abs(float64(fb.Depth[idx]-15.0)) > 1e-3 {
		t.Fatalf("transparent layers changed opaque depth to %f", fb.Depth[idx])
	}
	waterBlendedCol = fb.Color[idx]

	// Step 3: A transparent quad behind opaque stone at depth 16.0 must be culled.
	bv0 := NewScreenVertex(ScreenPoint{X: 10, Y: 10, Depth: 16.0}, 0, 0, 3, 2, 1.0)
	bv1 := NewScreenVertex(ScreenPoint{X: 50, Y: 10, Depth: 16.0}, 1, 0, 3, 2, 1.0)
	bv2 := NewScreenVertex(ScreenPoint{X: 10, Y: 50, Depth: 16.0}, 0, 1, 3, 2, 1.0)
	fb.DrawTranslucentVoxelTrianglePerspective(bv0, bv1, bv2)

	if fb.Color[idx] != waterBlendedCol {
		t.Fatalf("geometry behind water at depth 16.0 improperly overwrote water! Got color %X, expected %X", fb.Color[idx], waterBlendedCol)
	}
}

func TestTransparentBlockClassificationAndPortalFaceCulling(t *testing.T) {
	for _, id := range []int32{8, 9, 20, 90} {
		if !IsTranslucentBlock(id) {
			t.Errorf("block %d should use the transparent render pass", id)
		}
	}
	for _, id := range []int32{1, 18, 58} {
		if IsTranslucentBlock(id) {
			t.Errorf("block %d should not use the transparent render pass", id)
		}
	}
	if IsOpaqueCube(90) {
		t.Fatal("nether portal must not occlude neighboring opaque geometry")
	}

	blocks := map[[3]int32]int32{{0, 0, 0}: 90, {1, 0, 0}: 90}
	mesh := BuildSectionMesh(func(x, y, z int32) int32 {
		return blocks[[3]int32{x, y, z}]
	}, 0, 0, 0)
	if len(mesh.Vertices) != 40 {
		t.Fatalf("two adjacent portal blocks should cull their shared face, got %d vertices", len(mesh.Vertices))
	}

	const whiteStainedGlass int32 = 17100
	if IsOpaqueCube(whiteStainedGlass) || !IsTranslucentBlock(whiteStainedGlass) {
		t.Fatal("stained glass must be transparent and must not occlude opaque geometry")
	}
	glass := map[[3]int32]int32{{0, 0, 0}: whiteStainedGlass, {1, 0, 0}: whiteStainedGlass}
	glassMesh := BuildSectionMesh(func(x, y, z int32) int32 {
		return glass[[3]int32{x, y, z}]
	}, 0, 0, 0)
	if len(glassMesh.Vertices) != 40 {
		t.Fatalf("two adjacent same-color stained glass blocks should cull their shared face, got %d vertices", len(glassMesh.Vertices))
	}

	const whiteGlassPane int32 = 17116
	if IsOpaqueCube(whiteGlassPane) || !IsTranslucentBlock(whiteGlassPane) {
		t.Fatal("glass panes must use transparent rendering and must not occlude opaque geometry")
	}
	paneBlocks := map[[3]int32]int32{{0, 0, 0}: whiteGlassPane}
	paneMesh := BuildSectionMesh(func(x, y, z int32) int32 {
		return paneBlocks[[3]int32{x, y, z}]
	}, 0, 0, 0)
	if len(paneMesh.Vertices) != 24 {
		t.Fatalf("isolated glass pane should render a six-sided center post, got %d vertices", len(paneMesh.Vertices))
	}
	paneBlocks[[3]int32{-1, 0, 0}] = 1
	connectedPaneMesh := BuildSectionMesh(func(x, y, z int32) int32 {
		return paneBlocks[[3]int32{x, y, z}]
	}, 0, 0, 0)
	if len(connectedPaneMesh.Vertices) <= len(paneMesh.Vertices) {
		t.Fatalf("connected pane mesh has %d vertices, want arms beyond isolated %d", len(connectedPaneMesh.Vertices), len(paneMesh.Vertices))
	}
}

func TestOfficialItemAndBlockTextures(t *testing.T) {
	for _, id := range []int32{6, 24, 25, 35, 43, 79, 80, 82, 84, 91, 98, 112, 123, 124, 129, 133, 138, 152, 155, 159, 165, 168, 169, 170, 172, 173, 174, 179} {
		if Textures[id] == nil {
			t.Errorf("creative full-block entry %d has no registered block texture", id)
		}
	}
	for _, block := range inventory.ExtendedBlockDefinitions() {
		if Textures[block.RuntimeID] == nil {
			t.Errorf("extended creative block %s has no registered block texture", block.Name)
		}
	}

	// 1. Verify authentic official item textures load
	itemsToCheck := []struct {
		id   int32
		name string
	}{
		{inventory.ItemWoodenPickaxe, "wooden_pickaxe"},
		{276, "diamond_sword"},
		{inventory.ItemApple, "apple"},
		{inventory.ItemBread, "bread"},
		{inventory.ItemIronIngot, "iron_ingot"},
		{inventory.ItemDiamond, "diamond"},
		{inventory.ItemRawCopper, "raw_copper"},
		{inventory.ItemRawIron, "raw_iron"},
		{inventory.ItemRawGold, "raw_gold"},
		{inventory.ItemEmerald, "emerald"},
		{inventory.ItemLapisLazuli, "lapis_lazuli"},
		{inventory.ItemCopperIngot, "copper_ingot"},
		{inventory.ItemCopperSword, "copper_sword"},
		{inventory.ItemCopperShovel, "copper_shovel"},
		{inventory.ItemCopperPickaxe, "copper_pickaxe"},
		{inventory.ItemCopperAxe, "copper_axe"},
		{inventory.ItemCopperHoe, "copper_hoe"},
		{inventory.ItemNetheriteScrap, "netherite_scrap"},
		{inventory.ItemTorch, "torch"},
	}

	for _, item := range itemsToCheck {
		sprite, ok := GetOfficialItemSprite(item.id)
		if !ok {
			t.Fatalf("expected official texture for item %d (%s) to be found", item.id, item.name)
		}
		hasPixels := false
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if (sprite[y][x] >> 24) > 0 {
					hasPixels = true
					break
				}
			}
		}
		if !hasPixels {
			t.Fatalf("official texture for item %s has no non-transparent pixels", item.name)
		}
	}

	// 2. Verify GetItemSprite16x16 uses official texture
	pickSprite := GetItemSprite16x16(inventory.ItemWoodenPickaxe)
	// In official wooden_pickaxe.png: pixel (10, 3) is pickaxe head (opaque)
	if (pickSprite[3][10] >> 24) == 0 {
		t.Fatalf("expected pixel (10, 3) in wooden pickaxe to be opaque, got %X", pickSprite[3][10])
	}

	// 3. Verify RenderIsometricBlock16x16 renders isometric cube
	isoGrass := RenderIsometricBlock16x16(inventory.ItemGrass)
	// Center of top face (7, 5) should be green tinted
	centerCol := isoGrass[5][7]
	if (centerCol >> 24) == 0 {
		t.Fatalf("expected isometric grass top face to be rendered, got %X", centerCol)
	}

	// 4. Verify HUD DrawItemSprite16x16 renders to framebuffer
	fb := NewFramebuffer(32, 32)
	fb.DrawItemSprite16x16(5, 5, inventory.ItemApple)
	// Check that apple pixels were drawn
	appleDrawn := false
	for y := 5; y < 21; y++ {
		for x := 5; x < 21; x++ {
			if (fb.Color[y*fb.Width+x] >> 24) > 0 {
				appleDrawn = true
				break
			}
		}
	}
	if !appleDrawn {
		t.Fatalf("apple sprite was not drawn to framebuffer")
	}
}

func TestPoplarCreativeItemsHaveOfficialTexturesAndRenderShapes(t *testing.T) {
	for _, id := range []int32{10060, 10061, 10062, 10063} {
		if _, ok := GetOfficialItemSprite(id); !ok {
			t.Errorf("extended item %d has no official sprite", id)
		}
		if Textures[id] == nil {
			t.Errorf("extended block %d has no registered block texture", id)
		}
	}
	for _, id := range []int32{10060, 10061, 10062} {
		if IsOpaqueCube(id) {
			t.Errorf("poplar leaves %d must not occlude neighboring faces", id)
		}
		blocks := map[[3]int32]int32{{0, 0, 0}: id, {1, 0, 0}: id}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }, 0, 0, 0)
		if len(mesh.Vertices) != 40 {
			t.Errorf("adjacent leaf blocks %d should cull their shared face, got %d vertices", id, len(mesh.Vertices))
		}
	}
	if !IsFoliage(10063) || IsOpaqueCube(10063) {
		t.Fatal("poplar sapling must render as non-occluding crossed foliage")
	}
	if !IsFoliage(10100) || IsOpaqueCube(10100) {
		t.Fatal("red shrub must render as non-occluding crossed foliage")
	}
	mesh := BuildSectionMesh(func(x, y, z int32) int32 {
		if x == 0 && y == 0 && z == 0 {
			return 10100
		}
		return 0
	}, 0, 0, 0)
	if len(mesh.Vertices) != 8 {
		t.Fatalf("red shrub cross mesh has %d vertices, want 8", len(mesh.Vertices))
	}
	if texture, ok := GetOfficialItemSprite(10100); !ok || texture[7][7] == 0 {
		t.Fatal("26.3 red shrub creative icon is missing its official texture")
	}
}

func TestCreativeSpawnEggsHaveOfficialTextures(t *testing.T) {
	for _, egg := range inventory.SpawnEggDefinitions() {
		if got := GetOfficialItemTextureName(egg.RuntimeID); got != egg.Texture {
			t.Errorf("spawn egg %s texture = %q, want %q", egg.EntityName, got, egg.Texture)
		}
		if _, ok := GetOfficialItemSprite(egg.RuntimeID); !ok {
			t.Errorf("spawn egg %s has no official icon", egg.EntityName)
		}
	}
	if _, ok := GetOfficialItemSprite(inventory.ItemBoneMeal); !ok {
		t.Fatal("bone meal has no official item icon")
	}
}

func TestCopperArmorHasOfficialCreativeTextures(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		name string
	}{{inventory.ItemCopperHelmet, "copper_helmet"}, {inventory.ItemCopperChestplate, "copper_chestplate"}, {inventory.ItemCopperLeggings, "copper_leggings"}, {inventory.ItemCopperBoots, "copper_boots"}} {
		if got := GetOfficialItemTextureName(tc.id); got != tc.name {
			t.Errorf("copper armor %d texture = %q, want %q", tc.id, got, tc.name)
		}
		if _, ok := GetOfficialItemSprite(tc.id); !ok {
			t.Errorf("copper armor %d has no official icon", tc.id)
		}
	}
}

func TestNetheriteItemsHaveOfficialCreativeTextures(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		name string
	}{{inventory.ItemNetheriteIngot, "netherite_ingot"}, {inventory.ItemNetheriteHelmet, "netherite_helmet"}, {inventory.ItemNetheriteChestplate, "netherite_chestplate"}, {inventory.ItemNetheriteLeggings, "netherite_leggings"}, {inventory.ItemNetheriteBoots, "netherite_boots"}, {inventory.ItemNetheriteSword, "netherite_sword"}, {inventory.ItemNetheriteAxe, "netherite_axe"}, {inventory.ItemNetheritePickaxe, "netherite_pickaxe"}, {inventory.ItemNetheriteShovel, "netherite_shovel"}, {inventory.ItemNetheriteHoe, "netherite_hoe"}} {
		if got := GetOfficialItemTextureName(tc.id); got != tc.name {
			t.Errorf("netherite item %d texture = %q, want %q", tc.id, got, tc.name)
		}
		if _, ok := GetOfficialItemSprite(tc.id); !ok {
			t.Errorf("netherite item %d has no embedded vanilla icon", tc.id)
		}
	}
}

func TestDrawCloudsAestheticsAndDepth(t *testing.T) {
	fb := NewFramebuffer(160, 120)
	fb.ClearSky()

	// Camera looking up towards sky dome (pitch 45 degrees)
	cam := NewCamera(0, 66, 0)
	cam.Pitch = 45.0

	// Draw clouds at noon (worldTime = 6000)
	fb.DrawClouds(cam, 6000)

	// Verify that cloud pixels were written with depth around ~60 to 120m
	hasCloudPixel := false
	for i := 0; i < len(fb.Depth); i++ {
		d := fb.Depth[i]
		if d < 1e8 && d > 10.0 {
			hasCloudPixel = true
			break
		}
	}
	if !hasCloudPixel {
		t.Fatalf("expected clouds to be rasterized on sky dome when looking up")
	}
}

func TestVerifyMobAndHUDSprites(t *testing.T) {
	mobs := []string{
		"entity/pig/pig.png",
		"entity/cow/cow.png",
		"entity/chicken/chicken.png",
		"entity/sheep/sheep.png",
		"entity/zombie/zombie.png",
		"entity/skeleton/skeleton.png",
		"entity/creeper/creeper.png",
	}
	for _, m := range mobs {
		img, ok := GetOfficialImage(m)
		if !ok || img == nil {
			t.Errorf("failed to load mob image: %s", m)
		} else {
			t.Logf("loaded %s (%dx%d)", m, img.Width, img.Height)
		}
	}

	hudSprites := []string{
		"gui/sprites/hud/crosshair.png",
		"gui/sprites/hud/hotbar.png",
		"gui/sprites/hud/hotbar_selection.png",
		"gui/sprites/hud/experience_bar_background.png",
		"gui/sprites/hud/heart/container.png",
		"gui/sprites/hud/heart/full.png",
		"gui/sprites/hud/heart/half.png",
		"gui/sprites/hud/food_empty.png",
		"gui/sprites/hud/food_full.png",
		"gui/sprites/hud/food_half.png",
	}
	for _, h := range hudSprites {
		img, ok := GetOfficialImage(h)
		if !ok || img == nil {
			t.Errorf("failed to load hud sprite: %s", h)
		} else {
			t.Logf("loaded %s (%dx%d)", h, img.Width, img.Height)
		}
	}

	cImg, _ := GetOfficialImage("gui/sprites/hud/heart/container.png")
	fImg, _ := GetOfficialImage("gui/sprites/hud/heart/full.png")
	foodE, _ := GetOfficialImage("gui/sprites/hud/food_empty.png")
	foodF, _ := GetOfficialImage("gui/sprites/hud/food_full.png")
	t.Logf("container pixel (0,0)=%08X (4,4)=%08X", cImg.Pixels[0], cImg.Pixels[4*9+4])
	t.Logf("full heart pixel (0,0)=%08X (4,4)=%08X", fImg.Pixels[0], fImg.Pixels[4*9+4])
	t.Logf("food_empty pixel (0,0)=%08X (4,4)=%08X", foodE.Pixels[0], foodE.Pixels[4*9+4])
	t.Logf("food_full pixel (0,0)=%08X (4,4)=%08X", foodF.Pixels[0], foodF.Pixels[4*9+4])
}

func TestPigTextureFacialFeatures(t *testing.T) {
	img, ok := GetOfficialImage("entity/pig/pig.png")
	if !ok || img == nil {
		t.Fatalf("could not load pig image")
	}
	// Pig face is at texture rect [8..15, 8..15].
	// Row 11 contains the pig's eyes: pupils at x=8 and x=15 (near black), sclera at x=9 and x=14 (white 0xFFFFFF).
	pupilLeft := img.Pixels[11*img.Width+8] & 0xFFFFFF
	scleraLeft := img.Pixels[11*img.Width+9] & 0xFFFFFF
	scleraRight := img.Pixels[11*img.Width+14] & 0xFFFFFF
	pupilRight := img.Pixels[11*img.Width+15] & 0xFFFFFF

	if scleraLeft != 0xFFFFFF || scleraRight != 0xFFFFFF {
		t.Errorf("expected pig eyes to have white sclera 0xFFFFFF, got left=0x%X right=0x%X", scleraLeft, scleraRight)
	}
	if pupilLeft > 0x303030 || pupilRight > 0x303030 {
		t.Errorf("expected dark pupil pixels, got left=0x%X right=0x%X", pupilLeft, pupilRight)
	}
}

func TestDrawFullHUDSurvivalAndCreative(t *testing.T) {
	fb := NewFramebuffer(320, 240)
	var slots [9]inventory.ItemStack
	slots[0] = inventory.ItemStack{ItemID: 1, Count: 64} // Stone

	// Survival HUD: renders hearts, hunger, exp bar, level
	fb.Clear(0xFF000000)
	fb.DrawFullHUD(0, 15.0, 18, 0.65, 27, 0, slots)
	// Check that framebuffer has non-zero pixels rendered
	var nonZeroCount int
	for _, p := range fb.Color {
		if p != 0xFF000000 {
			nonZeroCount++
		}
	}
	if nonZeroCount < 500 {
		t.Errorf("expected Survival HUD to render hearts, hunger, and exp bar, nonZeroCount=%d", nonZeroCount)
	}

	// Creative HUD: only crosshair and hotbar
	fb.Clear(0xFF000000)
	fb.DrawFullHUD(1, 20.0, 20, 0.0, 0, 0, slots)
	var creativeCount int
	for _, p := range fb.Color {
		if p != 0xFF000000 {
			creativeCount++
		}
	}
	if creativeCount == 0 || creativeCount >= nonZeroCount {
		t.Errorf("expected Creative HUD to render fewer elements than Survival, creative=%d survival=%d", creativeCount, nonZeroCount)
	}
}

func TestCrossedQuadsFoliageUprightOrientation(t *testing.T) {
	worldMap := map[[3]int32]int32{
		{0, 0, 0}: 37, // Dandelion (yellow flower)
	}
	getBlock := func(x, y, z int32) int32 {
		return worldMap[[3]int32{x, y, z}]
	}
	mesh := BuildSectionMesh(getBlock, 0, 0, 0)
	if len(mesh.Vertices) == 0 {
		t.Fatalf("expected mesh vertices for flower block")
	}

	for _, v := range mesh.Vertices {
		if v.FaceIndex == 6 {
			// Ground level (Y=0) must have V=1.0 (stem/roots)
			if v.Y == 0 && v.V != 1.0 {
				t.Errorf("expected bottom vertex Y=0 to have V=1.0, got V=%f", v.V)
			}
			// Top level (Y=1) must have V=0.0 (flower blossom)
			if v.Y == 1 && v.V != 0.0 {
				t.Errorf("expected top vertex Y=1 to have V=0.0, got V=%f", v.V)
			}
		}
	}
}

func TestJava263FoodTabItemsHaveOfficialSprites(t *testing.T) {
	items := inventory.CreativeCatalog(inventory.CreativeFood)
	if len(items) != 42 {
		t.Fatalf("creative food tab has %d entries, want 42", len(items))
	}
	for _, item := range items {
		textureName := GetOfficialItemTextureName(item.ID)
		if textureName == "" {
			t.Errorf("food item %q (%d) has no vanilla texture mapping", item.Name, item.ID)
			continue
		}
		if _, ok := GetOfficialItemSprite(item.ID); !ok {
			t.Errorf("food item %q (%d) texture %q is missing from embedded assets", item.Name, item.ID, textureName)
		}
	}
}
