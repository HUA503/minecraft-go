package inventory

import "testing"

func TestStrawBedCreativeEntryAndRecipe(t *testing.T) {
	definition, ok := ExtendedBlockByID(ItemStrawBed)
	if !ok || definition.Name != "minecraft:straw_bed" || definition.Category != CreativeFunctional {
		t.Fatalf("straw bed block definition = %+v, %t", definition, ok)
	}
	if !IsPlaceableBlock(ItemStrawBed) || GetMaxStackSize(ItemStrawBed) != 1 {
		t.Fatal("straw bed should be a placeable, non-stackable item")
	}
	if items := SearchCreativeCatalog(CreativeFunctional, "minecraft:straw_bed"); len(items) != 1 || items[0].ID != ItemStrawBed {
		t.Fatalf("functional tab straw bed entry = %+v", items)
	}
	grid := []int32{ItemHayBale, ItemHayBale, ItemHayBale, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir, ItemAir}
	crafted, ok := NewCraftingManager().MatchCrafting(grid, 3)
	if !ok || crafted.ItemID != ItemStrawBed || crafted.Count != 4 {
		t.Fatalf("straw bed recipe = %+v, %t", crafted, ok)
	}
}

func TestStrawBedStatesPairHorizontallyForEveryFacing(t *testing.T) {
	for facing := 0; facing < 4; facing++ {
		foot := StrawBedBlockState(ItemStrawBed, facing, false)
		dx, dz, head, ok := StrawBedOtherHalf(foot)
		if !ok {
			t.Fatalf("foot state %d has no head half", foot)
		}
		item, gotFacing, isHead, ok := StrawBedStateProperties(head)
		if !ok || item != ItemStrawBed || gotFacing != facing || !isHead {
			t.Fatalf("head state %d properties = %d/%d/%t/%t", head, item, gotFacing, isHead, ok)
		}
		backX, backZ, pairedFoot, ok := StrawBedOtherHalf(head)
		if !ok || backX != -dx || backZ != -dz || pairedFoot != foot {
			t.Fatalf("head state %d reverse pair = (%d,%d,%d,%t), want (%d,%d,%d,true)", head, backX, backZ, pairedFoot, ok, -dx, -dz, foot)
		}
	}
	for _, tc := range []struct {
		yaw    float64
		facing int
	}{{0, 0}, {90, 1}, {180, 2}, {-90, 3}} {
		state, ok := StrawBedPlacementState(ItemStrawBed, tc.yaw)
		_, facing, head, valid := StrawBedStateProperties(state)
		if !ok || !valid || head || facing != tc.facing {
			t.Errorf("straw bed placement yaw %.0f => state %d, facing %d head %t ok %t; want facing %d foot", tc.yaw, state, facing, head, valid, tc.facing)
		}
	}
}
