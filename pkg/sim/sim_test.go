package sim

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
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

func TestTickDoesNotDeadlockAtHundredthTick(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("test", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	if player == nil {
		t.Fatal("expected player to be added")
	}
	eng.TickCount = 99

	// The removed 100-tick spawning path selected a random position 24–54
	// blocks from the player. Make every candidate a valid grass surface so the
	// regression reliably reaches the former nested SpawnMob lock.
	for chunkX := int32(-4); chunkX <= 3; chunkX++ {
		for chunkZ := int32(-4); chunkZ <= 3; chunkZ++ {
			pos := world.ChunkPos{X: chunkX, Z: chunkZ}
			chunk := world.NewChunk(pos)
			for localX := int32(0); localX < world.ChunkWidth; localX++ {
				for localZ := int32(0); localZ < world.ChunkWidth; localZ++ {
					chunk.SetBlock(chunkX*world.ChunkWidth+localX, 63, chunkZ*world.ChunkWidth+localZ, 2)
				}
			}
			eng.World[pos] = chunk
		}
	}

	done := make(chan struct{})
	go func() {
		eng.Tick()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Engine.Tick deadlocked during natural mob spawning at tick 100")
	}
}

func TestPoplarSaplingRandomTickGrowth(t *testing.T) {
	eng := NewEngine()
	for chunkX := int32(-1); chunkX <= 1; chunkX++ {
		for chunkZ := int32(-1); chunkZ <= 1; chunkZ++ {
			pos := world.ChunkPos{X: chunkX, Z: chunkZ}
			eng.World[pos] = world.NewChunk(pos)
		}
	}
	eng.SetBlock(15, 63, 15, 2)
	eng.SetBlock(15, 64, 15, 10063)

	eng.mu.Lock()
	grown := eng.growPoplarSaplingLocked(15, 64, 15)
	eng.mu.Unlock()
	if !grown {
		t.Fatal("supported poplar sapling with clear sky should grow")
	}
	for dy := int32(0); dy < 5; dy++ {
		if got := eng.GetBlock(15, 64+dy, 15); got != 10010 {
			t.Fatalf("poplar trunk at y=%d = %d, want block 10010", 64+dy, got)
		}
	}
	leafCount := 0
	for x := int32(13); x <= 17; x++ {
		for y := int32(67); y <= 69; y++ {
			for z := int32(13); z <= 17; z++ {
				id := eng.GetBlock(x, y, z)
				if id >= 10060 && id <= 10062 {
					leafCount++
				}
			}
		}
	}
	if leafCount < 30 {
		t.Fatalf("grown poplar has only %d leaves", leafCount)
	}
}

func TestPoplarSaplingGrowsThroughEngineRandomTickLoop(t *testing.T) {
	seed, saplingPos, ok := findPoplarGrowthRandomTick()
	if !ok {
		t.Fatal("failed to find a deterministic random-tick sample for poplar growth")
	}
	eng := NewEngine()
	eng.randomTickState = seed
	eng.World[world.ChunkPos{}] = world.NewChunk(world.ChunkPos{})
	eng.World[world.ChunkPos{}].SetBlock(saplingPos.X, saplingPos.Y-1, saplingPos.Z, 2)
	eng.World[world.ChunkPos{}].SetBlock(saplingPos.X, saplingPos.Y, saplingPos.Z, poplarSaplingID)
	eng.Players[1] = &PlayerEntity{
		ID: 1, IsAlive: true, ClientDriven: true,
		Position: Vec3{X: 0.5, Y: 64, Z: 0.5}, Inventory: inventory.NewPlayerInventory(),
	}

	eng.Tick()
	if got := eng.GetBlock(saplingPos.X, saplingPos.Y, saplingPos.Z); got != poplarLogID {
		t.Fatalf("engine random tick left block %d at sapling position; want poplar log %d", got, poplarLogID)
	}
}

func findPoplarGrowthRandomTick() (uint64, world.BlockPos, bool) {
	next := func(state *uint64) uint64 {
		value := *state
		value ^= value << 13
		value ^= value >> 7
		value ^= value << 17
		*state = value
		return value
	}
	for seed := uint64(1); seed < 10_000; seed++ {
		state := seed
		random := next(&state)
		pos := world.BlockPos{
			X: int32(random & 15), Y: world.MinWorldY + int32((random>>4)&15), Z: int32((random >> 8) & 15),
		}
		if next(&state)%7 == 0 && pos.X >= 2 && pos.X <= 13 && pos.Z >= 2 && pos.Z <= 13 && pos.Y > world.MinWorldY && pos.Y+5 <= world.MaxWorldY {
			return seed, pos, true
		}
	}
	return 0, world.BlockPos{}, false
}

func TestPoplarSaplingGrowthWaitsForObstructionToClear(t *testing.T) {
	eng := NewEngine()
	for chunkX := int32(-1); chunkX <= 1; chunkX++ {
		for chunkZ := int32(-1); chunkZ <= 1; chunkZ++ {
			pos := world.ChunkPos{X: chunkX, Z: chunkZ}
			eng.World[pos] = world.NewChunk(pos)
		}
	}
	eng.SetBlock(15, 63, 15, 2)
	eng.SetBlock(15, 64, 15, 10063)
	eng.SetBlock(17, 67, 15, 1)

	eng.mu.Lock()
	grown := eng.growPoplarSaplingLocked(15, 64, 15)
	eng.mu.Unlock()
	if grown || eng.GetBlock(15, 64, 15) != 10063 {
		t.Fatal("obstructed poplar growth must leave the sapling and terrain untouched")
	}
}

func TestPoplarLeavesBreakAndShearDrops(t *testing.T) {
	makeEngine := func() *Engine {
		eng := NewEngine()
		pos := world.ChunkPos{X: 0, Z: 0}
		eng.World[pos] = world.NewChunk(pos)
		eng.World[pos].SetBlock(2, 64, 2, 10060)
		eng.Players[1] = &PlayerEntity{ID: 1, IsAlive: true, Inventory: inventory.NewPlayerInventory()}
		eng.randomTickState = 1 // deterministic roll that does not drop a sapling
		return eng
	}

	hand := makeEngine()
	if dropped := hand.BreakBlock(1, world.BlockPos{X: 2, Y: 64, Z: 2}); dropped != inventory.ItemAir {
		t.Fatalf("hand-breaking poplar leaves dropped %d, want no drop", dropped)
	}
	if got := hand.Players[1].Inventory.Slots[0]; !got.IsEmpty() {
		t.Fatalf("hand-breaking leaves unexpectedly gave an item: %+v", got)
	}

	shears := makeEngine()
	drop, harvested := shears.BreakBlockWithTool(1, world.BlockPos{X: 2, Y: 64, Z: 2}, inventory.ItemShears)
	if !harvested || drop != 10060 {
		t.Fatalf("shears breaking poplar leaves = (drop %d, harvested %t), want (10060, true)", drop, harvested)
	}
}

func TestModernDeepslateHarvestAndDropRules(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("test", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	pos := world.BlockPos{X: 2, Y: 64, Z: 2}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 12006) // Deepslate
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenPickaxe); harvested || drop != inventory.ItemAir {
		t.Fatalf("wood pickaxe deepslate break = (%d, %v), want no drop", drop, harvested)
	}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 12006)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemStonePickaxe); !harvested || drop != 12007 {
		t.Fatalf("stone pickaxe deepslate break = (%d, %v), want cobbled deepslate (12007)", drop, harvested)
	}

	eng.SetBlock(pos.X, pos.Y, pos.Z, 12006)
	player.Inventory.Slots[0] = inventory.ItemStack{}
	if drop := eng.BreakBlock(player.ID, pos); drop != inventory.ItemAir {
		t.Fatalf("hand-mined deepslate drop = %d, want air/no item", drop)
	}
}

