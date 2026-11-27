package render

import (
	"strings"
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

func TestJava263ColoredStairItemIconsUseTheirShape(t *testing.T) {
	for _, stair := range inventory.ModernStairItemDefinitions() {
		source, ok := inventory.ExtendedBlockByName(strings.TrimSuffix(stair.Name, "_stairs"))
		if !ok {
			if stair.Name == "minecraft:white_wool_stairs" {
				source = inventory.ExtendedBlockDefinition{RuntimeID: 35} // legacy white-wool runtime ID
				ok = true
			}
			if !ok {
				t.Fatalf("stair %s source block is not registered", stair.Name)
			}
		}
		stairIcon := RenderIsometricBlock16x16(stair.RuntimeID)
		blockIcon := RenderIsometricBlock16x16(source.RuntimeID)
		visible := 0
		different := false
		for y := range stairIcon {
			for x := range stairIcon[y] {
				if stairIcon[y][x] != 0 {
					visible++
				}
				if stairIcon[y][x] != blockIcon[y][x] {
					different = true
				}
			}
		}
		if visible == 0 || !different {
			t.Errorf("stair %s icon has %d visible pixels and differs from its source block=%t", stair.Name, visible, different)
		}
	}
}

func TestWoodStairItemIconsUseTheirShapeAndTextures(t *testing.T) {
	for _, stair := range inventory.ModernWoodStairItemDefinitions() {
		t.Run(stair.Name, func(t *testing.T) {
			if Textures[stair.RuntimeID] == nil {
				t.Fatal("wood stair world texture is missing")
			}
			icon := RenderIsometricBlock16x16(stair.RuntimeID)
			visible := 0
			for _, row := range icon {
				for _, pixel := range row {
					if pixel>>24 != 0 {
						visible++
					}
				}
			}
			if visible < 40 {
				t.Fatalf("wood stair icon has only %d visible pixels", visible)
			}
		})
	}
}

func TestJava263ColoredCushionItemsUseOfficialSprites(t *testing.T) {
	for id := inventory.FirstCushionBlock; id <= inventory.LastCushionBlock; id++ {
		if name := GetOfficialItemTextureName(id); name == "" {
			t.Fatalf("cushion %d has no item texture mapping", id)
		}
		if _, ok := GetOfficialItemSprite(id); !ok {
			t.Errorf("official cushion sprite for item %d is missing", id)
		}
		if texture := Textures[id]; texture == nil {
			t.Errorf("cushion %d has no world texture", id)
		}
	}
}

func TestSlabItemIconsUsePartialBlockSilhouettes(t *testing.T) {
	full := RenderIsometricBlock16x16(inventory.ItemStone)
	bottom := RenderIsometricBlock16x16(inventory.ItemStoneSlab)
	top := RenderIsometricBlock16x16(inventory.BlockStoneSlabTop)
	if bottom == full || top == full || bottom == top {
		t.Fatal("top, bottom, and full stone block icons must have distinct silhouettes")
	}
	firstPixel := func(sprite [16][16]uint32) int {
		for y := range sprite {
			for x := range sprite[y] {
				if sprite[y][x]>>24 != 0 {
					return y
				}
			}
		}
		return 16
	}
	if got, want := firstPixel(bottom), firstPixel(full)+4; got != want {
		t.Errorf("bottom slab silhouette starts at y=%d, want %d (full block shifted by half-height)", got, want)
	}
	if got, want := firstPixel(top), firstPixel(full); got != want {
		t.Errorf("top slab silhouette starts at y=%d, want %d", got, want)
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

func TestJava263CatalogItemsHaveRealIcons(t *testing.T) {
	for _, item := range inventory.CreativeCatalog(inventory.CreativeAll) {
		if inventory.IsPlaceableBlock(item.ID) {
			continue // Block items use the isometric block-model renderer.
		}
		if _, ok := inventory.GetToolProperties(item.ID); ok {
			continue // Tools use their tier/type-specific renderer when no flat sprite exists.
		}
		if _, ok := GetOfficialItemSprite(item.ID); !ok {
			t.Errorf("creative item %q (%d) has no official icon and would use the generic fallback", item.Name, item.ID)
		}
	}
}

func TestRedstoneControlIconsUseOfficialBlockTextures(t *testing.T) {
	items := []int32{
		inventory.ItemStonePressurePlate, inventory.ItemOakPressurePlate,
		inventory.ItemStoneButton, inventory.ItemOakButton,
		inventory.ItemPolishedBlackstonePressurePlate, inventory.ItemPolishedBlackstoneButton,
	}
	for _, definition := range inventory.ControlBlockDefinitions() {
		items = append(items, definition.RuntimeID)
	}
	for _, id := range items {
		if Textures[id] == nil {
			t.Errorf("redstone control %d has no loaded official block texture", id)
			continue
		}
		icon := RenderIsometricBlock16x16(id)
		nonTransparent := 0
		for y := range icon {
			for x := range icon[y] {
				if icon[y][x]>>24 != 0 {
					nonTransparent++
				}
			}
		}
		if nonTransparent == 0 {
			t.Errorf("redstone control %d rendered a blank icon", id)
		}
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
