package test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/protocol"
)

// safeOutputBuffer captures process stdout thread-safely and provides wait-for-pattern matching.
type safeOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func newSafeOutputBuffer(r io.Reader) *safeOutputBuffer {
	sb := &safeOutputBuffer{}
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			sb.mu.Lock()
			sb.buf.WriteString(text)
			sb.buf.WriteString("\n")
			sb.mu.Unlock()
		}
	}()
	return sb
}

func (sb *safeOutputBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func (sb *safeOutputBuffer) WaitFor(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(sb.String(), substr) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return strings.Contains(sb.String(), substr)
}

// TestRealMultiplayerDualClientSyncAndPersistence implements the mandatory work order:
// "两台真实 Go 客户端进入同一世界，互相看见移动和挖放，服务端重启后还能读回改动"
func TestRealMultiplayerDualClientSyncAndPersistence(t *testing.T) {
	binDir, err := os.MkdirTemp("", "mcgo_bin_*")
	if err != nil {
		t.Fatalf("failed to create temp bin dir: %v", err)
	}
	defer os.RemoveAll(binDir)

	serverExe := filepath.Join(binDir, "server.exe")
	clientExe := filepath.Join(binDir, "client.exe")
	if runtime.GOOS != "windows" {
		serverExe = filepath.Join(binDir, "server")
		clientExe = filepath.Join(binDir, "client")
	}

	// 1. Compile real production binaries for server and client
	t.Log("[Build] Compiling real server binary...")
	cmdBuildSrv := exec.Command("go", "build", "-o", serverExe, "../cmd/server")
	if out, err := cmdBuildSrv.CombinedOutput(); err != nil {
		t.Fatalf("failed to build real server binary: %v\nOutput: %s", err, string(out))
	}

	t.Log("[Build] Compiling real client binary...")
	cmdBuildCli := exec.Command("go", "build", "-o", clientExe, "../cmd/client")
	if out, err := cmdBuildCli.CombinedOutput(); err != nil {
		t.Fatalf("failed to build real client binary: %v\nOutput: %s", err, string(out))
	}

	// 2. Allocate an ephemeral free port and temporary world storage
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind ephemeral port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)

	worldDir, err := os.MkdirTemp("", "mcgo_world_*")
	if err != nil {
		t.Fatalf("failed to create temp world directory: %v", err)
	}
	defer os.RemoveAll(worldDir)

	t.Logf("[Server 1] Starting real server on %s with world storage at %s...", serverAddr, worldDir)
	srv1Cmd := exec.Command(serverExe, "-port", strconv.Itoa(port), "-world", worldDir)
	srv1In, err := srv1Cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get srv1 stdin: %v", err)
	}
	srv1OutPipe, err := srv1Cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get srv1 stdout: %v", err)
	}
	srv1ErrPipe, err := srv1Cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to get srv1 stderr: %v", err)
	}
	srv1Out := newSafeOutputBuffer(io.MultiReader(srv1OutPipe, srv1ErrPipe))

	if err := srv1Cmd.Start(); err != nil {
		t.Fatalf("failed to start srv1: %v", err)
	}

	// Wait for server to listen
	ready := false
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", serverAddr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("server 1 did not become ready within 5s. Output:\n%s", srv1Out.String())
	}

	// 3. Verify Server Status SLP Ping: protocol 777, online: 0
	checkSLPPing := func(expectedOnline int) {
		pConn, err := net.DialTimeout("tcp", serverAddr, 2*time.Second)
		if err != nil {
			t.Fatalf("SLP ping failed to dial: %v", err)
		}
		defer pConn.Close()

		hs := &protocol.HandshakePacket{
			ProtocolVersion: protocol.ProtocolVersion,
			ServerAddress:   "127.0.0.1",
			ServerPort:      uint16(port),
			NextState:       1, // Status
		}
		if err := protocol.SendPacket(pConn, hs); err != nil {
			t.Fatalf("SLP ping failed to send handshake: %v", err)
		}
		if err := protocol.SendPacket(pConn, &protocol.StatusRequestPacket{}); err != nil {
			t.Fatalf("SLP ping failed to send request: %v", err)
		}
		raw, err := protocol.ReadRawPacket(pConn)
		if err != nil {
			t.Fatalf("SLP ping failed to read response: %v", err)
		}
		var resp protocol.StatusResponsePacket
		if err := resp.Decode(raw.Reader()); err != nil {
			t.Fatalf("SLP ping failed to decode response: %v", err)
		}

		var data struct {
			Version struct {
				Protocol int `json:"protocol"`
			} `json:"version"`
			Players struct {
				Online int `json:"online"`
			} `json:"players"`
		}
		if err := json.Unmarshal([]byte(resp.JSONResponse), &data); err != nil {
			t.Fatalf("SLP JSON unmarshal failed: %v, raw: %s", err, resp.JSONResponse)
		}
		if data.Version.Protocol != 777 {
			t.Fatalf("SLP protocol mismatch: expected 777, got %d", data.Version.Protocol)
		}
		if data.Players.Online != expectedOnline {
			t.Fatalf("SLP online players mismatch: expected %d, got %d", expectedOnline, data.Players.Online)
		}
	}

	checkSLPPing(0)

	// 4. Launch Client 1 (Alice) in real CLI mode
	t.Log("[Client 1] Launching real client Alice...")
	cliA := exec.Command(clientExe, "-server", serverAddr, "-username", "Alice", "-cli")
	cliAIn, err := cliA.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get cliA stdin: %v", err)
	}
	cliAOutPipe, err := cliA.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get cliA stdout: %v", err)
	}
	cliAOut := newSafeOutputBuffer(cliAOutPipe)
	if err := cliA.Start(); err != nil {
		t.Fatalf("failed to start cliA: %v", err)
	}

	// Verify Alice joins and status command does not crash
	cliAIn.Write([]byte("status\n"))
	if !cliAOut.WaitFor("Player: Alice", 5*time.Second) {
		t.Fatalf("Alice failed to join or status crashed:\n%s", cliAOut.String())
	}
	checkSLPPing(1)

	// 5. Launch Client 2 (Bob) in real CLI mode
	t.Log("[Client 2] Launching real client Bob...")
	cliB := exec.Command(clientExe, "-server", serverAddr, "-username", "Bob", "-cli")
	cliBIn, err := cliB.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get cliB stdin: %v", err)
	}
	cliBOutPipe, err := cliB.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get cliB stdout: %v", err)
	}
	cliBOut := newSafeOutputBuffer(cliBOutPipe)
	if err := cliB.Start(); err != nil {
		t.Fatalf("failed to start cliB: %v", err)
	}

	// Verify Bob joins and status command does not crash
	cliBIn.Write([]byte("status\n"))
	if !cliBOut.WaitFor("Player: Bob", 5*time.Second) {
		t.Fatalf("Bob failed to join or status crashed:\n%s", cliBOut.String())
	}
	checkSLPPing(2)

	// 6. Real Block Placement & Real-Time Sync:
	// Alice places Diamond Block (ID=57) at (5, 65, 5)
	t.Log("[Block Sync] Alice places Diamond Block at (5, 65, 5)...")
	cliAIn.Write([]byte("place_at 5 65 5 57\n"))
	time.Sleep(500 * time.Millisecond)

	// Bob queries the block at (5, 65, 5)
	cliBIn.Write([]byte("block 5 65 5\n"))
	if !cliBOut.WaitFor("Block at (5, 65, 5): ID=57", 5*time.Second) {
		t.Fatalf("Bob did not receive Alice's placed Diamond Block:\n%s", cliBOut.String())
	}
	t.Log("[Block Sync] Verified: Bob successfully observed Alice's block placement!")

	// 7. Real Player Movement & Real-Time Sync:
	// Alice moves to (12.0, 65.0, 14.0)
	t.Log("[Move Sync] Alice moves to (12.0, 65.0, 14.0)...")
	cliAIn.Write([]byte("goto 12 65 14\n"))
	time.Sleep(500 * time.Millisecond)

	// Bob runs status to verify Alice's position is tracked
	cliBIn.Write([]byte("status\n"))
	if !cliBOut.WaitFor("12.00, 65.00, 14.00", 5*time.Second) {
		t.Fatalf("Bob did not observe Alice's moved position:\n%s", cliBOut.String())
	}
	t.Log("[Move Sync] Verified: Bob successfully observed Alice's movement!")

	// 8. Gracefully exit both clients
	t.Log("[Shutdown] Disconnecting Alice and Bob...")
	cliAIn.Write([]byte("quit\n"))
	cliBIn.Write([]byte("quit\n"))
	cliA.Wait()
	cliB.Wait()

	// 9. Stop server gracefully (flushing dirty chunks to region files)
	t.Log("[Shutdown] Stopping server 1 gracefully via stop command...")
	srv1In.Write([]byte("stop\n"))
	srv1Cmd.Wait()
	t.Logf("[Server 1 Output]:\n%s", srv1Out.String())

	// Verify region file was generated on disk
	mcaPath := filepath.Join(worldDir, "region", "r.0.0.mca")
	fi, err := os.Stat(mcaPath)
	if err != nil || fi.Size() < 8192 {
		t.Fatalf("expected region file %s with size >= 8192, got err=%v", mcaPath, err)
	}
	t.Logf("[Persistence] Region file created on disk: %s (size: %d bytes)", mcaPath, fi.Size())

	// 10. Restart Server with the SAME world storage directory
	t.Logf("[Server 2] Restarting server on %s with existing world storage at %s...", serverAddr, worldDir)
	srv2Cmd := exec.Command(serverExe, "-port", strconv.Itoa(port), "-world", worldDir)
	srv2In, err := srv2Cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get srv2 stdin: %v", err)
	}
	srv2OutPipe, err := srv2Cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get srv2 stdout: %v", err)
	}
	srv2ErrPipe, err := srv2Cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to get srv2 stderr: %v", err)
	}
	srv2Out := newSafeOutputBuffer(io.MultiReader(srv2OutPipe, srv2ErrPipe))
	if err := srv2Cmd.Start(); err != nil {
		t.Fatalf("failed to start srv2: %v", err)
	}

	ready = false
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", serverAddr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("server 2 did not become ready within 5s. Output:\n%s", srv2Out.String())
	}

	// 11. Client Alice reconnects to restarted server and verifies block was restored from disk
	t.Log("[Client 1 Reconnect] Alice reconnects to restarted server...")
	cliA2 := exec.Command(clientExe, "-server", serverAddr, "-username", "Alice", "-cli")
	cliA2In, err := cliA2.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get cliA2 stdin: %v", err)
	}
	cliA2OutPipe, err := cliA2.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get cliA2 stdout: %v", err)
	}
	cliA2Out := newSafeOutputBuffer(cliA2OutPipe)
	if err := cliA2.Start(); err != nil {
		t.Fatalf("failed to start cliA2: %v", err)
	}

	// Verify Alice joins restarted server
	cliA2In.Write([]byte("status\n"))
	if !cliA2Out.WaitFor("Player: Alice", 5*time.Second) {
		t.Fatalf("Alice failed to join restarted server or status crashed:\n%s", cliA2Out.String())
	}
	time.Sleep(500 * time.Millisecond)

	// Query block placed in previous server run
	cliA2In.Write([]byte("block 5 65 5\n"))
	if !cliA2Out.WaitFor("Block at (5, 65, 5): ID=57", 5*time.Second) {
		t.Fatalf("Alice failed to read back persisted Diamond Block after server restart:\n%s", cliA2Out.String())
	}
	t.Log("[Persistence Verified] Success! Diamond Block at (5, 65, 5) restored from disk after server restart!")

	// 12. Cleanup
	cliA2In.Write([]byte("quit\n"))
	cliA2.Wait()
	srv2In.Write([]byte("stop\n"))
	srv2Cmd.Wait()
}
