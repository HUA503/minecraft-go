package inventory

const (
	FirstMasonryWallItem int32 = 18700
	MasonryWallCount           = 32
)

type masonryWallDefinition struct {
	ItemID  int32
	Name    string
	Display string
	Texture string
	Source  string
}

var masonryWallDefinitions = buildMasonryWallDefinitions()

func buildMasonryWallDefinitions() []masonryWallDefinition {
	materials := []struct{ id, display, texture, source string }{
		{"cobblestone", "圆石", "cobblestone", "cobblestone"}, {"mossy_cobblestone", "苔石", "mossy_cobblestone", "mossy_cobblestone"},
		{"stone_brick", "石砖", "stone_bricks", "stone_bricks"}, {"mossy_stone_brick", "苔石砖", "mossy_stone_bricks", "mossy_stone_bricks"},
		{"granite", "花岗岩", "granite", "granite"}, {"diorite", "闪长岩", "diorite", "diorite"}, {"andesite", "安山岩", "andesite", "andesite"},
		{"cobbled_deepslate", "深板岩圆石", "cobbled_deepslate", "cobbled_deepslate"}, {"polished_deepslate", "磨制深板岩", "polished_deepslate", "polished_deepslate"},
		{"deepslate_brick", "深板岩砖", "deepslate_bricks", "deepslate_bricks"}, {"deepslate_tile", "深板岩瓦", "deepslate_tiles", "deepslate_tiles"},
		{"tuff", "凝灰岩", "tuff", "tuff"}, {"polished_tuff", "磨制凝灰岩", "polished_tuff", "polished_tuff"}, {"tuff_brick", "凝灰岩砖", "tuff_bricks", "tuff_bricks"},
		{"brick", "砖块", "bricks", "bricks"}, {"mud_brick", "泥砖", "mud_bricks", "mud_bricks"}, {"resin_brick", "树脂砖", "resin_bricks", "resin_bricks"},
		{"sandstone", "砂岩", "sandstone", "sandstone"}, {"red_sandstone", "红砂岩", "red_sandstone", "red_sandstone"},
		{"cinnabar", "朱砂", "cinnabar", "cinnabar"}, {"polished_cinnabar", "磨制朱砂", "polished_cinnabar", "polished_cinnabar"}, {"cinnabar_brick", "朱砂砖", "cinnabar_bricks", "cinnabar_bricks"},
		{"sulfur", "硫磺", "sulfur", "sulfur"}, {"polished_sulfur", "磨制硫磺", "polished_sulfur", "polished_sulfur"}, {"sulfur_brick", "硫磺砖", "sulfur_bricks", "sulfur_bricks"},
		{"prismarine", "海晶石", "prismarine", "prismarine"}, {"nether_brick", "下界砖", "nether_bricks", "nether_bricks"}, {"red_nether_brick", "红色下界砖", "red_nether_bricks", "red_nether_bricks"},
		{"blackstone", "黑石", "blackstone", "blackstone"}, {"polished_blackstone", "磨制黑石", "polished_blackstone", "polished_blackstone"},
		{"polished_blackstone_brick", "磨制黑石砖", "polished_blackstone_bricks", "polished_blackstone_bricks"}, {"end_stone_brick", "末地石砖", "end_stone_bricks", "end_stone_bricks"},
	}
	definitions := make([]masonryWallDefinition, 0, len(materials))
	for i, material := range materials {
		definitions = append(definitions, masonryWallDefinition{
			ItemID: FirstMasonryWallItem + int32(i), Name: "minecraft:" + material.id + "_wall",
			Display: material.display + "墙", Texture: material.texture, Source: "minecraft:" + material.source,
		})
	}
	return definitions
}

func masonryWallBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(masonryWallDefinitions))
	for _, wall := range masonryWallDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: wall.ItemID, Name: wall.Name, TextureTop: wall.Texture, TextureBottom: wall.Texture,
			TextureSide: wall.Texture, Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolPickaxe,
			DropRuntimeID: wall.ItemID,
		})
		extendedBlockDisplayNames[wall.Name] = wall.Display
	}
	return definitions
}

// ModernMasonryWallItemDefinitions returns the 32 masonry wall entries in the
// Java 26.3 building-block creative tab.
func ModernMasonryWallItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), masonryWallBlockDefinitions()...)
}

func IsMasonryWallBlock(blockID int32) bool {
	return blockID >= FirstMasonryWallItem && blockID < FirstMasonryWallItem+MasonryWallCount
}

// MasonryWallCollisionBoxesAt builds the central post and connected arms from
// neighbors. Arms use the vanilla wall collision height; connectivity is
// derived from world data so it does not need a separately persisted state.
func MasonryWallCollisionBoxesAt(blockID, x, y, z int32, getBlock func(int32, int32, int32) int32) ([]BlockBox, bool) {
	if !IsMasonryWallBlock(blockID) {
		return nil, false
	}
	boxes := []BlockBox{{MinX: .25, MinY: 0, MinZ: .25, MaxX: .75, MaxY: 1.5, MaxZ: .75}}
	if getBlock == nil {
		return boxes, true
	}
	connect := [4]bool{
		wallConnectsTo(getBlock(x, y, z-1)), wallConnectsTo(getBlock(x+1, y, z)),
		wallConnectsTo(getBlock(x, y, z+1)), wallConnectsTo(getBlock(x-1, y, z)),
	}
	for _, railY := range [][2]float32{{0, .8125}, {.8125, 1.5}} {
		if connect[0] {
			boxes = append(boxes, BlockBox{MinX: .25, MinY: railY[0], MinZ: 0, MaxX: .75, MaxY: railY[1], MaxZ: .5})
		}
		if connect[1] {
			boxes = append(boxes, BlockBox{MinX: .5, MinY: railY[0], MinZ: .25, MaxX: 1, MaxY: railY[1], MaxZ: .75})
		}
		if connect[2] {
			boxes = append(boxes, BlockBox{MinX: .25, MinY: railY[0], MinZ: .5, MaxX: .75, MaxY: railY[1], MaxZ: 1})
		}
		if connect[3] {
			boxes = append(boxes, BlockBox{MinX: 0, MinY: railY[0], MinZ: .25, MaxX: .5, MaxY: railY[1], MaxZ: .75})
		}
	}
	return boxes, true
}

func wallConnectsTo(blockID int32) bool {
	if IsMasonryWallBlock(blockID) {
		return true
	}
	if blockID == 0 || IsWoodFenceBlock(blockID) || IsCushionBlock(blockID) || IsExtendedFoliage(blockID) {
		return false
	}
	if boxes, ok := BlockCollisionBoxes(blockID); ok {
		for _, box := range boxes {
			if box.MinX == 0 && box.MinY == 0 && box.MinZ == 0 && box.MaxX == 1 && box.MaxY == 1 && box.MaxZ == 1 {
				return true
			}
		}
		return false
	}
	switch blockID {
	case 6, 8, 9, 10, 11, 18, 20, 31, 32, 37, 38, 50, 65, 83:
		return false
	default:
		return true
	}
}
