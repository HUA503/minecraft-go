package gen

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/world"
)

// Dimension represents Minecraft dimensions.
type Dimension int

const (
	DimensionOverworld Dimension = iota
	DimensionNether
	DimensionEnd
)

// Block IDs catalog - aligned with canonical Minecraft Java Edition
const (
	BlockAir              int32 = 0
	BlockStone            int32 = 1
	BlockGrass            int32 = 2
	BlockDirt             int32 = 3
	BlockCobblestone      int32 = 4
	BlockOakPlanks        int32 = 5
	BlockBedrock          int32 = 7
	BlockWater            int32 = 8
	BlockFlowingWater     int32 = 9
	BlockLava             int32 = 10
	BlockFlowingLava      int32 = 11
	BlockSand             int32 = 12
	BlockGravel           int32 = 13
	BlockGoldOre          int32 = 14
	BlockIronOre          int32 = 15
	BlockCoalOre          int32 = 16
	BlockOakLog           int32 = 17
	BlockOakLeaves        int32 = 18
	BlockBirchLog         int32 = 19
	BlockGlass            int32 = 20
	BlockBirchLeaves      int32 = 21
	BlockSpruceLog        int32 = 22
	BlockSpruceLeaves     int32 = 23
	BlockTallGrass        int32 = 31
	BlockDeadBush         int32 = 32
	BlockFern             int32 = 33
	BlockDandelion        int32 = 37
	BlockPoppy            int32 = 38
	BlockMossyCobblestone int32 = 48
	BlockObsidian         int32 = 49
	BlockTorch            int32 = 50
	BlockMonsterSpawner   int32 = 52
	BlockChest            int32 = 54
	BlockDiamondOre       int32 = 56
	BlockCraftingTable    int32 = 58
	BlockFurnace          int32 = 61
	BlockRedstoneOre      int32 = 73
	BlockLapisOre         int32 = 74
	BlockCactus           int32 = 81
	BlockNetherrack       int32 = 87
	BlockSoulSand         int32 = 88
	BlockGlowstone        int32 = 89
	BlockNetherPortal     int32 = 90
	BlockLilyPad          int32 = 111
	BlockEndPortal        int32 = 119
	BlockEndPortalFrame   int32 = 120
	BlockEndStone         int32 = 121
	BlockDragonEgg        int32 = 122
	BlockDeepslate        int32 = 150
	BlockCopperOre        int32 = 153
	BlockNetherQuartzOre  int32 = 154
)

// ColInfo stores column height and biome information.
type ColInfo struct {
	Height int32
	Biome  BiomeType
}

// BiomeType defines ecological terrain types.
type BiomeType int

const (
	BiomePlains BiomeType = iota
	BiomeForest
	BiomeBirchForest
	BiomeTaiga
	BiomeDesert
	BiomeSwamp
	BiomeMountains
	BiomeOcean
)

// Generator produces procedural voxel chunks.
type Generator struct {
	Seed      int64
	Dimension Dimension
	noise     *PerlinNoise
}

func NewGenerator(seed int64, dim Dimension) *Generator {
	return &Generator{
		Seed:      seed,
		Dimension: dim,
		noise:     NewPerlinNoise(seed),
	}
}

// GenerateChunk fills a 16x384x16 Chunk with procedural terrain based on dimension.
func (g *Generator) GenerateChunk(cp world.ChunkPos) *world.Chunk {
	c := world.NewChunk(cp)

	switch g.Dimension {
	case DimensionOverworld:
		g.generateOverworld(c, cp)
	case DimensionNether:
		g.generateNether(c, cp)
	case DimensionEnd:
		g.generateEnd(c, cp)
	}

	return c
}

// SampleBiome returns the authoritative biome for world coordinate (wx, wz).
func (g *Generator) SampleBiome(wx, wz float64) BiomeType {
	continent := g.noise.Noise2D(wx*0.003, wz*0.003)
	temp := g.noise.Noise2D(wx*0.002+100.0, wz*0.002+100.0)
	humid := g.noise.Noise2D(wx*0.002-100.0, wz*0.002-100.0)

	if continent < -0.45 {
		return BiomeOcean
	}
	if continent > 0.42 {
		return BiomeMountains
	}
	if temp > 0.35 && humid < -0.15 {
		return BiomeDesert
	}
	if temp < -0.25 {
		return BiomeTaiga
	}
	if humid > 0.38 && temp > 0.05 {
		return BiomeSwamp
	}
	if humid > 0.15 {
		if temp < 0.1 {
			return BiomeBirchForest
		}
		return BiomeForest
	}
	return BiomePlains
}

