package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
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
	sessionWG      sync.WaitGroup
	writeWG        sync.WaitGroup
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
	outbound chan protocol.Packet
	done     chan struct{}
	doneOnce sync.Once
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
			if err := s.simEngine.SetStorage(sm); err != nil {
				log.Printf("[Server] Warning: failed to init dimension storage: %v", err)
				_ = sm.Close()
			} else {
				s.saveManager = sm
			}
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
	go func() {
		<-s.simEngine.TickLoopDone()
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()
		if running {
			log.Printf("[Server] Simulation tick loop exited unexpectedly; stopping server and saving world")
			s.Stop()
		}
	}()

	for {
		s.mu.Lock()
		running := s.running
		s.mu.Unlock()
		if !running {
			break
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if !s.running {
				break
			}
			log.Printf("[Server] Accept error: %v", err)
			continue
		}
		s.mu.Lock()
		if !s.running {
			s.mu.Unlock()
			_ = conn.Close()
			break
		}
		s.sessionWG.Add(1)
		s.mu.Unlock()
		go func(conn net.Conn) {
			defer s.sessionWG.Done()
			s.handleConnection(conn)
		}(conn)
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
	s.sessionWG.Wait()
	s.writeWG.Wait()

	s.simEngine.Stop()
	log.Printf("[Server] Flushing world chunks to storage...")
	if err := s.simEngine.SaveAll(); err != nil {
		log.Printf("[Server] Error in SaveAll: %v", err)
	}
	if err := s.simEngine.CloseStorage(); err != nil {
		log.Printf("[Server] Error closing world storage: %v", err)
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

func (sess *ClientSession) canModifyBlock(bu protocol.BlockUpdatePacket) bool {
	if bu.BlockID != 0 && !inventory.IsPlaceableBlock(bu.BlockID) {
		return false
	}
	_, playerID, _, _ := sess.snapshot()
	for _, player := range sess.server.simEngine.GetPlayerList() {
		if player.ID != playerID {
			continue
		}
		dx := float64(bu.X) + 0.5 - player.Position.X
		dy := float64(bu.Y) + 0.5 - player.Position.Y
		dz := float64(bu.Z) + 0.5 - player.Position.Z
		return !math.IsNaN(dx) && !math.IsNaN(dy) && !math.IsNaN(dz) &&
			!math.IsInf(dx, 0) && !math.IsInf(dy, 0) && !math.IsInf(dz, 0) &&
			dx*dx+dy*dy+dz*dz <= 36 // Six-block interaction limit.
	}
	return false
}

func validPlayerPosition(pos protocol.PlayerPositionPacket) bool {
	return !math.IsNaN(pos.X) && !math.IsNaN(pos.Y) && !math.IsNaN(pos.Z) &&
		!math.IsInf(pos.X, 0) && !math.IsInf(pos.Y, 0) && !math.IsInf(pos.Z, 0) &&
		math.Abs(pos.X) <= world.WorldBorderLimit && math.Abs(pos.Z) <= world.WorldBorderLimit &&
		pos.Y >= world.MinWorldY-64 && pos.Y <= world.MaxWorldY+64 &&
		!math.IsNaN(float64(pos.Yaw)) && !math.IsNaN(float64(pos.Pitch)) &&
		!math.IsInf(float64(pos.Yaw), 0) && !math.IsInf(float64(pos.Pitch), 0) &&
		pos.Pitch >= -90 && pos.Pitch <= 90
}

func (s *Server) broadcastLoop() {
	for ev := range s.simEngine.Events {
		s.mu.Lock()
		sessions := make([]*ClientSession, 0, len(s.sessions))
		for _, sess := range s.sessions {
			state, _, _, _ := sess.snapshot()
			if state == protocol.StatePlay {
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
				_, playerID, _, _ := sess.snapshot()
				if playerID != ev.PlayerID { // Broadcast to other players
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
		ID:       s.nextSessID,
		conn:     conn,
		server:   s,
		state:    protocol.StateHandshake,
		outbound: make(chan protocol.Packet, 256),
		done:     make(chan struct{}),
	}
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Session %d] Recovered panic while handling connection: %v\n%s", sess.ID, r, debug.Stack())
		}
		sess.Close()
		s.mu.Lock()
		delete(s.sessions, sess.ID)
		s.mu.Unlock()
	}()

	s.writeWG.Add(1)
	go sess.writeLoop()
	sess.readLoop()
}

func (sess *ClientSession) Send(pkt protocol.Packet) error {
	sess.mu.Lock()
	if sess.closed {
		sess.mu.Unlock()
		return net.ErrClosed
	}
	outbound := sess.outbound
	done := sess.done
	sess.mu.Unlock()
	select {
	case <-done:
		return net.ErrClosed
	default:
	}
	select {
	case <-done:
		return net.ErrClosed
	case outbound <- pkt:
		return nil
	default:
		err := fmt.Errorf("session %d outbound queue is full", sess.ID)
		log.Printf("[Session %d] Disconnecting slow client: %v", sess.ID, err)
		sess.Close()
		return err
	}
}

func (sess *ClientSession) writeLoop() {
	defer sess.server.writeWG.Done()
	for {
		select {
		case <-sess.done:
			return
		case pkt := <-sess.outbound:
			sess.mu.Lock()
			closed, conn := sess.closed, sess.conn
			sess.mu.Unlock()
			if closed {
				return
			}
			if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				log.Printf("[Session %d] Failed to set packet write deadline: %v", sess.ID, err)
				sess.Close()
				return
			}
			if err := protocol.SendPacket(conn, pkt); err != nil {
				log.Printf("[Session %d] Packet write failed: %v", sess.ID, err)
				sess.Close()
				return
			}
			if err := conn.SetWriteDeadline(time.Time{}); err != nil {
				log.Printf("[Session %d] Failed to clear packet write deadline: %v", sess.ID, err)
				sess.Close()
				return
			}
		}
	}
}

func (sess *ClientSession) Close() {
	sess.mu.Lock()
	if sess.closed {
		sess.mu.Unlock()
		return
	}
	sess.closed = true
	conn := sess.conn
	playerID := sess.playerID
	sess.mu.Unlock()
	sess.doneOnce.Do(func() { close(sess.done) })
	_ = conn.Close()
	if playerID > 0 {
		sess.server.simEngine.RemovePlayer(playerID)
	}
}

func (sess *ClientSession) snapshot() (protocol.ConnectionState, int32, string, [16]byte) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.state, sess.playerID, sess.username, sess.uuid
}

