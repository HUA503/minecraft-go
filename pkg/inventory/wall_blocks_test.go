package inventory

import "testing"

func TestJava263MasonryWallsAreCataloguedCraftableAndShaped(t *testing.T) {
	all := CreativeCatalog(CreativeAll)
	if len(all) != 969 {
		t.Fatalf("all-items search catalog length = %d, want 969", len(all))
	}
	for _, wall := range masonryWallDefinitions {
		if !IsMasonryWallBlock(wall.ItemID) {
			t.Errorf("wall %s is outside wall block range", wall.Name)
		}
		if _, ok := ExtendedBlockByName(wall.Name); !ok {
			t.Errorf("wall %s missing from block registry", wall.Name)
		}
		if matches := SearchCreativeCatalog(CreativeBlocks, wall.Name); len(matches) != 1 || matches[0].ID != wall.ItemID {
			t.Errorf("wall %s missing from building tab search: %+v", wall.Name, matches)
		}
		var sourceID int32
		for _, item := range all {
			if creativeItemRegistryName(item.ID) == wall.Source {
				sourceID = item.ID
				break
			}
		}
		if sourceID == 0 {
			t.Errorf("wall %s has no catalog source block %s", wall.Name, wall.Source)
			continue
		}
		grid := []int32{sourceID, sourceID, sourceID, sourceID, sourceID, sourceID, 0, 0, 0}
		if crafted, ok := NewCraftingManager().MatchCrafting(grid, 3); !ok || crafted.ItemID != wall.ItemID || crafted.Count != 6 {
			t.Errorf("wall recipe %s = %+v, matched=%t", wall.Name, crafted, ok)
		}
	}
	neighbors := map[[3]int32]int32{{0, 0, -1}: ItemOakPlanks, {1, 0, 0}: ItemOakPlanks, {0, 0, 1}: ItemOakPlanks, {-1, 0, 0}: ItemOakPlanks}
	boxes, ok := MasonryWallCollisionBoxesAt(FirstMasonryWallItem, 0, 0, 0, func(x, y, z int32) int32 {
		return neighbors[[3]int32{x, y, z}]
	})
	if !ok || len(boxes) != 9 {
		t.Fatalf("connected wall shape = %d boxes, ok=%t; want center post and eight rail sections", len(boxes), ok)
	}
	for _, box := range boxes {
		if box.MaxY > 1.5 {
			t.Fatalf("wall collision height = %v, exceeds vanilla wall height 1.5", box.MaxY)
		}
	}
	boxes, _ = MasonryWallCollisionBoxesAt(FirstMasonryWallItem, 0, 0, 0, func(x, y, z int32) int32 {
		if x == 1 && y == 0 && z == 0 {
			return FirstMasonryWallItem + 1
		}
		return 0
	})
	if len(boxes) != 3 {
		t.Fatalf("single adjacent wall shape = %d boxes, want post and two connected side sections", len(boxes))
	}
}
