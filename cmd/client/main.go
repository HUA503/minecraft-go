package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math"
	mrand "math/rand"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/sound"
	"github.com/HUA503/minecraft-go/pkg/world"
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

type Client struct {
	serverAddr   string
	username     string
	uuid         [16]byte
	conn         net.Conn
	state        protocol.ConnectionState
	framebuf     *render.Framebuffer
	camera       *render.Camera
	engine       *sim.Engine
	player       *sim.PlayerEntity
	craftingMgr  *inventory.CraftingManager
	isLocal      bool
	meshCache    map[[3]int32]*render.Mesh
	meshMu       sync.RWMutex
	targetedPos  world.BlockPos
	targetedFace world.BlockPos
	hasTarget    bool
	walkCycle    float64
	isMoving     bool
	isSprinting  bool
	isSneaking   bool
	wasSpaceDown bool
	genOverworld     *gen.Generator
	loadedChunks     map[world.ChunkPos]bool
	lastChunkX       int32
	lastChunkZ       int32
	// Authentic Minecraft Mining State
	isMining         bool
	breakProgress    float64 // 0.0 to 1.0
	miningPos        world.BlockPos
	hasMiningBlock   bool
	swingProgress    float64 // 0.0 to 1.0
	isSwinging       bool
	lastHitSoundTime time.Time
	placeCooldown    float64
	particles        []render.Particle
	showInventoryGUI bool
	showPauseMenu    bool
	backBtnRect      [4]int
	quitBtnRect      [4]int
	isEating         bool
	eatTimer         float64
	lastMunchSound   time.Time
	showDebugF3      bool
	lastStepTime     time.Time
	fpsCounter       int
	currentFPS       int
	fpsTimer         time.Time
	lastFrameDuration float64
	isChatOpen       bool
	chatInput        string
	chatMessages     []render.ChatMessage
	isCreative       bool
	isFlying         bool
	perspectiveMode  int // 0: 1st person, 1: 3rd person back, 2: 3rd person front
	lastPlayerHealth float32
	hurtFlashTimer   float64
	currentEyeHeight float64
	targetFov        float64
	equipProgress    float64
	lastSelectedSlot int
}

// findSafeSpawnPoint discovers an authentic, randomized safe surface spawn location in the procedural world.
// Evaluates terrain elevation across random coordinates within the spawn region, ensuring the player
// is placed firmly on a solid walkable surface (grass, dirt, sand, stone) strictly above sea level (>= 62),
// completely clear of water or lava hazards, and with unobstructed headroom.
func findSafeSpawnPoint(generator *gen.Generator, engine *sim.Engine) sim.Vec3 {
	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for attempt := 0; attempt < 100; attempt++ {
		// Randomize coordinates within [-96, 96]
		wx := int32(rng.Intn(192) - 96)
		wz := int32(rng.Intn(192) - 96)

		cp := world.BlockPos{X: wx, Y: 0, Z: wz}.ToChunkPos()
		chunk := engine.GetChunk(cp)
		if chunk == nil {
			chunk = generator.GenerateChunk(cp)
			engine.LoadChunk(cp, chunk)
		}

		for y := int32(90); y >= 62; y-- {
			b := chunk.GetBlock(wx, y, wz)
			if b == gen.BlockGrass || b == gen.BlockSand || b == gen.BlockDirt || b == gen.BlockStone {
				above1 := chunk.GetBlock(wx, y+1, wz)
				above2 := chunk.GetBlock(wx, y+2, wz)
				// Must not be solid blocks (air or foliage) and never water or lava
				if !sim.IsSolidBlock(above1) && !sim.IsSolidBlock(above2) &&
					above1 != 8 && above1 != 9 && above1 != 10 && above1 != 11 &&
					above2 != 8 && above2 != 9 && above2 != 10 && above2 != 11 {
					return sim.Vec3{
						X: float64(wx) + 0.5,
						Y: float64(y + 1),
						Z: float64(wz) + 0.5,
					}
				}
				break
			}
			if b == gen.BlockWater || b == gen.BlockFlowingWater {
				// Surface here is water, cannot spawn here
				break
			}
		}
	}
	return sim.Vec3{X: 8.5, Y: 66.0, Z: 8.5}
}

func NewClient(serverAddr, username string, isSingleplayer bool, width, height int) *Client {
	var u [16]byte
	rand.Read(u[:])

	cli := &Client{
		serverAddr:       serverAddr,
		username:         username,
		uuid:             u,
		state:            protocol.StateHandshake,
		framebuf:         render.NewFramebuffer(width, height),
		camera:           render.NewCamera(8.0, 66.62, 8.0),
		craftingMgr:      inventory.NewCraftingManager(),
		isLocal:          isSingleplayer,
		meshCache:        make(map[[3]int32]*render.Mesh),
		loadedChunks:     make(map[world.ChunkPos]bool),
		lastChunkX:       -999,
		lastChunkZ:       -999,
		currentEyeHeight: 1.62,
		targetFov:        70.0,
		equipProgress:    1.0,
		lastSelectedSlot: 1,
	}

	if isSingleplayer {
		cli.engine = sim.NewEngine()
		cli.genOverworld = gen.NewGenerator(12345, gen.DimensionOverworld)
		cli.engine.StartTickLoop()

		spawnPos := findSafeSpawnPoint(cli.genOverworld, cli.engine)
		cli.player = cli.engine.AddPlayer(username, u, spawnPos)
		cli.camera.X = spawnPos.X
		cli.camera.Y = spawnPos.Y + 1.62
		cli.camera.Z = spawnPos.Z

		// Classic starter items in hotbar
		cli.player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1}
		cli.player.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemOakPlanks, Count: 64}
		cli.player.Inventory.Slots[2] = inventory.ItemStack{ItemID: inventory.ItemCobblestone, Count: 64}
		cli.player.Inventory.Slots[3] = inventory.ItemStack{ItemID: inventory.ItemDirt, Count: 64}
		cli.player.Inventory.Slots[4] = inventory.ItemStack{ItemID: 268, Count: 1} // Wooden Sword
		cli.player.Inventory.Slots[5] = inventory.ItemStack{ItemID: 271, Count: 1} // Wooden Axe
		cli.player.Inventory.Slots[6] = inventory.ItemStack{ItemID: 269, Count: 1} // Wooden Shovel
		cli.player.Inventory.Slots[7] = inventory.ItemStack{ItemID: inventory.ItemCookedPorkchop, Count: 16}
		cli.player.Inventory.Slots[8] = inventory.ItemStack{ItemID: inventory.ItemApple, Count: 16}
		cli.player.Inventory.SelectedHotbar = 1 // Start on placeable Oak Planks by default

		// Spawn living mobs in meadow around player (safe distance from spawn point)
		cli.engine.SpawnMob(sim.MobPig, sim.Vec3{X: spawnPos.X + 3.5, Y: spawnPos.Y, Z: spawnPos.Z + 2.5})
		cli.engine.SpawnMob(sim.MobCow, sim.Vec3{X: spawnPos.X - 3.5, Y: spawnPos.Y, Z: spawnPos.Z + 4.5})
		cli.engine.SpawnMob(sim.MobZombie, sim.Vec3{X: spawnPos.X + 24.0, Y: spawnPos.Y, Z: spawnPos.Z + 24.0})
		cli.engine.SpawnMob(sim.MobCreeper, sim.Vec3{X: spawnPos.X + 30.0, Y: spawnPos.Y, Z: spawnPos.Z + 30.0})

		// Pre-mesh immediate spawn sections synchronously so terrain is immediately visible
		for i := 0; i < 25; i++ {
			cli.EnsureChunksAroundPlayer(spawnPos.X, spawnPos.Z, 1)
		}
	}

	return cli
}

// EnsureChunksAroundPlayer streams procedural world chunks and progressively meshes sections around the player.
func (c *Client) EnsureChunksAroundPlayer(playerX, playerZ float64, radius int32) {
	if c.genOverworld == nil || c.engine == nil {
		return
	}
	pcx := int32(math.Floor(playerX / 16.0))
	pcz := int32(math.Floor(playerZ / 16.0))

	// Pass 0: Prune distant meshes and chunks beyond (radius + 2) to maintain strictly bounded memory (< 40MB)
	c.meshMu.Lock()
	for key := range c.meshCache {
		diffX := key[0] - pcx
		if diffX < 0 {
			diffX = -diffX
		}
		diffZ := key[2] - pcz
		if diffZ < 0 {
			diffZ = -diffZ
		}
		if diffX > radius+1 || diffZ > radius+1 {
			delete(c.meshCache, key)
		}
	}
	c.meshMu.Unlock()

	for cp := range c.loadedChunks {
		diffX := cp.X - pcx
		if diffX < 0 {
			diffX = -diffX
		}
		diffZ := cp.Z - pcz
		if diffZ < 0 {
			diffZ = -diffZ
		}
		if diffX > radius+2 || diffZ > radius+2 {
			delete(c.loadedChunks, cp)
			c.engine.UnloadChunk(cp)
		}
	}

	// Pass 1: Progressively generate missing chunks in radius+1 (up to 2 chunks per frame to eliminate main-thread stutter)
	generatedCount := 0
	for dx := -radius - 1; dx <= radius + 1; dx++ {
		for dz := -radius - 1; dz <= radius + 1; dz++ {
			cp := world.ChunkPos{X: pcx + dx, Z: pcz + dz}
			if !c.loadedChunks[cp] {
				chunk := c.genOverworld.GenerateChunk(cp)
				c.engine.LoadChunk(cp, chunk)
				c.loadedChunks[cp] = true
				generatedCount++
				if generatedCount >= 2 {
					break
				}
			}
		}
		if generatedCount >= 2 {
			break
		}
	}

	getBlock := func(x, y, z int32) int32 {
		return c.engine.GetBlock(x, y, z)
	}

	// Pass 2: Progressive section meshing (build at most 2 sections per frame, prioritizing surface sections sy=4,3,5 first)
	c.meshMu.RLock()
	var unmeshed [][3]int32
	preferredSY := []int32{4, 3, 5, 2, 6, 1, 7, 0}
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			cp := world.ChunkPos{X: pcx + dx, Z: pcz + dz}
			for _, sy := range preferredSY {
				key := [3]int32{cp.X, sy, cp.Z}
				if c.meshCache[key] == nil {
					unmeshed = append(unmeshed, key)
				}
			}
		}
	}
	c.meshMu.RUnlock()

	if len(unmeshed) == 0 {
		return
	}

	maxBuild := 2
	if len(unmeshed) < maxBuild {
		maxBuild = len(unmeshed)
	}

	for i := 0; i < maxBuild; i++ {
		key := unmeshed[i]
		mesh := render.BuildSectionMesh(getBlock, key[0]*16, key[1]*16, key[2]*16)
		c.meshMu.Lock()
		c.meshCache[key] = mesh
		c.meshMu.Unlock()
	}
}

// InvalidateBlockMesh purges cached section mesh containing pos and affected neighbor boundary sections.
func (c *Client) InvalidateBlockMesh(pos world.BlockPos) {
	c.meshMu.Lock()
	defer c.meshMu.Unlock()

	secX := int32(math.Floor(float64(pos.X) / 16.0))
	secY := int32(math.Floor(float64(pos.Y) / 16.0))
	secZ := int32(math.Floor(float64(pos.Z) / 16.0))

	delete(c.meshCache, [3]int32{secX, secY, secZ})

	remX := ((pos.X % 16) + 16) % 16
	remY := ((pos.Y % 16) + 16) % 16
	remZ := ((pos.Z % 16) + 16) % 16

	if remX == 0 {
		delete(c.meshCache, [3]int32{secX - 1, secY, secZ})
	} else if remX == 15 {
		delete(c.meshCache, [3]int32{secX + 1, secY, secZ})
	}
	if remY == 0 && secY > 0 {
		delete(c.meshCache, [3]int32{secX, secY - 1, secZ})
	} else if remY == 15 && secY < 7 {
		delete(c.meshCache, [3]int32{secX, secY + 1, secZ})
	}
	if remZ == 0 {
		delete(c.meshCache, [3]int32{secX, secY, secZ - 1})
	} else if remZ == 15 {
		delete(c.meshCache, [3]int32{secX, secY, secZ + 1})
	}
}

// RebuildAffectedMeshes synchronously remeshes the section containing pos and any boundary neighbor sections.
// Guarantees zero frame delay, no flickering, and no disappearing void holes when placing or breaking blocks.
func (c *Client) RebuildAffectedMeshes(pos world.BlockPos) {
	secX := int32(math.Floor(float64(pos.X) / 16.0))
	secY := int32(math.Floor(float64(pos.Y) / 16.0))
	secZ := int32(math.Floor(float64(pos.Z) / 16.0))

	getBlock := func(x, y, z int32) int32 {
		return c.engine.GetBlock(x, y, z)
	}

	m0 := render.BuildSectionMesh(getBlock, secX*16, secY*16, secZ*16)

	c.meshMu.Lock()
	c.meshCache[[3]int32{secX, secY, secZ}] = m0

	remX := ((pos.X % 16) + 16) % 16
	remY := ((pos.Y % 16) + 16) % 16
	remZ := ((pos.Z % 16) + 16) % 16

	if remX == 0 {
		c.meshCache[[3]int32{secX - 1, secY, secZ}] = render.BuildSectionMesh(getBlock, (secX-1)*16, secY*16, secZ*16)
	} else if remX == 15 {
		c.meshCache[[3]int32{secX + 1, secY, secZ}] = render.BuildSectionMesh(getBlock, (secX+1)*16, secY*16, secZ*16)
	}
	if remY == 0 {
		c.meshCache[[3]int32{secX, secY - 1, secZ}] = render.BuildSectionMesh(getBlock, secX*16, (secY-1)*16, secZ*16)
	} else if remY == 15 {
		c.meshCache[[3]int32{secX, secY + 1, secZ}] = render.BuildSectionMesh(getBlock, secX*16, (secY+1)*16, secZ*16)
	}
	if remZ == 0 {
		c.meshCache[[3]int32{secX, secY, secZ - 1}] = render.BuildSectionMesh(getBlock, secX*16, secY*16, (secZ-1)*16)
	} else if remZ == 15 {
		c.meshCache[[3]int32{secX, secY, secZ + 1}] = render.BuildSectionMesh(getBlock, secX*16, secY*16, (secZ+1)*16)
	}
	c.meshMu.Unlock()
}

