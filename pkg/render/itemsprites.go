package render

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// colorMultipliers scales RGB channels by factor clamped to [0, 255].
func applyLighting(c uint32, factor float32) uint32 {
	a := (c >> 24) & 0xFF
	if a == 0 {
		return 0
	}
	r := uint32(math.Min(255, math.Max(0, float64(float32((c>>16)&0xFF)*factor))))
	g := uint32(math.Min(255, math.Max(0, float64(float32((c>>8)&0xFF)*factor))))
	b := uint32(math.Min(255, math.Max(0, float64(float32(c&0xFF)*factor))))
	return (a << 24) | (r << 16) | (g << 8) | b
}

// RenderIsometricBlock16x16 rasterizes an authentic 16x16 3D isometric voxel block icon.
// Light angles match vanilla Minecraft GUI: Top face (1.0x), Left face (0.8x), Right face (0.6x).
func RenderIsometricBlock16x16(blockID int32) [16][16]uint32 {
	if inventory.IsFenceBlock(blockID) || inventory.IsWoodFenceGateBlock(blockID) || inventory.IsWoodDoorBlock(blockID) || inventory.IsTrapdoorBlock(blockID) || inventory.IsMasonryWallBlock(blockID) || inventory.IsPaneBlock(blockID) || inventory.IsIronChainBlock(blockID) {
		return renderIsometricConnectedBlock16x16(blockID)
	}
	if itemID, ok := inventory.StairItemForBlockState(blockID); ok {
		state := blockID
		if blockID == itemID {
			state, _ = inventory.StairBlockState(itemID, 0, false)
		}
		return renderIsometricStair16x16(state)
	}
	if boxes, ok := inventory.BlockCollisionBoxes(blockID); ok && len(boxes) == 1 {
		box := boxes[0]
		if box.MinY > 0 || box.MaxY < 1 {
			return renderIsometricPartialBlock16x16(blockID, box)
		}
		blockID = inventory.BlockItemForState(blockID)
	}
	return renderIsometricCube16x16(blockID)
}

func renderIsometricConnectedBlock16x16(blockID int32) [16][16]uint32 {
	texture := Textures[blockID]
	if texture == nil {
		return renderIsometricCube16x16(blockID)
	}
	blocks := map[[3]int32]int32{{0, 0, -1}: inventory.ItemOakPlanks, {1, 0, 0}: inventory.ItemOakPlanks,
		{0, 0, 1}: inventory.ItemOakPlanks, {-1, 0, 0}: inventory.ItemOakPlanks}
	neighbors := func(x, y, z int32) int32 {
		return blocks[[3]int32{x, y, z}]
	}
	var boxes []inventory.BlockBox
	if inventory.IsWoodFenceGateBlock(blockID) {
		boxes, _ = inventory.FenceGateModelBoxes(blockID)
	} else if inventory.IsWoodDoorBlock(blockID) {
		boxes, _ = inventory.WoodDoorModelBoxes(blockID)
	} else if inventory.IsTrapdoorBlock(blockID) {
		boxes, _ = inventory.WoodTrapdoorModelBoxes(blockID)
	} else if inventory.IsIronChainBlock(blockID) {
		boxes, _ = inventory.IronChainModelBoxes(blockID)
	} else if inventory.IsPaneBlock(blockID) {
		boxes = inventory.PaneBoxesAt(0, 0, 0, neighbors)
	} else {
		boxes, _ = inventory.FenceCollisionBoxesAt(blockID, 0, 0, 0, neighbors)
	}
	if inventory.IsMasonryWallBlock(blockID) {
		boxes, _ = inventory.MasonryWallCollisionBoxesAt(blockID, 0, 0, 0, neighbors)
	}
	var out [16][16]uint32
	var depthBuffer [16][16]float32
	for y := range depthBuffer {
		for x := range depthBuffer[y] {
			depthBuffer[y][x] = float32(math.Inf(-1))
		}
	}
	for _, box := range boxes {
		for _, face := range []int{0, 2, 4} {
			positions, uvs := fenceIconFace(box, face)
			var quad [4]isoStairVertex
			for i, p := range positions {
				x, y, z := p[0], p[1], p[2]
				quad[i] = isoStairVertex{
					point: isoStairPoint{8 + (x-z)*6, 2 + (x+z)*3 + (1.5-y)*5},
					u:     uvs[i][0], v: uvs[i][1], depth: x + z + y*(4.0/3.0),
				}
			}
			shade := float32(1.05)
			tex := texture.Top
			if face == 0 {
				shade, tex = .62, texture.Side
			} else if face == 4 {
				shade, tex = .82, texture.Side
			}
			rasterIsoStairTriangle(&out, &depthBuffer, quad[0], quad[1], quad[2], tex, shade)
			rasterIsoStairTriangle(&out, &depthBuffer, quad[0], quad[2], quad[3], tex, shade)
		}
	}
	return out
}

func fenceIconFace(box inventory.BlockBox, face int) ([4][3]float32, [4][2]float32) {
	x0, x1, y0, y1, z0, z1 := box.MinX, box.MaxX, box.MinY, box.MaxY, box.MinZ, box.MaxZ
	switch face {
	case 0:
		return [4][3]float32{{x1, y0, z0}, {x1, y1, z0}, {x1, y1, z1}, {x1, y0, z1}}, [4][2]float32{{z0, 1 - y0}, {z0, 1 - y1}, {z1, 1 - y1}, {z1, 1 - y0}}
	case 2:
		return [4][3]float32{{x0, y1, z0}, {x0, y1, z1}, {x1, y1, z1}, {x1, y1, z0}}, [4][2]float32{{x0, z0}, {x0, z1}, {x1, z1}, {x1, z0}}
	default:
		return [4][3]float32{{x1, y0, z1}, {x1, y1, z1}, {x0, y1, z1}, {x0, y0, z1}}, [4][2]float32{{1 - x1, 1 - y0}, {1 - x1, 1 - y1}, {1 - x0, 1 - y1}, {1 - x0, 1 - y0}}
	}
}

type isoStairPoint struct{ x, y float32 }
type isoStairVertex struct {
	point isoStairPoint
	u, v  float32
	depth float32
}

