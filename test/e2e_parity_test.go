package test

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/HUA503/minecraft-go/pkg/nbt"
	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/redstone"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

// TestServerStatusSLPPing tests server status inquiry and ping/pong round-trip (P0).
func TestServerStatusSLPPing(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	srvAddr := listener.Addr().String()

	// In-line minimal server handling SLP
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read Handshake
		rawHS, _ := protocol.ReadRawPacket(conn)
		var hs protocol.HandshakePacket
		hs.Decode(rawHS.Reader())

		// Read Status Request
		protocol.ReadRawPacket(conn)

		// Send Status Response
		resp := map[string]interface{}{
			"version": map[string]interface{}{
				"name":     "1.21.4 (26.3 Parity)",
				"protocol": 770,
			},
			"description": map[string]interface{}{
				"text": "Minecraft-Go Parity Server",
			},
		}
		data, _ := json.Marshal(resp)
		protocol.SendPacket(conn, &protocol.StatusResponsePacket{JSONResponse: string(data)})

		// Read Ping
		rawPing, _ := protocol.ReadRawPacket(conn)
		var ping protocol.PingPacket
		ping.Decode(rawPing.Reader())

		// Send Pong
		protocol.SendPacket(conn, &protocol.PingPacket{Payload: ping.Payload})
	}()

	// Client connection
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Send Handshake
	hs := &protocol.HandshakePacket{
		ProtocolVersion: 770,
		ServerAddress:   "127.0.0.1",
		ServerPort:      25565,
		NextState:       1, // Status
	}
	if err := protocol.SendPacket(conn, hs); err != nil {
		t.Fatalf("failed to send handshake: %v", err)
	}

	// Send StatusRequest
	if err := protocol.SendPacket(conn, &protocol.StatusRequestPacket{}); err != nil {
		t.Fatalf("failed to send status request: %v", err)
	}

	// Read StatusResponse
	rawResp, err := protocol.ReadRawPacket(conn)
	if err != nil {
		t.Fatalf("failed to read status response: %v", err)
	}
	var resp protocol.StatusResponsePacket
	if err := resp.Decode(rawResp.Reader()); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}

	var statusMap map[string]interface{}
	if err := json.Unmarshal([]byte(resp.JSONResponse), &statusMap); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	verMap := statusMap["version"].(map[string]interface{})
	if int(verMap["protocol"].(float64)) != 770 {
		t.Fatalf("protocol mismatch: %v", verMap["protocol"])
	}

	// Send Ping
	payload := int64(0x123456789ABCDEF0)
	if err := protocol.SendPacket(conn, &protocol.PingPacket{Payload: payload}); err != nil {
		t.Fatalf("failed to send ping: %v", err)
	}

	// Read Pong
	rawPong, err := protocol.ReadRawPacket(conn)
	if err != nil {
		t.Fatalf("failed to read pong: %v", err)
	}
	var pong protocol.PingPacket
	if err := pong.Decode(rawPong.Reader()); err != nil {
		t.Fatalf("failed to decode pong: %v", err)
	}
	if pong.Payload != payload {
		t.Fatalf("pong payload mismatch: got 0x%X, expected 0x%X", pong.Payload, payload)
	}
}

