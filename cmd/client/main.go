package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
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
}

func NewClient(serverAddr, username string, isSingleplayer bool, width, height int) *Client {
	var u [16]byte
	rand.Read(u[:])

	cli := &Client{
		serverAddr:   serverAddr,
		username:     username,
		uuid:         u,
		state:        protocol.StateHandshake,
		framebuf:     render.NewFramebuffer(width, height),
		camera:       render.NewCamera(8.0, 66.62, 8.0),
		craftingMgr:  inventory.NewCraftingManager(),
		isLocal:      isSingleplayer,
		meshCache:    make(map[[3]int32]*render.Mesh),
		loadedChunks: make(map[world.ChunkPos]bool),
		lastChunkX:   -999,
		lastChunkZ:   -999,
	}

	if isSingleplayer {
		cli.engine = sim.NewEngine()
		cli.genOverworld = gen.NewGenerator(12345, gen.DimensionOverworld)
		cli.engine.StartTickLoop()
		cli.player = cli.engine.AddPlayer(username, u, sim.Vec3{X: 8.0, Y: 65.0, Z: 8.0})

		// Classic starter items in hotbar
		cli.player.Inventory.AddItem(inventory.ItemOakLog, 32)
		cli.player.Inventory.AddItem(inventory.ItemCobblestone, 64)
		cli.player.Inventory.AddItem(inventory.ItemDirt, 64)

		cli.EnsureChunksAroundPlayer(8.0, 8.0, 2)
	}

	return cli
}

// EnsureChunksAroundPlayer streams procedural world chunks and builds section meshes around the player.
func (c *Client) EnsureChunksAroundPlayer(playerX, playerZ float64, radius int32) {
	if c.genOverworld == nil || c.engine == nil {
		return
	}
	pcx := int32(math.Floor(playerX / 16.0))
	pcz := int32(math.Floor(playerZ / 16.0))

	if pcx == c.lastChunkX && pcz == c.lastChunkZ && len(c.meshCache) > 0 {
		return
	}
	c.lastChunkX = pcx
	c.lastChunkZ = pcz

	c.meshMu.Lock()
	defer c.meshMu.Unlock()

	getBlock := func(x, y, z int32) int32 {
		return c.engine.GetBlock(x, y, z)
	}

	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			cp := world.ChunkPos{X: pcx + dx, Z: pcz + dz}
			if !c.loadedChunks[cp] {
				chunk := c.genOverworld.GenerateChunk(cp)
				c.engine.LoadChunk(cp, chunk)
				c.loadedChunks[cp] = true
			}

			// Build mesh for sections Y=16 to Y=128
			for sy := int32(1); sy <= 7; sy++ {
				key := [3]int32{cp.X, sy, cp.Z}
				if c.meshCache[key] == nil {
					mesh := render.BuildSectionMesh(getBlock, cp.X*16, sy*16, cp.Z*16)
					c.meshCache[key] = mesh
				}
			}
		}
	}
}

