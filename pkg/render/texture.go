package render

// BlockTexture is a 16x16 RGBA pixel buffer.
type BlockTexture [16][16]uint32

// Face textures for standard Minecraft blocks.
type BlockFaceTextures struct {
	Top    BlockTexture
	Bottom BlockTexture
	Side   BlockTexture
}

// Global texture registry for voxel rendering.
var Textures = make(map[int32]*BlockFaceTextures)

func init() {
	initDefaultTextures()
}

func initDefaultTextures() {
	// Color constants (0xAABBGGRR)
	dirtDark := uint32(0xFF223A5C)
	dirtLight := uint32(0xFF2C4A75)
	dirtMid := uint32(0xFF28436B)

	grassGreenDark := uint32(0xFF1E6E36)
	grassGreenMid := uint32(0xFF258542)
	grassGreenLight := uint32(0xFF2E9E4F)

	stoneDark := uint32(0xFF5A5A5A)
	stoneMid := uint32(0xFF6E6E6E)
	stoneLight := uint32(0xFF7E7E7E)

	woodBarkDark := uint32(0xFF18304E)
	woodBarkLight := uint32(0xFF22446E)
	woodRingsDark := uint32(0xFF386898)
	woodRingsLight := uint32(0xFF5088C0)

	plankLine := uint32(0xFF285078)
	plankBody := uint32(0xFF3E75A8)
	plankLight := uint32(0xFF4C8ABE)

	leafDark := uint32(0xFF165022)
	leafLight := uint32(0xFF247A34)

	diamondCyan := uint32(0xFFE0E020) // Cyan in 0xAABBGGRR (Blue=0xE0, Green=0xE0, Red=0x20)

	// 1. Dirt Texture
	var dirtTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := dirtMid
			if (x*3+y*7)%5 == 0 {
				c = dirtDark
			} else if (x*7+y*3)%7 == 0 {
				c = dirtLight
			}
			dirtTex[y][x] = c
		}
	}

	// 2. Grass Top Texture
	var grassTopTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := grassGreenMid
			if (x+y*3)%4 == 0 {
				c = grassGreenDark
			} else if (x*2+y)%5 == 0 {
				c = grassGreenLight
			}
			grassTopTex[y][x] = c
		}
	}

	// 3. Grass Side Texture (Top 3-4 pixels grass fringe, bottom dirt)
	var grassSideTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y < 3 || (y == 3 && (x%3 == 0 || x%4 == 0)) {
				grassSideTex[y][x] = grassGreenMid
			} else {
				grassSideTex[y][x] = dirtTex[y][x]
			}
		}
	}

	// Register Grass Block (ID 2)
	Textures[2] = &BlockFaceTextures{
		Top:    grassTopTex,
		Bottom: dirtTex,
		Side:   grassSideTex,
	}

	// Register Dirt Block (ID 3)
	Textures[3] = &BlockFaceTextures{
		Top:    dirtTex,
		Bottom: dirtTex,
		Side:   dirtTex,
	}

	// 4. Stone Texture (ID 1)
	var stoneTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneMid
			if (x*5+y*11)%7 == 0 {
				c = stoneDark
			} else if (x*11+y*5)%9 == 0 {
				c = stoneLight
			}
			stoneTex[y][x] = c
		}
	}
	Textures[1] = &BlockFaceTextures{
		Top:    stoneTex,
		Bottom: stoneTex,
		Side:   stoneTex,
	}

	// 5. Cobblestone Texture (ID 4)
	var cobbleTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneMid
			if x%4 == 0 || y%4 == 0 || (x+y)%5 == 0 {
				c = stoneDark
			} else if (x*3+y*2)%6 == 0 {
				c = stoneLight
			}
			cobbleTex[y][x] = c
		}
	}
	Textures[4] = &BlockFaceTextures{
		Top:    cobbleTex,
		Bottom: cobbleTex,
		Side:   cobbleTex,
	}

	// 6. Oak Log (ID 17)
	var barkSideTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := woodBarkLight
			if x%4 == 0 || (x+y*2)%7 == 0 {
				c = woodBarkDark
			}
			barkSideTex[y][x] = c
		}
	}
	var logRingsTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			dx := float64(x - 8)
			dy := float64(y - 8)
			dist := int(dx*dx + dy*dy)
			c := woodRingsLight
			if dist < 9 || (dist > 25 && dist < 36) || dist > 55 {
				c = woodRingsDark
			}
			logRingsTex[y][x] = c
		}
	}
	Textures[17] = &BlockFaceTextures{
		Top:    logRingsTex,
		Bottom: logRingsTex,
		Side:   barkSideTex,
	}

	// 7. Oak Planks (ID 5)
	var plankTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := plankBody
			if y%4 == 0 {
				c = plankLine // horizontal wood planks
			} else if (x+y)%6 == 0 {
				c = plankLight
			}
			plankTex[y][x] = c
		}
	}
	Textures[5] = &BlockFaceTextures{
		Top:    plankTex,
		Bottom: plankTex,
		Side:   plankTex,
	}

	// 8. Oak Leaves (ID 18)
	var leavesTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := leafDark
			if (x+y)%2 == 0 {
				c = leafLight
			}
			leavesTex[y][x] = c
		}
	}
	Textures[18] = &BlockFaceTextures{
		Top:    leavesTex,
		Bottom: leavesTex,
		Side:   leavesTex,
	}

	// 9. Diamond Ore (ID 56)
	var diamondTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			// Diamond speckles in stone
			if (x == 4 && y == 4) || (x == 5 && y == 4) || (x == 4 && y == 5) ||
				(x == 11 && y == 9) || (x == 12 && y == 9) || (x == 11 && y == 10) ||
				(x == 8 && y == 13) || (x == 9 && y == 13) {
				c = diamondCyan
			}
			diamondTex[y][x] = c
		}
	}
	Textures[56] = &BlockFaceTextures{
		Top:    diamondTex,
		Bottom: diamondTex,
		Side:   diamondTex,
	}
}

// SampleBlockColor samples pixel color from block texture and applies directional lighting.
func SampleBlockColor(blockID int32, faceIndex int, u, v float32, lightMult float32) uint32 {
	tex, ok := Textures[blockID]
	if !ok || tex == nil {
		// Fallback for unregistered blocks
		return ApplyLighting(0xFF808080, lightMult)
	}

	var face *BlockTexture
	switch faceIndex {
	case 2: // +Y (Top)
		face = &tex.Top
	case 3: // -Y (Bottom)
		face = &tex.Bottom
	default: // Sides
		face = &tex.Side
	}

	// Wrap UV to [0, 1]
	tx := int(u * 16.0) % 16
	ty := int(v * 16.0) % 16
	if tx < 0 {
		tx += 16
	}
	if ty < 0 {
		ty += 16
	}

	pixel := face[ty][tx]
	return ApplyLighting(pixel, lightMult)
}

// ApplyLighting scales RGBA color channels by directional light multiplier.
func ApplyLighting(col uint32, mult float32) uint32 {
	if mult >= 1.0 {
		return col
	}
	r := float32(col & 0xFF) * mult
	g := float32((col >> 8) & 0xFF) * mult
	b := float32((col >> 16) & 0xFF) * mult
	a := (col >> 24) & 0xFF

	return (a << 24) | (uint32(b) << 16) | (uint32(g) << 8) | uint32(r)
}