func renderIsometricStair16x16(state int32) [16][16]uint32 {
	itemID, _, _, ok := inventory.StairStateProperties(state)
	if !ok {
		return renderIsometricCube16x16(inventory.BlockItemForState(state))
	}
	texture := Textures[itemID]
	if texture == nil {
		return renderIsometricCube16x16(itemID)
	}
	var out [16][16]uint32
	var depthBuffer [16][16]float32
	for y := range depthBuffer {
		for x := range depthBuffer[y] {
			depthBuffer[y][x] = float32(math.Inf(-1))
		}
	}
	faces := []int{0, 2, 4} // +X, top, +Z are the visible isometric faces.
	for cy := 0; cy < 2; cy++ {
		for cz := 0; cz < 2; cz++ {
			for cx := 0; cx < 2; cx++ {
				if !inventory.StairVoxelOccupied(state, cx, cy, cz) {
					continue
				}
				for _, face := range faces {
					dx, dy, dz := 0, 0, 0
					switch face {
					case 0:
						dx = 1
					case 2:
						dy = 1
					case 4:
						dz = 1
					}
					if inventory.StairVoxelOccupied(state, cx+dx, cy+dy, cz+dz) {
						continue
					}
					x0, x1 := float32(cx)*.5, float32(cx+1)*.5
					y0, y1 := float32(cy)*.5, float32(cy+1)*.5
					z0, z1 := float32(cz)*.5, float32(cz+1)*.5
					var positions [4][3]float32
					switch face {
					case 0:
						positions = [4][3]float32{{x1, y0, z0}, {x1, y1, z0}, {x1, y1, z1}, {x1, y0, z1}}
					case 2:
						positions = [4][3]float32{{x0, y1, z0}, {x0, y1, z1}, {x1, y1, z1}, {x1, y1, z0}}
					case 4:
						positions = [4][3]float32{{x1, y0, z1}, {x1, y1, z1}, {x0, y1, z1}, {x0, y0, z1}}
					}
					var quad [4]isoStairVertex
					for i, p := range positions {
						x, y, z := p[0], p[1], p[2]
						u, v := x, z
						if face == 0 {
							u, v = z, 1-y
						} else if face == 4 {
							u, v = 1-x, 1-y
						}
						quad[i] = isoStairVertex{
							point: isoStairPoint{8 + (x-z)*7.5, 1 + (x+z)*4 + (1-y)*6},
							u:     u, v: v, depth: x + z + y*(4.0/3.0),
						}
					}
					shade := float32(1.05)
					tex := texture.Top
					if face == 0 {
						shade, tex = .62, texture.Side
					} else if face == 4 {
						shade, tex = .82, texture.Side
					}
					rasterIsoStairTriangle(&out, &depthBuffer, quad[0], quad[1], quad[2], tex, shade)
					rasterIsoStairTriangle(&out, &depthBuffer, quad[0], quad[2], quad[3], tex, shade)
				}
			}
		}
	}
	return out
}

func rasterIsoStairTriangle(out *[16][16]uint32, depthBuffer *[16][16]float32, a, b, c isoStairVertex, texture BlockTexture, shade float32) {
	minX := max(0, int(math.Floor(float64(min(a.point.x, min(b.point.x, c.point.x))))))
	maxX := min(15, int(math.Ceil(float64(max(a.point.x, max(b.point.x, c.point.x))))))
	minY := max(0, int(math.Floor(float64(min(a.point.y, min(b.point.y, c.point.y))))))
	maxY := min(15, int(math.Ceil(float64(max(a.point.y, max(b.point.y, c.point.y))))))
	area := isoStairEdge(a.point, b.point, c.point.x, c.point.y)
	if math.Abs(float64(area)) < 1e-7 {
		return
	}
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			x, y := float32(px)+.5, float32(py)+.5
			wa := isoStairEdge(b.point, c.point, x, y) / area
			wb := isoStairEdge(c.point, a.point, x, y) / area
			wc := isoStairEdge(a.point, b.point, x, y) / area
			if wa < -1e-5 || wb < -1e-5 || wc < -1e-5 {
				continue
			}
			depth := wa*a.depth + wb*b.depth + wc*c.depth
			if depth <= depthBuffer[py][px] {
				continue
			}
			u := wa*a.u + wb*b.u + wc*c.u
			v := wa*a.v + wb*b.v + wc*c.v
			sx := max(0, min(15, int(u*16)))
			sy := max(0, min(15, int(v*16)))
			depthBuffer[py][px] = depth
			out[py][px] = applyLighting(texture[sy][sx], shade)
		}
	}
}

func isoStairEdge(a, b isoStairPoint, x, y float32) float32 {
	return (x-a.x)*(b.y-a.y) - (y-a.y)*(b.x-a.x)
}

func renderIsometricPartialBlock16x16(blockID int32, box inventory.BlockBox) [16][16]uint32 {
	full := renderIsometricCube16x16(inventory.BlockItemForState(blockID))
	var out [16][16]uint32
	if box.MaxY <= .5 { // Bottom half: shift the top face down and retain the lower side faces.
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				face, _ := getIsometricVoxelRegion(x, y)
				if face == 2 || face == 3 {
					if y >= 10 {
						out[y][x] = full[y][x]
					}
				}
				shiftedY := y - 4
				if shiftedY >= 0 {
					if topFace, _ := getIsometricVoxelRegion(x, shiftedY); topFace == 1 {
						out[y][x] = full[shiftedY][x]
					}
				}
			}
		}
		return out
	}

	// Top half: retain the original top face and only the upper half of both
	// side faces, leaving the bottom plane open as in the Java slab model.
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			face, _ := getIsometricVoxelRegion(x, y)
			if face == 1 || ((face == 2 || face == 3) && y <= 10) {
				out[y][x] = full[y][x]
			}
		}
	}
	return out
}

func renderIsometricCube16x16(blockID int32) [16][16]uint32 {
	var s [16][16]uint32
	tex := Textures[blockID]
	topBase, leftBase, rightBase := getBlockBaseColors(blockID)

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			face, isBorder := getIsometricVoxelRegion(x, y)
			if face == 0 {
				continue
			}

			var baseCol uint32
			var light float32

			if tex != nil {
				var u, v int
				switch face {
				case 1: // Top Face
					u = int((float64(x-7) + float64(y-1)*1.5) * 15.0 / 12.0)
					v = int((float64(7-x) + float64(y-1)*1.5) * 15.0 / 12.0)
					if u < 0 {
						u = 0
					} else if u > 15 {
						u = 15
					}
					if v < 0 {
						v = 0
					} else if v > 15 {
						v = 15
					}
					baseCol = tex.Top[v][u]
					light = 1.05
				case 2: // Left Face
					u = int(float64(x-1) * 15.0 / 6.0)
					v = int(float64(y-5) * 15.0 / 6.0)
					if u < 0 {
						u = 0
					} else if u > 15 {
						u = 15
					}
					if v < 0 {
						v = 0
					} else if v > 15 {
						v = 15
					}
					baseCol = tex.Side[v][u]
					light = 0.82
				case 3: // Right Face
					u = int(float64(x-7) * 15.0 / 7.0)
					v = int(float64(y-9) * 15.0 / 6.0)
					if u < 0 {
						u = 0
					} else if u > 15 {
						u = 15
					}
					if v < 0 {
						v = 0
					} else if v > 15 {
						v = 15
					}
					baseCol = tex.Side[v][u]
					light = 0.62
				}
			} else {
				switch face {
				case 1: // Top Face
					baseCol = sampleBlockSurface(blockID, 1, x, y, topBase)
					light = 1.05
				case 2: // Left Face
					baseCol = sampleBlockSurface(blockID, 2, x, y, leftBase)
					light = 0.82
				case 3: // Right Face
					baseCol = sampleBlockSurface(blockID, 3, x, y, rightBase)
					light = 0.62
				}
			}

			if isBorder {
				light *= 0.45 // Outer shadow bevel
			}

			s[y][x] = applyLighting(baseCol, light)
		}
	}

	return s
}

