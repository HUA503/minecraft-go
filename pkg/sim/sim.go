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
	Saturation   float32 // 0.0 to 20.0 (canonical vanilla saturation)
	Exhaustion   float32 // 0.0 to 4.0
	RegenTimer   int     // Tick counter for natural regeneration
	StarveTimer  int     // Tick counter for starvation damage
	FallDistance float32
	IsAlive      bool
	ClientDriven bool
	Inventory    *inventory.PlayerInventory
}

// EndCrystal represents an active healing crystal atop an obsidian tower.
type EndCrystal struct {
	ID        int32
	Position  Vec3
	IsAlive   bool
	BeamTarget Vec3
}

// EnderDragon represents the authoritative End dimension boss entity.
type EnderDragon struct {
	Position         Vec3
	Velocity         Vec3
	Yaw              float32
	Pitch            float32
	Health           float32 // 0.0 to 200.0 (100 hearts)
	MaxHealth        float32 // 200.0
	IsAlive          bool
	IsDying          bool
	DeathTicks       int
	Phase            int // 0: circling, 1: dive attack, 2: perched at exit portal
	TargetPos        Vec3
	HealingCrystalID int32
	WingAngle        float32
	DiveCooldown     int
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
	Type     string // "block_update", "player_move", "player_join", "player_leave", "player_death", "player_respawn", "item_pickup", "dimension_change"
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
	mu               sync.RWMutex
	World            map[world.ChunkPos]*world.Chunk
	DimensionWorlds  map[int]map[world.ChunkPos]*world.Chunk
	CurrentDimension int // 0: Overworld, 1: Nether, 2: End
	Players          map[int32]*PlayerEntity
	nextPlayerID     int32
	Mobs             map[int32]*MobEntity
	nextMobID        int32
	ItemDrops        map[int32]*ItemDrop
	nextItemDropID   int32
	EnderDragon      *EnderDragon
	EndCrystals      map[int32]*EndCrystal
	nextCrystalID    int32
	TickCount        int64
	WorldTime        int64
	Events           chan WorldEvent
	stopChan         chan struct{}
	Storage          *world.SaveManager
}

