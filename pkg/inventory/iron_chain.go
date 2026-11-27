package inventory

const IronChainStateBase int32 = 51000

// ironChainDefinition registers the canonical iron chain block and its
// official textures. The item runtime ID is also the default vertical state.
func ironChainDefinition() ExtendedBlockDefinition {
	const name = "minecraft:iron_chain"
	extendedBlockDisplayNames[name] = "铁链"
	return ExtendedBlockDefinition{
		RuntimeID: ItemIronChain, Name: name,
		TextureTop: "iron_chain", TextureBottom: "iron_chain", TextureSide: "iron_chain",
		Category: CreativeFunctional, Hardness: 5, HarvestTool: ToolPickaxe,
		RequiresHarvestTool: true, DropRuntimeID: ItemIronChain,
	}
}

// IronChainBlockState encodes the axis: 0 is vertical, 1 is east-west,
// and 2 is north-south. The vertical state uses the canonical item ID.
func IronChainBlockState(axis int) (int32, bool) {
	if axis < 0 || axis > 2 {
		return 0, false
	}
	if axis == 0 {
		return ItemIronChain, true
	}
	return IronChainStateBase + int32(axis-1), true
}

func IronChainStateProperties(blockID int32) (axis int, ok bool) {
	if blockID == ItemIronChain {
		return 0, true
	}
	if blockID < IronChainStateBase || blockID > IronChainStateBase+1 {
		return 0, false
	}
	return int(blockID-IronChainStateBase) + 1, true
}

// IronChainPlacementState aligns a chain with the face on which it is placed.
func IronChainPlacementState(itemID, faceX, faceY, faceZ int32) (int32, bool) {
	if itemID != ItemIronChain {
		return 0, false
	}
	axis := 2
	if faceY != 0 {
		axis = 0
	} else if faceX != 0 {
		axis = 1
	} else if faceZ == 0 {
		return 0, false
	}
	return IronChainBlockState(axis)
}

// IronChainModelBoxes returns the narrow, full-length model/collision outline.
func IronChainModelBoxes(blockID int32) ([]BlockBox, bool) {
	axis, ok := IronChainStateProperties(blockID)
	if !ok {
		return nil, false
	}
	const min, max = float32(6.5 / 16), float32(9.5 / 16)
	switch axis {
	case 0:
		return []BlockBox{{MinX: min, MinY: 0, MinZ: min, MaxX: max, MaxY: 1, MaxZ: max}}, true
	case 1:
		return []BlockBox{{MinX: 0, MinY: min, MinZ: min, MaxX: 1, MaxY: max, MaxZ: max}}, true
	default:
		return []BlockBox{{MinX: min, MinY: min, MinZ: 0, MaxX: max, MaxY: max, MaxZ: 1}}, true
	}
}
