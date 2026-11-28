package inventory

import "testing"

func TestSmallFlowerRegistryAndCreativeEntries(t *testing.T) {
	flowers := []struct {
		name string
		id   int32
	}{
		{"blue_orchid", FirstSmallFlower}, {"allium", FirstSmallFlower + 1}, {"azure_bluet", FirstSmallFlower + 2},
		{"red_tulip", FirstSmallFlower + 3}, {"orange_tulip", FirstSmallFlower + 4}, {"white_tulip", FirstSmallFlower + 5},
		{"pink_tulip", FirstSmallFlower + 6}, {"oxeye_daisy", FirstSmallFlower + 7}, {"cornflower", FirstSmallFlower + 8},
		{"lily_of_the_valley", FirstSmallFlower + 9}, {"dandelion", ItemDandelion}, {"poppy", ItemPoppy},
		{"wither_rose", ItemWitherRose},
	}
	for _, flower := range flowers {
		definition, ok := ExtendedBlockByName("minecraft:" + flower.name)
		if !ok || definition.RuntimeID != flower.id || definition.TextureSide != flower.name || definition.Category != CreativeNatural || definition.DropRuntimeID != flower.id {
			t.Errorf("flower %s definition = %+v, %v", flower.name, definition, ok)
		}
		if !IsPlaceableBlock(flower.id) || !IsSmallFlower(flower.id) || !IsExtendedFoliage(flower.id) || !IsReplaceableBlock(flower.id) {
			t.Errorf("flower %s is missing plant block behavior", flower.name)
		}
		if got := GetBlockHardness(flower.id); got != 0 || CalculateBlockDamagePerSecond(ItemAir, flower.id) < 100 {
			t.Errorf("flower %s hardness/damage = %v/%v; want instant break", flower.name, got, CalculateBlockDamagePerSecond(ItemAir, flower.id))
		}
		found := false
		for _, item := range CreativeCatalog(CreativeNatural) {
			if item.ID == flower.id && item.Name != "" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("flower %s is missing from the Natural creative tab", flower.name)
		}
		if got := CreativeItemDisplayName(flower.id); got == "" || got == "物品 #0" {
			t.Errorf("flower %s has no display name", flower.name)
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

func TestDandelionAndPoppyDyeRecipes(t *testing.T) {
	for _, tc := range []struct {
		flower int32
		dye    int32
	}{{ItemDandelion, ItemYellowDye}, {ItemPoppy, ItemRedDye}} {
		got, ok := NewCraftingManager().MatchCrafting([]int32{tc.flower, ItemAir, ItemAir, ItemAir}, 2)
		if !ok || got != (ItemStack{ItemID: tc.dye, Count: 1}) {
			t.Errorf("flower %d dye recipe = %+v, matched=%v; want one dye %d", tc.flower, got, ok, tc.dye)
		}
	}
}
