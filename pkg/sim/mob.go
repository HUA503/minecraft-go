package sim

import (
	"math"
	"time"

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
	MobEnderman
	MobSpider
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
	case MobEnderman:
		return "Enderman"
	case MobSpider:
		return "Spider"
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
	WalkDistance  float64 // Continuous walking distance for leg-swing animations
	IsMoving      bool
	WanderTicks   int       // Countdown for current wander or idle action
	IsIdle        bool      // Idle/grazing vs active walking
	TargetYaw     float32   // Target heading towards which animal smoothly steers
	HurtTime      int       // Authentic Minecraft hurt flash timer in ticks (10 ticks = 0.5s)
	LastHurtTime  time.Time // Real-time timestamp of damage to guarantee flash duration resets
	BurningTicks  int       // Daylight / fire burning tick counter
	IsAggro       bool      // Hostile aggro state (e.g. Enderman stared at)
	TeleportTimer int       // Enderman teleport cooldown
}

// IsHurt returns whether the mob is currently in the authentic 0.5s hurt flash window.
// Guaranteed to return false and reset HurtTime once 500ms has elapsed.
func (m *MobEntity) IsHurt() bool {
	if m.HurtTime <= 0 {
		return false
	}
	if !m.LastHurtTime.IsZero() && time.Since(m.LastHurtTime) > 500*time.Millisecond {
		m.HurtTime = 0
		return false
	}
	return true
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
	} else if mobType == MobEnderman {
		maxHP = 40.0
	} else if mobType == MobSpider {
		maxHP = 16.0
	}
	return &MobEntity{
		ID:          id,
		Type:        mobType,
		Position:    pos,
		Health:      maxHP,
		MaxHealth:   maxHP,
		IsAlive:     true,
		IsIdle:      true,
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
	if m.HurtTime > 0 {
		m.HurtTime--
	}
	if m.TeleportTimer > 0 {
		m.TeleportTimer--
	}

	// Daylight burning for undead (Zombie & Skeleton)
	if (m.Type == MobZombie || m.Type == MobSkeleton) && engine.WorldTime >= 0 && engine.WorldTime <= 12000 {
		footX := int32(math.Floor(m.Position.X))
		footY := int32(math.Floor(m.Position.Y))
		footZ := int32(math.Floor(m.Position.Z))
		underSky := true
		for y := footY + 2; y <= footY+30; y++ {
			if IsSolidBlock(engine.GetBlock(footX, y, footZ)) {
				underSky = false
				break
			}
		}
		inLiquid := engine.GetBlock(footX, footY, footZ) == 8 || engine.GetBlock(footX, footY, footZ) == 9
		if underSky && !inLiquid {
			m.BurningTicks = 20
			if engine.TickCount%20 == 0 {
				m.Health -= 1.0
				if m.Health <= 0 {
					m.Health = 0
					m.IsAlive = false
					drops := m.DeathDrops()
					for _, stack := range drops {
						engine.SpawnItemDropWithDelay(stack.ItemID, stack.Count, m.Position.Add(Vec3{X: 0, Y: 0.5, Z: 0}), Vec3{X: 0, Y: 0.15, Z: 0}, 0.2)
					}
					return
				}
			}
		}
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

	players := engine.GetPlayerList()
	for i := range players {
		p := &players[i]
		if !p.IsAlive || p.GameMode == 1 || p.GameMode == 3 {
			continue
		}
		if engine.Difficulty == 0 && (m.Type == MobZombie || m.Type == MobSkeleton || m.Type == MobCreeper || m.Type == MobEnderman || m.Type == MobSpider) {
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
	case MobEnderman:
		m.tickEndermanAI(engine, closestPlayer, math.Sqrt(closestDistSq))
	case MobSpider:
		m.tickSpiderAI(engine, closestPlayer, math.Sqrt(closestDistSq))
	case MobCow, MobPig, MobSheep, MobChicken:
		m.tickPassiveAI(engine)
	}

	// 4. Apply velocity with voxel collision resolution so mobs don't clip into terrain or freeze
	isSolid := func(x, y, z int32) bool {
		return IsSolidBlock(engine.GetBlock(x, y, z))
	}
	resPos, ground, _ := ResolveMovementCollision(m.Position, m.Velocity, isSolid)
	m.Position = resPos
	if ground {
		m.Velocity.Y = 0
	}
	m.Velocity.X *= 0.546 // Air / ground drag
	m.Velocity.Z *= 0.546
}

// hasLineOfSight tests if an unobstructed direct ray exists between two 3D points.
// Prevents hostile mobs (skeletons, zombies, creepers) from seeing, shooting, or damaging players through solid voxel blocks.
func hasLineOfSight(engine *Engine, from, to Vec3) bool {
	dir := to.Sub(from)
	dist := dir.Length()
	if dist < 0.2 {
		return true
	}
	steps := int(dist * 5.0) // Sample every 0.2 blocks along the ray
	if steps < 1 {
		steps = 1
	}
	stepVec := dir.Scale(1.0 / float64(steps))
	curr := from
	for i := 0; i < steps; i++ {
		curr = curr.Add(stepVec)
		bx := int32(math.Floor(curr.X))
		by := int32(math.Floor(curr.Y))
		bz := int32(math.Floor(curr.Z))
		if IsSolidBlock(engine.GetBlock(bx, by, bz)) {
			return false
		}
	}
	return true
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

	// Melee attack: within 1.5 blocks and line of sight
	if dist <= 1.5 && m.AttackCooldown <= 0 {
		mobChest := m.Position.Add(Vec3{Y: 1.2})
		playerChest := player.Position.Add(Vec3{Y: 1.0})
		if hasLineOfSight(engine, mobChest, playerChest) {
			engine.DamagePlayer(player.ID, 3.0, "zombie")
			m.AttackCooldown = 20 // 1 attack per second
		}
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

	// Shoot arrow every 30 ticks (1.5 seconds) - ONLY if line of sight is unobstructed!
	if m.AttackCooldown <= 0 {
		mobEye := m.Position.Add(Vec3{Y: 1.6})
		playerEye := player.Position.Add(Vec3{Y: 1.62})
		if hasLineOfSight(engine, mobEye, playerEye) {
			engine.DamagePlayer(player.ID, 4.0, "skeleton_arrow")
			m.AttackCooldown = 30
		}
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

	mobChest := m.Position.Add(Vec3{Y: 1.4})
	playerChest := player.Position.Add(Vec3{Y: 1.0})
	hasLOS := hasLineOfSight(engine, mobChest, playerChest)

	if dist > 3.0 || !hasLOS {
		if m.FuseTicks > 0 {
			m.FuseTicks--
		}
		if !hasLOS {
			m.IsMoving = false
			return
		}

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
	// If recently damaged (hurt), PANIC and flee!
	if m.HurtTime > 0 {
		m.IsIdle = false
		panicSpeed := 0.14
		rad := float64(m.Yaw) * math.Pi / 180.0
		fwdX := math.Sin(rad)
		fwdZ := math.Cos(rad)
		m.Velocity.X = fwdX * panicSpeed
		m.Velocity.Z = fwdZ * panicSpeed
		m.WalkDistance += panicSpeed
		m.IsMoving = true
		// 1-block step jump
		aheadX := int32(math.Floor(m.Position.X + fwdX*0.6))
		footY := int32(math.Floor(m.Position.Y + 0.1))
		aheadZ := int32(math.Floor(m.Position.Z + fwdZ*0.6))
		if IsSolidBlock(engine.GetBlock(aheadX, footY, aheadZ)) && !IsSolidBlock(engine.GetBlock(aheadX, footY+1, aheadZ)) && m.Velocity.Y <= 0 {
			m.Velocity.Y = 0.38
		}
		return
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
			m.TargetYaw = float32((int(m.ID*47) + int(time.Now().UnixNano()%360)) % 360)
		}
	} else {
		m.WanderTicks--
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
		// Blocked by tall wall: turn away
		m.TargetYaw = float32((int(m.Yaw) + 120 + int(m.ID*23)%90) % 360)
	} else if IsSolidBlock(blockAhead) && IsSolidBlock(blockAbove1) {
		// Blocked by a high wall: turn around!
		m.TargetYaw = float32(int(m.Yaw+120+float32(m.ID%60)) % 360)
		m.WanderTicks = 40
	}
}

func (m *MobEntity) tickEndermanAI(engine *Engine, player *PlayerEntity, dist float64) {
	// Water damage & panic teleport
	footX := int32(math.Floor(m.Position.X))
	footY := int32(math.Floor(m.Position.Y))
	footZ := int32(math.Floor(m.Position.Z))
	b := engine.GetBlock(footX, footY, footZ)
	if b == 8 || b == 9 {
		m.Health -= 2.0
		m.HurtTime = 10
		m.LastHurtTime = time.Now()
		m.teleportRandomly(engine)
		return
	}

	if player == nil || dist > 48.0 {
		m.IsAggro = false
		m.tickPassiveAI(engine)
		return
	}

	// Stare aggro check: is player looking directly at Enderman?
	if !m.IsAggro {
		radYaw := float64(player.Yaw) * math.Pi / 180.0
		radPitch := float64(player.Pitch) * math.Pi / 180.0
		pLook := Vec3{
			X: -math.Sin(radYaw) * math.Cos(radPitch),
			Y: -math.Sin(radPitch),
			Z: math.Cos(radYaw) * math.Cos(radPitch),
		}
		toMob := m.Position.Add(Vec3{Y: 2.2}).Sub(player.Position.Add(Vec3{Y: 1.62})).Normalize()
		dot := pLook.X*toMob.X + pLook.Y*toMob.Y + pLook.Z*toMob.Z
		if dot > 0.985 {
			m.IsAggro = true
		}
	}

	if !m.IsAggro {
		m.tickPassiveAI(engine)
		return
	}

	// Aggro pursuit
	dx := player.Position.X - m.Position.X
	dz := player.Position.Z - m.Position.Z
	m.Yaw = float32(math.Atan2(-dx, dz) * 180.0 / math.Pi)

	speed := 0.22
	m.Velocity.X = (dx / dist) * speed
	m.Velocity.Z = (dz / dist) * speed
	m.WalkDistance += speed
	m.IsMoving = true

	// Step-climb
	aheadX := int32(math.Floor(m.Position.X + (dx/dist)*0.6))
	aheadZ := int32(math.Floor(m.Position.Z + (dz/dist)*0.6))
	if IsSolidBlock(engine.GetBlock(aheadX, footY, aheadZ)) && !IsSolidBlock(engine.GetBlock(aheadX, footY+1, aheadZ)) && m.Velocity.Y <= 0 {
		m.Velocity.Y = 0.38
	}

	// Attack within 2.0 blocks
	if dist <= 2.0 && m.AttackCooldown <= 0 {
		mobEye := m.Position.Add(Vec3{Y: 2.4})
		playerChest := player.Position.Add(Vec3{Y: 1.0})
		if hasLineOfSight(engine, mobEye, playerChest) {
			engine.DamagePlayer(player.ID, 6.0, "enderman")
			m.AttackCooldown = 20
		}
	}

	// Combat teleport when hurt or at random interval
	if m.HurtTime == 9 && m.TeleportTimer <= 0 {
		m.teleportRandomly(engine)
		m.TeleportTimer = 40
	}
}

func (m *MobEntity) teleportRandomly(engine *Engine) {
	// Attempt random teleport within 16 blocks
	for attempt := 0; attempt < 16; attempt++ {
		ox := float64((int(m.ID*17)+attempt*13)%32 - 16)
		oz := float64((int(m.ID*31)+attempt*19)%32 - 16)
		targetX := m.Position.X + ox
		targetZ := m.Position.Z + oz

		bx := int32(math.Floor(targetX))
		bz := int32(math.Floor(targetZ))

		for by := int32(math.Floor(m.Position.Y)) + 4; by >= int32(math.Floor(m.Position.Y))-6; by-- {
			floor := engine.GetBlock(bx, by-1, bz)
			body := engine.GetBlock(bx, by, bz)
			head1 := engine.GetBlock(bx, by+1, bz)
			head2 := engine.GetBlock(bx, by+2, bz)

			if IsSolidBlock(floor) && !IsSolidBlock(body) && !IsSolidBlock(head1) && !IsSolidBlock(head2) &&
				body != 8 && body != 9 && body != 10 && body != 11 {
				m.Position = Vec3{X: targetX, Y: float64(by), Z: targetZ}
				m.Velocity = Vec3{}
				return
			}
		}
	}
}

func (m *MobEntity) tickSpiderAI(engine *Engine, player *PlayerEntity, dist float64) {
	// Spider neutral in bright daylight unless provoked
	isDay := engine.WorldTime >= 0 && engine.WorldTime <= 12000
	isHostile := !isDay || m.HurtTime > 0

	if player == nil || dist > 16.0 || !isHostile {
		m.tickPassiveAI(engine)
		return
	}

	dx := player.Position.X - m.Position.X
	dz := player.Position.Z - m.Position.Z
	m.Yaw = float32(math.Atan2(-dx, dz) * 180.0 / math.Pi)

	speed := 0.16
	m.Velocity.X = (dx / dist) * speed
	m.Velocity.Z = (dz / dist) * speed
	m.WalkDistance += speed
	m.IsMoving = true

	footY := int32(math.Floor(m.Position.Y))

	// Wall climbing: vertical climb if blocked by a solid wall
	aheadX := int32(math.Floor(m.Position.X + (dx/dist)*0.6))
	aheadZ := int32(math.Floor(m.Position.Z + (dz/dist)*0.6))
	if IsSolidBlock(engine.GetBlock(aheadX, footY, aheadZ)) || IsSolidBlock(engine.GetBlock(aheadX, footY+1, aheadZ)) {
		m.Velocity.Y = 0.22 // Wall climb
	}

	// Pounce leap within 4.5 blocks
	if dist <= 4.5 && dist >= 2.0 && m.Velocity.Y <= 0 {
		m.Velocity.Y = 0.32
		m.Velocity.X *= 1.4
		m.Velocity.Z *= 1.4
	}

	// Bite attack
	if dist <= 1.5 && m.AttackCooldown <= 0 {
		mobChest := m.Position.Add(Vec3{Y: 0.5})
		playerChest := player.Position.Add(Vec3{Y: 1.0})
		if hasLineOfSight(engine, mobChest, playerChest) {
			engine.DamagePlayer(player.ID, 3.0, "spider")
			m.AttackCooldown = 20
		}
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
	case MobEnderman:
		return []inventory.ItemStack{
			{ItemID: 368, Count: 1}, // Ender Pearl
		}
	case MobSpider:
		return []inventory.ItemStack{
			{ItemID: 287, Count: 2}, // String
			{ItemID: 375, Count: 1}, // Spider Eye
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
			{ItemID: 423, Count: 2}, // Raw Mutton
		}
	case MobChicken:
		return []inventory.ItemStack{
			{ItemID: 288, Count: 2}, // Feathers
			{ItemID: 365, Count: 1}, // Raw Chicken
		}
	default:
		return nil
	}
}

