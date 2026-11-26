package inventory

// BlockBox is a block-local axis-aligned box. Coordinates are in the [0, 1]
// range relative to the block position and are shared by collision and mesh
// generation so partial blocks cannot render and collide with different shapes.
type BlockBox struct {
	MinX, MinY, MinZ float32
	MaxX, MaxY, MaxZ float32
}

const (
	BlockStoneSlabTop      int32 = 18000
	BlockStoneSlabDouble   int32 = 18001
	BlockOakSlabTop        int32 = 18002
	BlockOakSlabDouble     int32 = 18003
	FirstWoodSlabItem      int32 = 18010
	LastWoodSlabItem       int32 = 18022
	FirstWoodSlabTop       int32 = 20000
	FirstWoodSlabDouble    int32 = 21000
	FirstColorSlabItem     int32 = 18100
	LastColorSlabItem      int32 = 18131
	FirstColorSlabTop      int32 = 22000
	FirstColorSlabDouble   int32 = 23000
	FirstMasonrySlabItem   int32 = 18200
	LastMasonrySlabItem    int32 = 18244
	FirstMasonrySlabTop    int32 = 25000
	FirstMasonrySlabDouble int32 = 26000
)

var bottomSlabBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: .5, MaxZ: 1}}
var topSlabBox = []BlockBox{{MinX: 0, MinY: .5, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}
var fullBlockBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}
var cushionVisualBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: .25, MaxZ: 1}}

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

type masonrySlabDefinition struct {
	ItemID        int32
	Name          string
	DisplayName   string
	BlockName     string
	TextureTop    string
	TextureBottom string
	TextureSide   string
	Hardness      float64
}

var masonrySlabDefinitions = buildMasonrySlabDefinitions()

