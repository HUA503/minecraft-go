package inventory

import "testing"

func TestJava263WoodFenceGatesCatalogRecipesAndStates(t *testing.T) {
	definitions := ModernWoodFenceGateItemDefinitions()
	gateByItem := make(map[int32]woodFenceGateDefinition, len(woodFenceGateDefinitions))
	for _, gate := range woodFenceGateDefinitions {
		gateByItem[gate.ItemID] = gate
	}
	if len(definitions) != WoodFenceGateCount {
		t.Fatalf("wood fence gate definitions = %d, want %d", len(definitions), WoodFenceGateCount)
	}
	catalog := CreativeCatalog(CreativeAll)
	byName := make(map[string]CreativeItem, len(catalog))
	for _, item := range catalog {
		byName[creativeItemRegistryName(item.ID)] = item
	}
	recipes := make(map[int32]Recipe)
	for _, recipe := range NewCraftingManager().Recipes {
		recipes[recipe.Output.ItemID] = recipe
	}
	creativeOrder := make(map[string]bool, len(vanillaCreativeTabOrder[CreativeBlocks]))
	for _, name := range vanillaCreativeTabOrder[CreativeBlocks] {
		creativeOrder[name] = true
	}
	seenStates := make(map[int32]bool)
	for _, gate := range definitions {
		spec := gateByItem[gate.RuntimeID]
		if !creativeOrder[gate.Name] {
			t.Errorf("%s missing from the Java 26.3 functional tab", gate.Name)
		}
		if item, ok := byName[gate.Name]; !ok || item.ID != gate.RuntimeID || item.Category != CreativeBlocks {
			t.Errorf("creative entry for %s = %+v, found=%v", gate.Name, item, ok)
		}
		definition, ok := ExtendedBlockByID(gate.RuntimeID)
		if !ok || definition.Name != gate.Name || definition.Hardness != 2 || definition.HarvestTool != ToolAxe {
			t.Errorf("block definition for %s = %+v, found=%v", gate.Name, definition, ok)
		}
		source, ok := byName[spec.Source]
		if !ok {
			t.Errorf("fence gate %s has missing plank source %s", gate.Name, spec.Source)
			continue
		}
		recipe, ok := recipes[gate.RuntimeID]
		if !ok || recipe.Width != 3 || recipe.Height != 2 || recipe.Output.Count != 1 || len(recipe.Pattern) != 6 || recipe.Pattern[0] != ItemStick || recipe.Pattern[1] != source.ID || recipe.Pattern[2] != ItemStick {
			t.Errorf("fence gate recipe for %s = %+v", gate.Name, recipe)
		}
		for facing := 0; facing < 4; facing++ {
			for _, open := range []bool{false, true} {
				for _, inWall := range []bool{false, true} {
					state := FenceGateBlockState(gate.RuntimeID, facing, open, inWall)
					itemID, gotFacing, gotOpen, gotInWall, valid := FenceGateStateProperties(state)
					if !valid || itemID != gate.RuntimeID || gotFacing != facing || gotOpen != open || gotInWall != inWall || BlockItemForState(state) != gate.RuntimeID {
						t.Errorf("state round trip %s facing=%d open=%v wall=%v -> %d,%d,%v,%v,%v", gate.Name, facing, open, inWall, itemID, gotFacing, gotOpen, gotInWall, valid)
					}
					if seenStates[state] {
						t.Errorf("duplicate fence gate state ID %d", state)
					}
					seenStates[state] = true
					if boxes, ok := BlockCollisionBoxes(state); !ok || len(boxes) != 4 {
						t.Errorf("fence gate collision/model boxes for state %d = %d, ok=%v", state, len(boxes), ok)
					} else {
						for _, box := range boxes {
							if box.MinY < 0 {
								t.Errorf("fence gate state %d extends below its block: %+v", state, box)
							}
						}
					}
				}
			}
		}
	}
	if all := CreativeCatalog(CreativeAll); len(all) != 913 {
		t.Fatalf("catalog count = %d after 13 fence gates, want 913", len(all))
	}
}
