package inventory

// Recipe defines a craftable output from an input grid.
type Recipe struct {
	Width     int
	Height    int
	Pattern   []int32
	Shapeless bool
	Output    ItemStack
}

// CraftingManager holds known recipes and matches crafting grids.
type CraftingManager struct {
	Recipes []Recipe
}

func NewCraftingManager() *CraftingManager {
	cm := &CraftingManager{}
	cm.registerDefaults()
	return cm
}

func (cm *CraftingManager) registerDefaults() {
	// Bone -> bone meal.
	cm.Recipes = append(cm.Recipes, Recipe{
		Width: 1, Height: 1,
		Pattern: []int32{ItemBone},
		Output:  ItemStack{ItemID: ItemBoneMeal, Count: 3},
	})

	// 1. Oak Log -> 4 Oak Planks (1x1)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:   1,
		Height:  1,
		Pattern: []int32{ItemOakLog},
		Output:  ItemStack{ItemID: ItemOakPlanks, Count: 4},
	})

	// 2. 2 Oak Planks (vertical 1x2) -> 4 Sticks
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:   1,
		Height:  2,
		Pattern: []int32{ItemOakPlanks, ItemOakPlanks},
		Output:  ItemStack{ItemID: ItemStick, Count: 4},
	})

	// 3. 4 Oak Planks (2x2) -> 1 Crafting Table
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  2,
		Height: 2,
		Pattern: []int32{
			ItemOakPlanks, ItemOakPlanks,
			ItemOakPlanks, ItemOakPlanks,
		},
		Output: ItemStack{ItemID: ItemCraftingTable, Count: 1},
	})

	// 4. Wooden Pickaxe (3x3)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemOakPlanks, ItemOakPlanks, ItemOakPlanks,
			ItemAir, ItemStick, ItemAir,
			ItemAir, ItemStick, ItemAir,
		},
		Output: ItemStack{ItemID: ItemWoodenPickaxe, Count: 1},
	})

	// 5. Stone Pickaxe (3x3)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemCobblestone, ItemCobblestone, ItemCobblestone,
			ItemAir, ItemStick, ItemAir,
			ItemAir, ItemStick, ItemAir,
		},
		Output: ItemStack{ItemID: ItemStonePickaxe, Count: 1},
	})
	cm.Recipes = append(cm.Recipes, Recipe{
		Width: 3, Height: 3,
		Pattern: []int32{ItemCopperIngot, ItemCopperIngot, ItemCopperIngot, ItemAir, ItemStick, ItemAir, ItemAir, ItemStick, ItemAir},
		Output:  ItemStack{ItemID: ItemCopperPickaxe, Count: 1},
	})

	// 6. Diamond Pickaxe (3x3)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemDiamond, ItemDiamond, ItemDiamond,
			ItemAir, ItemStick, ItemAir,
			ItemAir, ItemStick, ItemAir,
		},
		Output: ItemStack{ItemID: ItemDiamondPickaxe, Count: 1},
	})

	// 7. Furnace (3x3)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemCobblestone, ItemCobblestone, ItemCobblestone,
			ItemCobblestone, ItemAir, ItemCobblestone,
			ItemCobblestone, ItemCobblestone, ItemCobblestone,
		},
		Output: ItemStack{ItemID: ItemFurnace, Count: 1},
	})

	// 8. Torches (1 Coal over 1 Stick, 1x2)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:   1,
		Height:  2,
		Pattern: []int32{ItemCoal, ItemStick},
		Output:  ItemStack{ItemID: ItemTorch, Count: 4},
	})

	// 9. Swords (1x3: 2 material over 1 stick)
	cm.Recipes = append(cm.Recipes,
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemOakPlanks, ItemOakPlanks, ItemStick},
			Output:  ItemStack{ItemID: ItemWoodenSword, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemCobblestone, ItemCobblestone, ItemStick},
			Output:  ItemStack{ItemID: ItemStoneSword, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemCopperIngot, ItemCopperIngot, ItemStick},
			Output:  ItemStack{ItemID: ItemCopperSword, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemIronIngot, ItemIronIngot, ItemStick},
			Output:  ItemStack{ItemID: ItemIronSword, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemDiamond, ItemDiamond, ItemStick},
			Output:  ItemStack{ItemID: ItemDiamondSword, Count: 1},
		},
	)

	// 10. Shovels (1x3: 1 material over 2 sticks)
	cm.Recipes = append(cm.Recipes,
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemOakPlanks, ItemStick, ItemStick},
			Output:  ItemStack{ItemID: ItemWoodenShovel, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemCobblestone, ItemStick, ItemStick},
			Output:  ItemStack{ItemID: ItemStoneShovel, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemCopperIngot, ItemStick, ItemStick},
			Output:  ItemStack{ItemID: ItemCopperShovel, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemIronIngot, ItemStick, ItemStick},
			Output:  ItemStack{ItemID: ItemIronShovel, Count: 1},
		},
		Recipe{
			Width:   1,
			Height:  3,
			Pattern: []int32{ItemDiamond, ItemStick, ItemStick},
			Output:  ItemStack{ItemID: ItemDiamondShovel, Count: 1},
		},
	)

	// 11. Axes (2x3: 3 material, 2 sticks)
	cm.Recipes = append(cm.Recipes,
		Recipe{
			Width:  2,
			Height: 3,
			Pattern: []int32{
				ItemOakPlanks, ItemOakPlanks,
				ItemOakPlanks, ItemStick,
				ItemAir, ItemStick,
			},
			Output: ItemStack{ItemID: ItemWoodenAxe, Count: 1},
		},
		Recipe{
			Width:  2,
			Height: 3,
			Pattern: []int32{
				ItemCobblestone, ItemCobblestone,
				ItemCobblestone, ItemStick,
				ItemAir, ItemStick,
			},
			Output: ItemStack{ItemID: ItemStoneAxe, Count: 1},
		},
		Recipe{
			Width:  2,
			Height: 3,
			Pattern: []int32{
				ItemCopperIngot, ItemCopperIngot,
				ItemCopperIngot, ItemStick,
				ItemAir, ItemStick,
			},
			Output: ItemStack{ItemID: ItemCopperAxe, Count: 1},
		},
		Recipe{
			Width:  2,
			Height: 3,
			Pattern: []int32{
				ItemIronIngot, ItemIronIngot,
				ItemIronIngot, ItemStick,
				ItemAir, ItemStick,
			},
			Output: ItemStack{ItemID: ItemIronAxe, Count: 1},
		},
		Recipe{
			Width:  2,
			Height: 3,
			Pattern: []int32{
				ItemDiamond, ItemDiamond,
				ItemDiamond, ItemStick,
				ItemAir, ItemStick,
			},
			Output: ItemStack{ItemID: ItemDiamondAxe, Count: 1},
		},
	)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width: 3, Height: 3,
		Pattern: []int32{ItemCopperIngot, ItemCopperIngot, ItemAir, ItemAir, ItemStick, ItemAir, ItemAir, ItemStick, ItemAir},
		Output:  ItemStack{ItemID: ItemCopperHoe, Count: 1},
	})

	// 12. Iron Pickaxe (3x3)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemIronIngot, ItemIronIngot, ItemIronIngot,
			ItemAir, ItemStick, ItemAir,
			ItemAir, ItemStick, ItemAir,
		},
		Output: ItemStack{ItemID: ItemIronPickaxe, Count: 1},
	})

	// 13. Blocks (3x3: 9 ingots / gems)
	cm.Recipes = append(cm.Recipes,
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
			},
			Output: ItemStack{ItemID: ItemIronBlock, Count: 1},
		},
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemGoldIngot, ItemGoldIngot, ItemGoldIngot,
				ItemGoldIngot, ItemGoldIngot, ItemGoldIngot,
				ItemGoldIngot, ItemGoldIngot, ItemGoldIngot,
			},
			Output: ItemStack{ItemID: ItemGoldBlock, Count: 1},
		},
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemDiamond, ItemDiamond, ItemDiamond,
				ItemDiamond, ItemDiamond, ItemDiamond,
				ItemDiamond, ItemDiamond, ItemDiamond,
			},
			Output: ItemStack{ItemID: ItemDiamondBlock, Count: 1},
		},
	)

	// 14. Chest (3x3: 8 planks hollow)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemOakPlanks, ItemOakPlanks, ItemOakPlanks,
			ItemOakPlanks, ItemAir, ItemOakPlanks,
			ItemOakPlanks, ItemOakPlanks, ItemOakPlanks,
		},
		Output: ItemStack{ItemID: ItemChest, Count: 1},
	})

	// 15. Torch (1x2 vertical: Coal over Stick)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:   1,
		Height:  2,
		Pattern: []int32{ItemCoal, ItemStick},
		Output:  ItemStack{ItemID: ItemTorch, Count: 4},
	})

	// 16. Bread (3x1: 3 Wheat)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:   3,
		Height:  1,
		Pattern: []int32{ItemWheat, ItemWheat, ItemWheat},
		Output:  ItemStack{ItemID: ItemBread, Count: 1},
	})

	// 17. Bucket (3x2: 3 Iron Ingots in V shape)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 2,
		Pattern: []int32{
			ItemIronIngot, ItemAir, ItemIronIngot,
			ItemAir, ItemIronIngot, ItemAir,
		},
		Output: ItemStack{ItemID: ItemBucket, Count: 1},
	})

	// 18. Shears (2x2 diagonal Iron Ingots)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  2,
		Height: 2,
		Pattern: []int32{
			ItemAir, ItemIronIngot,
			ItemIronIngot, ItemAir,
		},
		Output: ItemStack{ItemID: ItemShears, Count: 1},
	})

	// 19. Golden Apple (3x3: 8 Gold Ingots surrounding Apple)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemGoldIngot, ItemGoldIngot, ItemGoldIngot,
			ItemGoldIngot, ItemApple, ItemGoldIngot,
			ItemGoldIngot, ItemGoldIngot, ItemGoldIngot,
		},
		Output: ItemStack{ItemID: ItemGoldenApple, Count: 1},
	})

	// 20. Iron Armor Set
	cm.Recipes = append(cm.Recipes,
		// Iron Helmet
		Recipe{
			Width:  3,
			Height: 2,
			Pattern: []int32{
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
				ItemIronIngot, ItemAir, ItemIronIngot,
			},
			Output: ItemStack{ItemID: ItemIronHelmet, Count: 1},
		},
		// Iron Chestplate
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemIronIngot, ItemAir, ItemIronIngot,
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
			},
			Output: ItemStack{ItemID: ItemIronChestplate, Count: 1},
		},
		// Iron Leggings
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemIronIngot, ItemIronIngot, ItemIronIngot,
				ItemIronIngot, ItemAir, ItemIronIngot,
				ItemIronIngot, ItemAir, ItemIronIngot,
			},
			Output: ItemStack{ItemID: ItemIronLeggings, Count: 1},
		},
		// Iron Boots
		Recipe{
			Width:  3,
			Height: 2,
			Pattern: []int32{
				ItemIronIngot, ItemAir, ItemIronIngot,
				ItemIronIngot, ItemAir, ItemIronIngot,
			},
			Output: ItemStack{ItemID: ItemIronBoots, Count: 1},
		},
	)

	// 21. Diamond Armor Set
	cm.Recipes = append(cm.Recipes,
		// Diamond Helmet
		Recipe{
			Width:  3,
			Height: 2,
			Pattern: []int32{
				ItemDiamond, ItemDiamond, ItemDiamond,
				ItemDiamond, ItemAir, ItemDiamond,
			},
			Output: ItemStack{ItemID: ItemDiamondHelmet, Count: 1},
		},
		// Diamond Chestplate
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemDiamond, ItemAir, ItemDiamond,
				ItemDiamond, ItemDiamond, ItemDiamond,
				ItemDiamond, ItemDiamond, ItemDiamond,
			},
			Output: ItemStack{ItemID: ItemDiamondChestplate, Count: 1},
		},
		// Diamond Leggings
		Recipe{
			Width:  3,
			Height: 3,
			Pattern: []int32{
				ItemDiamond, ItemDiamond, ItemDiamond,
				ItemDiamond, ItemAir, ItemDiamond,
				ItemDiamond, ItemAir, ItemDiamond,
			},
			Output: ItemStack{ItemID: ItemDiamondLeggings, Count: 1},
		},
		// Diamond Boots
		Recipe{
			Width:  3,
			Height: 2,
			Pattern: []int32{
				ItemDiamond, ItemAir, ItemDiamond,
				ItemDiamond, ItemAir, ItemDiamond,
			},
			Output: ItemStack{ItemID: ItemDiamondBoots, Count: 1},
		},
	)

	// 22. Wooden Slab & Stone Slab
	cm.Recipes = append(cm.Recipes,
		Recipe{
			Width:   3,
			Height:  1,
			Pattern: []int32{ItemOakPlanks, ItemOakPlanks, ItemOakPlanks},
			Output:  ItemStack{ItemID: ItemOakSlab, Count: 6},
		},
		Recipe{
			Width:   3,
			Height:  1,
			Pattern: []int32{ItemStone, ItemStone, ItemStone},
			Output:  ItemStack{ItemID: ItemStoneSlab, Count: 6},
		},
	)
	for _, slab := range woodSlabDefinitions {
		planks, ok := ExtendedBlockByName(slab.PlankBlock)
		if !ok {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width:   3,
			Height:  1,
			Pattern: []int32{planks.RuntimeID, planks.RuntimeID, planks.RuntimeID},
			Output:  ItemStack{ItemID: slab.ItemID, Count: 6},
		})
	}
	for _, slab := range colorSlabDefinitions {
		var blockID int32
		for _, item := range CreativeCatalog(CreativeAll) {
			if creativeItemRegistryName(item.ID) == slab.BlockName {
				blockID = item.ID
				break
			}
		}
		if blockID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 1,
			Pattern: []int32{blockID, blockID, blockID},
			Output:  ItemStack{ItemID: slab.ItemID, Count: 6},
		})
	}
	for i, color := range cushionColors {
		woolSlabName := "minecraft:" + color.name + "_wool_slab"
		var woolSlabID int32
		for _, slab := range colorSlabDefinitions {
			if slab.Name == woolSlabName {
				woolSlabID = slab.ItemID
				break
			}
		}
		if woolSlabID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 1,
			Pattern: []int32{woolSlabID, woolSlabID, woolSlabID},
			Output:  ItemStack{ItemID: FirstCushionBlock + int32(i), Count: 1},
		})
	}
	for _, slab := range masonrySlabDefinitions {
		var blockID int32
		for _, item := range CreativeCatalog(CreativeAll) {
			if creativeItemRegistryName(item.ID) == slab.BlockName {
				blockID = item.ID
				break
			}
		}
		if blockID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 1,
			Pattern: []int32{blockID, blockID, blockID},
			Output:  ItemStack{ItemID: slab.ItemID, Count: 6},
		})
	}
	for _, stair := range colorStairDefinitions {
		block, ok := ExtendedBlockByName(stair.SourceBlock)
		if !ok {
			for _, item := range CreativeCatalog(CreativeAll) {
				if creativeItemRegistryName(item.ID) == stair.SourceBlock {
					block = ExtendedBlockDefinition{RuntimeID: item.ID}
					ok = true
					break
				}
			}
		}
		if !ok {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 3,
			Pattern: []int32{
				block.RuntimeID, ItemAir, ItemAir,
				block.RuntimeID, block.RuntimeID, ItemAir,
				block.RuntimeID, block.RuntimeID, block.RuntimeID,
			},
			Output: ItemStack{ItemID: stair.ItemID, Count: 4},
		})
	}
	for _, stair := range woodStairDefinitions {
		var blockID int32
		for _, item := range CreativeCatalog(CreativeAll) {
			if creativeItemRegistryName(item.ID) == stair.SourceBlock {
				blockID = item.ID
				break
			}
		}
		if blockID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 3,
			Pattern: []int32{
				blockID, ItemAir, ItemAir,
				blockID, blockID, ItemAir,
				blockID, blockID, blockID,
			},
			Output: ItemStack{ItemID: stair.ItemID, Count: 4},
		})
	}
	for _, stair := range masonryStairDefinitions {
		var blockID int32
		if block, ok := ExtendedBlockByName(stair.SourceBlock); ok {
			blockID = block.RuntimeID
		} else {
			for _, item := range CreativeCatalog(CreativeAll) {
				if creativeItemRegistryName(item.ID) == stair.SourceBlock {
					blockID = item.ID
					break
				}
			}
		}
		if blockID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 3,
			Pattern: []int32{
				blockID, ItemAir, ItemAir,
				blockID, blockID, ItemAir,
				blockID, blockID, blockID,
			},
			Output: ItemStack{ItemID: stair.ItemID, Count: 4},
		})
	}
	for _, fence := range woodFenceDefinitions {
		var planksID int32
		if block, ok := ExtendedBlockByName(fence.Source); ok {
			planksID = block.RuntimeID
		} else {
			for _, item := range CreativeCatalog(CreativeAll) {
				if creativeItemRegistryName(item.ID) == fence.Source {
					planksID = item.ID
					break
				}
			}
		}
		if planksID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 2,
			Pattern: []int32{planksID, ItemStick, planksID, planksID, ItemStick, planksID},
			Output:  ItemStack{ItemID: fence.ItemID, Count: 3},
		})
	}
	for _, wall := range masonryWallDefinitions {
		var sourceID int32
		for _, item := range CreativeCatalog(CreativeAll) {
			if creativeItemRegistryName(item.ID) == wall.Source {
				sourceID = item.ID
				break
			}
		}
		if sourceID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 2,
			Pattern: []int32{sourceID, sourceID, sourceID, sourceID, sourceID, sourceID},
			Output:  ItemStack{ItemID: wall.ItemID, Count: 6},
		})
	}

	// 23. Ladder (7 Sticks in H pattern)
	cm.Recipes = append(cm.Recipes, Recipe{
		Width:  3,
		Height: 3,
		Pattern: []int32{
			ItemStick, ItemAir, ItemStick,
			ItemStick, ItemStick, ItemStick,
			ItemStick, ItemAir, ItemStick,
		},
		Output: ItemStack{ItemID: ItemLadder, Count: 3},
	})

	// Modern material conversions supported by the vanilla crafting grid.
	cm.Recipes = append(cm.Recipes,
		Recipe{Width: 1, Height: 1, Pattern: []int32{ItemBlazeRod}, Output: ItemStack{ItemID: ItemBlazePowder, Count: 2}},
		Recipe{Width: 1, Height: 1, Pattern: []int32{ItemCopperIngot}, Output: ItemStack{ItemID: ItemCopperNugget, Count: 9}},
		Recipe{Width: 3, Height: 3, Pattern: []int32{ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget, ItemCopperNugget}, Output: ItemStack{ItemID: ItemCopperIngot, Count: 1}},
		Recipe{Width: 1, Height: 1, Pattern: []int32{ItemIronIngot}, Output: ItemStack{ItemID: ItemIronNugget, Count: 9}},
		Recipe{Width: 3, Height: 3, Pattern: []int32{ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget, ItemIronNugget}, Output: ItemStack{ItemID: ItemIronIngot, Count: 1}},
		Recipe{Width: 1, Height: 2, Pattern: []int32{ItemCharcoal, ItemStick}, Output: ItemStack{ItemID: ItemTorch, Count: 4}},
		Recipe{Width: 1, Height: 1, Pattern: []int32{ItemSugarCane}, Output: ItemStack{ItemID: ItemSugar, Count: 1}},
		Recipe{Width: 3, Height: 1, Pattern: []int32{ItemSugarCane, ItemSugarCane, ItemSugarCane}, Output: ItemStack{ItemID: ItemPaper, Count: 3}},
		Recipe{Width: 1, Height: 1, Pattern: []int32{ItemGlowstoneBlock}, Output: ItemStack{ItemID: ItemGlowstoneDust, Count: 4}},
		Recipe{Width: 2, Height: 2, Pattern: []int32{ItemGlowstoneDust, ItemGlowstoneDust, ItemGlowstoneDust, ItemGlowstoneDust}, Output: ItemStack{ItemID: ItemGlowstoneBlock, Count: 1}},
		Recipe{Width: 1, Height: 1, Pattern: []int32{ItemSlimeBlock}, Output: ItemStack{ItemID: ItemSlimeBall, Count: 9}},
		Recipe{Width: 3, Height: 3, Pattern: []int32{ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall, ItemSlimeBall}, Output: ItemStack{ItemID: ItemSlimeBlock, Count: 1}},
	)
	if amethystBlock, ok := ExtendedBlockByName("minecraft:amethyst_block"); ok {
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 2, Height: 2,
			Pattern: []int32{ItemAmethystShard, ItemAmethystShard, ItemAmethystShard, ItemAmethystShard},
			Output:  ItemStack{ItemID: amethystBlock.RuntimeID, Count: 1},
		})
	}
	if honeycombBlock, ok := ExtendedBlockByName("minecraft:honeycomb_block"); ok {
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 2, Height: 2,
			Pattern: []int32{ItemHoneycomb, ItemHoneycomb, ItemHoneycomb, ItemHoneycomb},
			Output:  ItemStack{ItemID: honeycombBlock.RuntimeID, Count: 1},
		})
	}
	if resinBricks, ok := ExtendedBlockByName("minecraft:resin_bricks"); ok {
		cm.Recipes = append(cm.Recipes,
			Recipe{Width: 2, Height: 2, Pattern: []int32{ItemResinBrick, ItemResinBrick, ItemResinBrick, ItemResinBrick}, Output: ItemStack{ItemID: resinBricks.RuntimeID, Count: 1}},
			Recipe{Width: 1, Height: 1, Pattern: []int32{resinBricks.RuntimeID}, Output: ItemStack{ItemID: ItemResinBrick, Count: 4}},
		)
	}
	cm.Recipes = append(cm.Recipes, Recipe{
		Width: 3, Height: 3,
		Pattern: []int32{ItemGoldNugget, ItemGoldNugget, ItemGoldNugget, ItemGoldNugget, ItemMelonSlice, ItemGoldNugget, ItemGoldNugget, ItemGoldNugget, ItemGoldNugget},
		Output:  ItemStack{ItemID: ItemGlisteringMelon, Count: 1},
	})
	cm.registerDyeRecipes()
}