func TestRaycastHitBlock(t *testing.T) {
	solidBlocks := map[[3]int32]bool{
		{5, 64, 0}: true,
	}

	isSolid := func(x, y, z int32) bool {
		return solidBlocks[[3]int32{x, y, z}]
	}

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

func TestMiningDropsToInventory(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Miner", [16]byte{1}, Vec3{0, 64, 0})

	// 1. Break Stone (1) -> drops Cobblestone (4)
	eng.SetBlock(1, 64, 0, 1)
	drop := eng.BreakBlock(p.ID, world.BlockPos{X: 1, Y: 64, Z: 0})
	if drop != inventory.ItemCobblestone {
		t.Fatalf("expected cobblestone drop, got %d", drop)
	}
	if p.Inventory.Slots[0].ItemID != inventory.ItemCobblestone || p.Inventory.Slots[0].Count != 1 {
		t.Fatalf("item not added to inventory: %+v", p.Inventory.Slots[0])
	}

	// 2. Break Diamond Ore (56) -> drops Diamond (264)
	eng.SetBlock(2, 64, 0, 56)
	drop = eng.BreakBlock(p.ID, world.BlockPos{X: 2, Y: 64, Z: 0})
	if drop != inventory.ItemDiamond {
		t.Fatalf("expected diamond drop, got %d", drop)
	}
}

func TestCannotPlaceBlockInsidePlayer(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{10.5, 64.0, 10.5})

	// Attempt to place block right at (10, 64, 10) which intersects player
	success := eng.PlaceBlock(p.ID, world.BlockPos{X: 10, Y: 64, Z: 10}, 1)
	if success {
		t.Fatalf("should not be able to place block inside player")
	}

	// Place block next to player at (12, 64, 10) -> should succeed
	success = eng.PlaceBlock(p.ID, world.BlockPos{X: 12, Y: 64, Z: 10}, 1)
	if !success {
		t.Fatalf("should be able to place block adjacent to player")
	}
}

func TestRedShrubPlacementCollisionAndDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{1}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	if IsSolidBlock(10100) {
		t.Fatal("red shrub should not collide with players")
	}
	if eng.PlaceBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}, 10100) {
		t.Fatal("red shrub should not place without a supporting block")
	}
	eng.SetBlock(11, 63, 10, inventory.ItemStone)
	if !eng.PlaceBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}, 10100) {
		t.Fatal("red shrub should place on a solid block")
	}
	if got := eng.GetBlock(11, 64, 10); got != 10100 {
		t.Fatalf("placed red shrub ID = %d, want 10100", got)
	}
	if got := eng.BreakBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}); got != 10100 {
		t.Fatalf("breaking red shrub dropped item %d, want 10100", got)
	}
	if !inventory.IsReplaceableBlock(10100) {
		t.Fatal("red shrub should be replaceable by another block")
	}
}

func TestBoneMealGrowsRedShrubAndConsumesOnlyInSurvival(t *testing.T) {
	for _, gameMode := range []int{0, 1} {
		eng := NewEngine()
		player := eng.AddPlayer("Steve", [16]byte{byte(gameMode + 1)}, Vec3{X: 20.5, Y: 64, Z: 20.5})
		player.GameMode = gameMode
		pos := world.BlockPos{X: 10, Y: 64, Z: 10}
		eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemStone)
		eng.SetBlock(pos.X, pos.Y, pos.Z, 10100)
		for _, direction := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			eng.SetBlock(pos.X+direction[0], pos.Y-1, pos.Z+direction[1], inventory.ItemStone)
		}
		player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 2}

		grown, ok := eng.UseBoneMealOnRedShrub(player.ID, pos)
		if !ok {
			t.Fatalf("game mode %d: bone meal did not grow a supported red shrub", gameMode)
		}
		if got := eng.GetBlock(grown.X, grown.Y, grown.Z); got != 10100 {
			t.Fatalf("game mode %d: grown block ID = %d, want red shrub", gameMode, got)
		}
		wantCount := 1
		if gameMode == 1 {
			wantCount = 2
		}
		if got := player.Inventory.Slots[0].Count; got != wantCount {
			t.Fatalf("game mode %d: bone meal count = %d, want %d", gameMode, got, wantCount)
		}
	}
}

