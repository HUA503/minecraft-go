package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestJava263SpecialCubeMiningDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Miner", [16]byte{42}, Vec3{X: 0, Y: 64, Z: 0})
	if player == nil || player.Inventory == nil {
		t.Fatal("expected initialized mining player")
	}
	pos := world.BlockPos{X: 2, Y: 64, Z: 0}

	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemIronPickaxe, Count: 1}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 14200) // Crying Obsidian needs diamond-tier harvest.
	if drop, _ := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemIronPickaxe); drop != inventory.ItemAir {
		t.Fatalf("iron pickaxe drop from crying obsidian = %d, want no drop", drop)
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemDiamondPickaxe, Count: 1}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 14200)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemDiamondPickaxe); drop != 14200 || !harvested {
		t.Fatalf("diamond pickaxe drop from crying obsidian = %d, want block item", drop)
	}

	player.Inventory.Slots[0] = inventory.ItemStack{}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 14201) // Nylium drops netherrack without Silk Touch.
	if drop, _ := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != 87 {
		t.Fatalf("hand-mined crimson nylium drop = %d, want netherrack (87)", drop)
	}

	eng.SetBlock(pos.X, pos.Y, pos.Z, 14204) // Nether gold ore drops 2–6 nuggets.
	if drop, _ := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != inventory.ItemAir {
		t.Fatalf("hand-mined nether gold ore drop = %d, want no drop", drop)
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 14204)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenPickaxe); drop != inventory.ItemGoldNugget || !harvested {
		t.Fatalf("pickaxe-mined nether gold ore drop = %d, want gold nuggets", drop)
	}
	got := 0
	for _, drop := range eng.GetItemDrops() {
		if drop.ItemID == inventory.ItemGoldNugget {
			got += drop.Count
		}
	}
	if got < 2 || got > 6 {
		t.Fatalf("nether gold ore yielded %d nuggets, want 2..6", got)
	}
}