// getIsometricVoxelRegion returns (faceID: 0=none, 1=top, 2=left, 3=right, isBorder bool).
func getIsometricVoxelRegion(x, y int) (int, bool) {
	// 16x16 bounds with vertices:
	// Top: (7.5, 1)
	// Left: (1, 5)
	// Right: (14, 5)
	// Center: (7.5, 9)
	// BotLeft: (1, 11)
	// BotRight: (14, 11)
	// Bot: (7.5, 15)

	if y < 1 || y > 15 || x < 1 || x > 14 {
		return 0, false
	}

	// 1. Top face boundaries
	// upper diagonals:
	// y - 1 >= 0.57 * (7 - x) for x <= 7
	// y - 1 >= 0.57 * (x - 8) for x >= 8
	// lower diagonals:
	// y <= 5 + 0.57 * x for x <= 7
	// y <= 5 + 0.57 * (15 - x) for x >= 8

	distTopLeft := float64(y-1) - 0.58*float64(7-x)
	distTopRight := float64(y-1) - 0.58*float64(x-8)
	distMidLeft := float64(y-5) - 0.58*float64(x-1)
	distMidRight := float64(y-5) - 0.58*float64(14-x)

	if x <= 7 {
		if distTopLeft >= -0.3 && distMidLeft <= 0.3 {
			// Inside Top Face
			isB := distTopLeft < 0.7 || distMidLeft > -0.7 || x <= 1
			return 1, isB
		}
	} else {
		if distTopRight >= -0.3 && distMidRight <= 0.3 {
			// Inside Top Face
			isB := distTopRight < 0.7 || distMidRight > -0.7 || x >= 14
			return 1, isB
		}
	}

	// 2. Below Top Face: Left and Right faces
	distBotLeft := float64(y-11) - 0.58*float64(x-1)
	distBotRight := float64(y-11) - 0.58*float64(14-x)

	if x <= 7 {
		if distMidLeft > 0.1 && distBotLeft <= 0.4 && y <= 15 {
			isB := x == 1 || y == 15 || distBotLeft > -0.6
			return 2, isB
		}
	} else {
		if distMidRight > 0.1 && distBotRight <= 0.4 && y <= 15 {
			isB := x == 14 || y == 15 || distBotRight > -0.6
			return 3, isB
		}
	}

	return 0, false
}

// getBlockBaseColors returns base colors for (top, left, right) faces.
func getBlockBaseColors(blockID int32) (uint32, uint32, uint32) {
	switch blockID {
	case inventory.ItemGrass:
		return 0xFF5AA53C, 0xFF866043, 0xFF866043
	case inventory.ItemStone:
		return 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D
	case inventory.ItemCobblestone:
		return 0xFF6C6C6C, 0xFF6C6C6C, 0xFF6C6C6C
	case inventory.ItemDirt:
		return 0xFF866043, 0xFF866043, 0xFF866043
	case inventory.ItemOakPlanks:
		return 0xFFB8945F, 0xFFB8945F, 0xFFB8945F
	case inventory.ItemOakLog:
		return 0xFF9E7C4D, 0xFF4D3B26, 0xFF4D3B26
	case 18: // Oak Leaves
		return 0xFF358220, 0xFF2F741C, 0xFF2A6618
	case 20: // Glass
		return 0xCCB5E4F5, 0xCCB5E4F5, 0xCCB5E4F5
	case 12: // Sand
		return 0xFFDBD3A0, 0xFFDBD3A0, 0xFFDBD3A0
	case 13: // Gravel
		return 0xFF7E7777, 0xFF7E7777, 0xFF7E7777
	case 14: // Gold Ore
		return 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D
	case 15: // Iron Ore
		return 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D
	case 16: // Coal Ore
		return 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D
	case 56: // Diamond Ore
		return 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D
	case 7: // Bedrock
		return 0xFF2B2B2B, 0xFF2B2B2B, 0xFF2B2B2B
	case inventory.ItemCraftingTable:
		return 0xFFB8945F, 0xFF865F35, 0xFF865F35
	case inventory.ItemFurnace:
		return 0xFF6C6C6C, 0xFF555555, 0xFF555555
	case 45: // Bricks
		return 0xFF9C4133, 0xFF9C4133, 0xFF9C4133
	case 46: // TNT
		return 0xFF888888, 0xFFD62A20, 0xFFD62A20
	case 47: // Bookshelf
		return 0xFFB8945F, 0xFF8A5F35, 0xFF8A5F35
	case 49: // Obsidian
		return 0xFF18122B, 0xFF18122B, 0xFF18122B
	case 35: // White Wool
		return 0xFFECECEC, 0xFFECECEC, 0xFFECECEC
	default:
		return 0xFF888888, 0xFF777777, 0xFF666666
	}
}

