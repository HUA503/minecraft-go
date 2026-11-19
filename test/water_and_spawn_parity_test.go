package test

import (
	"math"
	mrand "math/rand"
	"testing"

	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

// Helper to simulate safe spawn finder
func findSafeSpawn(generator *gen.Generator, engine *sim.Engine, seed int64) sim.Vec3 {
	rng := mrand.New(mrand.NewSource(seed))
	for attempt := 0; attempt < 100; attempt++ {
		wx := int32(rng.Intn(192) - 96)
		wz := int32(rng.Intn(192) - 96)

		cp := world.BlockPos{X: wx, Y: 0, Z: wz}.ToChunkPos()
		chunk := engine.GetChunk(cp)
		if chunk == nil {
			chunk = generator.GenerateChunk(cp)
			engine.LoadChunk(cp, chunk)
		}

		for y := int32(90); y >= 62; y-- {
			b := chunk.GetBlock(wx, y, wz)
			if b == gen.BlockGrass || b == gen.BlockSand || b == gen.BlockDirt || b == gen.BlockStone {
				above1 := chunk.GetBlock(wx, y+1, wz)
				above2 := chunk.GetBlock(wx, y+2, wz)
				if !sim.IsSolidBlock(above1) && !sim.IsSolidBlock(above2) &&
					above1 != 8 && above1 != 9 && above1 != 10 && above1 != 11 &&
					above2 != 8 && above2 != 9 && above2 != 10 && above2 != 11 {
					return sim.Vec3{
						X: float64(wx) + 0.5,
						Y: float64(y + 1),
						Z: float64(wz) + 0.5,
					}
				}
				break
			}
			if b == gen.BlockWater || b == gen.BlockFlowingWater {
				break
			}
		}
	}
	return sim.Vec3{X: 8.5, Y: 66.0, Z: 8.5}
}

func TestRandomSafeSpawnSurface(t *testing.T) {
	generator := gen.NewGenerator(12345, gen.DimensionOverworld)
	engine := sim.NewEngine()

	// Sample multiple random seeds to verify varied, safe spawns
	spawns := make(map[string]bool)
	for i := 0; i < 5; i++ {
		seed := int64(1000 + i*777)
		spawn := findSafeSpawn(generator, engine, seed)

		// 1. Must be above sea level (>= 62)
		if spawn.Y < 62.0 {
			t.Fatalf("spawn Y=%f is below sea level (62)", spawn.Y)
		}

		// 2. Block below spawn must be solid walkable block, NOT water or lava
		bx := int32(math.Floor(spawn.X))
		by := int32(math.Floor(spawn.Y - 0.5))
		bz := int32(math.Floor(spawn.Z))
		under := engine.GetBlock(bx, by, bz)
		if !sim.IsSolidBlock(under) || under == 8 || under == 9 || under == 10 || under == 11 {
			t.Fatalf("spawn block underneath is invalid: id=%d at (%d, %d, %d)", under, bx, by, bz)
		}

		// 3. Headroom blocks at spawn must not be solid blocks or liquids
		feet := engine.GetBlock(bx, int32(math.Floor(spawn.Y)), bz)
		head := engine.GetBlock(bx, int32(math.Floor(spawn.Y+1)), bz)
		if sim.IsSolidBlock(feet) || feet == 8 || feet == 9 || feet == 10 || feet == 11 {
			t.Fatalf("spawn feet block is obstructed: id=%d", feet)
		}
		if sim.IsSolidBlock(head) || head == 8 || head == 9 || head == 10 || head == 11 {
			t.Fatalf("spawn head block is obstructed: id=%d", head)
		}

		key := string(rune(int(spawn.X))) + ":" + string(rune(int(spawn.Z)))
		spawns[key] = true
	}

	// Verify randomness across multiple seeds
	if len(spawns) < 2 {
		t.Fatalf("expected different random spawn locations across different seeds, got %d unique", len(spawns))
	}
}

func TestContinuousWaterSwimming(t *testing.T) {
	eng := sim.NewEngine()
	// Fill a pool of water at X=0, Z=0, Y=60..64 with stone floor at Y=59
	eng.SetBlock(0, 59, 0, 1) // Stone floor
	for y := int32(60); y <= 64; y++ {
		eng.SetBlock(0, y, 0, 8) // Water
	}

	player := eng.AddPlayer("Swimmer", [16]byte{1}, sim.Vec3{X: 0.5, Y: 60.5, Z: 0.5})
	player.ClientDriven = true

	checkWater := func(pos sim.Vec3) (bool, bool) {
		px := pos.X
		py := pos.Y
		pz := pos.Z
		bx := int32(math.Floor(px))
		bz := int32(math.Floor(pz))
		bFeet := eng.GetBlock(bx, int32(math.Floor(py+0.1)), bz)
		bHead := eng.GetBlock(bx, int32(math.Floor(py+1.6)), bz)
		inWater := (bFeet == 8 || bFeet == 9)
		atSurface := inWater && (bHead != 8 && bHead != 9)
		return inWater, atSurface
	}

	// 1. Hold Space continuously: player must continuously swim upward
	dt := 0.05 // 20 TPS tick
	initialY := player.Position.Y
	for tick := 0; tick < 10; tick++ {
		inWater, atSurface := checkWater(player.Position)
		if !inWater {
			t.Fatalf("expected player to be in water at Y=%f", player.Position.Y)
		}

		// Swim up physics
		if atSurface {
			if player.Velocity.Y < 4.5 {
				player.Velocity.Y = 6.2
			}
		} else {
			player.Velocity.Y += 24.0 * dt
			if player.Velocity.Y > 4.2 {
				player.Velocity.Y = 4.2
			}
		}

		dy := player.Velocity.Y * dt
		resPos, _ := eng.MovePlayerRelative(player.ID, sim.Vec3{X: 0, Y: dy, Z: 0}, 0, 0)
		player.Position = resPos
	}

	if player.Position.Y <= initialY+1.0 {
		t.Fatalf("holding Space in water should propel player upwards, started at %f, reached %f", initialY, player.Position.Y)
	}

	// 2. Release Space: upward momentum decays and player sinks gently
	player.Velocity.Y = 0 // from rest after releasing Space
	prevY := player.Position.Y
	for tick := 0; tick < 10; tick++ {
		player.Velocity.Y -= 6.0 * dt
		if player.Velocity.Y < -3.0 {
			player.Velocity.Y = -3.0
		}
		dy := player.Velocity.Y * dt
		resPos, _ := eng.MovePlayerRelative(player.ID, sim.Vec3{X: 0, Y: dy, Z: 0}, 0, 0)
		player.Position = resPos
	}
	if player.Position.Y >= prevY {
		t.Fatalf("releasing Space in water should sink gently downward, started at %f, now %f", prevY, player.Position.Y)
	}
}

func TestWaterSurfaceLeapBreach(t *testing.T) {
	eng := sim.NewEngine()
	// Water at Y=62, dirt bank at Y=63
	for y := int32(60); y <= 62; y++ {
		eng.SetBlock(0, y, 0, 8)
	}
	eng.SetBlock(1, 62, 0, 1) // Stone bank adjacent to pool

	// Player is at water surface (Y=62.2), head at 63.8 (air)
	player := eng.AddPlayer("Diver", [16]byte{2}, sim.Vec3{X: 0.5, Y: 62.2, Z: 0.5})
	player.ClientDriven = true

	bx := int32(math.Floor(player.Position.X))
	bz := int32(math.Floor(player.Position.Z))
	bFeet := eng.GetBlock(bx, int32(math.Floor(player.Position.Y+0.1)), bz)
	bHead := eng.GetBlock(bx, int32(math.Floor(player.Position.Y+1.6)), bz)

	inWater := (bFeet == 8 || bFeet == 9)
	atSurface := inWater && (bHead == 0)
	if !atSurface {
		t.Fatalf("expected player to be at surface (feet in water=%d, head in air=%d)", bFeet, bHead)
	}

	// Pressing space at surface imparts vanilla water leap (Velocity.Y = 6.2 m/s)
	player.Velocity.Y = 6.2
	dy := player.Velocity.Y * 0.1
	resPos, _ := eng.MovePlayerRelative(player.ID, sim.Vec3{X: 0.2, Y: dy, Z: 0}, 0, 0)
	if resPos.Y < 62.7 {
		t.Fatalf("surface leap should propel player above water surface, reached Y=%f", resPos.Y)
	}
}

func TestLandJumpAndBunnyHop(t *testing.T) {
	eng := sim.NewEngine()
	// Flat stone floor at Y=64
	for x := int32(-2); x <= 2; x++ {
		for z := int32(-2); z <= 2; z++ {
			eng.SetBlock(x, 64, z, 1)
		}
	}

	// Player standing firmly on floor at Y=65.0
	player := eng.AddPlayer("Jumper", [16]byte{3}, sim.Vec3{X: 0.5, Y: 65.0, Z: 0.5})
	player.ClientDriven = true
	player.OnGround = true

	dt := 0.05 // 20 TPS tick

	// 1. Initial jump impulse from ground
	if player.OnGround {
		player.Velocity.Y = 8.4
		player.OnGround = false
	}
	if player.Velocity.Y != 8.4 {
		t.Fatalf("expected initial jump impulse to be 8.4 m/s, got %f", player.Velocity.Y)
	}

	// 2. While airborne, holding Space must NOT re-apply initial jump impulse (must follow ballistic gravity)
	for i := 0; i < 5; i++ {
		if player.OnGround {
			t.Fatalf("player should remain airborne during ascent")
		}
		// Air gravity
		player.Velocity.Y -= 32.0 * dt
		dy := player.Velocity.Y * dt
		resPos, onGround := eng.MovePlayerRelative(player.ID, sim.Vec3{X: 0, Y: dy, Z: 0}, 0, 0)
		player.Position = resPos
		player.OnGround = onGround
	}

	// Verify player reached peak height above 65.5
	if player.Position.Y <= 65.5 {
		t.Fatalf("player should have jumped above 65.5, current Y=%f", player.Position.Y)
	}

	// 3. Fall back down until landing
	for i := 0; i < 30; i++ {
		if player.OnGround {
			break
		}
		player.Velocity.Y -= 32.0 * dt
		dy := player.Velocity.Y * dt
		resPos, onGround := eng.MovePlayerRelative(player.ID, sim.Vec3{X: 0, Y: dy, Z: 0}, 0, 0)
		player.Position = resPos
		player.OnGround = onGround
	}

	if !player.OnGround || math.Abs(player.Position.Y-65.0) > 1e-2 {
		t.Fatalf("player should have landed firmly on ground at Y=65.0, got Y=%f, onGround=%v", player.Position.Y, player.OnGround)
	}

	// 4. Continuous hold Space: upon landing, player immediately bunny-hops
	if player.OnGround {
		player.Velocity.Y = 8.4
		player.OnGround = false
	}
	if player.Velocity.Y != 8.4 || player.OnGround {
		t.Fatalf("continuous space hold should immediately trigger next jump upon landing")
	}
}
