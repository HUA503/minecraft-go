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
