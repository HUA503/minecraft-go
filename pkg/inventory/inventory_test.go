package inventory

import (
	"strings"
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
	for _, category := range []CreativeCategory{CreativeBlocks, CreativeDecoration} {
		for _, item := range CreativeCatalog(category) {
			if !IsPlaceableBlock(item.ID) {
				t.Errorf("creative block %d (%s) cannot be placed", item.ID, item.Name)
			}
		}
	}
	for _, definition := range ExtendedBlockDefinitions() {
		if !IsPlaceableBlock(definition.RuntimeID) {
			t.Errorf("extended block %s is not placeable", definition.Name)
		}
	}
	if got := CreativeCatalog(CreativeFood); len(got) == 0 {
		t.Fatal("food tab is empty")
	}
	if got := SearchCreativeCatalog(CreativeNatural, "red shrub"); len(got) != 1 || got[0].ID != 10100 || got[0].Name != "红灌木" {
		t.Fatalf("26.3 red shrub is missing from the natural creative tab: %+v", got)
	}
	if got := SearchCreativeCatalog(CreativeMaterials, "bone meal"); len(got) != 1 || got[0].ID != ItemBoneMeal {
		t.Fatalf("bone meal is missing from the materials tab: %+v", got)
	}
}

func TestJava263FoodCatalogEntriesAndStackLimits(t *testing.T) {
	food := CreativeCatalog(CreativeFood)
	if len(food) != 42 {
		t.Fatalf("food tab has %d entries, want 42 entries from Java 26.3", len(food))
	}
	if got := SearchCreativeCatalog(CreativeFood, "minecraft:glow_berries"); len(got) != 1 || got[0].ID != ItemGlowBerries {
		t.Fatalf("registry-name food search = %+v, want glow berries", got)
	}
	if got := SearchCreativeCatalog(CreativeFood, "兔肉煲"); len(got) != 1 || got[0].ID != ItemRabbitStew {
		t.Fatalf("localized food search = %+v, want rabbit stew", got)
	}
	for itemID, want := range map[int32]int{
		ItemMushroomStew: 1, ItemBeetrootSoup: 1, ItemRabbitStew: 1,
		ItemHoneyBottle: 16, ItemCake: 1, ItemCarrot: 64,
	} {
		if got := GetMaxStackSize(itemID); got != want {
			t.Errorf("GetMaxStackSize(%d) = %d, want %d", itemID, got, want)
		}
	}
	if _, ok := FoodRegistry[ItemChorusFruit]; !ok || !FoodRegistry[ItemChorusFruit].AlwaysEdible {
		t.Fatal("chorus fruit must be edible at full hunger in Java 26.3")
	}
	if _, ok := FoodRegistry[ItemGoldenDandelion]; ok {
		t.Fatal("golden dandelion is used on baby animals; it is not an edible food item")
	}
	if _, ok := FoodRegistry[ItemCake]; ok {
		t.Fatal("cake is consumed as a placed block, not directly from the inventory")
	}
	if got := FoodRegistry[ItemHoneyBottle]; got.RemainderItemID != 374 || !got.AlwaysEdible {
		t.Fatalf("honey bottle consume metadata = %+v, want glass-bottle remainder and always-edible", got)
	}
}

func TestCreativeItemDisplayNameResolvesCatalogAndUnknownIDs(t *testing.T) {
	if got := CreativeItemDisplayName(ItemStone); got != "石头" {
		t.Fatalf("stone display name = %q, want 石头", got)
	}
	if got := CreativeItemDisplayName(10100); got != "红灌木" {
		t.Fatalf("red shrub display name = %q, want 红灌木", got)
	}
	if got := CreativeItemDisplayName(987654); got != "物品 #987654" {
		t.Fatalf("unknown item display name = %q", got)
	}
}

func TestExtendedBlockRegistryIsNamespacedAndExposesColorSets(t *testing.T) {
	definitions := ExtendedBlockDefinitions()
	if len(definitions) < 60 {
		t.Fatalf("extended block registry has %d entries, want recent blocks plus 48 colored blocks", len(definitions))
	}
	seenIDs := make(map[int32]bool, len(definitions))
	seenNames := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		if definition.RuntimeID < 10000 || definition.Name == "" || !strings.HasPrefix(definition.Name, "minecraft:") {
			t.Errorf("invalid extended block identity: %+v", definition)
		}
		if seenIDs[definition.RuntimeID] || seenNames[definition.Name] {
			t.Errorf("duplicate extended block identity: %+v", definition)
		}
		seenIDs[definition.RuntimeID] = true
		seenNames[definition.Name] = true
	}
}

