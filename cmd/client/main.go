package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

type Client struct {
	serverAddr  string
	username    string
	uuid        [16]byte
	conn        net.Conn
	state       protocol.ConnectionState
	framebuf    *render.Framebuffer
	camera      *render.Camera
	engine      *sim.Engine
	player      *sim.PlayerEntity
	craftingMgr *inventory.CraftingManager
	isLocal     bool
}

func NewClient(serverAddr, username string, isSingleplayer bool) *Client {
	var u [16]byte
	rand.Read(u[:])

	cli := &Client{
		serverAddr:  serverAddr,
		username:    username,
		uuid:        u,
		state:       protocol.StateHandshake,
		framebuf:    render.NewFramebuffer(640, 480),
		camera:      render.NewCamera(8.0, 65.62, 8.0),
		craftingMgr: inventory.NewCraftingManager(),
		isLocal:     isSingleplayer,
	}

	if isSingleplayer {
		// Initialize local simulation engine and procedural world
		cli.engine = sim.NewEngine()
		genOverworld := gen.NewGenerator(12345, gen.DimensionOverworld)

		// Pre-generate 3x3 chunks around spawn
		for cx := int32(-1); cx <= 1; cx++ {
			for cz := int32(-1); cz <= 1; cz++ {
				chunk := genOverworld.GenerateChunk(world.ChunkPos{X: cx, Z: cz})
				cli.engine.LoadChunk(world.ChunkPos{X: cx, Z: cz}, chunk)
			}
		}

		cli.engine.StartTickLoop()
		cli.player = cli.engine.AddPlayer(username, u, sim.Vec3{X: 8.0, Y: 65.0, Z: 8.0})

		// Give starter items in hotbar
		cli.player.Inventory.AddItem(inventory.ItemOakLog, 16)
		cli.player.Inventory.AddItem(inventory.ItemCobblestone, 32)
	}

	return cli
}

func (c *Client) Connect() error {
	var err error
	c.conn, err = net.DialTimeout("tcp", c.serverAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.serverAddr, err)
	}
	log.Printf("[Client] Connected to server at %s", c.serverAddr)

	// 1. Handshake
	hs := &protocol.HandshakePacket{
		ProtocolVersion: 770,
		ServerAddress:   "localhost",
		ServerPort:      25565,
		NextState:       2, // Login
	}
	if err := protocol.SendPacket(c.conn, hs); err != nil {
		return err
	}

	// 2. LoginStart
	c.state = protocol.StateLogin
	ls := &protocol.LoginStartPacket{
		Username:   c.username,
		PlayerUUID: c.uuid,
	}
	if err := protocol.SendPacket(c.conn, ls); err != nil {
		return err
	}

	// 3. LoginSuccess
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

// Move updates player position and camera.
func (c *Client) Move(dx, dz float64) {
	if c.player == nil {
		return
	}
	newPos := c.player.Position.Add(sim.Vec3{X: dx, Y: 0, Z: dz})
	c.engine.MovePlayer(c.player.ID, newPos, c.player.Yaw, c.player.Pitch, c.player.OnGround)
	c.camera.X = newPos.X
	c.camera.Y = newPos.Y + 1.62 // Eye height
	c.camera.Z = newPos.Z
}

// MineTargetedBlock casts ray from eyes and mines the targeted voxel.
func (c *Client) MineTargetedBlock() (int32, bool) {
	if c.player == nil {
		return 0, false
	}
	origin := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	dir := sim.Vec3{X: lx, Y: ly, Z: lz}

	isSolid := func(x, y, z int32) bool {
		return c.engine.GetBlock(x, y, z) != 0
	}

	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if !res.Hit {
		return 0, false
	}

	drop := c.engine.BreakBlock(c.player.ID, world.BlockPos{X: res.BlockX, Y: res.BlockY, Z: res.BlockZ})
	return drop, true
}

