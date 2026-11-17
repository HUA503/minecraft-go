package inventory

import (
	"testing"
)

func TestInventoryStackingAndRemoval(t *testing.T) {
	inv := NewPlayerInventory()

	// Add 40 cobblestone
	left := inv.AddItem(ItemCobblestone, 40)
	if left != 0 {
		t.Fatalf("expected 0 left, got %d", left)
	}
	if inv.Slots[0].ItemID != ItemCobblestone || inv.Slots[0].Count != 40 {
		t.Fatalf("slot 0 mismatch: %+v", inv.Slots[0])
	}

	// Add 30 more cobblestone -> should stack to 64 in slot 0, and 6 in slot 1
	left = inv.AddItem(ItemCobblestone, 30)
	if left != 0 {
		t.Fatalf("expected 0 left, got %d", left)
	}
	if inv.Slots[0].Count != 64 || inv.Slots[1].Count != 6 {
		t.Fatalf("stacking mismatch: slot 0=%d, slot 1=%d", inv.Slots[0].Count, inv.Slots[1].Count)
	}

	// Remove 10 items from slot 0
	if err := inv.RemoveItem(0, 10); err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if inv.Slots[0].Count != 54 {
		t.Fatalf("expected 54 in slot 0, got %d", inv.Slots[0].Count)
	}
}

func TestCraftingRecipes(t *testing.T) {
	cm := NewCraftingManager()

	// Test 1: 1 Oak Log -> 4 Planks in 2x2 grid
	gridLog := []int32{
		ItemOakLog, ItemAir,
		ItemAir, ItemAir,
	}
	out, ok := cm.MatchCrafting(gridLog, 2)
	if !ok || out.ItemID != ItemOakPlanks || out.Count != 4 {
		t.Fatalf("oak log crafting failed: %+v (ok=%v)", out, ok)
	}

	// Test 2: 4 Planks -> 1 Crafting Table in 2x2 grid
	gridTable := []int32{
		ItemOakPlanks, ItemOakPlanks,
		ItemOakPlanks, ItemOakPlanks,
	}
	out, ok = cm.MatchCrafting(gridTable, 2)
	if !ok || out.ItemID != ItemCraftingTable || out.Count != 1 {
		t.Fatalf("crafting table recipe failed: %+v", out)
	}

	// Test 3: 3 Cobblestone + 2 Sticks -> Stone Pickaxe in 3x3 grid
	gridPickaxe := []int32{
		ItemCobblestone, ItemCobblestone, ItemCobblestone,
		ItemAir, ItemStick, ItemAir,
		ItemAir, ItemStick, ItemAir,
	}
	out, ok = cm.MatchCrafting(gridPickaxe, 3)
	if !ok || out.ItemID != ItemStonePickaxe || out.Count != 1 {
		t.Fatalf("stone pickaxe recipe failed: %+v", out)
	}

	// Test 4: Furnace recipe (donut of cobblestone) in 3x3 grid
	gridFurnace := []int32{
		ItemCobblestone, ItemCobblestone, ItemCobblestone,
		ItemCobblestone, ItemAir, ItemCobblestone,
		ItemCobblestone, ItemCobblestone, ItemCobblestone,
	}
	out, ok = cm.MatchCrafting(gridFurnace, 3)
	if !ok || out.ItemID != ItemFurnace || out.Count != 1 {
		t.Fatalf("furnace recipe failed: %+v", out)
	}
}
