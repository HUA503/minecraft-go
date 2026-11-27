package inventory

import "testing"

func TestJava263ColoredCarpetsCatalogRecipesDropsAndShape(t *testing.T) {
	carpetCount := 0
	for _, definition := range ExtendedBlockDefinitions() {
		if !IsCarpetBlock(definition.RuntimeID) {
			continue
		}
		carpetCount++
		if definition.Category != CreativeDecoration || definition.DropRuntimeID != definition.RuntimeID {
			t.Errorf("carpet definition %s has category/drop %d/%d", definition.Name, definition.Category, definition.DropRuntimeID)
		}
		if !IsPlaceableBlock(definition.RuntimeID) {
			t.Errorf("carpet %s is not placeable", definition.Name)
		}
		matches := SearchCreativeCatalog(CreativeDecoration, definition.Name)
		if len(matches) != 1 || matches[0].ID != definition.RuntimeID {
			t.Errorf("creative search for %s = %+v", definition.Name, matches)
		}
		boxes, ok := BlockCollisionBoxes(definition.RuntimeID)
		if !ok || len(boxes) != 1 || boxes[0].MaxY != 1.0/16.0 {
			t.Errorf("carpet %s collision shape = %+v, %v; want one 1/16-high full-footprint box", definition.Name, boxes, ok)
		}
	}
	if carpetCount != 16 {
		t.Fatalf("colored carpet definitions = %d, want 16", carpetCount)
	}
	if got := len(CreativeCatalog(CreativeAll)); got != 1009 {
		t.Fatalf("all-items creative catalog = %d, want 1009", got)
	}

	manager := NewCraftingManager()
	for _, definition := range ExtendedBlockDefinitions() {
		if !IsCarpetBlock(definition.RuntimeID) {
			continue
		}
		woolID := int32(35)
		if definition.Name != "minecraft:white_carpet" {
			woolName := definition.Name[:len(definition.Name)-len("carpet")] + "wool"
			wool, ok := ExtendedBlockByName(woolName)
			if !ok {
				t.Fatalf("matching wool %s missing for %s", woolName, definition.Name)
			}
			woolID = wool.RuntimeID
		}
		for _, grid := range [][]int32{
			{woolID, woolID, 0, 0, 0, 0, 0, 0, 0},
			{woolID, 0, 0, woolID, 0, 0, 0, 0, 0},
		} {
			got, ok := manager.MatchCrafting(grid, 3)
			if !ok || got.ItemID != definition.RuntimeID || got.Count != 3 {
				t.Errorf("crafting two %d into %s = %+v, %v; want 3 carpets", woolID, definition.Name, got, ok)
			}
		}
	}
}