func buildMasonrySlabDefinitions() []masonrySlabDefinition {
	specs := [...]struct {
		name, display, block, top, bottom, side string
		hardness                                float64
	}{
		{"cobblestone_slab", "圆石台阶", "cobblestone", "cobblestone", "cobblestone", "cobblestone", 2},
		{"mossy_cobblestone_slab", "苔石台阶", "mossy_cobblestone", "mossy_cobblestone", "mossy_cobblestone", "mossy_cobblestone", 2},
		{"smooth_stone_slab", "平滑石台阶", "smooth_stone", "smooth_stone", "smooth_stone", "smooth_stone_slab_side", 2},
		{"stone_brick_slab", "石砖台阶", "stone_bricks", "stone_bricks", "stone_bricks", "stone_bricks", 1.5},
		{"mossy_stone_brick_slab", "苔石砖台阶", "mossy_stone_bricks", "mossy_stone_bricks", "mossy_stone_bricks", "mossy_stone_bricks", 1.5},
		{"granite_slab", "花岗岩台阶", "granite", "granite", "granite", "granite", 1.5},
		{"polished_granite_slab", "磨制花岗岩台阶", "polished_granite", "polished_granite", "polished_granite", "polished_granite", 1.5},
		{"diorite_slab", "闪长岩台阶", "diorite", "diorite", "diorite", "diorite", 1.5},
		{"polished_diorite_slab", "磨制闪长岩台阶", "polished_diorite", "polished_diorite", "polished_diorite", "polished_diorite", 1.5},
		{"andesite_slab", "安山岩台阶", "andesite", "andesite", "andesite", "andesite", 1.5},
		{"polished_andesite_slab", "磨制安山岩台阶", "polished_andesite", "polished_andesite", "polished_andesite", "polished_andesite", 1.5},
		{"cobbled_deepslate_slab", "深板岩圆石台阶", "cobbled_deepslate", "cobbled_deepslate", "cobbled_deepslate", "cobbled_deepslate", 3.5},
		{"polished_deepslate_slab", "磨制深板岩台阶", "polished_deepslate", "polished_deepslate", "polished_deepslate", "polished_deepslate", 3.5},
		{"deepslate_brick_slab", "深板岩砖台阶", "deepslate_bricks", "deepslate_bricks", "deepslate_bricks", "deepslate_bricks", 3.5},
		{"deepslate_tile_slab", "深板岩瓦台阶", "deepslate_tiles", "deepslate_tiles", "deepslate_tiles", "deepslate_tiles", 3.5},
		{"tuff_slab", "凝灰岩台阶", "tuff", "tuff", "tuff", "tuff", 1.5},
		{"polished_tuff_slab", "磨制凝灰岩台阶", "polished_tuff", "polished_tuff", "polished_tuff", "polished_tuff", 1.5},
		{"tuff_brick_slab", "凝灰岩砖台阶", "tuff_bricks", "tuff_bricks", "tuff_bricks", "tuff_bricks", 1.5},
		{"brick_slab", "砖台阶", "bricks", "bricks", "bricks", "bricks", 2},
		{"mud_brick_slab", "泥砖台阶", "mud_bricks", "mud_bricks", "mud_bricks", "mud_bricks", 1.5},
		{"resin_brick_slab", "树脂砖台阶", "resin_bricks", "resin_bricks", "resin_bricks", "resin_bricks", 2},
		{"sandstone_slab", "砂岩台阶", "sandstone", "sandstone_top", "sandstone_bottom", "sandstone", .8},
		{"smooth_sandstone_slab", "平滑砂岩台阶", "smooth_sandstone", "sandstone_top", "sandstone_top", "sandstone_top", .8},
		{"cut_sandstone_slab", "切制砂岩台阶", "cut_sandstone", "cut_sandstone", "cut_sandstone", "cut_sandstone", .8},
		{"red_sandstone_slab", "红砂岩台阶", "red_sandstone", "red_sandstone_top", "red_sandstone_bottom", "red_sandstone", .8},
		{"smooth_red_sandstone_slab", "平滑红砂岩台阶", "smooth_red_sandstone", "red_sandstone_top", "red_sandstone_top", "red_sandstone_top", .8},
		{"cut_red_sandstone_slab", "切制红砂岩台阶", "cut_red_sandstone", "cut_red_sandstone", "cut_red_sandstone", "cut_red_sandstone", .8},
		{"cinnabar_slab", "朱砂台阶", "cinnabar", "cinnabar", "cinnabar", "cinnabar", 2},
		{"polished_cinnabar_slab", "磨制朱砂台阶", "polished_cinnabar", "polished_cinnabar", "polished_cinnabar", "polished_cinnabar", 2},
		{"cinnabar_brick_slab", "朱砂砖台阶", "cinnabar_bricks", "cinnabar_bricks", "cinnabar_bricks", "cinnabar_bricks", 2},
		{"sulfur_slab", "硫磺台阶", "sulfur", "sulfur", "sulfur", "sulfur", 2},
		{"polished_sulfur_slab", "磨制硫磺台阶", "polished_sulfur", "polished_sulfur", "polished_sulfur", "polished_sulfur", 2},
		{"sulfur_brick_slab", "硫磺砖台阶", "sulfur_bricks", "sulfur_bricks", "sulfur_bricks", "sulfur_bricks", 2},
		{"prismarine_slab", "海晶石台阶", "prismarine", "prismarine", "prismarine", "prismarine", 1.5},
		{"prismarine_brick_slab", "海晶石砖台阶", "prismarine_bricks", "prismarine_bricks", "prismarine_bricks", "prismarine_bricks", 1.5},
		{"dark_prismarine_slab", "暗海晶石台阶", "dark_prismarine", "dark_prismarine", "dark_prismarine", "dark_prismarine", 1.5},
		{"nether_brick_slab", "下界砖台阶", "nether_bricks", "nether_bricks", "nether_bricks", "nether_bricks", 2},
		{"red_nether_brick_slab", "红色下界砖台阶", "red_nether_bricks", "red_nether_bricks", "red_nether_bricks", "red_nether_bricks", 2},
		{"blackstone_slab", "黑石台阶", "blackstone", "blackstone", "blackstone", "blackstone", 2},
		{"polished_blackstone_slab", "磨制黑石台阶", "polished_blackstone", "polished_blackstone", "polished_blackstone", "polished_blackstone", 2},
		{"polished_blackstone_brick_slab", "磨制黑石砖台阶", "polished_blackstone_bricks", "polished_blackstone_bricks", "polished_blackstone_bricks", "polished_blackstone_bricks", 1.5},
		{"end_stone_brick_slab", "末地石砖台阶", "end_stone_bricks", "end_stone_bricks", "end_stone_bricks", "end_stone_bricks", 3},
		{"purpur_slab", "紫珀台阶", "purpur_block", "purpur_block", "purpur_block", "purpur_block", 1.5},
		{"quartz_slab", "石英台阶", "quartz_block", "quartz_block_top", "quartz_block_bottom", "quartz_block_side", .8},
		{"smooth_quartz_slab", "平滑石英台阶", "smooth_quartz", "quartz_block_top", "quartz_block_top", "quartz_block_top", .8},
	}
	definitions := make([]masonrySlabDefinition, 0, len(specs))
	for _, spec := range specs {
		name := "minecraft:" + spec.name
		definitions = append(definitions, masonrySlabDefinition{
			ItemID: FirstMasonrySlabItem + int32(len(definitions)), Name: name,
			DisplayName: spec.display, BlockName: "minecraft:" + spec.block,
			TextureTop: spec.top, TextureBottom: spec.bottom, TextureSide: spec.side, Hardness: spec.hardness,
		})
	}
	return definitions
}