func TestBoneMealDoesNotConsumeWhenRedShrubCannotSpread(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{1}, Vec3{X: 20.5, Y: 64, Z: 20.5})
	pos := world.BlockPos{X: 10, Y: 64, Z: 10}
	eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemStone)
	eng.SetBlock(pos.X, pos.Y, pos.Z, 10100)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 2}

	if _, ok := eng.UseBoneMealOnRedShrub(player.ID, pos); ok {
		t.Fatal("red shrub unexpectedly spread without adjacent supporting ground")
	}
	if got := player.Inventory.Slots[0].Count; got != 2 {
		t.Fatalf("failed bone meal use consumed an item: count = %d", got)
	}
}

func TestBlockPlacementAllFacesAndPillaring(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{8.5, 65.0, 8.5})
	p.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemOakPlanks, Count: 64}

	// 1. Place on top of ground block (adjacent at 9, 65, 8)
	ok := eng.PlaceBlock(p.ID, world.BlockPos{X: 9, Y: 65, Z: 8}, inventory.ItemOakPlanks)
	if !ok {
		t.Fatalf("failed to place block on adjacent ground face")
	}
	if eng.GetBlock(9, 65, 8) != inventory.ItemOakPlanks {
		t.Fatalf("block was not placed at (9, 65, 8)")
	}

	// 2. Place on +X face of newly placed block (10, 65, 8)
	ok = eng.PlaceBlock(p.ID, world.BlockPos{X: 10, Y: 65, Z: 8}, inventory.ItemCobblestone)
	if !ok {
		t.Fatalf("failed to place block on +X face")
	}

	// 3. Place on +Z face (9, 65, 9)
	ok = eng.PlaceBlock(p.ID, world.BlockPos{X: 9, Y: 65, Z: 9}, inventory.ItemDirt)
	if !ok {
		t.Fatalf("failed to place block on +Z face")
	}

	// 4. Pillaring: player jumps up to Y = 66.2, places block underneath feet at (8, 65, 8)
	p.Position.Y = 66.2
	ok = eng.PlaceBlock(p.ID, world.BlockPos{X: 8, Y: 65, Z: 8}, inventory.ItemStone)
	if !ok {
		t.Fatalf("pillaring failed: should place block underneath feet when jumping")
	}
	if eng.GetBlock(8, 65, 8) != inventory.ItemStone {
		t.Fatalf("pillared block not set at (8, 65, 8)")
	}
}

func TestFallDamageAndDeathRespawn(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{0, 64, 0})

	// Damage player by 15 HP
	eng.DamagePlayer(p.ID, 15.0, "attack")
	if p.Health != 5.0 {
		t.Fatalf("health mismatch: %f", p.Health)
	}

	// Fatal damage
	eng.DamagePlayer(p.ID, 10.0, "fatal")
	if p.IsAlive || p.Health != 0 {
		t.Fatalf("player should be dead: alive=%v, health=%f", p.IsAlive, p.Health)
	}

	// Respawn
	eng.RespawnPlayer(p.ID, Vec3{0, 64, 0})
	if !p.IsAlive || p.Health != 20.0 {
		t.Fatalf("player should be respawned with 20 HP: alive=%v, health=%f", p.IsAlive, p.Health)
	}
}

func TestZombieAIAndPlayerCombat(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Player", [16]byte{2}, Vec3{X: 10, Y: 64, Z: 10})

	// Spawn Zombie 5 blocks away
	zombie := eng.SpawnMob(MobZombie, Vec3{X: 10, Y: 64, Z: 15})
	if zombie.Health != 20.0 || !zombie.IsAlive {
		t.Fatalf("zombie should spawn with 20 HP")
	}

	// Tick simulation: zombie should detect player and walk towards player (-Z)
	eng.Tick()
	if zombie.Velocity.Z >= 0 {
		t.Fatalf("zombie should move towards player (-Z), got vz=%f", zombie.Velocity.Z)
	}

	// Move zombie directly next to player (dist < 1.5 blocks)
	zombie.Position = Vec3{X: 10, Y: 64, Z: 11}
	zombie.AttackCooldown = 0
	eng.Tick()

	// Zombie should have attacked player for 3.0 damage (20 - 3 = 17)
	if player.Health != 17.0 {
		t.Fatalf("expected player health 17.0 after zombie attack, got %f", player.Health)
	}

	// Player strikes back and kills zombie
	drops, killed := eng.DamageMob(zombie.ID, 20.0)
	if !killed || len(drops) == 0 {
		t.Fatalf("zombie should be killed and drop items: killed=%v, drops=%+v", killed, drops)
	}
	if drops[0].ItemID != 367 { // Rotten flesh
		t.Fatalf("expected rotten flesh drop (367), got %d", drops[0].ItemID)
	}
}

func TestCreeperExplosionDestruction(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Player", [16]byte{3}, Vec3{X: 0, Y: 64, Z: 0})

	// Place dirt blocks around player
	eng.SetBlock(1, 64, 1, 3)
	eng.SetBlock(2, 64, 1, 3)

	// Spawn Creeper 2 blocks away
	creeper := eng.SpawnMob(MobCreeper, Vec3{X: 1, Y: 64, Z: 1})

	// Tick 30 times to let fuse count down to explosion
	for i := 0; i < 30; i++ {
		eng.Tick()
	}

	// Creeper should have exploded
	if creeper.IsAlive {
		t.Fatalf("creeper should have exploded")
	}
	// Blocks in radius should have been blasted to Air (0)
	if eng.GetBlock(1, 64, 1) != 0 {
		t.Fatalf("dirt block at (1, 64, 1) should be destroyed by explosion")
	}
	// Player should have taken blast damage
	if player.Health >= 20.0 {
		t.Fatalf("player should have taken explosion damage, health=%f", player.Health)
	}
}