func (g *Generator) generateOverworld(c *world.Chunk, cp world.ChunkPos) {
	const seaLevel = 62
	worldStartX := cp.X * 16
	worldStartZ := cp.Z * 16

	// 1. Terrain Height & Surface Fill
	var cols [16][16]ColInfo

	for lx := int32(0); lx < 16; lx++ {
		for lz := int32(0); lz < 16; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)

			biome := g.SampleBiome(wx, wz)

			var height int32
			switch biome {
			case BiomeMountains:
				mElev := g.noise.FBM2D(wx*0.015, wz*0.015, 5, 0.55, 2.0)
				height = int32(76 + mElev*34)
			case BiomeDesert:
				dElev := g.noise.FBM2D(wx*0.012, wz*0.012, 3, 0.4, 2.0)
				height = int32(65 + dElev*10)
			case BiomeSwamp:
				sElev := g.noise.FBM2D(wx*0.015, wz*0.015, 2, 0.3, 2.0)
				height = int32(62 + sElev*4)
			case BiomeOcean:
				oElev := g.noise.FBM2D(wx*0.01, wz*0.01, 3, 0.5, 2.0)
				height = int32(46 + oElev*12)
			case BiomeTaiga:
				tElev := g.noise.FBM2D(wx*0.014, wz*0.014, 4, 0.5, 2.0)
				height = int32(66 + tElev*18)
			case BiomeForest, BiomeBirchForest:
				fElev := g.noise.FBM2D(wx*0.014, wz*0.014, 4, 0.5, 2.0)
				height = int32(65 + fElev*16)
			default: // BiomePlains
				elevation := g.noise.FBM2D(wx*0.015, wz*0.015, 4, 0.5, 2.0)
				height = int32(64 + elevation*14)
			}
			cols[lx][lz] = ColInfo{Height: height, Biome: biome}

			// Bedrock floor at -64
			c.SetBlock(toI32(wx), -64, toI32(wz), BlockBedrock)

			// Deepslate layer (-63 to -1)
			for y := int32(-63); y <= -1; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), BlockDeepslate)
			}

			// Stone layer (0 to height - 4)
			for y := int32(0); y <= height-4; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), BlockStone)
			}

			// Subsurface layer (3 blocks beneath surface)
			subBlock := BlockDirt
			if biome == BiomeDesert || biome == BiomeOcean {
				subBlock = BlockSand
			} else if biome == BiomeMountains && height >= 90 {
				subBlock = BlockStone
			}
			for y := height - 3; y < height; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), subBlock)
			}

			// Surface Block
			if height >= seaLevel {
				switch biome {
				case BiomeDesert:
					c.SetBlock(toI32(wx), height, toI32(wz), BlockSand)
				case BiomeMountains:
					if height >= 92 {
						c.SetBlock(toI32(wx), height, toI32(wz), BlockStone)
					} else {
						c.SetBlock(toI32(wx), height, toI32(wz), BlockGrass)
					}
				default:
					c.SetBlock(toI32(wx), height, toI32(wz), BlockGrass)
				}
			} else {
				// Underwater terrain
				c.SetBlock(toI32(wx), height, toI32(wz), BlockSand)
				for y := height + 1; y <= seaLevel; y++ {
					c.SetBlock(toI32(wx), y, toI32(wz), BlockWater)
				}
				if biome == BiomeSwamp && height == seaLevel-1 {
					// Swamp lily pads
					if g.noise.Noise2D(wx*0.4, wz*0.4) > 0.4 {
						c.SetBlock(toI32(wx), seaLevel+1, toI32(wz), BlockLilyPad)
					}
				}
			}
		}
	}

	// 2. Volumetric 3D Cave Carvers & Lava Lakes
	for lx := int32(0); lx < 16; lx++ {
		for lz := int32(0); lz < 16; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)
			colHeight := cols[lx][lz].Height

			maxCaveY := colHeight - 5
			if maxCaveY > 120 {
				maxCaveY = 120
			}

			for y := int32(-58); y <= maxCaveY; y++ {
				fy := float64(y)
				// Dual 3D noise worm / cheese tunnels
				n1 := g.noise.Noise3D(wx*0.038, fy*0.038, wz*0.038)
				n2 := g.noise.Noise3D((wx+64.0)*0.038, fy*0.038, (wz+64.0)*0.038)
				bubble := g.noise.Noise3D(wx*0.022, fy*0.026, wz*0.022)

				isCave := (math.Abs(n1) < 0.075 && math.Abs(n2) < 0.075) || bubble > 0.62
				if isCave {
					if y <= -54 {
						// Authentic deepslate lava lakes!
						c.SetBlock(toI32(wx), y, toI32(wz), BlockLava)
					} else {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockAir)
					}
				}
			}
		}
	}

	// 3. Authentic 3D Ore Vein Generation
	for lx := int32(0); lx < 16; lx++ {
		for lz := int32(0); lz < 16; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)
			colHeight := cols[lx][lz].Height

			for y := int32(-63); y <= colHeight-4; y++ {
				curr := c.GetBlock(toI32(wx), y, toI32(wz))
				if curr != BlockStone && curr != BlockDeepslate {
					continue // Don't replace air, water, or lava
				}

				fy := float64(y)
				oreNoise := g.noise.Noise3D(wx*0.12+fy*0.06, fy*0.12, wz*0.12)

				// Diamond Ore: Y in [-64, 16], highest density near bedrock (-58..-50)
				if y <= 16 {
					diamondThreshold := 0.76
					if y <= -45 {
						diamondThreshold = 0.68 // Denser near bottom
					}
					if oreNoise > diamondThreshold {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockDiamondOre)
						continue
					}
				}

				// Redstone Ore: Y in [-64, 16]
				if y <= 16 {
					rsNoise := g.noise.Noise3D((wx+30.0)*0.1+fy*0.05, fy*0.1, (wz+30.0)*0.1)
					if rsNoise > 0.70 {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockRedstoneOre)
						continue
					}
				}

				// Gold Ore: Y in [-64, 32]
				if y <= 32 {
					goldNoise := g.noise.Noise3D((wx+70.0)*0.09, fy*0.09, (wz+70.0)*0.09)
					if goldNoise > 0.72 {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockGoldOre)
						continue
					}
				}

				// Lapis Lazuli Ore: Y in [-64, 64]
				if y >= -64 && y <= 64 {
					lapisNoise := g.noise.Noise3D((wx+90.0)*0.1, fy*0.1, (wz+90.0)*0.1)
					if lapisNoise > 0.74 {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockLapisOre)
						continue
					}
				}

				// Iron Ore: Y in [-64, 72]
				if y <= 72 {
					ironNoise := g.noise.Noise3D((wx+120.0)*0.08, fy*0.08, (wz+120.0)*0.08)
					if ironNoise > 0.62 {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockIronOre)
						continue
					}
				}

				// Copper Ore: Y in [-16, 112]
				if y >= -16 && y <= 112 {
					copperNoise := g.noise.Noise3D((wx+150.0)*0.08, fy*0.08, (wz+150.0)*0.08)
					if copperNoise > 0.64 {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockCopperOre)
						continue
					}
				}

				// Coal Ore: Y in [0, 128]
				if y >= 0 && y <= 128 {
					coalNoise := g.noise.Noise3D((wx+180.0)*0.07, fy*0.07, (wz+180.0)*0.07)
					if coalNoise > 0.58 {
						c.SetBlock(toI32(wx), y, toI32(wz), BlockCoalOre)
						continue
					}
				}
			}
		}
	}

	// 4. Biome Vegetation, Trees & Flora
	for lx := int32(2); lx <= 13; lx++ {
		for lz := int32(2); lz <= 13; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)
			h := cols[lx][lz].Height
			biome := cols[lx][lz].Biome

			if h < seaLevel {
				continue
			}

			switch biome {
			case BiomeForest:
				treeChance := g.noise.Noise2D(wx*0.35, wz*0.35)
				if treeChance > 0.28 {
					if treeChance > 0.62 {
						g.growBirchTree(c, toI32(wx), h+1, toI32(wz))
					} else {
						g.growOakTree(c, toI32(wx), h+1, toI32(wz))
					}
				} else if treeChance > 0.05 {
					c.SetBlock(toI32(wx), h+1, toI32(wz), BlockTallGrass)
				}
			case BiomeBirchForest:
				treeChance := g.noise.Noise2D(wx*0.32, wz*0.32)
				if treeChance > 0.22 {
					g.growBirchTree(c, toI32(wx), h+1, toI32(wz))
				} else if treeChance > -0.1 {
					c.SetBlock(toI32(wx), h+1, toI32(wz), BlockDandelion)
				}
			case BiomeTaiga:
				treeChance := g.noise.Noise2D(wx*0.35, wz*0.35)
				if treeChance > 0.25 {
					g.growSpruceTree(c, toI32(wx), h+1, toI32(wz))
				} else if treeChance > 0.0 {
					c.SetBlock(toI32(wx), h+1, toI32(wz), BlockFern)
				}
			case BiomeSwamp:
				treeChance := g.noise.Noise2D(wx*0.3, wz*0.3)
				if treeChance > 0.45 {
					g.growSwampTree(c, toI32(wx), h+1, toI32(wz))
				}
			case BiomeDesert:
				cactChance := g.noise.Noise2D(wx*0.2, wz*0.2)
				if cactChance > 0.65 {
					g.growCactus(c, toI32(wx), h+1, toI32(wz))
				} else if cactChance < -0.65 {
					c.SetBlock(toI32(wx), h+1, toI32(wz), BlockDeadBush)
				}
			case BiomePlains:
				treeChance := g.noise.Noise2D(wx*0.5, wz*0.5)
				if treeChance > 0.72 {
					g.growOakTree(c, toI32(wx), h+1, toI32(wz))
				} else {
					flowerNoise := g.noise.Noise2D(wx*0.08, wz*0.08)
					if flowerNoise > 0.35 {
						detail := g.noise.Noise2D(wx*0.7+20.0, wz*0.7+20.0)
						if detail > 0.15 {
							c.SetBlock(toI32(wx), h+1, toI32(wz), BlockTallGrass)
						}
					} else if flowerNoise < -0.42 {
						if flowerNoise < -0.55 {
							c.SetBlock(toI32(wx), h+1, toI32(wz), BlockPoppy)
						} else {
							c.SetBlock(toI32(wx), h+1, toI32(wz), BlockDandelion)
						}
					}
				}
			}
		}
	}

	// 5. Procedural Structures
	g.generateStructures(c, cp, cols)
}

