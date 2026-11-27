package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestColoredCarpetPlacementSupportAndSupportBreakDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{77}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	carpet, ok := inventory.ExtendedBlockByName("minecraft:cyan_carpet")
	if !ok {
		t.Fatal("cyan carpet definition missing")
	}
	base := world.BlockPos{X: 2, Y: 64, Z: 2}
	place := world.BlockPos{X: base.X, Y: base.Y + 1, Z: base.Z}
	if eng.PlaceBlock(player.ID, place, carpet.RuntimeID) {
		t.Fatal("carpet placed without a supporting block")
	}
	eng.SetBlock(base.X, base.Y, base.Z, inventory.ItemStone)
	if !eng.PlaceBlock(player.ID, place, carpet.RuntimeID) {
		t.Fatal("carpet did not place on a solid block")
	}
	if eng.PlaceBlock(player.ID, world.BlockPos{X: place.X, Y: place.Y + 1, Z: place.Z}, carpet.RuntimeID) {
		t.Fatal("carpet incorrectly treated another carpet as a supporting block")
	}
	boxes := BlockCollisionAABBs(eng.GetBlock(place.X, place.Y, place.Z), place.X, place.Y, place.Z)
	if len(boxes) != 1 || boxes[0].Max.Y-boxes[0].Min.Y != 1.0/16.0 {
		t.Fatalf("placed carpet collision boxes = %+v, want a 1/16-high carpet", boxes)
	}

	if got := eng.BreakBlock(player.ID, base); got != inventory.ItemCobblestone {
		t.Fatalf("breaking carpet support returned %d, want cobblestone", got)
	}
	if got := eng.GetBlock(place.X, place.Y, place.Z); got != 0 {
		t.Fatalf("carpet remained unsupported after base break: block ID %d", got)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == carpet.RuntimeID {
			count += stack.Count
		}
	}
	if count != 1 {
		t.Fatalf("player received %d carpets after support break, want 1", count)
	}
}

func TestToolBreakOfCarpetSupportDropsCarpetEntity(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Alex", [16]byte{78}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	carpet, _ := inventory.ExtendedBlockByName("minecraft:red_carpet")
	base := world.BlockPos{X: 3, Y: 64, Z: 3}
	eng.SetBlock(base.X, base.Y, base.Z, inventory.ItemStone)
	if !eng.PlaceBlock(player.ID, world.BlockPos{X: base.X, Y: base.Y + 1, Z: base.Z}, carpet.RuntimeID) {
		t.Fatal("red carpet placement failed")
	}
	if _, harvested := eng.BreakBlockWithTool(player.ID, base, inventory.ItemWoodenPickaxe); !harvested {
		t.Fatal("stone support was not harvested with a wooden pickaxe")
	}
	if got := eng.GetBlock(base.X, base.Y+1, base.Z); got != 0 {
		t.Fatalf("carpet remained unsupported after tool break: block ID %d", got)
	}
	eng.mu.RLock()
	defer eng.mu.RUnlock()
	found := false
	for _, drop := range eng.ItemDrops {
		if drop.ItemID == carpet.RuntimeID && drop.Count == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("tool break of carpet support did not drop the carpet item")
	}
}
