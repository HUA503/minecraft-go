package inventory

import "testing"

func TestRedstoneControlItemsStatesAndRecipes(t *testing.T) {
	if len(pressurePlateDefinitions) != 15 || len(weightedPressurePlateDefinitions) != 2 || len(buttonDefinitions) != 15 {
		t.Fatalf("control family sizes = (%d,%d,%d), want 15 regular plates, 2 weighted plates, 15 buttons", len(pressurePlateDefinitions), len(weightedPressurePlateDefinitions), len(buttonDefinitions))
	}
	catalog := CreativeCatalog(CreativeAll)
	seen := make(map[int32]bool, len(catalog))
	for _, item := range catalog {
		seen[item.ID] = true
	}
	crafting := NewCraftingManager()
	for _, group := range [][]controlBlockDefinition{allPressurePlateDefinitions(), buttonDefinitions} {
		for _, definition := range group {
			if !seen[definition.itemID] {
				t.Errorf("%s missing from creative inventory", definition.name)
			}
			if !IsPlaceableBlock(definition.itemID) {
				t.Errorf("%s is not placeable", definition.name)
			}
			if definition.kind == "button" {
				state := ButtonBlockState(definition.itemID, 5, true)
				itemID, facing, powered, ok := ButtonStateProperties(state)
				if !ok || itemID != definition.itemID || facing != 5 || !powered || BlockItemForState(state) != definition.itemID || !IsPlaceableBlock(state) {
					t.Errorf("button %s state did not round trip: id=%d facing=%d powered=%t ok=%t", definition.name, itemID, facing, powered, ok)
				}
				if _, ok := BlockCollisionBoxes(state); !ok {
					t.Errorf("button %s has no model shape", definition.name)
				}
				out, matched := crafting.MatchCrafting([]int32{definitionInputID(t, definition), 0, 0, 0}, 2)
				if !matched || out != (ItemStack{ItemID: definition.itemID, Count: 1}) {
					t.Errorf("button recipe %s = %+v, matched=%t", definition.name, out, matched)
				}
			} else {
				state := PressurePlateBlockState(definition.itemID, true)
				itemID, powered, ok := PressurePlateStateProperties(state)
				if !ok || itemID != definition.itemID || !powered || BlockItemForState(state) != definition.itemID || !IsPlaceableBlock(state) {
					t.Errorf("pressure plate %s state did not round trip: id=%d powered=%t ok=%t", definition.name, itemID, powered, ok)
				}
				if _, ok := BlockCollisionBoxes(state); !ok {
					t.Errorf("pressure plate %s has no model shape", definition.name)
				}
				input := definitionInputID(t, definition)
				out, matched := crafting.MatchCrafting([]int32{input, input, 0, 0}, 2)
				if !matched || out != (ItemStack{ItemID: definition.itemID, Count: 1}) {
					t.Errorf("pressure plate recipe %s = %+v, matched=%t", definition.name, out, matched)
				}
			}
		}
	}
	for _, tc := range []struct {
		itemID int32
		count  int
		want   int
	}{
		{ItemLightWeightedPressurePlate, 1, 1}, {ItemLightWeightedPressurePlate, 100, 15},
		{ItemHeavyWeightedPressurePlate, 0, 0}, {ItemHeavyWeightedPressurePlate, 9, 1},
		{ItemHeavyWeightedPressurePlate, 10, 1}, {ItemHeavyWeightedPressurePlate, 11, 2},
		{ItemHeavyWeightedPressurePlate, 149, 15}, {ItemHeavyWeightedPressurePlate, 150, 15},
	} {
		if got := WeightedPressurePlateSignal(tc.itemID, tc.count); got != tc.want {
			t.Errorf("weighted plate %d with %d entities signal=%d, want %d", tc.itemID, tc.count, got, tc.want)
		}
	}
	for _, definition := range weightedPressurePlateDefinitions {
		for signal := 0; signal <= 15; signal++ {
			state := WeightedPressurePlateBlockState(definition.itemID, signal)
			itemID, powered, ok := PressurePlateStateProperties(state)
			if !ok || itemID != definition.itemID || powered != (signal > 0) || PressurePlateSignalStrength(state) != signal || BlockItemForState(state) != definition.itemID {
				t.Errorf("weighted plate %s signal %d did not round-trip: state=%d item=%d powered=%t ok=%t level=%d", definition.name, signal, state, itemID, powered, ok, PressurePlateSignalStrength(state))
			}
		}
	}
	if len(CreativeCatalog(CreativeRedstone)) == 0 {
		t.Fatal("redstone creative category is empty")
	}
}

func definitionInputID(t *testing.T, definition controlBlockDefinition) int32 {
	t.Helper()
	if block, ok := ExtendedBlockByName(definition.source); ok {
		return block.RuntimeID
	}
	for _, item := range CreativeCatalog(CreativeAll) {
		if creativeItemRegistryName(item.ID) == definition.source {
			return item.ID
		}
	}
	t.Fatalf("recipe input %s for %s is unavailable", definition.source, definition.name)
	return 0
}

func TestButtonFacingRequiresCorrectSupportOffset(t *testing.T) {
	tests := []struct {
		facing int
		want   [3]int32
	}{
		{0, [3]int32{0, 1, 0}}, {1, [3]int32{0, -1, 0}},
		{2, [3]int32{0, 0, 1}}, {3, [3]int32{0, 0, -1}},
		{4, [3]int32{1, 0, 0}}, {5, [3]int32{-1, 0, 0}},
	}
	for _, test := range tests {
		state := ButtonBlockState(ItemOakButton, test.facing, false)
		x, y, z, ok := ButtonSupportOffset(state)
		if !ok || [3]int32{x, y, z} != test.want {
			t.Errorf("facing %d support offset = (%d,%d,%d,%t), want %v", test.facing, x, y, z, ok, test.want)
		}
	}
}