// sampleBlockSurface provides distinctive texture details for each block.
func sampleBlockSurface(blockID int32, face, x, y int, baseCol uint32) uint32 {
	noise := (x*37 + y*17 + int(blockID)*13) % 7

	switch blockID {
	case inventory.ItemGrass:
		if face == 1 {
			// Top face: vibrant grass
			if noise == 0 {
				return 0xFF6CBF45
			} else if noise == 1 {
				return 0xFF46862E
			}
			return 0xFF5AA53C
		}
		// Sides: dirt with authentic green grass overhang at top
		relY := y - 5
		if relY <= 2 || (relY == 3 && (x%3 == 0)) {
			return 0xFF5AA53C // Green grass drip
		}
		if noise == 0 {
			return 0xFF6F4F37
		} else if noise == 1 {
			return 0xFF9B6F4E
		}
		return 0xFF866043

	case inventory.ItemOakLog:
		if face == 1 {
			// Tree ring
			distCenter := (x-7)*(x-7) + (y-5)*(y-5)
			if distCenter <= 2 {
				return 0xFFB8945F // Heartwood
			} else if distCenter <= 12 {
				return 0xFF9E7C4D // Rings
			}
			return 0xFF4D3B26 // Bark outer ring
		}
		// Side bark: vertical stripes
		if x%2 == 0 {
			return 0xFF4D3B26
		}
		return 0xFF624B31

	case inventory.ItemCobblestone:
		if noise == 0 || noise == 2 {
			return 0xFF888888
		} else if noise == 3 || noise == 5 {
			return 0xFF505050
		}
		return 0xFF6C6C6C

	case inventory.ItemStone:
		if noise == 0 {
			return 0xFF969696
		} else if noise == 1 {
			return 0xFF656565
		}
		return 0xFF7D7D7D

	case 56: // Diamond Ore
		if (x == 5 && y == 4) || (x == 9 && y == 3) || (x == 4 && y == 9) || (x == 11 && y == 11) {
			return 0xFF8CFFEE // Diamond glint
		}
		if (x >= 4 && x <= 6 && y >= 3 && y <= 5) || (x >= 3 && x <= 5 && y >= 8 && y <= 10) || (x >= 10 && x <= 12 && y >= 10 && y <= 12) {
			return 0xFF4AEDD7 // Diamond cyan
		}
		return 0xFF7D7D7D

	case 15: // Iron Ore
		if (x >= 4 && x <= 6 && y >= 3 && y <= 5) || (x >= 9 && x <= 11 && y >= 8 && y <= 10) {
			return 0xFFD8AF93
		}
		return 0xFF7D7D7D

	case 16: // Coal Ore
		if (x >= 4 && x <= 6 && y >= 3 && y <= 5) || (x >= 9 && x <= 11 && y >= 8 && y <= 10) {
			return 0xFF242424
		}
		return 0xFF7D7D7D

	case 14: // Gold Ore
		if (x >= 4 && x <= 6 && y >= 3 && y <= 5) || (x >= 9 && x <= 11 && y >= 8 && y <= 10) {
			return 0xFFFFDF40
		}
		return 0xFF7D7D7D

	case inventory.ItemOakPlanks:
		if y%3 == 0 {
			return 0xFF8F6E3E // Horizontal plank seam
		}
		return 0xFFB8945F

	case 20: // Glass
		if (x == 4 && y == 3) || (x == 5 && y == 4) || (x == 6 && y == 5) ||
			(x == 9 && y == 4) || (x == 10 && y == 5) || (x == 11 && y == 6) {
			return 0xFFFFFFFF // Glint
		}
		return 0xCCB5E4F5

	case inventory.ItemCraftingTable:
		if face == 1 {
			// Top 3x3 crafting grid
			if x == 5 || x == 10 || y == 3 || y == 7 {
				return 0xFF8F6E3E
			}
			return 0xFFB8945F
		}
		// Sides: tools hanging
		if (face == 2 && x >= 3 && x <= 5 && y >= 8 && y <= 12) || (face == 3 && x >= 9 && x <= 11 && y >= 8 && y <= 12) {
			return 0xFF555555 // Tool
		}
		return 0xFF865F35

	case inventory.ItemFurnace:
		if face == 2 && x >= 3 && x <= 6 && y >= 8 && y <= 12 {
			if y >= 11 {
				return 0xFFFF6600 // Glowing ember
			}
			return 0xFF1C1C1C // Dark mouth
		}
		return 0xFF6C6C6C

	case 46: // TNT
		if face != 1 {
			if y >= 7 && y <= 9 {
				if x == 4 || x == 11 {
					return 0xFF000000 // Letters
				}
				return 0xFFFFFFFF // White band
			}
			return 0xFFD62A20
		}
		return 0xFF888888
	}

	return baseCol
}

// GetItemSprite16x16 returns the authentic 16x16 pixel art sprite for non-block tools and items.
func GetItemSprite16x16(itemID int32) [16][16]uint32 {
	// 1. First priority: Load 100% authentic official vanilla Mojang texture if available
	if official, ok := GetOfficialItemSprite(itemID); ok {
		return official
	}
	if itemID == inventory.ItemChest {
		return generateChestItemSprite()
	}

	// 2. If placeable voxel block, render 3D isometric cube using block textures
	if inventory.IsPlaceableBlock(itemID) {
		return RenderIsometricBlock16x16(itemID)
	}

	// 3. Fallback to procedural generator
	props, isTool := inventory.GetToolProperties(itemID)
	if isTool {
		return generateToolSprite(props.Type, props.Tier)
	}

	switch itemID {
	// 1. Food items
	case inventory.ItemApple:
		return generateAppleSprite()
	case inventory.ItemBread:
		return generateBreadSprite()
	case inventory.ItemRawPorkchop:
		return generatePorkchopSprite(false)
	case inventory.ItemCookedPorkchop:
		return generatePorkchopSprite(true)
	case inventory.ItemRawBeef:
		return generateBeefSprite(false)
	case inventory.ItemSteak:
		return generateBeefSprite(true)
	case inventory.ItemRottenFlesh:
		return generateRottenFleshSprite()

	// 2. Materials & Minerals
	case inventory.ItemDiamond:
		return generateDiamondSprite()
	case inventory.ItemCoal:
		return generateCoalSprite()
	case inventory.ItemIronIngot:
		return generateIngotSprite(0xFFD8D8D8, 0xFFFFFFFF, 0xFFA0A0A0)
	case 266: // Gold Ingot
		return generateIngotSprite(0xFFFFDF40, 0xFFFFF6A0, 0xFFC8A010)
	case inventory.ItemStick:
		return generateStickSprite()
	case inventory.ItemTorch:
		return generateTorchSprite()
	case 331: // Redstone Dust
		return generateRedstoneSprite()
	case 352: // Bone
		return generateBoneSprite()
	case 288: // Feather
		return generateFeatherSprite()
	case 289: // Gunpowder
		return generateGunpowderSprite()
	case 334: // Leather
		return generateLeatherSprite()
	case 287: // String
		return generateStringSprite()
	case 261: // Bow
		return generateBowSprite()
	case 262: // Arrow
		return generateArrowSprite()

	// 3. Armor (Helmets, Chestplates, Leggings, Boots)
	case 306, 310, 298, 314: // Helmets (Iron, Diamond, Leather, Gold)
		return generateHelmetSprite(itemID)
	case 307, 311, 299, 315: // Chestplates
		return generateChestplateSprite(itemID)
	case 308, 312, 300, 316: // Leggings
		return generateLeggingsSprite(itemID)
	case 309, 313, 301, 317: // Boots
		return generateBootsSprite(itemID)

	default:
		// If placeable voxel block, render 3D isometric cube!
		if inventory.IsPlaceableBlock(itemID) {
			return RenderIsometricBlock16x16(itemID)
		}
		// Fallback simple icon
		return generateStickSprite()
	}
}

