package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestAABBIntersection(t *testing.T) {
	b1 := AABB{Min: Vec3{0, 0, 0}, Max: Vec3{1, 1, 1}}
	b2 := AABB{Min: Vec3{0.5, 0.5, 0.5}, Max: Vec3{1.5, 1.5, 1.5}}
	b3 := AABB{Min: Vec3{2, 2, 2}, Max: Vec3{3, 3, 3}}

	if !b1.Intersects(b2) {
		t.Fatalf("b1 and b2 should intersect")
	}
	if b1.Intersects(b3) {
		t.Fatalf("b1 and b3 should not intersect")
	}
}

func TestRaycastHitBlock(t *testing.T) {
	solidBlocks := map[[3]int32]bool{
		{5, 64, 0}: true,
	}

	isSolid := func(x, y, z int32) bool {
		return solidBlocks[[3]int32{x, y, z}]
	}

	// Cast ray from (0.5, 64.5, 0.5) along +X direction
	origin := Vec3{0.5, 64.5, 0.5}
	dir := Vec3{1.0, 0.0, 0.0}

	res := Raycast(origin, dir, 10.0, isSolid)
	if !res.Hit {
		t.Fatalf("raycast should hit solid block at (5, 64, 0)")
	}
	if res.BlockX != 5 || res.BlockY != 64 || res.BlockZ != 0 {
		t.Fatalf("hit block mismatch: (%d, %d, %d)", res.BlockX, res.BlockY, res.BlockZ)
	}
	if res.FaceX != -1 || res.FaceY != 0 || res.FaceZ != 0 {
		t.Fatalf("hit face mismatch: (%d, %d, %d)", res.FaceX, res.FaceY, res.FaceZ)
	}
}

func TestSimulationEngineTickAndGravity(t *testing.T) {
	eng := NewEngine()

	// Spawn player at Y = 70 (in the air)
	p := eng.AddPlayer("TestPlayer", [16]byte{1, 2, 3}, Vec3{X: 0, Y: 70, Z: 0})
	p.OnGround = false

	// Advance 5 ticks
	for i := 0; i < 5; i++ {
		eng.Tick()
	}

	if p.Position.Y >= 70 {
		t.Fatalf("player should have fallen due to gravity: Y=%f", p.Position.Y)
	}

	// Advance enough ticks to hit ground level (64)
	for i := 0; i < 50; i++ {
		eng.Tick()
	}

	if p.Position.Y != 64 {
		t.Fatalf("player should have landed at ground level 64: Y=%f", p.Position.Y)
	}
	if !p.OnGround {
		t.Fatalf("player OnGround should be true after landing")
	}
}

func TestBlockSetAndEventBroadcast(t *testing.T) {
	eng := NewEngine()
	chunk := world.NewChunk(world.ChunkPos{X: 0, Z: 0})
	eng.LoadChunk(world.ChunkPos{X: 0, Z: 0}, chunk)

	eng.SetBlock(10, 64, 10, 5) // Wood plank

	if b := eng.GetBlock(10, 64, 10); b != 5 {
		t.Fatalf("block mismatch: got %d, expected 5", b)
	}

	select {
	case ev := <-eng.Events:
		if ev.Type != "block_update" || ev.BlockPos.X != 10 || ev.BlockID != 5 {
			t.Fatalf("unexpected event: %+v", ev)
		}
	default:
		t.Fatalf("expected block_update event on channel")
	}
}
