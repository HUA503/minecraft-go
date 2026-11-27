package inventory

import "testing"

func TestDryGrassCatalogSupportTagsAndStates(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		name string
	}{
		{ItemDryShortGrass, "minecraft:dry_short_grass"},
		{ItemDryTallGrass, "minecraft:dry_tall_grass"},
	} {
		definition, ok := ExtendedBlockByName(tc.name)
		if !ok || definition.RuntimeID != tc.id || definition.Category != CreativeNatural || definition.DropRuntimeID != tc.id {
			t.Errorf("definition for %s = %+v, found=%v", tc.name, definition, ok)
		}
		if !IsPlaceableBlock(tc.id) || !IsDryGrass(tc.id) || !IsDecorativePlantBlock(tc.id) || !IsExtendedFoliage(tc.id) || !IsReplaceableBlock(tc.id) {
			t.Errorf("%s is missing dry grass plant behavior", tc.name)
		}
		if GetBlockHardness(tc.id) != 0 {
			t.Errorf("%s hardness = %v, want instant break", tc.name, GetBlockHardness(tc.id))
		}
		found := false
		for _, item := range CreativeCatalog(CreativeNatural) {
			if item.ID == tc.id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s missing from Natural creative tab", tc.name)
		}
	}
	for _, id := range []int32{ItemGrass, ItemDirt, 12, 60, 14205, 14206} { // grass/dirt, sand, farmland, podzol, mycelium
		if !IsDryVegetationSupport(id) {
			t.Errorf("dry grass support block %d rejected", id)
		}
	}
	if coarseDirt, ok := ExtendedBlockByName("minecraft:coarse_dirt"); !ok || !IsDryVegetationSupport(coarseDirt.RuntimeID) {
		t.Errorf("coarse dirt support definition = %+v, found=%v", coarseDirt, ok)
	}
	for _, id := range []int32{ItemDryShortGrass, ItemDryTallGrass} {
		if got := GetFuelBurnTime(id); got != 100 {
			t.Errorf("dry grass %d burn time = %d, want 100 ticks", id, got)
		}
	}
	for _, id := range []int32{0, ItemStone, ItemOakPlanks, 14207} {
		if IsDryVegetationSupport(id) {
			t.Errorf("non-tagged block %d supports dry grass", id)
		}
	}
}
