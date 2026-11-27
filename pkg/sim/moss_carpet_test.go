package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestMossCarpetPlacementAndSupportDrop(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("moss-carpet-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	base := world.BlockPos{X: 0, Y: 64, Z: 0}
	engine.SetBlock(base.X, base.Y-1, base.Z, inventory.ItemGrass)
	if !engine.PlaceBlock(player.ID, base, inventory.ItemMossCarpet) {
		t.Fatal("moss carpet placement on solid support failed")
	}
	if got := engine.GetBlock(base.X, base.Y, base.Z); got != inventory.ItemMossCarpet {
		t.Fatalf("placed moss carpet block = %d, want %d", got, inventory.ItemMossCarpet)
	}
	if engine.PlaceBlock(player.ID, world.BlockPos{X: 0, Y: 65, Z: 0}, inventory.ItemPaleMossCarpet) {
		t.Fatal("placing a second thin carpet on a moss carpet succeeded")
	}
	engine.BreakBlock(player.ID, world.BlockPos{X: base.X, Y: base.Y - 1, Z: base.Z})
	if got := engine.GetBlock(base.X, base.Y, base.Z); got != 0 {
		t.Fatalf("unsupported moss carpet remained as block %d", got)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == inventory.ItemMossCarpet {
			count += stack.Count
		}
	}
	if count != 1 {
		t.Fatalf("support break yielded %d moss carpets, want 1", count)
	}
}
