package inventory

import "testing"

func TestMaceAndHeavyCoreCreativeItems(t *testing.T) {
	for _, tc := range []struct {
		id       int32
		name     string
		category CreativeCategory
		stack    int
	}{
		{ItemMace, "minecraft:mace", CreativeCombat, 1},
		{ItemHeavyCore, "minecraft:heavy_core", CreativeMaterials, 64},
	} {
		definition, ok := ModernCreativeItemByID(tc.id)
		if !ok || definition.Name != tc.name || definition.Category != tc.category {
			t.Fatalf("creative definition %d = %+v, %t", tc.id, definition, ok)
		}
		if got := GetMaxStackSize(tc.id); got != tc.stack {
			t.Errorf("item %s max stack = %d, want %d", tc.name, got, tc.stack)
		}
		found := false
		for _, item := range CreativeCatalog(CreativeAll) {
			if item.ID == tc.id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s is absent from creative catalog", tc.name)
		}
		tabFound := false
		for _, item := range SearchCreativeCatalog(tc.category, tc.name) {
			if item.ID == tc.id {
				tabFound = true
				break
			}
		}
		if !tabFound {
			t.Errorf("%s is absent from its %d creative tab", tc.name, tc.category)
		}
	}
	props, ok := GetToolProperties(ItemMace)
	if !ok || props.MaxDurability != 500 || props.AttackDamage != 5 {
		t.Fatalf("mace combat properties = %+v, %t", props, ok)
	}
}

func TestMaceRecipeUsesHeavyCoreAndBreezeRod(t *testing.T) {
	grid := []int32{ItemAir, ItemHeavyCore, ItemBreezeRod, ItemAir}
	crafted, ok := NewCraftingManager().MatchCrafting(grid, 2)
	if !ok || crafted.ItemID != ItemMace || crafted.Count != 1 {
		t.Fatalf("mace recipe = %+v, %t", crafted, ok)
	}
}
