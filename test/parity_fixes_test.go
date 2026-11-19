package test

import (
	"math"
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestAntiPenetrationPushOut(t *testing.T) {
	// Setup world with a solid stone wall at X = 10 (blocks [10..11, 64..66, 10..11])
	isSolid := func(x, y, z int32) bool {
		return x == 10 && y == 64 && z == 10
	}

	// Player forced partially inside block at X = 9.85 (player box extends from 9.55 to 10.15, so overlaps block by 0.15)
	initialPos := sim.Vec3{X: 9.85, Y: 64.0, Z: 10.5}
	delta := sim.Vec3{X: 0.1, Y: 0, Z: 0} // Moving deeper into wall

	resolved, _, _ := sim.ResolveMovementCollision(initialPos, delta, isSolid)
	playerBox := sim.NewPlayerAABB(resolved)
	blockBox := sim.NewBlockAABB(10, 64, 10)

	// Verify player has been pushed completely outside the block box
	if playerBox.Max.X > blockBox.Min.X+1e-3 {
		t.Fatalf("player still penetrates solid block: player Max.X=%f, block Min.X=%f", playerBox.Max.X, blockBox.Min.X)
	}
}

func TestHitCeilingStopsUpwardMomentum(t *testing.T) {
	// Ceiling block at Y = 66
	isSolid := func(x, y, z int32) bool {
		return x == 10 && y == 66 && z == 10
	}

	// Player at Y = 64.1 (height 1.8, so head at 65.9). Ceiling is at 66.0.
	// Player jumps up with dy = 0.5 (trying to reach Y = 64.6, head at 66.4)
	pos := sim.Vec3{X: 10.5, Y: 64.1, Z: 10.5}
	delta := sim.Vec3{X: 0, Y: 0.5, Z: 0}

	resolved, onGround, hitCeiling := sim.ResolveMovementCollision(pos, delta, isSolid)
	if !hitCeiling {
		t.Fatalf("expected hitCeiling to be true")
	}
	if onGround {
		t.Fatalf("should not be onGround when hitting ceiling")
	}

	// Head must not penetrate ceiling (head <= 66.0)
	headY := resolved.Y + 1.8
	if headY > 66.0001 {
		t.Fatalf("player head penetrated ceiling: headY=%f, ceiling at 66.0", headY)
	}
}

func TestRaycastDistanceAndHitPos(t *testing.T) {
	isSolid := func(x, y, z int32) bool {
		return x == 5 && y == 64 && z == 0
	}

	origin := sim.Vec3{X: 0.0, Y: 64.5, Z: 0.5}
	dir := sim.Vec3{X: 1.0, Y: 0.0, Z: 0.0}

	res := sim.Raycast(origin, dir, 10.0, isSolid)
	if !res.Hit {
		t.Fatalf("expected raycast hit")
	}
	if res.BlockX != 5 || res.BlockY != 64 || res.BlockZ != 0 {
		t.Fatalf("wrong hit block: (%d, %d, %d)", res.BlockX, res.BlockY, res.BlockZ)
	}
	if res.FaceX != -1 || res.FaceY != 0 || res.FaceZ != 0 {
		t.Fatalf("wrong hit face: (%d, %d, %d)", res.FaceX, res.FaceY, res.FaceZ)
	}

	// Expected distance from origin X=0.0 to face X=5.0 is exactly 5.0
	if math.Abs(res.Distance-5.0) > 0.01 {
		t.Fatalf("expected distance 5.0, got %f", res.Distance)
	}
	if math.Abs(res.HitPos.X-5.0) > 0.01 || math.Abs(res.HitPos.Y-64.5) > 0.01 || math.Abs(res.HitPos.Z-0.5) > 0.01 {
		t.Fatalf("expected HitPos (5.0, 64.5, 0.5), got (%f, %f, %f)", res.HitPos.X, res.HitPos.Y, res.HitPos.Z)
	}
}

func TestChunkUnloadMemoryBound(t *testing.T) {
	eng := sim.NewEngine()
	cp1 := world.ChunkPos{X: 0, Z: 0}
	cp2 := world.ChunkPos{X: 10, Z: 10}

	c1 := world.NewChunk(cp1)
	c2 := world.NewChunk(cp2)
	eng.LoadChunk(cp1, c1)
	eng.LoadChunk(cp2, c2)

	if len(eng.World) != 2 {
		t.Fatalf("expected 2 loaded chunks, got %d", len(eng.World))
	}

	// Unload distant chunk
	eng.UnloadChunk(cp2)
	if len(eng.World) != 1 {
		t.Fatalf("expected 1 loaded chunk after unload, got %d", len(eng.World))
	}
	if eng.World[cp2] != nil {
		t.Fatalf("chunk cp2 was not removed from World map")
	}
}

func TestBlockPlacementOverReplaceableFoliage(t *testing.T) {
	eng := sim.NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, sim.Vec3{X: 8.5, Y: 65.0, Z: 8.5})
	p.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemOakPlanks, Count: 64}

	// Set ground and tall grass
	eng.SetBlock(10, 64, 10, 2)  // Grass block
	eng.SetBlock(10, 65, 10, 31) // Tall Grass

	// Place plank directly into tall grass location
	ok := eng.PlaceBlock(p.ID, world.BlockPos{X: 10, Y: 65, Z: 10}, inventory.ItemOakPlanks)
	if !ok {
		t.Fatalf("failed to place block over tall grass")
	}

	if eng.GetBlock(10, 65, 10) != inventory.ItemOakPlanks {
		t.Fatalf("expected block 5 (planks) at (10, 65, 10), got %d", eng.GetBlock(10, 65, 10))
	}

	// Hotbar count should have decreased from 64 to 63
	if p.Inventory.Slots[0].Count != 63 {
		t.Fatalf("expected 63 planks in inventory, got %d", p.Inventory.Slots[0].Count)
	}
}
