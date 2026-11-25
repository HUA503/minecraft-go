package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestRenderIsometricBlock16x16(t *testing.T) {
	blocks := []int32{
		inventory.ItemGrass,
		inventory.ItemStone,
		inventory.ItemCobblestone,
		inventory.ItemDirt,
		inventory.ItemOakPlanks,
		inventory.ItemOakLog,
		inventory.ItemCraftingTable,
		inventory.ItemFurnace,
		56, // Diamond Ore
	}

	for _, b := range blocks {
		sprite := RenderIsometricBlock16x16(b)
		nonZeroCount := 0
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if sprite[y][x] != 0 {
					nonZeroCount++
				}
			}
		}
		// A 16x16 isometric voxel cube occupies ~120 to 180 pixels inside the 256 pixel grid
		if nonZeroCount < 100 || nonZeroCount > 230 {
			t.Fatalf("block %d produced abnormal pixel count %d in 16x16 isometric cube", b, nonZeroCount)
		}
	}
}

func TestModernCreativeMaterialItemsUseOfficialIcons(t *testing.T) {
	for _, item := range inventory.ModernCreativeItemDefinitions() {
		t.Run(item.Name, func(t *testing.T) {
			if got := GetOfficialItemTextureName(item.RuntimeID); got != item.Texture {
				t.Fatalf("official texture = %q, want %q", got, item.Texture)
			}
			if _, ok := GetOfficialItemSprite(item.RuntimeID); !ok {
				t.Fatalf("official sprite %q is missing", item.Texture)
			}
			sprite := GetItemSprite16x16(item.RuntimeID)
			nonTransparent := 0
			for y := range sprite {
				for x := range sprite[y] {
					if sprite[y][x]>>24 != 0 {
						nonTransparent++
					}
				}
			}
			if nonTransparent == 0 {
				t.Fatal("generated item icon is empty")
			}
		})
	}
}

func TestGetItemSprite16x16(t *testing.T) {
	items := []int32{
		inventory.ItemWoodenPickaxe,
		inventory.ItemDiamondPickaxe,
		268, // Wooden Sword
		276, // Diamond Sword
		271, // Wooden Axe
		269, // Wooden Shovel
		inventory.ItemApple,
		inventory.ItemBread,
		inventory.ItemCookedPorkchop,
		inventory.ItemSteak,
		inventory.ItemDiamond,
		inventory.ItemCoal,
		inventory.ItemIronIngot,
		inventory.ItemStick,
		inventory.ItemTorch,
	}

	for _, id := range items {
		sprite := GetItemSprite16x16(id)
		nonZeroCount := 0
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if sprite[y][x] != 0 {
					nonZeroCount++
				}
			}
		}
		if nonZeroCount < 10 {
			t.Fatalf("item %d produced empty sprite with only %d pixels", id, nonZeroCount)
		}
	}
}