func TestMovementCollisionNoClipping(t *testing.T) {
	eng := NewEngine()
	// Solid floor at Y=63
	for x := int32(8); x <= 12; x++ {
		for z := int32(8); z <= 12; z++ {
			eng.SetBlock(x, 63, z, 1) // Stone
		}
	}
	// Solid stone wall at X=11 (blocks at (11, 64, 10) and (11, 65, 10))
	eng.SetBlock(11, 64, 10, 1)
	eng.SetBlock(11, 65, 10, 1)

	player := eng.AddPlayer("Steve", [16]byte{5}, Vec3{X: 10.0, Y: 64.0, Z: 10.0})

	// 1. Try to walk into the stone wall by delta X = +2.0
	resPos, _, _ := eng.MovePlayerRelative(player.ID, Vec3{X: 2.0, Y: 0, Z: 0}, 0, 0)

	// Wall is at X=11. Player has half-width 0.3. Max X allowed is 11.0 - 0.3 = 10.7
	if resPos.X > 10.701 {
		t.Fatalf("player clipped into wall! Expected X <= 10.7, got %f", resPos.X)
	}
	if resPos.X < 10.69 {
		t.Fatalf("player stopped too far from wall, got X=%f", resPos.X)
	}

	// 2. Try to fall through solid stone floor (Y=63)
	resPos, ground, _ := eng.MovePlayerRelative(player.ID, Vec3{X: 0, Y: -2.0, Z: 0}, 0, 0)
	if !ground {
		t.Fatalf("player should be flagged onGround after landing on floor")
	}
	// Floor top is Y=63+1 = 64.0
	if math.Abs(resPos.Y-64.0) > 1e-3 {
		t.Fatalf("player feet should rest exactly on floor Y=64.0, got %f", resPos.Y)
	}

	// 3. High-velocity fall from height (e.g. falling 10 blocks in a lag spike)
	highPlayer := eng.AddPlayer("FallenSteve", [16]byte{6}, Vec3{X: 10.0, Y: 75.0, Z: 10.0})
	resHighPos, highGround, _ := eng.MovePlayerRelative(highPlayer.ID, Vec3{X: 0, Y: -15.0, Z: 0}, 0, 0)
	if !highGround {
		t.Fatalf("high velocity fall should be caught by floor and flagged onGround")
	}
	if math.Abs(resHighPos.Y-64.0) > 1e-3 {
		t.Fatalf("high velocity fall should stop exactly at floor Y=64.0, got %f", resHighPos.Y)
	}

	// 4. Sprinting diagonally into wall corner
	cornerPlayer := eng.AddPlayer("CornerSteve", [16]byte{7}, Vec3{X: 10.0, Y: 64.0, Z: 9.0})
	resCornerPos, _, _ := eng.MovePlayerRelative(cornerPlayer.ID, Vec3{X: 2.0, Y: 0, Z: 2.0}, 0, 0)
	// Block at (11, 64, 10) prevents passing (X <= 10.7, Z <= 9.7)
	if resCornerPos.X > 10.701 && resCornerPos.Z > 9.701 {
		t.Fatalf("corner penetration occurred! Pos=%v", resCornerPos)
	}
}

func TestAuthenticMiningAndToolProgression(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Miner", [16]byte{9}, Vec3{X: 0, Y: 64, Z: 0})

	// 1. Bare hand breaking Stone (ID: 1): cannot harvest, yields no item
	eng.SetBlock(1, 64, 0, 1) // Stone
	drop, harvested := eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 1, Y: 64, Z: 0}, inventory.ItemAir)
	if harvested || drop != 0 {
		t.Fatalf("bare hands should NOT harvest stone, got drop=%d, harvested=%v", drop, harvested)
	}

	// 2. Wooden Pickaxe breaking Stone (ID: 1): can harvest, yields Cobblestone (4)
	eng.SetBlock(2, 64, 0, 1) // Stone
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1, Damage: 0}
	player.Inventory.SelectedHotbar = 0

	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 2, Y: 64, Z: 0}, inventory.ItemWoodenPickaxe)
	if !harvested || drop != inventory.ItemCobblestone {
		t.Fatalf("wooden pickaxe must harvest stone into cobblestone, got drop=%d, harvested=%v", drop, harvested)
	}
	// Tool durability must decrease by 1
	if player.Inventory.Slots[0].Damage != 1 {
		t.Fatalf("wooden pickaxe should have taken 1 damage, got %d", player.Inventory.Slots[0].Damage)
	}

	// 3. Diamond Ore (56) with Stone Pickaxe (Tier 1): cannot harvest (needs Iron Tier 2)
	eng.SetBlock(3, 64, 0, 56)
	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 3, Y: 64, Z: 0}, inventory.ItemStonePickaxe)
	if harvested || drop != 0 {
		t.Fatalf("stone pickaxe cannot harvest diamond ore, got drop=%d, harvested=%v", drop, harvested)
	}

	// 4. Diamond Ore (56) with Iron Pickaxe (Tier 2): yields Diamond (264)
	eng.SetBlock(4, 64, 0, 56)
	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 4, Y: 64, Z: 0}, inventory.ItemIronPickaxe)
	if !harvested || drop != inventory.ItemDiamond {
		t.Fatalf("iron pickaxe must harvest diamond ore into diamond, got drop=%d, harvested=%v", drop, harvested)
	}

	// 5. Bedrock (7) cannot be broken
	eng.SetBlock(5, 64, 0, 7)
	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 5, Y: 64, Z: 0}, inventory.ItemDiamondPickaxe)
	if harvested || eng.GetBlock(5, 64, 0) != 7 {
		t.Fatalf("bedrock must be unbreakable")
	}

	// 6. Mining break speed calculations
	dpsDirtHand := inventory.CalculateBlockDamagePerSecond(inventory.ItemAir, inventory.ItemDirt)
	// Time = 1.0 / dpsDirtHand ~ 0.75 seconds
	timeDirtHand := 1.0 / dpsDirtHand
	if math.Abs(timeDirtHand-0.75) > 0.05 {
		t.Fatalf("dirt hand break time should be ~0.75s, got %f", timeDirtHand)
	}

	dpsStonePick := inventory.CalculateBlockDamagePerSecond(inventory.ItemWoodenPickaxe, inventory.ItemStone)
	// Time = 1.0 / dpsStonePick ~ 1.125 seconds
	timeStonePick := 1.0 / dpsStonePick
	if math.Abs(timeStonePick-1.125) > 0.05 {
		t.Fatalf("stone wooden pick break time should be ~1.125s, got %f", timeStonePick)
	}
}

