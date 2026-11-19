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

// texHash produces a fast, deterministic pseudo-random hash for organic texture dithering.
func texHash(x, y, seed int) int {
	h := (x*374761393 + y*668265263 + seed*374761393) ^ 0x5DEECE66D
	h = (h ^ (h >> 13)) * 1274126177
	return (h ^ (h >> 16)) & 0x7FFFFFFF
}

func initDefaultTextures() {
	// 1. Dirt Texture (rich warm organic earth brown)
	dirtPalette := [4]uint32{0xFF866043, 0xFF775438, 0xFF966C4D, 0xFF65472E}
	var dirtTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 101) % 100
			c := dirtPalette[0]
			if h < 48 {
				c = dirtPalette[0]
			} else if h < 78 {
				c = dirtPalette[1]
			} else if h < 92 {
				c = dirtPalette[2]
			} else {
				c = dirtPalette[3]
			}
			dirtTex[y][x] = c
		}
	}

	// 2. Grass Top Texture (authentic Minecraft meadow green with natural pixel clustering)
	grassPalette := [4]uint32{0xFF5AA53C, 0xFF4E9333, 0xFF69BD46, 0xFF43812A}
	var grassTopTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 202) % 100
			c := grassPalette[0]
			if h < 52 {
				c = grassPalette[0]
			} else if h < 80 {
				c = grassPalette[1]
			} else if h < 93 {
				c = grassPalette[2]
			} else {
				c = grassPalette[3]
			}
			grassTopTex[y][x] = c
		}
	}

	// 3. Grass Side Texture (Top 3-4 pixels irregular grass fringe, bottom dirt)
	var grassSideTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			fringeStart := 13 - (texHash(x, 0, 303) % 3)
			if y >= fringeStart {
				h := texHash(x, y, 202) % 100
				if h < 65 {
					grassSideTex[y][x] = grassPalette[0]
				} else {
					grassSideTex[y][x] = grassPalette[1]
				}
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
	stonePalette := [4]uint32{0xFF7D7D7D, 0xFF696969, 0xFF8D8D8D, 0xFF5C5C5C}
	var stoneTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 404) % 100
			c := stonePalette[0]
			if h < 48 {
				c = stonePalette[0]
			} else if h < 78 {
				c = stonePalette[1]
			} else if h < 92 {
				c = stonePalette[2]
			} else {
				c = stonePalette[3]
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
			isBorder := (x%4 == 0 && (y+x/4)%4 != 2) || (y%4 == 0 && (x+y/4)%4 != 2)
			if isBorder {
				cobbleTex[y][x] = 0xFF444444
			} else {
				h := texHash(x, y, 505) % 100
				if h < 55 {
					cobbleTex[y][x] = 0xFF7D7D7D
				} else if h < 85 {
					cobbleTex[y][x] = 0xFF666666
				} else {
					cobbleTex[y][x] = 0xFF929292
				}
			}
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
			h := texHash(x, y/2, 707) % 100
			if x%4 == 0 || h < 25 {
				barkSideTex[y][x] = 0xFF4D3B26
			} else if h < 70 {
				barkSideTex[y][x] = 0xFF6B5336
			} else {
				barkSideTex[y][x] = 0xFF5C472E
			}
		}
	}
	var logRingsTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			dx := float64(x - 8)
			dy := float64(y - 8)
			dist := int(dx*dx + dy*dy)
			c := uint32(0xFFB8945F)
			if dist < 9 || (dist > 25 && dist < 36) || dist > 55 {
				c = 0xFF9E7C4D
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
			boardIdx := y / 4
			isBoardLine := (y % 4) == 0
			joinX := ((boardIdx * 7) + 3) % 16
			isJoinLine := (x == joinX)
			if isBoardLine || isJoinLine {
				plankTex[y][x] = 0xFF7D582A
			} else {
				h := texHash(x, y, 606) % 100
				if h < 60 {
					plankTex[y][x] = 0xFFB8945F
				} else if h < 85 {
					plankTex[y][x] = 0xFFA5824E
				} else {
					plankTex[y][x] = 0xFFC9A66F
				}
			}
		}
	}
	Textures[5] = &BlockFaceTextures{
		Top:    plankTex,
		Bottom: plankTex,
		Side:   plankTex,
	}

	// 8. Oak Leaves (ID 18) - with authentic fast/fancy alpha cutouts
	var leavesTex BlockTexture
	leafDark := uint32(0xFF2E631E)
	leafMid := uint32(0xFF387B25)
	leafLight := uint32(0xFF45942E)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 808) % 100
			if h < 25 {
				leavesTex[y][x] = 0x00000000 // Cutout transparency
			} else if h < 62 {
				leavesTex[y][x] = leafDark
			} else if h < 86 {
				leavesTex[y][x] = leafMid
			} else {
				leavesTex[y][x] = leafLight
			}
		}
	}
	Textures[18] = &BlockFaceTextures{
		Top:    leavesTex,
		Bottom: leavesTex,
		Side:   leavesTex,
	}

	// 9. Diamond Ore (ID 56)
	diamondCyan := uint32(0xFF4AEDD7)
	var diamondTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 4 && y == 4) || (x == 5 && y == 4) || (x == 4 && y == 5) ||
				(x == 11 && y == 9) || (x == 12 && y == 9) || (x == 11 && y == 10) ||
				(x == 8 && y == 13) || (x == 9 && y == 13) {
				c = diamondCyan
			}
			diamondTex[y][x] = c
		}
	}
	Textures[56] = &BlockFaceTextures{Top: diamondTex, Bottom: diamondTex, Side: diamondTex}

	// 10. Coal Ore (ID 16)
	coalBlack := uint32(0xFF242424)
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
	ironTan := uint32(0xFFD8AF93)
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
	goldYellow := uint32(0xFFFCEE4B)
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
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 1010) % 100
			if h < 55 {
				sandTex[y][x] = 0xFFDBCFA3
			} else if h < 85 {
				sandTex[y][x] = 0xFFCEBF8A
			} else {
				sandTex[y][x] = 0xFFE6DDB5
			}
		}
	}
	Textures[12] = &BlockFaceTextures{Top: sandTex, Bottom: sandTex, Side: sandTex}

	// 13. Bedrock (ID 7)
	var bedrockTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 1212) % 100
			if h < 50 {
				bedrockTex[y][x] = 0xFF1E1E1E
			} else if h < 80 {
				bedrockTex[y][x] = 0xFF353535
			} else {
				bedrockTex[y][x] = 0xFF4A4A4A
			}
		}
	}
	Textures[7] = &BlockFaceTextures{Top: bedrockTex, Bottom: bedrockTex, Side: bedrockTex}

	// 14. Water (ID 9 & ID 8) - semi-transparent blue
	var waterTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 909) % 100
			if h < 70 {
				waterTex[y][x] = 0xAA3F76E4
			} else {
				waterTex[y][x] = 0xBB528BEE
			}
		}
	}
	Textures[8] = &BlockFaceTextures{Top: waterTex, Bottom: waterTex, Side: waterTex}
	Textures[9] = &BlockFaceTextures{Top: waterTex, Bottom: waterTex, Side: waterTex}

	// 14b. Glass (ID 20) - clear glass with thin frame and glint streaks
	var glassTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			isBorder := x == 0 || x == 15 || y == 0 || y == 15
			isStreak := (x == y && x >= 3 && x <= 6) || (x == y-7 && x >= 8 && x <= 11)
			if isBorder {
				glassTex[y][x] = 0xCCFFFFFF
			} else if isStreak {
				glassTex[y][x] = 0x88FFFFFF
			} else {
				glassTex[y][x] = 0x00000000
			}
		}
	}
	Textures[20] = &BlockFaceTextures{Top: glassTex, Bottom: glassTex, Side: glassTex}

	// 14c. Deepslate (ID 150)
	var deepslateTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 1111) % 100
			if (y/2)%2 == 0 {
				if h < 60 {
					deepslateTex[y][x] = 0xFF3D3D43
				} else {
					deepslateTex[y][x] = 0xFF2F2F35
				}
			} else {
				if h < 60 {
					deepslateTex[y][x] = 0xFF2B2B30
				} else {
					deepslateTex[y][x] = 0xFF35353B
				}
			}
		}
	}
	Textures[150] = &BlockFaceTextures{Top: deepslateTex, Bottom: deepslateTex, Side: deepslateTex}

	// 15. Lava (ID 11 & ID 10)
	var lavaTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 1313) % 100
			if h < 65 {
				lavaTex[y][x] = 0xFFFF5500
			} else {
				lavaTex[y][x] = 0xFFFF9900
			}
		}
	}
	Textures[10] = &BlockFaceTextures{Top: lavaTex, Bottom: lavaTex, Side: lavaTex}
	Textures[11] = &BlockFaceTextures{Top: lavaTex, Bottom: lavaTex, Side: lavaTex}

	// 16. Crafting Table (ID 58)
	var craftTopTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := uint32(0xFFB8945F)
			if x == 0 || x == 15 || y == 0 || y == 15 || x == 5 || x == 10 || y == 5 || y == 10 {
				c = uint32(0xFF7D582A)
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
				c = 0xFF151515
				if y >= 9 && (x == 7 || x == 8) {
					c = 0xFFFF5500
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
	a := (col >> 24) & 0xFF
	if a == 0 {
		return 0 // Preserve full transparent cutouts
	}
	if mult >= 1.0 {
		return col
	}
	r := float32((col >> 16) & 0xFF) * mult
	g := float32((col >> 8) & 0xFF) * mult
	b := float32(col & 0xFF) * mult

	return (a << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}
