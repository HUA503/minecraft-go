package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

type Server struct {
	addr           string
	worldDir       string
	simEngine      *sim.Engine
	saveManager    *world.SaveManager
	listener       net.Listener
	mu             sync.Mutex
	sessions       map[int32]*ClientSession
	nextSessID     int32
	modifiedBlocks map[world.BlockPos]int32
	stopDone       chan struct{}
	running        bool
}

type ClientSession struct {
	ID       int32
	conn     net.Conn
	server   *Server
	state    protocol.ConnectionState
	username string
	uuid     [16]byte
	playerID int32
	closed   bool
	mu       sync.Mutex
}

func NewServer(addr, worldDir string) *Server {
	s := &Server{
		addr:           addr,
		worldDir:       worldDir,
		simEngine:      sim.NewEngine(),
		sessions:       make(map[int32]*ClientSession),
		modifiedBlocks: make(map[world.BlockPos]int32),
		stopDone:       make(chan struct{}),
		running:        true,
	}

	if worldDir != "" {
		sm, err := world.NewSaveManager(worldDir)
		if err == nil {
			s.saveManager = sm
			s.simEngine.SetStorage(sm)
		} else {
			log.Printf("[Server] Warning: failed to init SaveManager: %v", err)
		}
	}

	// Try loading spawn chunk from save manager first
	spawnPos := world.ChunkPos{X: 0, Z: 0}
	var spawnChunk *world.Chunk
	if s.saveManager != nil {
		if loaded, err := s.saveManager.LoadChunk(spawnPos); err == nil && loaded != nil {
			spawnChunk = loaded
			for pos, b := range spawnChunk.ModifiedBlocks {
				s.modifiedBlocks[pos] = b
			}
			log.Printf("[Server] Loaded spawn chunk from persistent storage (%d modified blocks)", len(s.modifiedBlocks))
		}
	}

	if spawnChunk == nil {
		// Initialize fresh spawn chunk at (0, 0)
		spawnChunk = world.NewChunk(spawnPos)
		for x := int32(0); x < 16; x++ {
			for z := int32(0); z < 16; z++ {
				spawnChunk.SetBlock(x, -64, z, 1)
				for y := int32(-63); y <= 60; y++ {
					spawnChunk.SetBlock(x, y, z, 1)
				}
				spawnChunk.SetBlock(x, 61, z, 3)
				spawnChunk.SetBlock(x, 62, z, 3)
				spawnChunk.SetBlock(x, 63, z, 3)
				spawnChunk.SetBlock(x, 64, z, 2)
			}
		}
		spawnChunk.ModifiedBlocks = make(map[world.BlockPos]int32)
		spawnChunk.IsDirty = false
	}
	s.simEngine.LoadChunk(spawnPos, spawnChunk)

	return s
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	log.Printf("[Server] Minecraft-Go listening on %s (20 TPS Authoritative Loop)", s.addr)

	s.simEngine.StartTickLoop()
	go s.broadcastLoop()

	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.running {
				break
			}
			log.Printf("[Server] Accept error: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
	<-s.stopDone
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}
	sessions := make([]*ClientSession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.Unlock()

	for _, sess := range sessions {
		sess.Close()
	}

	s.simEngine.Stop()
	log.Printf("[Server] Flushing world chunks to storage...")
	if err := s.simEngine.SaveAll(); err != nil {
		log.Printf("[Server] Error in SaveAll: %v", err)
	}
	if s.saveManager != nil {
		if err := s.saveManager.Close(); err != nil {
			log.Printf("[Server] Error closing SaveManager: %v", err)
		}
	}
	log.Printf("[Server] Server stopped successfully.")

	s.mu.Lock()
	select {
	case <-s.stopDone:
	default:
		close(s.stopDone)
	}
	s.mu.Unlock()
}

func (s *Server) broadcastLoop() {
	for ev := range s.simEngine.Events {
		s.mu.Lock()
		sessions := make([]*ClientSession, 0, len(s.sessions))
		for _, sess := range s.sessions {
			if sess.state == protocol.StatePlay {
				sessions = append(sessions, sess)
			}
		}
		s.mu.Unlock()

		switch ev.Type {
		case "block_update":
			pkt := &protocol.BlockUpdatePacket{
				X:       ev.BlockPos.X,
				Y:       ev.BlockPos.Y,
				Z:       ev.BlockPos.Z,
				BlockID: ev.BlockID,
			}
			for _, sess := range sessions {
				sess.Send(pkt)
			}
		case "player_move":
			pkt := &protocol.PlayerPositionPacket{
				X:        ev.Position.X,
				Y:        ev.Position.Y,
				Z:        ev.Position.Z,
				Yaw:      ev.Yaw,
				Pitch:    ev.Pitch,
				OnGround: true,
			}
			for _, sess := range sessions {
				if sess.playerID != ev.PlayerID { // Broadcast to other players
					sess.Send(pkt)
				}
			}
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()
	s.nextSessID++
	sess := &ClientSession{
		ID:     s.nextSessID,
		conn:   conn,
		server: s,
		state:  protocol.StateHandshake,
	}
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	defer func() {
		sess.Close()
		s.mu.Lock()
		delete(s.sessions, sess.ID)
		s.mu.Unlock()
	}()

	sess.readLoop()
}

func (sess *ClientSession) Send(pkt protocol.Packet) error {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if sess.closed {
		return net.ErrClosed
	}
	return protocol.SendPacket(sess.conn, pkt)
}

func (sess *ClientSession) Close() {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if !sess.closed {
		sess.closed = true
		sess.conn.Close()
		if sess.playerID > 0 {
			sess.server.simEngine.RemovePlayer(sess.playerID)
		}
	}
}

func (sess *ClientSession) readLoop() {
	for {
		raw, err := protocol.ReadRawPacket(sess.conn)
		if err != nil {
			return
		}

		if err := sess.handlePacket(raw); err != nil {
			log.Printf("[Session %d] Error handling packet (state %v, id 0x%02X): %v",
				sess.ID, sess.state, raw.ID, err)
			return
		}
	}
}

func (sess *ClientSession) handlePacket(raw *protocol.RawPacket) error {
	switch sess.state {
	case protocol.StateHandshake:
		if raw.ID == 0x00 {
			var hs protocol.HandshakePacket
			if err := hs.Decode(raw.Reader()); err != nil {
				return err
			}
			if hs.NextState == 1 {
				sess.state = protocol.StateStatus
			} else if hs.NextState == 2 {
				sess.state = protocol.StateLogin
			}
		}

	case protocol.StateStatus:
		if raw.ID == 0x00 { // StatusRequest
			sess.server.mu.Lock()
			onlineCount := 0
			for _, s := range sess.server.sessions {
				if s.state == protocol.StatePlay {
					onlineCount++
				}
			}
			sess.server.mu.Unlock()

			resp := map[string]interface{}{
				"version": map[string]interface{}{
					"name":     "Minecraft 26.3 (Go Horizon Parity)",
					"protocol": protocol.ProtocolVersion,
				},
				"players": map[string]interface{}{
					"max":    50,
					"online": onlineCount,
				},
				"description": map[string]interface{}{
					"text": "Minecraft-Go Authoritative Server",
				},
			}
			data, _ := json.Marshal(resp)
			sess.Send(&protocol.StatusResponsePacket{JSONResponse: string(data)})
		} else if raw.ID == 0x01 { // Ping
			var ping protocol.PingPacket
			if err := ping.Decode(raw.Reader()); err != nil {
				return err
			}
			sess.Send(&protocol.PingPacket{Payload: ping.Payload})
		}

	case protocol.StateLogin:
		if raw.ID == 0x00 { // LoginStart
			var ls protocol.LoginStartPacket
			if err := ls.Decode(raw.Reader()); err != nil {
				return err
			}
			sess.username = ls.Username
			sess.uuid = ls.PlayerUUID

			// Send LoginSuccess
			if err := sess.Send(&protocol.LoginSuccessPacket{
				UUID:     sess.uuid,
				Username: sess.username,
			}); err != nil {
				return err
			}

			// Add player to simulation engine at spawn (X=8, Y=65, Z=8)
			player := sess.server.simEngine.AddPlayer(sess.username, sess.uuid, sim.Vec3{X: 8, Y: 65, Z: 8})
			sess.playerID = player.ID
			sess.state = protocol.StatePlay

			// Send existing modified blocks to the new player
			sess.server.mu.Lock()
			for bpos, bid := range sess.server.modifiedBlocks {
				_ = sess.Send(&protocol.BlockUpdatePacket{
					X:       bpos.X,
					Y:       bpos.Y,
					Z:       bpos.Z,
					BlockID: bid,
				})
			}

			// Send existing players to the new player
			for _, other := range sess.server.sessions {
				if other.ID != sess.ID && other.state == protocol.StatePlay {
					if otherP, ok := sess.server.simEngine.Players[other.playerID]; ok {
						_ = sess.Send(&protocol.PlayerPositionPacket{
							X:        otherP.Position.X,
							Y:        otherP.Position.Y,
							Z:        otherP.Position.Z,
							Yaw:      otherP.Yaw,
							Pitch:    otherP.Pitch,
							OnGround: true,
						})
					}
				}
			}
			sess.server.mu.Unlock()

			log.Printf("[Server] Player joined: %s (UUID: %x, EntityID: %d)", sess.username, sess.uuid, sess.playerID)
		}

	case protocol.StatePlay:
		switch raw.ID {
		case 0x1A: // PlayerPosition
			var pos protocol.PlayerPositionPacket
			if err := pos.Decode(raw.Reader()); err != nil {
				return err
			}
			sess.server.simEngine.MovePlayer(sess.playerID, sim.Vec3{X: pos.X, Y: pos.Y, Z: pos.Z}, pos.Yaw, pos.Pitch, pos.OnGround)

		case 0x09: // BlockUpdate (Place/Break)
			var bu protocol.BlockUpdatePacket
			if err := bu.Decode(raw.Reader()); err != nil {
				return err
			}
			if bu.Y >= world.MinWorldY && bu.Y <= world.MaxWorldY {
				sess.server.simEngine.SetBlock(bu.X, bu.Y, bu.Z, bu.BlockID)
				sess.server.mu.Lock()
				sess.server.modifiedBlocks[world.BlockPos{X: bu.X, Y: bu.Y, Z: bu.Z}] = bu.BlockID
				sess.server.mu.Unlock()
			}

		case 0x26: // KeepAlive
			var ka protocol.KeepAlivePacket
			if err := ka.Decode(raw.Reader()); err != nil {
				return err
			}
			sess.Send(&ka)
		}
	}

	return nil
}

func main() {
	port := flag.Int("port", 25565, "Server listening port")
	worldDir := flag.String("world", "world", "World storage directory")
	flag.Parse()

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	srv := NewServer(addr, *worldDir)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Server] Shutting down...")
		srv.Stop()
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmd := strings.TrimSpace(scanner.Text())
			if cmd == "stop" || cmd == "exit" || cmd == "quit" {
				log.Println("[Server] Stop command received, shutting down...")
				srv.Stop()
				return
			}
		}
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("[Server] Fatal error: %v", err)
	}
}