func (c *Client) RebuildMeshes() {
	if c.player != nil {
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

// PhysicsStep executes one delta-time scaled physics step with authentic Minecraft Java Edition physics.
func (c *Client) PhysicsStep(fwdInput, strafeInput float64, jumpRequested bool, dt float64) {
	if c.player == nil {
		return
	}

	// 1. Jumping: Authentic Minecraft initial upward impulse of 8.4 m/s (0.42 blocks/tick * 20)
	if jumpRequested && c.player.OnGround {
		c.player.Velocity.Y = 8.4
		c.player.OnGround = false
	}

	// 2. Gravity: Authentic Minecraft 32.0 m/s^2 (0.08 blocks/tick^2 * 400), terminal velocity -78.4 m/s
	if !c.player.OnGround {
		c.player.Velocity.Y -= 32.0 * dt
		if c.player.Velocity.Y < -78.4 {
			c.player.Velocity.Y = -78.4
		}
	}

	// 3. Horizontal Speeds: Walk 4.317 m/s, Sprint 5.612 m/s, Sneak 1.295 m/s
	speed := 4.317
	if c.isSneaking {
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

	delta := sim.Vec3{X: dx, Y: dy, Z: dz}
	resolvedPos, onGround := c.engine.MovePlayerRelative(c.player.ID, delta, float32(c.camera.Yaw), float32(c.camera.Pitch))

	c.player.OnGround = onGround
	if onGround {
		c.player.Velocity.Y = 0
	}

	// 4. Update Camera & Bobbing
	c.isMoving = (lenInput > 0)
	if c.isMoving && onGround {
		c.walkCycle += 10.0 * dt // Authentic Minecraft walk cycle speed
	} else if onGround {
		c.walkCycle = 0
	}

	bobY := 0.0
	bobX := 0.0
	if onGround && c.isMoving {
		bobY = math.Sin(c.walkCycle) * 0.035
		bobX = math.Cos(c.walkCycle*0.5) * 0.02
	}

	c.camera.X = resolvedPos.X + bobX
	c.camera.Y = resolvedPos.Y + 1.62 + bobY // Eye level = 1.62 blocks above ground
	c.camera.Z = resolvedPos.Z

	// Stream chunks as player moves
	c.EnsureChunksAroundPlayer(resolvedPos.X, resolvedPos.Z, 2)
}

func (c *Client) MoveRelative(forward, strafe float64) {
	c.PhysicsStep(forward, strafe, false, 0.05)
}

func (c *Client) Jump() {
	if c.player != nil && c.player.OnGround {
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
		return c.engine.GetBlock(x, y, z) != 0
	}

	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if res.Hit {
		c.hasTarget = true
		c.targetedPos = world.BlockPos{X: res.BlockX, Y: res.BlockY, Z: res.BlockZ}
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

	if !c.hasTarget {
		// Punching air
		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		return
	}

	blockID := c.engine.GetBlock(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
	if blockID == 0 || blockID == 7 { // Air or Bedrock cannot be broken
		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		return
	}

	// Target changed: reset breaking progress
	if !c.hasMiningBlock || c.miningPos != c.targetedPos {
		c.miningPos = c.targetedPos
		c.hasMiningBlock = true
		c.breakProgress = 0.0
	}

	c.isMining = true
	heldToolID := int32(0)
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		heldToolID = sel.ItemID
	}

	dps := inventory.CalculateBlockDamagePerSecond(heldToolID, blockID)
	c.breakProgress += dps * dt

	// Periodically emit hitting sounds and small debris particles
	if time.Since(c.lastHitSoundTime) > 220*time.Millisecond {
		c.lastHitSoundTime = time.Now()
		sound.PlayHardwareFeedback("minecraft:block.hit")
		c.spawnBlockDebris(c.miningPos, blockID, 3)
	}

	// Block breaking completion threshold (100% progress)
	if c.breakProgress >= 1.0 {
		c.engine.BreakBlockWithTool(c.player.ID, c.miningPos, heldToolID)
		c.spawnBlockDebris(c.miningPos, blockID, 20) // Explosion of particles
		sound.PlayHardwareFeedback(sound.GetBlockBreakSound(blockID))

		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		c.RebuildMeshes()
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

func (c *Client) MineTargetedBlock() (int32, bool) {
	if !c.hasTarget || c.player == nil {
		return 0, false
	}
	heldToolID := int32(0)
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		heldToolID = sel.ItemID
	}
	drop, ok := c.engine.BreakBlockWithTool(c.player.ID, c.targetedPos, heldToolID)
	c.spawnBlockDebris(c.targetedPos, drop, 16)
	c.RebuildMeshes()
	return drop, ok
}

func (c *Client) PlaceBlockInFront() bool {
	if !c.hasTarget || c.player == nil {
		return false
	}
	selected := c.player.Inventory.GetSelectedItem()
	if selected.IsEmpty() {
		return false
	}

	origin := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	dir := sim.Vec3{X: lx, Y: ly, Z: lz}

	isSolid := func(x, y, z int32) bool {
		return c.engine.GetBlock(x, y, z) != 0
	}
	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if !res.Hit {
		return false
	}

	targetPos := world.BlockPos{
		X: res.BlockX + res.FaceX,
		Y: res.BlockY + res.FaceY,
		Z: res.BlockZ + res.FaceZ,
	}

	ok := c.engine.PlaceBlock(c.player.ID, targetPos, selected.ItemID)
	if ok {
		c.RebuildMeshes()
	}
	return ok
}

// RenderFrame renders Minecraft sky, perspective-correct textured voxels, block outline, animated hand, and HUD.
func (c *Client) RenderFrame() {
	c.UpdateRaycastTarget()

	// 1. Sky gradient background with dynamic day/night celestial simulation & distance fog
	worldTime := int64(6000)
	if c.engine != nil {
		worldTime = c.engine.TickCount * 10
	}
	c.framebuf.ClearSkyWithTime(worldTime)

	// 2. Render cached 3D chunk meshes with perspective-correct texturing & distance fog
	c.meshMu.RLock()
	lx, ly, lz := c.camera.GetLookVector()
	camX, camY, camZ := c.camera.X, c.camera.Y, c.camera.Z

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

		for i := 0; i < len(mesh.Indices); i += 3 {
			v0 := mesh.Vertices[mesh.Indices[i]]
			v1 := mesh.Vertices[mesh.Indices[i+1]]
			v2 := mesh.Vertices[mesh.Indices[i+2]]

			// World-space exact normal culling for planar voxel face (zero arbitrary epsilon to prevent flickering)
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

			clipped, count := render.ClipTriangleNear(vv0, vv1, vv2, 0.02)
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
	c.meshMu.RUnlock()

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
	}

	// 3.5 Render 3D physical debris particles
	if len(c.particles) > 0 {
		c.framebuf.DrawParticles(c.camera, c.particles)
	}

	// 4. Draw first-person held item with walking sway and authentic arm swing
	c.drawHeldItem()

	// 5. Draw 2D HUD (Centered crosshair, 20 HP hearts, 20 hunger drumsticks, 9 hotbar slots)
	health := float32(20.0)
	hunger := 20
	selectedSlot := 0
	if c.player != nil {
		health = c.player.Health
		hunger = c.player.Hunger
		selectedSlot = c.player.Inventory.SelectedHotbar
	}
	c.framebuf.DrawHUD(health, hunger, selectedSlot)
}

func (c *Client) drawHeldItem() {
	if c.player == nil {
		return
	}
	selected := c.player.Inventory.GetSelectedItem()

	// Walking sway calculation
	swayX := int(math.Cos(c.walkCycle*0.5) * 6.0)
	swayY := int(math.Abs(math.Sin(c.walkCycle)) * 5.0)

	// Arm / Tool swing animation trajectory
	swingOffset := 0
	swingDrop := 0
	if c.isSwinging {
		arc := math.Sin(c.swingProgress * math.Pi)
		swingOffset = int(arc * 40.0)
		swingDrop = int(arc * 25.0)
	}

	hx := c.framebuf.Width - 95 + swayX - swingOffset
	hy := c.framebuf.Height - 110 + swayY + swingDrop

	if selected.IsEmpty() {
		// Steve's authentic first-person player arm / fist
		skinTone := uint32(0xFFD6A07E)   // Tan skin
		skinShade := uint32(0xFFB57C5E)  // Shading
		sleeveTone := uint32(0xFF2E8690) // Classic Cyan shirt sleeve

		// Cyan sleeve
		c.framebuf.DrawRect(hx+20, hy+40, 45, 60, sleeveTone)
		// Forearm
		c.framebuf.DrawRect(hx+25, hy+15, 35, 25, skinTone)
		// Fist
		c.framebuf.DrawRect(hx+20, hy, 40, 15, skinShade)
		return
	}

	// 3D isometric held block
	colorTop := render.SampleBlockColor(selected.ItemID, 2, 0.5, 0.5, 1.0)
	colorLeft := render.SampleBlockColor(selected.ItemID, 1, 0.5, 0.5, 0.8)
	colorRight := render.SampleBlockColor(selected.ItemID, 0, 0.5, 0.5, 0.6)

	c.framebuf.DrawRect(hx+15, hy, 40, 20, colorTop)
	c.framebuf.DrawRect(hx, hy+20, 35, 45, colorLeft)
	c.framebuf.DrawRect(hx+35, hy+20, 35, 45, colorRight)
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
		switch vkCode {
		case 0x46: // F: Mine
			c.MineTargetedBlock()
		case 0x52: // R: Place
			c.PlaceBlockInFront()
		case 0x45: // E: Craft
			if c.player != nil {
				c.player.Inventory.AddItem(inventory.ItemOakPlanks, 4)
			}
		case 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39: // 1-9
			slot := vkCode - 0x31
			if c.player != nil {
				c.player.Inventory.SelectedHotbar = slot
			}
		case 0x1B: // Escape: Toggle mouse lock
			wnd.SetMouseCaptured(!wnd.MouseCaptured)
		}
	}

	// Mouse button clicks (instant single-tap actions)
	wnd.OnMouseClick = func(btn int) {
		if btn == 1 {
			// Trigger arm swing on click
			if !c.isSwinging {
				c.isSwinging = true
				c.swingProgress = 0.0
			}
		} else if btn == 2 {
			// Right click: Place block immediately if cooldown ready
			if c.placeCooldown <= 0 {
				if c.PlaceBlockInFront() {
					c.placeCooldown = 0.25
					sel := c.player.Inventory.GetSelectedItem()
					sound.PlayHardwareFeedback(sound.GetBlockPlaceSound(sel.ItemID))
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

		// 1. Edge-triggered Spacebar Jump (prevents continuous hopping)
		spaceDown := wnd.IsKeyDown(0x20)
		jumpNow := false
		if spaceDown && !c.wasSpaceDown {
			jumpNow = true
		}
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

		c.PhysicsStep(fwdInput, strafeInput, jumpNow, dt)

		// 4. Authentic Minecraft Mining: Hold Left Mouse to mine with cracking & tool times
		isLeftDown := wnd.IsMouseDown(1)
		c.UpdateMining(isLeftDown, dt)

		// 5. Placing Blocks: Hold Right Mouse Button (with 0.25s authentic cooldown)
		if wnd.IsMouseDown(2) {
			if c.placeCooldown <= 0 {
				if c.PlaceBlockInFront() {
					c.placeCooldown = 0.25
					sel := c.player.Inventory.GetSelectedItem()
					sound.PlayHardwareFeedback(sound.GetBlockPlaceSound(sel.ItemID))
				}
			}
		}

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