func TestDropSpawningAndAutoRespawn(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Miner", [16]byte{5}, Vec3{X: 8, Y: 64, Z: 8})

	// 1. Mine a dirt block: verify physical ItemDrop entity is spawned
	eng.SetBlock(8, 64, 9, 3) // Dirt
	dropID, ok := eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 8, Y: 64, Z: 9}, inventory.ItemAir)
	if !ok || dropID != inventory.ItemDirt {
		t.Fatalf("expected dirt drop, got %d, %v", dropID, ok)
	}

	drops := eng.GetItemDrops()
	if len(drops) == 0 {
		t.Fatalf("expected 3D ItemDrop entity in world after mining block")
	}
	drop := drops[0]
	if drop.ItemID != inventory.ItemDirt {
		t.Fatalf("expected ItemDrop to be dirt, got %d", drop.ItemID)
	}

	// Fast forward past pickup delay (0.4s = 8 ticks)
	for i := 0; i < 9; i++ {
		eng.Tick()
	}

	// ItemDrop should now be picked up into player inventory
	remainingDrops := eng.GetItemDrops()
	if len(remainingDrops) != 0 {
		t.Fatalf("expected ItemDrop to be collected by player after pickup delay")
	}
	foundDirt := 0
	for _, s := range player.Inventory.Slots {
		if s.ItemID == inventory.ItemDirt {
			foundDirt += s.Count
		}
	}
	if foundDirt != 1 {
		t.Fatalf("expected 1 dirt in inventory, got %d", foundDirt)
	}

	// 2. Fatal damage: test automatic safe respawn prevention of void teleportation
	eng.DamagePlayer(player.ID, 100.0, "explosion")
	if player.IsAlive {
		t.Fatalf("player should be dead after 100 damage")
	}

	// Next client movement must auto-respawn and return valid world position, never {0, 0, 0}
	newPos, ok, _ := eng.MovePlayerRelative(player.ID, Vec3{X: 0, Y: 0, Z: 0}, 0, 0)
	if !ok || newPos.Y < 60.0 {
		t.Fatalf("expected safe auto-respawn position, got %+v, ok=%v", newPos, ok)
	}
	if !player.IsAlive || player.Health != 20.0 {
		t.Fatalf("player should be fully respawned with 20 HP")
	}
}

func TestFoodConsumptionAndRegeneration(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("HungrySteve", [16]byte{9}, Vec3{X: 8, Y: 64, Z: 8})
	player.Hunger = 10
	player.Health = 15.0

	// Give Cooked Porkchop (320)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemCookedPorkchop, Count: 2}

	// Consume one porkchop (+8 hunger -> 10 + 8 = 18)
	ok, newHunger := eng.ConsumeFood(player.ID, 0)
	if !ok || newHunger != 18 {
		t.Fatalf("expected food to be consumed and hunger restored to 18, got %v, %d", ok, newHunger)
	}
	if player.Inventory.Slots[0].Count != 1 {
		t.Fatalf("expected remaining count 1, got %d", player.Inventory.Slots[0].Count)
	}

	// Run 80 ticks: natural health regeneration should trigger when hunger >= 18 (+1.0 HP)
	for i := 0; i < 80; i++ {
		eng.Tick()
	}

	if player.Health != 16.0 {
		t.Fatalf("expected health to regenerate from 15.0 to 16.0 at tick 80, got %f", player.Health)
	}

	// Eat second porkchop (+8 hunger -> capped at 20)
	ok, newHunger = eng.ConsumeFood(player.ID, 0)
	if !ok || newHunger != 20 {
		t.Fatalf("expected hunger to cap at 20, got %v, %d", ok, newHunger)
	}

	// Stomach is full: further food consumption should be rejected
	player.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemApple, Count: 1}
	ok, _ = eng.ConsumeFood(player.ID, 1)
	if ok {
		t.Fatalf("should not be able to eat when hunger is already full at 20")
	}
}

func TestMobWanderingAndObstacleJump(t *testing.T) {
	eng := NewEngine()
	// Place flat floor at Y=64, with 1-block step at X=6, Y=65, Z=5
	for x := int32(0); x <= 10; x++ {
		for z := int32(0); z <= 10; z++ {
			eng.SetBlock(x, 64, z, 1) // Stone floor
		}
	}
	eng.SetBlock(6, 65, 5, 1) // 1-block elevation step obstacle

	cow := eng.SpawnMob(MobCow, Vec3{X: 5.5, Y: 65.0, Z: 5.5})
	cow.IsIdle = false
	cow.WanderTicks = 50
	cow.TargetYaw = 90.0 // heading in +X direction towards the step at X=6
	cow.Yaw = 90.0

	// Tick the cow simulation
	eng.Tick()

	// Since block at X=6, Y=65 is solid stone and Y=66 is air, cow must trigger 1-block jump impulse
	if cow.Velocity.Y <= 0 {
		t.Fatalf("expected mob to trigger 1-block jump over obstacle, got Y velocity: %f", cow.Velocity.Y)
	}

	if cow.WalkDistance <= 0 {
		t.Fatalf("expected mob to accumulate walk distance for leg swing, got: %f", cow.WalkDistance)
	}
}

