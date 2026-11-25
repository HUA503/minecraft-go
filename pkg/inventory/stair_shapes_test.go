package inventory

import "testing"

func TestStairShapeNeighborRulesAndGeometry(t *testing.T) {
	item := FirstMasonryStairItem
	for facing := StairFacing(0); facing < 4; facing++ {
		state, _ := StairBlockState(item, facing, false)
		dx, dz := stairFacingOffset(facing)
		perpendicular := StairFacing((int(facing) + 3) & 3)
		corner, _ := StairBlockState(item, perpendicular, false)
		blocks := map[[3]int32]int32{{0, 0, 0}: state, {dx, 0, dz}: corner}
		get := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
		shape := StairShapeAt(state, 0, 0, 0, get)
		if shape != StairShapeOuterLeft {
			t.Errorf("facing %d front perpendicular stair shape = %d, want outer-left", facing, shape)
		}
		assertStairShapeCellCount(t, state, shape, 5)

		delete(blocks, [3]int32{dx, 0, dz})
		blocks[[3]int32{-dx, 0, -dz}] = corner
		shape = StairShapeAt(state, 0, 0, 0, get)
		if shape != StairShapeInnerLeft {
			t.Errorf("facing %d rear perpendicular stair shape = %d, want inner-left", facing, shape)
		}
		assertStairShapeCellCount(t, state, shape, 7)
	}
}

func TestStairShapeRejectsDifferentHalfAndSuppressedCorners(t *testing.T) {
	state, _ := StairBlockState(FirstMasonryStairItem, 0, false)
	otherHalf, _ := StairBlockState(FirstMasonryStairItem, 1, true)
	blocks := map[[3]int32]int32{{0, 0, 0}: state, {0, 0, -1}: otherHalf}
	get := func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }
	if got := StairShapeAt(state, 0, 0, 0, get); got != StairShapeStraight {
		t.Fatalf("different-half neighbor produced shape %d", got)
	}

	other, _ := StairBlockState(FirstMasonryStairItem, 3, false)
	side, _ := StairBlockState(FirstMasonryStairItem, 0, false)
	blocks[[3]int32{0, 0, -1}] = other
	blocks[[3]int32{1, 0, 0}] = side
	if got := StairShapeAt(state, 0, 0, 0, get); got != StairShapeStraight {
		t.Fatalf("side-connected corner was not suppressed: shape %d", got)
	}
}

func assertStairShapeCellCount(t *testing.T, state int32, shape StairShape, want int) {
	t.Helper()
	count := 0
	for y := 0; y < 2; y++ {
		for z := 0; z < 2; z++ {
			for x := 0; x < 2; x++ {
				if StairVoxelOccupiedWithShape(state, x, y, z, shape) {
					count++
				}
			}
		}
	}
	boxes, ok := StairCollisionBoxesWithShape(state, shape)
	if !ok || len(boxes) != count {
		t.Errorf("shape %d collision boxes = %d (ok=%t), occupied cells = %d", shape, len(boxes), ok, count)
	}
	if count != want {
		t.Errorf("shape %d occupied cells = %d, want %d", shape, count, want)
	}
}