func (cm *CraftingManager) registerDyeRecipes() {
	colors := []struct {
		name  string
		dyeID int32
	}{
		{"white", ItemWhiteDye}, {"orange", ItemOrangeDye}, {"magenta", ItemMagentaDye}, {"light_blue", ItemLightBlueDye},
		{"yellow", ItemYellowDye}, {"lime", ItemLimeDye}, {"pink", ItemPinkDye}, {"gray", ItemGrayDye},
		{"light_gray", ItemLightGrayDye}, {"cyan", ItemCyanDye}, {"purple", ItemPurpleDye}, {"blue", ItemBlueDye},
		{"brown", ItemBrownDye}, {"green", ItemGreenDye}, {"red", ItemRedDye}, {"black", ItemBlackDye},
	}
	if candle, ok := ExtendedBlockByName("minecraft:candle"); ok {
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 1, Height: 2,
			Pattern: []int32{ItemString, ItemHoneycomb},
			Output:  ItemStack{ItemID: candle.RuntimeID, Count: 1},
		})
	}
	for _, color := range colors {
		if candle, ok := ExtendedBlockByName("minecraft:" + color.name + "_candle"); ok {
			cm.Recipes = append(cm.Recipes, Recipe{
				Shapeless: true,
				Pattern:   []int32{17133, color.dyeID},
				Output:    ItemStack{ItemID: candle.RuntimeID, Count: 1},
			})
		}
	}
	if pane, ok := ExtendedBlockByName("minecraft:glass_pane"); ok {
		cm.Recipes = append(cm.Recipes, Recipe{
			Width: 3, Height: 2,
			Pattern: []int32{20, 20, 20, 20, 20, 20},
			Output:  ItemStack{ItemID: pane.RuntimeID, Count: 16},
		})
	}
	for _, color := range colors {
		if stainedGlass, ok := ExtendedBlockByName("minecraft:" + color.name + "_stained_glass"); ok {
			cm.Recipes = append(cm.Recipes, Recipe{
				Width: 3, Height: 3,
				Pattern: []int32{
					20, 20, 20,
					20, color.dyeID, 20,
					20, 20, 20,
				},
				Output: ItemStack{ItemID: stainedGlass.RuntimeID, Count: 8},
			})
		}
		if pane, ok := ExtendedBlockByName("minecraft:" + color.name + "_stained_glass_pane"); ok {
			glassID := int32(20)
			if stainedGlass, exists := ExtendedBlockByName("minecraft:" + color.name + "_stained_glass"); exists {
				glassID = stainedGlass.RuntimeID
			}
			cm.Recipes = append(cm.Recipes, Recipe{
				Width: 3, Height: 2,
				Pattern: []int32{glassID, glassID, glassID, glassID, glassID, glassID},
				Output:  ItemStack{ItemID: pane.RuntimeID, Count: 16},
			})
		}
		if color.name == "white" {
			continue // White dye cannot recolor white wool to a different item.
		}
		woolID := int32(0)
		if wool, ok := ExtendedBlockByName("minecraft:" + color.name + "_wool"); ok {
			woolID = wool.RuntimeID
		}
		if woolID == 0 {
			continue
		}
		cm.Recipes = append(cm.Recipes, Recipe{
			Pattern:   []int32{35, color.dyeID},
			Shapeless: true,
			Output:    ItemStack{ItemID: woolID, Count: 1},
		})
	}
}

