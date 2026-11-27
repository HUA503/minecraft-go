package inventory

import "testing"

func TestJava263WoodDoorsAndTrapdoorsCatalogRecipesAndStates(t *testing.T) {
	doors := ModernWoodDoorItemDefinitions()
	trapdoors := ModernWoodTrapdoorItemDefinitions()
	if len(doors) != WoodDoorCount || len(trapdoors) != WoodTrapdoorCount {
		t.Fatalf("wood openables = %d doors, %d trapdoors; want %d each", len(doors), len(trapdoors), WoodDoorCount)
	}
	catalog := CreativeCatalog(CreativeAll)
	byName := make(map[string]CreativeItem, len(catalog))
	for _, item := range catalog {
		byName[creativeItemRegistryName(item.ID)] = item
	}
	creativeOrder := make(map[string]bool, len(vanillaCreativeTabOrder[CreativeBlocks]))
	for _, name := range vanillaCreativeTabOrder[CreativeBlocks] {
		creativeOrder[name] = true
	}
	recipes := make(map[int32]Recipe)
	for _, recipe := range NewCraftingManager().Recipes {
		recipes[recipe.Output.ItemID] = recipe
	}
	for _, group := range [][]ExtendedBlockDefinition{doors, trapdoors} {
		for _, item := range group {
			if !creativeOrder[item.Name] {
				t.Errorf("%s missing from the Java 26.3 building tab", item.Name)
			}
			if creative, ok := byName[item.Name]; !ok || creative.ID != item.RuntimeID || creative.Category != CreativeBlocks {
				t.Errorf("creative entry for %s = %+v, found=%v", item.Name, creative, ok)
			}
			if item.RuntimeID == ItemOakDoor {
				if !IsPlaceableBlock(item.RuntimeID) {
					t.Errorf("legacy oak door ID %d is not placeable", item.RuntimeID)
				}
			} else if registered, ok := ExtendedBlockByID(item.RuntimeID); !ok || registered.Name != item.Name {
				t.Errorf("extended block registration for %s = %+v, found=%v", item.Name, registered, ok)
			}
			if _, ok := recipes[item.RuntimeID]; !ok {
				t.Errorf("no crafting recipe for %s", item.Name)
			}
		}
	}
	seen := make(map[int32]bool)
	for _, door := range woodDoorDefinitions {
		for facing := 0; facing < 4; facing++ {
			for _, open := range []bool{false, true} {
				for _, hinge := range []bool{false, true} {
					for _, upper := range []bool{false, true} {
						state := WoodDoorBlockState(door.ItemID, facing, open, hinge, upper)
						id, gotFacing, gotOpen, gotHinge, gotUpper, ok := WoodDoorStateProperties(state)
						if !ok || id != door.ItemID || gotFacing != facing || gotOpen != open || gotHinge != hinge || gotUpper != upper || BlockItemForState(state) != door.ItemID {
							t.Errorf("door state round trip failed for %s state %d", door.Name, state)
						}
						if seen[state] {
							t.Errorf("duplicate openable state ID %d", state)
						}
						seen[state] = true
						if boxes, ok := WoodDoorModelBoxes(state); !ok || len(boxes) != 1 {
							t.Errorf("door model shape for %d = %v, ok=%v", state, boxes, ok)
						}
					}
				}
			}
		}
	}
	for _, trapdoor := range woodTrapdoorDefinitions {
		for facing := 0; facing < 4; facing++ {
			for _, open := range []bool{false, true} {
				for _, top := range []bool{false, true} {
					state := WoodTrapdoorBlockState(trapdoor.ItemID, facing, open, top)
					id, gotFacing, gotOpen, gotTop, ok := WoodTrapdoorStateProperties(state)
					if !ok || id != trapdoor.ItemID || gotFacing != facing || gotOpen != open || gotTop != top || BlockItemForState(state) != trapdoor.ItemID {
						t.Errorf("trapdoor state round trip failed for %s state %d", trapdoor.Name, state)
					}
					if seen[state] {
						t.Errorf("duplicate openable state ID %d", state)
					}
					seen[state] = true
					if boxes, ok := WoodTrapdoorModelBoxes(state); !ok || len(boxes) != 1 {
						t.Errorf("trapdoor model shape for %d = %v, ok=%v", state, boxes, ok)
					}
				}
			}
		}
	}
	if got := len(CreativeCatalog(CreativeAll)); got != 1009 {
		t.Fatalf("all-items catalog count = %d, want 1009", got)
	}
}

func TestJava263IronTrapdoorCatalogRecipeAndRedstoneStates(t *testing.T) {
	definition, ok := ExtendedBlockByID(ItemIronTrapdoor)
	if !ok || definition.Name != "minecraft:iron_trapdoor" || definition.Category != CreativeBlocks {
		t.Fatalf("iron trapdoor block definition = %+v, found=%v", definition, ok)
	}
	if got := CreativeCatalog(CreativeBlocks); !containsCreativeItem(got, ItemIronTrapdoor) {
		t.Fatal("iron trapdoor missing from Building Blocks creative tab")
	}
	recipeFound := false
	for _, recipe := range NewCraftingManager().Recipes {
		if recipe.Output.ItemID == ItemIronTrapdoor && recipe.Output.Count == 1 && recipe.Width == 2 && recipe.Height == 2 {
			recipeFound = true
			for _, input := range recipe.Pattern {
				if input != ItemIronIngot {
					t.Fatalf("iron trapdoor recipe contains input %d, want iron ingots", input)
				}
			}
		}
	}
	if !recipeFound {
		t.Fatal("missing 2x2 iron trapdoor recipe")
	}
	for facing := 0; facing < 4; facing++ {
		for _, open := range []bool{false, true} {
			for _, top := range []bool{false, true} {
				state := IronTrapdoorBlockState(facing, open, top)
				gotFacing, gotOpen, gotTop, valid := IronTrapdoorStateProperties(state)
				if !valid || gotFacing != facing || gotOpen != open || gotTop != top || BlockItemForState(state) != ItemIronTrapdoor {
					t.Errorf("iron trapdoor state round trip failed for state %d", state)
				}
				if boxes, valid := WoodTrapdoorModelBoxes(state); !valid || len(boxes) != 1 {
					t.Errorf("iron trapdoor model shape for state %d = %v, valid=%v", state, boxes, valid)
				}
			}
		}
	}
}

func containsCreativeItem(items []CreativeItem, id int32) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