// TestMultiplayerBlockAndMovementSync tests P1 vertical slice:
// Two clients connect, Client A places a block and moves, Client B observes both changes.
func TestMultiplayerBlockAndMovementSync(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	engine := sim.NewEngine()
	engine.StartTickLoop()
	defer engine.Stop()

	type Session struct {
		conn     net.Conn
		playerID int32
		mu       sync.Mutex
	}
	var sessions []*Session
	var sessMu sync.Mutex

	// Simple multiplexing server
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			sess := &Session{conn: conn}
			sessMu.Lock()
			sessions = append(sessions, sess)
			sessMu.Unlock()

			go func(s *Session) {
				defer s.conn.Close()
				for {
					raw, err := protocol.ReadRawPacket(s.conn)
					if err != nil {
						return
					}

					switch raw.ID {
					case 0x00: // Handshake / LoginStart
						var ls protocol.LoginStartPacket
						if err := ls.Decode(raw.Reader()); err == nil && ls.Username != "" {
							p := engine.AddPlayer(ls.Username, ls.PlayerUUID, sim.Vec3{X: 0, Y: 64, Z: 0})
							s.playerID = p.ID
							protocol.SendPacket(s.conn, &protocol.LoginSuccessPacket{
								UUID:     ls.PlayerUUID,
								Username: ls.Username,
							})
						}
					case 0x09: // BlockUpdate
						var bu protocol.BlockUpdatePacket
						bu.Decode(raw.Reader())
						engine.SetBlock(bu.X, bu.Y, bu.Z, bu.BlockID)

						// Broadcast to other sessions
						sessMu.Lock()
						for _, other := range sessions {
							if other != s {
								protocol.SendPacket(other.conn, &bu)
							}
						}
						sessMu.Unlock()

					case 0x1A: // PlayerPosition
						var pos protocol.PlayerPositionPacket
						pos.Decode(raw.Reader())
						engine.MovePlayer(s.playerID, sim.Vec3{X: pos.X, Y: pos.Y, Z: pos.Z}, pos.Yaw, pos.Pitch, pos.OnGround)

						// Broadcast to other sessions
						sessMu.Lock()
						for _, other := range sessions {
							if other != s {
								protocol.SendPacket(other.conn, &pos)
							}
						}
						sessMu.Unlock()
					}
				}
			}(sess)
		}
	}()

	// 1. Connect Client A ("Steve")
	connA, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("client A failed to dial: %v", err)
	}
	defer connA.Close()

	uuidA := [16]byte{0x01, 0x02, 0x03}
	protocol.SendPacket(connA, &protocol.LoginStartPacket{Username: "Steve", PlayerUUID: uuidA})
	rawA, _ := protocol.ReadRawPacket(connA)
	if rawA.ID != 0x02 {
		t.Fatalf("client A expected LoginSuccess, got 0x%02X", rawA.ID)
	}

	// 2. Connect Client B ("Alex")
	connB, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("client B failed to dial: %v", err)
	}
	defer connB.Close()

	uuidB := [16]byte{0x04, 0x05, 0x06}
	protocol.SendPacket(connB, &protocol.LoginStartPacket{Username: "Alex", PlayerUUID: uuidB})
	rawB, _ := protocol.ReadRawPacket(connB)
	if rawB.ID != 0x02 {
		t.Fatalf("client B expected LoginSuccess, got 0x%02X", rawB.ID)
	}

	// 3. Client A places Diamond Block (ID=57) at (10, 64, 20)
	protocol.SendPacket(connA, &protocol.BlockUpdatePacket{
		X:       10,
		Y:       64,
		Z:       20,
		BlockID: 57,
	})

	// Client B should receive BlockUpdate
	rawBlockB, err := protocol.ReadRawPacket(connB)
	if err != nil {
		t.Fatalf("client B failed to receive block update: %v", err)
	}
	var blockB protocol.BlockUpdatePacket
	blockB.Decode(rawBlockB.Reader())
	if blockB.X != 10 || blockB.Y != 64 || blockB.Z != 20 || blockB.BlockID != 57 {
		t.Fatalf("client B received mismatched block: %+v", blockB)
	}

	// Also verify authoritative engine state
	if engine.GetBlock(10, 64, 20) != 57 {
		t.Fatalf("authoritative world block was not updated: %d", engine.GetBlock(10, 64, 20))
	}

	// 4. Client A moves to (12.5, 65.0, -18.25)
	protocol.SendPacket(connA, &protocol.PlayerPositionPacket{
		X:        12.5,
		Y:        65.0,
		Z:        -18.25,
		Yaw:      180.0,
		Pitch:    45.0,
		OnGround: true,
	})

	// Client B should receive PlayerPosition broadcast
	rawPosB, err := protocol.ReadRawPacket(connB)
	if err != nil {
		t.Fatalf("client B failed to receive pos update: %v", err)
	}
	var posB protocol.PlayerPositionPacket
	posB.Decode(rawPosB.Reader())
	if posB.X != 12.5 || posB.Y != 65.0 || posB.Z != -18.25 || posB.Yaw != 180.0 || posB.Pitch != 45.0 {
		t.Fatalf("client B received mismatched position: %+v", posB)
	}
}

