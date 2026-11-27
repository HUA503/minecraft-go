package inventory

import "testing"

func TestJava263WoodFencesCatalogRecipesAndConnectedShapes(t *testing.T) {
	if len(woodFenceDefinitions) != WoodFenceCount {
		t.Fatalf("wood fence definitions = %d, want %d", len(woodFenceDefinitions), WoodFenceCount)
	}
	all := CreativeCatalog(CreativeAll)
	for _, fence := range woodFenceDefinitions {
		definition, ok := ExtendedBlockByID(fence.ItemID)
		if !ok || definition.Name != fence.Name || definition.DropRuntimeID != fence.ItemID || definition.Category != CreativeBlocks {
			t.Errorf("fence registration for %s = %+v, ok=%t", fence.Name, definition, ok)
		}
		if matches := SearchCreativeCatalog(CreativeBlocks, fence.Name); len(matches) != 1 || matches[0].ID != fence.ItemID {
			t.Errorf("registry-name search for %s = %+v", fence.Name, matches)
		}
		found := false
		for _, item := range all {
			found = found || item.ID == fence.ItemID
		}
		if !found {
			t.Errorf("fence %s is missing from the all-items catalog", fence.Name)
		}
	}

	oakFence := woodFenceDefinitions[0]
	grid := []int32{ItemOakPlanks, ItemStick, ItemOakPlanks, ItemOakPlanks, ItemStick, ItemOakPlanks, 0, 0, 0}
	crafted, ok := NewCraftingManager().MatchCrafting(grid, 3)
	if !ok || crafted.ItemID != oakFence.ItemID || crafted.Count != 3 {
		t.Fatalf("oak fence recipe = %+v, matched=%t", crafted, ok)
	}

	blocks := map[[3]int32]int32{{0, 0, 0}: oakFence.ItemID, {0, 0, -1}: ItemOakPlanks}
	getBlock := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
	isolated, _ := WoodFenceCollisionBoxesAt(oakFence.ItemID, 0, 0, 0, func(int32, int32, int32) int32 { return 0 })
	if len(isolated) != 1 || isolated[0].MaxY != 1.5 {
		t.Fatalf("isolated fence collision boxes = %+v, want 1 post 1.5 blocks tall", isolated)
	}
	northConnected, _ := WoodFenceCollisionBoxesAt(oakFence.ItemID, 0, 0, 0, getBlock)
	if len(northConnected) != 3 {
		t.Fatalf("north-connected fence collision boxes = %d, want post and two north rails", len(northConnected))
	}
	for _, box := range northConnected[1:] {
		if box.MinZ != 0 || box.MaxZ != .5 {
			t.Errorf("north rail reaches the wrong side of post: %+v", box)
		}
	}
}

func TestJava263NetherBrickFenceCatalogRecipeAndConnectedShape(t *testing.T) {
	definition, ok := ExtendedBlockByID(ItemNetherBrickFence)
	if !ok || definition.Name != "minecraft:nether_brick_fence" || definition.DropRuntimeID != ItemNetherBrickFence {
		t.Fatalf("nether brick fence block definition = %+v, ok=%t", definition, ok)
	}
	if matches := SearchCreativeCatalog(CreativeBlocks, "minecraft:nether_brick_fence"); len(matches) != 1 || matches[0].ID != ItemNetherBrickFence {
		t.Fatalf("nether brick fence catalog lookup = %+v", matches)
	}
	netherBricks := int32(112)
	grid := []int32{
		netherBricks, ItemNetherBrick, netherBricks,
		netherBricks, ItemNetherBrick, netherBricks,
		0, 0, 0,
	}
	crafted, ok := NewCraftingManager().MatchCrafting(grid, 3)
	if !ok || crafted.ItemID != ItemNetherBrickFence || crafted.Count != 6 {
		t.Fatalf("nether brick fence recipe = %+v, matched=%t", crafted, ok)
	}
	blocks := map[[3]int32]int32{{0, 0, 0}: ItemNetherBrickFence, {1, 0, 0}: netherBricks}
	boxes, ok := FenceCollisionBoxesAt(ItemNetherBrickFence, 0, 0, 0, func(x, y, z int32) int32 {
		return blocks[[3]int32{x, y, z}]
	})
	if !ok || !IsFenceBlock(ItemNetherBrickFence) || IsWoodFenceBlock(ItemNetherBrickFence) || len(boxes) != 3 {
		t.Fatalf("nether brick fence collision boxes = %+v, ok=%t; want post and two connected rails", boxes, ok)
	}
}
