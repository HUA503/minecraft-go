package sim

import (
	"math"
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
	SpawnPos     Vec3
	Velocity     Vec3
	Yaw          float32
	Pitch        float32
	OnGround     bool
	Health       float32 // 0.0 to 20.0
	Hunger       int     // 0 to 20
	FallDistance float32
	IsAlive      bool
	ClientDriven bool
	Inventory    *inventory.PlayerInventory
}

// ItemDrop represents a 3D physical item entity in the world dropped from mined blocks or mobs.
type ItemDrop struct {
	ID          int32
	ItemID      int32
	Count       int
	Position    Vec3
	Velocity    Vec3
	Age         float64
	PickupDelay float64
}

// WorldEvent represents a simulation event to broadcast to connected clients.
type WorldEvent struct {
	Type     string // "block_update", "player_move", "player_join", "player_leave", "player_death", "player_respawn", "item_pickup"
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
	mu             sync.RWMutex
	World          map[world.ChunkPos]*world.Chunk
	Players        map[int32]*PlayerEntity
	nextPlayerID   int32
	Mobs           map[int32]*MobEntity
	nextMobID      int32
	ItemDrops      map[int32]*ItemDrop
	nextItemDropID int32
	TickCount      int64
	WorldTime      int64
	Events         chan WorldEvent
	stopChan       chan struct{}
	Storage        *world.SaveManager
}

func NewEngine() *Engine {
	return &Engine{
		World:          make(map[world.ChunkPos]*world.Chunk),
		Players:        make(map[int32]*PlayerEntity),
		nextPlayerID:   1,
		Mobs:           make(map[int32]*MobEntity),
		nextMobID:      100,
		ItemDrops:      make(map[int32]*ItemDrop),
		nextItemDropID: 1,
		WorldTime:      6000, // Canonical Minecraft noon daylight
		Events:         make(chan WorldEvent, 1024),
		stopChan:       make(chan struct{}),
	}
}

// GetWorldTime returns the authoritative 24000-tick Minecraft world time.
func (e *Engine) GetWorldTime() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.WorldTime
}

// SetWorldTime sets the authoritative 24000-tick Minecraft world time.
func (e *Engine) SetWorldTime(t int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.WorldTime = t % 24000
	if e.WorldTime < 0 {
		e.WorldTime += 24000
	}
}

func (e *Engine) SpawnItemDrop(itemID int32, count int, pos Vec3, vel Vec3) *ItemDrop {
	return e.SpawnItemDropWithDelay(itemID, count, pos, vel, 0.3)
}

func (e *Engine) SpawnItemDropWithDelay(itemID int32, count int, pos Vec3, vel Vec3, delay float64) *ItemDrop {
	e.mu.Lock()
	defer e.mu.Unlock()
	id := e.nextItemDropID
	e.nextItemDropID++
	drop := &ItemDrop{
		ID:          id,
		ItemID:      itemID,
		Count:       count,
		Position:    pos,
		Velocity:    vel,
		Age:         0,
		PickupDelay: delay,
	}
	e.ItemDrops[id] = drop
	return drop
}

func (e *Engine) GetItemDrops() []ItemDrop {
	e.mu.RLock()
	defer e.mu.RUnlock()
	drops := make([]ItemDrop, 0, len(e.ItemDrops))
	for _, d := range e.ItemDrops {
		drops = append(drops, *d)
	}
	return drops
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
	if mob.HurtTime > 5 {
		return nil, false
	}
	mob.Health -= damage
	mob.HurtTime = 10
	if mob.Health <= 0 {
		mob.Health = 0
		mob.IsAlive = false
		drops = mob.DeathDrops()
		for _, stack := range drops {
			e.nextItemDropID++
			e.ItemDrops[e.nextItemDropID] = &ItemDrop{
				ID:          e.nextItemDropID,
				ItemID:      stack.ItemID,
				Count:       stack.Count,
				Position:    mob.Position.Add(Vec3{X: 0, Y: 0.5, Z: 0}),
				Velocity:    Vec3{X: 0, Y: 0.15, Z: 0},
				PickupDelay: 0.2,
			}
		}
		delete(e.Mobs, mobID)
		return drops, true
	}
	return nil, false
}

