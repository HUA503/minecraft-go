package inventory

import "testing"

func TestJava263AdditionalTreeLeavesAndSaplingsCatalog(t *testing.T) {
	ids := make(map[int32]string, len(ExtendedBlockDefinitions()))
	for _, definition := range ExtendedBlockDefinitions() {
		if previous, duplicate := ids[definition.RuntimeID]; duplicate {
			t.Fatalf("extended block runtime ID %d is shared by %s and %s", definition.RuntimeID, previous, definition.Name)
		}
		ids[definition.RuntimeID] = definition.Name
	}
	leaves := []struct {
		id            int32
		name, sapling string
	}{
		{FirstAdditionalLeaves, "jungle_leaves", "jungle_sapling"},
		{FirstAdditionalLeaves + 1, "acacia_leaves", "acacia_sapling"},
		{FirstAdditionalLeaves + 2, "dark_oak_leaves", "dark_oak_sapling"},
		{FirstAdditionalLeaves + 3, "mangrove_leaves", "mangrove_propagule"},
		{FirstAdditionalLeaves + 4, "cherry_leaves", "cherry_sapling"},
		{FirstAdditionalLeaves + 5, "pale_oak_leaves", "pale_oak_sapling"},
		{FirstAdditionalLeaves + 6, "azalea_leaves", "azalea"},
		{FirstAdditionalLeaves + 7, "flowering_azalea_leaves", "flowering_azalea"},
	}
	for _, tc := range leaves {
		definition, ok := ExtendedBlockByName("minecraft:" + tc.name)
		if !ok || definition.RuntimeID != tc.id || definition.DropRuntimeID == 0 || !IsExtendedLeaves(tc.id) {
			t.Fatalf("leaf %s definition=%+v found=%v", tc.name, definition, ok)
		}
		sapling, ok := ExtendedBlockByName("minecraft:" + tc.sapling)
		if !ok || definition.DropRuntimeID != sapling.RuntimeID {
			t.Fatalf("leaf %s drop=%d, %s=%+v found=%v", tc.name, definition.DropRuntimeID, tc.sapling, sapling, ok)
		}
		if !containsCreativeItem(CreativeCatalog(CreativeNatural), tc.id) || !containsCreativeItem(CreativeCatalog(CreativeNatural), sapling.RuntimeID) {
			t.Fatalf("%s or %s missing from Natural tab", tc.name, tc.sapling)
		}
	}
	for id := FirstAdditionalSapling; id < FirstAdditionalSapling+10; id++ {
		if !IsAdditionalTreeSapling(id) || !IsExtendedFoliage(id) || !IsReplaceableBlock(id) || !IsPlaceableBlock(id) {
			t.Fatalf("tree sapling %d missing placement/foliage semantics", id)
		}
	}
}
