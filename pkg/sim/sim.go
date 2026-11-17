package sim

import (
	"sync"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

// PlayerEntity represents an active player within the simulation.
type PlayerEntity struct {
	ID           int32
	Username     string
	UUID         [16]byte
	Position     Vec3
	Velocity     Vec3
	Yaw          float32
	Pitch        float32
	OnGround     bool
	Health       float32 // 0.0 to 20.0
	Hunger       int     // 0 to 20
	FallDistance float32
	IsAlive      bool
	Inventory    *inventory.PlayerInventory
}

// WorldEvent represents a simulation event to broadcast to connected clients.
type WorldEvent struct {
	Type     string // "block_update", "player_move", "player_join", "player_leave", "player_death", "player_respawn"
	PlayerID int32
	BlockPos world.BlockPos
	BlockID  int32
	Position Vec3
	Yaw      float32
	Pitch    float32
	Health   float32
}

// Engine manages the authoritative 20 TPS game simulation.
type Engine struct {
	mu           sync.RWMutex
	World        map[world.ChunkPos]*world.Chunk
	Players      map[int32]*PlayerEntity
	nextPlayerID int32
	Mobs         map[int32]*MobEntity
	nextMobID    int32
	TickCount    int64
	Events       chan WorldEvent
	stopChan     chan struct{}
}

func NewEngine() *Engine {
	return &Engine{
		World:        make(map[world.ChunkPos]*world.Chunk),
		Players:      make(map[int32]*PlayerEntity),
		nextPlayerID: 1,
		Mobs:         make(map[int32]*MobEntity),
		nextMobID:    100,
		Events:       make(chan WorldEvent, 1024),
		stopChan:     make(chan struct{}),
	}
}

func (e *Engine) SpawnMob(mobType MobType, pos Vec3) *MobEntity {
	e.mu.Lock()
	defer e.mu.Unlock()
	id := e.nextMobID
	e.nextMobID++
	mob := NewMobEntity(id, mobType, pos)
	e.Mobs[id] = mob
	return mob
}

func (e *Engine) DamageMob(mobID int32, damage float32) (drops []inventory.ItemStack, killed bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	mob, ok := e.Mobs[mobID]
	if !ok || !mob.IsAlive {
		return nil, false
	}
	mob.Health -= damage
	if mob.Health <= 0 {
		mob.Health = 0
		mob.IsAlive = false
		drops = mob.DeathDrops()
		delete(e.Mobs, mobID)
		return drops, true
	}
	return nil, false
}

func (e *Engine) LoadChunk(cp world.ChunkPos, c *world.Chunk) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.World[cp] = c
}

func (e *Engine) GetChunk(cp world.ChunkPos) *world.Chunk {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.World[cp]
}

func (e *Engine) GetBlock(x, y, z int32) int32 {
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	e.mu.RLock()
	chunk, ok := e.World[cp]
	e.mu.RUnlock()
	if !ok || chunk == nil {
		return 0
	}
	return chunk.GetBlock(x, y, z)
}

func (e *Engine) SetBlock(x, y, z int32, state int32) {
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	e.mu.Lock()
	chunk, ok := e.World[cp]
	if !ok || chunk == nil {
		chunk = world.NewChunk(cp)
		e.World[cp] = chunk
	}
	chunk.SetBlock(x, y, z, state)
	e.mu.Unlock()

	// Broadcast block update event
	select {
	case e.Events <- WorldEvent{
		Type:     "block_update",
		BlockPos: world.BlockPos{X: x, Y: y, Z: z},
		BlockID:  state,
	}:
	default:
	}
}

// BreakBlock processes player mining, updates block to Air, and adds dropped item to inventory.
func (e *Engine) BreakBlock(playerID int32, pos world.BlockPos) int32 {
	origBlock := e.GetBlock(pos.X, pos.Y, pos.Z)
	if origBlock == 0 {
		return 0
	}

	e.SetBlock(pos.X, pos.Y, pos.Z, 0) // Air

	// Determine drop item
	dropItem := origBlock
	switch origBlock {
	case 1: // Stone -> Cobblestone
		dropItem = inventory.ItemCobblestone
	case 2: // Grass -> Dirt
		dropItem = inventory.ItemDirt
	case 16: // Coal Ore -> Coal
		dropItem = inventory.ItemCoal
	case 56: // Diamond Ore -> Diamond
		dropItem = inventory.ItemDiamond
	}

	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil {
		p.Inventory.AddItem(dropItem, 1)
	}
	e.mu.Unlock()

	return dropItem
}

// PlaceBlock places a block if it does not intersect the player's bounding box.
func (e *Engine) PlaceBlock(playerID int32, pos world.BlockPos, blockID int32) bool {
	blockBox := NewBlockAABB(pos.X, pos.Y, pos.Z)

	e.mu.RLock()
	for _, p := range e.Players {
		if p.IsAlive {
			playerBox := NewPlayerAABB(p.Position)
			if blockBox.Intersects(playerBox) {
				e.mu.RUnlock()
				return false // cannot place inside player
			}
		}
	}
	e.mu.RUnlock()

	// Decrement item from player's inventory
	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil {
		_ = p.Inventory.RemoveItem(p.Inventory.SelectedHotbar, 1)
	}
	e.mu.Unlock()

	e.SetBlock(pos.X, pos.Y, pos.Z, blockID)
	return true
}

