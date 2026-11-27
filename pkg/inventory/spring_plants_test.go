package inventory

import "testing"

func TestBushAndCactusFlowerCreativeAndDrops(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		name string
	}{
		{ItemBush, "minecraft:bush"},
		{ItemCactusFlower, "minecraft:cactus_flower"},
	} {
		definition, ok := ExtendedBlockByName(tc.name)
		if !ok || definition.RuntimeID != tc.id || definition.Category != CreativeNatural || definition.DropRuntimeID != tc.id {
			t.Errorf("definition for %s = %+v, found=%v", tc.name, definition, ok)
		}
		if !IsPlaceableBlock(tc.id) || !IsExtendedFoliage(tc.id) || !IsDecorativePlantBlock(tc.id) || !IsReplaceableBlock(tc.id) {
			t.Errorf("%s is missing placeable plant behavior", tc.name)
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
	bush, _ := ExtendedBlockByID(ItemBush)
	if !bush.DropsOnlyWithShears {
		t.Fatal("bush must require shears for its item drop")
	}
	flower, _ := ExtendedBlockByID(ItemCactusFlower)
	if flower.DropsOnlyWithShears {
		t.Fatal("cactus flower should drop without shears")
	}
	result, ok := NewCraftingManager().MatchCrafting([]int32{ItemCactusFlower, 0, 0, 0}, 2)
	if !ok || result.ItemID != ItemPinkDye || result.Count != 1 {
		t.Fatalf("cactus flower dye recipe = %+v, matched=%v; want one pink dye", result, ok)
	}
}
