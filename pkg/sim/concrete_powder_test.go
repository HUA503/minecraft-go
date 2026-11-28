package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestConcretePowderFallsAndSettlesOnSupport(t *testing.T) {
	engine := NewEngine()
	if !engine.SetBlockChecked(10, 60, 10, inventory.ItemStone) {
		t.Fatal("failed to place floor")
	}
	if !engine.SetBlockChecked(10, 64, 10, inventory.FirstConcretePowder) {
		t.Fatal("failed to place concrete powder")
	}
	for tick := 0; tick < 8; tick++ {
		engine.Tick()
	}
	if got := engine.GetBlock(10, 64, 10); got != 0 {
		t.Fatalf("powder source column still contains block %d", got)
	}
	if got := engine.GetBlock(10, 61, 10); got != inventory.FirstConcretePowder {
		t.Fatalf("settled powder = %d, want %d", got, inventory.FirstConcretePowder)
	}
	engine.Tick()
	if got := engine.GetBlock(10, 61, 10); got != inventory.FirstConcretePowder {
		t.Fatalf("supported powder unexpectedly changed to %d", got)
	}
}

func TestConcretePowderHardensOnWaterContactAndBreaksToPowder(t *testing.T) {
	engine := NewEngine()
	if !engine.SetBlockChecked(10, 64, 10, inventory.FirstConcretePowder) || !engine.SetBlockChecked(11, 64, 10, 8) {
		t.Fatal("failed to set powder and water blocks")
	}
	engine.Tick()
	concrete, ok := inventory.ExtendedBlockByName("minecraft:white_concrete")
	if !ok {
		t.Fatal("white concrete block is not registered")
	}
	if got := engine.GetBlock(10, 64, 10); got != concrete.RuntimeID {
		t.Fatalf("water-adjacent powder became block %d, want concrete %d", got, concrete.RuntimeID)
	}
	if drop, broken := engine.BreakBlockWithTool(0, world.BlockPos{X: 10, Y: 64, Z: 10}, inventory.ItemWoodenPickaxe); !broken || drop != concrete.RuntimeID {
		t.Fatalf("breaking concrete returned (%d,%t), want concrete item %d", drop, broken, concrete.RuntimeID)
	}

	placedIntoWater := NewEngine()
	if !placedIntoWater.SetBlockChecked(10, 64, 10, 8) || !placedIntoWater.SetBlockChecked(10, 64, 10, inventory.FirstConcretePowder) {
		t.Fatal("failed to replace source water with concrete powder")
	}
	if got := placedIntoWater.GetBlock(10, 64, 10); got != concrete.RuntimeID {
		t.Fatalf("powder placed into water became block %d, want concrete %d", got, concrete.RuntimeID)
	}
}

func TestConcretePowderTicksResumeAfterChunkLoad(t *testing.T) {
	cp := world.ChunkPos{X: 0, Z: 0}
	chunk := world.NewChunk(cp)
	chunk.SetBlock(10, 60, 10, inventory.ItemStone)
	chunk.SetBlock(10, 64, 10, inventory.FirstConcretePowder)
	reloaded, err := world.ChunkFromNBT(chunk.ToNBT())
	if err != nil {
		t.Fatalf("round-trip chunk containing concrete powder: %v", err)
	}
	engine := NewEngine()
	engine.LoadGeneratedChunk(cp, reloaded)
	for tick := 0; tick < 8; tick++ {
		engine.Tick()
	}
	if got := engine.GetBlock(10, 61, 10); got != inventory.FirstConcretePowder {
		t.Fatalf("powder loaded from chunk did not resume falling: state at landing position=%d", got)
	}
}

func TestConcretePowderTickQueueIsIsolatedByDimension(t *testing.T) {
	engine := NewEngine()
	if !engine.SetBlockChecked(10, 60, 10, inventory.ItemStone) || !engine.SetBlockChecked(10, 64, 10, inventory.FirstConcretePowder) {
		t.Fatal("failed to set overworld powder and support")
	}
	engine.SwitchDimension(1)
	for tick := 0; tick < 4; tick++ {
		engine.Tick()
	}
	engine.SwitchDimension(0)
	for tick := 0; tick < 8; tick++ {
		engine.Tick()
	}
	if got := engine.GetBlock(10, 61, 10); got != inventory.FirstConcretePowder {
		t.Fatalf("overworld powder was lost or not resumed after a dimension switch: state=%d", got)
	}
}