func (g *Generator) generateStructures(c *world.Chunk, cp world.ChunkPos, cols [16][16]ColInfo) {
	chunkSeed := (g.Seed ^ int64(cp.X)*341873128712 ^ int64(cp.Z)*132897987541)
	rng := NewPerlinNoise(chunkSeed)

	// Structure A: Underground Dungeon with Monster Spawner & Loot Chest (~1 in 16 chunks)
	if rng.Noise2D(float64(cp.X)*0.2, float64(cp.Z)*0.2) > 0.62 {
		dy := int32(-20 + int(math.Abs(rng.Noise2D(10.0, 20.0))*40.0)) // Y in [-20, 20]
		cx := cp.X*16 + 8
		cz := cp.Z*16 + 8

		// Carve 7x7x5 dungeon room
		for rx := int32(-3); rx <= 3; rx++ {
			for rz := int32(-3); rz <= 3; rz++ {
				for ry := int32(0); ry <= 4; ry++ {
					isWall := rx == -3 || rx == 3 || rz == -3 || rz == 3 || ry == 0 || ry == 4
					wx := cx + rx
					wy := dy + ry
					wz := cz + rz
					if isWall {
						if (rx+rz+ry)%3 == 0 {
							c.SetBlock(wx, wy, wz, BlockMossyCobblestone)
						} else {
							c.SetBlock(wx, wy, wz, BlockCobblestone)
						}
					} else {
						c.SetBlock(wx, wy, wz, BlockAir)
					}
				}
			}
		}
		// Center monster spawner
		c.SetBlock(cx, dy+1, cz, BlockMonsterSpawner)
		// Loot chests adjacent to wall
		c.SetBlock(cx+2, dy+1, cz, BlockChest)
		c.SetBlock(cx-2, dy+1, cz, BlockChest)
	}

	// Structure B: Ruined Nether Portal on surface or shallow underground (~1 in 24 chunks)
	if rng.Noise2D(float64(cp.X)*0.15+50.0, float64(cp.Z)*0.15+50.0) > 0.68 {
		surfaceY := cols[8][8].Height
		if surfaceY >= 62 {
			cx := cp.X*16 + 8
			cz := cp.Z*16 + 8

			// Base netherrack mound
			for bx := int32(-2); bx <= 2; bx++ {
				for bz := int32(-2); bz <= 2; bz++ {
					c.SetBlock(cx+bx, surfaceY, cz+bz, BlockNetherrack)
				}
			}
			// 4x5 Obsidian portal frame with broken top
			for px := int32(-1); px <= 2; px++ {
				c.SetBlock(cx+px, surfaceY+1, cz, BlockObsidian)
			}
			c.SetBlock(cx-1, surfaceY+2, cz, BlockObsidian)
			c.SetBlock(cx-1, surfaceY+3, cz, BlockObsidian)
			c.SetBlock(cx-1, surfaceY+4, cz, BlockObsidian)
			c.SetBlock(cx+2, surfaceY+2, cz, BlockObsidian)
			c.SetBlock(cx+2, surfaceY+3, cz, BlockObsidian)
			// Loot chest with Nether relics
			c.SetBlock(cx+1, surfaceY+1, cz+1, BlockChest)
		}
	}

	// Structure C: Stronghold End Portal Room (Guaranteed near chunk (4, 4))
	if cp.X == 4 && cp.Z == 4 {
		roomY := int32(-15)
		cx := cp.X*16 + 8
		cz := cp.Z*16 + 8

		// 9x9x6 stone brick chamber
		for rx := int32(-4); rx <= 4; rx++ {
			for rz := int32(-4); rz <= 4; rz++ {
				for ry := int32(0); ry <= 5; ry++ {
					isWall := rx == -4 || rx == 4 || rz == -4 || rz == 4 || ry == 0 || ry == 5
					wx := cx + rx
					wy := roomY + ry
					wz := cz + rz
					if isWall {
						c.SetBlock(wx, wy, wz, BlockStone)
					} else {
						c.SetBlock(wx, wy, wz, BlockAir)
					}
				}
			}
		}

		// Lava pool in center (3x3 at floor level)
		for lx := int32(-1); lx <= 1; lx++ {
			for lz := int32(-1); lz <= 1; lz++ {
				c.SetBlock(cx+lx, roomY, cz+lz, BlockLava)
				// Activated End Portal starry void blocks hovering over lava
				c.SetBlock(cx+lx, roomY+1, cz+lz, BlockEndPortal)
			}
		}

		// 12 End Portal Frame blocks around the 3x3 portal
		for i := int32(-1); i <= 1; i++ {
			c.SetBlock(cx+i, roomY+1, cz-2, BlockEndPortalFrame) // North
			c.SetBlock(cx+i, roomY+1, cz+2, BlockEndPortalFrame) // South
			c.SetBlock(cx-2, roomY+1, cz+i, BlockEndPortalFrame) // West
			c.SetBlock(cx+2, roomY+1, cz+i, BlockEndPortalFrame) // East
		}
	}
}

