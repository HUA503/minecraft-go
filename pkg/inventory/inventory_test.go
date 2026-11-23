package inventory

import (
	"testing"
)

func TestCreativeCatalogHasUniqueUsableItems(t *testing.T) {
	seen := make(map[int32]struct{})
	for _, item := range CreativeCatalog(CreativeAll) {
		if item.ID <= 0 || item.Name == "" {
			t.Errorf("invalid creative item: %+v", item)
		}
		if _, duplicate := seen[item.ID]; duplicate {
			t.Errorf("duplicate creative item ID %d", item.ID)
		}
		seen[item.ID] = struct{}{}
	}
	for _, item := range CreativeCatalog(CreativeBlocks) {
		if !IsPlaceableBlock(item.ID) {
			t.Errorf("creative block %d (%s) cannot be placed", item.ID, item.Name)
		}
	}
	if got := CreativeCatalog(CreativeFood); len(got) == 0 {
		t.Fatal("food tab is empty")
	}
}

func TestCreativeCatalogSearch(t *testing.T) {
	diamonds := SearchCreativeCatalog(CreativeAll, "钻石")
	if len(diamonds) < 3 {
		t.Fatalf("search for diamond items returned only %d entries", len(diamonds))
	}
	if got := SearchCreativeCatalog(CreativeAll, "不存在的物品"); len(got) != 0 {
		t.Fatalf("unknown query returned %d entries", len(got))
	}
	if got := SearchCreativeCatalog(CreativeFood, "钻石"); len(got) != 0 {
		t.Fatalf("category-filtered search returned %d unrelated entries", len(got))
	}
}

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
	furnace.InputSlot = ItemStack{ItemID: 15, Count: 1}      // 1 Raw Iron Ore
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

func TestSlotLeftAndRightClickLogic(t *testing.T) {
	// 1. Left-click pick up
	cursor := ItemStack{}
	slot := ItemStack{ItemID: ItemOakPlanks, Count: 30}
	HandleSlotLeftClick(&cursor, &slot)
	if cursor.ItemID != ItemOakPlanks || cursor.Count != 30 || !slot.IsEmpty() {
		t.Fatalf("left click pickup failed: cursor=%+v, slot=%+v", cursor, slot)
	}

	// 2. Left-click place on empty slot
	HandleSlotLeftClick(&cursor, &slot)
	if !cursor.IsEmpty() || slot.ItemID != ItemOakPlanks || slot.Count != 30 {
		t.Fatalf("left click place failed: cursor=%+v, slot=%+v", cursor, slot)
	}

	// 3. Left-click stack matching items
	cursor = ItemStack{ItemID: ItemOakPlanks, Count: 40}
	HandleSlotLeftClick(&cursor, &slot)
	// slot should have 64, cursor should have 6 left
	if slot.Count != 64 || cursor.Count != 6 {
		t.Fatalf("left click stack failed: slot=%d, cursor=%d", slot.Count, cursor.Count)
	}

	// 4. Left-click swap different items
	cursor = ItemStack{ItemID: ItemCobblestone, Count: 10}
	HandleSlotLeftClick(&cursor, &slot)
	if cursor.ItemID != ItemOakPlanks || cursor.Count != 64 || slot.ItemID != ItemCobblestone || slot.Count != 10 {
		t.Fatalf("left click swap failed: cursor=%+v, slot=%+v", cursor, slot)
	}

	// 5. Right-click pick up half stack (rounded up)
	cursor = ItemStack{}
	slot = ItemStack{ItemID: ItemCobblestone, Count: 11}
	HandleSlotRightClick(&cursor, &slot)
	if cursor.Count != 6 || slot.Count != 5 {
		t.Fatalf("right click half pickup failed: cursor=%d, slot=%d", cursor.Count, slot.Count)
	}

	// 6. Right-click place single item onto empty slot
	slotEmpty := ItemStack{}
	HandleSlotRightClick(&cursor, &slotEmpty)
	if cursor.Count != 5 || slotEmpty.Count != 1 || slotEmpty.ItemID != ItemCobblestone {
		t.Fatalf("right click place 1 failed: cursor=%d, slot=%d", cursor.Count, slotEmpty.Count)
	}

	// 7. Right-click place single item onto matching slot
	HandleSlotRightClick(&cursor, &slotEmpty)
	if cursor.Count != 4 || slotEmpty.Count != 2 {
		t.Fatalf("right click stack 1 failed: cursor=%d, slot=%d", cursor.Count, slotEmpty.Count)
	}
}