type chestModelPoint struct {
	x, y, z float64
}

type chestTexturePoint struct {
	u, v float64
}

type chestScreenPoint struct {
	x, y float64
}

// generateChestItemSprite projects the Java chest block-entity model into the
// same 16x16 isometric item-icon space used for block items. Its UV rectangles
// follow ChestModel.createSingleBodyLayer in the 26.3 client model.
func generateChestItemSprite() [16][16]uint32 {
	var out [16][16]uint32
	texture, ok := GetOfficialImage("entity/chest/normal.png")
	if !ok || texture.Width != 64 || texture.Height != 64 {
		return RenderIsometricBlock16x16(inventory.ItemChest)
	}
	project := func(point chestModelPoint) chestScreenPoint {
		return chestScreenPoint{
			x: 8 + (point.x-point.z)*0.52,
			y: 8 + (point.x+point.z-16)*0.18 + (15-point.y)*0.48,
		}
	}
	drawFace := func(points [4]chestModelPoint, uvs [4]chestTexturePoint, shade float64) {
		var screen [4]chestScreenPoint
		for i := range points {
			screen[i] = project(points[i])
		}
		drawChestTriangle(&out, texture, screen[0], screen[1], screen[2], uvs[0], uvs[1], uvs[2], shade)
		drawChestTriangle(&out, texture, screen[0], screen[2], screen[3], uvs[0], uvs[2], uvs[3], shade)
	}
	uv := func(u, v float64) chestTexturePoint { return chestTexturePoint{u: u, v: v} }
	// Lower chest body: 14 x 10 x 14, UV offset (0,19).
	// Render its two visible faces before the lid.
	drawFace([4]chestModelPoint{{15, 0, 15}, {15, 0, 1}, {15, 10, 1}, {15, 10, 15}}, [4]chestTexturePoint{uv(28, 33), uv(42, 33), uv(42, 43), uv(28, 43)}, 0.72)
	drawFace([4]chestModelPoint{{1, 0, 15}, {15, 0, 15}, {15, 10, 15}, {1, 10, 15}}, [4]chestTexturePoint{uv(42, 33), uv(56, 33), uv(56, 43), uv(42, 43)}, 0.9)
	// Lid top and sides: 14 x 5 x 14, UV offset (0,0).
	drawFace([4]chestModelPoint{{15, 14, 1}, {1, 14, 1}, {1, 14, 15}, {15, 14, 15}}, [4]chestTexturePoint{uv(28, 14), uv(42, 14), uv(42, 0), uv(28, 0)}, 1)
	drawFace([4]chestModelPoint{{15, 9, 15}, {15, 9, 1}, {15, 14, 1}, {15, 14, 15}}, [4]chestTexturePoint{uv(28, 14), uv(42, 14), uv(42, 19), uv(28, 19)}, 0.72)
	drawFace([4]chestModelPoint{{1, 9, 15}, {15, 9, 15}, {15, 14, 15}, {1, 14, 15}}, [4]chestTexturePoint{uv(42, 14), uv(56, 14), uv(56, 19), uv(42, 19)}, 0.9)
	// Java's lock is a separate 2x4x1 cuboid at texOffs(0,0), projected
	// on the chest's south/front face.
	drawFace([4]chestModelPoint{{7, 7, 16}, {9, 7, 16}, {9, 11, 16}, {7, 11, 16}}, [4]chestTexturePoint{uv(4, 1), uv(6, 1), uv(6, 5), uv(4, 5)}, 0.9)
	return out
}

func drawChestTriangle(out *[16][16]uint32, texture *RawImage, a, b, c chestScreenPoint, ua, ub, uc chestTexturePoint, shade float64) {
	denom := (b.y-c.y)*(a.x-c.x) + (c.x-b.x)*(a.y-c.y)
	if denom == 0 {
		return
	}
	minX := max(0, int(math.Floor(math.Min(a.x, math.Min(b.x, c.x)))))
	maxX := min(15, int(math.Ceil(math.Max(a.x, math.Max(b.x, c.x)))))
	minY := max(0, int(math.Floor(math.Min(a.y, math.Min(b.y, c.y)))))
	maxY := min(15, int(math.Ceil(math.Max(a.y, math.Max(b.y, c.y)))))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			wa := ((b.y-c.y)*(px-c.x) + (c.x-b.x)*(py-c.y)) / denom
			wb := ((c.y-a.y)*(px-c.x) + (a.x-c.x)*(py-c.y)) / denom
			wc := 1 - wa - wb
			if wa < -1e-6 || wb < -1e-6 || wc < -1e-6 {
				continue
			}
			u := wa*ua.u + wb*ub.u + wc*uc.u
			v := wa*ua.v + wb*ub.v + wc*uc.v
			tx, ty := int(u), int(v)
			if tx < 0 || tx >= texture.Width || ty < 0 || ty >= texture.Height {
				continue
			}
			color := texture.Pixels[ty*texture.Width+tx]
			if color>>24 == 0 {
				continue
			}
			out[y][x] = shadeChestColor(color, shade)
		}
	}
}

func shadeChestColor(color uint32, shade float64) uint32 {
	a := color & 0xFF000000
	r := uint32(float64((color>>16)&0xFF) * shade)
	g := uint32(float64((color>>8)&0xFF) * shade)
	b := uint32(float64(color&0xFF) * shade)
	return a | r<<16 | g<<8 | b
}

