package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/world"
)

type Client struct {
	serverAddr string
	username   string
	uuid       [16]byte
	conn       net.Conn
	state      protocol.ConnectionState
	framebuf   *render.Framebuffer
	localWorld map[world.BlockPos]int32
}

func NewClient(serverAddr, username string) *Client {
	var u [16]byte
	rand.Read(u[:])
	return &Client{
		serverAddr: serverAddr,
		username:   username,
		uuid:       u,
		state:      protocol.StateHandshake,
		framebuf:   render.NewFramebuffer(640, 480),
		localWorld: make(map[world.BlockPos]int32),
	}
}

func (c *Client) Connect() error {
	var err error
	c.conn, err = net.DialTimeout("tcp", c.serverAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.serverAddr, err)
	}
	log.Printf("[Client] Connected to server at %s", c.serverAddr)

	// 1. Send Handshake packet (NextState = 2: Login)
	hs := &protocol.HandshakePacket{
		ProtocolVersion: 770,
		ServerAddress:   "localhost",
		ServerPort:      25565,
		NextState:       2,
	}
	if err := protocol.SendPacket(c.conn, hs); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// 2. Send LoginStart packet
	c.state = protocol.StateLogin
	ls := &protocol.LoginStartPacket{
		Username:   c.username,
		PlayerUUID: c.uuid,
	}
	if err := protocol.SendPacket(c.conn, ls); err != nil {
		return fmt.Errorf("failed to send login start: %w", err)
	}

	// 3. Read LoginSuccess
	raw, err := protocol.ReadRawPacket(c.conn)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}
	if raw.ID != 0x02 {
		return fmt.Errorf("unexpected packet id during login: 0x%02X", raw.ID)
	}

	var succ protocol.LoginSuccessPacket
	if err := succ.Decode(raw.Reader()); err != nil {
		return fmt.Errorf("failed to decode login success: %w", err)
	}
	c.state = protocol.StatePlay
	log.Printf("[Client] Logged in successfully as %s (UUID: %x)", succ.Username, succ.UUID)

	return nil
}

func (c *Client) SendBlockUpdate(x, y, z int32, blockID int32) error {
	pkt := &protocol.BlockUpdatePacket{
		X:       x,
		Y:       y,
		Z:       z,
		BlockID: blockID,
	}
	return protocol.SendPacket(c.conn, pkt)
}

func (c *Client) SendPosition(x, y, z float64, yaw, pitch float32, onGround bool) error {
	pkt := &protocol.PlayerPositionPacket{
		X:        x,
		Y:        y,
		Z:        z,
		Yaw:      yaw,
		Pitch:    pitch,
		OnGround: onGround,
	}
	return protocol.SendPacket(c.conn, pkt)
}

func (c *Client) RenderFrame() {
	c.framebuf.Clear(0xFF87CEEB) // Sky blue (0xEB, 0xCE, 0x87)

	// Build chunk mesh for visible blocks
	getBlock := func(x, y, z int32) int32 {
		return c.localWorld[world.BlockPos{X: x, Y: y, Z: z}]
	}

	mesh := render.BuildSectionMesh(getBlock, 0, 60, 0)

	// Rasterize quads into software framebuffer
	for i := 0; i < len(mesh.Indices); i += 3 {
		v0 := mesh.Vertices[mesh.Indices[i]]
		v1 := mesh.Vertices[mesh.Indices[i+1]]
		v2 := mesh.Vertices[mesh.Indices[i+2]]

		// Orthographic projection to test viewport
		p0 := render.ScreenPoint{X: 320 + v0.X*16 - v0.Z*8, Y: 240 - v0.Y*12, Depth: 0.5}
		p1 := render.ScreenPoint{X: 320 + v1.X*16 - v1.Z*8, Y: 240 - v1.Y*12, Depth: 0.5}
		p2 := render.ScreenPoint{X: 320 + v2.X*16 - v2.Z*8, Y: 240 - v2.Y*12, Depth: 0.5}

		c.framebuf.DrawTriangle(p0, p1, p2, 0xFF2E8B57) // Sea green
	}
}

func main() {
	serverAddr := flag.String("server", "127.0.0.1:25565", "Server address")
	username := flag.String("username", "Steve", "Player username")
	benchmark := flag.Bool("benchmark", false, "Run client rendering benchmark")
	flag.Parse()

	client := NewClient(*serverAddr, *username)

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

	if err := client.Connect(); err != nil {
		log.Fatalf("[Client] Connection error: %v", err)
	}

	log.Println("[Client] Client running. Press Ctrl+C to exit.")
	for {
		time.Sleep(1 * time.Second)
	}
}
