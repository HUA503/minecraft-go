package inventory

import "testing"

func TestMossCarpetsCatalogRecipesAndShapes(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		name string
	}{
		{ItemMossCarpet, "minecraft:moss_carpet"},
		{ItemPaleMossCarpet, "minecraft:pale_moss_carpet"},
	} {
		definition, ok := ExtendedBlockByName(tc.name)
		if !ok || definition.RuntimeID != tc.id || definition.Category != CreativeNatural || definition.DropRuntimeID != tc.id {
			t.Errorf("definition for %s = %+v, found=%v", tc.name, definition, ok)
		}
		if !IsThinCarpetBlock(tc.id) || !IsPlaceableBlock(tc.id) || GetBlockHardness(tc.id) != 0.1 {
			t.Errorf("%s is missing thin carpet behavior", tc.name)
		}
		boxes, ok := BlockCollisionBoxes(tc.id)
		if !ok || len(boxes) != 1 || boxes[0].MaxY != 1.0/16.0 {
			t.Errorf("%s collision boxes = %+v, found=%v; want one 1/16-high box", tc.name, boxes, ok)
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
	manager := NewCraftingManager()
	for _, tc := range []struct {
		carpetID int32
		source   string
	}{{ItemMossCarpet, "minecraft:moss_block"}, {ItemPaleMossCarpet, "minecraft:pale_moss_block"}} {
		source, ok := ExtendedBlockByName(tc.source)
		if !ok {
			t.Fatalf("missing source block %s", tc.source)
		}
		result, ok := manager.MatchCrafting([]int32{source.RuntimeID, source.RuntimeID, 0, 0}, 2)
		if !ok || result.ItemID != tc.carpetID || result.Count != 3 {
			t.Errorf("crafting two source blocks returned %+v, matched=%v; want 3 of %d", result, ok, tc.carpetID)
		}
	}
}
