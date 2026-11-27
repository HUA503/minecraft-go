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
	eng.ItemDrops = make(map[int32]*ItemDrop)
	eng.SetBlock(pos.X, pos.Y, pos.Z, 14223) // Gilded blackstone requires a pickaxe and drops 2–5 nuggets.
	if drop, _ := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != inventory.ItemAir {
		t.Fatalf("hand-mined gilded blackstone drop = %d, want no drop", drop)
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 14223)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenPickaxe); drop != inventory.ItemGoldNugget || !harvested {
		t.Fatalf("pickaxe-mined gilded blackstone drop = %d, want gold nuggets", drop)
	}
	got = 0
	for _, drop := range eng.GetItemDrops() {
		if drop.ItemID == inventory.ItemGoldNugget {
			got += drop.Count
		}
	}
	if got < 2 || got > 5 {
		t.Fatalf("gilded blackstone yielded %d nuggets, want 2..5", got)
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

	eng.SetBlock(pos.X, pos.Y, pos.Z, 14222) // Reinforced deepslate takes a long time and drops nothing.
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != inventory.ItemAir || !harvested {
		t.Fatalf("reinforced deepslate mining result = drop %d, harvested %v; want no drop after successful break", drop, harvested)
	}
}

func TestJava263IronBarsRequirePickaxeForDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Miner", [16]byte{43}, Vec3{X: 0, Y: 64, Z: 0})
	if player == nil {
		t.Fatal("expected initialized mining player")
	}
	pos := world.BlockPos{X: 3, Y: 64, Z: 0}
	eng.SetBlock(pos.X, pos.Y, pos.Z, inventory.ItemIronBars)
	if drop, _ := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); drop != inventory.ItemAir {
		t.Fatalf("hand-mined iron bars drop = %d, want none", drop)
	}
	eng.SetBlock(pos.X, pos.Y, pos.Z, inventory.ItemIronBars)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenPickaxe); drop != inventory.ItemIronBars || !harvested {
		t.Fatalf("pickaxe-mined iron bars drop = %d, harvested=%t; want iron bars", drop, harvested)
	}
}

func TestJava263IronChainAxisStateDropsCanonicalItem(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Chain miner", [16]byte{44}, Vec3{X: 0, Y: 64, Z: 0})
	if player == nil {
		t.Fatal("expected initialized mining player")
	}
	for axis := 0; axis < 3; axis++ {
		state, ok := inventory.IronChainBlockState(axis)
		if !ok {
			t.Fatalf("axis %d has no iron-chain state", axis)
		}
		pos := world.BlockPos{X: int32(3 + axis*2), Y: 64, Z: 0}
		eng.SetBlock(pos.X, pos.Y, pos.Z, state)
		drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenPickaxe)
		if drop != inventory.ItemIronChain || !harvested {
			t.Errorf("axis %d chain state %d drop=%d harvested=%v, want canonical iron-chain item", axis, state, drop, harvested)
		}
	}
}

func TestJava263WeightedPressurePlatesCountEntitiesAndRelease(t *testing.T) {
	for _, tc := range []struct {
		name   string
		itemID int32
		drops  int
		mobs   int
		want   int
	}{
		{"light plate counts one item entity", inventory.ItemLightWeightedPressurePlate, 1, 0, 1},
		{"heavy plate is off with no entity", inventory.ItemHeavyWeightedPressurePlate, 0, 0, 0},
		{"heavy plate rounds one mob up to one signal", inventory.ItemHeavyWeightedPressurePlate, 0, 1, 1},
		{"heavy plate counts mixed mob and item entities", inventory.ItemHeavyWeightedPressurePlate, 1, 10, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eng := NewEngine()
			pos := world.BlockPos{X: 5, Y: 64, Z: 5}
			eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemStone)
			initial, ok := inventory.PressurePlatePlacementState(tc.itemID)
			if !ok || !eng.SetBlockChecked(pos.X, pos.Y, pos.Z, initial) {
				t.Fatal("failed to place weighted pressure plate")
			}
			for i := 0; i < tc.drops; i++ {
				eng.ItemDrops[int32(i+1)] = &ItemDrop{ID: int32(i + 1), ItemID: inventory.ItemStone, Count: 64, Position: Vec3{X: 5.5, Y: 65, Z: 5.5}}
			}
			for i := 0; i < tc.mobs; i++ {
				mob := NewMobEntity(int32(i+100), MobCow, Vec3{X: 5.5, Y: 64.1, Z: 5.5})
				eng.Mobs[mob.ID] = mob
			}
			eng.mu.Lock()
			eng.updatePressurePlatesLocked()
			eng.mu.Unlock()
			state := eng.GetBlock(pos.X, pos.Y, pos.Z)
			if got := inventory.PressurePlateSignalStrength(state); got != tc.want {
				t.Fatalf("signal with %d item drops and %d mobs=%d, want %d", tc.drops, tc.mobs, got, tc.want)
			}
			eng.ItemDrops = make(map[int32]*ItemDrop)
			eng.Mobs = make(map[int32]*MobEntity)
			eng.mu.Lock()
			eng.updatePressurePlatesLocked()
			eng.mu.Unlock()
			if got := inventory.PressurePlateSignalStrength(eng.GetBlock(pos.X, pos.Y, pos.Z)); got != 0 {
				t.Fatalf("signal after entities leave=%d, want 0", got)
			}
		})
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
