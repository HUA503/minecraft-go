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
	initOfficialBlockTextures()
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

	// 18. Tall Grass (ID 31 - Alpha cutout crossed foliage)
	var tallGrassTex BlockTexture
	grassColors := [4]uint32{0xFF4E9333, 0xFF5AA53C, 0xFF69BD46, 0xFF3D7828}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			// Natural slender grass blades rising from bottom (y=0) to top (y=15)
			bladeWidth := 1 + (15-y)/3
			isBlade := false
			bladeCenters := [5]int{2, 5, 8, 11, 14}
			for _, bc := range bladeCenters {
				diff := x - bc
				if diff < 0 {
					diff = -diff
				}
				if diff*2 <= bladeWidth && y <= 14-(bc%3)*2 {
					isBlade = true
					break
				}
			}
			if isBlade {
				h := texHash(x, y, 3131) % 4
				tallGrassTex[y][x] = grassColors[h]
			} else {
				tallGrassTex[y][x] = 0 // Full transparent cutout
			}
		}
	}
	Textures[31] = &BlockFaceTextures{Top: tallGrassTex, Bottom: tallGrassTex, Side: tallGrassTex}

	// 19. Dandelion (ID 37 - Yellow Flower with Green Stem)
	var dandelionTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y >= 7 && (x == 7 || x == 8) {
				// Green stem (bottom)
				dandelionTex[y][x] = 0xFF4E9333
			} else if y >= 1 && y <= 6 && x >= 5 && x <= 10 {
				// Yellow dandelion blossom (top)
				h := texHash(x, y, 3737) % 3
				switch h {
				case 0:
					dandelionTex[y][x] = 0xFFFFEE33
				case 1:
					dandelionTex[y][x] = 0xFFFFDD00
				default:
					dandelionTex[y][x] = 0xFFFFAA00
				}
			} else {
				dandelionTex[y][x] = 0 // Cutout
			}
		}
	}
	Textures[37] = &BlockFaceTextures{Top: dandelionTex, Bottom: dandelionTex, Side: dandelionTex}

	// 20. Poppy (ID 38 - Red Rose/Poppy)
	var poppyTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y >= 7 && (x == 7 || x == 8) {
				// Green stem (bottom)
				poppyTex[y][x] = 0xFF3D7828
			} else if y >= 2 && y <= 6 && x >= 5 && x <= 10 {
				// Red petals (top)
				if x >= 7 && x <= 8 && y >= 4 && y <= 5 {
					poppyTex[y][x] = 0xFF221111 // Dark flower core
				} else {
					h := texHash(x, y, 3838) % 3
					if h == 0 {
						poppyTex[y][x] = 0xFFE52020
					} else if h == 1 {
						poppyTex[y][x] = 0xFFC01515
					} else {
						poppyTex[y][x] = 0xFF8A0808
					}
				}
			} else {
				poppyTex[y][x] = 0 // Cutout
			}
		}
	}
	Textures[38] = &BlockFaceTextures{Top: poppyTex, Bottom: poppyTex, Side: poppyTex}

	// 21. Torch (ID 50 - Slender wooden post with glowing flame head)
	var torchTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x >= 7 && x <= 8 {
				if y <= 10 {
					// Wooden stick
					if (x+y)%2 == 0 {
						torchTex[y][x] = 0xFF6F4F37
					} else {
						torchTex[y][x] = 0xFF8F6849
					}
				} else {
					// Glowing flame
					if y >= 14 {
						torchTex[y][x] = 0xFFFF8800 // Orange tip
					} else {
						torchTex[y][x] = 0xFFFFEE55 // Yellow flame core
					}
				}
			} else if (x == 6 || x == 9) && y >= 11 && y <= 13 {
				torchTex[y][x] = 0xAAFF8800 // Flame outer glow
			} else {
				torchTex[y][x] = 0 // Cutout
			}
		}
	}
	Textures[50] = &BlockFaceTextures{Top: torchTex, Bottom: torchTex, Side: torchTex}

	// 22. Bricks (ID 45)
	var brickTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			// Mortar seams
			isMortar := (y%4 == 0) || ((y/4)%2 == 0 && x%8 == 0) || ((y/4)%2 == 1 && (x+4)%8 == 0)
			if isMortar {
				brickTex[y][x] = 0xFFD8D2CE // Mortar gray
			} else {
				h := texHash(x, y, 4545) % 3
				switch h {
				case 0:
					brickTex[y][x] = 0xFFB35238 // Rich clay red
				case 1:
					brickTex[y][x] = 0xFF9E432C
				default:
					brickTex[y][x] = 0xFFBD5E45
				}
			}
		}
	}
	Textures[45] = &BlockFaceTextures{Top: brickTex, Bottom: brickTex, Side: brickTex}

	// 23. Obsidian (ID 49)
	var obsidianTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 4949) % 100
			if h < 65 {
				obsidianTex[y][x] = 0xFF151020 // Deep dark purple-black
			} else if h < 90 {
				obsidianTex[y][x] = 0xFF241C35 // Lustrous sheen
			} else {
				obsidianTex[y][x] = 0xFF3D2E5C // Violet crystal highlight
			}
		}
	}
	Textures[49] = &BlockFaceTextures{Top: obsidianTex, Bottom: obsidianTex, Side: obsidianTex}

	// 24. Netherrack (ID 87)
	var netherrackTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 8787) % 100
			if h < 55 {
				netherrackTex[y][x] = 0xFF6B2424 // Crimson rock
			} else if h < 85 {
				netherrackTex[y][x] = 0xFF521818
			} else {
				netherrackTex[y][x] = 0xFF853030
			}
		}
	}
	Textures[87] = &BlockFaceTextures{Top: netherrackTex, Bottom: netherrackTex, Side: netherrackTex}

	// 25. Glowstone (ID 89)
	var glowstoneTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 8989) % 100
			if h < 50 {
				glowstoneTex[y][x] = 0xFFFFD466 // Golden luminous crystalline
			} else if h < 80 {
				glowstoneTex[y][x] = 0xFFECC04D
			} else {
				glowstoneTex[y][x] = 0xFFFFF099
			}
		}
	}
	Textures[89] = &BlockFaceTextures{Top: glowstoneTex, Bottom: glowstoneTex, Side: glowstoneTex}

	// 26. End Stone (ID 121)
	var endStoneTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 121121) % 100
			if h < 60 {
				endStoneTex[y][x] = 0xFFE2E4A6 // Pale yellow lunar rock
			} else if h < 85 {
				endStoneTex[y][x] = 0xFFD4D695
			} else {
				endStoneTex[y][x] = 0xFFEFF1BC
			}
		}
	}
	Textures[121] = &BlockFaceTextures{Top: endStoneTex, Bottom: endStoneTex, Side: endStoneTex}

	// 27. Birch Log (ID 19) & Birch Leaves (ID 21)
	var birchBarkTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 1919) % 100
			if (y%3 == 0 && (x+y)%5 == 0) || h < 12 {
				birchBarkTex[y][x] = 0xFF282828 // Black flecks
			} else if h < 60 {
				birchBarkTex[y][x] = 0xFFEBEBEB // Chalky white
			} else {
				birchBarkTex[y][x] = 0xFFD8D8D8
			}
		}
	}
	Textures[19] = &BlockFaceTextures{Top: logRingsTex, Bottom: logRingsTex, Side: birchBarkTex}

	var birchLeavesTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 2121) % 100
			if h < 24 {
				birchLeavesTex[y][x] = 0
			} else if h < 60 {
				birchLeavesTex[y][x] = 0xFF659832
			} else {
				birchLeavesTex[y][x] = 0xFF7BB33B
			}
		}
	}
	Textures[21] = &BlockFaceTextures{Top: birchLeavesTex, Bottom: birchLeavesTex, Side: birchLeavesTex}

	// 28. Spruce Log (ID 22) & Spruce Leaves (ID 23)
	var spruceBarkTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 2222) % 100
			if h < 40 {
				spruceBarkTex[y][x] = 0xFF3B2716
			} else if h < 75 {
				spruceBarkTex[y][x] = 0xFF2E1E11
			} else {
				spruceBarkTex[y][x] = 0xFF4A341E
			}
		}
	}
	Textures[22] = &BlockFaceTextures{Top: logRingsTex, Bottom: logRingsTex, Side: spruceBarkTex}

	var spruceLeavesTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 2323) % 100
			if h < 22 {
				spruceLeavesTex[y][x] = 0
			} else if h < 65 {
				spruceLeavesTex[y][x] = 0xFF2D502E
			} else {
				spruceLeavesTex[y][x] = 0xFF3B683C
			}
		}
	}
	Textures[23] = &BlockFaceTextures{Top: spruceLeavesTex, Bottom: spruceLeavesTex, Side: spruceLeavesTex}

	// 29. Cactus (ID 81)
	var cactusSideTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			isSpine := (x == 3 || x == 8 || x == 13) && (y%4 == 1)
			if isSpine {
				cactusSideTex[y][x] = 0xFFF0EBE0 // Pale spine
			} else if x%3 == 0 {
				cactusSideTex[y][x] = 0xFF35681E // Ridge shadow
			} else {
				cactusSideTex[y][x] = 0xFF558C2F // Cactus green
			}
		}
	}
	Textures[81] = &BlockFaceTextures{Top: cactusSideTex, Bottom: cactusSideTex, Side: cactusSideTex}

	// 30. Dead Bush (ID 32), Fern (ID 33), Lily Pad (ID 111)
	var deadBushTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y <= 12 && (x+y)%3 == 0 && (x >= 3 && x <= 12) {
				deadBushTex[y][x] = 0xFF886633
			} else {
				deadBushTex[y][x] = 0
			}
		}
	}
	Textures[32] = &BlockFaceTextures{Top: deadBushTex, Bottom: deadBushTex, Side: deadBushTex}

	var fernTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			diff := x - 8
			if diff < 0 {
				diff = -diff
			}
			if diff <= (14-y)/2 && y <= 14 {
				fernTex[y][x] = 0xFF3D7828
			} else {
				fernTex[y][x] = 0
			}
		}
	}
	Textures[33] = &BlockFaceTextures{Top: fernTex, Bottom: fernTex, Side: fernTex}

	var lilyPadTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			dx := x - 8
			dy := y - 8
			if dx*dx+dy*dy <= 45 && !(dx > 2 && dy == 0) {
				lilyPadTex[y][x] = 0xFF2A591E
			} else {
				lilyPadTex[y][x] = 0
			}
		}
	}
	Textures[111] = &BlockFaceTextures{Top: lilyPadTex, Bottom: lilyPadTex, Side: lilyPadTex}

	// 31. Ores: Redstone (ID 73), Lapis (ID 74), Copper (ID 153), Quartz (ID 154)
	var redstoneTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 4 && y == 4) || (x == 5 && y == 4) || (x == 10 && y == 11) || (x == 11 && y == 11) || (x == 7 && y == 9) {
				c = 0xFFFF1818
			}
			redstoneTex[y][x] = c
		}
	}
	Textures[73] = &BlockFaceTextures{Top: redstoneTex, Bottom: redstoneTex, Side: redstoneTex}

	var lapisTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 6 && y == 6) || (x == 7 && y == 6) || (x == 12 && y == 8) || (x == 3 && y == 11) {
				c = 0xFF1B4AB7
			}
			lapisTex[y][x] = c
		}
	}
	Textures[74] = &BlockFaceTextures{Top: lapisTex, Bottom: lapisTex, Side: lapisTex}

	var copperTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := stoneTex[y][x]
			if (x == 5 && y == 5) || (x == 6 && y == 5) || (x == 11 && y == 10) {
				c = 0xFFE0734A // Raw copper orange
			} else if (x == 6 && y == 6) || (x == 10 && y == 10) {
				c = 0xFF4EB890 // Verdigris patina green
			}
			copperTex[y][x] = c
		}
	}
	Textures[153] = &BlockFaceTextures{Top: copperTex, Bottom: copperTex, Side: copperTex}

	var quartzTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := netherrackTex[y][x]
			if (x == 4 && y == 4) || (x == 5 && y == 4) || (x == 10 && y == 11) || (x == 11 && y == 11) || (x == 8 && y == 7) {
				c = 0xFFEDE7DC // White quartz crystals
			}
			quartzTex[y][x] = c
		}
	}
	Textures[154] = &BlockFaceTextures{Top: quartzTex, Bottom: quartzTex, Side: quartzTex}

	// 32. Mossy Cobblestone (ID 48), Monster Spawner (ID 52), Chest (ID 54)
	var mossyCobbleTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := cobbleTex[y][x]
			if (x+y)%4 == 0 || (x*y)%7 == 0 {
				c = 0xFF4A7436 // Forest moss
			}
			mossyCobbleTex[y][x] = c
		}
	}
	Textures[48] = &BlockFaceTextures{Top: mossyCobbleTex, Bottom: mossyCobbleTex, Side: mossyCobbleTex}

	var spawnerTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			isBar := x == 0 || x == 15 || y == 0 || y == 15 || x == 5 || x == 10 || y == 5 || y == 10
			if isBar {
				spawnerTex[y][x] = 0xFF242630
			} else {
				spawnerTex[y][x] = 0 // Cage air openings
			}
		}
	}
	Textures[52] = &BlockFaceTextures{Top: spawnerTex, Bottom: spawnerTex, Side: spawnerTex}

	var chestSideTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			isBorder := x == 0 || x == 15 || y == 0 || y == 15 || y == 6
			isLatch := (x == 7 || x == 8) && (y >= 5 && y <= 8)
			if isLatch {
				chestSideTex[y][x] = 0xFFC0C0C0 // Silver latch
			} else if isBorder {
				chestSideTex[y][x] = 0xFF4D381F // Dark wooden band
			} else {
				chestSideTex[y][x] = 0xFF8A6538 // Chest wood
			}
		}
	}
	Textures[54] = &BlockFaceTextures{Top: plankTex, Bottom: plankTex, Side: chestSideTex}

	// 33. Soul Sand (ID 88)
	var soulSandTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 8888) % 100
			if h < 55 {
				soulSandTex[y][x] = 0xFF523D33
			} else if h < 85 {
				soulSandTex[y][x] = 0xFF433026
			} else {
				soulSandTex[y][x] = 0xFF35251D
			}
		}
	}
	Textures[88] = &BlockFaceTextures{Top: soulSandTex, Bottom: soulSandTex, Side: soulSandTex}

	// 34. Nether Portal (ID 90) - Swirling ethereal purple with alpha
	var portalTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 9090) % 100
			if h < 45 {
				portalTex[y][x] = 0xCC7F1ED6
			} else if h < 75 {
				portalTex[y][x] = 0xDDA53EFA
			} else {
				portalTex[y][x] = 0xEE4B0D80
			}
		}
	}
	Textures[90] = &BlockFaceTextures{Top: portalTex, Bottom: portalTex, Side: portalTex}

	// 35. End Portal Frame (ID 120) & End Portal (ID 119)
	var frameTopTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			isEye := x >= 5 && x <= 10 && y >= 5 && y <= 10
			if isEye {
				if (x == 7 || x == 8) && (y == 7 || y == 8) {
					frameTopTex[y][x] = 0xFF142B24 // Eye pupil
				} else {
					frameTopTex[y][x] = 0xFF2F8569 // Teal Eye of Ender
				}
			} else {
				frameTopTex[y][x] = 0xFF3E6958 // End Portal frame dark green rim
			}
		}
	}
	Textures[120] = &BlockFaceTextures{Top: frameTopTex, Bottom: endStoneTex, Side: endStoneTex}

	var endPortalTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 119119) % 100
			if h < 8 {
				endPortalTex[y][x] = 0xFF8AE8D0 // Twinkling cosmic cyan star
			} else if h < 14 {
				endPortalTex[y][x] = 0xFFFFFFFF // White star
			} else {
				endPortalTex[y][x] = 0xFF020712 // Deep space black
			}
		}
	}
	Textures[119] = &BlockFaceTextures{Top: endPortalTex, Bottom: endPortalTex, Side: endPortalTex}

	// 36. Dragon Egg (ID 122)
	var eggTex BlockTexture
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			h := texHash(x, y, 122122) % 100
			if h < 18 {
				eggTex[y][x] = 0xFF6D219C // Radiant dragon purple fleck
			} else if h < 60 {
				eggTex[y][x] = 0xFF0C0912 // Obsidian deep black
			} else {
				eggTex[y][x] = 0xFF1C1428
			}
		}
	}
	Textures[122] = &BlockFaceTextures{Top: eggTex, Bottom: eggTex, Side: eggTex}
}