func TestModernFullCubeBlocksDeclareMiningAndDropRules(t *testing.T) {
	byName := make(map[string]ExtendedBlockDefinition)
	for _, definition := range ExtendedBlockDefinitions() {
		byName[definition.Name] = definition
	}
	for _, name := range []string{
		"minecraft:andesite", "minecraft:polished_andesite", "minecraft:deepslate",
		"minecraft:cobbled_deepslate", "minecraft:calcite", "minecraft:tuff",
		"minecraft:dripstone_block", "minecraft:moss_block", "minecraft:mud",
		"minecraft:raw_iron_block", "minecraft:copper_block", "minecraft:oxidized_copper",
		"minecraft:blackstone", "minecraft:basalt", "minecraft:crimson_planks",
		"minecraft:pale_oak_planks", "minecraft:birch_planks", "minecraft:dark_oak_planks",
		"minecraft:birch_log", "minecraft:acacia_log", "minecraft:stripped_pale_oak_log",
		"minecraft:crimson_stem", "minecraft:stripped_bamboo_block",
	} {
		definition, ok := byName[name]
		if !ok {
			t.Errorf("missing modern full cube %s", name)
			continue
		}
		if !IsPlaceableBlock(definition.RuntimeID) || definition.Hardness <= 0 || definition.HarvestTool == ToolNone {
			t.Errorf("modern block lacks placement/mining metadata: %+v", definition)
		}
	}
	deepslate := byName["minecraft:deepslate"]
	cobbled := byName["minecraft:cobbled_deepslate"]
	if deepslate.DropRuntimeID != cobbled.RuntimeID || !deepslate.RequiresHarvestTool || deepslate.RequiredHarvestLevel != 1 {
		t.Fatalf("deepslate drop/harvest rules are incorrect: %+v (cobbled ID %d)", deepslate, cobbled.RuntimeID)
	}
	planks := byName["minecraft:pale_oak_planks"]
	if planks.HarvestTool != ToolAxe || planks.RequiresHarvestTool {
		t.Fatalf("planks must be axe-efficient without requiring an axe: %+v", planks)
	}
}

func TestColoredGlazedTerracottaIsPlaceableAndSearchable(t *testing.T) {
	items := CreativeCatalog(CreativeDecoration)
	count := 0
	for _, item := range items {
		if !strings.Contains(item.Name, "带釉陶瓦") {
			continue
		}
		count++
		block, ok := ExtendedBlockByID(item.ID)
		if !ok || !IsPlaceableBlock(item.ID) || block.Hardness <= 0 || block.HarvestTool != ToolPickaxe {
			t.Errorf("glazed terracotta is missing place/mining behavior: %+v, block=%+v", item, block)
		}
	}
	if count != 16 {
		t.Fatalf("creative catalog has %d glazed terracotta colors, want 16", count)
	}
	if got := SearchCreativeCatalog(CreativeDecoration, "cyan_glazed_terracotta"); len(got) != 1 {
		t.Fatalf("search for cyan glazed terracotta returned %d items, want one", len(got))
	}
}

func TestWoodLogsExposeStrippedVariants(t *testing.T) {
	for _, tc := range []struct {
		name   string
		baseID int32
	}{
		{"minecraft:stripped_oak_log", 17},
		{"minecraft:stripped_spruce_log", 22},
		{"minecraft:stripped_birch_log", 0},
		{"minecraft:stripped_jungle_log", 0},
		{"minecraft:stripped_acacia_log", 0},
		{"minecraft:stripped_dark_oak_log", 0},
		{"minecraft:stripped_mangrove_log", 0},
		{"minecraft:stripped_cherry_log", 0},
		{"minecraft:stripped_pale_oak_log", 0},
		{"minecraft:stripped_crimson_stem", 0},
		{"minecraft:stripped_warped_stem", 0},
		{"minecraft:stripped_bamboo_block", 0},
		{"minecraft:stripped_poplar_log", 0},
	} {
		var strippedID int32
		for _, definition := range ExtendedBlockDefinitions() {
			if definition.Name == tc.name {
				strippedID = definition.RuntimeID
				break
			}
		}
		if strippedID == 0 {
			t.Fatalf("missing stripped wood definition %s", tc.name)
		}
		baseID := tc.baseID
		if baseID == 0 {
			for _, definition := range ExtendedBlockDefinitions() {
				if definition.StrippedVariantID == strippedID {
					baseID = definition.RuntimeID
					break
				}
			}
		}
		got, ok := StrippedWoodBlockVariant(baseID)
		if !ok || got != strippedID {
			t.Errorf("stripped variant for %s base %d = (%d,%v), want %d", tc.name, baseID, got, ok, strippedID)
		}
		if _, ok := StrippedWoodBlockVariant(strippedID); ok {
			t.Errorf("already stripped block %s can be stripped again", tc.name)
		}
	}
}

