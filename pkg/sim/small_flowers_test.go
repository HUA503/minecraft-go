package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestSmallFlowerPlacementRequiresFlowerSoil(t *testing.T) {
	engine := NewEngine()
	engine.SetBlock(0, 63, 0, inventory.ItemGrass)
	engine.SetBlock(2, 63, 0, inventory.ItemStone)
	flower := inventory.FirstSmallFlower
	if !engine.PlaceBlock(0, world.BlockPos{X: 0, Y: 64, Z: 0}, flower) {
		t.Fatal("flower placement on grass block failed")
	}
	if engine.PlaceBlock(0, world.BlockPos{X: 2, Y: 64, Z: 0}, flower) {
		t.Fatal("flower placement on stone succeeded")
	}
	if got := engine.GetBlock(0, 64, 0); got != flower {
		t.Fatalf("placed flower state = %d, want %d", got, flower)
	}
}

func TestBreakingFlowerSupportDropsFlower(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("flower-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	if player == nil {
		t.Fatal("failed to add player")
	}
	engine.SetBlock(0, 63, 0, inventory.ItemGrass)
	flower := inventory.FirstSmallFlower
	engine.SetBlock(0, 64, 0, flower)
	if got := engine.BreakBlock(player.ID, world.BlockPos{X: 0, Y: 63, Z: 0}); got != inventory.ItemDirt {
		t.Fatalf("support break drop = %d, want dirt (%d)", got, inventory.ItemDirt)
	}
	if got := engine.GetBlock(0, 64, 0); got != 0 {
		t.Fatalf("unsupported flower remained in world with block id %d", got)
	}
	flowerCount := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == flower {
			flowerCount += stack.Count
		}
	}
	if flowerCount != 1 {
		t.Fatalf("support break returned %d flower items, want 1", flowerCount)
	}
}
