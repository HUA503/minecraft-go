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

// Block IDs catalog
const (
	BlockAir           int32 = 0
	BlockStone         int32 = 1
	BlockGrass         int32 = 2
	BlockDirt          int32 = 3
	BlockCobblestone   int32 = 4
	BlockOakPlanks     int32 = 5
	BlockBedrock       int32 = 7
	BlockWater         int32 = 8
	BlockFlowingWater  int32 = 9
	BlockLava          int32 = 10
	BlockFlowingLava   int32 = 11
	BlockSand          int32 = 12
	BlockGoldOre       int32 = 14
	BlockIronOre       int32 = 15
	BlockCoalOre       int32 = 16
	BlockOakLog        int32 = 17
	BlockOakLeaves     int32 = 18
	BlockGlass         int32 = 20
	BlockTallGrass     int32 = 31
	BlockDandelion     int32 = 37
	BlockPoppy         int32 = 38
	BlockObsidian      int32 = 49
	BlockTorch         int32 = 50
	BlockDiamondOre    int32 = 56
	BlockCraftingTable int32 = 58
	BlockFurnace       int32 = 61
	BlockNetherrack    int32 = 87
	BlockGlowstone     int32 = 89
	BlockEndStone      int32 = 121
	BlockDeepslate     int32 = 150
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

func (g *Generator) generateOverworld(c *world.Chunk, cp world.ChunkPos) {
	const seaLevel = 62
	worldStartX := cp.X * 16
	worldStartZ := cp.Z * 16

	for lx := int32(0); lx < 16; lx++ {
		for lz := int32(0); lz < 16; lz++ {
			wx := float64(worldStartX + lx)
			wz := float64(worldStartZ + lz)

			// Multi-biome terrain determination using broad Perlin noise
			biomeNoise := g.noise.Noise2D(wx*0.003, wz*0.003)
			isMountain := biomeNoise > 0.45
			isForest := biomeNoise >= -0.15 && biomeNoise <= 0.22
			isDesert := biomeNoise < -0.55

			var height int32
			if isMountain {
				// Rugged mountain peaks reaching Y=85..105
				mElev := g.noise.FBM2D(wx*0.02, wz*0.02, 5, 0.55, 2.0)
				height = int32(75 + mElev*30)
			} else if isDesert {
				// Gentle rolling desert dunes
				dElev := g.noise.FBM2D(wx*0.012, wz*0.012, 3, 0.4, 2.0)
				height = int32(65 + dElev*10)
			} else {
				// Plains / Forest rolling meadows
				elevation := g.noise.FBM2D(wx*0.015, wz*0.015, 4, 0.5, 2.0)
				height = int32(64 + elevation*18)
			}

			// Bedrock floor at -64
			c.SetBlock(toI32(wx), -64, toI32(wz), BlockBedrock)

			// Deepslate layer (-63 to -1)
			for y := int32(-63); y <= -1; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), BlockDeepslate)
			}

			// Stone layer (0 to height - 4)
			for y := int32(0); y <= height-4; y++ {
				// Procedural ore distribution
				oreNoise := g.noise.Noise2D(wx*0.1+float64(y)*0.05, wz*0.1)
				block := BlockStone
				if y < 16 && oreNoise > 0.75 {
					block = BlockDiamondOre
				} else if y < 48 && oreNoise > 0.65 {
					block = BlockIronOre
				} else if oreNoise > 0.6 {
					block = BlockCoalOre
				}
				c.SetBlock(toI32(wx), y, toI32(wz), block)
			}

			// Subsurface layer (3 blocks beneath surface)
			if isDesert {
				for y := height - 3; y < height; y++ {
					c.SetBlock(toI32(wx), y, toI32(wz), BlockSand)
				}
			} else if isMountain && height >= 90 {
				for y := height - 3; y < height; y++ {
					c.SetBlock(toI32(wx), y, toI32(wz), BlockStone)
				}
			} else {
				for y := height - 3; y < height; y++ {
					c.SetBlock(toI32(wx), y, toI32(wz), BlockDirt)
				}
			}

			// Surface block
			if height >= seaLevel {
				if isDesert {
					c.SetBlock(toI32(wx), height, toI32(wz), BlockSand)
				} else if isMountain && height >= 90 {
					// High mountain peak: exposed stone cliffs
					c.SetBlock(toI32(wx), height, toI32(wz), BlockStone)
				} else {
					c.SetBlock(toI32(wx), height, toI32(wz), BlockGrass)

					// Forest biome: dense woodland tree distribution
					// Plains biome: open expanses with occasional solitary trees
					treeMinChance := 0.68
					if isForest {
						treeMinChance = 0.40
					}
					treeChance := g.noise.Noise2D(wx*0.5, wz*0.5)
					if treeChance > treeMinChance && lx >= 2 && lx <= 13 && lz >= 2 && lz <= 13 {
						g.growOakTree(c, toI32(wx), height+1, toI32(wz))
					} else {
						// Clustered tall grass and flower patches
						patchNoise := g.noise.Noise2D(wx*0.06, wz*0.06)
						if patchNoise > 0.42 {
							detail := g.noise.Noise2D(wx*0.8+50.0, wz*0.8+50.0)
							if detail > 0.15 {
								c.SetBlock(toI32(wx), height+1, toI32(wz), BlockTallGrass)
							}
						} else if patchNoise < -0.48 {
							detail := g.noise.Noise2D(wx*0.8+150.0, wz*0.8+150.0)
							if detail > 0.22 {
								if patchNoise < -0.60 {
									c.SetBlock(toI32(wx), height+1, toI32(wz), BlockPoppy)
								} else {
									c.SetBlock(toI32(wx), height+1, toI32(wz), BlockDandelion)
								}
							}
						}
					}
				}
			} else {
				// Below sea level: Sand surface and fill with water
				c.SetBlock(toI32(wx), height, toI32(wz), BlockSand)
				for y := height + 1; y <= seaLevel; y++ {
					c.SetBlock(toI32(wx), y, toI32(wz), BlockWater)
				}
			}
		}
	}
}