func TestCreativeCatalogSearch(t *testing.T) {
	diamonds := SearchCreativeCatalog(CreativeAll, "钻石")
	if len(diamonds) < 3 {
		t.Fatalf("search for diamond items returned only %d entries", len(diamonds))
	}
	if got := SearchCreativeCatalog(CreativeAll, "minecraft:polished_deepslate"); len(got) != 1 || got[0].ID == 0 {
		t.Fatalf("registry-name search returned %+v, want polished deepslate", got)
	}
	if got := SearchCreativeCatalog(CreativeSpawnEggs, "zombie"); len(got) != 1 || got[0].Name != "僵尸刷怪蛋" {
		t.Fatalf("spawn-egg entity-name search returned %+v, want zombie egg", got)
	}
	if got := SearchCreativeCatalog(CreativeAll, "不存在的物品"); len(got) != 0 {
		t.Fatalf("unknown query returned %d entries", len(got))
	}
	if got := SearchCreativeCatalog(CreativeFood, "钻石"); len(got) != 0 {
		t.Fatalf("category-filtered search returned %d unrelated entries", len(got))
	}
}

func TestCreativeCatalogClickStackSizes(t *testing.T) {
	if got := CreativeItemStack(ItemStone, 1); got.ItemID != ItemStone || got.Count != 64 {
		t.Fatalf("left click stack = %+v, want 64 stone", got)
	}
	if got := CreativeItemStack(ItemStone, 2); got.ItemID != ItemStone || got.Count != 1 {
		t.Fatalf("right click stack = %+v, want 1 stone", got)
	}
	if got := CreativeItemStack(ItemStone, 3); got.ItemID != ItemStone || got.Count != 64 {
		t.Fatalf("middle click stack = %+v, want 64 stone", got)
	}
	if got := CreativeItemStack(267, 1); got.ItemID != 267 || got.Count != 1 {
		t.Fatalf("left click non-stackable = %+v, want 1 iron sword", got)
	}
	if got := CreativeItemStack(ItemStone, 0); !got.IsEmpty() {
		t.Fatalf("invalid button stack = %+v, want empty", got)
	}
}

