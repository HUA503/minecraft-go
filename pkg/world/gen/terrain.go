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

			// 2D terrain elevation using fBm
			elevation := g.noise.FBM2D(wx*0.015, wz*0.015, 4, 0.5, 2.0)
			height := int32(64 + elevation*18)

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

			// Dirt layer
			for y := height - 3; y < height; y++ {
				c.SetBlock(toI32(wx), y, toI32(wz), BlockDirt)
			}

			// Surface block
			if height >= seaLevel {
				c.SetBlock(toI32(wx), height, toI32(wz), BlockGrass)

				// Procedural oak tree generation (roughly 1 tree every 64 blocks)
				treeChance := g.noise.Noise2D(wx*0.5, wz*0.5)
				if treeChance > 0.65 && lx >= 2 && lx <= 13 && lz >= 2 && lz <= 13 {
					g.growOakTree(c, toI32(wx), height+1, toI32(wz))
				} else {
					// Authentic meadow foliage distribution (Tall Grass, Dandelions, Poppies)
					foliageNoise := g.noise.Noise2D(wx*0.75+313.0, wz*0.75+313.0)
					if foliageNoise > 0.70 {
						c.SetBlock(toI32(wx), height+1, toI32(wz), BlockTallGrass)
					} else if foliageNoise < 0.22 {
						flowerType := g.noise.Noise2D(wx*1.2+77.0, wz*1.2+77.0)
						if flowerType > 0.6 {
							c.SetBlock(toI32(wx), height+1, toI32(wz), BlockDandelion)
						} else if flowerType < 0.4 {
							c.SetBlock(toI32(wx), height+1, toI32(wz), BlockPoppy)
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
	// Trunk: 5 blocks high
	for y := int32(0); y < 5; y++ {
		c.SetBlock(x, baseY+y, z, BlockOakLog)
	}

	// Leaves canopy (5x5 layer at Y+3 and Y+4, 3x3 at Y+5)
	for dy := int32(3); dy <= 5; dy++ {
		radius := int32(2)
		if dy == 5 {
			radius = 1
		}
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if dx == 0 && dz == 0 && dy < 5 {
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
