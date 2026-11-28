package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestAttackMobsAlongRayUsesRangeAndPiercesMultipleTargets(t *testing.T) {
	engine := NewEngine()
	tooClose := engine.SpawnMob(MobZombie, Vec3{X: 1.8})
	first := engine.SpawnMob(MobZombie, Vec3{X: 2.5})
	second := engine.SpawnMob(MobCow, Vec3{X: 3.5})
	tooFar := engine.SpawnMob(MobZombie, Vec3{X: 4.7})
	offAxis := engine.SpawnMob(MobZombie, Vec3{X: 3, Z: 0.5})

	hits := engine.AttackMobsAlongRay(Vec3{Y: 0.9}, Vec3{X: 1}, 2, 4.5, 0.125, 2)
	if len(hits) != 2 {
		t.Fatalf("piercing attack hit %d mobs, want 2: %+v", len(hits), hits)
	}
	if hits[0].ID != first.ID || hits[1].ID != second.ID {
		t.Fatalf("hits were not returned front-to-back: %+v", hits)
	}
	for _, mob := range []*MobEntity{first, second} {
		if mob.Health != mob.MaxHealth-2 || mob.HurtTime != 10 {
			t.Errorf("hit mob state = health %.1f hurt %d, want %.1f and 10", mob.Health, mob.HurtTime, mob.MaxHealth-2)
		}
	}
	for _, mob := range []*MobEntity{tooClose, tooFar, offAxis} {
		if mob.Health != mob.MaxHealth || mob.HurtTime != 0 {
			t.Errorf("out-of-ray mob %d was hit: health %.1f hurt %d", mob.ID, mob.Health, mob.HurtTime)
		}
	}
}

func TestSpearChargeUsesRelativeSpeedAndKnockbackThreshold(t *testing.T) {
	properties, ok := inventory.GetSpearProperties(inventory.ItemWoodenSpear)
	if !ok {
		t.Fatal("wooden spear properties are missing")
	}

	newTarget := func() (*Engine, *MobEntity) {
		engine := NewEngine()
		mob := engine.SpawnMob(MobZombie, Vec3{X: 3})
		return engine, mob
	}
	origin := Vec3{Y: 0.9}
	look := Vec3{X: 1}

	engine, mob := newTarget()
	if hits := engine.AttackMobsWithSpearCharge(origin, look, Vec3{X: 4.5}, properties, 0); len(hits) != 0 {
		t.Fatalf("charge below the 4.6 relative-speed threshold hit: %+v", hits)
	}
	if mob.Health != mob.MaxHealth {
		t.Fatalf("below-threshold charge changed target health to %.1f", mob.Health)
	}

	engine, mob = newTarget()
	hits := engine.AttackMobsWithSpearCharge(origin, look, Vec3{X: 5.0}, properties, 0)
	if len(hits) != 1 || mob.Health != mob.MaxHealth-1 {
		t.Fatalf("valid low-speed charge hits=%+v health=%.1f, want one hit for 1", hits, mob.Health)
	}
	if mob.Velocity.X != 0 || mob.Velocity.Y != 0 || mob.Velocity.Z != 0 {
		t.Fatalf("charge below knockback threshold applied knockback: %+v", mob.Velocity)
	}

	engine, mob = newTarget()
	hits = engine.AttackMobsWithSpearCharge(origin, look, Vec3{X: 5.2}, properties, 0)
	if len(hits) != 1 || mob.Velocity.X <= 0 {
		t.Fatalf("charge above knockback threshold did not knock the target forward: hits=%+v velocity=%+v", hits, mob.Velocity)
	}
}

func TestSpearChargeExpiresAfterMaterialDamageWindow(t *testing.T) {
	properties, _ := inventory.GetSpearProperties(inventory.ItemIronSpear)
	engine := NewEngine()
	mob := engine.SpawnMob(MobZombie, Vec3{X: 3})
	hits := engine.AttackMobsWithSpearCharge(Vec3{Y: 0.9}, Vec3{X: 1}, Vec3{X: 20}, properties, properties.DamageWindowTicks+1)
	if len(hits) != 0 || mob.Health != mob.MaxHealth {
		t.Fatalf("expired spear charge hit target: hits=%+v health=%.1f", hits, mob.Health)
	}
}
