package sim

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// MobType categorizes Minecraft entities.
type MobType int

const (
	MobZombie MobType = iota
	MobSkeleton
	MobCreeper
	MobCow
	MobPig
)

func (m MobType) String() string {
	switch m {
	case MobZombie:
		return "Zombie"
	case MobSkeleton:
		return "Skeleton"
	case MobCreeper:
		return "Creeper"
	case MobCow:
		return "Cow"
	case MobPig:
		return "Pig"
	default:
		return "Unknown"
	}
}

// MobEntity represents an active biological entity in the simulation.
type MobEntity struct {
	ID             int32
	Type           MobType
	Position       Vec3
	Velocity       Vec3
	Yaw            float32
	Pitch          float32
	Health         float32
	MaxHealth      float32
	TargetPlayerID int32
	AttackCooldown int
	FuseTicks      int // Creeper ignition timer
	IsAlive        bool
}

// NewMobEntity creates an initialized mob.
func NewMobEntity(id int32, mobType MobType, pos Vec3) *MobEntity {
	maxHP := float32(20.0)
	if mobType == MobCow || mobType == MobPig {
		maxHP = 10.0
	}
	return &MobEntity{
		ID:        id,
		Type:      mobType,
		Position:  pos,
		Health:    maxHP,
		MaxHealth: maxHP,
		IsAlive:   true,
	}
}

// TickMob processes one simulation tick (20 TPS) for mob AI and movement.
func (m *MobEntity) TickMob(engine *Engine) {
	if !m.IsAlive {
		return
	}

	if m.AttackCooldown > 0 {
		m.AttackCooldown--
	}

	// 1. Gravity simulation
	footX := int32(math.Floor(m.Position.X))
	footY := int32(math.Floor(m.Position.Y - 0.1))
	footZ := int32(math.Floor(m.Position.Z))
	onGround := engine.GetBlock(footX, footY, footZ) != 0 || m.Position.Y <= 64.0
	if !onGround {
		m.Velocity.Y -= 0.08 // Minecraft gravity
		if m.Velocity.Y < -3.92 {
			m.Velocity.Y = -3.92
		}
	} else if m.Velocity.Y < 0 {
		m.Velocity.Y = 0
	}

	// 2. AI Target Acquisition (Find closest player within 16 blocks)
	var closestPlayer *PlayerEntity
	closestDistSq := 16.0 * 16.0

	for _, p := range engine.Players {
		if !p.IsAlive {
			continue
		}
		dx := p.Position.X - m.Position.X
		dy := p.Position.Y - m.Position.Y
		dz := p.Position.Z - m.Position.Z
		distSq := dx*dx + dy*dy + dz*dz
		if distSq < closestDistSq {
			closestDistSq = distSq
			closestPlayer = p
		}
	}

	// 3. AI Behavior Dispatch
	switch m.Type {
	case MobZombie:
		m.tickZombieAI(engine, closestPlayer, math.Sqrt(closestDistSq))
	case MobSkeleton:
		m.tickSkeletonAI(engine, closestPlayer, math.Sqrt(closestDistSq))
	case MobCreeper:
		m.tickCreeperAI(engine, closestPlayer, math.Sqrt(closestDistSq))
	case MobCow, MobPig:
		m.tickPassiveAI(engine)
	}

	// 4. Apply velocity
	m.Position = m.Position.Add(m.Velocity)
	m.Velocity.X *= 0.546 // Air / ground drag
	m.Velocity.Z *= 0.546
}

func (m *MobEntity) tickZombieAI(engine *Engine, player *PlayerEntity, dist float64) {
	if player == nil || dist > 16.0 {
		return
	}

	// Face player
	dx := player.Position.X - m.Position.X
	dz := player.Position.Z - m.Position.Z
	m.Yaw = float32(math.Atan2(-dx, dz) * 180.0 / math.Pi)

	// Move toward player
	speed := 0.12
	m.Velocity.X = (dx / dist) * speed
	m.Velocity.Z = (dz / dist) * speed

	// Melee attack: within 1.5 blocks
	if dist <= 1.5 && m.AttackCooldown <= 0 {
		engine.DamagePlayer(player.ID, 3.0, "zombie")
		m.AttackCooldown = 20 // 1 attack per second
	}
}

