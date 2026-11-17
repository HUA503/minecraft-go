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
}