func TestSheepAndChickenDrops(t *testing.T) {
	eng := NewEngine()
	sheep := eng.SpawnMob(MobSheep, Vec3{X: 10, Y: 65, Z: 10})
	chicken := eng.SpawnMob(MobChicken, Vec3{X: 12, Y: 65, Z: 12})

	dropsSheep, killedSheep := eng.DamageMob(sheep.ID, 20.0)
	if !killedSheep || len(dropsSheep) == 0 {
		t.Fatalf("expected sheep to be killed and drop wool and mutton, got %v", dropsSheep)
	}
	hasWool := false
	for _, d := range dropsSheep {
		if d.ItemID == 35 { // White wool
			hasWool = true
		}
	}
	if !hasWool {
		t.Fatalf("expected sheep drops to include White Wool (ID 35)")
	}

	dropsChicken, killedChicken := eng.DamageMob(chicken.ID, 10.0)
	if !killedChicken || len(dropsChicken) == 0 {
		t.Fatalf("expected chicken to be killed and drop feathers, got %v", dropsChicken)
	}
	hasFeather := false
	for _, d := range dropsChicken {
		if d.ItemID == 288 { // Feather
			hasFeather = true
		}
	}
	if !hasFeather {
		t.Fatalf("expected chicken drops to include Feathers (ID 288)")
	}
}

func TestItemPickupFullInventoryAndPartial(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("TestSteve", [16]byte{1}, Vec3{X: 10, Y: 65, Z: 10})

	// Fill all 36 slots completely with stone (stack of 64 each)
	for i := 0; i < len(player.Inventory.Slots); i++ {
		player.Inventory.Slots[i] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 64}
	}

	// Spawn a diamond drop near the player (within 1.5 blocks) with 0 pickup delay
	drop := eng.SpawnItemDropWithDelay(inventory.ItemDiamond, 5, Vec3{X: 10.5, Y: 65, Z: 10.5}, Vec3{}, 0.0)

	// Step simulation ticks
	for i := 0; i < 5; i++ {
		eng.Tick()
	}

	// Because inventory was completely full of stone, diamond drop must NOT be swallowed or deleted!
	eng.mu.RLock()
	d, exists := eng.ItemDrops[drop.ID]
	eng.mu.RUnlock()
	if !exists || d == nil {
		t.Fatalf("ground item drop was swallowed when inventory was full!")
	}
	if d.Count != 5 {
		t.Fatalf("expected ground drop count to remain 5, got %d", d.Count)
	}

	// Free up slot 0 but put 62 stone in it (so it has room for only 2 more stone)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 62}

	// Spawn a stone drop with 10 stone
	stoneDrop := eng.SpawnItemDropWithDelay(inventory.ItemStone, 10, Vec3{X: 10.2, Y: 65, Z: 10.2}, Vec3{}, 0.0)
	for i := 0; i < 5; i++ {
		eng.Tick()
	}

	// Slot 0 should now have 64 stone (+2 picked up)
	if player.Inventory.Slots[0].Count != 64 {
		t.Fatalf("expected slot 0 to fill to 64, got %d", player.Inventory.Slots[0].Count)
	}

	// Ground drop must have 8 remaining stone (10 - 2)
	eng.mu.RLock()
	sd, exists := eng.ItemDrops[stoneDrop.ID]
	eng.mu.RUnlock()
	if !exists || sd == nil {
		t.Fatalf("partial drop was completely deleted!")
	}
	if sd.Count != 8 {
		t.Fatalf("expected 8 stone remaining on ground, got %d", sd.Count)
	}
}

func TestEngineChunkPersistenceAndUnloadSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sim_save_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := world.NewSaveManager(tmpDir)
	if err != nil {
		t.Fatalf("NewSaveManager failed: %v", err)
	}
	defer sm.Close()

	eng := NewEngine()
	eng.SetStorage(sm)

	// Set a block (Diamond Block ID=57) at (5, 65, 5) which is in chunk (0, 0)
	eng.SetBlock(5, 65, 5, 57)
	if b := eng.GetBlock(5, 65, 5); b != 57 {
		t.Fatalf("expected block 57, got %d", b)
	}

	// Unload the chunk (simulating player walking away)
	cp := world.ChunkPos{X: 0, Z: 0}
	eng.UnloadChunk(cp)

	// Ensure chunk is no longer in memory
	eng.mu.RLock()
	_, inMem := eng.World[cp]
	eng.mu.RUnlock()
	if inMem {
		t.Fatalf("chunk should have been evicted from memory")
	}

	// Read block back: GetBlock should auto-load the chunk from region file!
	loadedBlock := eng.GetBlock(5, 65, 5)
	if loadedBlock != 57 {
		t.Fatalf("expected loaded block to be 57, got %d", loadedBlock)
	}

	// Test cross-process restart: create brand new Engine with the same save directory
	eng2 := NewEngine()
	eng2.SetStorage(sm)
	restartBlock := eng2.GetBlock(5, 65, 5)
	if restartBlock != 57 {
		t.Fatalf("expected block across engine restart to be 57, got %d", restartBlock)
	}
}