// generateToolSprite creates authentic 45-degree angled tools with shaded bevels.
func generateToolSprite(t inventory.ToolType, tier inventory.ToolTier) [16][16]uint32 {
	var s [16][16]uint32
	stickCol := uint32(0xFF8F6849)
	stickShade := uint32(0xFF5C3E28)
	borderCol := uint32(0xFF281C12)

	headCol := uint32(0xFFB8945F) // Wood
	shineCol := uint32(0xFFD4B07B)
	darkCol := uint32(0xFF8A6B3A)

	switch tier {
	case inventory.TierStone:
		headCol = 0xFF7D7D7D
		shineCol = 0xFF9E9E9E
		darkCol = 0xFF545454
	case inventory.TierIron:
		headCol = 0xFFD8D8D8
		shineCol = 0xFFFFFFFF
		darkCol = 0xFFA0A0A0
	case inventory.TierDiamond:
		headCol = 0xFF4AEDD7
		shineCol = 0xFF8CFFEE
		darkCol = 0xFF28A896
	case inventory.TierGold:
		headCol = 0xFFFFDF40
		shineCol = 0xFFFFF6A0
		darkCol = 0xFFC8A010
	}

	// 1. Diagonal handle from (2, 13) to (9, 6)
	for d := 1; d <= 8; d++ {
		x := 1 + d
		y := 14 - d
		s[y][x] = stickCol
		if x > 1 {
			s[y][x-1] = stickShade
		}
	}

	switch t {
	case inventory.ToolSword:
		// Pommel & grip
		s[14][1] = darkCol
		s[13][2] = stickCol
		s[12][3] = stickCol
		// Crossguard
		s[11][2] = darkCol
		s[11][3] = darkCol
		s[11][4] = headCol
		s[12][4] = darkCol
		s[10][4] = headCol
		s[10][5] = darkCol
		s[9][5] = headCol
		s[10][3] = darkCol
		// Blade: diagonal spine from (5, 9) to (13, 1)
		for d := 0; d < 7; d++ {
			bx := 6 + d
			by := 8 - d
			s[by][bx] = headCol
			s[by-1][bx] = shineCol
			s[by][bx-1] = darkCol
			s[by-1][bx-1] = borderCol
		}
		s[1][13] = shineCol // Blade tip
		s[2][14] = darkCol
		s[1][14] = borderCol

	case inventory.ToolPickaxe:
		// Curved pickaxe arch
		s[5][10] = darkCol
		s[5][11] = headCol
		s[4][11] = shineCol
		s[4][12] = shineCol
		s[3][12] = headCol
		s[3][13] = shineCol
		s[2][13] = headCol
		s[2][14] = darkCol
		s[1][14] = shineCol // Right pick tip

		s[6][9] = headCol
		s[7][8] = headCol
		s[8][7] = headCol
		s[9][6] = darkCol
		s[10][5] = headCol
		s[11][4] = headCol
		s[12][3] = headCol
		s[13][2] = darkCol
		s[14][1] = shineCol // Left pick tip

		// Center reinforcing plate
		s[6][8] = darkCol
		s[5][9] = shineCol
		s[4][10] = headCol

	case inventory.ToolAxe:
		// Broad axe head
		s[4][9] = darkCol
		s[3][9] = shineCol
		s[3][10] = shineCol
		s[2][10] = shineCol
		s[2][11] = headCol
		s[2][12] = headCol
		s[3][12] = darkCol
		s[4][12] = darkCol
		s[5][11] = darkCol
		s[5][10] = headCol
		s[4][10] = headCol
		s[4][11] = shineCol

		s[5][8] = darkCol
		s[6][8] = headCol
		s[6][7] = headCol
		s[7][7] = darkCol
		s[7][6] = darkCol
		s[8][6] = headCol
		s[8][5] = darkCol

	case inventory.ToolShovel:
		// Spade head
		s[5][9] = darkCol
		s[4][10] = headCol
		s[3][11] = headCol
		s[2][12] = shineCol
		s[1][13] = shineCol // Tip
		s[3][10] = headCol
		s[2][11] = shineCol
		s[4][11] = darkCol
		s[3][12] = darkCol
		s[2][13] = darkCol
		s[1][12] = shineCol
	}

	return s
}

func generateAppleSprite() [16][16]uint32 {
	var s [16][16]uint32
	red := uint32(0xFFD82020)
	darkRed := uint32(0xFF9E1010)
	glint := uint32(0xFFFFA090)
	stem := uint32(0xFF5C3A21)
	leaf := uint32(0xFF5CB832)

	// Stem & leaf
	s[1][8] = stem
	s[2][8] = stem
	s[1][9] = leaf
	s[2][10] = leaf

	// Apple body
	for y := 4; y <= 12; y++ {
		for x := 4; x <= 12; x++ {
			s[y][x] = red
		}
	}
	// Rounded corners
	s[4][4] = 0
	s[4][12] = 0
	s[12][4] = 0
	s[12][12] = 0
	s[4][8] = 0
	s[12][8] = 0

	// Highlights and shadows
	s[6][5] = glint
	s[7][5] = glint
	s[6][6] = glint
	s[11][5] = darkRed
	s[11][11] = darkRed
	s[10][11] = darkRed
	return s
}

func generateBreadSprite() [16][16]uint32 {
	var s [16][16]uint32
	crust := uint32(0xFF8A5D20)
	bread := uint32(0xFFC99742)
	lightBread := uint32(0xFFE5BC68)

	for y := 5; y <= 11; y++ {
		for x := 2; x <= 13; x++ {
			s[y][x] = bread
		}
	}
	s[5][2] = 0
	s[5][13] = 0
	s[11][2] = 0
	s[11][13] = 0

	// Top crust highlights
	for x := 4; x <= 11; x++ {
		s[5][x] = lightBread
	}

	// 3 Diagonal slash marks
	for _, sx := range []int{4, 7, 10} {
		s[7][sx] = crust
		s[8][sx+1] = crust
		s[9][sx+1] = crust
	}
	return s
}

func generatePorkchopSprite(cooked bool) [16][16]uint32 {
	var s [16][16]uint32
	bodyCol := uint32(0xFFE89098)
	rindCol := uint32(0xFFF5B5BC)
	boneCol := uint32(0xFFECE7E2)
	shadeCol := uint32(0xFFB86068)

	if cooked {
		bodyCol = 0xFF9E523A
		rindCol = 0xFFBD6D4E
		shadeCol = 0xFF6D3220
	}

	for y := 4; y <= 12; y++ {
		for x := 3; x <= 12; x++ {
			s[y][x] = bodyCol
		}
	}
	s[4][3] = 0
	s[4][12] = 0
	s[12][3] = 0
	s[12][12] = 0

	// Rind highlight
	for x := 4; x <= 11; x++ {
		s[4][x] = rindCol
	}
	// Bone
	s[11][12] = boneCol
	s[12][13] = boneCol
	s[10][13] = boneCol

	// Shading
	for x := 4; x <= 10; x++ {
		s[11][x] = shadeCol
	}
	return s
}

func generateBeefSprite(cooked bool) [16][16]uint32 {
	var s [16][16]uint32
	meatCol := uint32(0xFFB82424)
	fatCol := uint32(0xFFECE7E2)
	shadeCol := uint32(0xFF781414)

	if cooked {
		meatCol = 0xFF703C28
		fatCol = 0xFFBD8E70
		shadeCol = 0xFF4A2416
	}

	for y := 4; y <= 12; y++ {
		for x := 3; x <= 12; x++ {
			s[y][x] = meatCol
		}
	}
	s[4][3] = 0
	s[4][12] = 0
	s[12][3] = 0
	s[12][12] = 0

	// Fat marbling ribbon
	s[6][5] = fatCol
	s[7][6] = fatCol
	s[8][7] = fatCol
	s[9][8] = fatCol

	// Shading
	for x := 4; x <= 11; x++ {
		s[12][x] = shadeCol
	}
	return s
}