func (c *Client) RebuildMeshes() {
	if c.player != nil {
		c.meshMu.Lock()
		c.meshCache = make(map[[3]int32]*render.Mesh)
		c.meshMu.Unlock()

		c.lastChunkX = -999
		c.EnsureChunksAroundPlayer(c.player.Position.X, c.player.Position.Z, 2)
	}
}

func (c *Client) Connect() error {
	var err error
	c.conn, err = net.DialTimeout("tcp", c.serverAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.serverAddr, err)
	}
	log.Printf("[Client] Connected to server at %s", c.serverAddr)

	hs := &protocol.HandshakePacket{
		ProtocolVersion: 770,
		ServerAddress:   "localhost",
		ServerPort:      25565,
		NextState:       2,
	}
	if err := protocol.SendPacket(c.conn, hs); err != nil {
		return err
	}

	c.state = protocol.StateLogin
	ls := &protocol.LoginStartPacket{
		Username:   c.username,
		PlayerUUID: c.uuid,
	}
	if err := protocol.SendPacket(c.conn, ls); err != nil {
		return err
	}

	raw, err := protocol.ReadRawPacket(c.conn)
	if err != nil {
		return err
	}
	var succ protocol.LoginSuccessPacket
	if err := succ.Decode(raw.Reader()); err != nil {
		return err
	}
	c.state = protocol.StatePlay
	log.Printf("[Client] Logged in successfully as %s", succ.Username)

	return nil
}

// checkWaterState examines the player's bounding volume to determine if they are in water,
// and whether they are positioned near the water surface.
func (c *Client) checkWaterState() (inWater bool, atSurface bool) {
	if c.player == nil || c.engine == nil {
		return false, false
	}
	px := c.player.Position.X
	py := c.player.Position.Y
	pz := c.player.Position.Z

	isWater := func(b int32) bool {
		return b == 8 || b == 9
	}

	bx := int32(math.Floor(px))
	bz := int32(math.Floor(pz))

	feetY := int32(math.Floor(py + 0.1))
	torsoY := int32(math.Floor(py + 0.8))
	belowY := int32(math.Floor(py - 0.2))
	headY := int32(math.Floor(py + 1.6))

	bFeet := c.engine.GetBlock(bx, feetY, bz)
	bTorso := c.engine.GetBlock(bx, torsoY, bz)
	bBelow := c.engine.GetBlock(bx, belowY, bz)
	bHead := c.engine.GetBlock(bx, headY, bz)

	waterFeet := isWater(bFeet)
	waterTorso := isWater(bTorso)
	waterBelow := isWater(bBelow) && (py-math.Floor(py)) < 0.35

	// Check 4 horizontal corners at feet level
	cornerWater := false
	for _, dx := range []float64{-0.25, 0.25} {
		for _, dz := range []float64{-0.25, 0.25} {
			cb := c.engine.GetBlock(int32(math.Floor(px+dx)), feetY, int32(math.Floor(pz+dz)))
			if isWater(cb) {
				cornerWater = true
				break
			}
		}
		if cornerWater {
			break
		}
	}

	inWater = waterFeet || waterTorso || cornerWater || waterBelow
	atSurface = inWater && !isWater(bHead)
	return inWater, atSurface
}

// PhysicsStep executes one delta-time scaled physics step with authentic Minecraft Java Edition physics.
func (c *Client) PhysicsStep(fwdInput, strafeInput float64, jumpRequested bool, dt float64) {
	if c.player == nil {
		return
	}

	// 0. Creative Flight Handling
	if c.isFlying {
		speed := 9.0
		var fwd, strafe float64
		lenInput := math.Hypot(fwdInput, strafeInput)
		if lenInput > 0 {
			fwd = (fwdInput / lenInput) * speed * dt
			strafe = (strafeInput / lenInput) * speed * dt
		}
		radYaw := c.camera.Yaw * math.Pi / 180.0
		dx := math.Sin(radYaw)*fwd + math.Cos(radYaw)*strafe
		dz := math.Cos(radYaw)*fwd - math.Sin(radYaw)*strafe
		dy := 0.0
		if jumpRequested {
			dy = speed * dt // Fly upward smoothly while holding space
		} else if c.isSneaking {
			dy = -speed * dt // Fly downward
		}
		c.player.Position.X += dx
		c.player.Position.Y += dy
		c.player.Position.Z += dz
		c.player.Velocity = sim.Vec3{X: 0, Y: 0, Z: 0}
		c.player.OnGround = false
		c.camera.X = c.player.Position.X
		c.camera.Y = c.player.Position.Y + 1.62
		c.camera.Z = c.player.Position.Z
		c.EnsureChunksAroundPlayer(c.player.Position.X, c.player.Position.Z, 2)
		return
	}

	// 1. Water Detection & Aquatic Physics vs Air Gravity
	inWater, atSurface := c.checkWaterState()

	if inWater {
		c.player.FallDistance = 0 // No fall damage in water
		if jumpRequested {
			// Swimming upward in water: continuous upward propulsion while holding Space
			if atSurface {
				// Surface breach: authentic Minecraft leap out of water onto riverbanks / land
				if c.player.Velocity.Y < 4.5 {
					c.player.Velocity.Y = 6.2
				}
			} else {
				// Submerged: steady continuous swim ascent (~4.2 m/s vanilla equilibrium)
				c.player.Velocity.Y += 24.0 * dt
				if c.player.Velocity.Y > 4.2 {
					c.player.Velocity.Y = 4.2
				}
				if c.player.Velocity.Y < 1.5 {
					c.player.Velocity.Y = 1.5
				}
			}
			c.player.OnGround = false
		} else if c.isSneaking {
			// Fast dive downward in water (Sneak / Shift)
			c.player.Velocity.Y = -4.0
		} else {
			// Gentle sinking drift in water (vanilla water buoyancy)
			c.player.Velocity.Y -= 6.0 * dt
			if c.player.Velocity.Y < -3.0 {
				c.player.Velocity.Y = -3.0
			}
		}
	} else {
		// 2. Air / Land Physics
		// Jumping: Authentic Minecraft initial upward impulse of 8.4 m/s (0.42 blocks/tick * 20)
		if jumpRequested && c.player.OnGround {
			c.player.Velocity.Y = 8.4
			c.player.OnGround = false
		} else if !c.player.OnGround {
			// Air Gravity: Authentic Minecraft 32.0 m/s^2 (0.08 blocks/tick^2 * 400), terminal velocity -78.4 m/s
			c.player.Velocity.Y -= 32.0 * dt
			if c.player.Velocity.Y < -78.4 {
				c.player.Velocity.Y = -78.4
			}
		}
	}

	// 3. Horizontal Speeds: Walk 4.317 m/s, Sprint 5.612 m/s, Sneak/Eat 1.295 m/s, Swimming 2.2 m/s, Sprint-Swimming 3.6 m/s
	speed := 4.317
	if inWater {
		if c.isSprinting {
			speed = 3.6
		} else {
			speed = 2.2
		}
	} else if c.isSneaking || c.isEating {
		speed = 1.295
	} else if c.isSprinting {
		speed = 5.612
	}

	var fwd, strafe float64
	lenInput := math.Hypot(fwdInput, strafeInput)
	if lenInput > 0 {
		fwd = (fwdInput / lenInput) * speed * dt
		strafe = (strafeInput / lenInput) * speed * dt
	}

	radYaw := c.camera.Yaw * math.Pi / 180.0
	// Authentic Minecraft forward & right vectors:
	// Forward = (sin(Yaw), 0, cos(Yaw))
	// Right = (cos(Yaw), 0, -sin(Yaw))
	dx := math.Sin(radYaw)*fwd + math.Cos(radYaw)*strafe
	dz := math.Cos(radYaw)*fwd - math.Sin(radYaw)*strafe
	dy := c.player.Velocity.Y * dt

	// Sneak Cliff Edge Safeguard: prevent stepping off ledges when sneaking on ground
	if c.isSneaking && c.player.OnGround {
		checkGroundAt := func(px, pz float64) bool {
			feetY := int32(math.Floor(c.player.Position.Y - 0.5))
			minBX := int32(math.Floor(px - 0.25))
			maxBX := int32(math.Floor(px + 0.25))
			minBZ := int32(math.Floor(pz - 0.25))
			maxBZ := int32(math.Floor(pz + 0.25))
			for bx := minBX; bx <= maxBX; bx++ {
				for bz := minBZ; bz <= maxBZ; bz++ {
					if c.engine.GetBlock(bx, feetY, bz) != 0 {
						return true
					}
				}
			}
			return false
		}
		if !checkGroundAt(c.player.Position.X+dx, c.player.Position.Z) {
			dx = 0
		}
		if !checkGroundAt(c.player.Position.X, c.player.Position.Z+dz) {
			dz = 0
		}
	}

	delta := sim.Vec3{X: dx, Y: dy, Z: dz}
	resolvedPos, onGround, hitCeiling := c.engine.MovePlayerRelative(c.player.ID, delta, float32(c.camera.Yaw), float32(c.camera.Pitch))

	c.player.OnGround = onGround
	if onGround {
		if !inWater || !jumpRequested {
			c.player.Velocity.Y = 0
		}
	} else if hitCeiling && c.player.Velocity.Y > 0 {
		// Hit ceiling! Stop upward vertical momentum immediately
		c.player.Velocity.Y = 0
	}
	c.player.Position = resolvedPos

	// 4. Update Camera & Bobbing
	c.isMoving = (lenInput > 0)
	if c.isMoving && onGround {
		c.walkCycle += 10.0 * dt // Authentic Minecraft walk cycle speed

		// Authentic Minecraft Footstep Sound Feedback
		stepInterval := 380 * time.Millisecond
		if c.isSprinting {
			stepInterval = 270 * time.Millisecond
		} else if c.isSneaking {
			stepInterval = 550 * time.Millisecond
		}
		if time.Since(c.lastStepTime) > stepInterval {
			c.lastStepTime = time.Now()
			bx := int32(math.Floor(resolvedPos.X))
			by := int32(math.Floor(resolvedPos.Y - 0.2))
			bz := int32(math.Floor(resolvedPos.Z))
			underBlock := c.engine.GetBlock(bx, by, bz)
			switch underBlock {
			case 1, 4: // Stone, Cobblestone
				sound.PlayHardwareFeedback(sound.SoundBlockStoneStep)
			case 5, 17: // Planks, Log
				sound.PlayHardwareFeedback(sound.SoundBlockWoodStep)
			default: // Grass, Dirt, etc.
				sound.PlayHardwareFeedback(sound.SoundBlockGrassStep)
			}
		}
	} else if onGround {
		c.walkCycle = 0
	}

	bobY := 0.0
	bobX := 0.0
	if onGround && c.isMoving {
		bobY = math.Sin(c.walkCycle) * 0.035
		bobX = math.Cos(c.walkCycle*0.5) * 0.02
	}

	// Dynamic Sneak Eye-Height transition (1.62m walking -> 1.27m sneaking)
	targetEye := 1.62
	if c.isSneaking {
		targetEye = 1.27
	}
	c.currentEyeHeight += (targetEye - c.currentEyeHeight) * math.Min(1.0, dt*14.0)

	eyeX := resolvedPos.X + bobX
	eyeY := resolvedPos.Y + c.currentEyeHeight + bobY
	eyeZ := resolvedPos.Z

	// Dynamic FOV for sprint speed rush feel
	if c.isSprinting && c.isMoving && onGround {
		c.targetFov = 82.0
	} else {
		c.targetFov = 70.0
	}
	c.camera.Fov += (c.targetFov - c.camera.Fov) * math.Min(1.0, dt*8.0)

	// Perspective camera positioning with obstacle raycast clipping (F5)
	lx, ly, lz := c.camera.GetLookVector()
	switch c.perspectiveMode {
	case 1: // Third-person back: 3.5 blocks behind with obstacle raycast clipping
		camDist := 3.5
		ray := sim.Raycast(sim.Vec3{X: eyeX, Y: eyeY, Z: eyeZ}, sim.Vec3{X: -lx, Y: -ly, Z: -lz}, 3.5, func(x, y, z int32) bool {
			return sim.IsSolidBlock(c.engine.GetBlock(x, y, z))
		})
		if ray.Hit {
			camDist = math.Max(0.25, ray.Distance-0.25)
		}
		c.camera.X = eyeX - lx*camDist
		c.camera.Y = eyeY - ly*camDist
		c.camera.Z = eyeZ - lz*camDist
	case 2: // Third-person front: 3.5 blocks in front with obstacle raycast clipping
		camDist := 3.5
		ray := sim.Raycast(sim.Vec3{X: eyeX, Y: eyeY, Z: eyeZ}, sim.Vec3{X: lx, Y: ly, Z: lz}, 3.5, func(x, y, z int32) bool {
			return sim.IsSolidBlock(c.engine.GetBlock(x, y, z))
		})
		if ray.Hit {
			camDist = math.Max(0.25, ray.Distance-0.25)
		}
		c.camera.X = eyeX + lx*camDist
		c.camera.Y = eyeY + ly*camDist
		c.camera.Z = eyeZ + lz*camDist
	default: // First-person
		c.camera.X = eyeX
		c.camera.Y = eyeY
		c.camera.Z = eyeZ
	}

	// Hurt flash & sound detection
	if c.player.Health < c.lastPlayerHealth && c.lastPlayerHealth > 0 {
		sound.PlayHardwareFeedback(sound.SoundPlayerHurt)
		c.hurtFlashTimer = 0.4
	}
	c.lastPlayerHealth = c.player.Health
	if c.hurtFlashTimer > 0 {
		c.hurtFlashTimer -= dt
	}

	// Stream chunks as player moves
	c.EnsureChunksAroundPlayer(resolvedPos.X, resolvedPos.Z, 2)
}

func (c *Client) MoveRelative(forward, strafe float64) {
	c.PhysicsStep(forward, strafe, false, 0.05)
}

func (c *Client) Jump() {
	if c.player == nil {
		return
	}
	inWater, atSurface := c.checkWaterState()
	if inWater {
		if atSurface {
			c.player.Velocity.Y = 6.2
		} else {
			c.player.Velocity.Y = 4.2
		}
		c.player.OnGround = false
	} else if c.player.OnGround {
		c.player.Velocity.Y = 8.4
		c.player.OnGround = false
	}
}

func (c *Client) RotateView(dYaw, dPitch float64) {
	c.camera.Yaw += dYaw
	c.camera.Pitch += dPitch
	if c.camera.Pitch > 89.0 {
		c.camera.Pitch = 89.0
	}
	if c.camera.Pitch < -89.0 {
		c.camera.Pitch = -89.0
	}
}