func (e *Engine) GetMobs() []*MobEntity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]*MobEntity, 0, len(e.Mobs))
	for _, m := range e.Mobs {
		if m.IsAlive {
			res = append(res, m)
		}
	}
	return res
}

func (e *Engine) AttackMobInFront(origin, look Vec3, reach float64, damage float32) (int32, bool, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var closestMob *MobEntity
	closestDist := reach

	for _, mob := range e.Mobs {
		if !mob.IsAlive {
			continue
		}
		// Target mob center/chest (~0.9m above feet) for authentic hit registration
		mobCenter := mob.Position.Add(Vec3{X: 0, Y: 0.9, Z: 0})
		toMob := mobCenter.Sub(origin)
		dist := toMob.Length()
		if dist > reach {
			continue
		}
		dir := toMob.Normalize()
		dot := dir.X*look.X + dir.Y*look.Y + dir.Z*look.Z
		if dot > 0.80 && dist < closestDist {
			closestDist = dist
			closestMob = mob
		}
	}

	if closestMob == nil {
		return 0, false, false
	}
	if closestMob.HurtTime > 5 {
		return closestMob.ID, false, false
	}

	closestMob.Health -= damage
	closestMob.HurtTime = 10
	knockback := look.Scale(0.25)
	knockback.Y = 0.12
	closestMob.Velocity = closestMob.Velocity.Add(knockback)

	killed := false
	if closestMob.Health <= 0 {
		closestMob.Health = 0
		closestMob.IsAlive = false
		killed = true
		drops := closestMob.DeathDrops()
		for _, stack := range drops {
			e.nextItemDropID++
			e.ItemDrops[e.nextItemDropID] = &ItemDrop{
				ID:          e.nextItemDropID,
				ItemID:      stack.ItemID,
				Count:       stack.Count,
				Position:    closestMob.Position.Add(Vec3{X: 0, Y: 0.5, Z: 0}),
				Velocity:    Vec3{X: 0, Y: 0.15, Z: 0},
				PickupDelay: 0.2,
			}
		}
		delete(e.Mobs, closestMob.ID)
	}

	return closestMob.ID, killed, true
}

// SetStorage configures an optional world SaveManager for persistence.
func (e *Engine) SetStorage(s *world.SaveManager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Storage = s
}

// SaveAll flushes all dirty loaded chunks to storage.
func (e *Engine) SaveAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.Storage == nil {
		return nil
	}
	for _, chunk := range e.World {
		if chunk != nil && chunk.IsDirty {
			if err := e.Storage.SaveChunk(chunk); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) LoadChunk(cp world.ChunkPos, c *world.Chunk) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.World[cp] = c
}

func (e *Engine) GetChunk(cp world.ChunkPos) *world.Chunk {
	e.mu.Lock()
	chunk, ok := e.World[cp]
	if (!ok || chunk == nil) && e.Storage != nil {
		if loaded, err := e.Storage.LoadChunk(cp); err == nil && loaded != nil {
			e.World[cp] = loaded
			chunk = loaded
			ok = true
		}
	}
	e.mu.Unlock()
	if !ok {
		return nil
	}
	return chunk
}

func (e *Engine) GetBlock(x, y, z int32) int32 {
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	e.mu.Lock()
	chunk, ok := e.World[cp]
	if (!ok || chunk == nil) && e.Storage != nil {
		if loaded, err := e.Storage.LoadChunk(cp); err == nil && loaded != nil {
			e.World[cp] = loaded
			chunk = loaded
			ok = true
		}
	}
	e.mu.Unlock()
	if !ok || chunk == nil {
		return 0
	}
	return chunk.GetBlock(x, y, z)
}