func (g *Generator) growOakTree(c *world.Chunk, x, baseY, z int32) {
	trunkHeight := int32(4 + int(math.Abs(g.noise.Noise2D(float64(x)*0.3, float64(z)*0.3))*3.0))
	for y := int32(0); y < trunkHeight; y++ {
		c.SetBlock(x, baseY+y, z, BlockOakLog)
	}

	for dy := trunkHeight - 2; dy <= trunkHeight+1; dy++ {
		radius := int32(2)
		if dy >= trunkHeight {
			radius = 1
		}
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx == 0 && dz == 0 && dy < trunkHeight {
					continue
				}
				c.SetBlock(x+dx, baseY+dy, z+dz, BlockOakLeaves)
			}
		}
	}
}

func (g *Generator) growBirchTree(c *world.Chunk, x, baseY, z int32) {
	trunkHeight := int32(5 + int(math.Abs(g.noise.Noise2D(float64(x)*0.25, float64(z)*0.25))*2.0))
	for y := int32(0); y < trunkHeight; y++ {
		c.SetBlock(x, baseY+y, z, BlockBirchLog)
	}

	for dy := trunkHeight - 2; dy <= trunkHeight+1; dy++ {
		radius := int32(2)
		if dy >= trunkHeight {
			radius = 1
		}
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx == 0 && dz == 0 && dy < trunkHeight {
					continue
				}
				c.SetBlock(x+dx, baseY+dy, z+dz, BlockBirchLeaves)
			}
		}
	}
}

