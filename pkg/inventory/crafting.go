package inventory

// Recipe defines a craftable output from an input grid.
type Recipe struct {
	Width   int
	Height  int
	Pattern []int32
	Output  ItemStack
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
}

// MatchCrafting finds matching recipe for grid (gridSize = 2 for 2x2, 3 for 3x3).
func (cm *CraftingManager) MatchCrafting(grid []int32, gridSize int) (ItemStack, bool) {
	for _, recipe := range cm.Recipes {
		if recipe.Width > gridSize || recipe.Height > gridSize {
			continue
		}

		// Try matching recipe at all (ox, oy) offsets in grid
		maxOX := gridSize - recipe.Width
		maxOY := gridSize - recipe.Height

		for ox := 0; ox <= maxOX; ox++ {
			for oy := 0; oy <= maxOY; oy++ {
				if matchAtOffset(grid, gridSize, recipe, ox, oy) {
					return recipe.Output, true
				}
			}
		}
	}
	return ItemStack{}, false
}

func matchAtOffset(grid []int32, gridSize int, r Recipe, ox, oy int) bool {
	for gy := 0; gy < gridSize; gy++ {
		for gx := 0; gx < gridSize; gx++ {
			expectedItem := ItemAir
			rx := gx - ox
			ry := gy - oy

			if rx >= 0 && rx < r.Width && ry >= 0 && ry < r.Height {
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
