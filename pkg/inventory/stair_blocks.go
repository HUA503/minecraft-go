package inventory

import (
	"math"
	"strings"
)

const (
	FirstColorStairItem    int32 = 18300
	LastColorStairItem     int32 = FirstColorStairItem + 31
	FirstColorStairState   int32 = 27000
	FirstWoodStairItem     int32 = 18400
	FirstWoodStairState    int32 = 28000
	FirstMasonryStairItem  int32 = 18500
	FirstMasonryStairState int32 = 29000
	StairStatesPerItem           = 8 // four horizontal facings, lower and upper halves
)

type colorStairDefinition struct {
	ItemID       int32
	Name         string
	DisplayName  string
	Texture      string
	SourceBlock  string
	Hardness     float64
	HarvestTool  ToolType
	RequiresTool bool
}

var colorStairDefinitions = buildColorStairDefinitions()

type woodStairDefinition struct {
	ItemID      int32
	Name        string
	DisplayName string
	Texture     string
	SourceBlock string
}

var woodStairDefinitions = buildWoodStairDefinitions()

type masonryStairDefinition struct {
	ItemID                                 int32
	Name, DisplayName                      string
	TextureTop, TextureBottom, TextureSide string
	SourceBlock                            string
	Hardness                               float64
}

var masonryStairDefinitions = buildMasonryStairDefinitions()

var masonryStairNames = []string{
	"stone_stairs", "cobblestone_stairs", "mossy_cobblestone_stairs",
	"stone_brick_stairs", "mossy_stone_brick_stairs", "granite_stairs", "polished_granite_stairs",
	"diorite_stairs", "polished_diorite_stairs", "andesite_stairs", "polished_andesite_stairs",
	"cobbled_deepslate_stairs", "polished_deepslate_stairs", "deepslate_brick_stairs", "deepslate_tile_stairs",
	"tuff_stairs", "polished_tuff_stairs", "tuff_brick_stairs", "brick_stairs", "mud_brick_stairs", "resin_brick_stairs",
	"sandstone_stairs", "smooth_sandstone_stairs", "red_sandstone_stairs", "smooth_red_sandstone_stairs",
	"cinnabar_stairs", "polished_cinnabar_stairs", "cinnabar_brick_stairs",
	"sulfur_stairs", "polished_sulfur_stairs", "sulfur_brick_stairs",
	"prismarine_stairs", "prismarine_brick_stairs", "dark_prismarine_stairs",
	"nether_brick_stairs", "red_nether_brick_stairs", "blackstone_stairs", "polished_blackstone_stairs",
	"polished_blackstone_brick_stairs", "end_stone_brick_stairs", "purpur_stairs", "quartz_stairs", "smooth_quartz_stairs",
}

func buildMasonryStairDefinitions() []masonryStairDefinition {
	definitions := make([]masonryStairDefinition, 0, len(masonryStairNames))
	for _, stairName := range masonryStairNames {
		if stairName == "stone_stairs" {
			definitions = append(definitions, masonryStairDefinition{
				ItemID: FirstMasonryStairItem + int32(len(definitions)), Name: "minecraft:stone_stairs",
				DisplayName: "石楼梯", TextureTop: "stone", TextureBottom: "stone", TextureSide: "stone",
				SourceBlock: "minecraft:stone", Hardness: 1.5,
			})
			continue
		}
		slabName := "minecraft:" + strings.TrimSuffix(stairName, "_stairs") + "_slab"
		for _, slab := range masonrySlabDefinitions {
			if slab.Name != slabName {
				continue
			}
			display := strings.TrimSuffix(slab.DisplayName, "台阶") + "楼梯"
			definitions = append(definitions, masonryStairDefinition{
				ItemID: FirstMasonryStairItem + int32(len(definitions)), Name: "minecraft:" + stairName,
				DisplayName: display, TextureTop: slab.TextureTop, TextureBottom: slab.TextureBottom,
				TextureSide: slab.TextureSide, SourceBlock: slab.BlockName, Hardness: slab.Hardness,
			})
			break
		}
	}
	return definitions
}

func buildWoodStairDefinitions() []woodStairDefinition {
	definitions := []woodStairDefinition{{
		ItemID: FirstWoodStairItem, Name: "minecraft:oak_stairs", DisplayName: "橡木楼梯",
		Texture: "oak_planks", SourceBlock: "minecraft:oak_planks",
	}}
	for _, slab := range woodSlabDefinitions {
		woodName := strings.TrimSuffix(strings.TrimPrefix(slab.PlankBlock, "minecraft:"), "_planks")
		definitions = append(definitions, woodStairDefinition{
			ItemID: FirstWoodStairItem + int32(len(definitions)),
			Name:   "minecraft:" + woodName + "_stairs", DisplayName: slab.DisplayName[:len(slab.DisplayName)-len("台阶")] + "楼梯",
			Texture: slab.Texture, SourceBlock: slab.PlankBlock,
		})
	}
	return definitions
}

func woodStairBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(woodStairDefinitions))
	for _, stair := range woodStairDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: stair.ItemID, Name: stair.Name,
			TextureTop: stair.Texture, TextureBottom: stair.Texture, TextureSide: stair.Texture,
			Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe, DropRuntimeID: stair.ItemID,
		})
		extendedBlockDisplayNames[stair.Name] = stair.DisplayName
	}
	return definitions
}

func masonryStairBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(masonryStairDefinitions))
	for _, stair := range masonryStairDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: stair.ItemID, Name: stair.Name,
			TextureTop: stair.TextureTop, TextureBottom: stair.TextureBottom, TextureSide: stair.TextureSide,
			Category: CreativeBlocks, Hardness: stair.Hardness, HarvestTool: ToolPickaxe,
			RequiresHarvestTool: true, DropRuntimeID: stair.ItemID,
		})
		extendedBlockDisplayNames[stair.Name] = stair.DisplayName
	}
	return definitions
}

func buildColorStairDefinitions() []colorStairDefinition {
	colors := [...]struct{ id, display string }{
		{"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]colorStairDefinition, 0, 32)
	for _, color := range colors {
		for _, kind := range []struct {
			name, display, texture, source string
			hardness                       float64
			tool                           ToolType
			requiresTool                   bool
		}{
			{"wool", "羊毛", color.id + "_wool", color.id + "_wool", .8, ToolNone, false},
			{"concrete", "混凝土", color.id + "_concrete", color.id + "_concrete", 1.8, ToolPickaxe, true},
		} {
			blockName := "minecraft:" + color.id + "_" + kind.name
			definitions = append(definitions, colorStairDefinition{
				ItemID: FirstColorStairItem + int32(len(definitions)),
				Name:   blockName + "_stairs", DisplayName: color.display + kind.display + "楼梯",
				Texture: kind.texture, SourceBlock: blockName, Hardness: kind.hardness,
				HarvestTool: kind.tool, RequiresTool: kind.requiresTool,
			})
		}
	}
	return definitions
}

func colorStairBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(colorStairDefinitions))
	for _, stair := range colorStairDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: stair.ItemID, Name: stair.Name,
			TextureTop: stair.Texture, TextureBottom: stair.Texture, TextureSide: stair.Texture,
			Category: CreativeDecoration, Hardness: stair.Hardness, HarvestTool: stair.HarvestTool,
			RequiresHarvestTool: stair.RequiresTool,
		})
	}
	return definitions
}

func ModernWoodStairItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), woodStairBlockDefinitions()...)
}

// ModernMasonryStairItemDefinitions returns the Java 26.3 stone-family stairs.
func ModernMasonryStairItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), masonryStairBlockDefinitions()...)
}

// ModernStairItemDefinitions returns the Java 26.3 colored wool/concrete stairs.
func ModernStairItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), colorStairBlockDefinitions()...)
}

// StairFacing uses world directions: 0 north (-Z), 1 east (+X), 2 south (+Z), 3 west (-X).
type StairFacing uint8

func StairBlockState(itemID int32, facing StairFacing, upsideDown bool) (int32, bool) {
	base, index, ok := stairStateIndex(itemID)
	if !ok || facing > 3 {
		return itemID, false
	}
	state := int32(facing)
	if upsideDown {
		state |= 4
	}
	return base + index*StairStatesPerItem + state, true
}

func stairStateIndex(itemID int32) (base, index int32, ok bool) {
	if itemID >= FirstColorStairItem && itemID <= LastColorStairItem {
		return FirstColorStairState, itemID - FirstColorStairItem, true
	}
	woodLast := FirstWoodStairItem + int32(len(woodStairDefinitions)) - 1
	if itemID >= FirstWoodStairItem && itemID <= woodLast {
		return FirstWoodStairState, itemID - FirstWoodStairItem, true
	}
	masonryLast := FirstMasonryStairItem + int32(len(masonryStairDefinitions)) - 1
	if itemID >= FirstMasonryStairItem && itemID <= masonryLast {
		return FirstMasonryStairState, itemID - FirstMasonryStairItem, true
	}
	return 0, 0, false
}

