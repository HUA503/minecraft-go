package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestThrownTridentDamagesMobAndDropsWornStack(t *testing.T) {
	engine := NewEngine()
	engine.SetWorldTime(18000)
	mob := engine.SpawnMob(MobZombie, Vec3{Y: 64, Z: 3})
	engine.SpawnTridentProjectile(1, inventory.ItemStack{ItemID: inventory.ItemTrident, Count: 1, Damage: 27}, Vec3{Y: 65}, Vec3{Z: 3}, true)
	engine.Tick()
	if mob.Health != mob.MaxHealth-8 {
		t.Fatalf("trident damage left zombie at %.1f health, want %.1f", mob.Health, mob.MaxHealth-8)
	}
	if len(engine.GetTridentProjectiles()) != 0 {
		t.Fatal("trident projectile remained active after hitting a mob")
	}
	drops := engine.GetItemDrops()
	var returned bool
	for _, drop := range drops {
		if drop.ItemID == inventory.ItemTrident && drop.Count == 1 && drop.Damage == 27 {
			returned = true
		}
	}
	if !returned {
		t.Fatalf("trident impact did not create a worn recoverable item: %+v", drops)
	}
}

func TestThrownTridentStopsAtBlockAndCanBePickedUpWithDamage(t *testing.T) {
	engine := NewEngine()
	engine.SetWorldTime(18000)
	for z := int32(0); z <= 2; z++ {
		engine.SetBlock(0, 63, z, inventory.ItemStone)
	}
	engine.SetBlock(0, 65, 2, inventory.ItemStone)
	inv := inventory.NewPlayerInventory()
	engine.Players[1] = &PlayerEntity{ID: 1, Position: Vec3{Y: 64, Z: 2}, IsAlive: true, Inventory: inv}
	engine.SpawnTridentProjectile(1, inventory.ItemStack{ItemID: inventory.ItemTrident, Count: 1, Damage: 41}, Vec3{Y: 65, Z: 0}, Vec3{Z: 3}, true)
	engine.Tick()
	if len(engine.GetTridentProjectiles()) != 0 {
		t.Fatal("trident passed through a solid block")
	}
	for i := 0; i < 8; i++ {
		engine.Tick()
	}
	if got := inv.Slots[0]; got.ItemID != inventory.ItemTrident || got.Damage != 41 {
		t.Fatalf("picked-up trident = %+v, want original damage 41; drops=%+v players=%+v", got, engine.GetItemDrops(), engine.GetPlayerList())
	}
}

func TestTridentProjectileRejectsInvalidLaunch(t *testing.T) {
	engine := NewEngine()
	if got := engine.SpawnTridentProjectile(1, inventory.ItemStack{ItemID: inventory.ItemStone, Count: 1}, Vec3{}, Vec3{Z: 1}, true); got != nil {
		t.Fatalf("non-trident launch created projectile: %+v", got)
	}
	if got := engine.SpawnTridentProjectile(1, inventory.ItemStack{ItemID: inventory.ItemTrident, Count: 1}, Vec3{}, Vec3{}, true); got != nil {
		t.Fatalf("zero-velocity launch created projectile: %+v", got)
	}
}

func TestSegmentIntersectsAABBCoversHitAndMiss(t *testing.T) {
	min, max := Vec3{X: 2, Y: 0, Z: -1}, Vec3{X: 3, Y: 2, Z: 1}
	if tHit, hit := segmentIntersectsAABB(Vec3{}, Vec3{X: 4}, min, max); !hit || tHit < 0.49 || tHit > 0.51 {
		t.Fatalf("segment AABB hit = (%.3f, %t), want entry at 0.5", tHit, hit)
	}
	if _, hit := segmentIntersectsAABB(Vec3{}, Vec3{X: 1, Z: 4}, min, max); hit {
		t.Fatal("segment outside AABB reported a hit")
	}
}
