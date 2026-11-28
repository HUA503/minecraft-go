package sim

import (
	"math"
	"sort"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// TridentProjectile is a thrown, recoverable trident. Velocity is measured in
// blocks per simulation tick, matching the engine's 20 TPS update cadence.
type TridentProjectile struct {
	ID          int32
	OwnerID     int32
	Stack       inventory.ItemStack
	Recoverable bool
	Position    Vec3
	Velocity    Vec3
	AgeTicks    int
}

// SpawnTridentProjectile creates a recoverable trident projectile with its
// current durability preserved through the eventual item drop.
func (e *Engine) SpawnTridentProjectile(ownerID int32, stack inventory.ItemStack, position, velocity Vec3, recoverable bool) *TridentProjectile {
	if stack.ItemID != inventory.ItemTrident || stack.Count <= 0 || velocity.Length() == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.TridentProjectiles == nil {
		e.TridentProjectiles = make(map[int32]*TridentProjectile)
	}
	id := e.nextTridentID
	e.nextTridentID++
	projectile := &TridentProjectile{
		ID: id, OwnerID: ownerID,
		Stack:       inventory.ItemStack{ItemID: stack.ItemID, Count: 1, Damage: stack.Damage},
		Recoverable: recoverable,
		Position:    position, Velocity: velocity,
	}
	e.TridentProjectiles[id] = projectile
	return projectile
}

// GetTridentProjectiles returns stable value snapshots for rendering.
func (e *Engine) GetTridentProjectiles() []TridentProjectile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ids := make([]int, 0, len(e.TridentProjectiles))
	for id := range e.TridentProjectiles {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	projectiles := make([]TridentProjectile, 0, len(ids))
	for _, id := range ids {
		if projectile := e.TridentProjectiles[int32(id)]; projectile != nil {
			projectiles = append(projectiles, *projectile)
		}
	}
	return projectiles
}

func (e *Engine) tickTridentProjectilesLocked(getBlock func(int32, int32, int32) int32) {
	if e.TridentProjectiles == nil {
		return
	}
	isSolid := func(x, y, z int32) bool { return IsSolidBlock(getBlock(x, y, z)) }
	for id, projectile := range e.TridentProjectiles {
		if projectile == nil {
			delete(e.TridentProjectiles, id)
			continue
		}
		projectile.AgeTicks++
		if projectile.AgeTicks > 6000 {
			delete(e.TridentProjectiles, id)
			continue
		}
		velocity := projectile.Velocity
		speed := velocity.Length()
		if speed <= 1e-9 {
			if projectile.Recoverable {
				e.spawnItemDropStackLocked(projectile.Stack, projectile.Position, Vec3{}, 0.2)
			}
			delete(e.TridentProjectiles, id)
			continue
		}
		start := projectile.Position
		end := start.Add(velocity)
		look := velocity.Scale(1 / speed)
		blockHit := RaycastBlocks(start, look, speed, getBlock)
		nearestT := 2.0
		var hitMob *MobEntity
		for _, mob := range e.Mobs {
			if mob == nil || !mob.IsAlive {
				continue
			}
			height := mobProjectileHeight(mob.Type)
			min := Vec3{X: mob.Position.X - 0.35, Y: mob.Position.Y, Z: mob.Position.Z - 0.35}
			max := Vec3{X: mob.Position.X + 0.35, Y: mob.Position.Y + height, Z: mob.Position.Z + 0.35}
			t, intersects := segmentIntersectsAABB(start, end, min, max)
			if intersects && t < nearestT {
				nearestT, hitMob = t, mob
			}
		}
		if hitMob != nil && (!blockHit.Hit || nearestT*speed < blockHit.Distance) {
			projectile.Position = start.Add(velocity.Scale(nearestT))
			if hitMob.HurtTime <= 5 {
				hitMob.Health -= 8
				hitMob.HurtTime = 10
				hitMob.LastHurtTime = time.Now()
				hitMob.Velocity = hitMob.Velocity.Add(look.Scale(0.3)).Add(Vec3{Y: 0.1})
				if hitMob.Health <= 0 {
					hitMob.Health = 0
					hitMob.IsAlive = false
					for _, stack := range hitMob.DeathDrops() {
						e.spawnItemDropStackLocked(stack, hitMob.Position.Add(Vec3{Y: 0.5}), Vec3{Y: 0.15}, 0.2)
					}
					delete(e.Mobs, hitMob.ID)
				}
			}
			if projectile.Recoverable {
				e.spawnItemDropStackLocked(projectile.Stack, projectile.Position, Vec3{Y: 0.08}, 0.25)
			}
			delete(e.TridentProjectiles, id)
			continue
		}
		if blockHit.Hit {
			projectile.Position = blockHit.HitPos
			if projectile.Recoverable {
				e.spawnItemDropStackLocked(projectile.Stack, projectile.Position, Vec3{}, 0.25)
			}
			delete(e.TridentProjectiles, id)
			continue
		}
		projectile.Position = end
		projectile.Velocity = Vec3{X: velocity.X * 0.99, Y: velocity.Y*0.99 - 0.05, Z: velocity.Z * 0.99}
		if isSolid(int32(math.Floor(end.X)), int32(math.Floor(end.Y)), int32(math.Floor(end.Z))) {
			if projectile.Recoverable {
				e.spawnItemDropStackLocked(projectile.Stack, end, Vec3{}, 0.25)
			}
			delete(e.TridentProjectiles, id)
		}
	}
}

func mobProjectileHeight(mobType MobType) float64 {
	switch mobType {
	case MobEnderman:
		return 2.9
	case MobSpider:
		return 0.9
	case MobChicken:
		return 0.7
	case MobCow, MobPig, MobSheep:
		return 1.4
	default:
		return 1.8
	}
}

func segmentIntersectsAABB(start, end, min, max Vec3) (float64, bool) {
	delta := end.Sub(start)
	enter, leave := 0.0, 1.0
	starts := [3]float64{start.X, start.Y, start.Z}
	deltas := [3]float64{delta.X, delta.Y, delta.Z}
	mins := [3]float64{min.X, min.Y, min.Z}
	maxs := [3]float64{max.X, max.Y, max.Z}
	for axis := 0; axis < 3; axis++ {
		if math.Abs(deltas[axis]) < 1e-12 {
			if starts[axis] < mins[axis] || starts[axis] > maxs[axis] {
				return 0, false
			}
			continue
		}
		t0 := (mins[axis] - starts[axis]) / deltas[axis]
		t1 := (maxs[axis] - starts[axis]) / deltas[axis]
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		enter = math.Max(enter, t0)
		leave = math.Min(leave, t1)
		if enter > leave {
			return 0, false
		}
	}
	return enter, enter >= 0 && enter <= 1
}