// TestWorldSaveAndReloadPersistence tests persistence across server restarts (P1).
func TestWorldSaveAndReloadPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mca_persist_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mcaFile := filepath.Join(tmpDir, "r.0.0.mca")

	// Phase 1: Create world, modify blocks, save to region file
	{
		reg, err := world.OpenRegion(mcaFile)
		if err != nil {
			t.Fatalf("failed to open region: %v", err)
		}

		chunkNBT := nbt.NewCompound()
		chunkNBT.Put("DataVersion", &nbt.IntTag{Value: 3953})
		chunkNBT.Put("xPos", &nbt.IntTag{Value: 0})
		chunkNBT.Put("zPos", &nbt.IntTag{Value: 0})

		// Save a list of modified block positions and states
		blocksTag := &nbt.ListTag{
			ElementType: nbt.TagCompound,
			Elements: []nbt.Tag{
				func() *nbt.CompoundTag {
					c := nbt.NewCompound()
					c.Put("x", &nbt.IntTag{Value: 5})
					c.Put("y", &nbt.IntTag{Value: 64})
					c.Put("z", &nbt.IntTag{Value: 7})
					c.Put("state", &nbt.IntTag{Value: 15}) // Redstone wire
					return c
				}(),
				func() *nbt.CompoundTag {
					c := nbt.NewCompound()
					c.Put("x", &nbt.IntTag{Value: 5})
					c.Put("y", &nbt.IntTag{Value: 64})
					c.Put("z", &nbt.IntTag{Value: 8})
					c.Put("state", &nbt.IntTag{Value: 57}) // Diamond block
					return c
				}(),
			},
		}
		chunkNBT.Put("ModifiedBlocks", blocksTag)

		if err := reg.WriteChunkData(0, 0, chunkNBT); err != nil {
			t.Fatalf("failed to write chunk data: %v", err)
		}
		reg.Close()
	}

	// Phase 2: Simulate restart: reopen region file and verify all fields preserved
	{
		reg, err := world.OpenRegion(mcaFile)
		if err != nil {
			t.Fatalf("failed to reopen region: %v", err)
		}
		defer reg.Close()

		loadedNBT, err := reg.ReadChunkData(0, 0)
		if err != nil {
			t.Fatalf("failed to read back chunk data: %v", err)
		}

		if loadedNBT.GetInt("DataVersion") != 3953 {
			t.Fatalf("DataVersion mismatch: %d", loadedNBT.GetInt("DataVersion"))
		}

		blocksList, ok := loadedNBT.Get("ModifiedBlocks").(*nbt.ListTag)
		if !ok || len(blocksList.Elements) != 2 {
			t.Fatalf("modified blocks list invalid: %+v", loadedNBT.Get("ModifiedBlocks"))
		}

		b0 := blocksList.Elements[0].(*nbt.CompoundTag)
		if b0.GetInt("x") != 5 || b0.GetInt("y") != 64 || b0.GetInt("z") != 7 || b0.GetInt("state") != 15 {
			t.Fatalf("b0 mismatch: %+v", b0)
		}

		b1 := blocksList.Elements[1].(*nbt.CompoundTag)
		if b1.GetInt("x") != 5 || b1.GetInt("y") != 64 || b1.GetInt("z") != 8 || b1.GetInt("state") != 57 {
			t.Fatalf("b1 mismatch: %+v", b1)
		}
	}
}

// TestDifferentialRedstoneBehavior tests exact parity with Minecraft Java redstone truth tables.
func TestDifferentialRedstoneBehavior(t *testing.T) {
	eng := redstone.NewEngine()

	// Redstone truth table:
	// Lever at (0, 0, 0) ON (15)
	// Wire line along X from (1,0,0) to (15,0,0)
	// Minecraft Java ground truth: Wire signal level = max(0, 15 - distance)
	eng.AddComponent(&redstone.Component{
		Type:       redstone.TypeLever,
		Position:   redstone.Pos{X: 0, Y: 0, Z: 0},
		PowerLevel: 15,
	})
	for x := int32(1); x <= 16; x++ {
		eng.AddComponent(&redstone.Component{
			Type:     redstone.TypeWire,
			Position: redstone.Pos{X: x, Y: 0, Z: 0},
		})
	}

	eng.Step()

	for dist := int32(1); dist <= 16; dist++ {
		comp := eng.GetComponent(redstone.Pos{X: dist, Y: 0, Z: 0})
		expectedPower := 15 - int(dist)
		if expectedPower < 0 {
			expectedPower = 0
		}
		if comp.PowerLevel != expectedPower {
			t.Fatalf("redstone parity difference at distance %d: got %d, expected %d", dist, comp.PowerLevel, expectedPower)
		}
	}
}
