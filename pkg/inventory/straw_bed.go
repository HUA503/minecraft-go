package inventory

import "math"

const StrawBedStateBase int32 = 53000

var strawBedCollisionBox = []BlockBox{{MinX: 0, MinY: 0, MinZ: 0, MaxX: 1, MaxY: 9.0 / 16, MaxZ: 1}}

func strawBedBlockDefinition() ExtendedBlockDefinition {
	return ExtendedBlockDefinition{
		RuntimeID: ItemStrawBed, Name: "minecraft:straw_bed",
		TextureTop: "straw_bed_particle", TextureBottom: "straw_bed_particle", TextureSide: "straw_bed_particle",
		TextureItem: "straw_bed_particle", Category: CreativeFunctional, Hardness: 0.2, DropRuntimeID: ItemStrawBed,
	}
}

func IsStrawBedItem(itemID int32) bool { return itemID == ItemStrawBed }

// StrawBedBlockState encodes the horizontal facing and head/foot half.
// Facing values use Minecraft's order: south, west, north, east.
func StrawBedBlockState(itemID int32, facing int, head bool) int32 {
	if !IsStrawBedItem(itemID) || facing < 0 || facing > 3 {
		return 0
	}
	state := StrawBedStateBase + int32(facing*2)
	if head {
		state++
	}
	return state
}

func StrawBedStateProperties(state int32) (itemID int32, facing int, head bool, ok bool) {
	offset := state - StrawBedStateBase
	if offset < 0 || offset >= 8 {
		return 0, 0, false, false
	}
	return ItemStrawBed, int(offset / 2), offset%2 == 1, true
}

func StrawBedPlacementState(itemID int32, yaw float64) (int32, bool) {
	if !IsStrawBedItem(itemID) {
		return 0, false
	}
	// Player yaw 0 faces south (+Z), matching Minecraft's Direction enum.
	facing := int(math.Round(yaw/90)) % 4
	if facing < 0 {
		facing += 4
	}
	return StrawBedBlockState(itemID, facing, false), true
}

// StrawBedOtherHalf returns the neighboring head/foot state paired with this bed state.
func StrawBedOtherHalf(state int32) (dx, dz, otherState int32, ok bool) {
	itemID, facing, head, ok := StrawBedStateProperties(state)
	if !ok {
		return 0, 0, 0, false
	}
	directions := [...]struct{ x, z int32 }{{0, 1}, {-1, 0}, {0, -1}, {1, 0}}
	direction := directions[facing]
	if head {
		direction.x, direction.z = -direction.x, -direction.z
	}
	return direction.x, direction.z, StrawBedBlockState(itemID, facing, !head), true
}
