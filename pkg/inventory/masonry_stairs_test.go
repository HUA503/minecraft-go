package inventory

import "testing"

func TestJava263MasonryStairsCatalogRecipesAndStates(t *testing.T) {
	if got := len(masonryStairDefinitions); got != 43 {
		t.Fatalf("masonry stair definitions = %d, want 43", got)
	}
	all := CreativeCatalog(CreativeAll)
	byName := make(map[string]CreativeItem, len(all))
	for _, item := range all {
		byName[creativeItemRegistryName(item.ID)] = item
	}
	recipes := make(map[int32]Recipe)
	for _, recipe := range NewCraftingManager().Recipes {
		if recipe.Output.Count > 0 {
			recipes[recipe.Output.ItemID] = recipe
		}
	}
	vanilla := make(map[string]bool, len(vanillaCreativeTabOrder[CreativeBlocks]))
	for _, name := range vanillaCreativeTabOrder[CreativeBlocks] {
		vanilla[name] = true
	}
	for _, stair := range masonryStairDefinitions {
		if !vanilla[stair.Name] {
			t.Errorf("masonry stair %s is not present in the Java 26.3 building tab order", stair.Name)
		}
		if got, ok := byName[stair.Name]; !ok || got.ID != stair.ItemID || got.Category != CreativeBlocks {
			t.Errorf("creative entry for %s = %+v, found=%v", stair.Name, got, ok)
		}
		block, ok := ExtendedBlockByID(stair.ItemID)
		if !ok || block.Name != stair.Name || block.TextureTop != stair.TextureTop || block.TextureSide != stair.TextureSide || !block.RequiresHarvestTool || block.HarvestTool != ToolPickaxe {
			t.Errorf("block definition for %s is incomplete: %+v", stair.Name, block)
		}
		source, ok := byName[stair.SourceBlock]
		if !ok {
			t.Errorf("stair %s has unregistered source %s", stair.Name, stair.SourceBlock)
			continue
		}
		recipe, ok := recipes[stair.ItemID]
		if !ok || recipe.Width != 3 || recipe.Height != 3 || recipe.Output.Count != 4 || len(recipe.Pattern) != 9 || recipe.Pattern[0] != source.ID || recipe.Pattern[3] != source.ID || recipe.Pattern[4] != source.ID || recipe.Pattern[6] != source.ID || recipe.Pattern[7] != source.ID || recipe.Pattern[8] != source.ID {
			t.Errorf("stair recipe missing or incorrect for %s (source=%d): %+v", stair.Name, source.ID, recipe)
		}
		for facing := StairFacing(0); facing < 4; facing++ {
			for _, upsideDown := range []bool{false, true} {
				state, ok := StairBlockState(stair.ItemID, facing, upsideDown)
				if !ok {
					t.Fatalf("failed to encode %s facing=%d upsideDown=%v", stair.Name, facing, upsideDown)
				}
				itemID, gotFacing, gotUpsideDown, ok := StairStateProperties(state)
				if !ok || itemID != stair.ItemID || gotFacing != facing || gotUpsideDown != upsideDown || BlockItemForState(state) != stair.ItemID {
					t.Errorf("state round trip for %s = item %d facing %d upper %v ok %v", stair.Name, itemID, gotFacing, gotUpsideDown, ok)
				}
				if boxes, ok := BlockCollisionBoxes(state); !ok || len(boxes) != 2 {
					t.Errorf("collision shape for %s state %d = %v, ok=%v", stair.Name, state, boxes, ok)
				}
			}
		}
	}
}