func generateRottenFleshSprite() [16][16]uint32 {
	var s [16][16]uint32
	zombieMeat := uint32(0xFF4C6A2E)
	darkSpot := uint32(0xFF2C3E18)
	highlight := uint32(0xFF6E8E42)

	for y := 4; y <= 12; y++ {
		for x := 3; x <= 12; x++ {
			if (x+y)%3 == 0 {
				s[y][x] = darkSpot
			} else {
				s[y][x] = zombieMeat
			}
		}
	}
	s[4][3] = 0
	s[4][12] = 0
	s[12][3] = 0
	s[12][12] = 0
	s[5][6] = highlight
	s[6][7] = highlight
	return s
}

func generateDiamondSprite() [16][16]uint32 {
	var s [16][16]uint32
	base := uint32(0xFF4AEDD7)
	shine := uint32(0xFF8CFFEE)
	dark := uint32(0xFF28A896)
	outline := uint32(0xFF16685C)

	// Iconic 8-sided faceted gemstone
	// Top point at (7, 2)-(8, 2), widening to (3, 6)-(12, 6), tapering to bottom (7, 13)-(8, 13)
	for y := 2; y <= 13; y++ {
		minX := 7
		maxX := 8
		if y <= 6 {
			minX = 7 - (y - 2)
			maxX = 8 + (y - 2)
		} else {
			minX = 3 + (y-6)*4/7
			maxX = 12 - (y-6)*4/7
		}
		for x := minX; x <= maxX; x++ {
			s[y][x] = base
		}
	}

	// Facet lines & glints
	s[4][6] = shine
	s[4][7] = shine
	s[5][5] = shine
	s[5][6] = shine

	for y := 7; y <= 12; y++ {
		s[y][7] = shine
		s[y][10] = dark
	}

	// Dark bottom facets
	s[12][7] = outline
	s[12][8] = outline
	s[13][7] = outline
	s[13][8] = outline

	return s
}

func generateCoalSprite() [16][16]uint32 {
	var s [16][16]uint32
	coalDark := uint32(0xFF1E1E1E)
	coalMid := uint32(0xFF333333)
	coalShine := uint32(0xFF555555)

	for y := 4; y <= 12; y++ {
		for x := 4; x <= 12; x++ {
			s[y][x] = coalDark
		}
	}
	s[4][4] = 0
	s[4][11] = 0
	s[12][4] = 0
	s[12][12] = 0

	// Facets
	s[6][6] = coalShine
	s[6][7] = coalMid
	s[7][6] = coalMid
	s[9][8] = coalShine
	s[10][9] = coalMid
	return s
}

func generateIngotSprite(midCol, shineCol, darkCol uint32) [16][16]uint32 {
	var s [16][16]uint32

	// Angled 3D metallic ingot bar
	for y := 6; y <= 11; y++ {
		for x := 3; x <= 13; x++ {
			s[y][x] = midCol
		}
	}
	s[6][3] = 0
	s[11][13] = 0

	// Top chamfer (shine)
	for x := 4; x <= 13; x++ {
		s[6][x] = shineCol
	}
	s[7][4] = shineCol

	// Bottom shadow
	for x := 3; x <= 12; x++ {
		s[11][x] = darkCol
	}
	s[10][12] = darkCol
	return s
}

func generateStickSprite() [16][16]uint32 {
	var s [16][16]uint32
	stickCol := uint32(0xFF8F6849)
	shadeCol := uint32(0xFF5C3E28)

	for d := 2; d <= 13; d++ {
		x := d
		y := 15 - d
		s[y][x] = stickCol
		if x > 2 {
			s[y][x-1] = shadeCol
		}
	}
	return s
}

func generateTorchSprite() [16][16]uint32 {
	var s [16][16]uint32
	stickCol := uint32(0xFF8F6849)
	shadeCol := uint32(0xFF5C3E28)
	flameCore := uint32(0xFFFFFFA0)
	flameOrange := uint32(0xFFFF8800)
	flameRed := uint32(0xFFD83010)

	// Wood post
	for y := 6; y <= 14; y++ {
		s[y][7] = shadeCol
		s[y][8] = stickCol
	}

	// Flame
	s[2][7] = flameOrange
	s[2][8] = flameCore
	s[3][6] = flameRed
	s[3][7] = flameCore
	s[3][8] = flameCore
	s[3][9] = flameOrange
	s[4][7] = flameOrange
	s[4][8] = flameOrange
	s[5][7] = 0xFF352B20 // Burnt tip
	s[5][8] = 0xFF453628
	return s
}

func generateRedstoneSprite() [16][16]uint32 {
	var s [16][16]uint32
	redBright := uint32(0xFFFF3333)
	redMid := uint32(0xFFCC1111)
	redDark := uint32(0xFF880000)

	for y := 8; y <= 13; y++ {
		for x := 4; x <= 12; x++ {
			if (x+y)%2 == 0 {
				s[y][x] = redBright
			} else {
				s[y][x] = redMid
			}
		}
	}
	s[8][4] = 0
	s[8][12] = 0
	s[13][4] = 0
	s[13][12] = 0
	s[9][8] = redBright
	s[12][8] = redDark
	return s
}

func generateBoneSprite() [16][16]uint32 {
	var s [16][16]uint32
	boneWhite := uint32(0xFFECE7E2)
	boneShade := uint32(0xFFC8C2BA)

	for d := 3; d <= 12; d++ {
		x := d
		y := 15 - d
		s[y][x] = boneWhite
		s[y][x-1] = boneShade
	}
	// Knobs at bottom-left
	s[14][2] = boneWhite
	s[13][1] = boneShade
	s[12][2] = boneWhite
	// Knobs at top-right
	s[3][13] = boneWhite
	s[2][14] = boneWhite
	s[4][14] = boneShade
	return s
}

func generateFeatherSprite() [16][16]uint32 {
	var s [16][16]uint32
	white := uint32(0xFFFFFFFF)
	shade := uint32(0xFFD4D4D4)
	shaft := uint32(0xFF9E8A78)

	for d := 3; d <= 12; d++ {
		x := d
		y := 15 - d
		s[y][x] = shaft
		if d < 12 {
			s[y-1][x] = white
			s[y-2][x] = white
			s[y][x+1] = shade
		}
	}
	return s
}

