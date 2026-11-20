package gen

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestPerlinNoiseProperties(t *testing.T) {
	p := NewPerlinNoise(1337)
	for x := -10.0; x <= 10.0; x += 0.5 {
		for y := -10.0; y <= 10.0; y += 0.5 {
			val := p.Noise2D(x, y)
			if val < -1.0 || val > 1.0 {
				t.Fatalf("noise value out of bounds: %f at (%f, %f)", val, x, y)
			}
		}
	}
}

func TestOverworldGeneration(t *testing.T) {
	gen := NewGenerator(42, DimensionOverworld)
	c := gen.GenerateChunk(world.ChunkPos{X: 0, Z: 0})

	// Check Bedrock at Y = -64
	if b := c.GetBlock(5, -64, 5); b != BlockBedrock {
		t.Fatalf("bedrock floor mismatch: got %d, expected %d", b, BlockBedrock)
	}

	// Check Deepslate at Y = -20
	if b := c.GetBlock(5, -20, 5); b != BlockDeepslate {
		t.Fatalf("deepslate layer mismatch: got %d, expected %d", b, BlockDeepslate)
	}

	// Check that surface has grass or water
	surfaceFound := false
	for y := int32(50); y <= 90; y++ {
		b := c.GetBlock(8, y, 8)
		if b == BlockGrass || b == BlockWater {
			surfaceFound = true
			break
		}
	}
	if !surfaceFound {
		t.Fatalf("no surface grass or water found in spawn chunk")
	}
}

func TestNetherGeneration(t *testing.T) {
	gen := NewGenerator(42, DimensionNether)
	c := gen.GenerateChunk(world.ChunkPos{X: 0, Z: 0})

	// Bedrock floor (0) and ceiling (127)
	if b := c.GetBlock(0, 0, 0); b != BlockBedrock {
		t.Fatalf("nether floor bedrock mismatch: got %d", b)
	}
	if b := c.GetBlock(0, 127, 0); b != BlockBedrock {
		t.Fatalf("nether ceiling bedrock mismatch: got %d", b)
	}

	// Check presence of Netherrack
	foundNetherrack := false
	for y := int32(5); y <= 30; y++ {
		if c.GetBlock(4, y, 4) == BlockNetherrack {
			foundNetherrack = true
			break
		}
	}
	if !foundNetherrack {
		t.Fatalf("no netherrack generated in nether chunk")
	}
}

func TestEndGeneration(t *testing.T) {
	gen := NewGenerator(42, DimensionEnd)
	c := gen.GenerateChunk(world.ChunkPos{X: 0, Z: 0})

	// Center island at Y=60 should have End Stone
	if b := c.GetBlock(0, 60, 0); b != BlockEndStone {
		t.Fatalf("end stone mismatch at center: got %d, expected %d", b, BlockEndStone)
	}

	// Exit portal fountain on top at Y=65
	if b := c.GetBlock(0, 66, 0); b != BlockBedrock {
		t.Fatalf("bedrock pillar mismatch: got %d, expected %d", b, BlockBedrock)
	}
	if b := c.GetBlock(0, 69, 0); b != BlockDragonEgg {
		t.Fatalf("dragon egg mismatch: got %d, expected %d", b, BlockDragonEgg)
	}
}

func TestStrongholdPortalRoomGeneration(t *testing.T) {
	gen := NewGenerator(42, DimensionOverworld)
	c := gen.GenerateChunk(world.ChunkPos{X: 4, Z: 4})

	foundFrame := false
	foundPortal := false
	for x := int32(4 * 16); x < int32(5 * 16); x++ {
		for z := int32(4 * 16); z < int32(5 * 16); z++ {
			for y := int32(-20); y <= int32(-10); y++ {
				b := c.GetBlock(x, y, z)
				if b == BlockEndPortalFrame {
					foundFrame = true
				}
				if b == BlockEndPortal {
					foundPortal = true
				}
			}
		}
	}
	if !foundFrame {
		t.Fatalf("expected End Portal Frame in stronghold chunk (4, 4)")
	}
	if !foundPortal {
		t.Fatalf("expected active End Portal block in stronghold chunk (4, 4)")
	}
}

func TestBiomeSamplingAndOres(t *testing.T) {
	gen := NewGenerator(42, DimensionOverworld)
	// Sample various coordinates to test biomes
	foundPlains := false
	foundForest := false
	foundDesert := false
	foundTaiga := false

	for x := -500.0; x <= 500.0; x += 100.0 {
		for z := -500.0; z <= 500.0; z += 100.0 {
			biome := gen.SampleBiome(x, z)
			switch biome {
			case BiomePlains:
				foundPlains = true
			case BiomeForest, BiomeBirchForest:
				foundForest = true
			case BiomeDesert:
				foundDesert = true
			case BiomeTaiga:
				foundTaiga = true
			}
		}
	}

	if !foundPlains && !foundForest && !foundDesert && !foundTaiga {
		t.Fatalf("expected varied biomes sampled across map")
	}

	// Check ore presence in chunks
	foundOre := false
	for cx := int32(-1); cx <= 1; cx++ {
		for cz := int32(-1); cz <= 1; cz++ {
			c := gen.GenerateChunk(world.ChunkPos{X: cx, Z: cz})
			for x := cx * 16; x < (cx+1)*16; x++ {
				for z := cz * 16; z < (cz+1)*16; z++ {
					for y := int32(-60); y <= 80; y++ {
						b := c.GetBlock(x, y, z)
						if b == BlockCoalOre || b == BlockIronOre || b == BlockDiamondOre || b == BlockRedstoneOre || b == BlockGoldOre {
							foundOre = true
							break
						}
					}
				}
			}
		}
	}
	if !foundOre {
		t.Fatalf("expected ores generated in overworld chunks")
	}

	// Verify chunk sections store the sampled biomes
	chk := gen.GenerateChunk(world.ChunkPos{X: 0, Z: 0})
	bAtSurface := chk.GetBiome(4, 64, 4)
	if bAtSurface < 0 || bAtSurface > 7 {
		t.Fatalf("expected valid biome ID (0..7) in chunk section container, got %d", bAtSurface)
	}
	bInCave := chk.GetBiome(4, -32, 4)
	if bInCave < 0 || bInCave > 7 {
		t.Fatalf("expected valid biome ID (0..7) in cave section container, got %d", bInCave)
	}
}