// MatchCrafting finds matching recipe for grid (gridSize = 2 for 2x2, 3 for 3x3).
func (cm *CraftingManager) MatchCrafting(grid []int32, gridSize int) (ItemStack, bool) {
	if cm == nil || (gridSize != 2 && gridSize != 3) || len(grid) != gridSize*gridSize {
		return ItemStack{}, false
	}
	for _, recipe := range cm.Recipes {
		if recipe.Shapeless {
			if matchShapeless(grid, recipe.Pattern) {
				return recipe.Output, true
			}
			continue
		}
		if recipe.Width > gridSize || recipe.Height > gridSize {
			continue
		}

		// Try matching recipe at all (ox, oy) offsets in grid
		maxOX := gridSize - recipe.Width
		maxOY := gridSize - recipe.Height

		for ox := 0; ox <= maxOX; ox++ {
			for oy := 0; oy <= maxOY; oy++ {
				if matchAtOffset(grid, gridSize, recipe, ox, oy, false) || matchAtOffset(grid, gridSize, recipe, ox, oy, true) {
					return recipe.Output, true
				}
			}
		}
	}
	return ItemStack{}, false
}

func matchShapeless(grid, inputs []int32) bool {
	remaining := make(map[int32]int, len(inputs))
	for _, input := range inputs {
		if input != ItemAir {
			remaining[input]++
		}
	}
	for _, item := range grid {
		if item == ItemAir {
			continue
		}
		if remaining[item] == 0 {
			return false
		}
		remaining[item]--
	}
	for _, count := range remaining {
		if count != 0 {
			return false
		}
	}
	return len(remaining) != 0
}

func matchAtOffset(grid []int32, gridSize int, r Recipe, ox, oy int, mirrored bool) bool {
	for gy := 0; gy < gridSize; gy++ {
		for gx := 0; gx < gridSize; gx++ {
			expectedItem := ItemAir
			rx := gx - ox
			ry := gy - oy

			if rx >= 0 && rx < r.Width && ry >= 0 && ry < r.Height {
				if mirrored {
					rx = r.Width - 1 - rx
				}
				expectedItem = r.Pattern[ry*r.Width+rx]
			}

			gridItem := grid[gy*gridSize+gx]
			if gridItem != expectedItem {
				return false
			}
		}
	}
	return true
}