func (c *Client) UpdateRaycastTarget() {
	if c.player == nil {
		return
	}
	origin := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	dir := sim.Vec3{X: lx, Y: ly, Z: lz}

	isSolid := func(x, y, z int32) bool {
		b := c.engine.GetBlock(x, y, z)
		return b != 0 && b != 8 && b != 9 && b != 10 && b != 11
	}

	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if res.Hit {
		c.hasTarget = true
		c.targetedPos = world.BlockPos{X: res.BlockX, Y: res.BlockY, Z: res.BlockZ}
		c.targetedFace = world.BlockPos{X: res.FaceX, Y: res.FaceY, Z: res.FaceZ}
	} else {
		c.hasTarget = false
	}
}

func (c *Client) UpdateMining(isLeftMouseDown bool, dt float64) {
	if c.player == nil || c.engine == nil {
		return
	}

	// Update block place cooldown
	if c.placeCooldown > 0 {
		c.placeCooldown -= dt
	}

	// Update active debris particles
	var activeParticles []render.Particle
	for i := range c.particles {
		p := &c.particles[i]
		p.Life -= dt
		if p.Life > 0 {
			p.X += p.Vx * dt
			p.Y += p.Vy * dt
			p.Z += p.Vz * dt
			p.Vy -= 12.0 * dt // Particle gravity
			activeParticles = append(activeParticles, *p)
		}
	}
	c.particles = activeParticles

	// Arm swing animation decay when not actively mining
	if !isLeftMouseDown {
		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		if c.isSwinging {
			c.swingProgress += dt * 4.0
			if c.swingProgress >= 1.0 {
				c.isSwinging = false
				c.swingProgress = 0.0
			}
		}
		return
	}

	// Left click is held down: start or continue swinging arm
	c.isSwinging = true
	c.swingProgress += dt * 4.0 // Authentic rhythmic arm swing (4 swings per second)
	if c.swingProgress >= 1.0 {
		c.swingProgress -= 1.0
	}

	// Mining target resolution with authentic target-lock:
	// If already mining a block, stay locked to it as long as it remains within reach (~5.5 blocks)
	var activePos world.BlockPos
	if c.hasMiningBlock {
		blockPosVec := sim.Vec3{X: float64(c.miningPos.X) + 0.5, Y: float64(c.miningPos.Y) + 0.5, Z: float64(c.miningPos.Z) + 0.5}
		camPosVec := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
		dist := camPosVec.DistanceTo(blockPosVec)
		blockID := c.engine.GetBlock(c.miningPos.X, c.miningPos.Y, c.miningPos.Z)
		if dist <= 5.5 && blockID != 0 && blockID != 7 {
			activePos = c.miningPos
		} else {
			c.hasMiningBlock = false
			c.breakProgress = 0.0
		}
	}

	if !c.hasMiningBlock {
		if !c.hasTarget {
			c.breakProgress = 0.0
			c.isMining = false
			return
		}
		blockID := c.engine.GetBlock(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
		if blockID == 0 || blockID == 7 { // Air or Bedrock
			c.breakProgress = 0.0
			c.isMining = false
			return
		}
		activePos = c.targetedPos
		c.miningPos = activePos
		c.hasMiningBlock = true
		c.breakProgress = 0.0
	}

	blockID := c.engine.GetBlock(activePos.X, activePos.Y, activePos.Z)
	if blockID == 0 || blockID == 7 {
		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		return
	}

	c.isMining = true
	heldToolID := int32(0)
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		heldToolID = sel.ItemID
	}

	dps := inventory.CalculateBlockDamagePerSecond(heldToolID, blockID)
	if c.isCreative {
		c.breakProgress = 1.0 // Instant block destruction in Creative Mode
	} else {
		c.breakProgress += dps * dt
	}

	// Periodically emit hitting sounds and small debris particles
	if time.Since(c.lastHitSoundTime) > 220*time.Millisecond {
		c.lastHitSoundTime = time.Now()
		sound.PlayHardwareFeedback("minecraft:block.hit")
		c.spawnBlockDebris(activePos, blockID, 3)
	}

	// Block breaking completion threshold (100% progress)
	if c.breakProgress >= 1.0 {
		minedPos := activePos
		c.engine.BreakBlockWithTool(c.player.ID, minedPos, heldToolID)
		c.spawnBlockDebris(minedPos, blockID, 24) // Explosion of particles
		sound.PlayHardwareFeedback(sound.GetBlockBreakSound(blockID))

		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		c.RebuildAffectedMeshes(minedPos)
	}
}

func (c *Client) spawnBlockDebris(pos world.BlockPos, blockID int32, count int) {
	col := render.SampleBlockColor(blockID, 2, 0.5, 0.5, 1.0)
	for i := 0; i < count; i++ {
		rx := (randFloat() - 0.5) * 0.8
		ry := (randFloat() - 0.5) * 0.8
		rz := (randFloat() - 0.5) * 0.8
		vx := (randFloat() - 0.5) * 3.5
		vy := randFloat()*3.0 + 1.2
		vz := (randFloat() - 0.5) * 3.5
		c.particles = append(c.particles, render.Particle{
			X:       float64(pos.X) + 0.5 + rx,
			Y:       float64(pos.Y) + 0.5 + ry,
			Z:       float64(pos.Z) + 0.5 + rz,
			Vx:      vx,
			Vy:      vy,
			Vz:      vz,
			Color:   col,
			Life:    0.4 + randFloat()*0.3,
			MaxLife: 0.7,
		})
	}
}

func randFloat() float64 {
	var b [1]byte
	rand.Read(b[:])
	return float64(b[0]) / 255.0
}

// AddChatMessage appends a new message to the client's in-game chat history.
func (c *Client) AddChatMessage(text string) {
	c.chatMessages = append(c.chatMessages, render.ChatMessage{
		Text:      text,
		Timestamp: time.Now(),
	})
	if len(c.chatMessages) > 50 {
		c.chatMessages = c.chatMessages[len(c.chatMessages)-50:]
	}
}

// ExecuteCommand parses and executes in-game commands or sends player chat.
func (c *Client) ExecuteCommand(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	if !strings.HasPrefix(input, "/") {
		// Normal player chat
		c.AddChatMessage("<" + c.username + "> " + input)
		return
	}

	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		c.AddChatMessage("Commands: /gamemode <c|s>, /fly, /time set <day|noon|night|midnight>, /give <item> [n], /tp <x> <y> <z>, /clear")

	case "/gamemode":
		if len(parts) < 2 {
			c.AddChatMessage("Usage: /gamemode <creative|survival|c|s|1|0>")
			return
		}
		mode := strings.ToLower(parts[1])
		if mode == "creative" || mode == "c" || mode == "1" {
			c.isCreative = true
			c.AddChatMessage("[GameMode] Set own game mode to Creative Mode")
		} else if mode == "survival" || mode == "s" || mode == "0" {
			c.isCreative = false
			c.isFlying = false
			c.AddChatMessage("[GameMode] Set own game mode to Survival Mode")
		} else {
			c.AddChatMessage("Unknown game mode: " + parts[1])
		}

	case "/fly":
		if !c.isCreative {
			c.isCreative = true
		}
		c.isFlying = !c.isFlying
		if c.isFlying {
			c.AddChatMessage("[Flight] Creative flight enabled (Space=Up, Shift=Down)")
		} else {
			c.AddChatMessage("[Flight] Creative flight disabled")
		}

	case "/time":
		if len(parts) >= 3 && strings.ToLower(parts[1]) == "set" {
			val := strings.ToLower(parts[2])
			if c.engine != nil {
				switch val {
				case "day":
					c.engine.SetWorldTime(1000)
					c.AddChatMessage("[Time] Set the time to 1000 (Day)")
				case "noon":
					c.engine.SetWorldTime(6000)
					c.AddChatMessage("[Time] Set the time to 6000 (Noon)")
				case "sunset":
					c.engine.SetWorldTime(12000)
					c.AddChatMessage("[Time] Set the time to 12000 (Sunset)")
				case "night":
					c.engine.SetWorldTime(13000)
					c.AddChatMessage("[Time] Set the time to 13000 (Night)")
				case "midnight":
					c.engine.SetWorldTime(18000)
					c.AddChatMessage("[Time] Set the time to 18000 (Midnight)")
				default:
					if t, err := strconv.ParseInt(val, 10, 64); err == nil {
						c.engine.SetWorldTime(t)
						c.AddChatMessage(fmt.Sprintf("[Time] Set the time to %d", t))
					} else {
						c.AddChatMessage("Usage: /time set <day|noon|sunset|night|midnight|<ticks>>")
					}
				}
			}
			return
		} else {
			c.AddChatMessage("Usage: /time set <day|noon|sunset|night|midnight|<ticks>>")
		}

	case "/give":
		if len(parts) < 2 {
			c.AddChatMessage("Usage: /give <item> [count]")
			return
		}
		itemStr := strings.ToLower(parts[1])
		count := 1
		if len(parts) >= 3 {
			if n, err := strconv.Atoi(parts[2]); err == nil && n > 0 {
				count = n
			}
		}

		var itemID int32
		switch itemStr {
		case "stone":
			itemID = 1
		case "grass", "grass_block":
			itemID = 2
		case "dirt":
			itemID = 3
		case "cobblestone", "cobble":
			itemID = 4
		case "planks", "wood_planks", "oak_planks":
			itemID = 5
		case "log", "oak_log", "wood":
			itemID = 17
		case "leaves", "oak_leaves":
			itemID = 18
		case "diamond":
			itemID = 264
		case "iron", "iron_ingot":
			itemID = 265
		case "gold", "gold_ingot":
			itemID = 266
		case "apple":
			itemID = inventory.ItemApple
		case "bread":
			itemID = inventory.ItemBread
		case "porkchop", "cooked_porkchop":
			itemID = inventory.ItemCookedPorkchop
		case "beef", "cooked_beef", "steak":
			itemID = inventory.ItemSteak
		case "pickaxe", "diamond_pickaxe":
			itemID = inventory.ItemDiamondPickaxe
		case "sword", "diamond_sword":
			itemID = 276
		case "crafting_table", "workbench":
			itemID = 58
		case "furnace":
			itemID = 61
		default:
			if id, err := strconv.Atoi(itemStr); err == nil && id > 0 {
				itemID = int32(id)
			} else {
				c.AddChatMessage("Unknown item: " + itemStr)
				return
			}
		}

		if c.player != nil && c.player.Inventory != nil {
			c.player.Inventory.AddItem(itemID, count)
			c.AddChatMessage(fmt.Sprintf("[Give] Gave %d [%s] to %s", count, itemStr, c.username))
			sound.PlayHardwareFeedback("minecraft:item.pickup")
		}

	case "/tp":
		if len(parts) < 4 {
			c.AddChatMessage("Usage: /tp <x> <y> <z>")
			return
		}
		x, err1 := strconv.ParseFloat(parts[1], 64)
		y, err2 := strconv.ParseFloat(parts[2], 64)
		z, err3 := strconv.ParseFloat(parts[3], 64)
		if err1 == nil && err2 == nil && err3 == nil && c.player != nil {
			c.player.Position = sim.Vec3{X: x, Y: y, Z: z}
			c.camera.X = x
			c.camera.Y = y + 1.62
			c.camera.Z = z
			c.EnsureChunksAroundPlayer(x, z, 2)
			c.AddChatMessage(fmt.Sprintf("[Teleport] Teleported %s to (%.1f, %.1f, %.1f)", c.username, x, y, z))
		}

	case "/clear":
		if c.player != nil && c.player.Inventory != nil {
			for i := range c.player.Inventory.Slots {
				c.player.Inventory.Slots[i] = inventory.ItemStack{}
			}
			c.AddChatMessage("[Inventory] Cleared player inventory")
		}

	default:
		c.AddChatMessage("Unknown command. Type /help for help.")
	}
}

func (c *Client) MineTargetedBlock() (int32, bool) {
	if !c.hasTarget || c.player == nil {
		return 0, false
	}
	heldToolID := int32(0)
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		heldToolID = sel.ItemID
	}
	targetPos := c.targetedPos
	blockID := c.engine.GetBlock(targetPos.X, targetPos.Y, targetPos.Z)
	drop, ok := c.engine.BreakBlockWithTool(c.player.ID, targetPos, heldToolID)
	c.spawnBlockDebris(targetPos, blockID, 20)
	sound.PlayHardwareFeedback(sound.GetBlockBreakSound(blockID))
	c.RebuildAffectedMeshes(targetPos)
	return drop, ok
}

func (c *Client) PlaceBlockInFront() bool {
	if !c.hasTarget || c.player == nil || c.engine == nil {
		return false
	}
	selected := c.player.Inventory.GetSelectedItem()
	if selected.IsEmpty() {
		return false
	}
	if !inventory.IsPlaceableBlock(selected.ItemID) {
		return false
	}

	origin := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	dir := sim.Vec3{X: lx, Y: ly, Z: lz}

	isSolid := func(x, y, z int32) bool {
		b := c.engine.GetBlock(x, y, z)
		return b != 0 && b != 8 && b != 9 && b != 10 && b != 11
	}
	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if !res.Hit {
		return false
	}

	hitBlockID := c.engine.GetBlock(res.BlockX, res.BlockY, res.BlockZ)
	var targetPos world.BlockPos
	if inventory.IsReplaceableBlock(hitBlockID) {
		// Replaceable grass/flowers are directly replaced at their position
		targetPos = world.BlockPos{X: res.BlockX, Y: res.BlockY, Z: res.BlockZ}
		if sim.IsSolidBlock(selected.ItemID) {
			bBox := sim.NewBlockAABB(targetPos.X, targetPos.Y, targetPos.Z)
			pBox := sim.NewPlayerAABB(c.player.Position)
			if sim.PlacementIntersects(pBox, bBox) {
				// Player is standing in this foliage! Pass through foliage to hit the solid block underneath/behind
				solidRes := sim.Raycast(origin, dir, 5.0, func(x, y, z int32) bool {
					b := c.engine.GetBlock(x, y, z)
					return sim.IsSolidBlock(b)
				})
				if solidRes.Hit {
					targetPos = world.BlockPos{
						X: solidRes.BlockX + solidRes.FaceX,
						Y: solidRes.BlockY + solidRes.FaceY,
						Z: solidRes.BlockZ + solidRes.FaceZ,
					}
				}
			}
		}
	} else {
		// Solid block: place against exposed face
		targetPos = world.BlockPos{
			X: res.BlockX + res.FaceX,
			Y: res.BlockY + res.FaceY,
			Z: res.BlockZ + res.FaceZ,
		}
	}

	ok := c.engine.PlaceBlock(c.player.ID, targetPos, selected.ItemID)
	if ok {
		sound.PlayHardwareFeedback(sound.GetBlockPlaceSound(selected.ItemID))
		c.RebuildAffectedMeshes(targetPos)
		c.isSwinging = true
		c.swingProgress = 0.0
	}
	return ok
}

