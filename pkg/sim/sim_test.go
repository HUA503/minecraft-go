package sim

import (
	"testing"

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