func (g *Generator) growOakTree(c *world.Chunk, x, baseY, z int32) {
	// Varied trunk height based on coordinate hash: 4 to 6 blocks high
	trunkHeight := int32(4 + int(g.noise.Noise2D(float64(x)*0.3, float64(z)*0.3)*2.0+1.0)%3)
	for y := int32(0); y < trunkHeight; y++ {
		c.SetBlock(x, baseY+y, z, BlockOakLog)
	}

	// Leaves canopy (5x5 layer at trunkHeight-2 and trunkHeight-1, 3x3 at trunkHeight)
	for dy := trunkHeight - 2; dy <= trunkHeight; dy++ {
		radius := int32(2)
		if dy == trunkHeight {
			radius = 1
		}
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx == 0 && dz == 0 && dy < trunkHeight {
					continue // trunk
				}
				c.SetBlock(x+dx, baseY+dy, z+dz, BlockOakLeaves)
			}
		}
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

			// Netherrack floor up to Y=40
			noiseVal := g.noise.FBM2D(wx*0.03, wz*0.03, 3, 0.5, 2.0)
			terrainHeight := int32(35 + noiseVal*20)

			for y := int32(1); y <= terrainHeight; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), BlockNetherrack)
			}

			// Lava ocean at sea level 31
			for y := terrainHeight + 1; y <= 31; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), BlockLava)
			}

			// Hanging glowstone clusters near ceiling
			if g.noise.Noise2D(wx*0.2, wz*0.2) > 0.7 {
				for dy := int32(0); dy < 3; dy++ {
					c.SetBlock(toI32(wx), 126-dy, toI32(wz), BlockGlowstone)
				}
			}
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

			// Floating central island centered around (0, 0)
			distFromCenter := math.Sqrt(wx*wx + wz*wz)
			if distFromCenter < 80 {
				islandThickness := int32(25 * (1.0 - distFromCenter/80.0))
				baseY := int32(60)

				for dy := -islandThickness / 2; dy <= islandThickness/2; dy++ {
					c.SetBlock(toI32(wx), baseY+dy, toI32(wz), BlockEndStone)
				}
			}
		}
	}
}

func toI32(v float64) int32 {
	return int32(math.Floor(v))
}