func NewEngine() *Engine {
	eng := &Engine{
		World:            make(map[world.ChunkPos]*world.Chunk),
		DimensionWorlds:  make(map[int]map[world.ChunkPos]*world.Chunk),
		CurrentDimension: 0,
		Players:          make(map[int32]*PlayerEntity),
		nextPlayerID:     1,
		Mobs:             make(map[int32]*MobEntity),
		nextMobID:        100,
		ItemDrops:        make(map[int32]*ItemDrop),
		nextItemDropID:   1,
		EndCrystals:      make(map[int32]*EndCrystal),
		nextCrystalID:    1,
		WorldTime:        6000, // Canonical Minecraft noon daylight
		Events:           make(chan WorldEvent, 1024),
		stopChan:         make(chan struct{}),
	}
	eng.DimensionWorlds[0] = eng.World
	eng.DimensionWorlds[1] = make(map[world.ChunkPos]*world.Chunk)
	eng.DimensionWorlds[2] = make(map[world.ChunkPos]*world.Chunk)
	return eng
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
		ID:         id,
		Username:   username,
		UUID:       uuid,
		Position:   spawnPos,
		SpawnPos:   spawnPos,
		OnGround:   true,
		Health:     20.0,
		Hunger:     20,
		Saturation: 5.0,
		IsAlive:    true,
		Inventory:  inventory.NewPlayerInventory(),
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
		p.Saturation = 5.0
		p.Exhaustion = 0
		p.RegenTimer = 0
		p.StarveTimer = 0
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
		// Track fall distance only for server-simulated players (client-driven handles its own)
		if !p.ClientDriven {
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

	// Check if player is in liquid (water 8/9 or lava 10/11) before acquiring lock
	bx := int32(math.Floor(resolvedPos.X))
	by := int32(math.Floor(resolvedPos.Y))
	bz := int32(math.Floor(resolvedPos.Z))
	bFeet := e.GetBlock(bx, by, bz)
	inLiquid := bFeet == 8 || bFeet == 9 || bFeet == 10 || bFeet == 11

	e.mu.Lock()
	if p, ok := e.Players[id]; ok && p.IsAlive {
		if inLiquid {
			p.FallDistance = 0
		} else if ground {
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
		} else if resolvedPos.Y >= oldPos.Y {
			p.FallDistance = 0
		}
		p.Position = resolvedPos
		p.Yaw = yaw
		p.Pitch = pitch
		p.OnGround = ground

		// Vanilla movement exhaustion
		horizDist := math.Sqrt(delta.X*delta.X + delta.Z*delta.Z)
		if horizDist > 0 {
			p.Exhaustion += float32(horizDist * 0.01)
		}
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

	// Natural Health Regeneration & Starvation per canonical Minecraft Java rules
	for _, p := range e.Players {
		if !p.IsAlive {
			continue
		}

		// Drain exhaustion: each 4.0 exhaustion consumes 1.0 saturation, or 1 hunger
		for p.Exhaustion >= 4.0 {
			p.Exhaustion -= 4.0
			if p.Saturation > 0 {
				p.Saturation = float32(math.Max(0, float64(p.Saturation-1.0)))
			} else if p.Hunger > 0 {
				p.Hunger--
			}
		}

		// Fast natural regeneration: when Hunger == 20 && Saturation > 0
		if p.Hunger == 20 && p.Saturation > 0 && p.Health < 20.0 {
			p.RegenTimer++
			if p.RegenTimer >= 10 { // Every 0.5s (10 ticks)
				p.Health = float32(math.Min(20.0, float64(p.Health+1.0)))
				p.Exhaustion += 6.0 // Drains saturation quickly
				p.RegenTimer = 0
			}
		} else if p.Hunger >= 18 && p.Health < 20.0 {
			// Normal natural regeneration: when Hunger >= 18
			p.RegenTimer++
			if p.RegenTimer >= 80 { // Every 4.0s (80 ticks)
				p.Health = float32(math.Min(20.0, float64(p.Health+1.0)))
				p.Exhaustion += 6.0
				p.RegenTimer = 0
			}
		} else {
			p.RegenTimer = 0
		}

		// Starvation: when Hunger == 0
		if p.Hunger == 0 {
			p.StarveTimer++
			if p.StarveTimer >= 80 { // Every 4.0s (80 ticks)
				if p.Health > 1.0 {
					p.Health -= 1.0
				}
				p.StarveTimer = 0
			}
		} else {
			p.StarveTimer = 0
		}
	}

	// Tick Ender Dragon Boss when in End dimension
	if e.CurrentDimension == 2 && e.EnderDragon != nil {
		e.tickEnderDragon()
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

// ConsumeFood eats one unit of food from player's inventory, restoring hunger and saturation points.
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
	p.Saturation += food.Saturation
	if p.Saturation > float32(p.Hunger) {
		p.Saturation = float32(p.Hunger)
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

// SwitchDimension transitions the active simulation to another dimension (0: Overworld, 1: Nether, 2: The End).
func (e *Engine) SwitchDimension(dim int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if dim < 0 || dim > 2 || dim == e.CurrentDimension {
		return
	}
	e.DimensionWorlds[e.CurrentDimension] = e.World
	e.CurrentDimension = dim
	if e.DimensionWorlds[dim] == nil {
		e.DimensionWorlds[dim] = make(map[world.ChunkPos]*world.Chunk)
	}
	e.World = e.DimensionWorlds[dim]

	// If entering The End for the first time, spawn Ender Dragon and End Crystals
	if dim == 2 && e.EnderDragon == nil {
		e.spawnEndBossLocked()
	}

	select {
	case e.Events <- WorldEvent{
		Type:     "dimension_change",
		BlockID:  int32(dim),
	}:
	default:
	}
}

// GetDimension returns current active dimension index (0: Overworld, 1: Nether, 2: End).
func (e *Engine) GetDimension() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.CurrentDimension
}

func (e *Engine) spawnEndBossLocked() {
	e.EnderDragon = &EnderDragon{
		Position:  Vec3{X: 0, Y: 75.0, Z: 0},
		Health:    200.0,
		MaxHealth: 200.0,
		IsAlive:   true,
		Phase:     0,
		TargetPos: Vec3{X: 0, Y: 75.0, Z: 0},
	}

	// 8 End Crystals atop obsidian towers
	pillarAngles := []float64{0, 45, 90, 135, 180, 225, 270, 315}
	pillarHeights := []int32{78, 86, 92, 82, 98, 84, 90, 80}

	for idx, angDeg := range pillarAngles {
		rad := angDeg * math.Pi / 180.0
		px := math.Round(42.0 * math.Cos(rad))
		pz := math.Round(42.0 * math.Sin(rad))
		topY := float64(pillarHeights[idx] + 2)

		cid := e.nextCrystalID
		e.nextCrystalID++
		e.EndCrystals[cid] = &EndCrystal{
			ID:       cid,
			Position: Vec3{X: px, Y: topY, Z: pz},
			IsAlive:  true,
		}
	}
}

// SpawnEndBoss publicly initializes the boss encounter in The End.
func (e *Engine) SpawnEndBoss() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spawnEndBossLocked()
}

// AttackEndCrystal detonates an active End Crystal when attacked.
func (e *Engine) AttackEndCrystal(crystalID int32) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	c, ok := e.EndCrystals[crystalID]
	if !ok || !c.IsAlive {
		return false
	}
	c.IsAlive = false

	// If dragon is tethered to this crystal, deal 10.0 HP damage to dragon!
	if e.EnderDragon != nil && e.EnderDragon.IsAlive && e.EnderDragon.HealingCrystalID == crystalID {
		e.EnderDragon.Health -= 10.0
		e.EnderDragon.HealingCrystalID = 0
		if e.EnderDragon.Health <= 0 {
			e.EnderDragon.Health = 0
			e.EnderDragon.IsDying = true
		}
	}

	// Detonate in 4.0 block radius
	for _, p := range e.Players {
		if p.IsAlive && p.Position.DistanceTo(c.Position) < 6.0 {
			p.Health -= 8.0
			if p.Health <= 0 {
				p.Health = 0
				p.IsAlive = false
			}
		}
	}
	return true
}

// SetDragonHealingCrystal sets the active healing tether crystal ID.
func (e *Engine) SetDragonHealingCrystal(crystalID int32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.EnderDragon != nil {
		e.EnderDragon.HealingCrystalID = crystalID
	}
}

// DamageEnderDragon inflicts direct damage on the boss.
func (e *Engine) DamageEnderDragon(damage float32) (killed bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.EnderDragon == nil || !e.EnderDragon.IsAlive || e.EnderDragon.IsDying {
		return false
	}
	e.EnderDragon.Health -= damage
	if e.EnderDragon.Health <= 0 {
		e.EnderDragon.Health = 0
		e.EnderDragon.IsDying = true
		return true
	}
	return false
}

// GetEnderDragon returns snapshot of current Ender Dragon state.
func (e *Engine) GetEnderDragon() *EnderDragon {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.EnderDragon == nil {
		return nil
	}
	cp := *e.EnderDragon
	return &cp
}

// GetEndCrystals returns snapshot of active End Crystals.
func (e *Engine) GetEndCrystals() []EndCrystal {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]EndCrystal, 0, len(e.EndCrystals))
	for _, c := range e.EndCrystals {
		res = append(res, *c)
	}
	return res
}

