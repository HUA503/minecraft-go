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

func TestToolTiersAndDurability(t *testing.T) {
	// Diamond pickaxe mining speed on stone vs bare hand
	handMult := CalculateMiningSpeedMultiplier(ItemAir, ItemStone)
	if handMult != 1.0 {
		t.Fatalf("expected 1.0 for hand, got %f", handMult)
	}
	diaMult := CalculateMiningSpeedMultiplier(ItemDiamondPickaxe, ItemStone)
	if diaMult != 8.0 {
		t.Fatalf("expected 8.0 for diamond pickaxe, got %f", diaMult)
	}

	// Harvestability: Diamond Ore requires at least Iron Pickaxe
	if CanHarvestBlock(ItemWoodenPickaxe, 56) {
		t.Fatalf("wooden pickaxe should NOT be able to harvest diamond ore")
	}
	if CanHarvestBlock(ItemStonePickaxe, 56) {
		t.Fatalf("stone pickaxe should NOT be able to harvest diamond ore")
	}
	if !CanHarvestBlock(ItemIronPickaxe, 56) {
		t.Fatalf("iron pickaxe should be able to harvest diamond ore")
	}
	if !CanHarvestBlock(ItemDiamondPickaxe, 56) {
		t.Fatalf("diamond pickaxe should be able to harvest diamond ore")
	}

	// Tool wear & breaking: 59 uses total (Wood)
	stack := &ItemStack{ItemID: ItemWoodenPickaxe, Count: 1, Damage: 57}
	broken := ApplyToolWear(stack)
	if broken || stack.Damage != 58 {
		t.Fatalf("wooden pickaxe at 57 wear + 1 should reach 58 without breaking yet: broken=%v, damage=%d", broken, stack.Damage)
	}
	// 59th use breaks it
	broken = ApplyToolWear(stack)
	if !broken || !stack.IsEmpty() {
		t.Fatalf("wooden pickaxe should break when reaching 59 damage: broken=%v, stack=%+v", broken, stack)
	}
}

func TestFurnaceSmeltingCycle(t *testing.T) {
	furnace := NewFurnaceBlockEntity()
	furnace.InputSlot = ItemStack{ItemID: 15, Count: 1}  // 1 Raw Iron Ore
	furnace.FuelSlot = ItemStack{ItemID: ItemCoal, Count: 1} // 1 Coal (burns 1600 ticks)

	// Before ticking: not burning, cook progress 0
	if furnace.IsBurning() || furnace.CookTimeProgress != 0 {
		t.Fatalf("furnace should be idle initially")
	}

	// Tick 1: Coal ignites, burn time becomes 1600, cook progress becomes 1
	furnace.Tick()
	if !furnace.IsBurning() || furnace.FuelSlot.Count != 0 {
		t.Fatalf("coal should have been consumed to ignite furnace: fuelCount=%d, burning=%v", furnace.FuelSlot.Count, furnace.IsBurning())
	}
	if furnace.BurnTimeRemaining != 1600 {
		t.Fatalf("expected 1600 burn time, got %d", furnace.BurnTimeRemaining)
	}

	// Fast-forward 199 ticks (total 200 ticks = 10s cook time)
	for i := 0; i < 199; i++ {
		furnace.Tick()
	}

	// Check completed smelt: Input should be empty, Output should be 1 Iron Ingot (ItemIronIngot = 265)
	if !furnace.InputSlot.IsEmpty() {
		t.Fatalf("input should be consumed, got %+v", furnace.InputSlot)
	}
	if furnace.OutputSlot.ItemID != ItemIronIngot || furnace.OutputSlot.Count != 1 {
		t.Fatalf("expected 1 Iron Ingot in output, got %+v", furnace.OutputSlot)
	}
	// Furnace still has 1600 - 199 = 1401 ticks of fire left
	if furnace.BurnTimeRemaining != 1401 {
		t.Fatalf("expected 1401 burn time left, got %d", furnace.BurnTimeRemaining)
	}
}

func TestToolStackLimit(t *testing.T) {
	inv := NewPlayerInventory()

	// Adding 2 wooden pickaxes should occupy slot 0 and slot 1 separately (Count=1 each), not stack together
	left := inv.AddItem(ItemWoodenPickaxe, 2)
	if left != 0 {
		t.Fatalf("expected 0 remaining, got %d", left)
	}
	if inv.Slots[0].ItemID != ItemWoodenPickaxe || inv.Slots[0].Count != 1 {
		t.Fatalf("slot 0 must have exactly 1 pickaxe: %+v", inv.Slots[0])
	}
	if inv.Slots[1].ItemID != ItemWoodenPickaxe || inv.Slots[1].Count != 1 {
		t.Fatalf("slot 1 must have exactly 1 pickaxe: %+v", inv.Slots[1])
	}

	// Adding another pickaxe should go into slot 2, never stack onto slot 0 or slot 1
	left = inv.AddItem(ItemWoodenPickaxe, 1)
	if left != 0 {
		t.Fatalf("expected 0 remaining, got %d", left)
	}
	if inv.Slots[0].Count != 1 || inv.Slots[1].Count != 1 || inv.Slots[2].Count != 1 {
		t.Fatalf("tools must never stack: slot0=%d, slot1=%d, slot2=%d",
			inv.Slots[0].Count, inv.Slots[1].Count, inv.Slots[2].Count)
	}
}
