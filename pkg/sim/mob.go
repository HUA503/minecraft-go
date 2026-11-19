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
	MobSheep
	MobChicken
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
	case MobSheep:
		return "Sheep"
	case MobChicken:
		return "Chicken"
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

	// Dynamic authentic AI & Animation states
	WalkDistance float64 // Continuous walking distance for leg-swing animations
	IsMoving     bool
	WanderTicks  int     // Countdown for current wander or idle action
	IsIdle       bool    // Idle/grazing vs active walking
	TargetYaw    float32 // Target heading towards which animal smoothly steers
}

// NewMobEntity creates an initialized mob.
func NewMobEntity(id int32, mobType MobType, pos Vec3) *MobEntity {
	maxHP := float32(20.0)
	if mobType == MobCow || mobType == MobPig {
		maxHP = 10.0
	} else if mobType == MobSheep {
		maxHP = 8.0
	} else if mobType == MobChicken {
		maxHP = 4.0
	}
	return &MobEntity{
		ID:        id,
		Type:      mobType,
		Position:  pos,
		Health:    maxHP,
		MaxHealth: maxHP,
		IsAlive:   true,
		IsIdle:    true,
		WanderTicks: 20 + int(id*11)%40,
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
	case MobCow, MobPig, MobSheep, MobChicken:
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
	m.WalkDistance += speed
	m.IsMoving = true

	// Hill climbing / 1-block step jump AI
	aheadX := int32(math.Floor(m.Position.X + (dx/dist)*0.6))
	footY := int32(math.Floor(m.Position.Y + 0.1))
	aheadZ := int32(math.Floor(m.Position.Z + (dz/dist)*0.6))
	blockAhead := engine.GetBlock(aheadX, footY, aheadZ)
	blockAbove1 := engine.GetBlock(aheadX, footY+1, aheadZ)
	if IsSolidBlock(blockAhead) && !IsSolidBlock(blockAbove1) && m.Velocity.Y <= 0 {
		m.Velocity.Y = 0.38
	}

	// Melee attack: within 1.5 blocks
	if dist <= 1.5 && m.AttackCooldown <= 0 {
		engine.DamagePlayer(player.ID, 3.0, "zombie")
		m.AttackCooldown = 20 // 1 attack per second
	}
}

func (m *MobEntity) tickSkeletonAI(engine *Engine, player *PlayerEntity, dist float64) {
	if player == nil || dist > 16.0 {
		m.IsMoving = false
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
		m.WalkDistance += 0.1
		m.IsMoving = true
	} else if dist > 10.0 {
		// Approach
		m.Velocity.X = (dx / dist) * 0.1
		m.Velocity.Z = (dz / dist) * 0.1
		m.WalkDistance += 0.1
		m.IsMoving = true
	} else {
		m.IsMoving = false
	}

	// Hill climbing / 1-block step jump AI
	aheadX := int32(math.Floor(m.Position.X + m.Velocity.X*3.0))
	footY := int32(math.Floor(m.Position.Y + 0.1))
	aheadZ := int32(math.Floor(m.Position.Z + m.Velocity.Z*3.0))
	blockAhead := engine.GetBlock(aheadX, footY, aheadZ)
	blockAbove1 := engine.GetBlock(aheadX, footY+1, aheadZ)
	if IsSolidBlock(blockAhead) && !IsSolidBlock(blockAbove1) && m.Velocity.Y <= 0 {
		m.Velocity.Y = 0.38
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
		m.IsMoving = false
		return
	}

	dx := player.Position.X - m.Position.X
	dz := player.Position.Z - m.Position.Z
	m.Yaw = float32(math.Atan2(-dx, dz) * 180.0 / math.Pi)

	if dist > 3.0 {
		// Approach player
		m.Velocity.X = (dx / dist) * 0.14
		m.Velocity.Z = (dz / dist) * 0.14
		m.WalkDistance += 0.14
		m.IsMoving = true

		// Hill climbing / 1-block step jump AI
		aheadX := int32(math.Floor(m.Position.X + (dx/dist)*0.6))
		footY := int32(math.Floor(m.Position.Y + 0.1))
		aheadZ := int32(math.Floor(m.Position.Z + (dz/dist)*0.6))
		blockAhead := engine.GetBlock(aheadX, footY, aheadZ)
		blockAbove1 := engine.GetBlock(aheadX, footY+1, aheadZ)
		if IsSolidBlock(blockAhead) && !IsSolidBlock(blockAbove1) && m.Velocity.Y <= 0 {
			m.Velocity.Y = 0.38
		}

		if m.FuseTicks > 0 {
			m.FuseTicks--
		}
	} else {
		// Stop and charge fuse! (30 ticks = 1.5s countdown)
		m.Velocity.X = 0
		m.Velocity.Z = 0
		m.IsMoving = false
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
	// 1. Check if nearby player exists to look at
	var nearbyPlayer *PlayerEntity
	for _, p := range engine.Players {
		if p.IsAlive {
			dx := p.Position.X - m.Position.X
			dz := p.Position.Z - m.Position.Z
			if dx*dx+dz*dz < 25.0 {
				nearbyPlayer = p
				break
			}
		}
	}

	if m.WanderTicks <= 0 {
		// Switch between Idle (40-90 ticks) and Walking (50-120 ticks)
		m.IsIdle = !m.IsIdle
		if m.IsIdle {
			m.WanderTicks = 40 + int(m.ID*17)%50
			m.IsMoving = false
			m.Velocity.X = 0
			m.Velocity.Z = 0
			if m.Type == MobCow || m.Type == MobSheep {
				m.Pitch = 25.0 // Head down to graze grass
			}
		} else {
			m.WanderTicks = 50 + int(m.ID*31)%70
			m.IsMoving = true
			m.Pitch = 0
			// Pick a new random wandering target angle
			m.TargetYaw = float32((int(m.ID*47) + m.WanderTicks*13) % 360)
		}
	} else {
		m.WanderTicks--
	}

	if nearbyPlayer != nil && m.IsIdle {
		// Look at nearby player
		dx := nearbyPlayer.Position.X - m.Position.X
		dz := nearbyPlayer.Position.Z - m.Position.Z
		m.Yaw = float32(math.Atan2(dx, dz) * 180.0 / math.Pi)
		m.Pitch = 0
		return
	}

	if m.IsIdle {
		m.Velocity.X = 0
		m.Velocity.Z = 0
		return
	}

	// Active wandering: smoothly interpolate Yaw towards TargetYaw
	diffYaw := m.TargetYaw - m.Yaw
	for diffYaw < -180 {
		diffYaw += 360
	}
	for diffYaw > 180 {
		diffYaw -= 360
	}
	if diffYaw > 8.0 {
		m.Yaw += 8.0
	} else if diffYaw < -8.0 {
		m.Yaw -= 8.0
	} else {
		m.Yaw = m.TargetYaw
	}

	rad := float64(m.Yaw) * math.Pi / 180.0
	fwdX := math.Sin(rad)
	fwdZ := math.Cos(rad)

	speed := 0.05
	if m.Type == MobChicken {
		speed = 0.04
	}
	m.Velocity.X = fwdX * speed
	m.Velocity.Z = fwdZ * speed
	m.WalkDistance += speed

	// Hill climbing / 1-block step jump AI
	aheadX := int32(math.Floor(m.Position.X + fwdX*0.8))
	footY := int32(math.Floor(m.Position.Y + 0.1))
	aheadZ := int32(math.Floor(m.Position.Z + fwdZ*0.8))

	blockAhead := engine.GetBlock(aheadX, footY, aheadZ)
	blockAbove1 := engine.GetBlock(aheadX, footY+1, aheadZ)
	blockAbove2 := engine.GetBlock(aheadX, footY+2, aheadZ)

	// If block ahead is solid and above is open air: jump up the 1-block hill!
	if IsSolidBlock(blockAhead) && !IsSolidBlock(blockAbove1) && !IsSolidBlock(blockAbove2) {
		if m.Velocity.Y <= 0 {
			m.Velocity.Y = 0.38 // Standard 1-block jump
		}
	} else if IsSolidBlock(blockAhead) && IsSolidBlock(blockAbove1) {
		// Blocked by a high wall: turn around!
		m.TargetYaw = float32(int(m.Yaw+120+float32(m.ID%60)) % 360)
		m.WanderTicks = 40
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
	case MobSheep:
		return []inventory.ItemStack{
			{ItemID: 35, Count: 1},  // White Wool
			{ItemID: 363, Count: 2}, // Raw Mutton / Beef
		}
	case MobChicken:
		return []inventory.ItemStack{
			{ItemID: 288, Count: 2}, // Feathers
			{ItemID: 363, Count: 1}, // Raw Chicken Meat
		}
	default:
		return nil
	}
}
