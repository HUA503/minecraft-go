package fluid

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

func TestFluidDownwardFlow(t *testing.T) {
	blocks := make(map[[3]int32]int32)
	getter := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
	setter := func(x, y, z int32, id int32) { blocks[[3]int32{x, y, z}] = id }

	sim := NewSimulator(getter, setter)

	// Place water source at (0, 10, 0)
	setter(0, 10, 0, gen.BlockWater)
	sim.SpreadStep(0, 10, 0, FluidWater)

	// Block below (0, 9, 0) should become flowing water
	if b := getter(0, 9, 0); b != gen.BlockFlowingWater {
		t.Fatalf("expected downward flowing water at (0, 9, 0), got %d", b)
	}
}

func TestFluidHorizontalSpread(t *testing.T) {
	blocks := make(map[[3]int32]int32)
	getter := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
	setter := func(x, y, z int32, id int32) { blocks[[3]int32{x, y, z}] = id }

	sim := NewSimulator(getter, setter)

	// Solid floor at Y=9
	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			setter(dx, 9, dz, gen.BlockStone)
		}
	}
	// Water source at (0, 10, 0)
	setter(0, 10, 0, gen.BlockWater)

	sim.SpreadStep(0, 10, 0, FluidWater)

	// Neighbors at Y=10 should receive flowing water
	if getter(1, 10, 0) != gen.BlockFlowingWater || getter(-1, 10, 0) != gen.BlockFlowingWater {
		t.Fatalf("horizontal water spread failed: (1,10,0)=%d, (-1,10,0)=%d",
			getter(1, 10, 0), getter(-1, 10, 0))
	}
}

func TestWaterMeetingLavaObsidianAndCobblestone(t *testing.T) {
	blocks := make(map[[3]int32]int32)
	getter := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
	setter := func(x, y, z int32, id int32) { blocks[[3]int32{x, y, z}] = id }

	sim := NewSimulator(getter, setter)

	// 1. Water touching Lava Source -> Obsidian
	setter(0, 64, 0, gen.BlockWater)
	setter(1, 64, 0, gen.BlockLava)
	setter(0, 63, 0, gen.BlockStone) // floor

	sim.SpreadStep(0, 64, 0, FluidWater)
	if b := getter(1, 64, 0); b != gen.BlockObsidian {
		t.Fatalf("water touching lava source should produce Obsidian, got %d", b)
	}

	// 2. Water touching Flowing Lava -> Cobblestone
	setter(0, 64, 5, gen.BlockWater)
	setter(1, 64, 5, gen.BlockFlowingLava)
	setter(0, 63, 5, gen.BlockStone) // floor

	sim.SpreadStep(0, 64, 5, FluidWater)
	if b := getter(1, 64, 5); b != gen.BlockCobblestone {
		t.Fatalf("water touching flowing lava should produce Cobblestone, got %d", b)
	}
}

func TestInfiniteWaterSource(t *testing.T) {
	blocks := make(map[[3]int32]int32)
	getter := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
	setter := func(x, y, z int32, id int32) { blocks[[3]int32{x, y, z}] = id }

	sim := NewSimulator(getter, setter)

	// Solid floor at Y=63
	setter(0, 63, 0, gen.BlockStone)

	// Two adjacent water sources at (1, 64, 0) and (-1, 64, 0)
	setter(1, 64, 0, gen.BlockWater)
	setter(-1, 64, 0, gen.BlockWater)

	// Empty center at (0, 64, 0)
	created := sim.CheckInfiniteWaterSource(0, 64, 0)
	if !created {
		t.Fatalf("infinite water source was not formed")
	}
	if b := getter(0, 64, 0); b != gen.BlockWater {
		t.Fatalf("center should become water source, got %d", b)
	}
}
