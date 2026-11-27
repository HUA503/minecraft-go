package inventory

const (
	FirstWoodDoorItem     int32 = 18900
	WoodDoorCount               = 13
	FirstWoodTrapdoorItem int32 = 19000
	WoodTrapdoorCount           = 13
	WoodDoorStateBase     int32 = 43000
	WoodTrapdoorStateBase int32 = 44000
)

type woodOpenableDefinition struct {
	ItemID  int32
	Name    string
	Display string
	Texture string
	Source  string
}

var woodDoorDefinitions = buildWoodOpenableDefinitions(FirstWoodDoorItem, "door", "门")
var woodTrapdoorDefinitions = buildWoodOpenableDefinitions(FirstWoodTrapdoorItem, "trapdoor", "活板门")

func buildWoodOpenableDefinitions(firstID int32, suffix, displaySuffix string) []woodOpenableDefinition {
	woods := []struct{ name, display string }{
		{"oak", "橡木"}, {"spruce", "云杉木"}, {"birch", "白桦木"}, {"jungle", "丛林木"},
		{"acacia", "金合欢木"}, {"dark_oak", "深色橡木"}, {"mangrove", "红树木"}, {"cherry", "樱花木"},
		{"pale_oak", "苍白橡木"}, {"poplar", "杨木"}, {"bamboo", "竹"}, {"crimson", "绯红菌柄"}, {"warped", "诡异菌柄"},
	}
	definitions := make([]woodOpenableDefinition, 0, len(woods))
	for i, wood := range woods {
		itemID := firstID + int32(i)
		if suffix == "door" && i == 0 {
			// Keep the long-standing oak door item ID so old inventories and
			// worlds use the canonical registry item.
			itemID = ItemOakDoor
		}
		definitions = append(definitions, woodOpenableDefinition{
			ItemID: itemID, Name: "minecraft:" + wood.name + "_" + suffix,
			Display: wood.display + displaySuffix, Texture: wood.name + "_" + suffix,
			Source: "minecraft:" + wood.name + "_planks",
		})
	}
	return definitions
}

func woodOpenableBlockDefinitions(definitions []woodOpenableDefinition, door bool) []ExtendedBlockDefinition {
	blocks := make([]ExtendedBlockDefinition, 0, len(definitions))
	for _, openable := range definitions {
		top, bottom, side := openable.Texture, openable.Texture, openable.Texture
		if door {
			top += "_top"
			bottom += "_bottom"
			side = bottom
		}
		blocks = append(blocks, ExtendedBlockDefinition{
			RuntimeID: openable.ItemID, Name: openable.Name, TextureTop: top, TextureBottom: bottom,
			TextureSide: side, Category: CreativeBlocks, Hardness: 3, HarvestTool: ToolAxe,
			DropRuntimeID: openable.ItemID,
		})
		extendedBlockDisplayNames[openable.Name] = openable.Display
	}
	return blocks
}

func woodDoorBlockDefinitions() []ExtendedBlockDefinition {
	definitions := woodOpenableBlockDefinitions(woodDoorDefinitions, true)
	filtered := definitions[:0]
	for _, definition := range definitions {
		if definition.RuntimeID != ItemOakDoor {
			filtered = append(filtered, definition)
		}
	}
	return filtered
}

func woodTrapdoorBlockDefinitions() []ExtendedBlockDefinition {
	return woodOpenableBlockDefinitions(woodTrapdoorDefinitions, false)
}

func ModernWoodDoorItemDefinitions() []ExtendedBlockDefinition {
	// The oak door keeps its historical runtime ID (324), so it is not part of
	// the namespaced extended block registry. It still needs a full definition
	// for creative inventory rendering, textures and recipes.
	return append([]ExtendedBlockDefinition(nil), woodOpenableBlockDefinitions(woodDoorDefinitions, true)...)
}

func ModernWoodTrapdoorItemDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), woodTrapdoorBlockDefinitions()...)
}

func IsWoodDoorItem(id int32) bool {
	for _, door := range woodDoorDefinitions {
		if door.ItemID == id {
			return true
		}
	}
	return false
}

func woodDoorItemIndex(itemID int32) int {
	for index, door := range woodDoorDefinitions {
		if door.ItemID == itemID {
			return index
		}
	}
	return -1
}

func IsWoodTrapdoorItem(id int32) bool {
	return id >= FirstWoodTrapdoorItem && id < FirstWoodTrapdoorItem+WoodTrapdoorCount
}

func IsWoodDoorBlock(id int32) bool {
	if IsWoodDoorItem(id) {
		return true
	}
	_, _, _, _, _, ok := WoodDoorStateProperties(id)
	return ok
}

func IsWoodTrapdoorBlock(id int32) bool {
	if IsWoodTrapdoorItem(id) {
		return true
	}
	_, _, _, _, ok := WoodTrapdoorStateProperties(id)
	return ok
}

func IsTrapdoorBlock(id int32) bool {
	return IsWoodTrapdoorBlock(id) || IsIronTrapdoorBlock(id)
}