// BlockCollisionBoxes returns the collision/model boxes for implemented
// partial block states. A false result means the caller should use its normal
// full-cube or non-solid rule.
func BlockCollisionBoxes(blockID int32) ([]BlockBox, bool) {
	return BlockCollisionBoxesAt(blockID, 0, 0, 0, nil)
}

// BlockCollisionBoxesAt resolves neighbor-dependent block outlines (currently
// stair corners) at a world position. A nil callback gives the straight shape.
func BlockCollisionBoxesAt(blockID, x, y, z int32, getBlock func(int32, int32, int32) int32) ([]BlockBox, bool) {
	if IsGlassPaneBlock(blockID) {
		return GlassPaneBoxesAt(x, y, z, getBlock), true
	}
	if itemID, count, _, ok := CandleStateProperties(blockID); ok {
		return CandleBoxes(itemID, count), true
	}
	if IsCandleItem(blockID) {
		return CandleBoxes(blockID, 1), true
	}
	if IsCushionBlock(blockID) {
		return cushionVisualBox, true
	}
	if boxes, ok := WoodFenceCollisionBoxesAt(blockID, x, y, z, getBlock); ok {
		return boxes, true
	}
	if boxes, ok := FenceGateModelBoxes(blockID); ok {
		return boxes, true
	}
	if boxes, ok := MasonryWallCollisionBoxesAt(blockID, x, y, z, getBlock); ok {
		return boxes, true
	}
	if _, _, _, stair := StairStateProperties(blockID); stair {
		shape := StairShapeAt(blockID, x, y, z, getBlock)
		return StairCollisionBoxesWithShape(blockID, shape)
	}
	if boxes, ok := StairCollisionBoxes(blockID); ok {
		return boxes, true
	}
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

var candleCollisionBox = []BlockBox{{MinX: 7.0 / 16.0, MinY: 0, MinZ: 7.0 / 16.0, MaxX: 9.0 / 16.0, MaxY: 6.0 / 16.0, MaxZ: 9.0 / 16.0}}

// CandleBoxes returns the individual 2x2-pixel candle bodies in the selected
// vanilla count state; lit candles share the same collision outline.
func CandleBoxes(itemID int32, count int) []BlockBox {
	if !IsCandleItem(itemID) || count < 1 || count > 4 {
		return nil
	}
	positions := [4][2]int{}
	switch count {
	case 1:
		positions[0] = [2]int{7, 7}
	case 2:
		positions[0], positions[1] = [2]int{5, 5}, [2]int{9, 9}
	case 3:
		positions[0], positions[1], positions[2] = [2]int{5, 5}, [2]int{9, 5}, [2]int{7, 9}
	case 4:
		positions = [4][2]int{{5, 5}, {9, 5}, {5, 9}, {9, 9}}
	}
	boxes := make([]BlockBox, 0, count)
	for i := 0; i < count; i++ {
		x, z := float32(positions[i][0])/16, float32(positions[i][1])/16
		boxes = append(boxes, BlockBox{MinX: x, MinY: 0, MinZ: z, MaxX: x + 2.0/16, MaxY: 6.0 / 16, MaxZ: z + 2.0/16})
	}
	return boxes
}

// GlassPaneBoxesAt returns the pane's center post and connected arms. The
// 2-pixel-wide arms use the same footprint for rendering and player collision.
func GlassPaneBoxesAt(x, y, z int32, getBlock func(int32, int32, int32) int32) []BlockBox {
	const min = float32(7.0 / 16.0)
	const max = float32(9.0 / 16.0)
	boxes := []BlockBox{{MinX: min, MinY: 0, MinZ: min, MaxX: max, MaxY: 1, MaxZ: max}}
	if getBlock == nil {
		return boxes
	}
	connects := func(id int32) bool {
		if id == 0 || id == 8 || id == 9 || id == 10 || id == 11 || id == 31 || id == 32 || id == 37 || id == 38 || id == 50 || id == 65 || id == 83 || id == 10063 || id == 10100 {
			return false
		}
		return true
	}
	if connects(getBlock(x-1, y, z)) {
		boxes = append(boxes, BlockBox{MinX: 0, MinY: 0, MinZ: min, MaxX: min, MaxY: 1, MaxZ: max})
	}
	if connects(getBlock(x+1, y, z)) {
		boxes = append(boxes, BlockBox{MinX: max, MinY: 0, MinZ: min, MaxX: 1, MaxY: 1, MaxZ: max})
	}
	if connects(getBlock(x, y, z-1)) {
		boxes = append(boxes, BlockBox{MinX: min, MinY: 0, MinZ: 0, MaxX: max, MaxY: 1, MaxZ: min})
	}
	if connects(getBlock(x, y, z+1)) {
		boxes = append(boxes, BlockBox{MinX: min, MinY: 0, MinZ: max, MaxX: max, MaxY: 1, MaxZ: 1})
	}
	return boxes
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
	if blockID >= FirstMasonrySlabItem && blockID <= LastMasonrySlabItem {
		return blockID, true
	}
	if blockID >= FirstMasonrySlabTop && blockID < FirstMasonrySlabTop+int32(len(masonrySlabDefinitions)) {
		return FirstMasonrySlabItem + (blockID - FirstMasonrySlabTop), true
	}
	if blockID >= FirstMasonrySlabDouble && blockID < FirstMasonrySlabDouble+int32(len(masonrySlabDefinitions)) {
		return FirstMasonrySlabItem + (blockID - FirstMasonrySlabDouble), true
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
	if itemID >= FirstMasonrySlabItem && itemID < FirstMasonrySlabItem+int32(len(masonrySlabDefinitions)) {
		if top {
			return FirstMasonrySlabTop + (itemID - FirstMasonrySlabItem), true
		}
		return itemID, true
	}
	return itemID, false
}

// IsDoubleSlabState reports whether the state is a full double slab.
func IsDoubleSlabState(blockID int32) bool {
	return blockID == BlockStoneSlabDouble || blockID == BlockOakSlabDouble ||
		(blockID >= FirstWoodSlabDouble && blockID < FirstWoodSlabDouble+int32(len(woodSlabDefinitions))) ||
		(blockID >= FirstColorSlabDouble && blockID < FirstColorSlabDouble+int32(len(colorSlabDefinitions))) ||
		(blockID >= FirstMasonrySlabDouble && blockID < FirstMasonrySlabDouble+int32(len(masonrySlabDefinitions)))
}

func isTopSlabState(blockID int32) bool {
	return blockID == BlockStoneSlabTop || blockID == BlockOakSlabTop ||
		(blockID >= FirstWoodSlabTop && blockID < FirstWoodSlabTop+int32(len(woodSlabDefinitions))) ||
		(blockID >= FirstColorSlabTop && blockID < FirstColorSlabTop+int32(len(colorSlabDefinitions))) ||
		(blockID >= FirstMasonrySlabTop && blockID < FirstMasonrySlabTop+int32(len(masonrySlabDefinitions)))
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
	if itemID >= FirstMasonrySlabItem && itemID < FirstMasonrySlabItem+int32(len(masonrySlabDefinitions)) {
		return FirstMasonrySlabDouble + (itemID - FirstMasonrySlabItem), true
	}
	return 0, false
}

// BlockItemForState returns the placeable item represented by a runtime block
// state, leaving ordinary block IDs unchanged.
func BlockItemForState(blockID int32) int32 {
	if itemID, _, _, _, ok := FenceGateStateProperties(blockID); ok {
		return itemID
	}
	if itemID, _, _, ok := CandleStateProperties(blockID); ok {
		return itemID
	}
	if itemID, ok := StairItemForBlockState(blockID); ok {
		return itemID
	}
	if itemID, ok := SlabItemForBlockState(blockID); ok {
		return itemID
	}
	return blockID
}