// UpdateRightClick handles eating food and placing blocks/interacting with blocks.
func (c *Client) UpdateRightClick(isRightDown bool, dt float64) {
	if c.player == nil || c.engine == nil {
		return
	}

	sel := c.player.Inventory.GetSelectedItem()

	// 1. Food Consumption
	if !sel.IsEmpty() && inventory.IsFood(sel.ItemID) && c.player.Hunger < 20 {
		if isRightDown {
			c.isEating = true
			c.eatTimer += dt

			// Periodic munching crunch sound (every 0.22s)
			if time.Since(c.lastMunchSound) > 220*time.Millisecond {
				c.lastMunchSound = time.Now()
				sound.PlayHardwareFeedback("minecraft:player.eat")
				c.spawnBlockDebris(world.BlockPos{
					X: int32(math.Floor(c.camera.X)),
					Y: int32(math.Floor(c.camera.Y)),
					Z: int32(math.Floor(c.camera.Z)),
				}, sel.ItemID, 2)
			}

			// Eat complete (1.6s canonical Minecraft duration)
			if c.eatTimer >= 1.6 {
				c.engine.ConsumeFood(c.player.ID, c.player.Inventory.SelectedHotbar)
				sound.PlayHardwareFeedback("minecraft:player.burp")
				c.isEating = false
				c.eatTimer = 0.0
			}
			return
		} else {
			c.isEating = false
			c.eatTimer = 0.0
		}
	} else {
		c.isEating = false
		c.eatTimer = 0.0
	}

	// 2. Block Placement or Interactive Block Clicking (when Right Click is held)
	if isRightDown {
		if c.placeCooldown <= 0 {
			// Check if clicking on Crafting Table
			if c.hasTarget {
				targetID := c.engine.GetBlock(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
				if targetID == 58 { // Crafting Table
					c.showInventoryGUI = !c.showInventoryGUI
					c.placeCooldown = 0.3
					return
				}
			}

			if c.PlaceBlockInFront() {
				c.placeCooldown = 0.25
				sound.PlayHardwareFeedback(sound.GetBlockPlaceSound(sel.ItemID))
			}
		}
	}
}

// RenderFrame renders Minecraft sky, perspective-correct textured voxels, block outline, animated hand, and HUD.
func (c *Client) RenderFrame() {
	c.UpdateRaycastTarget()

	// Hotbar equip animation trigger on slot change
	if c.player != nil && c.player.Inventory != nil {
		if c.player.Inventory.SelectedHotbar != c.lastSelectedSlot {
			c.lastSelectedSlot = c.player.Inventory.SelectedHotbar
			c.equipProgress = 0.0
		}
	}

	// 1. Sky gradient background with dynamic day/night celestial simulation & distance fog
	worldTime := int64(6000)
	if c.engine != nil {
		worldTime = c.engine.GetWorldTime()
	}
	c.framebuf.ClearSkyWithCamera(worldTime, c.camera)
	c.framebuf.DrawCelestialBodies(c.camera, worldTime)

	// 2. Render cached 3D chunk meshes with authentic two-pass rendering (Pass 1: Opaque, Pass 2: Translucent Water)
	c.meshMu.RLock()
	lx, ly, lz := c.camera.GetLookVector()
	camX, camY, camZ := c.camera.X, c.camera.Y, c.camera.Z

	type visibleSection struct {
		mesh        *render.Mesh
		distForward float64
	}
	visibleSections := make([]visibleSection, 0, len(c.meshCache))
	for key, mesh := range c.meshCache {
		if mesh == nil {
			continue
		}

		// Section bounding sphere culling against camera plane (safe -18.0 threshold)
		scx := float64(key[0]*16 + 8)
		scy := float64(key[1]*16 + 8)
		scz := float64(key[2]*16 + 8)
		distForward := (scx-camX)*lx + (scy-camY)*ly + (scz-camZ)*lz
		if distForward < -18.0 {
			// Entire section is safely behind camera plane, skip!
			continue
		}
		visibleSections = append(visibleSections, visibleSection{mesh: mesh, distForward: distForward})
	}
	c.meshMu.RUnlock()

	// Pass 1: Render all OPAQUE solid/cutout terrain blocks first (Stone, Dirt, Sand, Wood, Leaves, etc.)
	for _, sec := range visibleSections {
		mesh := sec.mesh
		for i := 0; i < len(mesh.Indices); i += 3 {
			v0 := mesh.Vertices[mesh.Indices[i]]
			if v0.BlockID == 8 || v0.BlockID == 9 {
				continue // Deferred to translucent Pass 2 to eliminate all water flickering
			}
			v1 := mesh.Vertices[mesh.Indices[i+1]]
			v2 := mesh.Vertices[mesh.Indices[i+2]]

			// World-space exact normal culling for planar voxel face
			switch v0.FaceIndex {
			case 0: // +X face (visible only if camX > v0.X)
				if camX <= float64(v0.X) {
					continue
				}
			case 1: // -X face (visible only if camX < v0.X)
				if camX >= float64(v0.X) {
					continue
				}
			case 2: // +Y face (top, visible only if camY > v0.Y)
				if camY <= float64(v0.Y) {
					continue
				}
			case 3: // -Y face (bottom, visible only if camY < v0.Y)
				if camY >= float64(v0.Y) {
					continue
				}
			case 4: // +Z face (visible only if camZ > v0.Z)
				if camZ <= float64(v0.Z) {
					continue
				}
			case 5: // -Z face (visible only if camZ < v0.Z)
				if camZ >= float64(v0.Z) {
					continue
				}
			}

			vx0, vy0, vz0 := c.camera.ToViewSpace(float64(v0.X), float64(v0.Y), float64(v0.Z))
			vx1, vy1, vz1 := c.camera.ToViewSpace(float64(v1.X), float64(v1.Y), float64(v1.Z))
			vx2, vy2, vz2 := c.camera.ToViewSpace(float64(v2.X), float64(v2.Y), float64(v2.Z))

			vv0 := render.ViewVertex{X: vx0, Y: vy0, Z: vz0, U: v0.U, V: v0.V, BlockID: v0.BlockID, FaceIndex: v0.FaceIndex, LightingMult: v0.LightingMult}
			vv1 := render.ViewVertex{X: vx1, Y: vy1, Z: vz1, U: v1.U, V: v1.V, BlockID: v1.BlockID, FaceIndex: v1.FaceIndex, LightingMult: v1.LightingMult}
			vv2 := render.ViewVertex{X: vx2, Y: vy2, Z: vz2, U: v2.U, V: v2.V, BlockID: v2.BlockID, FaceIndex: v2.FaceIndex, LightingMult: v2.LightingMult}

			clipped, count := render.ClipTriangleNear(vv0, vv1, vv2, 0.10)
			for ci := 0; ci < count; ci++ {
				tri := clipped[ci]
				p0 := c.camera.ProjectViewToScreen(tri.V0.X, tri.V0.Y, tri.V0.Z, c.framebuf.Width, c.framebuf.Height)
				p1 := c.camera.ProjectViewToScreen(tri.V1.X, tri.V1.Y, tri.V1.Z, c.framebuf.Width, c.framebuf.Height)
				p2 := c.camera.ProjectViewToScreen(tri.V2.X, tri.V2.Y, tri.V2.Z, c.framebuf.Width, c.framebuf.Height)

				sv0 := render.NewScreenVertex(p0, tri.V0.U, tri.V0.V, tri.V0.BlockID, tri.V0.FaceIndex, tri.V0.LightingMult)
				sv1 := render.NewScreenVertex(p1, tri.V1.U, tri.V1.V, tri.V1.BlockID, tri.V1.FaceIndex, tri.V1.LightingMult)
				sv2 := render.NewScreenVertex(p2, tri.V2.U, tri.V2.V, tri.V2.BlockID, tri.V2.FaceIndex, tri.V2.LightingMult)

				c.framebuf.DrawVoxelTrianglePerspective(sv0, sv1, sv2)
			}
		}
	}

	// 3. Draw targeted block wireframe outline and progressive destruction cracks
	if c.hasTarget {
		c.framebuf.DrawBlockOutline(c.camera, c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
		if c.isMining && c.breakProgress > 0 {
			stage := int(c.breakProgress * 10)
			if stage > 9 {
				stage = 9
			}
			c.framebuf.DrawBlockDamage(c.camera, c.miningPos.X, c.miningPos.Y, c.miningPos.Z, stage)
		}

		// Draw subtle placement ghost wireframe when player holds a placeable block
		if c.player != nil && c.player.Inventory != nil {
			sel := c.player.Inventory.GetSelectedItem()
			if inventory.IsPlaceableBlock(sel.ItemID) {
				hitBlockID := c.engine.GetBlock(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
				var placePos world.BlockPos
				if inventory.IsReplaceableBlock(hitBlockID) {
					placePos = c.targetedPos
				} else {
					placePos = world.BlockPos{
						X: c.targetedPos.X + c.targetedFace.X,
						Y: c.targetedPos.Y + c.targetedFace.Y,
						Z: c.targetedPos.Z + c.targetedFace.Z,
					}
				}
				c.framebuf.DrawPlacementGhostOutline(c.camera, placePos.X, placePos.Y, placePos.Z)
			}
		}
	}

	// 3.5 Render 3D physical debris particles
	if len(c.particles) > 0 {
		c.framebuf.DrawParticles(c.camera, c.particles)
	}

	// 3.6 Render floating 3D item drops in the world
	if c.engine != nil {
		drops := c.engine.GetItemDrops()
		for _, drop := range drops {
			c.drawItemDrop(drop)
		}
	}

	// 3.7 Render 3D living mobs (Zombies, Creepers, Cows, Pigs, Skeletons)
	if c.engine != nil {
		mobs := c.engine.GetMobs()
		for _, m := range mobs {
			c.drawMob(m)
		}
	}

	// 3.8 Pass 2: Render TRANSLUCENT water blocks (back-to-front sorting for smooth blending & zero flicker)
	sort.Slice(visibleSections, func(a, b int) bool {
		return visibleSections[a].distForward > visibleSections[b].distForward
	})
	for _, sec := range visibleSections {
		mesh := sec.mesh
		for i := 0; i < len(mesh.Indices); i += 3 {
			v0 := mesh.Vertices[mesh.Indices[i]]
			if v0.BlockID != 8 && v0.BlockID != 9 {
				continue
			}
			v1 := mesh.Vertices[mesh.Indices[i+1]]
			v2 := mesh.Vertices[mesh.Indices[i+2]]

			vx0, vy0, vz0 := c.camera.ToViewSpace(float64(v0.X), float64(v0.Y), float64(v0.Z))
			vx1, vy1, vz1 := c.camera.ToViewSpace(float64(v1.X), float64(v1.Y), float64(v1.Z))
			vx2, vy2, vz2 := c.camera.ToViewSpace(float64(v2.X), float64(v2.Y), float64(v2.Z))

			vv0 := render.ViewVertex{X: vx0, Y: vy0, Z: vz0, U: v0.U, V: v0.V, BlockID: v0.BlockID, FaceIndex: v0.FaceIndex, LightingMult: v0.LightingMult}
			vv1 := render.ViewVertex{X: vx1, Y: vy1, Z: vz1, U: v1.U, V: v1.V, BlockID: v1.BlockID, FaceIndex: v1.FaceIndex, LightingMult: v1.LightingMult}
			vv2 := render.ViewVertex{X: vx2, Y: vy2, Z: vz2, U: v2.U, V: v2.V, BlockID: v2.BlockID, FaceIndex: v2.FaceIndex, LightingMult: v2.LightingMult}

			clipped, count := render.ClipTriangleNear(vv0, vv1, vv2, 0.10)
			for ci := 0; ci < count; ci++ {
				tri := clipped[ci]
				p0 := c.camera.ProjectViewToScreen(tri.V0.X, tri.V0.Y, tri.V0.Z, c.framebuf.Width, c.framebuf.Height)
				p1 := c.camera.ProjectViewToScreen(tri.V1.X, tri.V1.Y, tri.V1.Z, c.framebuf.Width, c.framebuf.Height)
				p2 := c.camera.ProjectViewToScreen(tri.V2.X, tri.V2.Y, tri.V2.Z, c.framebuf.Width, c.framebuf.Height)

				sv0 := render.NewScreenVertex(p0, tri.V0.U, tri.V0.V, tri.V0.BlockID, tri.V0.FaceIndex, tri.V0.LightingMult)
				sv1 := render.NewScreenVertex(p1, tri.V1.U, tri.V1.V, tri.V1.BlockID, tri.V1.FaceIndex, tri.V1.LightingMult)
				sv2 := render.NewScreenVertex(p2, tri.V2.U, tri.V2.V, tri.V2.BlockID, tri.V2.FaceIndex, tri.V2.LightingMult)

				c.framebuf.DrawVoxelTrianglePerspective(sv0, sv1, sv2)
			}
		}
	}

	// 4. Draw player model in third person or held item in first person
	if c.perspectiveMode != 0 {
		c.drawPlayerModel()
	} else {
		c.drawHeldItem()
	}

	// 4.5 Draw hurt vignette if recently damaged
	if c.hurtFlashTimer > 0 {
		c.framebuf.DrawHurtVignette(float32(c.hurtFlashTimer * 2.5))
	}

	// 4.6 Draw underwater aquatic blue overlay only when camera is truly submerged under water
	if c.engine != nil {
		bx := int32(math.Floor(c.camera.X))
		by := int32(math.Floor(c.camera.Y))
		bz := int32(math.Floor(c.camera.Z))
		camEyeBlock := c.engine.GetBlock(bx, by, bz)
		aboveEyeBlock := c.engine.GetBlock(bx, by+1, bz)
		// Truly submerged: block at eye is water AND block above eye is water, or eye is deeply submerged (not skimming air surface)
		if (camEyeBlock == 8 || camEyeBlock == 9) && (aboveEyeBlock == 8 || aboveEyeBlock == 9 || c.camera.Y-float64(by) < 0.75) {
			c.framebuf.DrawUnderwaterOverlay()
		}
	}

	// 5. Draw 2D HUD (Centered crosshair, 20 HP hearts, 20 hunger drumsticks, 9 hotbar slots with item icons & counts)
	health := float32(20.0)
	hunger := 20
	selectedSlot := 0
	var hotbarSlots [9]inventory.ItemStack
	if c.player != nil && c.player.Inventory != nil {
		health = c.player.Health
		hunger = c.player.Hunger
		selectedSlot = c.player.Inventory.SelectedHotbar
		for i := 0; i < 9; i++ {
			hotbarSlots[i] = c.player.Inventory.Slots[i]
		}
	}
	c.framebuf.DrawHUDWithInventory(health, hunger, selectedSlot, hotbarSlots)

	// 6. Draw 2D Inventory & Crafting GUI if toggled (E key)
	if c.showInventoryGUI && c.player != nil && c.player.Inventory != nil {
		c.framebuf.DrawInventoryGUI(c.player.Inventory)
	}

	// 7. Draw F3 Debug Screen overlay if toggled (F3 key)
	if c.showDebugF3 && c.player != nil {
		c.fpsCounter++
		if time.Since(c.fpsTimer) >= 1*time.Second {
			c.currentFPS = c.fpsCounter
			c.fpsCounter = 0
			c.fpsTimer = time.Now()
		}

		yawNorm := math.Mod(c.camera.Yaw, 360.0)
		if yawNorm < 0 {
			yawNorm += 360.0
		}
		facing := "south (towards +Z)"
		if yawNorm >= 45.0 && yawNorm < 135.0 {
			facing = "west (towards -X)"
		} else if yawNorm >= 135.0 && yawNorm < 225.0 {
			facing = "north (towards -Z)"
		} else if yawNorm >= 225.0 && yawNorm < 315.0 {
			facing = "east (towards +X)"
		}

		bx := int32(math.Floor(c.player.Position.X))
		by := int32(math.Floor(c.player.Position.Y))
		bz := int32(math.Floor(c.player.Position.Z))
		cx := int32(math.Floor(float64(bx) / 16.0))
		cz := int32(math.Floor(float64(bz) / 16.0))

		targetName := "minecraft:air"
		if c.hasTarget && c.engine != nil {
			tID := c.engine.GetBlock(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
			switch tID {
			case 1:
				targetName = "minecraft:stone"
			case 2:
				targetName = "minecraft:grass_block"
			case 3:
				targetName = "minecraft:dirt"
			case 4:
				targetName = "minecraft:cobblestone"
			case 5:
				targetName = "minecraft:oak_planks"
			case 7:
				targetName = "minecraft:bedrock"
			case 12:
				targetName = "minecraft:sand"
			case 13:
				targetName = "minecraft:gravel"
			case 14:
				targetName = "minecraft:gold_ore"
			case 15:
				targetName = "minecraft:iron_ore"
			case 16:
				targetName = "minecraft:coal_ore"
			case 17:
				targetName = "minecraft:oak_log"
			case 18:
				targetName = "minecraft:oak_leaves"
			case 56:
				targetName = "minecraft:diamond_ore"
			case 58:
				targetName = "minecraft:crafting_table"
			case 61:
				targetName = "minecraft:furnace"
			default:
				targetName = "minecraft:block"
			}
		}

		fpsDisp := c.currentFPS
		if fpsDisp <= 0 {
			fpsDisp = 60
		}
		ftMs := 1000.0 / float64(fpsDisp)

		dInfo := render.DebugInfo{
			FPS:           fpsDisp,
			FrameTimeMs:   ftMs,
			X:             c.player.Position.X,
			Y:             c.player.Position.Y,
			Z:             c.player.Position.Z,
			BlockX:        bx,
			BlockY:        by,
			BlockZ:        bz,
			ChunkX:        cx,
			ChunkZ:        cz,
			Facing:        facing,
			Yaw:           c.camera.Yaw,
			Pitch:         c.camera.Pitch,
			TimeOfDay:     worldTime,
			TargetedBlock: targetName,
			HasTarget:     c.hasTarget,
			TargetPos:     [3]int32{c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z},
		}
		c.framebuf.DrawDebugScreen(dInfo)
	}

	// 8. Draw in-game chat log & command input bar
	c.framebuf.DrawChatOverlay(c.chatMessages, c.isChatOpen, c.chatInput)
}

func (c *Client) drawMob(m *sim.MobEntity) {
	pos := m.Position
	dist := math.Hypot(math.Hypot(c.camera.X-pos.X, c.camera.Y-pos.Y), c.camera.Z-pos.Z)
	if dist > 64.0 {
		return
	}

	yaw := float64(m.Yaw)
	pitch := float64(m.Pitch)
	hurtFlash := m.Health < m.MaxHealth && (int(m.Health*10)%2 == 0)

	switch m.Type {
	case sim.MobZombie:
		headCol := uint32(0xFF4C7838)  // Zombie green
		torsoCol := uint32(0xFF2E8690) // Cyan shirt
		pantsCol := uint32(0xFF2A3663) // Indigo pants
		if hurtFlash {
			headCol = 0xFFFF3333
			torsoCol = 0xFFFF2222
			pantsCol = 0xFFFF1111
		}
		// Head (facing yaw & pitch)
		c.render3DBox(pos.X, pos.Y+1.45, pos.Z, 0.22, 0.22, 0.22, yaw, pitch, headCol, headCol, headCol)
		// Torso
		c.render3DBox(pos.X, pos.Y+0.88, pos.Z, 0.24, 0.35, 0.12, yaw, 0, torsoCol, torsoCol, torsoCol)
		// Arms outstretched forward (Minecraft classic Zombie pose: 90 deg pitch)
		radY := yaw * math.Pi / 180.0
		cosY, sinY := math.Cos(radY), math.Sin(radY)
		armOffX := 0.35 * cosY
		armOffZ := -0.35 * sinY
		forwardX := 0.25 * sinY
		forwardZ := 0.25 * cosY
		c.render3DBox(pos.X-armOffX+forwardX, pos.Y+0.88, pos.Z-armOffZ+forwardZ, 0.11, 0.12, 0.35, yaw, 0, torsoCol, headCol, torsoCol)
		c.render3DBox(pos.X+armOffX+forwardX, pos.Y+0.88, pos.Z+armOffZ+forwardZ, 0.11, 0.12, 0.35, yaw, 0, torsoCol, headCol, torsoCol)
		// Legs
		legOffX := 0.12 * cosY
		legOffZ := -0.12 * sinY
		c.render3DBox(pos.X-legOffX, pos.Y+0.36, pos.Z-legOffZ, 0.11, 0.35, 0.12, yaw, 0, pantsCol, pantsCol, pantsCol)
		c.render3DBox(pos.X+legOffX, pos.Y+0.36, pos.Z+legOffZ, 0.11, 0.35, 0.12, yaw, 0, pantsCol, pantsCol, pantsCol)

	case sim.MobCreeper:
		headCol := uint32(0xFF48B33B) // Creeper green
		bodyCol := uint32(0xFF3DA330)
		legCol := uint32(0xFF359428)
		if m.FuseTicks > 0 && (m.FuseTicks%4 < 2) {
			headCol = 0xFFFFFFFF // Flashing fuse white
			bodyCol = 0xFFEEEEEE
			legCol = 0xFFDDDDDD
		} else if hurtFlash {
			headCol = 0xFFFF3333
			bodyCol = 0xFFFF2222
			legCol = 0xFFFF1111
		}
		// Creeper Head
		c.render3DBox(pos.X, pos.Y+1.35, pos.Z, 0.22, 0.22, 0.22, yaw, pitch, headCol, headCol, headCol)
		// Creeper Body
		c.render3DBox(pos.X, pos.Y+0.75, pos.Z, 0.20, 0.38, 0.12, yaw, 0, bodyCol, bodyCol, bodyCol)
		// Creeper 4 Feet
		radY := yaw * math.Pi / 180.0
		cosY, sinY := math.Cos(radY), math.Sin(radY)
		for _, f := range [][2]float64{{-0.12, -0.12}, {0.12, -0.12}, {-0.12, 0.12}, {0.12, 0.12}} {
			fx := f[0]*cosY + f[1]*sinY
			fz := -f[0]*sinY + f[1]*cosY
			c.render3DBox(pos.X+fx, pos.Y+0.18, pos.Z+fz, 0.08, 0.18, 0.08, yaw, 0, legCol, legCol, legCol)
		}

	case sim.MobPig:
		bodyCol := uint32(0xFFF0A0A0) // Pig pink
		headCol := uint32(0xFFF0A0A0)
		snoutCol := uint32(0xFFE88A8A)
		legCol := uint32(0xFFE08E8E)
		if hurtFlash {
			bodyCol = 0xFFFF3333
			headCol = 0xFFFF3333
			legCol = 0xFFFF2222
		}
		radY := yaw * math.Pi / 180.0
		cosY, sinY := math.Cos(radY), math.Sin(radY)
		// Horizontal Quadruped Body
		c.render3DBox(pos.X, pos.Y+0.55, pos.Z, 0.26, 0.22, 0.38, yaw, 0, bodyCol, bodyCol, bodyCol)
		// Head
		frontX := 0.42 * sinY
		frontZ := 0.42 * cosY
		c.render3DBox(pos.X+frontX, pos.Y+0.65, pos.Z+frontZ, 0.20, 0.20, 0.20, yaw, pitch, headCol, headCol, headCol)
		// Snout
		snoutX := 0.58 * sinY
		snoutZ := 0.58 * cosY
		c.render3DBox(pos.X+snoutX, pos.Y+0.60, pos.Z+snoutZ, 0.10, 0.08, 0.06, yaw, pitch, snoutCol, snoutCol, snoutCol)
		// 4 Legs
		for _, f := range [][2]float64{{-0.15, -0.22}, {0.15, -0.22}, {-0.15, 0.22}, {0.15, 0.22}} {
			lx := f[0]*cosY + f[1]*sinY
			lz := -f[0]*sinY + f[1]*cosY
			c.render3DBox(pos.X+lx, pos.Y+0.20, pos.Z+lz, 0.08, 0.20, 0.08, yaw, 0, legCol, legCol, legCol)
		}

	case sim.MobCow:
		bodyCol := uint32(0xFF5A4432)    // Brown cow body
		spottedCol := uint32(0xFFE5E5E5) // White spots
		headCol := uint32(0xFF5A4432)
		hornCol := uint32(0xFFCCCCCC)
		legCol := uint32(0xFF423225)
		if hurtFlash {
			bodyCol = 0xFFFF3333
			headCol = 0xFFFF3333
			legCol = 0xFFFF2222
		}
		radY := yaw * math.Pi / 180.0
		cosY, sinY := math.Cos(radY), math.Sin(radY)
		// Body
		c.render3DBox(pos.X, pos.Y+0.75, pos.Z, 0.30, 0.28, 0.45, yaw, 0, spottedCol, bodyCol, bodyCol)
		// Head
		frontX := 0.50 * sinY
		frontZ := 0.50 * cosY
		c.render3DBox(pos.X+frontX, pos.Y+0.90, pos.Z+frontZ, 0.22, 0.22, 0.22, yaw, pitch, headCol, headCol, headCol)
		// Horns
		c.render3DBox(pos.X+frontX-0.18*cosY, pos.Y+1.08, pos.Z+frontZ+0.18*sinY, 0.04, 0.08, 0.04, yaw, 0, hornCol, hornCol, hornCol)
		c.render3DBox(pos.X+frontX+0.18*cosY, pos.Y+1.08, pos.Z+frontZ-0.18*sinY, 0.04, 0.08, 0.04, yaw, 0, hornCol, hornCol, hornCol)
		// 4 Legs
		for _, f := range [][2]float64{{-0.18, -0.28}, {0.18, -0.28}, {-0.18, 0.28}, {0.18, 0.28}} {
			lx := f[0]*cosY + f[1]*sinY
			lz := -f[0]*sinY + f[1]*cosY
			c.render3DBox(pos.X+lx, pos.Y+0.28, pos.Z+lz, 0.09, 0.28, 0.09, yaw, 0, legCol, legCol, legCol)
		}

	case sim.MobSkeleton:
		boneCol := uint32(0xFFE8E8E8) // Bone white
		if hurtFlash {
			boneCol = 0xFFFF3333
		}
		// Head
		c.render3DBox(pos.X, pos.Y+1.45, pos.Z, 0.20, 0.20, 0.20, yaw, pitch, boneCol, boneCol, boneCol)
		// Ribcage / Torso
		c.render3DBox(pos.X, pos.Y+0.88, pos.Z, 0.20, 0.35, 0.10, yaw, 0, boneCol, boneCol, boneCol)
		// Arms holding bow forward
		radY := yaw * math.Pi / 180.0
		cosY, sinY := math.Cos(radY), math.Sin(radY)
		armOffX := 0.30 * cosY
		armOffZ := -0.30 * sinY
		forwardX := 0.20 * sinY
		forwardZ := 0.20 * cosY
		c.render3DBox(pos.X-armOffX+forwardX, pos.Y+0.88, pos.Z-armOffZ+forwardZ, 0.08, 0.08, 0.35, yaw, 0, boneCol, boneCol, boneCol)
		c.render3DBox(pos.X+armOffX+forwardX, pos.Y+0.88, pos.Z+armOffZ+forwardZ, 0.08, 0.08, 0.35, yaw, 0, boneCol, boneCol, boneCol)
		// Legs
		legOffX := 0.10 * cosY
		legOffZ := -0.10 * sinY
		c.render3DBox(pos.X-legOffX, pos.Y+0.36, pos.Z-legOffZ, 0.08, 0.35, 0.08, yaw, 0, boneCol, boneCol, boneCol)
		c.render3DBox(pos.X+legOffX, pos.Y+0.36, pos.Z+legOffZ, 0.08, 0.35, 0.08, yaw, 0, boneCol, boneCol, boneCol)
	}

	// Health bar above mob if damaged
	if m.Health < m.MaxHealth {
		sp, ok := c.camera.ProjectToScreen(pos.X, pos.Y+1.9, pos.Z, c.framebuf.Width, c.framebuf.Height)
		if ok && sp.Depth > 0.2 {
			barW := 36
			barH := 3
			barX := int(sp.X) - barW/2
			barY := int(sp.Y)
			c.framebuf.DrawRect(barX, barY, barW, barH, 0xFF000000)
			hpFrac := m.Health / m.MaxHealth
			if hpFrac < 0 {
				hpFrac = 0
			}
			fillW := int(float32(barW-2) * hpFrac)
			c.framebuf.DrawRect(barX+1, barY+1, fillW, barH-2, 0xFF00FF00)
		}
	}
}

// render3DBox renders a volumetric 3D cuboid transformed by position, yaw, and pitch with depth testing.
func (c *Client) render3DBox(cx, cy, cz, hx, hy, hz, yawDeg, pitchDeg float64, colTop, colBottom, colSide uint32) {
	radY := yawDeg * math.Pi / 180.0
	radP := pitchDeg * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)
	cosP, sinP := math.Cos(radP), math.Sin(radP)

	rot := func(lx, ly, lz float64) (float64, float64, float64) {
		// Pitch around local X axis
		ry := ly*cosP - lz*sinP
		rz := ly*sinP + lz*cosP
		// Yaw around world Y axis
		rx := lx*cosY + rz*sinY
		rz2 := -lx*sinY + rz*cosY
		return cx + rx, cy + ry, cz + rz2
	}

	corners := [8][3]float64{
		{-hx, -hy, -hz},
		{hx, -hy, -hz},
		{hx, hy, -hz},
		{-hx, hy, -hz},
		{-hx, -hy, hz},
		{hx, -hy, hz},
		{hx, hy, hz},
		{-hx, hy, hz},
	}

	var screenPts [8]render.ScreenPoint
	var valid [8]bool
	for i := 0; i < 8; i++ {
		wx, wy, wz := rot(corners[i][0], corners[i][1], corners[i][2])
		screenPts[i], valid[i] = c.camera.ProjectToScreen(wx, wy, wz, c.framebuf.Width, c.framebuf.Height)
	}

	faces := [6][4]int{
		{2, 3, 7, 6}, // Top (+Y)
		{4, 5, 1, 0}, // Bottom (-Y)
		{3, 2, 1, 0}, // North (-Z)
		{6, 7, 4, 5}, // South (+Z)
		{7, 3, 0, 4}, // West (-X)
		{2, 6, 5, 1}, // East (+X)
	}

	for fIdx, f := range faces {
		if !valid[f[0]] || !valid[f[1]] || !valid[f[2]] || !valid[f[3]] {
			continue
		}
		p0, p1, p2, p3 := screenPts[f[0]], screenPts[f[1]], screenPts[f[2]], screenPts[f[3]]
		// Screen-space 2D normal check for back-face culling
		area := (p1.X-p0.X)*(p2.Y-p0.Y) - (p1.Y-p0.Y)*(p2.X-p0.X)
		if area <= 0 {
			continue
		}

		col := colSide
		if fIdx == 0 {
			col = colTop
		} else if fIdx == 1 {
			col = colBottom
		} else if fIdx >= 2 && fIdx <= 3 {
			col = render.ApplyLighting(colSide, 0.85)
		} else {
			col = render.ApplyLighting(colSide, 0.70)
		}

		c.framebuf.DrawTriangle(p0, p1, p2, col)
		c.framebuf.DrawTriangle(p0, p2, p3, col)
	}
}

// drawPlayerModel renders the authentic volumetric 3D Steve character in third-person view.
func (c *Client) drawPlayerModel() {
	if c.player == nil {
		return
	}
	pos := c.player.Position

	// Steve color palette
	headSkin := uint32(0xFFD6A07E)
	headHair := uint32(0xFF4D3B26)
	torsoCol := uint32(0xFF2E8690) // Classic Cyan shirt
	pantsCol := uint32(0xFF2A3663) // Indigo pants
	shoesCol := uint32(0xFF555555) // Gray shoes

	if c.hurtFlashTimer > 0 {
		headSkin = 0xFFFF4444
		headHair = 0xFFFF2222
		torsoCol = 0xFFFF3333
		pantsCol = 0xFFFF2222
	}

	yaw := c.camera.Yaw
	pitch := c.camera.Pitch

	walkSwing := 0.0
	if c.isMoving && c.player.OnGround {
		walkSwing = math.Sin(c.walkCycle) * 28.0
	}

	armSwing := 0.0
	if c.isSwinging {
		armSwing = math.Sin(math.Sqrt(c.swingProgress)*math.Pi) * 65.0
	}

	yOffset := 0.0
	torsoPitch := 0.0
	if c.isSneaking {
		yOffset = -0.20
		torsoPitch = 12.0
	}

	// 1. Head (rotates with camera yaw & pitch)
	c.render3DBox(pos.X, pos.Y+1.45+yOffset, pos.Z, 0.22, 0.22, 0.22, yaw, pitch, headHair, headSkin, headSkin)

	// 2. Torso (rotates with body yaw & sneak pitch)
	c.render3DBox(pos.X, pos.Y+0.88+yOffset, pos.Z, 0.24, 0.35, 0.12, yaw, torsoPitch, torsoCol, torsoCol, torsoCol)

	// Compute shoulder offsets relative to body yaw
	radY := yaw * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)
	armOffX := 0.35 * cosY
	armOffZ := -0.35 * sinY

	// 3. Left Arm (swings opposite)
	c.render3DBox(pos.X-armOffX, pos.Y+0.88+yOffset, pos.Z-armOffZ, 0.11, 0.35, 0.12, yaw, walkSwing, torsoCol, headSkin, torsoCol)

	// 4. Right Arm (swings with walk cycle + chopping attack arc)
	c.render3DBox(pos.X+armOffX, pos.Y+0.88+yOffset, pos.Z+armOffZ, 0.11, 0.35, 0.12, yaw, -walkSwing+armSwing, torsoCol, headSkin, torsoCol)

	// Hip offsets relative to body yaw
	legOffX := 0.12 * cosY
	legOffZ := -0.12 * sinY

	// 5. Left Leg (swings forward/backward)
	c.render3DBox(pos.X-legOffX, pos.Y+0.36, pos.Z-legOffZ, 0.11, 0.35, 0.12, yaw, -walkSwing, pantsCol, shoesCol, pantsCol)

	// 6. Right Leg (swings opposite)
	c.render3DBox(pos.X+legOffX, pos.Y+0.36, pos.Z+legOffZ, 0.11, 0.35, 0.12, yaw, walkSwing, pantsCol, shoesCol, pantsCol)

	// 7. Render Held Item/Block in Steve's right hand in 3D
	sel := c.player.Inventory.GetSelectedItem()
	if !sel.IsEmpty() {
		rightArmPitch := (-walkSwing + armSwing) * math.Pi / 180.0
		handForward := math.Sin(rightArmPitch) * 0.35
		handDown := math.Cos(rightArmPitch) * 0.35

		hx := pos.X + armOffX + math.Sin(radY)*handForward
		hy := pos.Y + 0.88 + yOffset - handDown
		hz := pos.Z + armOffZ + math.Cos(radY)*handForward

		if inventory.IsPlaceableBlock(sel.ItemID) {
			blockCol := render.SampleBlockColor(sel.ItemID, 2, 0.5, 0.5, 1.0)
			c.render3DBox(hx, hy, hz, 0.14, 0.14, 0.14, yaw, 0, blockCol, blockCol, blockCol)
		} else {
			toolCol := uint32(0xFFB8945F)
			if props, ok := inventory.GetToolProperties(sel.ItemID); ok {
				switch props.Tier {
				case inventory.TierStone:
					toolCol = 0xFF7D7D7D
				case inventory.TierIron:
					toolCol = 0xFFD8D8D8
				case inventory.TierDiamond:
					toolCol = 0xFF2CCCCC
				}
			}
			c.render3DBox(hx, hy, hz, 0.08, 0.20, 0.08, yaw, -walkSwing+armSwing, toolCol, 0xFF6F4F37, toolCol)
		}
	}
}

// DropHeldItem throws 1 item (or full stack if dropAll is true) from active hotbar slot into the world.
func (c *Client) DropHeldItem(dropAll bool) {
	if c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return
	}
	sel := c.player.Inventory.GetSelectedItem()
	if sel.IsEmpty() {
		return
	}

	dropCount := 1
	if dropAll || sel.Count <= 1 {
		dropCount = sel.Count
	}

	_ = c.player.Inventory.RemoveItem(c.player.Inventory.SelectedHotbar, dropCount)

	lx, ly, lz := c.camera.GetLookVector()
	spawnPos := sim.Vec3{
		X: c.camera.X + lx*0.5,
		Y: c.camera.Y + ly*0.5 - 0.1,
		Z: c.camera.Z + lz*0.5,
	}
	vel := sim.Vec3{
		X: lx * 0.35,
		Y: ly*0.35 + 0.18,
		Z: lz * 0.35,
	}

	c.engine.SpawnItemDropWithDelay(sel.ItemID, dropCount, spawnPos, vel, 1.2)
	sound.PlayHardwareFeedback(sound.SoundBlockWoodStep)
	c.isSwinging = true
	c.swingProgress = 0.0
}

func (c *Client) AttackMob() bool {
	if c.player == nil || c.engine == nil {
		return false
	}
	dmg := float32(1.0) // Bare fist base damage
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		if props, ok := inventory.GetToolProperties(sel.ItemID); ok {
			dmg = props.AttackDamage
		}
	}

	camPos := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	look := sim.Vec3{X: lx, Y: ly, Z: lz}

	mobID, killed, hit := c.engine.AttackMobInFront(camPos, look, 3.5, dmg)
	if hit {
		c.isSwinging = true
		c.swingProgress = 0.0
		sound.PlayHardwareFeedback("minecraft:entity.generic.hurt")

		// Emit red blood / strike particles
		for i := 0; i < 8; i++ {
			rx := (randFloat() - 0.5) * 0.5
			ry := (randFloat() - 0.5) * 0.5
			rz := (randFloat() - 0.5) * 0.5
			c.particles = append(c.particles, render.Particle{
				X:       c.camera.X + lx*2.0 + rx,
				Y:       c.camera.Y + ly*2.0 + ry,
				Z:       c.camera.Z + lz*2.0 + rz,
				Vx:      (randFloat() - 0.5) * 2.0,
				Vy:      randFloat()*2.0 + 1.0,
				Vz:      (randFloat() - 0.5) * 2.0,
				Color:   0xFFFF2222,
				Life:    0.3,
				MaxLife: 0.3,
			})
		}

		if killed {
			sound.PlayHardwareFeedback("minecraft:entity.generic.death")
			// Mob death puff particles
			for i := 0; i < 16; i++ {
				rx := (randFloat() - 0.5) * 0.8
				ry := (randFloat() - 0.5) * 0.8
				rz := (randFloat() - 0.5) * 0.8
				c.particles = append(c.particles, render.Particle{
					X:       c.camera.X + lx*2.0 + rx,
					Y:       c.camera.Y + ly*2.0 + ry,
					Z:       c.camera.Z + lz*2.0 + rz,
					Vx:      (randFloat() - 0.5) * 3.0,
					Vy:      randFloat()*3.0 + 1.0,
					Vz:      (randFloat() - 0.5) * 3.0,
					Color:   0xFFEEEEEE, // Smoke white
					Life:    0.5,
					MaxLife: 0.5,
				})
			}
			log.Printf("[Combat] Slew mob #%d!", mobID)
		}
		return true
	}
	return false
}

