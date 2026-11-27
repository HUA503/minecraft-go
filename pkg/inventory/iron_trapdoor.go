package inventory

const (
	ItemIronTrapdoor      int32 = 19100
	IronTrapdoorStateBase int32 = 45000
)

func ironTrapdoorDefinition() ExtendedBlockDefinition {
	const name = "minecraft:iron_trapdoor"
	extendedBlockDisplayNames[name] = "铁活板门"
	return ExtendedBlockDefinition{
		RuntimeID: ItemIronTrapdoor, Name: name,
		TextureTop: "iron_trapdoor", TextureBottom: "iron_trapdoor", TextureSide: "iron_trapdoor",
		Category: CreativeBlocks, Hardness: 5, HarvestTool: ToolPickaxe,
		RequiresHarvestTool: true, RequiredHarvestLevel: 0, DropRuntimeID: ItemIronTrapdoor,
	}
}

func IsIronTrapdoorItem(id int32) bool { return id == ItemIronTrapdoor }

func IsIronTrapdoorBlock(id int32) bool {
	if IsIronTrapdoorItem(id) {
		return true
	}
	_, _, _, ok := IronTrapdoorStateProperties(id)
	return ok
}

func IronTrapdoorBlockState(facing int, open, top bool) int32 {
	if facing < 0 || facing > 3 {
		return 0
	}
	props := facing
	if open {
		props |= 4
	}
	if top {
		props |= 8
	}
	return IronTrapdoorStateBase + int32(props)
}

func IronTrapdoorStateProperties(id int32) (facing int, open, top, ok bool) {
	offset := id - IronTrapdoorStateBase
	if offset < 0 || offset >= 16 {
		return 0, false, false, false
	}
	props := int(offset)
	return props & 3, props&4 != 0, props&8 != 0, true
}

func IronTrapdoorPlacementState(yaw float64, faceY int32, localHitY float64) int32 {
	top := localHitY >= .5
	if faceY < 0 {
		top = false
	} else if faceY > 0 {
		top = true
	}
	return IronTrapdoorBlockState(placementFacing(yaw), false, top)
}