func WoodDoorBlockState(itemID int32, facing int, open, rightHinge, upper bool) int32 {
	itemIndex := woodDoorItemIndex(itemID)
	if itemIndex < 0 || facing < 0 || facing > 3 {
		return 0
	}
	props := facing
	if open {
		props |= 4
	}
	if rightHinge {
		props |= 8
	}
	if upper {
		props |= 16
	}
	return WoodDoorStateBase + int32(itemIndex*32+props)
}

func WoodDoorStateProperties(id int32) (itemID int32, facing int, open, rightHinge, upper, ok bool) {
	offset := id - WoodDoorStateBase
	if offset < 0 || offset >= WoodDoorCount*32 {
		return 0, 0, false, false, false, false
	}
	itemID = woodDoorDefinitions[offset/32].ItemID
	props := int(offset % 32)
	return itemID, props & 3, props&4 != 0, props&8 != 0, props&16 != 0, true
}

func WoodDoorPlacementState(itemID int32, yaw float64) (int32, bool) {
	if !IsWoodDoorItem(itemID) {
		return 0, false
	}
	facing := placementFacing(yaw)
	return WoodDoorBlockState(itemID, facing, false, false, false), true
}

func WoodTrapdoorBlockState(itemID int32, facing int, open, top bool) int32 {
	if !IsWoodTrapdoorItem(itemID) || facing < 0 || facing > 3 {
		return 0
	}
	props := facing
	if open {
		props |= 4
	}
	if top {
		props |= 8
	}
	return WoodTrapdoorStateBase + (itemID-FirstWoodTrapdoorItem)*16 + int32(props)
}

func WoodTrapdoorStateProperties(id int32) (itemID int32, facing int, open, top, ok bool) {
	offset := id - WoodTrapdoorStateBase
	if offset < 0 || offset >= WoodTrapdoorCount*16 {
		return 0, 0, false, false, false
	}
	itemID = FirstWoodTrapdoorItem + offset/16
	props := int(offset % 16)
	return itemID, props & 3, props&4 != 0, props&8 != 0, true
}

func WoodTrapdoorPlacementState(itemID int32, yaw float64, faceY int32, localHitY float64) (int32, bool) {
	if !IsWoodTrapdoorItem(itemID) {
		return 0, false
	}
	top := localHitY >= .5
	if faceY < 0 {
		top = false
	} else if faceY > 0 {
		top = true
	}
	return WoodTrapdoorBlockState(itemID, placementFacing(yaw), false, top), true
}

func placementFacing(yaw float64) int {
	quarter := int((yaw+45)/90) % 4
	if quarter < 0 {
		quarter += 4
	}
	return (2 - quarter + 4) % 4
}

func WoodDoorModelBoxes(id int32) ([]BlockBox, bool) {
	itemID, facing, open, rightHinge, _, ok := WoodDoorStateProperties(id)
	if !ok {
		if IsWoodDoorItem(id) {
			return WoodDoorModelBoxes(WoodDoorBlockState(id, 0, false, false, false))
		}
		return nil, false
	}
	_ = itemID
	if !open {
		if facing&1 == 0 {
			return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: .1875}}, true
		}
		return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: .1875, MaxY: 1, MaxZ: 1}}, true
	}
	near := rightHinge
	if facing == 1 || facing == 2 {
		near = !near
	}
	if facing&1 == 0 {
		if near {
			return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: .1875, MaxY: 1, MaxZ: 1}}, true
		}
		return []BlockBox{{MinX: .8125, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}, true
	}
	if near {
		return []BlockBox{{MinX: 0, MinY: 0, MinZ: .8125, MaxX: 1, MaxY: 1, MaxZ: 1}}, true
	}
	return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: .1875}}, true
}

func WoodTrapdoorModelBoxes(id int32) ([]BlockBox, bool) {
	itemID, facing, open, top, ok := WoodTrapdoorStateProperties(id)
	if !ok {
		if ironFacing, ironOpen, ironTop, valid := IronTrapdoorStateProperties(id); valid {
			itemID, facing, open, top, ok = ItemIronTrapdoor, ironFacing, ironOpen, ironTop, true
		}
	}
	if !ok {
		if IsWoodTrapdoorItem(id) {
			return WoodTrapdoorModelBoxes(WoodTrapdoorBlockState(id, 0, false, false))
		}
		if IsIronTrapdoorItem(id) {
			return WoodTrapdoorModelBoxes(IronTrapdoorBlockState(0, false, false))
		}
		return nil, false
	}
	_ = itemID
	if !open {
		if top {
			return []BlockBox{{MinX: 0, MinY: .8125, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}, true
		}
		return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: .1875, MaxZ: 1}}, true
	}
	switch facing {
	case 0:
		return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: .1875}}, true
	case 1:
		return []BlockBox{{MinX: .8125, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}, true
	case 2:
		return []BlockBox{{MinX: 0, MinY: 0, MinZ: .8125, MaxX: 1, MaxY: 1, MaxZ: 1}}, true
	default:
		return []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: .1875, MaxY: 1, MaxZ: 1}}, true
	}
}