func TestMobHurtFlashDecay(t *testing.T) {
	eng := NewEngine()
	zombie := eng.SpawnMob(MobZombie, Vec3{X: 10, Y: 65, Z: 10})
	if zombie.HurtTime != 0 {
		t.Fatalf("new mob should start with HurtTime 0, got %d", zombie.HurtTime)
	}

	// Player attacks the zombie with 3.0 damage
	eng.DamageMob(zombie.ID, 3.0)
	if zombie.HurtTime != 10 {
		t.Fatalf("mob should have HurtTime 10 after taking damage, got %d", zombie.HurtTime)
	}
	if zombie.Health != 17.0 {
		t.Fatalf("zombie health should be 17.0, got %f", zombie.Health)
	}

	// Tick the engine for 5 ticks
	for i := 0; i < 5; i++ {
		eng.Tick()
	}
	if zombie.HurtTime != 5 {
		t.Fatalf("mob HurtTime should decay to 5 after 5 ticks, got %d", zombie.HurtTime)
	}

	// Tick 5 more ticks (total 10 ticks = 0.5s)
	for i := 0; i < 5; i++ {
		eng.Tick()
	}
	if zombie.HurtTime != 0 {
		t.Fatalf("mob HurtTime should decay to 0 and not stay red indefinitely, got %d", zombie.HurtTime)
	}

	// Test AttackMobInFront
	camPos := Vec3{X: zombie.Position.X - 1.5, Y: zombie.Position.Y + 0.9, Z: zombie.Position.Z}
	look := Vec3{X: 1, Y: 0, Z: 0}
	mobID, killed, hit := eng.AttackMobInFront(camPos, look, 3.0, 4.0)
	if !hit || killed || mobID != zombie.ID {
		t.Fatalf("expected hit on zombie, got hit=%v killed=%v mobID=%d", hit, killed, mobID)
	}
	if zombie.HurtTime != 10 {
		t.Fatalf("mob should reset HurtTime to 10 on new attack, got %d", zombie.HurtTime)
	}

	// Step 10 ticks again
	for i := 0; i < 10; i++ {
		eng.Tick()
	}
	if zombie.HurtTime != 0 {
		t.Fatalf("mob HurtTime should decay back to 0, got %d", zombie.HurtTime)
	}
}

func TestHostileMobLineOfSightThroughWall(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{X: 10, Y: 64, Z: 10})
	p.Health = 20.0

	// Build a 3-high solid stone wall between player and skeleton
	eng.SetBlock(10, 64, 11, 1) // Stone
	eng.SetBlock(10, 65, 11, 1) // Stone
	eng.SetBlock(10, 66, 11, 1) // Stone

	// Spawn skeleton on other side of wall
	skeleton := eng.SpawnMob(MobSkeleton, Vec3{X: 10, Y: 64, Z: 12})
	skeleton.AttackCooldown = 0

	// Tick AI - Skeleton should NOT be able to damage player through solid wall!
	skeleton.tickSkeletonAI(eng, p, 2.0)
	if p.Health < 20.0 {
		t.Fatalf("player took damage through solid wall from skeleton: health = %f", p.Health)
	}

	// Remove wall - now line of sight is clear
	eng.SetBlock(10, 64, 11, 0)
	eng.SetBlock(10, 65, 11, 0)
	eng.SetBlock(10, 66, 11, 0)

	skeleton.AttackCooldown = 0
	skeleton.tickSkeletonAI(eng, p, 2.0)
	if p.Health >= 20.0 {
		t.Fatalf("player should have taken damage after wall removed, health = %f", p.Health)
	}
}

func TestFallDamageResetInWater(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{2}, Vec3{X: 5, Y: 70, Z: 5})
	p.Health = 20.0
	p.FallDistance = 8.0 // Accumulated high fall

	// Place water block at player destination
	eng.SetBlock(5, 64, 5, 9) // Water block
	eng.SetBlock(5, 63, 5, 1) // Riverbed stone floor

	// Step down into water
	_, ground, _ := eng.MovePlayerRelative(p.ID, Vec3{X: 0, Y: -6.0, Z: 0}, 0, 0)
	_ = ground

	// Fall distance should be completely reset in water!
	if p.FallDistance != 0 {
		t.Fatalf("fall distance in water should be 0, got %f", p.FallDistance)
	}
	if p.Health != 20.0 {
		t.Fatalf("player took fall damage entering water: health = %f", p.Health)
	}
}

func TestVanillaHungerSaturationAndFastRegen(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{3}, Vec3{X: 0, Y: 64, Z: 0})
	p.Health = 15.0
	p.Hunger = 20
	p.Saturation = 5.0
	p.Exhaustion = 0

	// In 10 ticks, fast regen should restore 1 HP and consume saturation via exhaustion
	for i := 0; i < 10; i++ {
		eng.Tick()
	}

	if p.Health != 16.0 {
		t.Fatalf("fast natural regen expected health 16.0, got %f", p.Health)
	}

	// Step another 50 ticks to drain exhaustion
	for i := 0; i < 50; i++ {
		eng.Tick()
	}
	if p.Saturation >= 5.0 {
		t.Fatalf("expected saturation to decrease from fast regen exhaustion, got %f", p.Saturation)
	}
}

func TestEndermanStareAggroAndTeleport(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{4}, Vec3{X: 10, Y: 64, Z: 10})
	p.Yaw = 0   // looking south (+Z)
	p.Pitch = 0 // looking horizontal

	// Spawn Enderman directly in front along +Z
	enderman := eng.SpawnMob(MobEnderman, Vec3{X: 10, Y: 64, Z: 18})
	if enderman.IsAggro {
		t.Fatalf("enderman should start unprovoked")
	}

	// Tick mob AI: player is staring directly at enderman
	enderman.TickMob(eng)
	if !enderman.IsAggro {
		t.Fatalf("enderman should become aggressive when stared at")
	}

	// Test drops
	drops := enderman.DeathDrops()
	if len(drops) == 0 || drops[0].ItemID != 368 {
		t.Fatalf("expected ender pearl drop from enderman, got %v", drops)
	}
}