// initOfficialBlockTextures loads authentic official textures from embedded Mojang vanilla assets.
func initOfficialBlockTextures() {
	loadBlock := func(id int32, top, bottom, side string) {
		topTex, topOk := GetOfficialTexture("block", top)
		bottomTex, botOk := GetOfficialTexture("block", bottom)
		sideTex, sideOk := GetOfficialTexture("block", side)
		if topOk && botOk && sideOk {
			if top == "grass_block_top" {
				for y := 0; y < 16; y++ {
					for x := 0; x < 16; x++ {
						grey := (topTex[y][x] >> 16) & 0xFF
						r := (grey * 0x5A) / 255
						g := (grey * 0xA5) / 255
						b := (grey * 0x3C) / 255
						topTex[y][x] = 0xFF000000 | (r << 16) | (g << 8) | b
					}
				}
			}
			Textures[id] = &BlockFaceTextures{
				Top:    topTex,
				Bottom: bottomTex,
				Side:   sideTex,
			}
		}
	}

	loadSingle := func(id int32, name string) {
		loadBlock(id, name, name, name)
	}

	loadSingle(1, "stone")
	loadBlock(2, "grass_block_top", "dirt", "grass_block_side")
	loadSingle(3, "dirt")
	loadSingle(4, "cobblestone")
	loadSingle(5, "oak_planks")
	loadSingle(6, "oak_sapling")
	loadSingle(7, "bedrock")
	loadBlock(24, "sandstone_top", "sandstone_bottom", "sandstone")
	loadSingle(25, "note_block")
	loadSingle(35, "white_wool")
	loadSingle(43, "stone")
	loadSingle(12, "sand")
	loadSingle(13, "gravel")
	loadSingle(14, "gold_ore")
	loadSingle(15, "iron_ore")
	loadSingle(16, "coal_ore")
	loadBlock(17, "oak_log_top", "oak_log_top", "oak_log")
	loadSingle(18, "oak_leaves")
	loadSingle(20, "glass")
	loadSingle(37, "dandelion")
	loadSingle(38, "poppy")
	loadSingle(45, "bricks")
	loadBlock(46, "tnt_top", "tnt_bottom", "tnt_side")
	loadSingle(49, "obsidian")
	loadSingle(50, "torch")
	loadSingle(56, "diamond_ore")
	loadSingle(57, "diamond_block")
	loadBlock(58, "crafting_table_top", "oak_planks", "crafting_table_side")
	loadBlock(61, "furnace_top", "furnace_top", "furnace_front")
	loadSingle(65, "ladder")
	loadSingle(79, "ice")
	loadSingle(80, "snow")
	loadSingle(82, "clay")
	loadSingle(83, "sugar_cane")
	loadBlock(86, "pumpkin_top", "pumpkin_top", "pumpkin_side")
	loadBlock(84, "jukebox_top", "jukebox_top", "jukebox_side")
	loadSingle(91, "jack_o_lantern")
	loadSingle(123, "redstone_lamp_on")
	loadSingle(124, "redstone_lamp")
	loadSingle(129, "emerald_ore")
	loadSingle(133, "emerald_block")
	loadSingle(138, "beacon")
	loadSingle(152, "redstone_block")
	loadSingle(165, "slime_block")
	loadSingle(168, "prismarine")
	loadSingle(169, "sea_lantern")
	loadBlock(170, "hay_block_top", "hay_block_top", "hay_block_side")
	loadSingle(87, "netherrack")
	loadSingle(88, "soul_sand")
	loadSingle(89, "glowstone")
	loadSingle(98, "stone_bricks")
	loadSingle(112, "nether_bricks")
	loadBlock(103, "melon_top", "melon_top", "melon_side")
	loadSingle(121, "end_stone")
	loadBlock(155, "quartz_block_top", "quartz_block_bottom", "quartz_block_side")
	loadSingle(159, "white_terracotta")
	loadSingle(172, "terracotta")
	loadSingle(173, "coal_block")
	loadSingle(174, "packed_ice")
	loadBlock(179, "red_sandstone_top", "red_sandstone_bottom", "red_sandstone")
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

	// Sample UV coordinates mapped to 16x16 texels with boundary clamping
	if u < 0 {
		u = 0
	} else if u > 0.9999 {
		u = 0.9999
	}
	if v < 0 {
		v = 0
	} else if v > 0.9999 {
		v = 0.9999
	}
	tx := int(u * 16.0)
	ty := int(v * 16.0)
	if tx < 0 {
		tx = 0
	} else if tx > 15 {
		tx = 15
	}
	if ty < 0 {
		ty = 0
	} else if ty > 15 {
		ty = 15
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
	r := float32((col>>16)&0xFF) * mult
	g := float32((col>>8)&0xFF) * mult
	b := float32(col&0xFF) * mult

	return (a << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}