func (e *Engine) SetBlock(x, y, z int32, state int32) {
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	e.mu.Lock()
	chunk, ok := e.World[cp]
	if (!ok || chunk == nil) && e.Storage != nil {
		if loaded, err := e.Storage.LoadChunk(cp); err == nil && loaded != nil {
			e.World[cp] = loaded
			chunk = loaded
			ok = true
		}
	}
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

// UnloadChunk purges a distant chunk from active simulation, saving it to disk if dirty.
func (e *Engine) UnloadChunk(pos world.ChunkPos) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if chunk, exists := e.World[pos]; exists {
		if e.Storage != nil && chunk != nil && chunk.IsDirty {
			_ = e.Storage.SaveChunk(chunk)
		}
		delete(e.World, pos)
	}
}

// BreakBlockWithTool processes player mining using held tool, enforcing authentic harvest requirements and tool wear.
func (e *Engine) BreakBlockWithTool(playerID int32, pos world.BlockPos, toolID int32) (int32, bool) {
	origBlock := e.GetBlock(pos.X, pos.Y, pos.Z)
	if origBlock == 0 || origBlock == 7 { // Air or Bedrock cannot be mined
		return 0, false
	}

	e.SetBlock(pos.X, pos.Y, pos.Z, 0) // Air

	canHarvest := inventory.CanHarvestBlock(toolID, origBlock)
	var dropItem int32

	if canHarvest {
		// Determine drop item
		dropItem = origBlock
		switch origBlock {
		case 1, 4: // Stone, Cobblestone -> Cobblestone
			dropItem = inventory.ItemCobblestone
		case 2, 3: // Grass, Dirt -> Dirt
			dropItem = inventory.ItemDirt
		case 5: // Oak Planks
			dropItem = inventory.ItemOakPlanks
		case 17: // Oak Log
			dropItem = inventory.ItemOakLog
		case 16: // Coal Ore -> Coal
			dropItem = inventory.ItemCoal
		case 56: // Diamond Ore -> Diamond
			dropItem = inventory.ItemDiamond
		case 58: // Crafting Table
			dropItem = inventory.ItemCraftingTable
		case 61: // Furnace
			dropItem = inventory.ItemFurnace
		case 31: // Tall Grass -> Wheat Seeds
			dropItem = 295
		case 37, 38, 45, 49, 50, 87, 89, 121, 150:
			dropItem = origBlock
		}

		dropPos := Vec3{X: float64(pos.X) + 0.5, Y: float64(pos.Y) + 0.5, Z: float64(pos.Z) + 0.5}
		e.mu.Lock()
		e.nextItemDropID++
		e.ItemDrops[e.nextItemDropID] = &ItemDrop{
			ID:          e.nextItemDropID,
			ItemID:      dropItem,
			Count:       1,
			Position:    dropPos,
			Velocity:    Vec3{X: 0, Y: 0.18, Z: 0},
			PickupDelay: 0.4,
		}
		e.mu.Unlock()
	}

	// Apply tool durability loss if player used a tool
	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil {
		slot := p.Inventory.SelectedHotbar
		if slot >= 0 && slot < len(p.Inventory.Slots) {
			item := &p.Inventory.Slots[slot]
			if !item.IsEmpty() {
				inventory.ApplyToolWear(item)
			}
		}
	}
	e.mu.Unlock()

	return dropItem, canHarvest
}