func (m *MobEntity) tickSkeletonAI(engine *Engine, player *PlayerEntity, dist float64) {
	if player == nil || dist > 16.0 {
		return
	}

	dx := player.Position.X - m.Position.X
	dz := player.Position.Z - m.Position.Z
	m.Yaw = float32(math.Atan2(-dx, dz) * 180.0 / math.Pi)

	// Skeletons strafe and maintain ~8 blocks range
	if dist < 6.0 {
		// Back up
		m.Velocity.X = -(dx / dist) * 0.1
		m.Velocity.Z = -(dz / dist) * 0.1
	} else if dist > 10.0 {
		// Approach
		m.Velocity.X = (dx / dist) * 0.1
		m.Velocity.Z = (dz / dist) * 0.1
	}

	// Shoot arrow every 30 ticks (1.5 seconds)
	if m.AttackCooldown <= 0 {
		engine.DamagePlayer(player.ID, 4.0, "skeleton_arrow")
		m.AttackCooldown = 30
	}
}

func (m *MobEntity) tickCreeperAI(engine *Engine, player *PlayerEntity, dist float64) {
	if player == nil || dist > 16.0 {
		m.FuseTicks = 0
		return
	}

	dx := player.Position.X - m.Position.X
	dz := player.Position.Z - m.Position.Z
	m.Yaw = float32(math.Atan2(-dx, dz) * 180.0 / math.Pi)

	if dist > 3.0 {
		// Approach player
		m.Velocity.X = (dx / dist) * 0.14
		m.Velocity.Z = (dz / dist) * 0.14
		if m.FuseTicks > 0 {
			m.FuseTicks--
		}
	} else {
		// Stop and charge fuse! (30 ticks = 1.5s countdown)
		m.Velocity.X = 0
		m.Velocity.Z = 0
		m.FuseTicks++
		if m.FuseTicks >= 30 {
			// BOOM! Explode
			m.IsAlive = false
			m.Health = 0
			// Explode in 3.0 block radius, dealing damage and breaking blocks
			engine.DamagePlayer(player.ID, 20.0, "explosion")
			cx := int32(math.Floor(m.Position.X))
			cy := int32(math.Floor(m.Position.Y))
			cz := int32(math.Floor(m.Position.Z))
			for ex := int32(-2); ex <= 2; ex++ {
				for ey := int32(-1); ey <= 2; ey++ {
					for ez := int32(-2); ez <= 2; ez++ {
						if ex*ex+ey*ey+ez*ez <= 5 {
							engine.SetBlock(cx+ex, cy+ey, cz+ez, 0)
						}
					}
				}
			}
		}
	}
}

func (m *MobEntity) tickPassiveAI(engine *Engine) {
	// Passive wandering motion
	if m.AttackCooldown <= 0 {
		m.AttackCooldown = 40 + int(m.ID*7)%30
		angle := float64(m.ID*37%360) * math.Pi / 180.0
		m.Velocity.X = math.Cos(angle) * 0.05
		m.Velocity.Z = math.Sin(angle) * 0.05
		m.Yaw = float32(angle * 180.0 / math.Pi)
	}
}

// DeathDrops generates authentic Minecraft loot when mob is slain.
func (m *MobEntity) DeathDrops() []inventory.ItemStack {
	switch m.Type {
	case MobZombie:
		return []inventory.ItemStack{
			{ItemID: 367, Count: 1}, // Rotten Flesh
		}
	case MobSkeleton:
		return []inventory.ItemStack{
			{ItemID: 352, Count: 2}, // Bones
			{ItemID: 262, Count: 2}, // Arrows
		}
	case MobCreeper:
		return []inventory.ItemStack{
			{ItemID: 289, Count: 1}, // Gunpowder
		}
	case MobCow:
		return []inventory.ItemStack{
			{ItemID: 363, Count: 2}, // Raw Beef
			{ItemID: 334, Count: 1}, // Leather
		}
	case MobPig:
		return []inventory.ItemStack{
			{ItemID: 319, Count: 2}, // Raw Porkchop
		}
	default:
		return nil
	}
}