// PlaceBlockInFront places the active hotbar block against the targeted face.
func (c *Client) PlaceBlockInFront() bool {
	if c.player == nil {
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

	return c.engine.PlaceBlock(c.player.ID, targetPos, selected.ItemID)
}

// RenderFrame renders first-person 3D world and 2D HUD.
func (c *Client) RenderFrame() {
	c.framebuf.Clear(0xFF87CEEB) // Sky blue

	if c.engine != nil {
		// Visible chunks mesh
		getBlock := func(x, y, z int32) int32 {
			return c.engine.GetBlock(x, y, z)
		}

		mesh := render.BuildSectionMesh(getBlock, 0, 60, 0)
		for i := 0; i < len(mesh.Indices); i += 3 {
			v0 := mesh.Vertices[mesh.Indices[i]]
			v1 := mesh.Vertices[mesh.Indices[i+1]]
			v2 := mesh.Vertices[mesh.Indices[i+2]]

			p0, ok0 := c.camera.ProjectToScreen(float64(v0.X), float64(v0.Y), float64(v0.Z), c.framebuf.Width, c.framebuf.Height)
			p1, ok1 := c.camera.ProjectToScreen(float64(v1.X), float64(v1.Y), float64(v1.Z), c.framebuf.Width, c.framebuf.Height)
			p2, ok2 := c.camera.ProjectToScreen(float64(v2.X), float64(v2.Y), float64(v2.Z), c.framebuf.Width, c.framebuf.Height)

			if ok0 && ok1 && ok2 {
				c.framebuf.DrawTriangle(p0, p1, p2, 0xFF2E8B57) // Grass green
			}
		}
	}

	// Draw HUD (Crosshair, 20 HP, 20 Hunger, Selected Hotbar slot)
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

func main() {
	singleplayer := flag.Bool("singleplayer", true, "Run singleplayer embedded server with procedural world")
	serverAddr := flag.String("server", "", "Multiplayer server address (e.g. 127.0.0.1:25565)")
	username := flag.String("username", "Steve", "Player username")
	benchmark := flag.Bool("benchmark", false, "Run client rendering benchmark")
	interactive := flag.Bool("interactive", false, "Run interactive survival game CLI")
	flag.Parse()

	isSP := *singleplayer
	if *serverAddr != "" {
		isSP = false
	}

	client := NewClient(*serverAddr, *username, isSP)

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

	if *interactive || isSP {
		runInteractiveLoop(client)
	} else {
		for {
			time.Sleep(1 * time.Second)
		}
	}
}

func runInteractiveLoop(c *Client) {
	fmt.Println("\nCommands:")
	fmt.Println("  w / s / a / d   : Move player")
	fmt.Println("  look <yaw> <pitch> : Rotate camera view")
	fmt.Println("  mine            : Mine targeted block in crosshair")
	fmt.Println("  place           : Place active hotbar block")
	fmt.Println("  slot <1-9>      : Switch active hotbar slot")
	fmt.Println("  inv             : View inventory")
	fmt.Println("  craft <recipe>  : Craft items (e.g. 'craft planks', 'craft table', 'craft pickaxe')")
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
			c.Move(1.0, 0)
			fmt.Printf("Moved forward. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "s":
			c.Move(-1.0, 0)
			fmt.Printf("Moved backward. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "a":
			c.Move(0, -1.0)
			fmt.Printf("Strafed left. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "d":
			c.Move(0, 1.0)
			fmt.Printf("Strafed right. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
		case "look":
			if len(parts) >= 3 {
				yaw, _ := strconv.ParseFloat(parts[1], 64)
				pitch, _ := strconv.ParseFloat(parts[2], 64)
				c.camera.Yaw = yaw
				c.camera.Pitch = pitch
				fmt.Printf("Camera rotated: Yaw=%.1f, Pitch=%.1f\n", yaw, pitch)
			} else {
				fmt.Println("Usage: look <yaw> <pitch>")
			}
		case "mine":
			drop, ok := c.MineTargetedBlock()
			if ok {
				fmt.Printf("Mined block! Gained item ID: %d in inventory.\n", drop)
			} else {
				fmt.Println("No block in reach (within 5 blocks).")
			}
		case "place":
			ok := c.PlaceBlockInFront()
			if ok {
				fmt.Println("Placed block successfully!")
			} else {
				fmt.Println("Could not place block (no target block in reach or intersecting player).")
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
					// Try craft 1 log -> 4 planks
					if err := c.player.Inventory.RemoveItem(0, 1); err == nil {
						c.player.Inventory.AddItem(inventory.ItemOakPlanks, 4)
						fmt.Println("Crafted 4 Oak Planks!")
					} else {
						fmt.Println("Not enough Oak Logs in slot 1.")
					}
				case "table":
					if err := c.player.Inventory.RemoveItem(0, 4); err == nil {
						c.player.Inventory.AddItem(inventory.ItemCraftingTable, 1)
						fmt.Println("Crafted 1 Crafting Table!")
					} else {
						fmt.Println("Not enough Planks.")
					}
				case "pickaxe":
					c.player.Inventory.AddItem(inventory.ItemWoodenPickaxe, 1)
					fmt.Println("Crafted 1 Wooden Pickaxe!")
				}
			} else {
				fmt.Println("Usage: craft <planks|table|pickaxe>")
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
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for options.\n", cmd)
		}
	}
}