func (e *Engine) tickEnderDragon() {
	dragon := e.EnderDragon
	if dragon == nil {
		return
	}

	if dragon.IsDying {
		dragon.DeathTicks++
		if dragon.DeathTicks >= 100 {
			dragon.IsAlive = false
			dragon.IsDying = false
			// Spawn massive XP orbs
			e.SpawnItemDrop(264, 64, Vec3{X: 0, Y: 68, Z: 0}, Vec3{Y: 0.2}) // Diamonds celebration
		}
		return
	}

	if !dragon.IsAlive {
		return
	}

	// Flap wings
	dragon.WingAngle += 0.2
	if dragon.WingAngle > math.Pi*2 {
		dragon.WingAngle -= math.Pi * 2
	}

	// Healing tether: find closest alive End Crystal within 45 blocks
	dragon.HealingCrystalID = 0
	for _, c := range e.EndCrystals {
		if c.IsAlive && dragon.Position.DistanceTo(c.Position) < 45.0 {
			dragon.HealingCrystalID = c.ID
			dragon.Health = float32(math.Min(200.0, float64(dragon.Health+0.2))) // Regenerate
			break
		}
	}

	// Dragon AI Flight: Smooth 3D circular flight circling (0, 75, 0)
	flightAngle := float64(e.TickCount%1200) * (2.0 * math.Pi / 300.0)
	targetRadius := 40.0
	targetHeight := 75.0 + math.Sin(flightAngle*2.0)*6.0

	// Dive attack mode
	if dragon.DiveCooldown > 0 {
		dragon.DiveCooldown--
	} else if len(e.Players) > 0 {
		// Pick closest player to swoop down
		for _, p := range e.Players {
			if p.IsAlive {
				targetRadius = 8.0
				targetHeight = p.Position.Y + 2.0
				dist := dragon.Position.DistanceTo(p.Position)
				if dist < 4.0 {
					p.Health -= 6.0
					if p.Health <= 0 {
						p.Health = 0
						p.IsAlive = false
					}
					dragon.DiveCooldown = 300 // 15s between dives
				}
				break
			}
		}
	}

	destX := targetRadius * math.Cos(flightAngle)
	destZ := targetRadius * math.Sin(flightAngle)
	destPos := Vec3{X: destX, Y: targetHeight, Z: destZ}

	toDest := destPos.Sub(dragon.Position)
	distToDest := toDest.Length()
	if distToDest > 0.1 {
		dir := toDest.Scale(1.0 / distToDest)
		speed := 0.35
		dragon.Velocity = dir.Scale(speed)
		dragon.Position = dragon.Position.Add(dragon.Velocity)
		dragon.Yaw = float32(math.Atan2(-dir.X, dir.Z) * 180.0 / math.Pi)
		dragon.Pitch = float32(-math.Asin(dir.Y) * 180.0 / math.Pi)
	}
}
