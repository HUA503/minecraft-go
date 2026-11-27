package inventory

import "testing"

func TestSmallFlowerRegistryAndCreativeEntries(t *testing.T) {
	flowers := []string{
		"blue_orchid", "allium", "azure_bluet", "red_tulip", "orange_tulip",
		"white_tulip", "pink_tulip", "oxeye_daisy", "cornflower", "lily_of_the_valley",
	}
	for index, name := range flowers {
		id := FirstSmallFlower + int32(index)
		definition, ok := ExtendedBlockByName("minecraft:" + name)
		if !ok || definition.RuntimeID != id || definition.TextureSide != name || definition.Category != CreativeNatural || definition.DropRuntimeID != id {
			t.Errorf("flower %s definition = %+v, %v", name, definition, ok)
		}
		if !IsPlaceableBlock(id) || !IsSmallFlower(id) || !IsExtendedFoliage(id) || !IsReplaceableBlock(id) {
			t.Errorf("flower %s is missing plant block behavior", name)
		}
		if got := GetBlockHardness(id); got != 0 || CalculateBlockDamagePerSecond(ItemAir, id) < 100 {
			t.Errorf("flower %s hardness/damage = %v/%v; want instant break", name, got, CalculateBlockDamagePerSecond(ItemAir, id))
		}
		found := false
		for _, item := range CreativeCatalog(CreativeNatural) {
			if item.ID == id && item.Name != "" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("flower %s is missing from the Natural creative tab", name)
		}
		if got := CreativeItemDisplayName(id); got == "" || got == "物品 #0" {
			t.Errorf("flower %s has no display name", name)
		}
	}
}

func TestSmallFlowerSoilWhitelist(t *testing.T) {
	for _, block := range []int32{ItemGrass, ItemDirt, 14205, 14206} { // grass, dirt, podzol, mycelium
		if !IsFlowerSoilBlock(block) {
			t.Errorf("flower soil block %d was rejected", block)
		}
	}
	for _, block := range []int32{0, ItemStone, ItemOakPlanks, 14204} {
		if IsFlowerSoilBlock(block) {
			t.Errorf("non-soil block %d can support a flower", block)
		}
	}
}
