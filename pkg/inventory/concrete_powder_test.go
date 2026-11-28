package inventory

import "testing"

func TestJava263ColoredConcretePowderCatalogAndHardening(t *testing.T) {
	colors := []string{"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray", "light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black"}
	seen := make(map[int32]bool)
	for index, color := range colors {
		name := "minecraft:" + color + "_concrete_powder"
		powder, ok := ExtendedBlockByName(name)
		if !ok || powder.RuntimeID != FirstConcretePowder+int32(index) || powder.Category != CreativeDecoration || powder.Hardness != 0.5 || powder.DropRuntimeID != powder.RuntimeID {
			t.Fatalf("powder %s definition = %+v found=%v", name, powder, ok)
		}
		if seen[powder.RuntimeID] {
			t.Fatalf("duplicate concrete powder runtime ID %d", powder.RuntimeID)
		}
		seen[powder.RuntimeID] = true
		if got := SearchCreativeCatalog(CreativeAll, name); len(got) != 1 || got[0].ID != powder.RuntimeID {
			t.Errorf("search for %s = %+v", name, got)
		}
		concrete, ok := ConcretePowderHardenedBlock(powder.RuntimeID)
		if !ok {
			t.Fatalf("powder %s has no hardened concrete block", name)
		}
		want, exists := ExtendedBlockByName("minecraft:" + color + "_concrete")
		if !exists || concrete != want.RuntimeID {
			t.Errorf("powder %s hardens to %d, want %+v", name, concrete, want)
		}
	}
	if !IsConcretePowderBlock(FirstConcretePowder) || !IsConcretePowderBlock(FirstConcretePowder+15) || IsConcretePowderBlock(FirstConcretePowder+16) {
		t.Fatal("concrete powder ID range is incorrect")
	}
}

func TestConcretePowderRecipesUseMatchingDye(t *testing.T) {
	manager := NewCraftingManager()
	colors := []struct {
		name string
		dye  int32
	}{
		{"white", ItemWhiteDye}, {"orange", ItemOrangeDye}, {"magenta", ItemMagentaDye}, {"light_blue", ItemLightBlueDye},
		{"yellow", ItemYellowDye}, {"lime", ItemLimeDye}, {"pink", ItemPinkDye}, {"gray", ItemGrayDye},
		{"light_gray", ItemLightGrayDye}, {"cyan", ItemCyanDye}, {"purple", ItemPurpleDye}, {"blue", ItemBlueDye},
		{"brown", ItemBrownDye}, {"green", ItemGreenDye}, {"red", ItemRedDye}, {"black", ItemBlackDye},
	}
	for _, color := range colors {
		powder, _ := ExtendedBlockByName("minecraft:" + color.name + "_concrete_powder")
		grid := []int32{12, 13, 12, 13, color.dye, 13, 12, 13, 12}
		crafted, ok := manager.MatchCrafting(grid, 3)
		if !ok || crafted.ItemID != powder.RuntimeID || crafted.Count != 8 {
			t.Errorf("crafting %s powder = %+v, found=%v", color.name, crafted, ok)
		}
	}
}