func (g *Generator) growSpruceTree(c *world.Chunk, x, baseY, z int32) {
	trunkHeight := int32(7 + int(math.Abs(g.noise.Noise2D(float64(x)*0.2, float64(z)*0.2))*3.0))
	for y := int32(0); y < trunkHeight; y++ {
		c.SetBlock(x, baseY+y, z, BlockSpruceLog)
	}

	// Conical tiered spruce foliage
	for dy := int32(2); dy <= trunkHeight+1; dy++ {
		radius := int32(1)
		distFromTop := (trunkHeight + 1) - dy
		if distFromTop >= 4 && distFromTop%2 == 0 {
			radius = 2
		} else if distFromTop >= 2 {
			radius = 1
		} else {
			radius = 1
		}
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx == 0 && dz == 0 && dy < trunkHeight {
					continue
				}
				if math.Abs(float64(dx)) == 2 && math.Abs(float64(dz)) == 2 {
					continue // trim corners
				}
				c.SetBlock(x+dx, baseY+dy, z+dz, BlockSpruceLeaves)
			}
		}
	}
}

func (g *Generator) growSwampTree(c *world.Chunk, x, baseY, z int32) {
	trunkHeight := int32(5)
	for y := int32(0); y < trunkHeight; y++ {
		c.SetBlock(x, baseY+y, z, BlockOakLog)
	}
	// Wide flat swamp canopy
	for dx := int32(-3); dx <= 3; dx++ {
		for dz := int32(-3); dz <= 3; dz++ {
			if math.Abs(float64(dx)) == 3 && math.Abs(float64(dz)) == 3 {
				continue
			}
			c.SetBlock(x+dx, baseY+trunkHeight-1, z+dz, BlockOakLeaves)
			c.SetBlock(x+dx, baseY+trunkHeight, z+dz, BlockOakLeaves)
		}
	}
}

