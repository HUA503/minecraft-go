package inventory

import "testing"

func TestJava263SpearsAreInCombatCreativeCatalog(t *testing.T) {
	want := []struct {
		id   int32
		name string
	}{
		{ItemWoodenSpear, "minecraft:wooden_spear"},
		{ItemStoneSpear, "minecraft:stone_spear"},
		{ItemCopperSpear, "minecraft:copper_spear"},
		{ItemIronSpear, "minecraft:iron_spear"},
		{ItemGoldenSpear, "minecraft:golden_spear"},
		{ItemDiamondSpear, "minecraft:diamond_spear"},
		{ItemNetheriteSpear, "minecraft:netherite_spear"},
	}

	items := CreativeCatalog(CreativeCombat)
	for _, expected := range want {
		definition, ok := ModernCreativeItemByID(expected.id)
		if !ok || definition.Name != expected.name || definition.Category != CreativeCombat || definition.MaxStack != 1 {
			t.Errorf("spear definition for %q = %+v, found=%t", expected.name, definition, ok)
		}
		found := false
		for _, item := range items {
			if item.ID == expected.id && creativeItemRegistryName(item.ID) == expected.name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s missing from combat creative tab", expected.name)
		}
	}
}