func StairPlacementState(itemID int32, yaw float64, faceY int32, localHitY float64) (int32, bool) {
	normalized := math.Mod(yaw, 360)
	if normalized < 0 {
		normalized += 360
	}
	yawQuarter := int(math.Floor((normalized+45)/90)) % 4
	facing := StairFacing((2 - yawQuarter + 4) % 4)
	upsideDown := faceY < 0 || (faceY == 0 && localHitY > .5)
	return StairBlockState(itemID, facing, upsideDown)
}

func StairItemForBlockState(blockID int32) (int32, bool) {
	if blockID >= FirstColorStairItem && blockID <= LastColorStairItem {
		return blockID, true
	}
	if blockID >= FirstMasonryStairItem && blockID < FirstMasonryStairItem+int32(len(masonryStairDefinitions)) {
		return blockID, true
	}
	colorStateEnd := FirstColorStairState + int32(len(colorStairDefinitions))*StairStatesPerItem
	if blockID >= FirstColorStairState && blockID < colorStateEnd {
		itemIndex := (blockID - FirstColorStairState) / StairStatesPerItem
		return FirstColorStairItem + itemIndex, true
	}
	woodStateEnd := FirstWoodStairState + int32(len(woodStairDefinitions))*StairStatesPerItem
	if blockID >= FirstWoodStairState && blockID < woodStateEnd {
		itemIndex := (blockID - FirstWoodStairState) / StairStatesPerItem
		return FirstWoodStairItem + itemIndex, true
	}
	masonryStateEnd := FirstMasonryStairState + int32(len(masonryStairDefinitions))*StairStatesPerItem
	if blockID >= FirstMasonryStairState && blockID < masonryStateEnd {
		itemIndex := (blockID - FirstMasonryStairState) / StairStatesPerItem
		return FirstMasonryStairItem + itemIndex, true
	}
	if blockID >= FirstWoodStairItem && blockID < FirstWoodStairItem+int32(len(woodStairDefinitions)) {
		return blockID, true
	}
	return 0, false
}

func StairStateProperties(blockID int32) (itemID int32, facing StairFacing, upsideDown bool, ok bool) {
	itemID, ok = StairItemForBlockState(blockID)
	if !ok || blockID == itemID {
		return itemID, 0, false, false
	}
	state := int32(0)
	if blockID >= FirstColorStairState && blockID < FirstColorStairState+int32(len(colorStairDefinitions))*StairStatesPerItem {
		state = (blockID - FirstColorStairState) % StairStatesPerItem
	} else if blockID >= FirstWoodStairState && blockID < FirstWoodStairState+int32(len(woodStairDefinitions))*StairStatesPerItem {
		state = (blockID - FirstWoodStairState) % StairStatesPerItem
	} else if blockID >= FirstMasonryStairState && blockID < FirstMasonryStairState+int32(len(masonryStairDefinitions))*StairStatesPerItem {
		state = (blockID - FirstMasonryStairState) % StairStatesPerItem
	} else {
		return itemID, 0, false, false
	}
	return itemID, StairFacing(state & 3), state&4 != 0, true
}

// StairCollisionBoxes returns the union of the half-block base and raised step.
func StairCollisionBoxes(blockID int32) ([]BlockBox, bool) {
	_, facing, upsideDown, ok := StairStateProperties(blockID)
	if !ok {
		return nil, false
	}
	var step BlockBox
	switch facing {
	case 0: // north: raised half is behind the lower front
		step = BlockBox{MinX: 0, MinY: .5, MinZ: .5, MaxX: 1, MaxY: 1, MaxZ: 1}
	case 1: // east
		step = BlockBox{MinX: 0, MinY: .5, MinZ: 0, MaxX: .5, MaxY: 1, MaxZ: 1}
	case 2: // south
		step = BlockBox{MinX: 0, MinY: .5, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: .5}
	case 3: // west
		step = BlockBox{MinX: .5, MinY: .5, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}
	}
	base := BlockBox{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: .5, MaxZ: 1}
	if upsideDown {
		base = BlockBox{MinX: 0, MinY: .5, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}
		step.MinY, step.MaxY = 0, .5
	}
	return []BlockBox{base, step}, true
}

func StairVoxelOccupied(blockID int32, x, y, z int) bool {
	_, facing, upsideDown, ok := StairStateProperties(blockID)
	if !ok || x < 0 || x > 1 || y < 0 || y > 1 || z < 0 || z > 1 {
		return false
	}
	if (!upsideDown && y == 0) || (upsideDown && y == 1) {
		return true
	}
	if (!upsideDown && y != 1) || (upsideDown && y != 0) {
		return false
	}
	switch facing {
	case 0:
		return z == 1
	case 1:
		return x == 0
	case 2:
		return z == 0
	default:
		return x == 1
	}
}