func generateGunpowderSprite() [16][16]uint32 {
	var s [16][16]uint32
	light := uint32(0xFF7A7A7A)
	mid := uint32(0xFF555555)
	dark := uint32(0xFF333333)

	for y := 8; y <= 13; y++ {
		for x := 4; x <= 12; x++ {
			if (x+y)%3 == 0 {
				s[y][x] = light
			} else if (x+y)%3 == 1 {
				s[y][x] = mid
			} else {
				s[y][x] = dark
			}
		}
	}
	s[8][4] = 0
	s[8][12] = 0
	s[13][4] = 0
	s[13][12] = 0
	return s
}

func generateLeatherSprite() [16][16]uint32 {
	var s [16][16]uint32
	leatherCol := uint32(0xFF9E5C32)
	lightCol := uint32(0xFFBD7442)
	darkCol := uint32(0xFF6B3A1C)

	for y := 4; y <= 12; y++ {
		for x := 4; x <= 12; x++ {
			s[y][x] = leatherCol
		}
	}
	s[4][4] = 0
	s[4][12] = 0
	s[12][4] = 0
	s[12][12] = 0
	s[6][6] = lightCol
	s[10][10] = darkCol
	return s
}

func generateStringSprite() [16][16]uint32 {
	var s [16][16]uint32
	white := uint32(0xFFFFFFFF)
	shade := uint32(0xFFC0C0C0)

	s[4][4] = white
	s[5][5] = white
	s[6][6] = shade
	s[7][7] = white
	s[8][8] = white
	s[9][9] = shade
	s[10][10] = white
	s[11][11] = white
	return s
}

func generateBowSprite() [16][16]uint32 {
	var s [16][16]uint32
	wood := uint32(0xFF8F6849)
	stringCol := uint32(0xFFFFFFFF)

	// Curved bow stave
	s[2][8] = wood
	s[3][7] = wood
	s[4][6] = wood
	s[5][5] = wood
	s[6][4] = wood
	s[7][4] = wood
	s[8][4] = wood
	s[9][5] = wood
	s[10][6] = wood
	s[11][7] = wood
	s[12][8] = wood

	// Taut string connecting tips
	for y := 3; y <= 11; y++ {
		s[y][9] = stringCol
	}
	return s
}

func generateArrowSprite() [16][16]uint32 {
	var s [16][16]uint32
	flint := uint32(0xFF666666)
	shaft := uint32(0xFF8F6849)
	feather := uint32(0xFFFFFFFF)

	// Tip
	s[2][13] = flint
	s[3][13] = flint
	s[3][12] = flint

	// Shaft
	for d := 4; d <= 11; d++ {
		s[15-d][d] = shaft
	}

	// Fletching feathers
	s[12][3] = feather
	s[13][3] = feather
	s[13][2] = feather
	return s
}

func generateHelmetSprite(itemID int32) [16][16]uint32 {
	var s [16][16]uint32
	col := uint32(0xFFD8D8D8) // Iron
	if itemID == 310 {
		col = 0xFF4AEDD7 // Diamond
	} else if itemID == 298 {
		col = 0xFF9E5C32 // Leather
	} else if itemID == 314 {
		col = 0xFFFFDF40 // Gold
	}

	for y := 4; y <= 11; y++ {
		for x := 3; x <= 12; x++ {
			s[y][x] = col
		}
	}
	// Eye cutouts
	s[9][5] = 0
	s[9][6] = 0
	s[9][9] = 0
	s[9][10] = 0
	// Open neck bottom
	s[11][6] = 0
	s[11][7] = 0
	s[11][8] = 0
	s[11][9] = 0
	return s
}

func generateChestplateSprite(itemID int32) [16][16]uint32 {
	var s [16][16]uint32
	col := uint32(0xFFD8D8D8)
	if itemID == 311 {
		col = 0xFF4AEDD7
	} else if itemID == 299 {
		col = 0xFF9E5C32
	} else if itemID == 315 {
		col = 0xFFFFDF40
	}

	for y := 3; y <= 13; y++ {
		for x := 3; x <= 12; x++ {
			s[y][x] = col
		}
	}
	// Neck opening
	s[3][6] = 0
	s[3][7] = 0
	s[3][8] = 0
	s[3][9] = 0
	s[4][7] = 0
	s[4][8] = 0
	// Armpit cutouts
	s[7][3] = 0
	s[7][12] = 0
	s[8][3] = 0
	s[8][12] = 0
	return s
}

func generateLeggingsSprite(itemID int32) [16][16]uint32 {
	var s [16][16]uint32
	col := uint32(0xFFD8D8D8)
	if itemID == 312 {
		col = 0xFF4AEDD7
	} else if itemID == 300 {
		col = 0xFF9E5C32
	} else if itemID == 316 {
		col = 0xFFFFDF40
	}

	for y := 3; y <= 13; y++ {
		for x := 4; x <= 11; x++ {
			s[y][x] = col
		}
	}
	// Crotch / leg separation
	for y := 7; y <= 13; y++ {
		s[y][7] = 0
		s[y][8] = 0
	}
	return s
}

func generateBootsSprite(itemID int32) [16][16]uint32 {
	var s [16][16]uint32
	col := uint32(0xFFD8D8D8)
	if itemID == 313 {
		col = 0xFF4AEDD7
	} else if itemID == 301 {
		col = 0xFF9E5C32
	} else if itemID == 317 {
		col = 0xFFFFDF40
	}

	// Left boot
	for y := 7; y <= 13; y++ {
		for x := 3; x <= 6; x++ {
			s[y][x] = col
		}
	}
	s[12][2] = col // Toe
	s[13][2] = col

	// Right boot
	for y := 7; y <= 13; y++ {
		for x := 9; x <= 12; x++ {
			s[y][x] = col
		}
	}
	s[12][13] = col // Toe
	s[13][13] = col
	return s
}

// DrawItemSprite16x16 draws an authentic 16x16 item sprite or 3D isometric block at (x, y).
func (fb *Framebuffer) DrawItemSprite16x16(x, y int, itemID int32) {
	sprite := GetItemSprite16x16(itemID)
	for sy := 0; sy < 16; sy++ {
		py := y + sy
		if py < 0 || py >= fb.Height {
			continue
		}
		rowOffset := py * fb.Width
		for sx := 0; sx < 16; sx++ {
			col := sprite[sy][sx]
			if (col >> 24) == 0 {
				continue
			}
			px := x + sx
			if px >= 0 && px < fb.Width {
				fb.Color[rowOffset+px] = col
			}
		}
	}
}
