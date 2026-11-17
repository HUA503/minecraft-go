package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

type Server struct {
	addr       string
	simEngine  *sim.Engine
	listener   net.Listener
	mu         sync.Mutex
	sessions   map[int32]*ClientSession
	nextSessID int32
	running    bool
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

func NewServer(addr string) *Server {
	s := &Server{
		addr:      addr,
		simEngine: sim.NewEngine(),
		sessions:  make(map[int32]*ClientSession),
		running:   true,
	}

	// Initialize spawn chunk at (0, 0)
	spawnChunk := world.NewChunk(world.ChunkPos{X: 0, Z: 0})
	for x := int32(0); x < 16; x++ {
		for z := int32(0); z < 16; z++ {
			// Bedrock layer at -64
			spawnChunk.SetBlock(x, -64, z, 1)
			// Stone layer from -63 to 60
			for y := int32(-63); y <= 60; y++ {
				spawnChunk.SetBlock(x, y, z, 1)
			}
			// Dirt from 61 to 63
			spawnChunk.SetBlock(x, 61, z, 3)
			spawnChunk.SetBlock(x, 62, z, 3)
			spawnChunk.SetBlock(x, 63, z, 3)
			// Grass at 64
			spawnChunk.SetBlock(x, 64, z, 2)
		}
	}
	s.simEngine.LoadChunk(world.ChunkPos{X: 0, Z: 0}, spawnChunk)

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
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}
	for _, sess := range s.sessions {
		sess.Close()
	}
	s.mu.Unlock()
	s.simEngine.Stop()
	log.Printf("[Server] Server stopped successfully.")
}

func (s *Server) broadcastLoop() {
	for ev := range s.simEngine.Events {
		s.mu.Lock()
		sessions := make([]*ClientSession, 0, len(s.sessions))
		for _, sess := range s.sessions {
			sessions = append(sessions, sess)
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
			resp := map[string]interface{}{
				"version": map[string]interface{}{
					"name":     "Minecraft 26.3 (Go Horizon Parity)",
					"protocol": 770,
				},
				"players": map[string]interface{}{
					"max":    50,
					"online": len(sess.server.sessions),
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
			sess.server.simEngine.SetBlock(bu.X, bu.Y, bu.Z, bu.BlockID)

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
	flag.Parse()

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	srv := NewServer(addr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Server] Shutting down...")
		srv.Stop()
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("[Server] Fatal error: %v", err)
	}
}