// BreakBlock processes player mining, updates block to Air, and adds dropped item to inventory.
func (e *Engine) BreakBlock(playerID int32, pos world.BlockPos) int32 {
	origBlock := e.GetBlock(pos.X, pos.Y, pos.Z)
	if origBlock == 0 || origBlock == 7 {
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
	existing := e.GetBlock(pos.X, pos.Y, pos.Z)
	// Air (0), water (8, 9), lava (10, 11), tall grass (31, 32), flowers (37, 38), and torches (50) are replaceable
	if existing != 0 && existing != 8 && existing != 9 && existing != 10 && existing != 11 &&
		existing != 31 && existing != 32 && existing != 37 && existing != 38 && existing != 50 {
		return false
	}

	// For solid blocks, prevent placing inside living players
	if IsSolidBlock(blockID) {
		blockBox := NewBlockAABB(pos.X, pos.Y, pos.Z)

		e.mu.RLock()
		for _, p := range e.Players {
			if p.IsAlive {
				playerBox := NewPlayerAABB(p.Position)
				if PlacementIntersects(playerBox, blockBox) {
					e.mu.RUnlock()
					return false // cannot place inside player
				}
			}
		}
		e.mu.RUnlock()
	}

	// Decrement item from player's inventory if matching item is held
	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil {
		sel := p.Inventory.GetSelectedItem()
		if sel.ItemID == blockID && sel.Count > 0 {
			_ = p.Inventory.RemoveItem(p.Inventory.SelectedHotbar, 1)
		}
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
		SpawnPos:  spawnPos,
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
		p.SpawnPos = spawnPos
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
func (e *Engine) MovePlayerRelative(id int32, delta Vec3, yaw, pitch float32) (Vec3, bool, bool) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if !ok || !p.IsAlive {
		if ok && !p.IsAlive {
			// Auto respawn player at safe spawn location
			p.Health = 20.0
			p.Hunger = 20
			p.IsAlive = true
			respawnPos := p.SpawnPos
			if respawnPos.X == 0 && respawnPos.Y == 0 && respawnPos.Z == 0 {
				respawnPos = Vec3{X: 8.0, Y: 66.0, Z: 8.0}
			}
			p.Position = respawnPos
			p.Velocity = Vec3{}
			p.FallDistance = 0
			p.OnGround = true
			p.ClientDriven = true
			e.mu.Unlock()
			return p.Position, true, false
		}
		e.mu.Unlock()
		return Vec3{X: 8.0, Y: 66.0, Z: 8.0}, false, false
	}
	p.ClientDriven = true
	oldPos := p.Position
	e.mu.Unlock()

	isSolid := func(x, y, z int32) bool {
		return IsSolidBlock(e.GetBlock(x, y, z))
	}

	resolvedPos, ground, hitCeiling := ResolveMovementCollision(oldPos, delta, isSolid)

	e.mu.Lock()
	if p, ok := e.Players[id]; ok && p.IsAlive {
		if ground {
			p.Velocity.Y = 0
			if p.FallDistance > 3.0 {
				fallDmg := p.FallDistance - 3.0
				p.Health -= fallDmg
				if p.Health <= 0 {
					p.Health = 0
					p.IsAlive = false
				}
			}
			p.FallDistance = 0
		} else if hitCeiling && p.Velocity.Y > 0 {
			p.Velocity.Y = 0
		} else if resolvedPos.Y < oldPos.Y {
			p.FallDistance += float32(oldPos.Y - resolvedPos.Y)
		}
		p.Position = resolvedPos
		p.Yaw = yaw
		p.Pitch = pitch
		p.OnGround = ground
	}
	e.mu.Unlock()

	select {
	case e.Events <- WorldEvent{
		Type:     "player_move",
		PlayerID: id,
		Position: resolvedPos,
		Yaw:      yaw,
		Pitch:    pitch,
	}:
	default:
	}

	return resolvedPos, ground, hitCeiling
}

// Tick executes a single 50ms simulation step.
func (e *Engine) Tick() {
	e.mu.Lock()
	e.TickCount++
	e.WorldTime = (e.WorldTime + 1) % 24000

	isSolid := func(x, y, z int32) bool {
		cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
		chunk, ok := e.World[cp]
		if !ok || chunk == nil {
			return false
		}
		return IsSolidBlock(chunk.GetBlock(x, y, z))
	}

	// Apply gravity and drag for free-falling players (skip client-driven players to avoid jitter/double gravity)
	for _, p := range e.Players {
		if !p.IsAlive || p.OnGround || p.ClientDriven {
			continue
		}
		p.Velocity.Y -= 0.08               // Minecraft standard gravity per tick
		p.Velocity = p.Velocity.Scale(0.98) // Air resistance
		delta := Vec3{X: p.Velocity.X, Y: p.Velocity.Y, Z: p.Velocity.Z}
		resPos, ground, hitCeiling := ResolveMovementCollision(p.Position, delta, isSolid)
		if ground {
			p.Velocity.Y = 0
			if p.FallDistance > 3.0 {
				fallDmg := p.FallDistance - 3.0
				p.Health -= fallDmg
				if p.Health <= 0 {
					p.Health = 0
					p.IsAlive = false
				}
			}
			p.FallDistance = 0
		} else if hitCeiling && p.Velocity.Y > 0 {
			p.Velocity.Y = 0
		} else if resPos.Y < p.Position.Y {
			p.FallDistance += float32(p.Position.Y - resPos.Y)
		}
		p.Position = resPos
		p.OnGround = ground
	}

	// Update physical floating item drops
	for id, drop := range e.ItemDrops {
		drop.Age += 0.05
		if drop.PickupDelay > 0 {
			drop.PickupDelay -= 0.05
		}
		// Item physics: gravity & gentle air resistance
		drop.Velocity.Y -= 0.04
		drop.Position.X += drop.Velocity.X
		drop.Position.Y += drop.Velocity.Y
		drop.Position.Z += drop.Velocity.Z

		// Ground rest collision
		bx := int32(math.Floor(drop.Position.X))
		by := int32(math.Floor(drop.Position.Y))
		bz := int32(math.Floor(drop.Position.Z))
		if isSolid(bx, by, bz) {
			drop.Position.Y = float64(by + 1)
			drop.Velocity = Vec3{}
		}

		// Proximity magnetic pull & pickup by any player within reach (2.5 blocks)
		if drop.PickupDelay <= 0 {
			for _, p := range e.Players {
				if !p.IsAlive {
					continue
				}
				dist := p.Position.DistanceTo(drop.Position)
				if dist < 2.5 {
					// Magnetic acceleration towards player
					dx := p.Position.X - drop.Position.X
					dy := (p.Position.Y + 0.6) - drop.Position.Y
					dz := p.Position.Z - drop.Position.Z
					drop.Position.X += dx * 0.2
					drop.Position.Y += dy * 0.2
					drop.Position.Z += dz * 0.2
				}
				if dist < 1.8 {
					rem := p.Inventory.AddItem(drop.ItemID, drop.Count)
					pickedUp := drop.Count - rem
					if pickedUp > 0 {
						select {
						case e.Events <- WorldEvent{
							Type:     "item_pickup",
							PlayerID: p.ID,
							BlockID:  drop.ItemID,
						}:
						default:
						}
					}
					if rem <= 0 {
						delete(e.ItemDrops, id)
					} else {
						drop.Count = rem
					}
					break
				}
			}
		}

		// Despawn ancient item drops after 5 minutes
		if drop.Age > 300.0 {
			delete(e.ItemDrops, id)
		}
	}

	// Natural Health Regeneration & Starvation check every 4 seconds (80 ticks)
	if e.TickCount%80 == 0 {
		for _, p := range e.Players {
			if !p.IsAlive {
				continue
			}
			if p.Hunger >= 18 && p.Health < 20.0 {
				p.Health += 1.0
				if p.Health > 20.0 {
					p.Health = 20.0
				}
			} else if p.Hunger == 0 {
				if p.Health > 1.0 {
					p.Health -= 1.0
				}
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

// ConsumeFood eats one unit of food from player's inventory, restoring hunger points.
func (e *Engine) ConsumeFood(playerID int32, slotIdx int) (bool, int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.Players[playerID]
	if !ok || !p.IsAlive || p.Inventory == nil {
		return false, 0
	}
	if slotIdx < 0 || slotIdx >= len(p.Inventory.Slots) {
		return false, 0
	}
	slot := &p.Inventory.Slots[slotIdx]
	if slot.IsEmpty() {
		return false, 0
	}
	food, isFood := inventory.FoodRegistry[slot.ItemID]
	if !isFood {
		return false, 0
	}
	if p.Hunger >= 20 {
		return false, 0 // Full stomach
	}

	p.Hunger += food.HungerRestored
	if p.Hunger > 20 {
		p.Hunger = 20
	}

	slot.Count--
	if slot.Count <= 0 {
		*slot = inventory.ItemStack{}
	}

	select {
	case e.Events <- WorldEvent{
		Type:     "player_eat",
		PlayerID: playerID,
		BlockID:  slot.ItemID,
	}:
	default:
	}

	return true, p.Hunger
}