func (g *Generator) growCactus(c *world.Chunk, x, baseY, z int32) {
	height := int32(2 + int(math.Abs(g.noise.Noise2D(float64(x), float64(z))*2.0))%2)
	for y := int32(0); y < height; y++ {
		c.SetBlock(x, baseY+y, z, BlockCactus)
	}
}

func (g *Generator) generateNether(c *world.Chunk, cp world.ChunkPos) {
	worldStartX := cp.X * 16
	worldStartZ := cp.Z * 16

	for lx := int32(0); lx < 16; lx++ {
		for lz := int32(0); lz < 16; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)

			// Bedrock floor (0) and ceiling (127)
			c.SetBlock(toI32(wx), 0, toI32(wz), BlockBedrock)
			c.SetBlock(toI32(wx), 127, toI32(wz), BlockBedrock)

			// Cavernous Netherrack landscape with 3D noise
			for y := int32(1); y < 127; y++ {
				fy := float64(y)
				baseSolid := fy < 32 || fy > 110
				netherNoise := g.noise.Noise3D(wx*0.035, fy*0.035, wz*0.035)

				if baseSolid || netherNoise > 0.15 {
					block := BlockNetherrack
					// Soul sand patches near lava shore
					if fy >= 30 && fy <= 34 && g.noise.Noise2D(wx*0.1, wz*0.1) > 0.4 {
						block = BlockSoulSand
					}
					// Nether Quartz Ore veins
					if g.noise.Noise3D((wx+50)*0.1, fy*0.1, (wz+50)*0.1) > 0.72 {
						block = BlockNetherQuartzOre
					}
					c.SetBlock(toI32(wx), y, toI32(wz), block)
				} else if fy <= 31 {
					// Vast Nether Lava Sea at Y=31
					c.SetBlock(toI32(wx), y, toI32(wz), BlockLava)
				} else {
					c.SetBlock(toI32(wx), y, toI32(wz), BlockAir)
				}
			}

			// Hanging Glowstone clusters from ceiling (Y=115..125)
			if g.noise.Noise2D(wx*0.15, wz*0.15) > 0.55 {
				for dy := int32(0); dy < 4; dy++ {
					c.SetBlock(toI32(wx), 125-dy, toI32(wz), BlockGlowstone)
				}
			}
		}
	}

	// Ready-to-use Nether Return Portal at spawn chunk (0, 0)
	if cp.X == 0 && cp.Z == 0 {
		py := int32(65)
		// 4x5 Obsidian Portal Frame filled with Nether Portal blocks
		for px := int32(-1); px <= 2; px++ {
			c.SetBlock(px, py, 0, BlockObsidian)
			c.SetBlock(px, py+4, 0, BlockObsidian)
		}
		for pyLevel := int32(1); pyLevel <= 3; pyLevel++ {
			c.SetBlock(-1, py+pyLevel, 0, BlockObsidian)
			c.SetBlock(2, py+pyLevel, 0, BlockObsidian)
			// Active swirling purple portal blocks inside
			c.SetBlock(0, py+pyLevel, 0, BlockNetherPortal)
			c.SetBlock(1, py+pyLevel, 0, BlockNetherPortal)
		}
	}
}