func (e *Engine) AddPlayer(username string, uuid [16]byte, spawnPos Vec3) *PlayerEntity {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := e.nextPlayerID
	e.nextPlayerID++

	p := &PlayerEntity{
		ID:        id,
		Username:  username,
		UUID:      uuid,
		Position:  spawnPos,
		OnGround:  true,
		Health:    20.0,
		Hunger:    20,
		IsAlive:   true,
		Inventory: inventory.NewPlayerInventory(),
	}
	e.Players[id] = p

	select {
	case e.Events <- WorldEvent{
		Type:     "player_join",
		PlayerID: id,
		Position: spawnPos,
		Health:   p.Health,
	}:
	default:
	}

	return p
}

func (e *Engine) RemovePlayer(id int32) {
	e.mu.Lock()
	delete(e.Players, id)
	e.mu.Unlock()

	select {
	case e.Events <- WorldEvent{
		Type:     "player_leave",
		PlayerID: id,
	}:
	default:
	}
}

func (e *Engine) DamagePlayer(id int32, amount float32, reason string) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if !ok || !p.IsAlive {
		e.mu.Unlock()
		return
	}

	p.Health -= amount
	if p.Health <= 0 {
		p.Health = 0
		p.IsAlive = false
		e.mu.Unlock()
		select {
		case e.Events <- WorldEvent{
			Type:     "player_death",
			PlayerID: id,
		}:
		default:
		}
		return
	}
	e.mu.Unlock()
}

func (e *Engine) RespawnPlayer(id int32, spawnPos Vec3) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if ok {
		p.Health = 20.0
		p.Hunger = 20
		p.IsAlive = true
		p.Position = spawnPos
		p.Velocity = Vec3{}
		p.FallDistance = 0
		p.OnGround = true
	}
	e.mu.Unlock()

	if ok {
		select {
		case e.Events <- WorldEvent{
			Type:     "player_respawn",
			PlayerID: id,
			Position: spawnPos,
			Health:   20.0,
		}:
		default:
		}
	}
}

func (e *Engine) MovePlayer(id int32, pos Vec3, yaw, pitch float32, onGround bool) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if ok && p.IsAlive {
		// Track fall distance
		if !onGround && pos.Y < p.Position.Y {
			p.FallDistance += float32(p.Position.Y - pos.Y)
		} else if onGround {
			if p.FallDistance > 3.0 {
				damage := p.FallDistance - 3.0
				e.mu.Unlock()
				e.DamagePlayer(id, damage, "fall")
				e.mu.Lock()
			}
			p.FallDistance = 0
		}

		p.Position = pos
		p.Yaw = yaw
		p.Pitch = pitch
		p.OnGround = onGround
	}
	e.mu.Unlock()

	if ok {
		select {
		case e.Events <- WorldEvent{
			Type:     "player_move",
			PlayerID: id,
			Position: pos,
			Yaw:      yaw,
			Pitch:    pitch,
		}:
		default:
		}
	}
}

// MovePlayerRelative handles movement inputs with authentic AABB voxel collision resolution.
// Prevents any wall or floor clipping, and supports smooth sliding along solid voxel faces.
func (e *Engine) MovePlayerRelative(id int32, delta Vec3, yaw, pitch float32) (Vec3, bool) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if !ok || !p.IsAlive {
		e.mu.Unlock()
		return Vec3{}, false
	}
	oldPos := p.Position
	e.mu.Unlock()

	isSolid := func(x, y, z int32) bool {
		return e.GetBlock(x, y, z) != 0
	}

	resolvedPos, ground := ResolveMovementCollision(oldPos, delta, isSolid)

	e.MovePlayer(id, resolvedPos, yaw, pitch, ground)
	return resolvedPos, ground
}

// Tick executes a single 50ms simulation step.
func (e *Engine) Tick() {
	e.mu.Lock()
	e.TickCount++

	// Apply gravity and drag for free-falling players
	for _, p := range e.Players {
		if !p.IsAlive {
			continue
		}
		if !p.OnGround {
			p.Velocity.Y -= 0.08               // Minecraft standard gravity per tick
			p.Velocity = p.Velocity.Scale(0.98) // Air resistance
			prevY := p.Position.Y
			p.Position = p.Position.Add(p.Velocity)
			if p.Position.Y < prevY {
				p.FallDistance += float32(prevY - p.Position.Y)
			}

			// Simple ground check against block at feet
			footBlockY := int32(p.Position.Y)
			if footBlockY <= 64 { // Flat surface ground level
				p.Position.Y = 64
				p.Velocity.Y = 0
				p.OnGround = true
				if p.FallDistance > 3.0 {
					fallDmg := p.FallDistance - 3.0
					p.Health -= fallDmg
					if p.Health <= 0 {
						p.Health = 0
						p.IsAlive = false
					}
				}
				p.FallDistance = 0
			}
		}
	}

	var activeMobs []*MobEntity
	for _, m := range e.Mobs {
		if m.IsAlive {
			activeMobs = append(activeMobs, m)
		}
	}
	e.mu.Unlock()

	// Tick mob AI and behaviors outside the mutex
	for _, m := range activeMobs {
		m.TickMob(e)
	}
}

// StartTickLoop runs the 20 TPS loop in a background goroutine.
func (e *Engine) StartTickLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				e.Tick()
			case <-e.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (e *Engine) Stop() {
	close(e.stopChan)
}