func TestEndDimensionBossAndCrystalExplosion(t *testing.T) {
	eng := NewEngine()
	eng.SwitchDimension(2) // The End

	if eng.GetDimension() != 2 {
		t.Fatalf("expected active dimension 2 (The End)")
	}

	dragon := eng.GetEnderDragon()
	if dragon == nil {
		t.Fatalf("expected Ender Dragon boss spawned in The End")
	}
	if dragon.Health != 200.0 {
		t.Fatalf("expected 200.0 HP for Ender Dragon, got %f", dragon.Health)
	}

	crystals := eng.GetEndCrystals()
	if len(crystals) != 8 {
		t.Fatalf("expected 8 End Crystals on towers, got %d", len(crystals))
	}

	// Tether dragon to crystal 1 and explode crystal
	eng.SetDragonHealingCrystal(crystals[0].ID)
	eng.AttackEndCrystal(crystals[0].ID)

	dragonAfter := eng.GetEnderDragon()
	if dragonAfter.Health >= 200.0 {
		t.Fatalf("exploding tethered End Crystal should damage dragon, got health %f", dragonAfter.Health)
	}
}

func TestEnderDragonDeathSettlementNoDeadlock(t *testing.T) {
	eng := NewEngine()
	eng.SwitchDimension(2) // The End

	dragon := eng.GetEnderDragon()
	if dragon == nil {
		t.Fatalf("expected Ender Dragon boss spawned in The End")
	}

	// Deal fatal damage
	killed := eng.DamageEnderDragon(200.0)
	if !killed {
		t.Fatalf("expected DamageEnderDragon to trigger death state")
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 110; i++ {
			eng.Tick()
		}
		close(done)
	}()

	select {
	case <-done:
		// Completed 110 ticks successfully without deadlocking!
	case <-time.After(2 * time.Second):
		t.Fatalf("FATAL: Engine.Tick deadlocked during Ender Dragon death settlement!")
	}

	finalDragon := eng.GetEnderDragon()
	if finalDragon.IsAlive || finalDragon.IsDying {
		t.Fatalf("expected dragon to be fully defeated after 110 death ticks, got alive=%v dying=%v", finalDragon.IsAlive, finalDragon.IsDying)
	}

	drops := eng.GetItemDrops()
	if len(drops) == 0 {
		t.Fatalf("expected celebration drops spawned after dragon death")
	}
}

func TestDimensionSwitchRetainsModifiedBlocks(t *testing.T) {
	eng := NewEngine()
	// 1. Overworld block placement
	eng.SetBlock(-85, 110, 25, 57) // Diamond block
	if b := eng.GetBlock(-85, 110, 25); b != 57 {
		t.Fatalf("expected diamond block 57 in Overworld, got %d", b)
	}

	// 2. Switch to Nether
	eng.SwitchDimension(1)
	if b := eng.GetBlock(-85, 110, 25); b == 57 {
		t.Fatalf("overworld block should not be present at nether coordinates")
	}
	eng.SetBlock(10, 66, 10, 89) // Glowstone
	if b := eng.GetBlock(10, 66, 10); b != 89 {
		t.Fatalf("expected glowstone 89 in Nether, got %d", b)
	}

	// 3. Switch back to Overworld
	eng.SwitchDimension(0)
	if b := eng.GetBlock(-85, 110, 25); b != 57 {
		t.Fatalf("re-entering Overworld lost placed block! Expected 57, got %d", b)
	}

	// 4. Switch back to Nether
	eng.SwitchDimension(1)
	if b := eng.GetBlock(10, 66, 10); b != 89 {
		t.Fatalf("re-entering Nether lost placed block! Expected 89, got %d", b)
	}
}

func TestFurnaceSmeltingAndDrop(t *testing.T) {
	eng := NewEngine()
	furnacePos := world.BlockPos{X: 10, Y: 64, Z: 10}

	// 1. Place Furnace (ID 61)
	eng.PlaceBlock(1, furnacePos, 61)
	if b := eng.GetBlock(10, 64, 10); b != 61 {
		t.Fatalf("expected furnace block at (10, 64, 10), got %d", b)
	}

	furnace := eng.GetOrCreateFurnace(furnacePos)
	if furnace == nil {
		t.Fatalf("expected furnace entity created")
	}

	// 2. Add Coal (263) and Iron Ore (15)
	furnace.FuelSlot = inventory.ItemStack{ItemID: inventory.ItemCoal, Count: 1}
	furnace.InputSlot = inventory.ItemStack{ItemID: 15, Count: 1} // Iron Ore

	// 3. Run ticks until smelting completes (200 ticks = 1 item)
	for i := 0; i < 205; i++ {
		eng.Tick()
	}

	// Verify OutputSlot has Iron Ingot (ID 265)
	if furnace.OutputSlot.ItemID != inventory.ItemIronIngot || furnace.OutputSlot.Count != 1 {
		t.Fatalf("expected 1 Iron Ingot smelted in furnace, got ItemID=%d Count=%d", furnace.OutputSlot.ItemID, furnace.OutputSlot.Count)
	}

	// 4. Break furnace with pickaxe (tool ID 270: Wooden Pickaxe)
	dropItem, ok := eng.BreakBlockWithTool(1, furnacePos, 270)
	if !ok || dropItem != 61 {
		t.Fatalf("expected furnace block to drop when mined, got drop=%d, ok=%v", dropItem, ok)
	}

	// Verify furnace dropped its smelted iron ingot
	drops := eng.GetItemDrops()
	foundIngot := false
	for _, d := range drops {
		if d.ItemID == inventory.ItemIronIngot {
			foundIngot = true
			break
		}
	}
	if !foundIngot {
		t.Fatalf("expected furnace contents (iron ingot) dropped when broken")
	}
}