func (sess *ClientSession) setState(state protocol.ConnectionState) {
	sess.mu.Lock()
	sess.state = state
	sess.mu.Unlock()
}

func (sess *ClientSession) setIdentity(username string, uuid [16]byte) {
	sess.mu.Lock()
	sess.username = username
	sess.uuid = uuid
	sess.mu.Unlock()
}

func (sess *ClientSession) activatePlayer(playerID int32) {
	sess.mu.Lock()
	sess.playerID = playerID
	sess.state = protocol.StatePlay
	sess.mu.Unlock()
}

func (sess *ClientSession) readLoop() {
	for {
		raw, err := protocol.ReadRawPacket(sess.conn)
		if err != nil {
			return
		}

		if err := sess.handlePacket(raw); err != nil {
			state, _, _, _ := sess.snapshot()
			log.Printf("[Session %d] Error handling packet (state %v, id 0x%02X): %v",
				sess.ID, state, raw.ID, err)
			return
		}
	}
}

func (sess *ClientSession) handlePacket(raw *protocol.RawPacket) error {
	state, _, _, _ := sess.snapshot()
	switch state {
	case protocol.StateHandshake:
		if raw.ID == 0x00 {
			var hs protocol.HandshakePacket
			if err := hs.Decode(raw.Reader()); err != nil {
				return err
			}
			if hs.NextState == 1 {
				sess.setState(protocol.StateStatus)
			} else if hs.NextState == 2 {
				sess.setState(protocol.StateLogin)
			}
		}

	case protocol.StateStatus:
		if raw.ID == 0x00 { // StatusRequest
			sess.server.mu.Lock()
			onlineCount := 0
			for _, s := range sess.server.sessions {
				state, _, _, _ := s.snapshot()
				if state == protocol.StatePlay {
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
			sess.setIdentity(ls.Username, ls.PlayerUUID)

			// Send LoginSuccess
			if err := sess.Send(&protocol.LoginSuccessPacket{
				UUID:     ls.PlayerUUID,
				Username: ls.Username,
			}); err != nil {
				return err
			}

			// Add player to simulation engine at spawn (X=8, Y=65, Z=8)
			player := sess.server.simEngine.AddPlayer(ls.Username, ls.PlayerUUID, sim.Vec3{X: 8, Y: 65, Z: 8})
			sess.activatePlayer(player.ID)

			// Snapshot shared state while locked, but never hold the global
			// session lock across socket writes to a newly connected client.
			sess.server.mu.Lock()
			modifiedBlocks := make(map[world.BlockPos]int32, len(sess.server.modifiedBlocks))
			for bpos, bid := range sess.server.modifiedBlocks {
				modifiedBlocks[bpos] = bid
			}

			others := make([]*ClientSession, 0, len(sess.server.sessions))
			for _, other := range sess.server.sessions {
				if other.ID != sess.ID {
					others = append(others, other)
				}
			}
			sess.server.mu.Unlock()
			for bpos, bid := range modifiedBlocks {
				_ = sess.Send(&protocol.BlockUpdatePacket{
					X:       bpos.X,
					Y:       bpos.Y,
					Z:       bpos.Z,
					BlockID: bid,
				})
			}
			playersByID := make(map[int32]sim.PlayerEntity)
			for _, otherPlayer := range sess.server.simEngine.GetPlayerList() {
				playersByID[otherPlayer.ID] = otherPlayer
			}
			for _, other := range others {
				otherState, otherID, _, _ := other.snapshot()
				otherP, exists := playersByID[otherID]
				if otherState == protocol.StatePlay && exists {
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

			log.Printf("[Server] Player joined: %s (UUID: %x, EntityID: %d)", ls.Username, ls.PlayerUUID, player.ID)
		}

	case protocol.StatePlay:
		switch raw.ID {
		case 0x1A: // PlayerPosition
			var pos protocol.PlayerPositionPacket
			if err := pos.Decode(raw.Reader()); err != nil {
				return err
			}
			if !validPlayerPosition(pos) {
				_, _, username, _ := sess.snapshot()
				return fmt.Errorf("invalid player position from %q", username)
			}
			_, playerID, _, _ := sess.snapshot()
			sess.server.simEngine.MovePlayer(playerID, sim.Vec3{X: pos.X, Y: pos.Y, Z: pos.Z}, pos.Yaw, pos.Pitch, pos.OnGround)

		case 0x09: // BlockUpdate (Place/Break)
			var bu protocol.BlockUpdatePacket
			if err := bu.Decode(raw.Reader()); err != nil {
				return err
			}
			if world.ValidBlockCoordinates(bu.X, bu.Y, bu.Z) && bu.BlockID >= 0 && sess.canModifyBlock(bu) {
				if !sess.server.simEngine.SetBlockChecked(bu.X, bu.Y, bu.Z, bu.BlockID) {
					return nil
				}
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

