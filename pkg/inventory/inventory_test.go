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

func TestSlabBlockStatesShareCanonicalItemsAndShapes(t *testing.T) {
	if min, max := BlockDropCountRange(BlockStoneSlabDouble); min != 2 || max != 2 {
		t.Fatalf("double stone slab drop range=(%d,%d), want (2,2)", min, max)
	}
	if min, max := BlockDropCountRange(BlockOakSlabDouble); min != 2 || max != 2 {
		t.Fatalf("double oak slab drop range=(%d,%d), want (2,2)", min, max)
	}
	for _, slab := range woodSlabDefinitions {
		definition, ok := ExtendedBlockByID(slab.ItemID)
		if !ok || definition.Name != slab.Name || definition.TextureTop != slab.Texture {
			t.Fatalf("wood slab %s is not registered with its texture: %+v, found=%t", slab.Name, definition, ok)
		}
		if matches := SearchCreativeCatalog(CreativeBlocks, slab.Name); len(matches) != 1 || matches[0].ID != slab.ItemID {
			t.Errorf("wood slab %s missing from Blocks creative tab: %+v", slab.Name, matches)
		}
		for _, state := range []struct {
			id     int32
			minY   float32
			maxY   float32
			double bool
		}{
			{slab.ItemID, 0, .5, false},
			{FirstWoodSlabTop + (slab.ItemID - FirstWoodSlabItem), .5, 1, false},
			{FirstWoodSlabDouble + (slab.ItemID - FirstWoodSlabItem), 0, 1, true},
		} {
			boxes, found := BlockCollisionBoxes(state.id)
			if !found || len(boxes) != 1 || boxes[0].MinY != state.minY || boxes[0].MaxY != state.maxY {
				t.Errorf("slab %s state %d shape = %+v, found=%t", slab.Name, state.id, boxes, found)
			}
			if got, found := SlabItemForBlockState(state.id); !found || got != slab.ItemID {
				t.Errorf("slab %s state %d canonical item = %d, found=%t", slab.Name, state.id, got, found)
			}
			if state.double {
				if min, max := BlockDropCountRange(state.id); min != 2 || max != 2 {
					t.Errorf("double slab %s drop range=(%d,%d), want (2,2)", slab.Name, min, max)
				}
			}
		}
		if got, ok := SlabPlacementState(slab.ItemID, 0, .75); !ok || got != FirstWoodSlabTop+(slab.ItemID-FirstWoodSlabItem) {
			t.Errorf("top placement for %s = (%d,%t)", slab.Name, got, ok)
		}
		if got, ok := SlabMergeState(slab.ItemID, slab.ItemID, 0, .75); !ok || got != FirstWoodSlabDouble+(slab.ItemID-FirstWoodSlabItem) {
			t.Errorf("same-material merge for %s = (%d,%t)", slab.Name, got, ok)
		}
	}
	for _, slab := range colorSlabDefinitions {
		definition, ok := ExtendedBlockByID(slab.ItemID)
		if !ok || definition.Name != slab.Name || definition.TextureTop != slab.Texture {
			t.Fatalf("colored slab %s is not registered with its texture: %+v, found=%t", slab.Name, definition, ok)
		}
		foundInDecorationTab := false
		for _, item := range CreativeCatalog(CreativeDecoration) {
			if item.ID == slab.ItemID && creativeItemRegistryName(item.ID) == slab.Name {
				foundInDecorationTab = true
				break
			}
		}
		if !foundInDecorationTab {
			t.Errorf("colored slab %s missing from Decoration creative tab", slab.Name)
		}
		if slab.RequiresTool && CanHarvestBlock(ItemAir, slab.ItemID) {
			t.Errorf("colored slab %s should require a pickaxe to drop", slab.Name)
		}
		if slab.RequiresTool && !CanHarvestBlock(ItemWoodenPickaxe, slab.ItemID) {
			t.Errorf("colored slab %s should drop with a wooden pickaxe", slab.Name)
		}
		for _, state := range []struct {
			id   int32
			minY float32
			maxY float32
		}{
			{slab.ItemID, 0, .5},
			{FirstColorSlabTop + (slab.ItemID - FirstColorSlabItem), .5, 1},
			{FirstColorSlabDouble + (slab.ItemID - FirstColorSlabItem), 0, 1},
		} {
			boxes, found := BlockCollisionBoxes(state.id)
			if !found || len(boxes) != 1 || boxes[0].MinY != state.minY || boxes[0].MaxY != state.maxY {
				t.Errorf("colored slab %s state %d shape = %+v, found=%t", slab.Name, state.id, boxes, found)
			}
			if got, found := SlabItemForBlockState(state.id); !found || got != slab.ItemID {
				t.Errorf("colored slab %s state %d canonical item = %d, found=%t", slab.Name, state.id, got, found)
			}
		}
	}
	if len(masonrySlabDefinitions) != 45 {
		t.Fatalf("masonry slab family has %d definitions, want 45", len(masonrySlabDefinitions))
	}
	for _, slab := range masonrySlabDefinitions {
		definition, ok := ExtendedBlockByID(slab.ItemID)
		if !ok || definition.Name != slab.Name || definition.TextureTop != slab.TextureTop || !definition.RequiresHarvestTool || definition.HarvestTool != ToolPickaxe {
			t.Fatalf("masonry slab %s registration = %+v, found=%t", slab.Name, definition, ok)
		}
		foundInBlocksTab := false
		for _, item := range CreativeCatalog(CreativeBlocks) {
			if item.ID == slab.ItemID && creativeItemRegistryName(item.ID) == slab.Name {
				foundInBlocksTab = true
				break
			}
		}
		if !foundInBlocksTab {
			t.Errorf("masonry slab %s missing from Blocks creative tab", slab.Name)
		}
		if CanHarvestBlock(ItemAir, slab.ItemID) || !CanHarvestBlock(ItemWoodenPickaxe, slab.ItemID) {
			t.Errorf("masonry slab %s should require a pickaxe and accept a wooden pickaxe", slab.Name)
		}
		for _, state := range []struct {
			id   int32
			minY float32
			maxY float32
		}{
			{slab.ItemID, 0, .5},
			{FirstMasonrySlabTop + (slab.ItemID - FirstMasonrySlabItem), .5, 1},
			{FirstMasonrySlabDouble + (slab.ItemID - FirstMasonrySlabItem), 0, 1},
		} {
			boxes, found := BlockCollisionBoxes(state.id)
			if !found || len(boxes) != 1 || boxes[0].MinY != state.minY || boxes[0].MaxY != state.maxY {
				t.Errorf("masonry slab %s state %d shape = %+v, found=%t", slab.Name, state.id, boxes, found)
			}
			if got, found := SlabItemForBlockState(state.id); !found || got != slab.ItemID {
				t.Errorf("masonry slab %s state %d canonical item = %d, found=%t", slab.Name, state.id, got, found)
			}
			if IsDoubleSlabState(state.id) {
				if min, max := BlockDropCountRange(state.id); min != 2 || max != 2 {
					t.Errorf("double masonry slab %s drop range=(%d,%d), want (2,2)", slab.Name, min, max)
				}
			}
		}
	}
	for _, tc := range []struct {
		state int32
		item  int32
		minY  float32
		maxY  float32
	}{
		{ItemStoneSlab, ItemStoneSlab, 0, .5},
		{BlockStoneSlabTop, ItemStoneSlab, .5, 1},
		{BlockStoneSlabDouble, ItemStoneSlab, 0, 1},
		{ItemOakSlab, ItemOakSlab, 0, .5},
		{BlockOakSlabTop, ItemOakSlab, .5, 1},
		{BlockOakSlabDouble, ItemOakSlab, 0, 1},
	} {
		boxes, ok := BlockCollisionBoxes(tc.state)
		if !ok || len(boxes) != 1 {
			t.Fatalf("slab state %d boxes = %+v, recognized=%t", tc.state, boxes, ok)
		}
		if boxes[0].MinY != tc.minY || boxes[0].MaxY != tc.maxY {
			t.Errorf("slab state %d vertical shape = [%g,%g], want [%g,%g]", tc.state, boxes[0].MinY, boxes[0].MaxY, tc.minY, tc.maxY)
		}
		if got, found := SlabItemForBlockState(tc.state); !found || got != tc.item || BlockItemForState(tc.state) != tc.item {
			t.Errorf("slab state %d canonical item = %d, recognized=%t; want %d", tc.state, got, found, tc.item)
		}
	}
	if got, ok := SlabPlacementState(ItemOakSlab, 0, .75); !ok || got != BlockOakSlabTop {
		t.Errorf("placing against upper side half = (%d,%t), want top oak slab", got, ok)
	}
	if got, ok := SlabPlacementState(ItemOakSlab, 0, .25); !ok || got != ItemOakSlab {
		t.Errorf("placing against lower side half = (%d,%t), want bottom oak slab", got, ok)
	}
	if got, ok := SlabMergeState(ItemOakSlab, ItemOakSlab, 0, .75); !ok || got != BlockOakSlabDouble {
		t.Errorf("upper-side click on bottom slab merge = (%d,%t), want double oak slab", got, ok)
	}
	if _, ok := SlabMergeState(ItemOakSlab, ItemOakSlab, 0, .25); ok {
		t.Error("lower-side click on bottom slab should not merge same half")
	}
	if !IsPlaceableBlock(ItemStoneSlab) {
		t.Fatal("stone slab item should remain placeable")
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
		"minecraft:poplar_wood", "minecraft:stripped_poplar_wood",
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

func TestPoplarWoodVariantsAreCreativePlaceableAndStrippable(t *testing.T) {
	wood, ok := ExtendedBlockByName("minecraft:poplar_wood")
	if !ok {
		t.Fatal("poplar wood is missing from the block registry")
	}
	stripped, ok := ExtendedBlockByName("minecraft:stripped_poplar_wood")
	if !ok {
		t.Fatal("stripped poplar wood is missing from the block registry")
	}
	if wood.StrippedVariantID != stripped.RuntimeID {
		t.Fatalf("poplar wood strips to %d, want %d", wood.StrippedVariantID, stripped.RuntimeID)
	}
	if got, ok := StrippedWoodBlockVariant(wood.RuntimeID); !ok || got != stripped.RuntimeID {
		t.Fatalf("StrippedWoodBlockVariant(poplar_wood) = %d, %t; want %d, true", got, ok, stripped.RuntimeID)
	}
	for _, tc := range []struct {
		definition ExtendedBlockDefinition
		name       string
	}{
		{wood, "杨木"},
		{stripped, "去皮杨木"},
	} {
		if !IsPlaceableBlock(tc.definition.RuntimeID) || tc.definition.Hardness != 2 || tc.definition.HarvestTool != ToolAxe {
			t.Errorf("poplar wood block lacks placement or mining behavior: %+v", tc.definition)
		}
		if got := CreativeItemDisplayName(tc.definition.RuntimeID); got != tc.name {
			t.Errorf("creative display name for %s = %q, want %q", tc.definition.Name, got, tc.name)
		}
	}
}

func TestWoodBlockVariantsAreCreativePlaceableAndStrippable(t *testing.T) {
	for _, tc := range []struct {
		wood, stripped, texture, display string
	}{
		{"oak_wood", "stripped_oak_wood", "oak_log", "橡木"},
		{"spruce_wood", "stripped_spruce_wood", "spruce_log", "云杉木"},
		{"birch_wood", "stripped_birch_wood", "birch_log", "白桦木"},
		{"jungle_wood", "stripped_jungle_wood", "jungle_log", "丛林木"},
		{"acacia_wood", "stripped_acacia_wood", "acacia_log", "金合欢木"},
		{"dark_oak_wood", "stripped_dark_oak_wood", "dark_oak_log", "深色橡木"},
		{"mangrove_wood", "stripped_mangrove_wood", "mangrove_log", "红树木"},
		{"cherry_wood", "stripped_cherry_wood", "cherry_log", "樱花木"},
		{"pale_oak_wood", "stripped_pale_oak_wood", "pale_oak_log", "苍白橡木"},
		{"crimson_hyphae", "stripped_crimson_hyphae", "crimson_stem", "绯红菌核"},
		{"warped_hyphae", "stripped_warped_hyphae", "warped_stem", "诡异菌核"},
	} {
		wood, ok := ExtendedBlockByName("minecraft:" + tc.wood)
		if !ok {
			t.Fatalf("missing wood block %s", tc.wood)
		}
		stripped, ok := ExtendedBlockByName("minecraft:" + tc.stripped)
		if !ok {
			t.Fatalf("missing stripped wood block %s", tc.stripped)
		}
		if wood.StrippedVariantID != stripped.RuntimeID || wood.TextureSide != tc.texture || wood.DropRuntimeID != wood.RuntimeID {
			t.Errorf("wood definition is inconsistent: %+v; stripped=%+v", wood, stripped)
		}
		if stripped.TextureSide != "stripped_"+tc.texture || stripped.DropRuntimeID != stripped.RuntimeID {
			t.Errorf("stripped wood definition is inconsistent: %+v", stripped)
		}
		for _, definition := range []ExtendedBlockDefinition{wood, stripped} {
			if !IsPlaceableBlock(definition.RuntimeID) || definition.Hardness != 2 || definition.HarvestTool != ToolAxe {
				t.Errorf("wood block has incorrect placement/mining data: %+v", definition)
			}
			found := false
			for _, item := range SearchCreativeCatalog(CreativeAll, definition.Name) {
				if item.ID == definition.RuntimeID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("wood block %s is missing from the creative catalog", definition.Name)
			}
		}
		if got := CreativeItemDisplayName(wood.RuntimeID); got != tc.display {
			t.Errorf("%s display name = %q, want %q", wood.Name, got, tc.display)
		}
		if got := CreativeItemDisplayName(stripped.RuntimeID); got != "去皮"+tc.display {
			t.Errorf("%s display name = %q, want 去皮%s", stripped.Name, got, tc.display)
		}
		if got, ok := StrippedWoodBlockVariant(wood.RuntimeID); !ok || got != stripped.RuntimeID {
			t.Errorf("stripping %s returned (%d, %t), want %d", wood.Name, got, ok, stripped.RuntimeID)
		}
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
		{"minecraft:stripped_poplar_wood", 0},
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
	if got := SearchCreativeCatalog(CreativeAll, "minecraft:polished_deepslate"); len(got) < 1 || got[0].ID == 0 {
		t.Fatalf("registry-name search returned %+v, want polished deepslate matches", got)
	} else {
		foundBlock := false
		for _, item := range got {
			if creativeItemRegistryName(item.ID) == "minecraft:polished_deepslate" {
				foundBlock = true
				break
			}
		}
		if !foundBlock {
			t.Fatalf("registry-name search omitted the exact polished deepslate block: %+v", got)
		}
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
	for _, slab := range woodSlabDefinitions {
		planks, found := ExtendedBlockByName(slab.PlankBlock)
		if !found {
			t.Fatalf("planks %s missing for slab %s recipe", slab.PlankBlock, slab.Name)
		}
		crafted, ok := cm.MatchCrafting([]int32{planks.RuntimeID, planks.RuntimeID, planks.RuntimeID, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir}, 3)
		if !ok || crafted.ItemID != slab.ItemID || crafted.Count != 6 {
			t.Errorf("three %s planks craft %+v (ok=%t), want six %s", slab.PlankBlock, crafted, ok, slab.Name)
		}
	}
	for _, slab := range colorSlabDefinitions {
		var blockID int32
		for _, item := range CreativeCatalog(CreativeAll) {
			if creativeItemRegistryName(item.ID) == slab.BlockName {
				blockID = item.ID
				break
			}
		}
		if blockID == 0 {
			t.Fatalf("block %s missing for slab recipe", slab.BlockName)
		}
		crafted, ok := cm.MatchCrafting([]int32{blockID, blockID, blockID, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir}, 3)
		if !ok || crafted.ItemID != slab.ItemID || crafted.Count != 6 {
			t.Errorf("three %s blocks craft %+v (ok=%t), want six %s", slab.BlockName, crafted, ok, slab.Name)
		}
	}
	for _, slab := range masonrySlabDefinitions {
		var blockID int32
		for _, item := range CreativeCatalog(CreativeAll) {
			if creativeItemRegistryName(item.ID) == slab.BlockName {
				blockID = item.ID
				break
			}
		}
		if blockID == 0 {
			t.Fatalf("block %s missing for masonry slab recipe", slab.BlockName)
		}
		crafted, ok := cm.MatchCrafting([]int32{blockID, blockID, blockID, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir}, 3)
		if !ok || crafted.ItemID != slab.ItemID || crafted.Count != 6 {
			t.Errorf("three %s blocks craft %+v (ok=%t), want six %s", slab.BlockName, crafted, ok, slab.Name)
		}
	}
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

	for _, tc := range []struct {
		name   string
		grid   []int32
		itemID int32
	}{
		{"copper pickaxe", []int32{ItemCopperIngot, ItemCopperIngot, ItemCopperIngot, ItemAir, ItemStick, ItemAir, ItemAir, ItemStick, ItemAir}, ItemCopperPickaxe},
		{"copper sword", []int32{ItemAir, ItemCopperIngot, ItemAir, ItemAir, ItemCopperIngot, ItemAir, ItemAir, ItemStick, ItemAir}, ItemCopperSword},
		{"copper shovel", []int32{ItemAir, ItemCopperIngot, ItemAir, ItemAir, ItemStick, ItemAir, ItemAir, ItemStick, ItemAir}, ItemCopperShovel},
		{"copper axe", []int32{ItemCopperIngot, ItemCopperIngot, ItemAir, ItemCopperIngot, ItemStick, ItemAir, ItemAir, ItemStick, ItemAir}, ItemCopperAxe},
		{"copper hoe", []int32{ItemCopperIngot, ItemCopperIngot, ItemAir, ItemAir, ItemStick, ItemAir, ItemAir, ItemStick, ItemAir}, ItemCopperHoe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			crafted, found := cm.MatchCrafting(tc.grid, 3)
			if !found || crafted.ItemID != tc.itemID || crafted.Count != 1 {
				t.Fatalf("%s recipe = %+v, found=%t", tc.name, crafted, found)
			}
		})
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

func TestNetheriteArmorHasCreativeItemsStackLimitsAndVanillaProtection(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		slot int
		name string
	}{{ItemNetheriteBoots, 0, "netherite_boots"}, {ItemNetheriteLeggings, 1, "netherite_leggings"}, {ItemNetheriteChestplate, 2, "netherite_chestplate"}, {ItemNetheriteHelmet, 3, "netherite_helmet"}} {
		if slot, ok := ArmorSlotForItem(tc.id); !ok || slot != tc.slot {
			t.Errorf("netherite armor %d slot=(%d,%t), want %d", tc.id, slot, ok, tc.slot)
		}
		if GetMaxStackSize(tc.id) != 1 {
			t.Errorf("netherite armor %d max stack=%d, want 1", tc.id, GetMaxStackSize(tc.id))
		}
		if found := SearchCreativeCatalog(CreativeCombat, tc.name); len(found) != 1 || found[0].ID != tc.id {
			t.Errorf("creative search %q = %+v", tc.name, found)
		}
	}
	if found := SearchCreativeCatalog(CreativeMaterials, "minecraft:netherite_ingot"); len(found) != 1 || found[0].ID != ItemNetheriteIngot {
		t.Fatalf("netherite ingot creative search = %+v", found)
	}
	var armor [4]ItemStack
	armor[0] = ItemStack{ItemID: ItemNetheriteBoots, Count: 1}
	armor[1] = ItemStack{ItemID: ItemNetheriteLeggings, Count: 1}
	armor[2] = ItemStack{ItemID: ItemNetheriteChestplate, Count: 1}
	armor[3] = ItemStack{ItemID: ItemNetheriteHelmet, Count: 1}
	if got := CalculateArmorProtection(armor); got.Defense != 20 || got.Toughness != 12 {
		t.Fatalf("full netherite protection=%+v, want 20 defense and 12 toughness", got)
	}
}

func TestNetheriteSwordIsAFunctionalCreativeCombatItem(t *testing.T) {
	for _, tc := range []struct {
		id       int32
		name     string
		display  string
		category CreativeCategory
		typeID   ToolType
		speed    float64
		level    int
		damage   float32
	}{
		{ItemNetheriteSword, "netherite_sword", "下界合金剑", CreativeCombat, ToolSword, 1.5, 0, 8},
		{ItemNetheritePickaxe, "netherite_pickaxe", "下界合金镐", CreativeTools, ToolPickaxe, 9, 3, 6},
		{ItemNetheriteAxe, "netherite_axe", "下界合金斧", CreativeTools, ToolAxe, 9, 3, 10},
		{ItemNetheriteShovel, "netherite_shovel", "下界合金锹", CreativeTools, ToolShovel, 9, 3, 6.5},
		{ItemNetheriteHoe, "netherite_hoe", "下界合金锄", CreativeTools, ToolHoe, 9, 3, 1},
	} {
		items := SearchCreativeCatalog(tc.category, "minecraft:"+tc.name)
		if len(items) != 1 || items[0].ID != tc.id || items[0].Name != tc.display {
			t.Errorf("netherite item %q creative search = %+v", tc.name, items)
		}
		props, ok := GetToolProperties(tc.id)
		if !ok || props.Type != tc.typeID || props.Tier != TierNetherite || props.MaxDurability != 2031 || props.MiningSpeed != tc.speed || props.HarvestLevel != tc.level || props.AttackDamage != tc.damage {
			t.Errorf("netherite item %q properties = %+v, found=%t", tc.name, props, ok)
		}
		if got := GetMaxStackSize(tc.id); got != 1 {
			t.Errorf("netherite item %q max stack = %d, want 1", tc.name, got)
		}
	}
	netheriteBlock, ok := ExtendedBlockByName("minecraft:netherite_block")
	if !ok || !CanHarvestBlock(ItemNetheritePickaxe, netheriteBlock.RuntimeID) {
		t.Fatalf("netherite pickaxe cannot harvest the netherite block: block=%+v found=%t", netheriteBlock, ok)
	}
	if got := CalculateMiningSpeedMultiplier(ItemNetheritePickaxe, ItemStone); got != 9 {
		t.Fatalf("netherite pickaxe mining speed multiplier = %v, want 9", got)
	}
	stack := ItemStack{ItemID: ItemNetheritePickaxe, Count: 1}
	if ApplyToolWear(&stack) || stack.Damage != 1 {
		t.Fatalf("netherite pickaxe wear result = %+v, want one durability used", stack)
	}
}

func TestCopperToolsMatchJava263CreativeItemsAndProperties(t *testing.T) {
	for _, tc := range []struct {
		id       int32
		name     string
		display  string
		category CreativeCategory
		typeID   ToolType
		speed    float64
		level    int
		damage   float32
	}{
		{ItemCopperSword, "copper_sword", "铜剑", CreativeCombat, ToolSword, 1.5, 0, 4.0},
		{ItemCopperShovel, "copper_shovel", "铜锹", CreativeTools, ToolShovel, 5.0, 1, 2.5},
		{ItemCopperPickaxe, "copper_pickaxe", "铜镐", CreativeTools, ToolPickaxe, 5.0, 1, 2.0},
		{ItemCopperAxe, "copper_axe", "铜斧", CreativeTools, ToolAxe, 5.0, 1, 8.0},
		{ItemCopperHoe, "copper_hoe", "铜锄", CreativeTools, ToolHoe, 5.0, 0, 0.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := SearchCreativeCatalog(tc.category, "minecraft:"+tc.name)
			if len(items) != 1 || items[0].ID != tc.id || items[0].Name != tc.display {
				t.Fatalf("creative search for %q = %+v", tc.name, items)
			}
			props, ok := GetToolProperties(tc.id)
			if !ok || props.Type != tc.typeID || props.Tier != TierCopper || props.MaxDurability != 190 || props.MiningSpeed != tc.speed || props.HarvestLevel != tc.level || props.AttackDamage != tc.damage {
				t.Fatalf("copper item %q properties = %+v, found=%t", tc.name, props, ok)
			}
			if got := GetMaxStackSize(tc.id); got != 1 {
				t.Fatalf("copper item %q max stack = %d, want 1", tc.name, got)
			}
		})
	}
	if !CanHarvestBlock(ItemCopperPickaxe, 15) {
		t.Fatal("copper pickaxe should harvest iron ore in the Java 26.3 tier")
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

func TestAddedVanillaFullCubesHaveCreativeItemsAndCorrectHarvestRules(t *testing.T) {
	want := map[string]struct {
		display  string
		hardness float64
		level    int
		requires bool
	}{
		"minecraft:resin_bricks":          {"树脂砖", 1.5, 0, false},
		"minecraft:chiseled_resin_bricks": {"錾制树脂砖", 1.5, 0, false},
		"minecraft:lapis_block":           {"青金石块", 3, 1, true},
		"minecraft:cut_sandstone":         {"切制砂岩", 0.8, 0, true},
		"minecraft:cut_red_sandstone":     {"切制红砂岩", 0.8, 0, true},
	}
	for name, expected := range want {
		t.Run(name, func(t *testing.T) {
			definition, ok := ExtendedBlockByName(name)
			if !ok {
				t.Fatalf("missing block definition for %s", name)
			}
			if definition.Hardness != expected.hardness || definition.HarvestTool != ToolPickaxe || definition.RequiresHarvestTool != expected.requires || definition.RequiredHarvestLevel != expected.level || definition.DropRuntimeID != definition.RuntimeID {
				t.Errorf("block behavior = %+v", definition)
			}
			items := SearchCreativeCatalog(CreativeBlocks, name)
			var exact []CreativeItem
			for _, item := range items {
				if creativeItemRegistryName(item.ID) == name {
					exact = append(exact, item)
				}
			}
			if len(exact) != 1 || exact[0].ID != definition.RuntimeID || exact[0].Name != expected.display {
				t.Errorf("creative item = %+v", items)
			}
			if CanHarvestBlock(ItemAir, definition.RuntimeID) == expected.requires {
				t.Errorf("hand harvest result is wrong for %s", name)
			}
			if name == "minecraft:lapis_block" && CanHarvestBlock(ItemWoodenPickaxe, definition.RuntimeID) {
				t.Error("wooden pickaxe should not harvest lapis block drops")
			}
			if !CanHarvestBlock(ItemStonePickaxe, definition.RuntimeID) {
				t.Errorf("stone pickaxe should harvest %s", name)
			}
		})
	}
}

func TestModernCreativeMaterialsHaveRegistryNamesAndUsefulRecipes(t *testing.T) {
	for _, definition := range ModernCreativeItemDefinitions() {
		if definition.Category != CreativeMaterials || definition.MaxStack < 1 || definition.MaxStack > 64 {
			t.Errorf("item %s has category/stack limit %d/%d", definition.Name, definition.Category, definition.MaxStack)
		}
		if got := GetMaxStackSize(definition.RuntimeID); got != definition.MaxStack {
			t.Errorf("item %s stack limit = %d, want %d", definition.Name, got, definition.MaxStack)
		}
		matches := SearchCreativeCatalog(CreativeMaterials, definition.Name)
		var exact []CreativeItem
		for _, item := range matches {
			if creativeItemRegistryName(item.ID) == definition.Name {
				exact = append(exact, item)
			}
		}
		if len(exact) != 1 || exact[0].ID != definition.RuntimeID || exact[0].Name != definition.Display {
			t.Errorf("creative catalog lookup for %s = %+v", definition.Name, exact)
		}
	}
	var materialOrder []string
	for _, item := range CreativeCatalog(CreativeMaterials) {
		materialOrder = append(materialOrder, creativeItemRegistryName(item.ID))
	}
	for i, want := range []string{"minecraft:iron_ingot", "minecraft:coal", "minecraft:charcoal", "minecraft:emerald", "minecraft:lapis_lazuli", "minecraft:amethyst_shard", "minecraft:quartz"} {
		if i >= len(materialOrder) || materialOrder[i] != want {
			t.Fatalf("Java 26.3 material tab order prefix = %v, want item %d %s", materialOrder[:min(len(materialOrder), 8)], i, want)
		}
	}

	crafting := NewCraftingManager()
	for _, tc := range []struct {
		name   string
		grid   []int32
		size   int
		output ItemStack
	}{
		{"blaze powder", []int32{ItemBlazeRod, 0, 0, 0}, 2, ItemStack{ItemID: ItemBlazePowder, Count: 2}},
		{"torch from charcoal", []int32{ItemCharcoal, 0, ItemStick, 0}, 2, ItemStack{ItemID: ItemTorch, Count: 4}},
		{"copper nuggets", []int32{ItemCopperIngot, 0, 0, 0}, 2, ItemStack{ItemID: ItemCopperNugget, Count: 9}},
		{"copper ingot", []int32{ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget}, 3, ItemStack{ItemID: ItemCopperIngot, Count: 1}},
		{"iron nuggets", []int32{ItemIronIngot, 0, 0, 0}, 2, ItemStack{ItemID: ItemIronNugget, Count: 9}},
		{"iron ingot", []int32{ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget}, 3, ItemStack{ItemID: ItemIronIngot, Count: 1}},
		{"sugar", []int32{ItemSugarCane, 0, 0, 0}, 2, ItemStack{ItemID: ItemSugar, Count: 1}},
		{"paper", []int32{ItemSugarCane, ItemSugarCane, ItemSugarCane, 0, 0, 0, 0, 0, 0}, 3, ItemStack{ItemID: ItemPaper, Count: 3}},
		{"glowstone block", []int32{ItemGlowstoneDust, ItemGlowstoneDust, ItemGlowstoneDust, ItemGlowstoneDust}, 2, ItemStack{ItemID: ItemGlowstoneBlock, Count: 1}},
		{"slime block", []int32{ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall}, 3, ItemStack{ItemID: ItemSlimeBlock, Count: 1}},
		{"amethyst block", []int32{ItemAmethystShard, ItemAmethystShard, ItemAmethystShard, ItemAmethystShard}, 2, ItemStack{ItemID: 14000, Count: 1}},
		{"honeycomb block", []int32{ItemHoneycomb, ItemHoneycomb, ItemHoneycomb, ItemHoneycomb}, 2, ItemStack{ItemID: 14106, Count: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := crafting.MatchCrafting(tc.grid, tc.size)
			if !ok || got != tc.output {
				t.Fatalf("craft result = %+v, matched=%t; want %+v", got, ok, tc.output)
			}
		})
	}

	if smelt, ok := MatchSmelting(ItemOakLog); !ok || smelt.OutputID != ItemCharcoal {
		t.Errorf("oak log smelting = %+v, matched=%t; want distinct charcoal", smelt, ok)
	}
	for _, name := range []string{"minecraft:birch_log", "minecraft:stripped_oak_log", "minecraft:oak_wood"} {
		block, ok := ExtendedBlockByName(name)
		if !ok {
			t.Fatalf("missing log block %s", name)
		}
		if smelt, ok := MatchSmelting(block.RuntimeID); !ok || smelt.OutputID != ItemCharcoal {
			t.Errorf("%s smelting = %+v, matched=%t; want charcoal", name, smelt, ok)
		}
	}
	if got := GetFuelBurnTime(ItemCharcoal); got != 1600 {
		t.Errorf("charcoal burn time = %d, want 1600", got)
	}
	if got := GetFuelBurnTime(ItemBlazeRod); got != 2400 {
		t.Errorf("blaze rod burn time = %d, want 2400", got)
	}
	if smelt, ok := MatchSmelting(ItemResinClump); !ok || smelt.OutputID != ItemResinBrick {
		t.Errorf("resin clump smelting = %+v, matched=%t", smelt, ok)
	}
	if smelt, ok := MatchSmelting(87); !ok || smelt.OutputID != ItemNetherBrick {
		t.Errorf("netherrack smelting = %+v, matched=%t", smelt, ok)
	}
	if slot, ok := ArmorSlotForItem(ItemTurtleHelmet); !ok || slot != 3 || !CanEquipArmor(3, ItemTurtleHelmet) {
		t.Errorf("turtle helmet equipment slot = %d, valid=%t", slot, ok)
	}
}

func TestCreativeMaterialsIncludeJava263DyesInVanillaTagOrder(t *testing.T) {
	want := []string{
		"minecraft:white_dye", "minecraft:orange_dye", "minecraft:magenta_dye", "minecraft:light_blue_dye",
		"minecraft:yellow_dye", "minecraft:lime_dye", "minecraft:pink_dye", "minecraft:gray_dye",
		"minecraft:light_gray_dye", "minecraft:cyan_dye", "minecraft:purple_dye", "minecraft:blue_dye",
		"minecraft:brown_dye", "minecraft:green_dye", "minecraft:red_dye", "minecraft:black_dye",
	}
	var got []string
	for _, item := range CreativeCatalog(CreativeMaterials) {
		name := creativeItemRegistryName(item.ID)
		if strings.HasSuffix(name, "_dye") {
			got = append(got, name)
			if GetMaxStackSize(item.ID) != 64 {
				t.Errorf("%s max stack = %d, want 64", name, GetMaxStackSize(item.ID))
			}
		}
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Java 26.3 dye order = %v, want %v", got, want)
	}
}

func TestDyesCraftColoredWoolShapelesslyAndRejectMalformedGrids(t *testing.T) {
	crafting := NewCraftingManager()
	colors := []struct {
		name  string
		dyeID int32
	}{
		{"orange", ItemOrangeDye}, {"magenta", ItemMagentaDye}, {"light_blue", ItemLightBlueDye},
		{"yellow", ItemYellowDye}, {"lime", ItemLimeDye}, {"pink", ItemPinkDye}, {"gray", ItemGrayDye},
		{"light_gray", ItemLightGrayDye}, {"cyan", ItemCyanDye}, {"purple", ItemPurpleDye}, {"blue", ItemBlueDye},
		{"brown", ItemBrownDye}, {"green", ItemGreenDye}, {"red", ItemRedDye}, {"black", ItemBlackDye},
	}
	for _, color := range colors {
		wool, ok := ExtendedBlockByName("minecraft:" + color.name + "_wool")
		if !ok {
			t.Fatalf("missing %s wool definition", color.name)
		}
		for _, tc := range []struct {
			name string
			grid []int32
			size int
		}{
			{"2x2 dye first", []int32{color.dyeID, ItemAir, 35, ItemAir}, 2},
			{"2x2 wool first", []int32{35, ItemAir, ItemAir, color.dyeID}, 2},
			{"3x3 separated", []int32{ItemAir, 35, ItemAir, ItemAir, ItemAir, ItemAir, color.dyeID, ItemAir, ItemAir}, 3},
		} {
			got, matched := crafting.MatchCrafting(tc.grid, tc.size)
			if !matched || got != (ItemStack{ItemID: wool.RuntimeID, Count: 1}) {
				t.Errorf("%s dye recipe %q = %+v, matched=%t", color.name, tc.name, got, matched)
			}
		}
	}
	if _, matched := crafting.MatchCrafting([]int32{35, ItemRedDye, ItemStone, ItemAir}, 2); matched {
		t.Error("dye recipe accepted an unrelated extra item")
	}
	for _, malformed := range []struct {
		grid []int32
		size int
	}{{[]int32{35}, 2}, {make([]int32, 16), 4}, {nil, 3}} {
		if _, matched := crafting.MatchCrafting(malformed.grid, malformed.size); matched {
			t.Errorf("malformed grid size %d unexpectedly matched", malformed.size)
		}
	}
}

func TestCreativeCategoriesFollowJava263DirectItemOrder(t *testing.T) {
	for _, category := range []CreativeCategory{
		CreativeBlocks, CreativeDecoration, CreativeNatural, CreativeFunctional, CreativeRedstone,
		CreativeTools, CreativeCombat, CreativeFood, CreativeMaterials, CreativeSpawnEggs,
	} {
		catalog := CreativeCatalog(category)
		if len(catalog) == 0 {
			continue
		}
		var expected string
		for _, name := range vanillaCreativeTabOrder[category] {
			for _, item := range catalog {
				if creativeItemRegistryName(item.ID) == name {
					expected = name
					break
				}
			}
			if expected != "" {
				break
			}
		}
		if expected == "" {
			continue
		}
		if got := creativeItemRegistryName(catalog[0].ID); got != expected {
			t.Errorf("creative category %d begins with %s; Java 26.3's first implemented entry is %s", category, got, expected)
		}
	}
}

func TestCreativeTabsIncludeVanillaCrossCategoryItems(t *testing.T) {
	// Java 26.3 intentionally shows these items in Ingredients as well as in
	// their primary Food, Tools, or Functional tab.
	wantIngredients := []string{
		"minecraft:ancient_debris", "minecraft:snowball", "minecraft:ender_pearl",
		"minecraft:ender_eye", "minecraft:book", "minecraft:golden_carrot", "minecraft:pufferfish",
	}
	for _, name := range wantIngredients {
		matches := SearchCreativeCatalog(CreativeMaterials, name)
		found := false
		for _, item := range matches {
			if creativeItemRegistryName(item.ID) == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Java 26.3 ingredients tab is missing %s: %+v", name, matches)
		}
	}

	seen := make(map[int32]struct{})
	for _, item := range CreativeCatalog(CreativeMaterials) {
		if _, duplicate := seen[item.ID]; duplicate {
			t.Errorf("ingredients tab contains duplicate item id %d", item.ID)
		}
		seen[item.ID] = struct{}{}
	}
	if all := CreativeCatalog(CreativeAll); len(all) != 900 {
		t.Fatalf("canonical Search tab contains %d items after adding the current supported block families, want 900", len(all))
	}
}

func TestWoodStairsCatalogStatesAndRecipes(t *testing.T) {
	if len(woodStairDefinitions) != 14 {
		t.Fatalf("wood stair family has %d definitions, want 14", len(woodStairDefinitions))
	}
	for _, stair := range woodStairDefinitions {
		t.Run(stair.Name, func(t *testing.T) {
			block, ok := ExtendedBlockByID(stair.ItemID)
			if !ok || block.Name != stair.Name || block.TextureTop != stair.Texture || block.Category != CreativeBlocks || block.DropRuntimeID != stair.ItemID {
				t.Fatalf("stair block definition = %+v, %t", block, ok)
			}
			matches := SearchCreativeCatalog(CreativeBlocks, stair.Name)
			if len(matches) != 1 || matches[0].ID != stair.ItemID {
				t.Fatalf("creative search %q = %+v", stair.Name, matches)
			}
			for facing := StairFacing(0); facing < 4; facing++ {
				for _, upsideDown := range []bool{false, true} {
					state, ok := StairBlockState(stair.ItemID, facing, upsideDown)
					if !ok {
						t.Fatalf("failed to create stair state for facing=%d upsideDown=%t", facing, upsideDown)
					}
					item, gotFacing, gotUpsideDown, ok := StairStateProperties(state)
					if !ok || item != stair.ItemID || gotFacing != facing || gotUpsideDown != upsideDown {
						t.Errorf("state %d decoded as (%d,%d,%t,%t)", state, item, gotFacing, gotUpsideDown, ok)
					}
					if boxes, ok := BlockCollisionBoxes(state); !ok || len(boxes) != 2 {
						t.Errorf("stair state %d collision boxes = %+v, %t", state, boxes, ok)
					}
				}
			}
			var sourceID int32
			for _, item := range CreativeCatalog(CreativeAll) {
				if creativeItemRegistryName(item.ID) == stair.SourceBlock {
					sourceID = item.ID
					break
				}
			}
			if sourceID == 0 {
				t.Fatalf("source block %s is missing", stair.SourceBlock)
			}
			crafted, ok := NewCraftingManager().MatchCrafting([]int32{sourceID, 0, 0, sourceID, sourceID, 0, sourceID, sourceID, sourceID}, 3)
			if !ok || crafted.ItemID != stair.ItemID || crafted.Count != 4 {
				t.Errorf("recipe output = %+v, %t; want 4 %s", crafted, ok, stair.Name)
			}
		})
	}
}

func TestJava263ColoredCushionsHaveCatalogEntriesAndSprites(t *testing.T) {
	colors := []string{
		"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
	}
	for i, color := range colors {
		id := FirstCushionBlock + int32(i)
		name := "minecraft:" + color + "_cushion"
		item, ok := ExtendedBlockByID(id)
		if !ok || item.Name != name || item.Category != CreativeDecoration || item.Hardness <= 0 || item.DropRuntimeID != id {
			t.Fatalf("cushion %s definition = %+v, %t", name, item, ok)
		}
		if !IsPlaceableBlock(id) || GetMaxStackSize(id) != 16 {
			t.Fatalf("cushion %s is not placeable or has stack size %d", name, GetMaxStackSize(id))
		}
		matches := SearchCreativeCatalog(CreativeDecoration, name)
		if len(matches) != 1 || matches[0].ID != id {
			t.Fatalf("search %q = %+v, want one item with id %d", name, matches, id)
		}
		if got := ExtendedBlockItemTextureName(id); got != color+"_cushion" {
			t.Errorf("item texture for %s = %q", name, got)
		}
		var slabID int32
		for _, slab := range colorSlabDefinitions {
			if slab.Name == "minecraft:"+color+"_wool_slab" {
				slabID = slab.ItemID
				break
			}
		}
		if slabID == 0 {
			t.Fatalf("matching wool slab for %s is missing", name)
		}
		crafted, ok := NewCraftingManager().MatchCrafting([]int32{slabID, slabID, slabID, 0, 0, 0, 0, 0, 0}, 3)
		if !ok || crafted.ItemID != id || crafted.Count != 1 {
			t.Errorf("recipe for %s = %+v, %t; want one cushion from three matching wool slabs", name, crafted, ok)
		}
	}
}

func TestJava263StainedGlassCatalogRecipesAndDrops(t *testing.T) {
	colors := []string{
		"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
	}
	catalog := CreativeCatalog(CreativeAll)
	crafting := NewCraftingManager()
	for index, color := range colors {
		id := int32(17100 + index)
		name := "minecraft:" + color + "_stained_glass"
		block, ok := ExtendedBlockByID(id)
		if !ok || block.Name != name || block.Category != CreativeDecoration || block.TextureTop != color+"_stained_glass" || !block.DropsNothing {
			t.Fatalf("stained glass definition %s = %+v, %t", name, block, ok)
		}
		matches := SearchCreativeCatalog(CreativeDecoration, name)
		if len(matches) != 1 || matches[0].ID != id {
			t.Fatalf("search %q = %+v, want one item with id %d", name, matches, id)
		}
		found := false
		for _, item := range catalog {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s missing from all-items catalog", name)
		}
		dyeID := []int32{ItemWhiteDye, ItemOrangeDye, ItemMagentaDye, ItemLightBlueDye, ItemYellowDye, ItemLimeDye, ItemPinkDye, ItemGrayDye,
			ItemLightGrayDye, ItemCyanDye, ItemPurpleDye, ItemBlueDye, ItemBrownDye, ItemGreenDye, ItemRedDye, ItemBlackDye}[index]
		grid := []int32{20, 20, 20, 20, dyeID, 20, 20, 20, 20}
		if got, matched := crafting.MatchCrafting(grid, 3); !matched || got != (ItemStack{ItemID: id, Count: 8}) {
			t.Errorf("recipe for %s = %+v, matched=%t; want 8 blocks", name, got, matched)
		}
	}
	if got := len(SearchCreativeCatalog(CreativeDecoration, "stained_glass")); got != 2*len(colors) {
		t.Fatalf("stained glass and pane search returned %d items, want %d", got, 2*len(colors))
	}
}

func TestJava263GlassPanesHaveCatalogShapesAndRecipes(t *testing.T) {
	colors := []string{
		"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
	}
	catalog := CreativeCatalog(CreativeAll)
	if len(catalog) != 900 {
		t.Fatalf("all-items catalog contains %d entries after adding panes and candles, want 900", len(catalog))
	}
	crafting := NewCraftingManager()
	checkPane := func(id int32, name, texture string, glassID int32) {
		t.Helper()
		definition, ok := ExtendedBlockByID(id)
		if !ok || definition.Name != name || definition.TextureTop != texture || !definition.DropsNothing {
			t.Fatalf("pane definition %s = %+v, found=%t", name, definition, ok)
		}
		if !IsPlaceableBlock(id) || !IsGlassPaneBlock(id) || GetMaxStackSize(id) != 64 {
			t.Errorf("pane %s lacks placeable/catalog behavior or has stack size %d", name, GetMaxStackSize(id))
		}
		matches := SearchCreativeCatalog(CreativeDecoration, name)
		if len(matches) != 1 || matches[0].ID != id {
			t.Errorf("creative search %q = %+v", name, matches)
		}
		grid := []int32{glassID, glassID, glassID, glassID, glassID, glassID, 0, 0, 0}
		if got, matched := crafting.MatchCrafting(grid, 3); !matched || got != (ItemStack{ItemID: id, Count: 16}) {
			t.Errorf("pane recipe %s = %+v, matched=%t; want 16 panes", name, got, matched)
		}
	}
	checkPane(17116, "minecraft:glass_pane", "glass_pane_top", 20)
	for index, color := range colors {
		glass, ok := ExtendedBlockByName("minecraft:" + color + "_stained_glass")
		if !ok {
			t.Fatalf("missing %s stained glass for pane recipe", color)
		}
		checkPane(17117+int32(index), "minecraft:"+color+"_stained_glass_pane", color+"_stained_glass_pane_top", glass.RuntimeID)
	}
	isolated := GlassPaneBoxesAt(0, 0, 0, func(int32, int32, int32) int32 { return 0 })
	if len(isolated) != 1 || isolated[0] != (BlockBox{MinX: 7.0 / 16.0, MinY: 0, MinZ: 7.0 / 16.0, MaxX: 9.0 / 16.0, MaxY: 1, MaxZ: 9.0 / 16.0}) {
		t.Errorf("isolated pane shape = %+v", isolated)
	}
	connected := GlassPaneBoxesAt(0, 0, 0, func(x, y, z int32) int32 {
		if x == -1 || x == 1 || z == -1 || z == 1 {
			return 1
		}
		return 0
	})
	if len(connected) != 5 {
		t.Errorf("pane connected on four sides has %d collision pieces, want 5", len(connected))
	}
}

func TestJava263CandleCatalogCraftingAndShape(t *testing.T) {
	colors := []struct {
		name  string
		dyeID int32
	}{
		{"white", ItemWhiteDye}, {"orange", ItemOrangeDye}, {"magenta", ItemMagentaDye}, {"light_blue", ItemLightBlueDye},
		{"yellow", ItemYellowDye}, {"lime", ItemLimeDye}, {"pink", ItemPinkDye}, {"gray", ItemGrayDye},
		{"light_gray", ItemLightGrayDye}, {"cyan", ItemCyanDye}, {"purple", ItemPurpleDye}, {"blue", ItemBlueDye},
		{"brown", ItemBrownDye}, {"green", ItemGreenDye}, {"red", ItemRedDye}, {"black", ItemBlackDye},
	}
	catalog := CreativeCatalog(CreativeAll)
	if len(catalog) != 900 {
		t.Fatalf("all-items catalog contains %d entries after adding candle variants, want 900", len(catalog))
	}
	crafting := NewCraftingManager()
	plain, ok := ExtendedBlockByID(17133)
	if !ok || plain.Name != "minecraft:candle" || plain.TextureTop != "candle" || !IsCandleBlock(plain.RuntimeID) {
		t.Fatalf("plain candle registration = %+v, found=%t", plain, ok)
	}
	if out, matched := crafting.MatchCrafting([]int32{ItemString, 0, ItemHoneycomb, 0}, 2); !matched || out != (ItemStack{ItemID: plain.RuntimeID, Count: 1}) {
		t.Errorf("string and honeycomb recipe = %+v, matched=%t", out, matched)
	}
	shape, ok := BlockCollisionBoxes(plain.RuntimeID)
	if !ok || len(shape) != 1 || shape[0] != candleCollisionBox[0] {
		t.Errorf("single candle collision shape = %+v, found=%t", shape, ok)
	}
	for index, color := range colors {
		id := int32(17134 + index)
		name := "minecraft:" + color.name + "_candle"
		candle, found := ExtendedBlockByID(id)
		if !found || candle.Name != name || candle.TextureTop != color.name+"_candle" || !IsCandleBlock(id) {
			t.Fatalf("%s definition = %+v, found=%t", name, candle, found)
		}
		if got := SearchCreativeCatalog(CreativeDecoration, name); len(got) != 1 || got[0].ID != id {
			t.Errorf("creative lookup %s = %+v", name, got)
		}
		if out, matched := crafting.MatchCrafting([]int32{plain.RuntimeID, color.dyeID, 0, 0}, 2); !matched || out != (ItemStack{ItemID: id, Count: 1}) {
			t.Errorf("dye recipe for %s = %+v, matched=%t", name, out, matched)
		}
	}
}

func TestJava263WoolAndConcreteStairItemsAndStates(t *testing.T) {
	if len(colorStairDefinitions) != 32 {
		t.Fatalf("colored stair family has %d definitions, want 32", len(colorStairDefinitions))
	}
	catalog := CreativeCatalog(CreativeAll)
	for _, stair := range colorStairDefinitions {
		definition, ok := ExtendedBlockByID(stair.ItemID)
		if !ok || definition.Name != stair.Name || definition.TextureTop != stair.Texture {
			t.Fatalf("stair %s registration = %+v, found=%t", stair.Name, definition, ok)
		}
		if got := SearchCreativeCatalog(CreativeAll, stair.Name); len(got) != 1 || got[0].ID != stair.ItemID {
			t.Errorf("registry search for %s = %+v", stair.Name, got)
		}
		foundDecoration := false
		for _, item := range CreativeCatalog(CreativeDecoration) {
			if item.ID == stair.ItemID {
				foundDecoration = true
				break
			}
		}
		if !foundDecoration {
			t.Errorf("stair %s missing from decoration tab", stair.Name)
		}
		if GetMaxStackSize(stair.ItemID) != 64 {
			t.Errorf("stair %s max stack = %d, want 64", stair.Name, GetMaxStackSize(stair.ItemID))
		}
		if stair.RequiresTool {
			if CanHarvestBlock(ItemAir, stair.ItemID) || !CanHarvestBlock(ItemWoodenPickaxe, stair.ItemID) {
				t.Errorf("concrete stair %s should require a pickaxe", stair.Name)
			}
		} else if !CanHarvestBlock(ItemAir, stair.ItemID) {
			t.Errorf("wool stair %s should not require a tool", stair.Name)
		}
		for facing := StairFacing(0); facing < 4; facing++ {
			for _, upsideDown := range []bool{false, true} {
				state, ok := StairBlockState(stair.ItemID, facing, upsideDown)
				if !ok {
					t.Fatalf("failed to create state for %s", stair.Name)
				}
				item, gotFacing, gotUpsideDown, ok := StairStateProperties(state)
				if !ok || item != stair.ItemID || gotFacing != facing || gotUpsideDown != upsideDown || BlockItemForState(state) != stair.ItemID {
					t.Errorf("stair %s state %d decoded as (%d,%d,%t,%t)", stair.Name, state, item, gotFacing, gotUpsideDown, ok)
				}
				boxes, found := BlockCollisionBoxes(state)
				if !found || len(boxes) != 2 {
					t.Errorf("stair %s state %d collision boxes = %+v, found=%t", stair.Name, state, boxes, found)
				}
				occupied := 0
				for y := 0; y < 2; y++ {
					for z := 0; z < 2; z++ {
						for x := 0; x < 2; x++ {
							if StairVoxelOccupied(state, x, y, z) {
								occupied++
							}
						}
					}
				}
				if occupied != 6 {
					t.Errorf("stair %s state %d occupies %d half-block cells, want 6", stair.Name, state, occupied)
				}
			}
		}
	}
	if len(catalog) != 900 {
		t.Fatalf("all-items search catalog length = %d, want 900", len(catalog))
	}
	whiteWoolCount := 0
	for _, item := range catalog {
		if creativeItemRegistryName(item.ID) == "minecraft:white_wool" {
			whiteWoolCount++
		}
	}
	if whiteWoolCount != 1 {
		t.Errorf("white wool appears %d times in the canonical catalog, want once", whiteWoolCount)
	}
	for _, tc := range []struct {
		yaw    float64
		facing StairFacing
	}{{0, 2}, {90, 1}, {180, 0}, {270, 3}} {
		state, ok := StairPlacementState(FirstColorStairItem, tc.yaw, 0, .25)
		_, facing, upsideDown, decoded := StairStateProperties(state)
		if !ok || !decoded || facing != tc.facing || upsideDown {
			t.Errorf("yaw %.0f stair placement = (%d,%t,%t), want facing %d lower", tc.yaw, facing, upsideDown, decoded, tc.facing)
		}
	}
	stairRecipes := 0
	for _, recipe := range NewCraftingManager().Recipes {
		if recipe.Output.ItemID < FirstColorStairItem || recipe.Output.ItemID > LastColorStairItem {
			continue
		}
		stairRecipes++
		if recipe.Width != 3 || recipe.Height != 3 || len(recipe.Pattern) != 9 || recipe.Output.Count != 4 {
			t.Errorf("stair recipe for item %d = %+v, want 3x3 yielding 4", recipe.Output.ItemID, recipe)
		}
		var sourceID int32
		for _, item := range catalog {
			if creativeItemRegistryName(item.ID) == colorStairDefinitions[recipe.Output.ItemID-FirstColorStairItem].SourceBlock {
				sourceID = item.ID
				break
			}
		}
		wantPattern := []int32{sourceID, ItemAir, ItemAir, sourceID, sourceID, ItemAir, sourceID, sourceID, sourceID}
		for i := range wantPattern {
			if i >= len(recipe.Pattern) || recipe.Pattern[i] != wantPattern[i] {
				t.Errorf("stair recipe for item %d pattern = %v, want %v", recipe.Output.ItemID, recipe.Pattern, wantPattern)
				break
			}
		}
	}
	if stairRecipes != 32 {
		t.Errorf("registered %d colored stair recipes, want 32", stairRecipes)
	}
	var whiteWool int32
	for _, item := range catalog {
		if creativeItemRegistryName(item.ID) == "minecraft:white_wool" {
			whiteWool = item.ID
			break
		}
	}
	mirrored := []int32{
		ItemAir, ItemAir, whiteWool,
		ItemAir, whiteWool, whiteWool,
		whiteWool, whiteWool, whiteWool,
	}
	if output, ok := NewCraftingManager().MatchCrafting(mirrored, 3); !ok || output.ItemID != FirstColorStairItem || output.Count != 4 {
		t.Errorf("mirrored wool stair recipe output = %+v, matched=%t", output, ok)
	}
}

func TestCreativeSearchTabUsesVanillaCategoryAndItemOrder(t *testing.T) {
	items := CreativeCatalog(CreativeAll)
	if len(items) == 0 || creativeItemRegistryName(items[0].ID) != "minecraft:bricks" {
		var first string
		if len(items) > 0 {
			first = creativeItemRegistryName(items[0].ID)
		}
		t.Fatalf("search tab first item = %q, want minecraft:bricks", first)
	}
	for i := 1; i < len(items); i++ {
		if items[i-1].Category > items[i].Category {
			t.Fatalf("search tab category order regressed at %d: %d after %d", i, items[i].Category, items[i-1].Category)
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

func TestCreativeCatalogDoesNotListNonItemBlockStates(t *testing.T) {
	// These legacy runtime IDs represent fluid/portal/lit-state blocks or the
	// double-slab block state. Java's creative item catalog offers the bucket,
	// portal frame, single slab, or unlit lamp item instead.
	nonItemStates := map[int32]bool{8: true, 9: true, 10: true, 11: true, 43: true, 90: true, 119: true, 123: true}
	for _, item := range CreativeCatalog(CreativeAll) {
		if nonItemStates[item.ID] {
			t.Errorf("creative catalog exposes non-item block state %q (id %d)", item.Name, item.ID)
		}
	}
}

func TestAdditional26FullCubesAreRegisteredAndSearchable(t *testing.T) {
	want := []string{
		"amethyst_block", "bamboo_mosaic", "chiseled_nether_bricks", "chiseled_polished_blackstone",
		"chiseled_stone_bricks", "cracked_nether_bricks", "cracked_stone_bricks", "dark_prismarine",
		"mossy_stone_bricks", "nether_wart_block", "netherite_block", "pale_moss_block", "prismarine",
		"prismarine_bricks", "purpur_block", "quartz_bricks", "red_nether_bricks", "resin_block",
	}
	seenIDs := make(map[int32]string)
	for _, name := range want {
		definition, ok := ExtendedBlockByName("minecraft:" + name)
		if !ok {
			t.Fatalf("26.3 block %q is missing its extended definition", name)
		}
		if previous, duplicate := seenIDs[definition.RuntimeID]; duplicate {
			t.Fatalf("runtime ID %d is shared by %q and %q", definition.RuntimeID, previous, name)
		}
		seenIDs[definition.RuntimeID] = name
		if !IsPlaceableBlock(definition.RuntimeID) {
			t.Errorf("%q is not placeable", name)
		}
		if definition.DropRuntimeID != definition.RuntimeID {
			t.Errorf("%q drop = %d, want itself (%d)", name, definition.DropRuntimeID, definition.RuntimeID)
		}
		if name == "netherite_block" {
			diamondTool, _ := GetToolProperties(ItemDiamondPickaxe)
			if !definition.RequiresHarvestTool || definition.RequiredHarvestLevel != diamondTool.HarvestLevel {
				t.Errorf("netherite block harvest requirement = (%t, %d), want diamond-tier tool", definition.RequiresHarvestTool, definition.RequiredHarvestLevel)
			}
		} else if definition.RequiresHarvestTool {
			t.Errorf("%q requires a tool although its 26.3 loot table has no tool condition", name)
		}
		matches := SearchCreativeCatalog(CreativeAll, "minecraft:"+name)
		found := false
		for _, match := range matches {
			if match.ID == definition.RuntimeID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("creative search for %q = %+v, want runtime ID %d", name, matches, definition.RuntimeID)
		}
	}
}

func TestAdditionalCommonVanillaCubesAreRegisteredAndSearchable(t *testing.T) {
	want := []string{
		"bone_block", "chiseled_quartz_block", "chiseled_red_sandstone", "chiseled_sandstone", "coarse_dirt",
		"end_stone_bricks", "honeycomb_block", "lodestone", "muddy_mangrove_roots", "purpur_pillar",
		"quartz_pillar", "rooted_dirt", "shroomlight", "smooth_quartz", "smooth_red_sandstone",
		"smooth_sandstone", "smooth_stone", "soul_soil", "wet_sponge", "red_sand",
	}
	for _, name := range want {
		definition, ok := ExtendedBlockByName("minecraft:" + name)
		if !ok {
			t.Fatalf("26.3 block %q is missing its extended definition", name)
		}
		if !IsPlaceableBlock(definition.RuntimeID) {
			t.Errorf("%q is not placeable", name)
		}
		if definition.DropRuntimeID != definition.RuntimeID {
			t.Errorf("%q drop = %d, want itself (%d)", name, definition.DropRuntimeID, definition.RuntimeID)
		}
		if definition.RequiresHarvestTool {
			t.Errorf("%q requires a harvest tool although its 26.3 loot table has no tool condition", name)
		}
		found := false
		for _, item := range SearchCreativeCatalog(CreativeAll, "minecraft:"+name) {
			found = found || item.ID == definition.RuntimeID
		}
		if !found {
			t.Errorf("%q is absent from exact-name creative search", name)
		}
	}
}

func TestVanillaSmoothBlockSmeltingRecipes(t *testing.T) {
	for _, tc := range []struct {
		input int32
		name  string
	}{
		{ItemStone, "smooth_stone"},
		{155, "smooth_quartz"},
		{24, "smooth_sandstone"},
		{179, "smooth_red_sandstone"},
	} {
		output, ok := ExtendedBlockByName("minecraft:" + tc.name)
		if !ok {
			t.Fatalf("missing output block %s", tc.name)
		}
		recipe, ok := MatchSmelting(tc.input)
		if !ok || recipe.OutputID != output.RuntimeID {
			t.Errorf("smelting input %d = %+v, %t; want output %d (%s)", tc.input, recipe, ok, output.RuntimeID, tc.name)
			continue
		}
		furnace := NewFurnaceBlockEntity()
		furnace.InputSlot = ItemStack{ItemID: tc.input, Count: 1}
		furnace.FuelSlot = ItemStack{ItemID: ItemCoal, Count: 1}
		for tick := 0; tick < furnace.CookTimeTotal; tick++ {
			furnace.Tick()
		}
		if got := furnace.OutputSlot; got.ItemID != output.RuntimeID || got.Count != 1 {
			t.Errorf("furnace output for %s = %+v, want one output block %d", tc.name, got, output.RuntimeID)
		}
	}
}
