package inventory

import "testing"

func TestSpecialCubeItemsAreCreativelyAvailableAndMineCorrectly(t *testing.T) {
	cases := []struct {
		name         string
		id           int32
		drop         int32
		min, max     int
		category     CreativeCategory
		hardness     float64
		requiredTier int
	}{
		{"crying_obsidian", 14200, 14200, 1, 1, CreativeBlocks, 50, 3},
		{"crimson_nylium", 14201, 87, 1, 1, CreativeNatural, 0.4, 0},
		{"warped_nylium", 14202, 87, 1, 1, CreativeNatural, 0.4, 0},
		{"warped_wart_block", 14203, 14203, 1, 1, CreativeNatural, 1, 0},
		{"nether_gold_ore", 14204, ItemGoldNugget, 2, 6, CreativeNatural, 3, 0},
	}
	all := CreativeCatalog(CreativeAll)
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			block, ok := ExtendedBlockByName("minecraft:" + test.name)
			if !ok {
				t.Fatal("block has no extended definition")
			}
			if block.RuntimeID != test.id || block.DropRuntimeID != test.drop || block.Hardness != test.hardness || block.RequiredHarvestLevel != test.requiredTier {
				t.Fatalf("block definition = %+v", block)
			}
			if min, max := BlockDropCountRange(test.id); min != test.min || max != test.max {
				t.Fatalf("drop range = %d..%d, want %d..%d", min, max, test.min, test.max)
			}
			if !IsPlaceableBlock(test.id) || BlockItemForState(test.id) != test.id {
				t.Fatalf("item %d is not registered as a placeable block", test.id)
			}
			found := false
			for _, item := range all {
				if item.ID == test.id && item.Category == test.category {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("item missing from creative category %d", test.category)
			}
		})
	}
}
