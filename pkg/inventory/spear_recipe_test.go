package inventory

import "testing"

func TestJava263SpearCraftingRecipes(t *testing.T) {
	manager := NewCraftingManager()
	tests := []struct {
		name     string
		material int32
		output   int32
	}{
		{"oak planks", ItemOakPlanks, ItemWoodenSpear},
		{"poplar planks", mustExtendedItemID(t, "minecraft:poplar_planks"), ItemWoodenSpear},
		{"blackstone", mustExtendedItemID(t, "minecraft:blackstone"), ItemStoneSpear},
		{"cobbled deepslate", mustExtendedItemID(t, "minecraft:cobbled_deepslate"), ItemStoneSpear},
		{"copper ingot", ItemCopperIngot, ItemCopperSpear},
		{"iron ingot", ItemIronIngot, ItemIronSpear},
		{"gold ingot", ItemGoldIngot, ItemGoldenSpear},
		{"diamond", ItemDiamond, ItemDiamondSpear},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grid := []int32{
				ItemAir, ItemAir, test.material,
				ItemAir, ItemStick, ItemAir,
				ItemStick, ItemAir, ItemAir,
			}
			got, ok := manager.MatchCrafting(grid, 3)
			if !ok || got.ItemID != test.output || got.Count != 1 {
				t.Fatalf("MatchCrafting() = %+v, %t; want item %d x1", got, ok, test.output)
			}
		})
	}
}

func mustExtendedItemID(t *testing.T, name string) int32 {
	t.Helper()
	definition, ok := ExtendedBlockByName(name)
	if !ok {
		t.Fatalf("extended block %s is missing", name)
	}
	return definition.RuntimeID
}