func (g *Generator) generateEnd(c *world.Chunk, cp world.ChunkPos) {
	worldStartX := cp.X * 16
	worldStartZ := cp.Z * 16

	for lx := int32(0); lx < 16; lx++ {
		for lz := int32(0); lz < 16; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)

			// Floating central End Stone island centered around (0, 0)
			distFromCenter := math.Sqrt(wx*wx + wz*wz)
			if distFromCenter < 85 {
				islandThickness := int32(24 * (1.0 - distFromCenter/85.0))
				baseY := int32(60)

				for dy := -islandThickness / 2; dy <= islandThickness/2; dy++ {
					c.SetBlock(toI32(wx), baseY+dy, toI32(wz), BlockEndStone)
				}
			}
		}
	}

	// Ring of Obsidian Towers (Pillars) around central island at radius 42
	// 8 tall obsidian pillars
	pillarAngles := []float64{0, 45, 90, 135, 180, 225, 270, 315}
	pillarHeights := []int32{78, 86, 92, 82, 98, 84, 90, 80}

	for idx, angDeg := range pillarAngles {
		rad := angDeg * math.Pi / 180.0
		pillarX := int32(math.Round(42.0 * math.Cos(rad)))
		pillarZ := int32(math.Round(42.0 * math.Sin(rad)))
		pChunk := world.BlockPos{X: pillarX, Y: 0, Z: pillarZ}.ToChunkPos()

		if pChunk == cp {
			topY := pillarHeights[idx]
			// Build 3x3 obsidian tower
			for px := pillarX - 1; px <= pillarX+1; px++ {
				for pz := pillarZ - 1; pz <= pillarZ+1; pz++ {
					for y := int32(58); y <= topY; y++ {
						c.SetBlock(px, y, pz, BlockObsidian)
					}
				}
			}
			// Bedrock plate and End Crystal mount on top
			c.SetBlock(pillarX, topY+1, pillarZ, BlockBedrock)
		}
	}

	// Central Bedrock Exit Portal Fountain on top of End Island at (0, 65, 0)
	if cp.X == 0 && cp.Z == 0 {
		for rx := int32(-2); rx <= 2; rx++ {
			for rz := int32(-2); rz <= 2; rz++ {
				if math.Abs(float64(rx)) == 2 && math.Abs(float64(rz)) == 2 {
					continue
				}
				c.SetBlock(rx, 65, rz, BlockBedrock)
			}
		}
		// Central 3-block tall bedrock pillar
		c.SetBlock(0, 66, 0, BlockBedrock)
		c.SetBlock(0, 67, 0, BlockBedrock)
		c.SetBlock(0, 68, 0, BlockBedrock)
		// Dragon Egg sitting on top
		c.SetBlock(0, 69, 0, BlockDragonEgg)
		// Active starry exit portal
		c.SetBlock(1, 65, 0, BlockEndPortal)
		c.SetBlock(-1, 65, 0, BlockEndPortal)
		c.SetBlock(0, 65, 1, BlockEndPortal)
		c.SetBlock(0, 65, -1, BlockEndPortal)
	}
}

func toI32(v float64) int32 {
	return int32(math.Floor(v))
}
