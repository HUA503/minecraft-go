package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
)

func saveFBToPNG(fb *render.Framebuffer, filename string) error {
	img := image.NewRGBA(image.Rect(0, 0, fb.Width, fb.Height))
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.Color[y*fb.Width+x]
			a := uint8((c >> 24) & 0xFF)
			r := uint8((c >> 16) & 0xFF)
			g := uint8((c >> 8) & 0xFF)
			b := uint8(c & 0xFF)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func TestExportFrames(t *testing.T) {
	// 1. World Select Screen
	fbWS := render.NewFramebuffer(800, 600)
	worlds := []render.WorldInfo{
		{DirName: "world1", LevelName: "我的生存世界", GameMode: 0, LastTime: time.Now()},
		{DirName: "creative_city", LevelName: "创造大都市", GameMode: 1, LastTime: time.Now().Add(-2 * time.Hour)},
	}
	fbWS.DrawWorldSelectScreen(worlds, 0, 400, 300)
	_ = saveFBToPNG(fbWS, "verify_world_select.png")

	// 2. In-Game Scene
	fbGame := render.NewFramebuffer(800, 600)
	cam := render.NewCamera(0, 65.62, 0)
	cam.Yaw = 0
	cam.Pitch = 5.0

	pInv := inventory.NewPlayerInventory()
	pInv.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemIronSword, Count: 1}
	pInv.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemDiamondPickaxe, Count: 1}
	pInv.Slots[2] = inventory.ItemStack{ItemID: 3, Count: 64} // Dirt
	pInv.Slots[3] = inventory.ItemStack{ItemID: 260, Count: 16} // Apple
	pInv.SelectedHotbar = 0

	c := &Client{
		framebuf:     fbGame,
		camera:       cam,
		isLocal:      true,
		engine:       sim.NewEngine(),
		player: &sim.PlayerEntity{
			Position:  sim.Vec3{X: 0, Y: 64, Z: 0},
			Health:    18.0,
			Hunger:    16,
			OnGround:  true,
			GameMode:  0,
			Inventory: pInv,
		},
		equipProgress: 1.0,
	}

	fbGame.Clear(0xFF87CEEB) // Sky blue

	// Spawn a pig 4 meters ahead
	pigPos := sim.Vec3{X: 1.0, Y: 64.0, Z: 4.0}
	pig := c.engine.SpawnMob(sim.MobPig, pigPos)
	pig.Yaw = 210.0 // 3/4 angle
	c.drawMob(pig)

	// Spawn a cow 4.5 meters ahead slightly to the left
	cowPos := sim.Vec3{X: -1.8, Y: 64.0, Z: 4.5}
	cow := c.engine.SpawnMob(sim.MobCow, cowPos)
	cow.Yaw = 160.0
	c.drawMob(cow)

	// Draw held item
	c.drawHeldItem()

	// Draw HUD with hearts, hunger, exp
	var hotbar [9]inventory.ItemStack
	copy(hotbar[:], pInv.Slots[:9])
	c.framebuf.DrawFullHUD(0, 18.0, 16, 0.65, 12, 0, hotbar)

	_ = saveFBToPNG(fbGame, "verify_ingame_hud.png")
}
func TestInspectTextures(t *testing.T) {
	pigTex, ok := render.GetOfficialImage("entity/pig/pig.png")
	if !ok || pigTex == nil {
		t.Fatalf("pigTex not loaded")
	}
	t.Logf("pigTex: %dx%d, total pixels: %d", pigTex.Width, pigTex.Height, len(pigTex.Pixels))
	// Sample non-zero count
	nonZero := 0
	for _, p := range pigTex.Pixels {
		if (p >> 24) != 0 {
			nonZero++
		}
	}
	t.Logf("pigTex non-zero alpha pixels: %d / %d", nonZero, len(pigTex.Pixels))

	// Check body UV region: [28..64, 8..32]
	bodyNonZero := 0
	for y := 8; y < 32; y++ {
		for x := 28; x < 64; x++ {
			p := pigTex.Pixels[y*pigTex.Width+x]
			if (p >> 24) != 0 {
				bodyNonZero++
			}
		}
	}
	t.Logf("pigTex body region [28..64, 8..32] non-zero: %d", bodyNonZero)

	cowTex, ok := render.GetOfficialImage("entity/cow/cow.png")
	if !ok || cowTex == nil {
		t.Fatalf("cowTex not loaded")
	}
	t.Log("Cow Texture (Row 0 to 16, Col 0 to 32):")
	for y := 0; y < 16; y++ {
		rowStr := ""
		for x := 0; x < 32; x++ {
			p := cowTex.Pixels[y*cowTex.Width+x]
			r := (p >> 16) & 0xFF
			g := (p >> 8) & 0xFF
			b := p & 0xFF
			if (p >> 24) == 0 {
				rowStr += " "
			} else if r > 180 && g > 180 && b > 180 { // White/Grey spots
				rowStr += "W"
			} else if r < 50 && g < 50 && b < 50 { // Black
				rowStr += "B"
			} else if r > 150 && g < 120 && b < 120 { // Pink (snout/udder)
				rowStr += "P"
			} else { // Brown
				rowStr += "."
			}
		}
		t.Logf("y=%2d: %s", y, rowStr)
	}
}
func TestTracePigBodyFaces(t *testing.T) {
	fb := render.NewFramebuffer(800, 600)
	cam := render.NewCamera(0, 65.62, 0)
	cam.Yaw = 0
	cam.Pitch = 5.0

	radY := 180.0 * 3.141592653589793 / 180.0
	cosY, sinY := -1.0, 0.0

	rot := func(lx, ly, lz float64) (float64, float64, float64) {
		rx := lx*cosY + lz*sinY
		rz2 := -lx*sinY + lz*cosY
		return 0.5 + rx, 64.55 + ly, 4.0 + rz2
	}
	rotNormal := func(nx, ny, nz float64) (float64, float64, float64) {
		rx := nx*cosY + nz*sinY
		rz2 := -nx*sinY + nz*cosY
		return rx, ny, rz2
	}

	hx, hy, hz := 0.26, 0.22, 0.38
	corners := [8][3]float64{
		{-hx, -hy, -hz}, // 0
		{hx, -hy, -hz},  // 1
		{hx, hy, -hz},   // 2
		{-hx, hy, -hz},  // 3
		{-hx, -hy, hz},  // 4
		{hx, -hy, hz},   // 5
		{hx, hy, hz},    // 6
		{-hx, hy, hz},   // 7
	}

	type boxFaceDef struct {
		indices [4]int
		normal  [3]float64
		center  [3]float64
		name    string
	}
	faces := [6]boxFaceDef{
		{indices: [4]int{3, 2, 6, 7}, normal: [3]float64{0, 1, 0}, center: [3]float64{0, hy, 0}, name: "Top"},
		{indices: [4]int{4, 5, 1, 0}, normal: [3]float64{0, -1, 0}, center: [3]float64{0, -hy, 0}, name: "Bottom"},
		{indices: [4]int{2, 3, 0, 1}, normal: [3]float64{0, 0, -1}, center: [3]float64{0, 0, -hz}, name: "North"},
		{indices: [4]int{7, 6, 5, 4}, normal: [3]float64{0, 0, 1}, center: [3]float64{0, 0, hz}, name: "South"},
		{indices: [4]int{3, 7, 4, 0}, normal: [3]float64{-1, 0, 0}, center: [3]float64{-hx, 0, 0}, name: "West"},
		{indices: [4]int{6, 2, 1, 5}, normal: [3]float64{1, 0, 0}, center: [3]float64{hx, 0, 0}, name: "East"},
	}

	for _, f := range faces {
		rnx, rny, rnz := rotNormal(f.normal[0], f.normal[1], f.normal[2])
		fcx, fcy, fcz := rot(f.center[0], f.center[1], f.center[2])
		vx := fcx - cam.X
		vy := fcy - cam.Y
		vz := fcz - cam.Z
		dot := rnx*vx + rny*vy + rnz*vz
		t.Logf("Face %s: normal=(%.1f, %.1f, %.1f) rotNormal=(%.1f, %.1f, %.1f) v=(%.1f, %.1f, %.1f) dot=%.2f culled=%v",
			f.name, f.normal[0], f.normal[1], f.normal[2], rnx, rny, rnz, vx, vy, vz, dot, dot >= 0)
	}

	_ = fb
	_ = radY
	_ = corners
}
func TestTracePigBodyDrawing(t *testing.T) {
	fb := render.NewFramebuffer(800, 600)
	cam := render.NewCamera(0, 65.62, 0)
	cam.Yaw = 0
	cam.Pitch = 5.0

	c := &Client{
		framebuf: fb,
		camera:   cam,
	}

	fb.Clear(0xFF87CEEB)

	mobTex, ok := render.GetOfficialImage("entity/pig/pig.png")
	if !ok || mobTex == nil {
		t.Fatalf("pigTex not found")
	}

	bodyUV := makeBoxFaceUVs(64, 64, [4]float32{36, 16, 46, 32}, [4]float32{54, 16, 64, 32}, [4]float32{46, 8, 56, 16}, [4]float32{36, 8, 46, 16}, [4]float32{28, 16, 36, 32}, [4]float32{46, 16, 54, 32})
	fbPigBody := [6]uint32{0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0}
	
	// Count non-zero pixels before
	beforePix := 0
	for _, col := range fb.Color {
		if col != 0 {
			beforePix++
		}
	}

	c.renderTextured3DBox(0.5, 64.55, 4.0, 0.26, 0.22, 0.38, 180.0, 0, mobTex, bodyUV, fbPigBody, 0)

	changedPix := 0
	for _, col := range fb.Color {
		if col != 0xFF87CEEB {
			changedPix++
		}
	}
	t.Logf("Pig body rendered pixels: %d", changedPix)
}
func TestDebugPigBodyFaces(t *testing.T) {
	fb := render.NewFramebuffer(800, 600)
	fb.Clear(0xFF87CEEB)
	cam := render.NewCamera(0, 65.62, 0)
	cam.Yaw = 0
	cam.Pitch = 5.0

	cx, cy, cz := 0.5, 64.55, 4.0
	hx, hy, hz := 0.26, 0.22, 0.38
	yawDeg, pitchDeg := 180.0, 0.0

	radY := yawDeg * math.Pi / 180.0
	radP := pitchDeg * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)
	cosP, sinP := math.Cos(radP), math.Sin(radP)

	rot := func(lx, ly, lz float64) (float64, float64, float64) {
		ry := ly*cosP - lz*sinP
		rz := ly*sinP + lz*cosP
		rx := lx*cosY + rz*sinY
		rz2 := -lx*sinY + rz*cosY
		return cx + rx, cy + ry, cz + rz2
	}

	corners := [8][3]float64{
		{-hx, -hy, -hz},
		{hx, -hy, -hz},
		{hx, hy, -hz},
		{-hx, hy, -hz},
		{-hx, -hy, hz},
		{hx, -hy, hz},
		{hx, hy, hz},
		{-hx, hy, hz},
	}

	var screenPts [8]render.ScreenPoint
	var valid [8]bool
	for i := 0; i < 8; i++ {
		wx, wy, wz := rot(corners[i][0], corners[i][1], corners[i][2])
		screenPts[i], valid[i] = cam.ProjectToScreen(wx, wy, wz, fb.Width, fb.Height)
		t.Logf("corner %d: (%.2f, %.2f, %.2f) -> screen (%.1f, %.1f) depth=%.2f valid=%v",
			i, wx, wy, wz, screenPts[i].X, screenPts[i].Y, screenPts[i].Depth, valid[i])
	}
}
func TestExportAngledMobView(t *testing.T) {
	fb := render.NewFramebuffer(800, 600)
	fb.Clear(0xFF87CEEB)
	cam := render.NewCamera(-3.0, 65.5, 1.0)
	cam.Yaw = 35.0
	cam.Pitch = 10.0

	c := &Client{
		framebuf: fb,
		camera:   cam,
		engine:   sim.NewEngine(),
	}

	// Pig at (-0.6, 64, 4), yaw 220 (3/4 front angle facing camera)
	pigPos := sim.Vec3{X: -0.6, Y: 64.0, Z: 4.0}
	pig := c.engine.SpawnMob(sim.MobPig, pigPos)
	pig.Yaw = 220.0
	c.drawMob(pig)

	// Cow at (1.5, 64, 4.5), yaw 210 (3/4 front angle facing camera)
	cowPos := sim.Vec3{X: 1.5, Y: 64.0, Z: 4.5}
	cow := c.engine.SpawnMob(sim.MobCow, cowPos)
	cow.Yaw = 210.0
	c.drawMob(cow)

	_ = saveFBToPNG(fb, "verify_mobs_angled.png")

	// Cow Side Profile
	fbCowSide := render.NewFramebuffer(800, 600)
	fbCowSide.Clear(0xFF87CEEB)
	camSide := render.NewCamera(0, 65.5, 0)
	camSide.Yaw = 0
	camSide.Pitch = 5.0
	cSide := &Client{framebuf: fbCowSide, camera: camSide, engine: sim.NewEngine()}
	cowSide := cSide.engine.SpawnMob(sim.MobCow, sim.Vec3{X: 0, Y: 64, Z: 4.5})
	cowSide.Yaw = 90.0 // 90 degrees = side view (profile)
	cSide.drawMob(cowSide)
	_ = saveFBToPNG(fbCowSide, "verify_cow_side.png")
}
func TestInspectPigTextureLayout(t *testing.T) {
	pigTex, _ := render.GetOfficialImage("entity/pig/pig.png")
	// Print ascii representation of the 32x32 upper left texture
	t.Log("Pig Texture (Row 8 to 16, Col 0 to 32):")
	for y := 8; y < 16; y++ {
		rowStr := ""
		for x := 0; x < 32; x++ {
			p := pigTex.Pixels[y*pigTex.Width+x]
			r := (p >> 16) & 0xFF
			g := (p >> 8) & 0xFF
			b := p & 0xFF
			if (p >> 24) == 0 {
				rowStr += " "
			} else if r > 200 && g < 150 { // Pink/Red
				rowStr += "."
			} else if r < 50 && g < 50 && b < 50 { // Black (eyes/nostrils)
				rowStr += "B"
			} else if r > 200 && g > 200 && b > 200 { // White (eyes)
				rowStr += "W"
			} else {
				rowStr += "#"
			}
		}
		t.Logf("y=%2d: %s", y, rowStr)
	}
}
func TestInspectPigSnoutLayout(t *testing.T) {
	pigTex, _ := render.GetOfficialImage("entity/pig/pig.png")
	t.Log("Pig Texture (Row 16 to 24, Col 0 to 32):")
	for y := 16; y < 24; y++ {
		rowStr := ""
		for x := 0; x < 32; x++ {
			p := pigTex.Pixels[y*pigTex.Width+x]
			r := (p >> 16) & 0xFF
			g := (p >> 8) & 0xFF
			b := p & 0xFF
			if (p >> 24) == 0 {
				rowStr += " "
			} else if r > 200 && g < 150 { // Pink/Red
				rowStr += "."
			} else if r < 50 && g < 50 && b < 50 { // Black (nostrils)
				rowStr += "B"
			} else if r > 200 && g > 200 && b > 200 { // White
				rowStr += "W"
			} else {
				rowStr += "#"
			}
		}
		t.Logf("y=%2d: %s", y, rowStr)
	}
}
func TestExportHeldBlock(t *testing.T) {
	fb := render.NewFramebuffer(800, 600)
	fb.Clear(0xFF87CEEB)
	cam := render.NewCamera(0, 65.62, 0)
	cam.Yaw = 0
	cam.Pitch = 5.0

	pInv := inventory.NewPlayerInventory()
	pInv.Slots[0] = inventory.ItemStack{ItemID: 3, Count: 64} // Dirt block
	pInv.SelectedHotbar = 0

	c := &Client{
		framebuf:     fb,
		camera:       cam,
		isLocal:      true,
		engine:       sim.NewEngine(),
		player: &sim.PlayerEntity{
			Position:  sim.Vec3{X: 0, Y: 64, Z: 0},
			Health:    20.0,
			Hunger:    20,
			OnGround:  true,
			GameMode:  0,
			Inventory: pInv,
		},
		equipProgress: 1.0,
	}

	c.drawHeldItem()

	var hotbar [9]inventory.ItemStack
	copy(hotbar[:], pInv.Slots[:9])
	c.framebuf.DrawFullHUD(0, 20.0, 20, 0.0, 0, 0, hotbar)

	_ = saveFBToPNG(fb, "verify_held_block.png")
}
