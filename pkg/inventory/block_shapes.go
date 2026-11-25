package inventory

// BlockBox is a block-local axis-aligned box. Coordinates are in the [0, 1]
// range relative to the block position and are shared by collision and mesh
// generation so partial blocks cannot render and collide with different shapes.
type BlockBox struct {
	MinX, MinY, MinZ float32
	MaxX, MaxY, MaxZ float32
}

const (
	BlockStoneSlabTop    int32 = 18000
	BlockStoneSlabDouble int32 = 18001
	BlockOakSlabTop      int32 = 18002
	BlockOakSlabDouble   int32 = 18003
	FirstWoodSlabItem    int32 = 18010
	LastWoodSlabItem     int32 = 18022
	FirstWoodSlabTop     int32 = 20000
	FirstWoodSlabDouble  int32 = 21000
	FirstColorSlabItem   int32 = 18100
	LastColorSlabItem    int32 = 18131
	FirstColorSlabTop    int32 = 22000
	FirstColorSlabDouble int32 = 23000
)

var bottomSlabBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: .5, MaxZ: 1}}
var topSlabBox = []BlockBox{{MinX: 0, MinY: .5, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}
var fullBlockBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}

type woodSlabDefinition struct {
	ItemID      int32
	Name        string
	DisplayName string
	Texture     string
	PlankBlock  string
}

var woodSlabDefinitions = [...]woodSlabDefinition{
	{FirstWoodSlabItem + 0, "minecraft:spruce_slab", "云杉木台阶", "spruce_planks", "minecraft:spruce_planks"},
	{FirstWoodSlabItem + 1, "minecraft:birch_slab", "白桦木台阶", "birch_planks", "minecraft:birch_planks"},
	{FirstWoodSlabItem + 2, "minecraft:jungle_slab", "丛林木台阶", "jungle_planks", "minecraft:jungle_planks"},
	{FirstWoodSlabItem + 3, "minecraft:acacia_slab", "金合欢木台阶", "acacia_planks", "minecraft:acacia_planks"},
	{FirstWoodSlabItem + 4, "minecraft:dark_oak_slab", "深色橡木台阶", "dark_oak_planks", "minecraft:dark_oak_planks"},
	{FirstWoodSlabItem + 5, "minecraft:mangrove_slab", "红树木台阶", "mangrove_planks", "minecraft:mangrove_planks"},
	{FirstWoodSlabItem + 6, "minecraft:cherry_slab", "樱花木台阶", "cherry_planks", "minecraft:cherry_planks"},
	{FirstWoodSlabItem + 7, "minecraft:pale_oak_slab", "苍白橡木台阶", "pale_oak_planks", "minecraft:pale_oak_planks"},
	{FirstWoodSlabItem + 8, "minecraft:poplar_slab", "杨木台阶", "poplar_planks", "minecraft:poplar_planks"},
	{FirstWoodSlabItem + 9, "minecraft:bamboo_slab", "竹台阶", "bamboo_planks", "minecraft:bamboo_planks"},
	{FirstWoodSlabItem + 10, "minecraft:bamboo_mosaic_slab", "竹马赛克台阶", "bamboo_mosaic", "minecraft:bamboo_mosaic"},
	{FirstWoodSlabItem + 11, "minecraft:crimson_slab", "绯红菌柄台阶", "crimson_planks", "minecraft:crimson_planks"},
	{FirstWoodSlabItem + 12, "minecraft:warped_slab", "诡异菌柄台阶", "warped_planks", "minecraft:warped_planks"},
}

var woodSlabByItem = func() map[int32]woodSlabDefinition {
	defs := make(map[int32]woodSlabDefinition, len(woodSlabDefinitions))
	for _, definition := range woodSlabDefinitions {
		defs[definition.ItemID] = definition
	}
	return defs
}()

type colorSlabDefinition struct {
	ItemID       int32
	Name         string
	DisplayName  string
	Texture      string
	BlockName    string
	Hardness     float64
	HarvestTool  ToolType
	RequiresTool bool
}

var colorSlabDefinitions = buildColorSlabDefinitions()

