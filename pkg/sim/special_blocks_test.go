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

	for _, block := range []struct {
		id   int32
		name string
	}{
		{14205, "podzol"},
		{14206, "mycelium"},
	} {
		eng.SetBlock(pos.X, pos.Y, pos.Z, block.id)
		if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != inventory.ItemDirt || !harvested {
			t.Fatalf("hand-mined %s drop = %d, harvested=%v; want dirt", block.name, drop, harvested)
		}
	}

	for _, block := range []struct {
		id, wantDrop int32
		name         string
	}{
		{14209, 14214, "tube_coral_block"},
		{14210, 14215, "brain_coral_block"},
		{14211, 14216, "bubble_coral_block"},
		{14212, 14217, "fire_coral_block"},
		{14213, 14218, "horn_coral_block"},
		{14214, 14214, "dead_tube_coral_block"},
		{14215, 14215, "dead_brain_coral_block"},
		{14216, 14216, "dead_bubble_coral_block"},
		{14217, 14217, "dead_fire_coral_block"},
		{14218, 14218, "dead_horn_coral_block"},
		{14219, 14219, "ochre_froglight"},
		{14220, 14220, "verdant_froglight"},
		{14221, 14221, "pearlescent_froglight"},
	} {
		eng.SetBlock(pos.X, pos.Y, pos.Z, block.id)
		if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != block.wantDrop || !harvested {
			t.Fatalf("hand-mined %s drop = %d, harvested=%v; want %d", block.name, drop, harvested, block.wantDrop)
		}
	}
}

func TestJava263LiveCoralRandomTick(t *testing.T) {
	t.Run("water keeps coral alive", func(t *testing.T) {
		eng := NewEngine()
		eng.SetBlock(0, 64, 0, 14209)
		eng.SetBlock(1, 64, 0, 9)
		eng.mu.Lock()
		changed := eng.tickCoralBlockLocked(0, 64, 0)
		eng.mu.Unlock()
		if changed || eng.GetBlock(0, 64, 0) != 14209 {
			t.Fatal("live coral adjacent to water should remain live")
		}
	})

	t.Run("exposed dry coral dies", func(t *testing.T) {
		eng := NewEngine()
		eng.SetBlock(0, 64, 0, 14212)
		eng.mu.Lock()
		changed := eng.tickCoralBlockLocked(0, 64, 0)
		eng.mu.Unlock()
		if !changed || eng.GetBlock(0, 64, 0) != 14217 {
			t.Fatalf("exposed dry fire coral = %d, changed=%v; want dead fire coral (14217)", eng.GetBlock(0, 64, 0), changed)
		}
	})

	t.Run("sealed dry coral survives", func(t *testing.T) {
		eng := NewEngine()
		eng.SetBlock(0, 64, 0, 14210)
		for _, offset := range [][3]int32{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
			eng.SetBlock(offset[0], 64+offset[1], offset[2], inventory.ItemStone)
		}
		eng.mu.Lock()
		changed := eng.tickCoralBlockLocked(0, 64, 0)
		eng.mu.Unlock()
		if changed || eng.GetBlock(0, 64, 0) != 14210 {
			t.Fatal("dry coral fully surrounded by solid blocks should remain live")
		}
	})
}
