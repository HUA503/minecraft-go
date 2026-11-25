package inventory

const (
	FirstWoodFenceItem int32 = 18600
	WoodFenceCount           = 13
)

type woodFenceDefinition struct {
	ItemID  int32
	Name    string
	Display string
	Texture string
	Source  string
}

var woodFenceDefinitions = buildWoodFenceDefinitions()

func buildWoodFenceDefinitions() []woodFenceDefinition {
	woods := []struct{ name, display string }{
		{"oak", "橡木"}, {"spruce", "云杉木"}, {"birch", "白桦木"}, {"jungle", "丛林木"},
		{"acacia", "金合欢木"}, {"dark_oak", "深色橡木"}, {"mangrove", "红树木"}, {"cherry", "樱花木"},
		{"pale_oak", "苍白橡木"}, {"poplar", "杨木"}, {"bamboo", "竹"}, {"crimson", "绯红菌柄"}, {"warped", "诡异菌柄"},
	}
	definitions := make([]woodFenceDefinition, 0, len(woods))
	for i, wood := range woods {
		definitions = append(definitions, woodFenceDefinition{
			ItemID: FirstWoodFenceItem + int32(i), Name: "minecraft:" + wood.name + "_fence",
			Display: wood.display + "栅栏", Texture: wood.name + "_planks", Source: "minecraft:" + wood.name + "_planks",
		})
	}
	return definitions
}

func woodFenceBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(woodFenceDefinitions))
	for _, fence := range woodFenceDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: fence.ItemID, Name: fence.Name, TextureTop: fence.Texture, TextureBottom: fence.Texture,
			TextureSide: fence.Texture, Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe,
			DropRuntimeID: fence.ItemID,
		})
		extendedBlockDisplayNames[fence.Name] = fence.Display
	}
	return definitions
}

// ModernWoodFenceItemDefinitions returns Java 26.3's thirteen wood fences.
func ModernWoodFenceItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), woodFenceBlockDefinitions()...)
}

// IsWoodFenceBlock reports whether blockID is one of the placeable wood fence items.
func IsWoodFenceBlock(blockID int32) bool {
	return blockID >= FirstWoodFenceItem && blockID < FirstWoodFenceItem+WoodFenceCount
}

// WoodFenceCollisionBoxesAt returns the post and connected horizontal rails.
// Neighbors are evaluated on demand so placement and removal update the outline
// without storing redundant connection bits in world data.
func WoodFenceCollisionBoxesAt(blockID, x, y, z int32, getBlock func(int32, int32, int32) int32) ([]BlockBox, bool) {
	if !IsWoodFenceBlock(blockID) {
		return nil, false
	}
	boxes := []BlockBox{{MinX: .375, MinY: 0, MinZ: .375, MaxX: .625, MaxY: 1.5, MaxZ: .625}}
	connect := [4]bool{}
	if getBlock != nil {
		connect[0] = fenceConnectsTo(getBlock(x, y, z-1)) // north
		connect[1] = fenceConnectsTo(getBlock(x+1, y, z)) // east
		connect[2] = fenceConnectsTo(getBlock(x, y, z+1)) // south
		connect[3] = fenceConnectsTo(getBlock(x-1, y, z)) // west
	}
	for _, railY := range [][2]float32{{.375, .5625}, {.75, .9375}} {
		if connect[0] {
			boxes = append(boxes, BlockBox{MinX: .375, MinY: railY[0], MinZ: 0, MaxX: .625, MaxY: railY[1], MaxZ: .5})
		}
		if connect[1] {
			boxes = append(boxes, BlockBox{MinX: .5, MinY: railY[0], MinZ: .375, MaxX: 1, MaxY: railY[1], MaxZ: .625})
		}
		if connect[2] {
			boxes = append(boxes, BlockBox{MinX: .375, MinY: railY[0], MinZ: .5, MaxX: .625, MaxY: railY[1], MaxZ: 1})
		}
		if connect[3] {
			boxes = append(boxes, BlockBox{MinX: 0, MinY: railY[0], MinZ: .375, MaxX: .5, MaxY: railY[1], MaxZ: .625})
		}
	}
	return boxes, true
}

func fenceConnectsTo(blockID int32) bool {
	if blockID == 0 || IsCushionBlock(blockID) || IsExtendedFoliage(blockID) {
		return false
	}
	if IsWoodFenceBlock(blockID) {
		return true
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
