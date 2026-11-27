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
		{"podzol", 14205, ItemDirt, 1, 1, CreativeNatural, 0.5, 0},
		{"mycelium", 14206, ItemDirt, 1, 1, CreativeNatural, 0.6, 0},
		{"magma_block", 14207, 14207, 1, 1, CreativeNatural, 0.5, 0},
		{"dried_kelp_block", 14208, 14208, 1, 1, CreativeNatural, 0.5, 0},
		{"tube_coral_block", 14209, 14214, 1, 1, CreativeNatural, 1.5, 0},
		{"brain_coral_block", 14210, 14215, 1, 1, CreativeNatural, 1.5, 0},
		{"bubble_coral_block", 14211, 14216, 1, 1, CreativeNatural, 1.5, 0},
		{"fire_coral_block", 14212, 14217, 1, 1, CreativeNatural, 1.5, 0},
		{"horn_coral_block", 14213, 14218, 1, 1, CreativeNatural, 1.5, 0},
		{"dead_tube_coral_block", 14214, 14214, 1, 1, CreativeNatural, 1.5, 0},
		{"dead_brain_coral_block", 14215, 14215, 1, 1, CreativeNatural, 1.5, 0},
		{"dead_bubble_coral_block", 14216, 14216, 1, 1, CreativeNatural, 1.5, 0},
		{"dead_fire_coral_block", 14217, 14217, 1, 1, CreativeNatural, 1.5, 0},
		{"dead_horn_coral_block", 14218, 14218, 1, 1, CreativeNatural, 1.5, 0},
		{"ochre_froglight", 14219, 14219, 1, 1, CreativeNatural, 0.3, 0},
		{"verdant_froglight", 14220, 14220, 1, 1, CreativeNatural, 0.3, 0},
		{"pearlescent_froglight", 14221, 14221, 1, 1, CreativeNatural, 0.3, 0},
		{"reinforced_deepslate", 14222, 0, 0, 0, CreativeBlocks, 55, 0},
		{"gilded_blackstone", 14223, ItemGoldNugget, 2, 5, CreativeBlocks, 1.5, 0},
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
			if test.name == "reinforced_deepslate" && !block.DropsNothing {
				t.Fatal("reinforced deepslate must not drop an item when mined")
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
					if item.Name == "" {
						t.Fatal("creative display name is empty")
					}
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
