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
)

var bottomSlabBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: .5, MaxZ: 1}}
var topSlabBox = []BlockBox{{MinX: 0, MinY: .5, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}
var fullBlockBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 1, MaxZ: 1}}

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
	default:
		return nil, false
	}
}

// SlabItemForBlockState resolves a placed slab state to its inventory item.
func SlabItemForBlockState(blockID int32) (int32, bool) {
	switch blockID {
	case ItemStoneSlab, BlockStoneSlabTop, BlockStoneSlabDouble:
		return ItemStoneSlab, true
	case ItemOakSlab, BlockOakSlabTop, BlockOakSlabDouble:
		return ItemOakSlab, true
	default:
		return 0, false
	}
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
	default:
		return itemID, false
	}
}

// IsDoubleSlabState reports whether the state is a full double slab.
func IsDoubleSlabState(blockID int32) bool {
	return blockID == BlockStoneSlabDouble || blockID == BlockOakSlabDouble
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
		existingTop := existingBlockID == BlockStoneSlabTop || existingBlockID == BlockOakSlabTop
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
