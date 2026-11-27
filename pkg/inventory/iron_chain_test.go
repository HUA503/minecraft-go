package inventory

import "testing"

func TestJava263IronChainCatalogRecipeStateAndCollision(t *testing.T) {
	definition, ok := ExtendedBlockByID(ItemIronChain)
	if !ok || definition.Name != "minecraft:iron_chain" || definition.TextureSide != "iron_chain" || definition.Category != CreativeFunctional || definition.Hardness != 5 || definition.HarvestTool != ToolPickaxe || !definition.RequiresHarvestTool || definition.DropRuntimeID != ItemIronChain {
		t.Fatalf("iron chain definition = %+v, found=%v", definition, ok)
	}
	if !IsPlaceableBlock(ItemIronChain) || GetMaxStackSize(ItemIronChain) != 64 {
		t.Fatal("iron chain should be a placeable stackable block")
	}
	for _, category := range []CreativeCategory{CreativeBlocks, CreativeFunctional} {
		found := false
		for _, item := range CreativeCatalog(category) {
			if item.ID == ItemIronChain {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("iron chain missing from creative category %d", category)
		}
	}

	crafted, matched := NewCraftingManager().MatchCrafting([]int32{
		ItemAir, ItemIronNugget, ItemAir,
		ItemAir, ItemIronIngot, ItemAir,
		ItemAir, ItemIronNugget, ItemAir,
	}, 3)
	if !matched || crafted != (ItemStack{ItemID: ItemIronChain, Count: 1}) {
		t.Fatalf("iron chain recipe = %+v, matched=%v", crafted, matched)
	}

	for axis := 0; axis < 3; axis++ {
		state, ok := IronChainBlockState(axis)
		gotAxis, stateOK := IronChainStateProperties(state)
		itemID := BlockItemForState(state)
		boxes, shapeOK := BlockCollisionBoxes(state)
		if !ok || !stateOK || gotAxis != axis || itemID != ItemIronChain || !shapeOK || len(boxes) != 1 {
			t.Errorf("axis %d state=%d axis=%d/%v item=%d shape=%v boxes=%v", axis, state, gotAxis, stateOK, itemID, shapeOK, boxes)
			continue
		}
		if boxes[0].MaxX-boxes[0].MinX != 1 && boxes[0].MaxY-boxes[0].MinY != 1 && boxes[0].MaxZ-boxes[0].MinZ != 1 {
			t.Errorf("axis %d chain box has no full-length axis: %+v", axis, boxes[0])
		}
	}
	for _, face := range [][3]int32{{0, 1, 0}, {1, 0, 0}, {0, 0, -1}} {
		state, ok := IronChainPlacementState(ItemIronChain, face[0], face[1], face[2])
		axis, stateOK := IronChainStateProperties(state)
		if !ok || !stateOK || axis != intAxis(face) {
			t.Errorf("face %v placement state=%d axis=%d valid=%v/%v", face, state, axis, ok, stateOK)
		}
	}
	if _, ok := IronChainPlacementState(ItemIronChain, 0, 0, 0); ok {
		t.Fatal("zero face vector should not place a chain")
	}
	if _, ok := IronChainPlacementState(ItemIronIngot, 0, 1, 0); ok {
		t.Fatal("non-chain item should not produce chain state")
	}
	if CanHarvestBlock(ItemAir, ItemIronChain) || !CanHarvestBlock(ItemWoodenPickaxe, ItemIronChain) {
		t.Fatal("iron chain harvest rule should require a pickaxe")
	}
	if got := len(CreativeCatalog(CreativeAll)); got != 973 {
		t.Fatalf("all-items catalog count=%d, want 973", got)
	}
}

func intAxis(face [3]int32) int {
	if face[1] != 0 {
		return 0
	}
	if face[0] != 0 {
		return 1
	}
	return 2
}