func TestSwapSelectedWithOffhand(t *testing.T) {
	inv := NewPlayerInventory()
	inv.SelectedHotbar = 4
	inv.Slots[4] = ItemStack{ItemID: ItemTorch, Count: 12}
	inv.Offhand = ItemStack{ItemID: ItemDiamond, Count: 1}
	if !inv.SwapSelectedWithOffhand() {
		t.Fatal("valid selected slot did not swap")
	}
	if inv.Slots[4] != (ItemStack{ItemID: ItemDiamond, Count: 1}) || inv.Offhand != (ItemStack{ItemID: ItemTorch, Count: 12}) {
		t.Fatalf("main/offhand swap result: main=%+v offhand=%+v", inv.Slots[4], inv.Offhand)
	}
	inv.SelectedHotbar = 9
	if inv.SwapSelectedWithOffhand() {
		t.Fatal("invalid selected slot was accepted")
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
	boneMeal, ok := cm.MatchCrafting([]int32{ItemBone, ItemAir, ItemAir, ItemAir}, 2)
	if !ok || boneMeal.ItemID != ItemBoneMeal || boneMeal.Count != 3 {
		t.Fatalf("bone meal recipe = %+v, %v; want 3 bone meal", boneMeal, ok)
	}

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

func TestArmorSlotMappingAndCreativeClickRules(t *testing.T) {
	for _, tc := range []struct {
		itemID int32
		slot   int
	}{{301, 0}, {305, 0}, {300, 1}, {304, 1}, {299, 2}, {303, 2}, {298, 3}, {302, 3}, {317, 0}, {314, 3}, {ItemCopperBoots, 0}, {ItemCopperLeggings, 1}, {ItemCopperChestplate, 2}, {ItemCopperHelmet, 3}} {
		if slot, ok := ArmorSlotForItem(tc.itemID); !ok || slot != tc.slot {
			t.Errorf("ArmorSlotForItem(%d) = (%d, %v), want (%d, true)", tc.itemID, slot, ok, tc.slot)
		}
	}
	if CanEquipArmor(3, ItemDiamondBoots) || CanEquipArmor(0, ItemDiamondHelmet) {
		t.Fatal("armor item accepted in the wrong equipment slot")
	}

	var cursor = ItemStack{ItemID: ItemDiamondHelmet, Count: 1}
	var helmet ItemStack
	HandleArmorSlotClick(&cursor, &helmet, 3, 1)
	if !cursor.IsEmpty() || helmet.ItemID != ItemDiamondHelmet || helmet.Count != 1 {
		t.Fatalf("valid helmet was not equipped: cursor=%+v helmet=%+v", cursor, helmet)
	}
	cursor = ItemStack{ItemID: ItemDiamondBoots, Count: 1}
	HandleArmorSlotClick(&cursor, &helmet, 3, 1)
	if cursor.ItemID != ItemDiamondBoots || helmet.ItemID != ItemDiamondHelmet {
		t.Fatalf("invalid armor swap changed equipment: cursor=%+v helmet=%+v", cursor, helmet)
	}
	HandleArmorSlotClick(&cursor, &helmet, 3, 3)
	if cursor.ItemID != ItemDiamondHelmet || helmet.ItemID != ItemDiamondHelmet {
		t.Fatalf("middle click did not clone the equipped armor: cursor=%+v helmet=%+v", cursor, helmet)
	}
}

func TestArmorProtectionTotalsMatchArmorMaterials(t *testing.T) {
	var armor [4]ItemStack
	armor[0] = ItemStack{ItemID: ItemDiamondBoots, Count: 1}
	armor[1] = ItemStack{ItemID: ItemDiamondLeggings, Count: 1}
	armor[2] = ItemStack{ItemID: ItemDiamondChestplate, Count: 1}
	armor[3] = ItemStack{ItemID: ItemDiamondHelmet, Count: 1}
	if got := CalculateArmorProtection(armor); got.Defense != 20 || got.Toughness != 8 {
		t.Fatalf("full diamond protection = %+v, want 20 defense and 8 toughness", got)
	}
	armor[3] = ItemStack{ItemID: ItemIronHelmet, Count: 1}
	if got := CalculateArmorProtection(armor); got.Defense != 19 || got.Toughness != 6 {
		t.Fatalf("mixed armor protection = %+v, want 19 defense and 6 toughness", got)
	}
	armor[0] = ItemStack{ItemID: ItemDiamondHelmet, Count: 1} // invalid boots slot
	if got := CalculateArmorProtection(armor); got.Defense != 16 || got.Toughness != 4 {
		t.Fatalf("wrong-slot item contributed protection: %+v", got)
	}
}

func TestCopperArmorIsSearchableAndHasVanillaProtection(t *testing.T) {
	items := SearchCreativeCatalog(CreativeCombat, "copper_helmet")
	if len(items) != 1 || items[0].ID != ItemCopperHelmet {
		t.Fatalf("copper armor registry-name search = %+v", items)
	}
	var armor [4]ItemStack
	armor[0] = ItemStack{ItemID: ItemCopperBoots, Count: 1}
	armor[1] = ItemStack{ItemID: ItemCopperLeggings, Count: 1}
	armor[2] = ItemStack{ItemID: ItemCopperChestplate, Count: 1}
	armor[3] = ItemStack{ItemID: ItemCopperHelmet, Count: 1}
	if got := CalculateArmorProtection(armor); got.Defense != 10 || got.Toughness != 0 {
		t.Fatalf("full copper protection = %+v, want 10 defense and no toughness", got)
	}
	for _, id := range []int32{ItemCopperHelmet, ItemCopperChestplate, ItemCopperLeggings, ItemCopperBoots} {
		if GetMaxStackSize(id) != 1 {
			t.Errorf("copper armor %d max stack size = %d, want 1", id, GetMaxStackSize(id))
		}
	}
}

func TestAddedOreBlocksAreSearchableAndHaveHarvestDrops(t *testing.T) {
	want := map[string]struct {
		drop  int32
		level int
		tool  ToolType
		min   int
		max   int
	}{
		"minecraft:copper_ore":             {ItemRawCopper, 1, ToolPickaxe, 2, 5},
		"minecraft:deepslate_copper_ore":   {ItemRawCopper, 1, ToolPickaxe, 2, 5},
		"minecraft:deepslate_coal_ore":     {ItemCoal, 0, ToolPickaxe, 1, 1},
		"minecraft:deepslate_iron_ore":     {ItemRawIron, 1, ToolPickaxe, 1, 1},
		"minecraft:deepslate_gold_ore":     {ItemRawGold, 2, ToolPickaxe, 1, 1},
		"minecraft:deepslate_diamond_ore":  {ItemDiamond, 2, ToolPickaxe, 1, 1},
		"minecraft:deepslate_emerald_ore":  {ItemEmerald, 2, ToolPickaxe, 1, 1},
		"minecraft:deepslate_lapis_ore":    {ItemLapisLazuli, 1, ToolPickaxe, 4, 9},
		"minecraft:deepslate_redstone_ore": {331, 2, ToolPickaxe, 4, 5},
		"minecraft:nether_quartz_ore":      {ItemNetherQuartz, 0, ToolPickaxe, 1, 1},
		"minecraft:ancient_debris":         {0, 3, ToolPickaxe, 1, 1},
	}
	for name, expected := range want {
		block, ok := ExtendedBlockByName(name)
		if !ok {
			t.Errorf("missing block registration %s", name)
			continue
		}
		if !IsPlaceableBlock(block.RuntimeID) || block.Hardness <= 0 || !block.RequiresHarvestTool || block.HarvestTool != expected.tool || block.RequiredHarvestLevel != expected.level {
			t.Errorf("%s has incomplete placement/mining metadata: %+v", name, block)
		}
		if expected.drop != 0 && block.DropRuntimeID != expected.drop {
			t.Errorf("%s drop=%d, want %d", name, block.DropRuntimeID, expected.drop)
		}
		if min, max := BlockDropCountRange(block.RuntimeID); min != expected.min || max != expected.max {
			t.Errorf("%s drop count range=(%d,%d), want (%d,%d)", name, min, max, expected.min, expected.max)
		}
		found := false
		for _, item := range SearchCreativeCatalog(CreativeAll, name) {
			if item.ID == block.RuntimeID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("creative search %q did not include block ID %d", name, block.RuntimeID)
		}
	}
}

func TestNewOreMaterialsHaveSmeltingRecipesAndCreativeEntries(t *testing.T) {
	want := map[int32]int32{
		ItemRawCopper: ItemCopperIngot,
		ItemRawIron:   ItemIronIngot,
		ItemRawGold:   ItemGoldIngot,
	}
	for input, output := range want {
		recipe, ok := MatchSmelting(input)
		if !ok || recipe.OutputID != output {
			t.Errorf("smelting recipe for %d = %+v, %t; want output %d", input, recipe, ok, output)
		}
		if got := SearchCreativeCatalog(CreativeMaterials, creativeItemRegistryName(input)); len(got) != 1 || got[0].ID != input {
			t.Errorf("creative material search for %d = %+v", input, got)
		}
	}
	ancientDebris, ok := ExtendedBlockByName("minecraft:ancient_debris")
	if !ok {
		t.Fatal("ancient debris block is missing")
	}
	recipe, ok := MatchSmelting(ancientDebris.RuntimeID)
	if !ok || recipe.OutputID != ItemNetheriteScrap {
		t.Fatalf("ancient debris smelting = %+v, %t; want netherite scrap", recipe, ok)
	}
	furnace := NewFurnaceBlockEntity()
	furnace.InputSlot = ItemStack{ItemID: ItemRawCopper, Count: 1}
	furnace.FuelSlot = ItemStack{ItemID: ItemCoal, Count: 1}
	for i := 0; i < furnace.CookTimeTotal; i++ {
		furnace.Tick()
	}
	if got := furnace.OutputSlot; got.ItemID != ItemCopperIngot || got.Count != 1 {
		t.Fatalf("raw copper smelt output = %+v, want 1 copper ingot", got)
	}
}

func TestLapisLazuliHasCreativeRegistryName(t *testing.T) {
	items := SearchCreativeCatalog(CreativeMaterials, "minecraft:lapis_lazuli")
	if len(items) != 1 || items[0].ID != ItemLapisLazuli {
		t.Fatalf("lapis creative search = %+v, want one lapis lazuli item", items)
	}
}

func TestEveryCreativeCatalogItemHasSearchableVanillaRegistryName(t *testing.T) {
	for _, item := range CreativeCatalog(CreativeAll) {
		registryName := creativeItemRegistryName(item.ID)
		if !strings.HasPrefix(registryName, "minecraft:") {
			t.Errorf("creative item %q (id %d) has no vanilla registry name", item.Name, item.ID)
			continue
		}
		query := strings.TrimPrefix(registryName, "minecraft:")
		matches := SearchCreativeCatalog(CreativeAll, query)
		found := false
		for _, match := range matches {
			if match.ID == item.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("creative item %q (id %d) cannot be found by %q", item.Name, item.ID, query)
		}
	}
}