func (c *Client) TryQuickCraft(recipeIndex int) bool {
	if c.player == nil || c.player.Inventory == nil {
		return false
	}
	inv := c.player.Inventory

	findItem := func(id int32, minCount int) int {
		for i, s := range inv.Slots {
			if s.ItemID == id && s.Count >= minCount {
				return i
			}
		}
		return -1
	}

	switch recipeIndex {
	case 1: // 1 Log -> 4 Planks
		slot := findItem(inventory.ItemOakLog, 1)
		if slot >= 0 {
			_ = inv.RemoveItem(slot, 1)
			inv.AddItem(inventory.ItemOakPlanks, 4)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 2: // 2 Planks -> 4 Sticks (ID 280)
		slot := findItem(inventory.ItemOakPlanks, 2)
		if slot >= 0 {
			_ = inv.RemoveItem(slot, 2)
			inv.AddItem(280, 4)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 3: // 4 Planks -> Crafting Table
		slot := findItem(inventory.ItemOakPlanks, 4)
		if slot >= 0 {
			_ = inv.RemoveItem(slot, 4)
			inv.AddItem(inventory.ItemCraftingTable, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 4: // 3 Planks + 2 Sticks -> Wooden Pickaxe
		pSlot := findItem(inventory.ItemOakPlanks, 3)
		sSlot := findItem(280, 2)
		if pSlot >= 0 && sSlot >= 0 {
			_ = inv.RemoveItem(pSlot, 3)
			_ = inv.RemoveItem(sSlot, 2)
			inv.AddItem(inventory.ItemWoodenPickaxe, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 5: // 2 Planks + 1 Stick -> Wooden Sword
		pSlot := findItem(inventory.ItemOakPlanks, 2)
		sSlot := findItem(280, 1)
		if pSlot >= 0 && sSlot >= 0 {
			_ = inv.RemoveItem(pSlot, 2)
			_ = inv.RemoveItem(sSlot, 1)
			inv.AddItem(268, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 6: // 3 Cobble + 2 Sticks -> Stone Pickaxe
		cSlot := findItem(inventory.ItemCobblestone, 3)
		sSlot := findItem(280, 2)
		if cSlot >= 0 && sSlot >= 0 {
			_ = inv.RemoveItem(cSlot, 3)
			_ = inv.RemoveItem(sSlot, 2)
			inv.AddItem(inventory.ItemStonePickaxe, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 7: // 8 Cobble -> Furnace
		cSlot := findItem(inventory.ItemCobblestone, 8)
		if cSlot >= 0 {
			_ = inv.RemoveItem(cSlot, 8)
			inv.AddItem(inventory.ItemFurnace, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	}
	return false
}

// drawItemDrop renders a rotating, bobbing 3D voxel item entity floating in the world.
func (c *Client) drawItemDrop(drop sim.ItemDrop) {
	bobY := math.Sin(drop.Age*3.5) * 0.08
	cx, cy, cz := drop.Position.X, drop.Position.Y+bobY+0.20, drop.Position.Z
	topCol := render.SampleBlockColor(drop.ItemID, 2, 0.5, 0.5, 1.0)
	sideCol := render.SampleBlockColor(drop.ItemID, 0, 0.5, 0.5, 0.80)
	darkCol := render.SampleBlockColor(drop.ItemID, 1, 0.5, 0.5, 0.60)

	// Authentic Minecraft item rotation: continuous 90 deg/s around Y axis
	spinYaw := drop.Age * 90.0
	c.render3DBox(cx, cy, cz, 0.12, 0.12, 0.12, spinYaw, 15.0, topCol, darkCol, sideCol)
}

func darkenColor(c uint32, factor float32) uint32 {
	a := (c >> 24) & 0xFF
	r := uint32(float32((c>>16)&0xFF) * factor)
	g := uint32(float32((c>>8)&0xFF) * factor)
	b := uint32(float32(c&0xFF) * factor)
	return (a << 24) | (r << 16) | (g << 8) | b
}

func getToolSprite16x16(toolType inventory.ToolType, tier inventory.ToolTier) [16][16]uint32 {
	var s [16][16]uint32
	stickCol := uint32(0xFF8F6849)
	stickShade := uint32(0xFF5C3E28)

	headCol := uint32(0xFFB8945F) // Wood
	shineCol := uint32(0xFFD4B07B)
	darkCol := uint32(0xFF8A6B3A)
	switch tier {
	case inventory.TierStone:
		headCol = 0xFF7D7D7D
		shineCol = 0xFF9E9E9E
		darkCol = 0xFF545454
	case inventory.TierIron:
		headCol = 0xFFD8D8D8
		shineCol = 0xFFFFFFFF
		darkCol = 0xFFA0A0A0
	case inventory.TierDiamond:
		headCol = 0xFF4AEDD7
		shineCol = 0xFF8CFFEE
		darkCol = 0xFF28A896
	}

	// Stick handle from (2, 2) to (9, 9)
	for d := 1; d <= 9; d++ {
		s[d][d] = stickCol
		if d > 1 {
			s[d-1][d] = stickShade
		}
	}

	switch toolType {
	case inventory.ToolPickaxe:
		// Pickaxe head arched from (6, 13) through (10, 10) to (13, 6)
		s[10][10] = darkCol
		s[11][10] = headCol
		s[10][11] = headCol
		s[11][11] = shineCol
		s[12][9] = headCol
		s[13][8] = headCol
		s[14][7] = headCol
		s[14][6] = shineCol
		s[15][5] = headCol
		s[9][12] = headCol
		s[8][13] = headCol
		s[7][14] = headCol
		s[6][14] = shineCol
		s[5][15] = headCol
		s[11][12] = shineCol
		s[12][11] = shineCol
	case inventory.ToolSword:
		// Double-edged blade from (5, 5) to (14, 14)
		s[1][1] = headCol // Pommel
		s[2][2] = stickCol
		s[3][3] = stickCol
		// Crossguard
		s[3][5] = darkCol
		s[4][4] = darkCol
		s[5][3] = darkCol
		s[4][5] = headCol
		s[5][4] = headCol
		s[2][5] = darkCol
		s[5][2] = darkCol
		// Blade
		for d := 5; d <= 13; d++ {
			s[d][d] = headCol
			s[d+1][d] = shineCol
			s[d][d+1] = darkCol
		}
		s[14][14] = shineCol // Blade tip
	case inventory.ToolAxe:
		// Broad axe blade
		s[10][10] = darkCol
		s[10][11] = headCol
		s[11][11] = shineCol
		s[11][12] = shineCol
		s[12][12] = headCol
		s[12][13] = shineCol
		s[13][13] = headCol
		s[10][12] = headCol
		s[9][12] = headCol
		s[9][11] = darkCol
		s[13][12] = headCol
		s[13][11] = darkCol
		s[12][10] = darkCol
	case inventory.ToolShovel:
		// Spade scoop
		s[10][10] = darkCol
		s[11][11] = headCol
		s[12][12] = shineCol
		s[13][13] = shineCol
		s[12][11] = headCol
		s[11][12] = headCol
		s[13][12] = shineCol
		s[12][13] = headCol
		s[14][13] = shineCol
		s[13][14] = shineCol
	}
	return s
}

func getTorchSprite16x16() [16][16]uint32 {
	var s [16][16]uint32
	// Handle: columns 7 and 8, rows 1 to 9
	for y := 1; y <= 9; y++ {
		s[y][7] = 0xFF6F4F37
		s[y][8] = 0xFF8F6849
	}
	s[10][7] = 0xFF352B20 // Burnt top
	s[10][8] = 0xFF453628
	// Flame head
	s[11][7] = 0xFFFF6600
	s[11][8] = 0xFFFF8800
	s[12][6] = 0xFFFF6600
	s[12][7] = 0xFFFFEE55 // Hot core
	s[12][8] = 0xFFFFEE55
	s[12][9] = 0xFFFF8800
	s[13][7] = 0xFFFFFFFF // White hot tip
	s[13][8] = 0xFFFFEE55
	s[14][7] = 0xFFFF8800
	return s
}

func getFoodSprite16x16(itemID int32) [16][16]uint32 {
	var s [16][16]uint32
	switch itemID {
	case inventory.ItemApple:
		// Iconic Red Apple
		for y := 3; y <= 11; y++ {
			for x := 4; x <= 12; x++ {
				s[y][x] = 0xFFD82020
			}
		}
		s[3][4] = 0
		s[3][12] = 0
		s[11][4] = 0
		s[11][12] = 0
		s[11][8] = 0
		s[3][8] = 0
		// Glint
		s[9][5] = 0xFFFFA090
		s[8][5] = 0xFFFFA090
		s[9][6] = 0xFFFFA090
		// Stem & Leaf
		s[12][8] = 0xFF5C3A21
		s[13][8] = 0xFF5C3A21
		s[13][9] = 0xFF4E9333
		s[14][10] = 0xFF69BD46
	case inventory.ItemBread:
		// Golden Bread Loaf
		for y := 4; y <= 11; y++ {
			for x := 2; x <= 13; x++ {
				s[y][x] = 0xFFC99742
			}
		}
		s[4][2] = 0
		s[4][13] = 0
		s[11][2] = 0
		s[11][13] = 0
		for x := 4; x <= 11; x++ {
			s[11][x] = 0xFFE0B058
		}
		// Crust score lines
		s[7][5] = 0xFF8A5D20
		s[8][5] = 0xFF8A5D20
		s[7][8] = 0xFF8A5D20
		s[8][8] = 0xFF8A5D20
		s[7][11] = 0xFF8A5D20
		s[8][11] = 0xFF8A5D20
	default: // Cooked Porkchop / Steak
		for y := 4; y <= 12; y++ {
			for x := 3; x <= 12; x++ {
				s[y][x] = 0xFF9E432C
			}
		}
		s[4][3] = 0
		s[4][12] = 0
		s[12][3] = 0
		s[12][12] = 0
		for y := 6; y <= 10; y++ {
			for x := 5; x <= 10; x++ {
				s[y][x] = 0xFFB35238
			}
		}
		// Bone
		s[11][12] = 0xFFECE7E2
		s[12][13] = 0xFFECE7E2
		s[10][13] = 0xFFECE7E2
	}
	return s
}

func (c *Client) drawHeldItem() {
	if c.player == nil {
		return
	}
	selected := c.player.Inventory.GetSelectedItem()

	// Walking sway calculation
	swayX := int(math.Cos(c.walkCycle*0.5) * 6.0)
	swayY := int(math.Abs(math.Sin(c.walkCycle)) * 5.0)

	// Arm / Tool swing animation trajectory using authentic Minecraft Java curve
	swingOffset := 0
	swingDrop := 0
	if c.isSwinging {
		f1 := math.Sin(math.Sqrt(c.swingProgress) * math.Pi)
		f2 := math.Sin(c.swingProgress * math.Pi)
		swingOffset = int(f1 * 55.0)
		swingDrop = int(f2 * 35.0)
	}

	// Hotbar equip raise/lower animation
	equipDrop := int((1.0 - c.equipProgress) * 45.0)

	hx := c.framebuf.Width - 110 + swayX - swingOffset
	hy := c.framebuf.Height - 125 + swayY + swingDrop + equipDrop

	skinTone := uint32(0xFFD6A07E)   // Tan skin
	skinShade := uint32(0xFFB57C5E)  // Shading
	sleeveTone := uint32(0xFF2E8690) // Classic Cyan shirt sleeve
	cuffTone := uint32(0xFF1B5960)   // Cuff border

	if selected.IsEmpty() {
		// Steve's authentic first-person player arm / fist (3D angled limb)
		c.framebuf.DrawRect(hx+20, hy+40, 52, 65, sleeveTone)
		c.framebuf.DrawRect(hx+18, hy+38, 56, 4, cuffTone)
		// Forearm & wrist
		c.framebuf.DrawRect(hx+25, hy+15, 42, 25, skinTone)
		c.framebuf.DrawRect(hx+22, hy+2, 46, 16, skinShade)
		// Knuckles & thumb details
		c.framebuf.DrawRect(hx+20, hy+4, 12, 10, skinTone)
		c.framebuf.DrawRect(hx+32, hy+1, 8, 4, skinTone)
		c.framebuf.DrawRect(hx+42, hy+1, 8, 4, skinTone)
		return
	}

	// 1. Food Eating Hand & Jitter
	if inventory.IsFood(selected.ItemID) {
		fx := hx - 5
		fy := hy - 15
		if c.isEating {
			fy -= 18 + int(math.Sin(c.eatTimer*40.0)*4.0)
			fx -= 12 + int(math.Cos(c.eatTimer*40.0)*2.0)
		}
		sprite := getFoodSprite16x16(selected.ItemID)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := fx + x*3
				sy := fy + (15-y)*3
				c.framebuf.DrawRect(sx+1, sy+1, 3, 3, darkenColor(col, 0.65))
				c.framebuf.DrawRect(sx, sy, 3, 3, col)
			}
		}
		// Steve's grasping hand holding bottom of food item
		c.framebuf.DrawRect(fx+10, fy+32, 26, 16, skinTone)
		c.framebuf.DrawRect(fx+16, fy+46, 34, 45, sleeveTone)
		c.framebuf.DrawRect(fx+14, fy+44, 38, 4, cuffTone)
		return
	}

	// 2. Torch (ID 50)
	if selected.ItemID == 50 {
		tx := hx + 5
		ty := hy - 25
		sprite := getTorchSprite16x16()
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := tx + x*3
				sy := ty + (15-y)*3
				c.framebuf.DrawRect(sx+1, sy+1, 3, 3, darkenColor(col, 0.65))
				c.framebuf.DrawRect(sx, sy, 3, 3, col)
			}
		}
		// Hand grasping lower post
		c.framebuf.DrawRect(tx+14, ty+28, 20, 16, skinTone)
		c.framebuf.DrawRect(tx+18, ty+44, 30, 42, sleeveTone)
		c.framebuf.DrawRect(tx+16, ty+42, 34, 4, cuffTone)
		return
	}

	// 3. Tools (Pickaxe, Sword, Axe, Shovel)
	props, isTool := inventory.GetToolProperties(selected.ItemID)
	if isTool {
		sprite := getToolSprite16x16(props.Type, props.Tier)
		toolX := hx - 20
		toolY := hy - 35
		// Render authentic 2.5D extruded tool sprite at 3x scale
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := toolX + x*3
				sy := toolY + (15-y)*3
				c.framebuf.DrawRect(sx+1, sy+1, 3, 3, darkenColor(col, 0.65))
				c.framebuf.DrawRect(sx, sy, 3, 3, col)
			}
		}
		// Steve's fist grasping the lower handle
		c.framebuf.DrawRect(toolX+4, toolY+32, 22, 16, skinTone)
		c.framebuf.DrawRect(toolX+10, toolY+46, 32, 45, sleeveTone)
		c.framebuf.DrawRect(toolX+8, toolY+44, 36, 4, cuffTone)
		return
	}

	// 4. Authentic 3D Isometric Textured Voxel Block
	cx := hx - 15
	cy := hy - 18

	// Top Face (Diamond quad, Face 2, 1.0 full light)
	for ty := 0; ty < 16; ty++ {
		for tx := 0; tx < 16; tx++ {
			u := float32(tx) / 16.0
			v := float32(ty) / 16.0
			col := render.SampleBlockColor(selected.ItemID, 2, u, v, 1.0)
			sx := cx + (tx-ty)*2
			sy := cy - 16 + (tx + ty)
			c.framebuf.DrawRect(sx, sy, 2, 1, col)
		}
	}

	// Left Face (Parallelogram, Face 1, 0.8 directional shade)
	for ty := 0; ty < 16; ty++ {
		for tx := 0; tx < 16; tx++ {
			u := float32(tx) / 16.0
			v := float32(ty) / 16.0
			col := render.SampleBlockColor(selected.ItemID, 1, u, v, 0.80)
			sx := cx - 32 + tx*2
			sy := cy + tx + ty*2
			c.framebuf.DrawRect(sx, sy, 2, 2, col)
		}
	}

	// Right Face (Parallelogram, Face 0, 0.6 directional shade)
	for ty := 0; ty < 16; ty++ {
		for tx := 0; tx < 16; tx++ {
			u := float32(tx) / 16.0
			v := float32(ty) / 16.0
			col := render.SampleBlockColor(selected.ItemID, 0, u, v, 0.60)
			sx := cx + tx*2
			sy := cy + 16 - tx + ty*2
			c.framebuf.DrawRect(sx, sy, 2, 2, col)
		}
	}

	// Steve's grasping hand holding bottom-right corner of the 3D block
	c.framebuf.DrawRect(cx+12, cy+36, 26, 14, skinTone)
	c.framebuf.DrawRect(cx+20, cy+48, 36, 45, sleeveTone)
	c.framebuf.DrawRect(cx+18, cy+46, 40, 4, cuffTone)
}

func main() {
	singleplayer := flag.Bool("singleplayer", true, "Run singleplayer embedded server with procedural world")
	serverAddr := flag.String("server", "", "Multiplayer server address (e.g. 127.0.0.1:25565)")
	username := flag.String("username", "Steve", "Player username")
	benchmark := flag.Bool("benchmark", false, "Run client rendering benchmark")
	cliMode := flag.Bool("cli", false, "Run in terminal CLI mode instead of GUI window")
	width := flag.Int("width", 800, "Window width")
	height := flag.Int("height", 600, "Window height")
	flag.Parse()

	isSP := *singleplayer
	if *serverAddr != "" {
		isSP = false
	}

	client := NewClient(*serverAddr, *username, isSP, *width, *height)

	if *benchmark {
		log.Println("[Client] Running rendering benchmark (1000 frames)...")
		start := time.Now()
		for i := 0; i < 1000; i++ {
			client.RenderFrame()
		}
		elapsed := time.Since(start)
		fps := 1000.0 / elapsed.Seconds()
		mspt := elapsed.Seconds() * 1000.0 / 1000.0
		log.Printf("[Benchmark] Completed: %.2f FPS, %.3f ms/frame", fps, mspt)
		return
	}

	if !isSP {
		if err := client.Connect(); err != nil {
			log.Fatalf("[Client] Connection error: %v", err)
		}
	}

	log.Printf("==================================================")
	log.Printf("   Minecraft-Go (Java 26.3 / Horizon Parity)     ")
	log.Printf("   Player: %s | Mode: %s                        ", *username, map[bool]string{true: "Singleplayer (Embedded)", false: "Multiplayer"}[isSP])
	log.Printf("==================================================")

	if *cliMode {
		runInteractiveLoop(client)
		return
	}

	runGraphicalWindow(client, *width, *height)
}

func runGraphicalWindow(c *Client, w, h int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	title := fmt.Sprintf("Minecraft-Go [Java 26.3 Parity] - %s", c.username)
	wnd, err := render.NewNativeWindow(title, w, h)
	if err != nil {
		log.Printf("[Client] Failed to open native graphical window: %v. Falling back to CLI.", err)
		runInteractiveLoop(c)
		return
	}
	defer wnd.Close()

	log.Printf("[Client] Native 3D game window opened (%dx%d)! Controls: WASD to move, Space to jump, Mouse rotates camera directly, Left-click to mine, Right-click to place, 1-9 to select hotbar.", w, h)

	// Single key-press actions
	wnd.OnKeyClick = func(vkCode int) {
		if c.isChatOpen {
			if vkCode == 0x1B { // Escape: cancel chat
				c.isChatOpen = false
				c.chatInput = ""
				wnd.SetMouseCaptured(true)
			}
			return
		}
		switch vkCode {
		case 0x46: // F: Mine
			c.MineTargetedBlock()
		case 0x52: // R: Place
			c.PlaceBlockInFront()
		case 0x45: // E: Toggle Inventory & Crafting GUI
			if !c.showPauseMenu {
				c.showInventoryGUI = !c.showInventoryGUI
				wnd.SetMouseCaptured(!c.showInventoryGUI)
			}
		case 0x54: // T: Open Chat
			if !c.showPauseMenu && !c.showInventoryGUI {
				c.isChatOpen = true
				c.chatInput = ""
				wnd.SetMouseCaptured(false)
			}
		case 0xBF: // / (Slash): Open Command Line
			if !c.showPauseMenu && !c.showInventoryGUI {
				c.isChatOpen = true
				c.chatInput = "/"
				wnd.SetMouseCaptured(false)
			}
		case 0x1B: // Escape: Close GUI or toggle Pause Menu
			if c.showInventoryGUI {
				c.showInventoryGUI = false
				wnd.SetMouseCaptured(true)
			} else {
				c.showPauseMenu = !c.showPauseMenu
				wnd.SetMouseCaptured(!c.showPauseMenu)
			}
		case 0x72: // F3: Toggle Debug Screen
			c.showDebugF3 = !c.showDebugF3
		case 0x74: // F5: Toggle Perspective (First-Person, Third-Person Back, Third-Person Front)
			c.perspectiveMode = (c.perspectiveMode + 1) % 3
		case 0x51: // Q: Quit immediately if in pause menu, or throw/drop held item in game
			if c.showPauseMenu {
				wnd.Close()
				os.Exit(0)
			} else if !c.showInventoryGUI && !c.isChatOpen {
				c.DropHeldItem(wnd.IsKeyDown(0x11)) // Ctrl+Q drops full stack
			}
		case 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39: // 1-9
			idx := vkCode - 0x31 + 1
			if c.showInventoryGUI {
				c.TryQuickCraft(idx)
			} else if c.player != nil && !c.showPauseMenu {
				c.player.Inventory.SelectedHotbar = idx - 1
			}
		}
	}

	// Mouse wheel hotbar switching
	wnd.OnMouseWheel = func(delta int) {
		if c.showPauseMenu || c.showInventoryGUI || c.isChatOpen || c.player == nil {
			return
		}
		if delta > 0 {
			c.player.Inventory.SelectedHotbar = (c.player.Inventory.SelectedHotbar + 8) % 9
		} else if delta < 0 {
			c.player.Inventory.SelectedHotbar = (c.player.Inventory.SelectedHotbar + 1) % 9
		}
	}

	// Character input for in-game chat and slash commands
	wnd.OnCharInput = func(ch rune) {
		if !c.isChatOpen {
			return
		}
		if ch == '\r' || ch == '\n' { // Enter: execute
			cmdToRun := c.chatInput
			c.chatInput = ""
			c.isChatOpen = false
			wnd.SetMouseCaptured(true)
			c.ExecuteCommand(cmdToRun)
			return
		}
		if ch == '\b' { // Backspace
			if len(c.chatInput) > 0 {
				runes := []rune(c.chatInput)
				c.chatInput = string(runes[:len(runes)-1])
			}
			return
		}
		if ch == 27 { // Escape: cancel
			c.isChatOpen = false
			c.chatInput = ""
			wnd.SetMouseCaptured(true)
			return
		}
		if ch >= 32 { // Printable characters
			c.chatInput += string(ch)
		}
	}

	// Mouse button clicks (instant single-tap actions)
	wnd.OnMouseClick = func(btn int) {
		if c.isChatOpen {
			return
		}
		if c.showPauseMenu {
			if btn == 1 {
				mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
				// Check "Back to Game" button
				b := c.backBtnRect
				if mx >= b[0] && mx <= b[0]+b[2] && my >= b[1] && my <= b[1]+b[3] {
					c.showPauseMenu = false
					wnd.SetMouseCaptured(true)
					return
				}
				// Check "Quit Game" button
				q := c.quitBtnRect
				if mx >= q[0] && mx <= q[0]+q[2] && my >= q[1] && my <= q[1]+q[3] {
					wnd.Close()
					os.Exit(0)
				}
			}
			return
		}
		if c.showInventoryGUI {
			return
		}
		if !wnd.MouseCaptured {
			wnd.SetMouseCaptured(true)
		}
		if btn == 1 {
			// Check if attacking living mob in front
			if c.AttackMob() {
				return
			}
			// Trigger arm swing on click
			if !c.isSwinging {
				c.isSwinging = true
				c.swingProgress = 0.0
			}
		} else if btn == 2 {
			// Right click: Place block immediately if cooldown ready
			if c.placeCooldown <= 0 {
				c.UpdateRaycastTarget()
				if c.PlaceBlockInFront() {
					c.placeCooldown = 0.25
				}
			}
		}
	}

	// High-FPS precision delta-time game loop
	mouseSensitivity := 0.15
	lastTime := time.Now()

	for wnd.PollEvents() {
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		if dt <= 0 {
			dt = 0.001
		}
		if dt > 0.05 {
			dt = 0.05 // Cap max dt to avoid physics skipping
		}
		lastTime = now

		// Update item equip animation progress
		if c.equipProgress < 1.0 {
			c.equipProgress += dt * 5.0
			if c.equipProgress > 1.0 {
				c.equipProgress = 1.0
			}
		}

		// If Pause Menu is open, freeze world controls and display interactive Pause Menu
		if c.showPauseMenu {
			time.Sleep(16 * time.Millisecond)
			c.RenderFrame()
			mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
			c.backBtnRect, c.quitBtnRect = c.framebuf.DrawPauseMenu(mx, my)
			wnd.Present(c.framebuf)
			continue
		}

		// If Inventory & Crafting GUI is open, freeze world controls and display GUI
		if c.showInventoryGUI {
			time.Sleep(16 * time.Millisecond)
			c.RenderFrame()
			wnd.Present(c.framebuf)
			continue
		}

		// If Chat / Command input is open, freeze world controls and display chat overlay
		if c.isChatOpen {
			time.Sleep(16 * time.Millisecond)
			c.RenderFrame()
			wnd.Present(c.framebuf)
			continue
		}

		// 1. Spacebar Jump & Swim input (Authentic Minecraft: continuous hold for water swimming, flight ascent & bunny hop)
		spaceDown := wnd.IsKeyDown(0x20)
		c.wasSpaceDown = spaceDown

		// 2. Sprint & Sneak modifiers
		c.isSneaking = wnd.IsKeyDown(0x10)  // Shift
		c.isSprinting = wnd.IsKeyDown(0x11) // Ctrl

		// 3. Movement input polling
		var fwdInput, strafeInput float64
		if wnd.IsKeyDown(0x57) { // W
			fwdInput += 1.0
		}
		if wnd.IsKeyDown(0x53) { // S
			fwdInput -= 1.0
		}
		if wnd.IsKeyDown(0x41) { // A
			strafeInput -= 1.0
		}
		if wnd.IsKeyDown(0x44) { // D
			strafeInput += 1.0
		}

		c.PhysicsStep(fwdInput, strafeInput, spaceDown, dt)

		// Synchronize crosshair raycast target with current position & orientation
		c.UpdateRaycastTarget()

		// 4. Authentic Minecraft Mining: Hold Left Mouse to mine with cracking & tool times
		isLeftDown := wnd.IsMouseDown(1)
		c.UpdateMining(isLeftDown, dt)

		// 5. Placing Blocks or Consuming Food: Hold Right Mouse Button
		isRightDown := wnd.IsMouseDown(2)
		c.UpdateRightClick(isRightDown, dt)

		// Keyboard arrow view look fallback (smooth dt scaled)
		if wnd.IsKeyDown(0x25) { // Left (turn left)
			c.RotateView(-120.0*dt, 0)
		}
		if wnd.IsKeyDown(0x27) { // Right (turn right)
			c.RotateView(120.0*dt, 0)
		}
		if wnd.IsKeyDown(0x26) { // Up (look up)
			c.RotateView(0, 120.0*dt)
		}
		if wnd.IsKeyDown(0x28) { // Down (look down)
			c.RotateView(0, -120.0*dt)
		}

		// 2. Hardware 1:1 Mouse delta polling (Horizon & Minecraft standard centering)
		// Standard FPS mouse look: +mdx turns yaw right, -mdy pitches camera up
		mdx, mdy := wnd.PollMouseDelta()
		if mdx != 0 || mdy != 0 {
			c.RotateView(mdx*mouseSensitivity, -mdy*mouseSensitivity)
		}

		// 3. Render frame and present directly to native window
		c.RenderFrame()
		wnd.Present(c.framebuf)

		// Brief yield to avoid pegging 100% CPU while delivering maximum smoothness
		time.Sleep(1 * time.Millisecond)
	}

	log.Println("[Client] Game window closed.")
}

func runInteractiveLoop(c *Client) {
	fmt.Println("\nCommands:")
	fmt.Println("  w / s / a / d   : Move player")
	fmt.Println("  look <yaw> <pitch> : Rotate camera view")
	fmt.Println("  mine            : Mine targeted block in crosshair")
	fmt.Println("  place           : Place active hotbar block")
	fmt.Println("  slot <1-9>      : Switch active hotbar slot")
	fmt.Println("  inv             : View inventory")
	fmt.Println("  craft <recipe>  : Craft items")
	fmt.Println("  status          : View health, hunger, XYZ position")
	fmt.Println("  render          : Render a full 3D frame and HUD")
	fmt.Println("  quit            : Exit game")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("minecraft> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "w":
			c.MoveRelative(1.0, 0)
			fmt.Printf("Moved forward. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "s":
			c.MoveRelative(-1.0, 0)
			fmt.Printf("Moved backward. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "a":
			c.MoveRelative(0, -1.0)
			fmt.Printf("Strafed left. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "d":
			c.MoveRelative(0, 1.0)
			fmt.Printf("Strafed right. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "look":
			if len(parts) >= 3 {
				yaw, _ := strconv.ParseFloat(parts[1], 64)
				pitch, _ := strconv.ParseFloat(parts[2], 64)
				c.camera.Yaw = yaw
				c.camera.Pitch = pitch
				fmt.Printf("Camera rotated: Yaw=%.1f, Pitch=%.1f\n", yaw, pitch)
			}
		case "mine":
			drop, ok := c.MineTargetedBlock()
			if ok {
				fmt.Printf("Mined block! Gained item ID: %d in inventory.\n", drop)
			}
		case "place":
			ok := c.PlaceBlockInFront()
			if ok {
				fmt.Println("Placed block successfully!")
			}
		case "slot":
			if len(parts) >= 2 {
				s, _ := strconv.Atoi(parts[1])
				if s >= 1 && s <= 9 {
					c.player.Inventory.SelectedHotbar = s - 1
					fmt.Printf("Active hotbar slot switched to: %d (%+v)\n", s, c.player.Inventory.GetSelectedItem())
				}
			}
		case "inv":
			fmt.Println("=== Player Hotbar ===")
			for i := 0; i < 9; i++ {
				marker := " "
				if i == c.player.Inventory.SelectedHotbar {
					marker = "*"
				}
				slot := c.player.Inventory.Slots[i]
				fmt.Printf("[%sSlot %d] ID: %d, Count: %d\n", marker, i+1, slot.ItemID, slot.Count)
			}
		case "craft":
			if len(parts) >= 2 {
				target := strings.ToLower(parts[1])
				switch target {
				case "planks":
					if err := c.player.Inventory.RemoveItem(0, 1); err == nil {
						c.player.Inventory.AddItem(inventory.ItemOakPlanks, 4)
						fmt.Println("Crafted 4 Oak Planks!")
					}
				case "table":
					if err := c.player.Inventory.RemoveItem(0, 4); err == nil {
						c.player.Inventory.AddItem(inventory.ItemCraftingTable, 1)
						fmt.Println("Crafted 1 Crafting Table!")
					}
				case "pickaxe":
					c.player.Inventory.AddItem(inventory.ItemWoodenPickaxe, 1)
					fmt.Println("Crafted 1 Wooden Pickaxe!")
				}
			}
		case "status":
			fmt.Printf("Player: %s | Health: %.1f/20 | Hunger: %d/20 | Pos: (%.2f, %.2f, %.2f)\n",
				c.player.Username, c.player.Health, c.player.Hunger, c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "render":
			c.RenderFrame()
			fmt.Printf("Rendered 640x480 frame with 3D terrain and HUD. Active Slot: %d\n", c.player.Inventory.SelectedHotbar+1)
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		}
	}
}
