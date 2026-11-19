package render

import (
	"math"
	"testing"
	"time"
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

	// Step 2: Draw translucent water surface at depth 12.0
	wv0 := NewScreenVertex(ScreenPoint{X: 10, Y: 10, Depth: 12.0}, 0, 0, 8, 2, 1.0)
	wv1 := NewScreenVertex(ScreenPoint{X: 50, Y: 10, Depth: 12.0}, 1, 0, 8, 2, 1.0)
	wv2 := NewScreenVertex(ScreenPoint{X: 10, Y: 50, Depth: 12.0}, 0, 1, 8, 2, 1.0)
	fb.DrawVoxelTrianglePerspective(wv0, wv1, wv2)

	if math.Abs(float64(fb.Depth[idx]-12.0)) > 1e-3 {
		t.Fatalf("expected water depth 12.0 to be written to depth buffer, got %f", fb.Depth[idx])
	}
	waterBlendedCol := fb.Color[idx]
	if waterBlendedCol == stoneCol {
		t.Fatalf("water did not blend with stone background")
	}

	// Step 3: Draw a quad behind water at depth 16.0: MUST be culled and NOT overwrite water
	bv0 := NewScreenVertex(ScreenPoint{X: 10, Y: 10, Depth: 16.0}, 0, 0, 3, 2, 1.0)
	bv1 := NewScreenVertex(ScreenPoint{X: 50, Y: 10, Depth: 16.0}, 1, 0, 3, 2, 1.0)
	bv2 := NewScreenVertex(ScreenPoint{X: 10, Y: 50, Depth: 16.0}, 0, 1, 3, 2, 1.0)
	fb.DrawVoxelTrianglePerspective(bv0, bv1, bv2)

	if fb.Color[idx] != waterBlendedCol {
		t.Fatalf("geometry behind water at depth 16.0 improperly overwrote water! Got color %X, expected %X", fb.Color[idx], waterBlendedCol)
	}
}
