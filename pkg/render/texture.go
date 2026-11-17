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
	// Authentic Minecraft Java color palette (0xAARRGGBB for standard 32-bit native display)
	// Dirt: rich warm earth brown
	dirtDark := uint32(0xFF6F4F37)  // R=111, G=79, B=55
	dirtMid := uint32(0xFF866043)   // R=134, G=96, B=67
	dirtLight := uint32(0xFF9B6F4E) // R=155, G=111, B=78

	// Grass: vibrant Minecraft meadow green
	grassGreenDark := uint32(0xFF46862E)  // R=70, G=134, B=46
	grassGreenMid := uint32(0xFF5AA53C)   // R=90, G=165, B=60
	grassGreenLight := uint32(0xFF6FBF4C) // R=111, G=191, B=76

	// Stone: natural mottled gray
	stoneDark := uint32(0xFF636363)
	stoneMid := uint32(0xFF7D7D7D)
	stoneLight := uint32(0xFF969696)

	// Wood Bark: oak tree trunk
	woodBarkDark := uint32(0xFF4D3B26)
	woodBarkLight := uint32(0xFF6B5336)
	woodRingsDark := uint32(0xFF9E7C4D)
	woodRingsLight := uint32(0xFFB8945F)

	// Oak Planks: crafted wooden building boards
	plankLine := uint32(0xFF8F6E3E)
	plankBody := uint32(0xFFB8945F)
	plankLight := uint32(0xFFC9A66F)

	// Oak Leaves: lush foliage
	leafDark := uint32(0xFF2E631E)
	leafLight := uint32(0xFF3D8228)

	// Diamond: dazzling cyan gemstone
	diamondCyan := uint32(0xFF4AEDD7) // R=74, G=237, B=215
	coalBlack := uint32(0xFF242424)
	ironTan := uint32(0xFFD8AF93)
	goldYellow := uint32(0xFFFCEE4B)

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

	// 10. Coal Ore (ID 16)
	var coalTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 3 && y == 3) || (x == 4 && y == 3) || (x == 3 && y == 4) ||
				(x == 10 && y == 11) || (x == 11 && y == 11) || (x == 10 && y == 12) {
				c = coalBlack
			}
			coalTex[y][x] = c
		}
	}
	Textures[16] = &BlockFaceTextures{Top: coalTex, Bottom: coalTex, Side: coalTex}

	// 11. Iron Ore (ID 15)
	var ironTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 6 && y == 5) || (x == 7 && y == 5) || (x == 6 && y == 6) ||
				(x == 12 && y == 7) || (x == 13 && y == 7) || (x == 12 && y == 8) {
				c = ironTan
			}
			ironTex[y][x] = c
		}
	}
	Textures[15] = &BlockFaceTextures{Top: ironTex, Bottom: ironTex, Side: ironTex}

	// 11b. Gold Ore (ID 14)
	var goldTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 5 && y == 6) || (x == 6 && y == 6) || (x == 10 && y == 4) || (x == 9 && y == 12) {
				c = goldYellow
			}
			goldTex[y][x] = c
		}
	}
	Textures[14] = &BlockFaceTextures{Top: goldTex, Bottom: goldTex, Side: goldTex}

	// 12. Sand (ID 12)
	var sandTex BlockTexture
	sandMid := uint32(0xFFDBCFA3)
	sandDark := uint32(0xFFCEBF8A)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := sandMid
			if (x*3+y*7)%4 == 0 {
				c = sandDark
			}
			sandTex[y][x] = c
		}
	}
	Textures[12] = &BlockFaceTextures{Top: sandTex, Bottom: sandTex, Side: sandTex}

	// 13. Bedrock (ID 7)
	var bedrockTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := uint32(0xFF353535)
			if (x*7+y*13)%3 == 0 {
				c = uint32(0xFF1A1A1A)
			} else if (x*11+y*5)%4 == 0 {
				c = uint32(0xFF555555)
			}
			bedrockTex[y][x] = c
		}
	}
	Textures[7] = &BlockFaceTextures{Top: bedrockTex, Bottom: bedrockTex, Side: bedrockTex}

	// 14. Water (ID 9)
	var waterTex BlockTexture
	waterBlue := uint32(0xFF3F76E4)
	waterLight := uint32(0xFF4C85F2)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := waterBlue
			if (x+y)%3 == 0 {
				c = waterLight
			}
			waterTex[y][x] = c
		}
	}
	Textures[9] = &BlockFaceTextures{Top: waterTex, Bottom: waterTex, Side: waterTex}

	// 15. Lava (ID 11)
	var lavaTex BlockTexture
	lavaOrange := uint32(0xFFFF5500)
	lavaYellow := uint32(0xFFFF9900)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := lavaOrange
			if (x*5+y*3)%4 == 0 {
				c = lavaYellow
			}
			lavaTex[y][x] = c
		}
	}
	Textures[11] = &BlockFaceTextures{Top: lavaTex, Bottom: lavaTex, Side: lavaTex}

	// 16. Crafting Table (ID 58)
	var craftTopTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := plankBody
			if x == 0 || x == 15 || y == 0 || y == 15 || x == 8 || y == 8 {
				c = plankLine
			}
			craftTopTex[y][x] = c
		}
	}
	Textures[58] = &BlockFaceTextures{
		Top:    craftTopTex,
		Bottom: plankTex,
		Side:   plankTex,
	}

	// 17. Furnace (ID 61)
	var furnaceFrontTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := cobbleTex[y][x]
			if y >= 6 && y <= 11 && x >= 4 && x <= 11 {
				c = uint32(0xFF151515) // furnace opening
				if y >= 9 && (x == 7 || x == 8) {
					c = lavaOrange // warm coal embers
				}
			}
			furnaceFrontTex[y][x] = c
		}
	}
	Textures[61] = &BlockFaceTextures{
		Top:    cobbleTex,
		Bottom: cobbleTex,
		Side:   furnaceFrontTex,
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

// ApplyLighting scales color channels by directional light multiplier in 0xAARRGGBB format.
func ApplyLighting(col uint32, mult float32) uint32 {
	if mult >= 1.0 {
		return col
	}
	r := float32((col >> 16) & 0xFF) * mult
	g := float32((col >> 8) & 0xFF) * mult
	b := float32(col & 0xFF) * mult
	a := (col >> 24) & 0xFF

	return (a << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}