func buildColorSlabDefinitions() []colorSlabDefinition {
	colors := [...]struct{ id, name string }{
		{"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]colorSlabDefinition, 0, len(colors)*2)
	for _, color := range colors {
		for _, kind := range []struct {
			suffix, display string
			hardness        float64
			tool            ToolType
			requiresTool    bool
		}{
			{"wool", "羊毛台阶", .8, ToolNone, false},
			{"concrete", "混凝土台阶", 1.8, ToolPickaxe, true},
		} {
			blockName := "minecraft:" + color.id + "_" + kind.suffix
			itemName := blockName + "_slab"
			definitions = append(definitions, colorSlabDefinition{
				ItemID: FirstColorSlabItem + int32(len(definitions)), Name: itemName,
				DisplayName: color.name + kind.display, Texture: color.id + "_" + kind.suffix,
				BlockName: blockName, Hardness: kind.hardness, HarvestTool: kind.tool, RequiresTool: kind.requiresTool,
			})
		}
	}
	return definitions
}

// BlockCollisionBoxes returns the collision/model boxes for implemented
// partial block states. A false result means the caller should use its normal
// full-cube or non-solid rule.
func BlockCollisionBoxes(blockID int32) ([]BlockBox, bool) {
	switch blockID {
	case ItemStoneSlab, ItemOakSlab:
		return bottomSlabBox, true
	case BlockStoneSlabTop, BlockOakSlabTop:
		return topSlabBox, true
	case BlockStoneSlabDouble, BlockOakSlabDouble:
		return fullBlockBox, true
	}
	if IsDoubleSlabState(blockID) {
		return fullBlockBox, true
	}
	if _, ok := SlabItemForBlockState(blockID); ok {
		if isTopSlabState(blockID) {
			return topSlabBox, true
		}
		return bottomSlabBox, true
	}
	return nil, false
}

// SlabItemForBlockState resolves a placed slab state to its inventory item.
func SlabItemForBlockState(blockID int32) (int32, bool) {
	switch blockID {
	case ItemStoneSlab, BlockStoneSlabTop, BlockStoneSlabDouble:
		return ItemStoneSlab, true
	case ItemOakSlab, BlockOakSlabTop, BlockOakSlabDouble:
		return ItemOakSlab, true
	}
	if blockID >= FirstWoodSlabItem && blockID <= LastWoodSlabItem {
		return blockID, true
	}
	if blockID >= FirstWoodSlabTop && blockID < FirstWoodSlabTop+int32(len(woodSlabDefinitions)) {
		return FirstWoodSlabItem + (blockID - FirstWoodSlabTop), true
	}
	if blockID >= FirstWoodSlabDouble && blockID < FirstWoodSlabDouble+int32(len(woodSlabDefinitions)) {
		return FirstWoodSlabItem + (blockID - FirstWoodSlabDouble), true
	}
	if blockID >= FirstColorSlabItem && blockID <= LastColorSlabItem {
		return blockID, true
	}
	if blockID >= FirstColorSlabTop && blockID < FirstColorSlabTop+int32(len(colorSlabDefinitions)) {
		return FirstColorSlabItem + (blockID - FirstColorSlabTop), true
	}
	if blockID >= FirstColorSlabDouble && blockID < FirstColorSlabDouble+int32(len(colorSlabDefinitions)) {
		return FirstColorSlabItem + (blockID - FirstColorSlabDouble), true
	}
	return 0, false
}

// SlabPlacementState chooses the top or bottom half from the hit face and hit
// height. The inventory item ID remains the canonical slab item ID.
func SlabPlacementState(itemID int32, faceY int32, localHitY float64) (int32, bool) {
	top := faceY < 0 || (faceY == 0 && localHitY > .5)
	switch itemID {
	case ItemStoneSlab:
		if top {
			return BlockStoneSlabTop, true
		}
		return ItemStoneSlab, true
	case ItemOakSlab:
		if top {
			return BlockOakSlabTop, true
		}
		return ItemOakSlab, true
	}
	if _, ok := woodSlabByItem[itemID]; ok {
		if top {
			return FirstWoodSlabTop + (itemID - FirstWoodSlabItem), true
		}
		return itemID, true
	}
	if itemID >= FirstColorSlabItem && itemID <= LastColorSlabItem {
		if top {
			return FirstColorSlabTop + (itemID - FirstColorSlabItem), true
		}
		return itemID, true
	}
	return itemID, false
}

// IsDoubleSlabState reports whether the state is a full double slab.
func IsDoubleSlabState(blockID int32) bool {
	return blockID == BlockStoneSlabDouble || blockID == BlockOakSlabDouble ||
		(blockID >= FirstWoodSlabDouble && blockID < FirstWoodSlabDouble+int32(len(woodSlabDefinitions))) ||
		(blockID >= FirstColorSlabDouble && blockID < FirstColorSlabDouble+int32(len(colorSlabDefinitions)))
}

func isTopSlabState(blockID int32) bool {
	return blockID == BlockStoneSlabTop || blockID == BlockOakSlabTop ||
		(blockID >= FirstWoodSlabTop && blockID < FirstWoodSlabTop+int32(len(woodSlabDefinitions))) ||
		(blockID >= FirstColorSlabTop && blockID < FirstColorSlabTop+int32(len(colorSlabDefinitions)))
}

// SlabMergeState returns the double-slab state when a click on a slab of the
// same material targets its opposite half or one of its horizontal surfaces.
func SlabMergeState(existingBlockID, itemID, faceY int32, localHitY float64) (int32, bool) {
	existingItem, ok := SlabItemForBlockState(existingBlockID)
	if !ok || existingItem != itemID {
		return 0, false
	}
	if IsDoubleSlabState(existingBlockID) {
		return 0, false
	}
	if faceY == 0 {
		existingTop := isTopSlabState(existingBlockID)
		if (existingTop && localHitY >= .5) || (!existingTop && localHitY <= .5) {
			return 0, false
		}
	}
	if itemID == ItemStoneSlab {
		return BlockStoneSlabDouble, true
	}
	if itemID == ItemOakSlab {
		return BlockOakSlabDouble, true
	}
	if _, ok := woodSlabByItem[itemID]; ok {
		return FirstWoodSlabDouble + (itemID - FirstWoodSlabItem), true
	}
	if itemID >= FirstColorSlabItem && itemID <= LastColorSlabItem {
		return FirstColorSlabDouble + (itemID - FirstColorSlabItem), true
	}
	return 0, false
}

// BlockItemForState returns the placeable item represented by a runtime block
// state, leaving ordinary block IDs unchanged.
func BlockItemForState(blockID int32) int32 {
	if itemID, ok := SlabItemForBlockState(blockID); ok {
		return itemID
	}
	return blockID
}
