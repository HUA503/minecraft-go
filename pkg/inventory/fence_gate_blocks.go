package inventory

const (
	FirstWoodFenceGateItem int32 = 18800
	WoodFenceGateCount           = 13
	FenceGateStateBase     int32 = 42000
)

type woodFenceGateDefinition struct {
	ItemID  int32
	Name    string
	Display string
	Texture string
	Source  string
}

var woodFenceGateDefinitions = buildWoodFenceGateDefinitions()

func buildWoodFenceGateDefinitions() []woodFenceGateDefinition {
	woods := []struct{ name, display string }{
		{"oak", "橡木"}, {"spruce", "云杉木"}, {"birch", "白桦木"}, {"jungle", "丛林木"},
		{"acacia", "金合欢木"}, {"dark_oak", "深色橡木"}, {"mangrove", "红树木"}, {"cherry", "樱花木"},
		{"pale_oak", "苍白橡木"}, {"poplar", "杨木"}, {"bamboo", "竹"}, {"crimson", "绯红菌柄"}, {"warped", "诡异菌柄"},
	}
	definitions := make([]woodFenceGateDefinition, 0, len(woods))
	for i, wood := range woods {
		texture := wood.name + "_planks"
		if wood.name == "bamboo" {
			texture = "bamboo_fence_gate"
		}
		definitions = append(definitions, woodFenceGateDefinition{
			ItemID: FirstWoodFenceGateItem + int32(i), Name: "minecraft:" + wood.name + "_fence_gate",
			Display: wood.display + "栅栏门", Texture: texture, Source: "minecraft:" + wood.name + "_planks",
		})
	}
	return definitions
}

func woodFenceGateBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(woodFenceGateDefinitions))
	for _, gate := range woodFenceGateDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: gate.ItemID, Name: gate.Name, TextureTop: gate.Texture, TextureBottom: gate.Texture,
			TextureSide: gate.Texture, Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe,
			DropRuntimeID: gate.ItemID,
		})
		extendedBlockDisplayNames[gate.Name] = gate.Display
	}
	return definitions
}

func ModernWoodFenceGateItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), woodFenceGateBlockDefinitions()...)
}

func IsWoodFenceGateItem(blockID int32) bool {
	return blockID >= FirstWoodFenceGateItem && blockID < FirstWoodFenceGateItem+WoodFenceGateCount
}

func IsWoodFenceGateBlock(blockID int32) bool {
	if IsWoodFenceGateItem(blockID) {
		return true
	}
	_, _, _, _, ok := FenceGateStateProperties(blockID)
	return ok
}

// FenceGateBlockState stores the item, horizontal facing, open flag and
// in-wall model variant in a collision-free internal state ID.
func FenceGateBlockState(itemID int32, facing int, open, inWall bool) int32 {
	if !IsWoodFenceGateItem(itemID) || facing < 0 || facing > 3 {
		return 0
	}
	properties := facing
	if open {
		properties |= 4
	}
	if inWall {
		properties |= 8
	}
	return FenceGateStateBase + (itemID-FirstWoodFenceGateItem)*16 + int32(properties)
}

func FenceGateStateProperties(stateID int32) (itemID int32, facing int, open, inWall, ok bool) {
	offset := stateID - FenceGateStateBase
	if offset < 0 || offset >= WoodFenceGateCount*16 {
		return 0, 0, false, false, false
	}
	itemID = FirstWoodFenceGateItem + offset/16
	properties := int(offset % 16)
	return itemID, properties & 3, properties&4 != 0, properties&8 != 0, true
}

func FenceGatePlacementState(itemID int32, yaw float64) (int32, bool) {
	if !IsWoodFenceGateItem(itemID) {
		return 0, false
	}
	quarter := int(((yaw + 45) / 90)) % 4
	if quarter < 0 {
		quarter += 4
	}
	return FenceGateBlockState(itemID, (2-quarter+4)%4, false, false), true
}

// FenceGateModelBoxes returns the four timber pieces, rotated into the open
// orientation when needed. The wall flag lowers the model by one pixel.
func FenceGateModelBoxes(stateID int32) ([]BlockBox, bool) {
	itemID, facing, open, inWall, ok := FenceGateStateProperties(stateID)
	if !ok {
		if IsWoodFenceGateItem(stateID) {
			return FenceGateModelBoxes(FenceGateBlockState(stateID, 0, false, false))
		}
		return nil, false
	}
	_ = itemID
	boxes := []BlockBox{
		{MinX: 0, MinY: 0, MinZ: .375, MaxX: .1875, MaxY: 1.5, MaxZ: .625},
		{MinX: .8125, MinY: 0, MinZ: .375, MaxX: 1, MaxY: 1.5, MaxZ: .625},
		{MinX: .1875, MinY: .375, MinZ: .375, MaxX: .8125, MaxY: .5625, MaxZ: .625},
		{MinX: .1875, MinY: .9375, MinZ: .375, MaxX: .8125, MaxY: 1.125, MaxZ: .625},
	}
	if inWall {
		for i := range boxes {
			boxes[i].MaxY -= 1.0 / 16
		}
	}
	if open {
		for i := range boxes {
			boxes[i].MinX, boxes[i].MinZ = boxes[i].MinZ, boxes[i].MinX
			boxes[i].MaxX, boxes[i].MaxZ = boxes[i].MaxZ, boxes[i].MaxX
		}
	}
	if facing&1 == 1 {
		for i := range boxes {
			boxes[i].MinX, boxes[i].MinZ = boxes[i].MinZ, boxes[i].MinX
			boxes[i].MaxX, boxes[i].MaxZ = boxes[i].MaxZ, boxes[i].MaxX
		}
	}
	return boxes, true
}
