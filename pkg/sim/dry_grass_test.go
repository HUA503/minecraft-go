package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestDryGrassPlacementRequiresVanillaSupport(t *testing.T) {
	engine := NewEngine()
	engine.SetBlock(0, 63, 0, inventory.ItemDirt)
	engine.SetBlock(2, 63, 0, inventory.ItemStone)
	if !engine.PlaceBlock(0, world.BlockPos{X: 0, Y: 64, Z: 0}, inventory.ItemDryShortGrass) {
		t.Fatal("dry grass placement on dirt failed")
	}
	if engine.PlaceBlock(0, world.BlockPos{X: 2, Y: 64, Z: 0}, inventory.ItemDryShortGrass) {
		t.Fatal("dry grass placement on stone succeeded")
	}
}

func TestBoneMealConvertsAndSpreadsDryGrass(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("dry-grass-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 2}
	player.Inventory.SelectedHotbar = 0
	engine.SetBlock(0, 63, 0, inventory.ItemDirt)
	engine.SetBlock(0, 64, 0, inventory.ItemDryShortGrass)
	pos, state, ok := engine.UseBoneMealOnDryGrass(player.ID, world.BlockPos{X: 0, Y: 64, Z: 0})
	if !ok || state != inventory.ItemDryTallGrass || pos != (world.BlockPos{X: 0, Y: 64, Z: 0}) {
		t.Fatalf("short grass bone meal result = %v/%d/%v", pos, state, ok)
	}
	if got := engine.GetBlock(0, 64, 0); got != inventory.ItemDryTallGrass {
		t.Fatalf("short grass grew into block %d, want tall dry grass", got)
	}
	if player.Inventory.Slots[0].Count != 1 {
		t.Fatalf("bone meal remaining after growth = %d, want 1", player.Inventory.Slots[0].Count)
	}
	for _, direction := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		engine.SetBlock(direction[0], 63, direction[1], inventory.ItemDirt)
	}
	spread, spreadState, ok := engine.UseBoneMealOnDryGrass(player.ID, world.BlockPos{X: 0, Y: 64, Z: 0})
	if !ok || spreadState != inventory.ItemDryShortGrass {
		t.Fatalf("tall grass spread result = %v/%d/%v", spread, spreadState, ok)
	}
	if engine.GetBlock(spread.X, spread.Y, spread.Z) != inventory.ItemDryShortGrass {
		t.Fatalf("spread position %v was not filled with short dry grass", spread)
	}
	if player.Inventory.Slots[0].Count != 0 {
		t.Fatalf("bone meal remaining after spread = %d, want 0", player.Inventory.Slots[0].Count)
	}
}

func TestBreakingDryGrassSupportDropsPlant(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("dry-grass-support-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	engine.SetBlock(0, 63, 0, inventory.ItemDirt)
	engine.SetBlock(0, 64, 0, inventory.ItemDryShortGrass)
	engine.BreakBlock(player.ID, world.BlockPos{X: 0, Y: 63, Z: 0})
	if got := engine.GetBlock(0, 64, 0); got != 0 {
		t.Fatalf("unsupported dry grass remained as block %d", got)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == inventory.ItemDryShortGrass {
			count += stack.Count
		}
	}
	if count != 1 {
		t.Fatalf("support break yielded %d dry grass items, want 1", count)
	}
}
